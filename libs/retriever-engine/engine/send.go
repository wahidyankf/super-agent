package engine

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// reply is the part of one exchange that the classifier reads: the status, the Retry-After value, and the body of a
// successful reply. The body of any other reply is never read, so it never reaches an error.
type reply struct {
	status     int
	retryAfter string
	body       []byte
}

// send runs one request of an operation until it gets a reply it can use. A rate limit is retried after its wait, at
// most maxRetries times, for a read and for a write alike, because a 429 reply says the source refused the request before
// it acted. Every other failure stops at once. For a write, a failure after the request left is an unknown outcome, and
// the write is not sent again.
func (e *Engine) send(ctx context.Context, o spec.Operation, r request) (any, error) {
	write := o.Access != "read"
	for attempt := 0; ; attempt++ {
		rep, err := e.exchange(ctx, r, write)
		if err != nil {
			return nil, err
		}
		wait, retry, err := classify(o, rep.status, rep.retryAfter)
		if retry {
			if attempt >= maxRetries {
				return nil, domain.ErrSourceRateLimited
			}
			if err := e.ports.Clock.Sleep(ctx, wait); err != nil {
				return nil, err
			}
			continue
		}
		if err != nil {
			if write && unusableWriteReply(rep.status) {
				return nil, domain.ErrWriteOutcomeUnknown
			}
			return nil, err
		}
		root, err := decodeReply(rep.body)
		if err != nil && write {
			return nil, domain.ErrWriteOutcomeUnknown
		}
		return root, err
	}
}

// exchange sends one request once. It reads the key for this request, sets the authorization header, and runs the
// request under the time limit. The body of a successful reply is read up to the reply cap, and a longer one fails. For a
// write, the reply is read whole, up to the cap, whatever its status, and a failure after the request left is an unknown
// outcome. A connection that never opened means the request did not leave, so that failure is known.
func (e *Engine) exchange(ctx context.Context, r request, write bool) (reply, error) {
	key, err := e.ports.Credentials.APIKey(e.src.Credential())
	if err != nil {
		return reply{}, err
	}
	secret := key.Reveal()
	tctx, cancel := e.ports.Clock.WithTimeout(ctx, e.timeout)
	defer cancel()
	var connected atomic.Bool
	trace := &httptrace.ClientTrace{
		GotConn: func(httptrace.GotConnInfo) { connected.Store(true) },
	}
	var body io.Reader
	if r.body != nil {
		body = bytes.NewReader(r.body)
	}
	req, err := http.NewRequestWithContext(httptrace.WithClientTrace(tctx, trace), r.method, e.base+r.target, body)
	if err != nil {
		return reply{}, fmt.Errorf("the request could not be built: %w", err)
	}
	req.Header = http.Header{}
	for name, values := range r.header {
		req.Header[name] = values
	}
	req.Header.Set("Accept", "application/json")
	if r.body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if err := e.authenticate(req, key); err != nil {
		return reply{}, err
	}
	resp, err := e.client.Do(req)
	if err != nil {
		// An injected transport stands for the network, so its failure follows the request. The guarded transport
		// has made a connection only once the trace reports one.
		if write && (connected.Load() || e.ports.Transport != nil) {
			return reply{}, domain.ErrWriteOutcomeUnknown
		}
		return reply{}, transportError(ctx, err, secret)
	}
	defer resp.Body.Close()
	rep := reply{status: resp.StatusCode, retryAfter: resp.Header.Get("Retry-After")}
	if write {
		raw, err := readCapped(resp.Body, e.maxReply)
		if err != nil {
			return reply{}, domain.ErrWriteOutcomeUnknown
		}
		rep.body = raw
		return rep, nil
	}
	if rep.status < 200 || rep.status >= 300 {
		return rep, nil
	}
	raw, err := readCapped(resp.Body, e.maxReply)
	if errors.Is(err, domain.ErrReplyTooLarge) {
		return reply{}, err
	}
	if err != nil {
		return reply{}, transportError(ctx, err, secret)
	}
	rep.body = raw
	return rep, nil
}

