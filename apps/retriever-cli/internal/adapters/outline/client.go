package outline

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/sources/operations"
)

// call sends the request of one operation and decodes a 2xx reply into out. The method comes from a closed endpoint
// table, so only a method of the table reaches the network. It
// retries a 429 reply after its Retry-After wait, at most maxRetries times. A write is retried for no other reason
// . The request lock is held for the whole call, wait included, so no other request goes out while
// the source asks for a pause.
func (g *Gateway) call(ctx context.Context, op domain.OperationID, method string, in, out any) error {
	payload, err := json.Marshal(in)
	if err != nil {
		return fmt.Errorf("encode %s request: %w", method, err)
	}
	write := writes(op)
	g.mu.Lock()
	defer g.mu.Unlock()
	for attempt := 0; ; attempt++ {
		wait, retry, err := g.attempt(ctx, method, payload, out, write)
		if err != nil {
			return err
		}
		if !retry {
			return nil
		}
		if attempt >= maxRetries {
			return domain.ErrSourceRateLimited
		}
		if err := g.sleep(ctx, wait); err != nil {
			return err
		}
	}
}

// attempt performs one request. The caller holds the request lock. It reads the key for
// this request only, paces the request after the previous one, and classifies the
// reply. The returned wait and retry flag apply only to a 429 reply. write marks a request
// that changes the wiki, whose failed exchange has an unknown outcome.
func (g *Gateway) attempt(ctx context.Context, method string, payload []byte, out any, write bool) (time.Duration, bool, error) {
	key, err := g.creds.APIKey(g.credential)
	if err != nil {
		return 0, false, fmt.Errorf("outline credentials: %w", err)
	}

	if err := g.pace(ctx); err != nil {
		return 0, false, err
	}

	// The trace records whether the TLS handshake failed. A failed handshake means no byte of the request was sent.
	var handshakeFailed atomic.Bool
	trace := &httptrace.ClientTrace{
		TLSHandshakeDone: func(_ tls.ConnectionState, err error) {
			if err != nil {
				handshakeFailed.Store(true)
			}
		},
	}
	reqCtx, cancel := context.WithTimeout(httptrace.WithClientTrace(ctx, trace), g.timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, g.base+"/api/"+method, bytes.NewReader(payload))
	if err != nil {
		return 0, false, errInvalidBaseURL
	}
	req.Host = g.host
	req.Header.Set("Authorization", "Bearer "+key.Reveal())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-API-Version", apiVersion)

	resp, err := g.client.Do(req)
	if err != nil {
		g.finish()
		return 0, false, failedExchange(ctx, err, write, handshakeFailed.Load())
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes+1))
	g.finish()
	if err != nil {
		// The wiki answered its status before the body broke, so a write may already have run.
		if write {
			return 0, false, domain.ErrWriteOutcomeUnknown
		}
		return 0, false, transportError(ctx, err)
	}
	if len(raw) > maxBodyBytes {
		// A write has no limit to lower, so its over-size reply is an unknown outcome, not a request to ask for less (m1).
		if write {
			return 0, false, domain.ErrWriteOutcomeUnknown
		}
		return 0, false, domain.ErrReplyTooLarge
	}
	wait, retry, err := interpret(method, resp.StatusCode, resp.Header.Get("Retry-After"), raw, out)
	if write && err != nil && unusableWriteReply(resp.StatusCode) {
		return 0, false, domain.ErrWriteOutcomeUnknown
	}
	return wait, retry, err
}

// unusableWriteReply reports whether a reply to a write says nothing usable about the change: a status below 300, which
// is no final answer or a 2xx the client could not read, or a 5xx. A 3xx, a 4xx, and a 429 are answers the wiki gave
// about the request, so they keep their own sentinels (finding m1).
func unusableWriteReply(status int) bool {
	return status < http.StatusMultipleChoices || status >= http.StatusInternalServerError
}

// pace waits until minDelay has passed since the previous request ended. The caller holds the lock.
func (g *Gateway) pace(ctx context.Context) error {
	if !g.hasLast || g.minDelay <= 0 {
		return nil
	}
	wait := g.minDelay - g.now().Sub(g.lastEnd)
	if wait <= 0 {
		return nil
	}
	return g.sleep(ctx, wait)
}

// finish records when the previous request ended. The caller holds the lock.
func (g *Gateway) finish() {
	g.lastEnd = g.now()
	g.hasLast = true
}

