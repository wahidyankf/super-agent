package kiwi

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
)

// rpcRequest is one JSON-RPC 2.0 request. Params is a named object, and the adapter sends one call per HTTP request.
type rpcRequest struct {
	JSONRPC string         `json:"jsonrpc"`
	Method  string         `json:"method"`
	Params  map[string]any `json:"params"`
	ID      int64          `json:"id"`
}

// rpcEnvelope is the reply envelope. Result and Error are kept raw or pointed, so the caller checks which one is present.
type rpcEnvelope struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int64          `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *rpcErrorObject `json:"error"`
}

// rpcErrorObject is the error member of a JSON-RPC reply. Its message is matched by errors.go and never shown.
type rpcErrorObject struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// call sends one allowlisted method with its named params and returns the raw result of a 2xx reply. The method must be one
// the source's access level allows, so a write method on a read source never reaches the network. It retries a 429 reply
// after its Retry-After wait, at most maxRetries times; a write is retried for no other reason. The request lock is held
// for the whole call, wait included, so no other request goes out while Kiwi asks for a pause.
func (g *Gateway) call(ctx context.Context, method string, params map[string]any) (json.RawMessage, error) {
	if !permittedMethod(method, g.access) {
		return nil, errMethodNotAllowed
	}
	if err := guardArguments(params); err != nil {
		return nil, err
	}
	write := methodAccess[method] == domain.AccessWrite
	g.mu.Lock()
	defer g.mu.Unlock()
	g.nextID++
	payload, err := json.Marshal(rpcRequest{JSONRPC: "2.0", Method: method, Params: params, ID: g.nextID})
	if err != nil {
		return nil, fmt.Errorf("encode %s request: %w", method, err)
	}
	for attempt := 0; ; attempt++ {
		raw, wait, retry, err := g.attempt(ctx, method, payload, g.nextID, write)
		if err != nil {
			return nil, err
		}
		if !retry {
			return raw, nil
		}
		if attempt >= maxRetries {
			return nil, domain.ErrSourceRateLimited
		}
		if err := g.sleep(ctx, wait); err != nil {
			return nil, err
		}
	}
}

// attempt performs one request. The caller holds the request lock. It reads the key for this request only, paces the
// request after the previous one, and classifies the reply. The returned wait and retry flag apply only to a 429 reply.
// write marks a request that changes Kiwi, whose failed exchange has an unknown outcome.
func (g *Gateway) attempt(ctx context.Context, method string, payload []byte, id int64, write bool) (json.RawMessage, time.Duration, bool, error) {
	key, err := g.creds.APIKey(g.credential)
	if err != nil {
		return nil, 0, false, fmt.Errorf("kiwi credentials: %w", err)
	}
	if err := g.pace(ctx); err != nil {
		return nil, 0, false, err
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
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, g.base+endpointPath, bytes.NewReader(payload))
	if err != nil {
		return nil, 0, false, errInvalidBaseURL
	}
	req.Host = g.host
	req.Header.Set("Authorization", "Bearer "+key.Reveal())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		g.finish()
		return nil, 0, false, failedExchange(ctx, err, write, handshakeFailed.Load())
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes+1))
	g.finish()
	if err != nil {
		// Kiwi answered its status before the body broke, so a write may already have run.
		if write {
			return nil, 0, false, domain.ErrWriteOutcomeUnknown
		}
		return nil, 0, false, transportError(ctx, err)
	}
	if len(raw) > maxBodyBytes {
		if write {
			return nil, 0, false, domain.ErrWriteOutcomeUnknown
		}
		return nil, 0, false, domain.ErrReplyTooLarge
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		wait, retry, err := retryAfter(resp.Header.Get("Retry-After"))
		return nil, wait, retry, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, 0, false, statusError(resp.StatusCode, write)
	}
	result, err := decodeEnvelope(method, raw, id, write)
	return result, 0, false, err
}

// decodeEnvelope reads a 2xx reply. The id must be the id of the request, and the reply must carry exactly the result or
// the error of JSON-RPC. A reply to a write that cannot be read has an unknown outcome, so the caller never retries it.
func decodeEnvelope(method string, raw []byte, id int64, write bool) (json.RawMessage, error) {
	unreadable := domain.ErrUnexpectedResponse
	if write {
		unreadable = domain.ErrWriteOutcomeUnknown
	}
	var env rpcEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, unreadable
	}
	if env.ID == nil || *env.ID != id {
		return nil, unreadable
	}
	if env.Error != nil {
		return nil, rpcError(method, env.Error.Code, env.Error.Message, write)
	}
	result := bytes.TrimSpace(env.Result)
	if len(result) == 0 || bytes.Equal(result, []byte("null")) {
		return nil, unreadable
	}
	return result, nil
}

// retryAfter reads a 429 reply's Retry-After. A usable value at most maxRetryAfter seconds asks for a retry after that
// wait. A longer one is ErrRetryAfterTooLong, and a missing or unusable one is ErrSourceRateLimited, which is not retried.
func retryAfter(value string) (time.Duration, bool, error) {
	seconds, ok := parseRetryAfter(value)
	switch {
	case !ok:
		return 0, false, domain.ErrSourceRateLimited
	case seconds > maxRetryAfter:
		return 0, false, domain.ErrRetryAfterTooLong
	}
	return time.Duration(seconds) * time.Second, true, nil
}

// parseRetryAfter reads Retry-After as whole seconds. A missing, negative, or non-numeric value is unusable. A value too
// large to hold is reported as one over the limit.
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

// failedExchange maps an exchange that failed before a reply. A write whose request may have left the client has an
// unknown outcome, so Retriever never retries it. A connection that was never made, or a TLS handshake
// that failed, did not send the request, so its outcome is known and it maps as a read does.
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

// transportError maps a failed exchange onto a sentinel. A cancelled caller gets its own error. The underlying error text
// is never used, because it can name the request.
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