// authenticate sets the key on a request in the form the spec names. The key goes only in the one header that the auth
// type uses, and a basic identity is read from its own port.
func (e *Engine) authenticate(req *http.Request, key domain.ApiKey) error {
	secret := key.Reveal()
	switch e.model.Auth.Type {
	case "basic":
		id, err := e.ports.Identities.Identity(e.src.Identity())
		if err != nil {
			return err
		}
		pair := id.Reveal() + ":" + secret
		req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(pair)))
	case "header":
		req.Header.Set(e.model.Auth.HeaderName, secret)
	default:
		req.Header.Set("Authorization", "Bearer "+secret)
	}
	return nil
}

// readCapped reads at most limit bytes. A reply one byte longer than the limit fails with ErrReplyTooLarge.
func readCapped(r io.Reader, limit int) ([]byte, error) {
	raw, err := io.ReadAll(io.LimitReader(r, int64(limit)+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > limit {
		return nil, domain.ErrReplyTooLarge
	}
	return raw, nil
}

// classify maps the status of a reply to its outcome. It returns the wait and true when the read should retry after
// the wait, a nil error for a success, and otherwise the closed error of the reply. A redirect is refused before the
// spec's own mapping, and a rate limit is handled before it too.
func classify(o spec.Operation, status int, retryAfter string) (time.Duration, bool, error) {
	switch {
	case status >= 300 && status < 400:
		return 0, false, domain.ErrRedirected
	case status == http.StatusTooManyRequests:
		n, ok := parseRetryAfter(retryAfter)
		if !ok {
			return 0, false, domain.ErrSourceRateLimited
		}
		if n > maxRetryAfter {
			return 0, false, domain.ErrRetryAfterTooLong
		}
		return time.Duration(n) * time.Second, true, nil
	case status >= 200 && status < 300:
		return 0, false, nil
	}
	if reason, ok := o.Response.Errors[strconv.Itoa(status)]; ok {
		return 0, false, reasonError(reason, o.Method)
	}
	switch {
	case status == http.StatusUnauthorized:
		return 0, false, domain.ErrKeyRejected
	case status == http.StatusForbidden:
		return 0, false, domain.ErrAccessDenied
	case status == http.StatusBadRequest:
		return 0, false, domain.ErrBadRequest
	case status == http.StatusNotFound:
		return 0, false, domain.ErrNotFound
	case status >= 500 && status < 600:
		return 0, false, domain.ErrUpstreamFailure
	}
	return 0, false, domain.ErrUnexpectedResponse
}

// reasonError returns the closed error that a spec names for a status. A reason the engine does not know is an
// unexpected response, so a spec can never invent an outcome.
func reasonError(reason, method string) error {
	switch reason {
	case "key_rejected":
		return domain.ErrKeyRejected
	case "access_denied":
		return domain.ErrAccessDenied
	case "rate_limited":
		return domain.ErrSourceRateLimited
	case "not_found":
		return domain.ErrNotFound
	case "document_conflict":
		return domain.ErrDocumentConflict
	case "upstream_failure":
		return domain.ErrUpstreamFailure
	case "operation_unavailable":
		return domain.OperationUnavailableError{Method: method}
	case "unresolved_name":
		return domain.ErrUnresolvedName
	default:
		return domain.ErrUnexpectedResponse
	}
}

// parseRetryAfter reads a Retry-After value as a count of whole seconds. A value that is not digits is not usable. A
// value too large to count is returned one past the limit, so it is refused as too long.
func parseRetryAfter(value string) (int, bool) {
	s := strings.TrimSpace(value)
	if s == "" {
		return 0, false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, false
		}
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return maxRetryAfter + 1, true
	}
	return n, true
}

// transportError maps a failed exchange to its error. A deadline is a timeout. A refused network keeps its own error.
// A cancelled caller keeps its own error. Any other failure is unreachable, and its text has the key redacted.
func transportError(ctx context.Context, err error, secret string) error {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return domain.ErrTimedOut
	case errors.Is(err, ErrNetworkRefused):
		return err
	case ctx.Err() != nil:
		return ctx.Err()
	default:
		return fmt.Errorf("%w: %s", domain.ErrUnreachable, redact(err.Error(), secret))
	}
}

// redact replaces every occurrence of the secret in a text. An empty secret changes nothing.
func redact(text, secret string) string {
	if secret == "" {
		return text
	}
	return strings.ReplaceAll(text, secret, "[REDACTED]")
}
