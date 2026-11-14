package engine

import (
	"context"
	"errors"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// ErrNetworkRefused reports a source address that the engine never dials: an IP literal, a local name, or an address
// of a private, loopback, or link-local range, unless the catalog allows it. Its code is in the connector area, as ADR
// 001 names it, because the refusal belongs to the connector rules and not to one source kind.
var ErrNetworkRefused = errors.New("the source address is refused: an IP literal, a local name, or a private address")

// ErrInvalidHeader reports a header value that holds a control character or is not UTF-8. It is refused before any
// request leaves, so a value can never add a header line.
var ErrInvalidHeader = errors.New("a header value holds a control character or is not UTF-8")

// ErrNotConfigured reports an engine that is built without a port or a setting that it needs to run.
var ErrNotConfigured = errors.New("the engine is missing a port or a setting that it needs")

// The exit statuses of a failed read, which the command line returns as its exit code. A timeout and a rate limit that
// is not waited out exit 124, as the ADR's exit contract says; every other failure exits 2.
const (
	// ExitFailed is the status of a read that did not complete.
	ExitFailed = 2
	// ExitTimedOut is the status of a read that timed out, or that a rate limit kept from completing.
	ExitTimedOut = 124
)

// reasons maps each sentinel the engine can return to the reason of its stable code. The reason names match the
// command line, so one error has one code on every surface.
var reasons = []struct {
	err    error
	reason string
}{
	{domain.ErrTimedOut, "timed_out"},
	{context.DeadlineExceeded, "timed_out"},
	{domain.ErrRedirected, "redirected"},
	{domain.ErrReplyTooLarge, "reply_too_large"},
	{domain.ErrUnexpectedResponse, "unexpected_response"},
	{ErrInvalidHeader, "invalid_header"},
	{ErrNotConfigured, "not_configured"},
	{domain.ErrKeyRejected, "key_rejected"},
	{domain.ErrAccessDenied, "access_denied"},
	{domain.ErrSourceRateLimited, "rate_limited"},
	{domain.ErrRetryAfterTooLong, "retry_after_too_long"},
	{domain.ErrNotFound, "not_found"},
	{domain.ErrBadRequest, "bad_request"},
	{domain.ErrUpstreamFailure, "upstream_failure"},
	{domain.ErrUnreachable, "unreachable"},
	{domain.ErrDocumentConflict, "document_conflict"},
	{domain.ErrUnresolvedName, "unresolved_name"},
	{domain.ErrOperationUnavailable, "operation_unavailable"},
	{domain.ErrUnsupportedOperation, "unsupported_operation"},
	{domain.ErrWrongOperationClass, "wrong_operation_class"},
	{domain.ErrWrongSourceKind, "wrong_source_kind"},
	{domain.ErrInvalidEndpoint, "invalid_endpoint"},
	{domain.ErrInvalidID, "invalid_id"},
	{domain.ErrInvalidLimit, "invalid_limit"},
	{domain.ErrInvalidOffset, "invalid_offset"},
	{domain.ErrInvalidInt, "invalid_number"},
	{domain.ErrInvalidText, "invalid_text"},
	{domain.ErrInvalidName, "invalid_name"},
	{domain.ErrMissingArgument, "missing_argument"},
	{domain.ErrUnknownArgument, "unknown_argument"},
}

// Code returns the stable code of a failed read: retriever.<area>.<reason>. The area is the source kind, except for a
// refused network, whose area is the connector. A code names the failure, never a value of the request.
func Code(kind string, err error) string {
	if errors.Is(err, ErrNetworkRefused) {
		return "retriever.connector.network_refused"
	}
	for _, r := range reasons {
		if errors.Is(err, r.err) {
			return "retriever." + kind + "." + r.reason
		}
	}
	return "retriever." + kind + ".failed"
}

// ExitStatus returns the exit status of a failed read. A nil error exits 0.
func ExitStatus(err error) int {
	if err == nil {
		return 0
	}
	if errors.Is(err, domain.ErrTimedOut) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, domain.ErrSourceRateLimited) {
		return ExitTimedOut
	}
	return ExitFailed
}
