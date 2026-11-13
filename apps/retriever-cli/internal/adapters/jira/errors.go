package jira

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// errMethodNotAllowed reports a request outside the closed table. Callers never see it, because the adapter's own rows are all
// in the table.
var errMethodNotAllowed = errors.New("jira method is not allowed")

// errInvalidBaseURL reports a connection base that does not parse, has no host, or holds user information.
var errInvalidBaseURL = errors.New("invalid Jira base URL")

// kindError names Jira as the kind of a failure, so a presenter prints "Jira rejected the API key", never
// a text about another system. A context error is returned as it is, because the caller cancelled it, and an error that is
// already a KindError is not wrapped twice.
func kindError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	var named domain.KindError
	if errors.As(err, &named) {
		return err
	}
	return domain.KindError{Kind: domain.KindJira, Err: err}
}

// interpret maps one reply onto a decoded value or a sentinel. Its errors never carry the body, the headers, the token, or the
// e-mail address. A 410 Gone means the server does not offer the path, so the reply is the unavailable
// sentinel, whose text names the endpoint, not a method.
func interpret(ep endpoint, status int, retryAfter string, raw []byte, out any) (time.Duration, bool, error) {
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
		return 0, false, notFoundFor(ep)
	case status == http.StatusGone:
		return 0, false, domain.ErrOperationUnavailable
	case status == http.StatusTooManyRequests:
		seconds, ok := parseRetryAfter(retryAfter)
		if !ok {
			return 0, false, domain.ErrSourceRateLimited
		}
		if seconds > maxRetryAfter {
			return 0, false, domain.ErrRetryAfterTooLong
		}
		return time.Duration(seconds) * time.Second, true, nil
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

// notFoundFor names the not-found sentinel of a 404 reply. A path with an issue key is about one issue, so it is the document
// sentinel; every other path is ErrNotFound. Atlassian answers 404 both for an issue that does not exist and for one the user
// cannot see, so the text claims neither.
func notFoundFor(ep endpoint) error {
	if strings.Contains(ep.path, keyPlaceholder) {
		return domain.ErrDocumentNotFound
	}
	return domain.ErrNotFound
}

// parseRetryAfter reads Retry-After as whole seconds. A missing, negative, or non-numeric value is unusable, and the caller then
// fails without a retry. A value too large to hold is reported as one over the limit.
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

// transportError maps a failed exchange onto a sentinel. A cancelled caller gets its own error. The underlying error text is
// never used, because it can name the request and its URL.
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