// interpret maps one reply onto a decoded value or a sentinel. Its errors never carry the body.
func interpret(method string, status int, retryAfter string, raw []byte, out any) (time.Duration, bool, error) {
	switch {
	case status >= 300 && status < 400:
		return 0, false, domain.ErrRedirected
	case status == http.StatusUnauthorized:
		return 0, false, domain.ErrKeyRejected
	case status == http.StatusForbidden:
		return 0, false, domain.ErrAccessDenied
	case status == http.StatusBadRequest:
		return 0, false, domain.ErrBadRequest
	case status == http.StatusNotFound:
		if endpointUnavailable(raw) {
			return 0, false, domain.OperationUnavailableError{Method: method}
		}
		return 0, false, notFoundFor(method)
	case status == http.StatusTooManyRequests:
		seconds, ok := parseRetryAfter(retryAfter)
		if !ok {
			return 0, false, domain.ErrSourceRateLimited
		}
		if seconds > maxRetryAfter {
			return 0, false, domain.ErrRetryAfterTooLong
		}
		return time.Duration(seconds) * time.Second, true, nil
	case status == http.StatusNoContent:
		return 0, false, noContentFor(method)
	case status == http.StatusConflict:
		// A write with last_revision is refused when the document moved on.
		if apiErrorID(raw) == "document_conflict" {
			return 0, false, domain.ErrDocumentConflict
		}
		return 0, false, domain.ErrUnexpectedResponse
	case status >= 500:
		return 0, false, domain.ErrUpstreamFailure
	case status < 200 || status >= 300:
		return 0, false, domain.ErrUnexpectedResponse
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return 0, false, domain.ErrUnexpectedResponse
	}
	return 0, false, nil
}

// apiErrorID returns the snake_case error ID of an Outline error body, such as document_conflict, or "" when the body
// holds none. The ID is never shown as it is: a caller maps it to a fixed sentinel.
func apiErrorID(raw []byte) string {
	var body struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(raw, &body) != nil {
		return ""
	}
	return body.Error
}

// writes reports whether an operation changes the wiki, which is every operation of a non-read effect. Its failed
// exchange has an unknown outcome.
func writes(op domain.OperationID) bool {
	row, ok := operations.Lookup(op)
	return ok && row.Effect != domain.EffectRead
}

// failedExchange maps an exchange that failed before a reply. A write whose request may have left the client has an
// unknown outcome: the wiki may have run it, so Retriever never retries it. A connection that was
// never made, or a TLS handshake that failed, did not send the request, so its outcome is known and it maps as a read
// does (finding m2).
func failedExchange(ctx context.Context, err error, write, handshakeFailed bool) error {
	if write && !notConnected(err) && !handshakeFailed {
		return domain.ErrWriteOutcomeUnknown
	}
	return transportError(ctx, err)
}

// notConnected reports whether an exchange failed while it dialed, before any byte of the request was sent.
func notConnected(err error) bool {
	var op *net.OpError
	return errors.As(err, &op) && op.Op == "dial"
}

// endpointUnavailable reports whether a 404 reply is the router's catch-all, "Endpoint not found", which means the
// server does not offer the method. A 404 for one document or collection carries another message.
func endpointUnavailable(raw []byte) bool {
	var body struct {
		Message string `json:"message"`
	}
	return json.Unmarshal(raw, &body) == nil && body.Message == endpointNotFoundMessage
}

// notFoundFor names the not-found sentinel for a 404 that is not the router's catch-all. A read of
// a collection or a document has its own sentinel, so the message names what is missing. Every other read answers
// ErrNotFound, whatever the list or object it asked for, so no 404 becomes an unexpected reply.
func notFoundFor(method string) error {
	switch method {
	case methodCollectionsInfo, methodCollectionsDocuments, methodCollectionsMemberships,
		methodCollectionsGroupMemberships:
		return domain.ErrCollectionNotFound
	case methodDocumentsInfo, methodDocumentsExport, methodDocumentsDocuments, methodDocumentsInsights,
		methodDocumentsMemberships, methodDocumentsGroupMemberships, methodDocumentsUsers:
		return domain.ErrDocumentNotFound
	}
	return domain.ErrNotFound
}

// noContentFor names the sentinel of a 204 reply. Outline answers 204 with no body when the object an info method asks
// for does not exist (pins.info and subscriptions.info, and shares.info when nothing is shared). The object's own
// not-found sentinel is used for an info method that has one, and every other 204 is ErrNotFound, so an empty answer
// is never read as a success or as an unexpected reply.
func noContentFor(method string) error {
	if strings.HasSuffix(method, ".info") {
		return notFoundFor(method)
	}
	return domain.ErrNotFound
}

// parseRetryAfter reads Retry-After as whole seconds. A missing, negative, or non-numeric
// value is unusable, and the caller then fails without a retry. A value too large to
// hold is reported as one over the limit.
func parseRetryAfter(value string) (int, bool) {
	n, err := strconv.Atoi(strings.TrimSpace(value))
	switch {
	case errors.Is(err, strconv.ErrRange):
		return maxRetryAfter + 1, true
	case err != nil || n < 0:
		return 0, false
	}
	return n, true
}

// transportError maps a failed exchange onto a sentinel. A cancelled caller gets its own error.
// The underlying error text is never used, because it can name the request.
func transportError(parent context.Context, err error) error {
	if perr := parent.Err(); perr != nil {
		return perr
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return domain.ErrTimedOut
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return domain.ErrTimedOut
	}
	return domain.ErrUnreachable
}

// sleepContext waits for d or until ctx ends, whichever comes first.
func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
