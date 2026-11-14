package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// errNotConfigured reports a command whose collaborator the composition root did not supply.
var errNotConfigured = errors.New("retriever is not configured for this command")

// setupError is a setup error from the composition root. Its text is the root's own text about the local wiring, such as
// the working directory, so it is printed as it is and the user sees why the wiring is missing.
type setupError struct {
	err error
}

func (e setupError) Error() string { return e.err.Error() }

func (e setupError) Unwrap() error { return e.err }

// usageError is a mistake in the command line that the CLI found itself, such as a --port value out of range. Its text is
// written by the CLI and names only the flag and the value the user typed, so publicMessage shows it as it is.
type usageError struct {
	msg string
}

func (e usageError) Error() string { return e.msg }

// formatFor returns the output format a --output value names. Run refuses any value other than text or json before a
// command runs, so an empty value reads as text.
func formatFor(name string) domain.OutputFormat {
	if name == "json" {
		return domain.FormatJSON
	}
	return domain.FormatText
}

// writeJSON writes one JSON document, followed by a newline, to w.
func writeJSON(w io.Writer, v any) error {
	return json.NewEncoder(w).Encode(v)
}

// publicMessage maps a use-case error onto text that is safe to print. Only fixed texts reach the terminal, plus the
// texts of the typed errors that name a source or a setting and carry no secret. Any other error gets a generic text,
// because its raw text can carry upstream detail or a local path.
func publicMessage(err error) string {
	var usage usageError
	if errors.As(err, &usage) {
		return usage.msg
	}
	var refused refusedError
	if errors.As(err, &refused) {
		return refused.Error()
	}
	var catalog domain.CatalogError
	if errors.As(err, &catalog) {
		return catalog.Error()
	}
	var setup setupError
	if errors.As(err, &setup) {
		return setup.Error()
	}
	var missing domain.MissingKeyError
	if errors.As(err, &missing) {
		return missing.Error()
	}
	var invalidIdentity domain.InvalidIdentityError
	if errors.As(err, &invalidIdentity) {
		return invalidIdentity.Error()
	}
	var unknown domain.UnknownSourceError
	if errors.As(err, &unknown) {
		return fmt.Sprintf("unknown source %q; run retriever sources list", unknown.ID)
	}
	var ambiguous domain.AmbiguousSourceError
	if errors.As(err, &ambiguous) {
		return fmt.Sprintf("more than one %s source exists (%s); pass --source <id>",
			ambiguous.Kind, strings.Join(ambiguous.IDs, ", "))
	}
	var unavailable domain.OperationUnavailableError
	if errors.As(err, &unavailable) {
		return unavailable.Error()
	}
	var noSource domain.NoSourceOfKindError
	if errors.As(err, &noSource) {
		return noSource.Error()
	}
	if errors.Is(err, domain.ErrNoSourceOfKind) {
		// A raw sentinel names no kind, so it gets the default kind's text.
		return "no outline source is configured"
	}
	if errors.Is(err, errNotConfigured) {
		return err.Error()
	}
	var kindErr domain.KindError
	if errors.As(err, &kindErr) {
		if text, ok := kindErr.PublicText(); ok {
			return text
		}
	}
	for _, typed := range domain.PublicErrors() {
		if errors.Is(err, typed) {
			return typed.Error()
		}
	}
	return "unexpected error"
}

// errorReasons maps every public sentinel onto the reason of its JSON error code. Each entry has its own code, and a test
// checks that the codes are distinct and that none is the generic code.
var errorReasons = []struct {
	err    error
	reason string
}{
	{domain.ErrInvalidCollectionID, "invalid_collection_id"},
	{domain.ErrInvalidDocumentID, "invalid_document_id"},
	{domain.ErrExportWriteFailed, "export_write_failed"},
	{domain.ErrUnsupportedSource, "unsupported_source"},
	{domain.ErrCollectionNotFound, "collection_not_found"},
	{domain.ErrDocumentNotFound, "document_not_found"},
	{domain.ErrUnresolvedName, "unresolved_name"},
	{domain.ErrNotFound, "not_found"},
	{domain.ErrInvalidShareID, "invalid_share_id"},
	{domain.ErrExactlyOneArgument, "exactly_one_argument"},
	{domain.ErrKeyRejected, "key_rejected"},
	{domain.ErrAccessDenied, "access_denied"},
	{domain.ErrSourceRateLimited, "rate_limited"},
	{domain.ErrRetryAfterTooLong, "retry_after_too_long"},
	{domain.ErrRedirected, "redirected"},
	{domain.ErrTimedOut, "timed_out"},
	{domain.ErrHTTPSRequired, "https_required"},
	{domain.ErrUnreachable, "unreachable"},
	{domain.ErrUpstreamFailure, "upstream_failure"},
	{domain.ErrUnexpectedResponse, "unexpected_response"},
	{domain.ErrPageCapReached, "page_cap_reached"},
	{domain.ErrReplyTooLarge, "reply_too_large"},
	{domain.ErrInvalidSourceID, "invalid_source_id"},
	{domain.ErrWrongSourceKind, "wrong_source_kind"},
	{domain.ErrUnknownSourceKind, "unknown_source_kind"},
	{domain.ErrInvalidTag, "invalid_tag"},
	{domain.ErrTooManyTags, "too_many_tags"},
	{domain.ErrInvalidLimit, "invalid_limit"},
	{domain.ErrInvalidOffset, "invalid_offset"},
	{domain.ErrInvalidQuery, "invalid_query"},
	{domain.ErrInvalidOutputFormat, "invalid_output_format"},
	{domain.ErrOperationUnavailable, "operation_unavailable"},
	{domain.ErrUnsupportedOperation, "unsupported_operation"},
	{domain.ErrBadRequest, "bad_request"},
	{domain.ErrInvalidID, "invalid_id"},
	{domain.ErrInvalidUUID, "invalid_uuid"},
	{domain.ErrInvalidName, "invalid_name"},
	{domain.ErrInvalidText, "invalid_text"},
	{domain.ErrStandardInputUnsupported, "standard_input_unsupported"},
	{domain.ErrInvalidInt, "invalid_number"},
	{domain.ErrInvalidBool, "invalid_bool"},
	{domain.ErrInvalidEnum, "invalid_enum"},
	{domain.ErrInvalidIDList, "invalid_id_list"},
	{domain.ErrInvalidTime, "invalid_time"},
	{domain.ErrInvalidURL, "invalid_url"},
	{domain.ErrInvalidIndex, "invalid_index"},
	{domain.ErrInvalidColor, "invalid_color"},
	{domain.ErrInvalidIcon, "invalid_icon"},
	{domain.ErrInvalidScore, "invalid_score"},
	{domain.ErrMissingArgument, "missing_argument"},
	{domain.ErrUnknownArgument, "unknown_argument"},
	{domain.ErrSourceRequired, "source_required"},
	{domain.ErrWrongOperationClass, "wrong_operation_class"},
	{domain.ErrWriteOutcomeUnknown, "write_outcome_unknown"},
	{domain.ErrInvalidEmail, "invalid_email"},
	{domain.ErrInvalidReason, "invalid_reason"},
	{domain.ErrDocumentConflict, "document_conflict"},
	{errNotConfigured, "not_configured"},
	{domain.ErrCatalogMissing, "catalog_missing"},
}

// errorCode returns the code of a failed command: retriever.<area>.<reason>. The area is the source kind of a kind-named
// error, and otherwise the area of the command. A code names the failure, never a value. The generic
// code is used only for an error that no entry names.
func errorCode(err error, area string) string {
	prefix := "retriever." + areaFor(err, area) + "."
	var usage usageError
	if errors.As(err, &usage) {
		return prefix + "usage_invalid_argument"
	}
	var refused refusedError
	if errors.As(err, &refused) {
		return prefix + "refused"
	}
	var missing domain.MissingKeyError
	if errors.As(err, &missing) {
		return prefix + "key_missing"
	}
	var invalidIdentity domain.InvalidIdentityError
	if errors.As(err, &invalidIdentity) {
		return prefix + "identity_invalid"
	}
	var unknown domain.UnknownSourceError
	if errors.As(err, &unknown) {
		return prefix + "unknown_source"
	}
	var ambiguous domain.AmbiguousSourceError
	if errors.As(err, &ambiguous) {
		return prefix + "ambiguous_source"
	}
	var unavailable domain.OperationUnavailableError
	if errors.As(err, &unavailable) {
		return prefix + "operation_unavailable"
	}
	for _, e := range errorReasons {
		if errors.Is(err, e.err) {
			return prefix + e.reason
		}
	}
	var catalog domain.CatalogError
	if errors.As(err, &catalog) {
		return prefix + "catalog_invalid"
	}
	return prefix + "failed"
}

// areaFor returns the namespace of an error's code: the kind of a kind-named error, or the command's area.
func areaFor(err error, area string) string {
	var named domain.KindError
	if errors.As(err, &named) {
		return named.Kind.String()
	}
	return area
}

// retryable reports whether a failure may succeed when it is tried again later: a rate limit, a timeout, or a network
// failure before any reply. The JSON error body carries the flag, so a script can wait and retry.
func retryable(err error) bool {
	return errors.Is(err, domain.ErrSourceRateLimited) || errors.Is(err, domain.ErrTimedOut) ||
		errors.Is(err, domain.ErrUnreachable)
}

// withKind names the source kind of an error from one command, so its text and its JSON code carry that kind. An Outline error is returned as it is, so every Outline text and code stays the same. An
// error that already names a kind keeps it.
func withKind(kind domain.SourceKind, err error) error {
	if err == nil || kind == domain.KindOutline {
		return err
	}
	var named domain.KindError
	if errors.As(err, &named) {
		return err
	}
	return domain.KindError{Kind: kind, Err: err}
}
