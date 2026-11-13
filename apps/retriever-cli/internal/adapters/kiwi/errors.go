package kiwi

import (
	"net/http"
	"strings"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// JSON-RPC 2.0 error codes that name a bad request (ADR 009 section 5).
const (
	rpcInvalidRequest = -32600
	rpcMethodNotFound = -32601
	rpcInvalidParams  = -32602
	rpcServerError    = -32603
)

// kindErr wraps a failure in the Kiwi kind, so a presenter names Kiwi and not Outline (ADR 010 section 7). A nil error
// stays nil.
func kindErr(err error) error {
	if err == nil {
		return nil
	}
	return domain.KindError{Kind: domain.KindKiwi, Err: err}
}

// rpcError maps a JSON-RPC error object, received inside an HTTP 200 reply, to a sentinel. The code decides a request
// error; for a server error (-32603) the message is matched, never shown. write marks a method that changes Kiwi: a reply
// that matches nothing on a write has an unknown outcome, so the caller never retries it (ADR 009 section 5).
func rpcError(method string, code int, message string, write bool) error {
	switch code {
	case rpcInvalidRequest, rpcMethodNotFound, rpcInvalidParams:
		return domain.ErrBadRequest
	case rpcServerError:
		return serverErrorMessage(method, strings.ToLower(message), write)
	}
	return unmatched(write)
}

// serverErrorMessage maps a -32603 message onto a sentinel. The first matching group wins, in the order of ADR 009
// section 5. The lower-case message is only compared, never stored.
func serverErrorMessage(method, message string, write bool) error {
	switch {
	case containsAny(message, "authentication", "credentials", "token", "login required"):
		return domain.ErrKeyRejected
	case containsAny(message, "permission", "not allowed", "forbidden", "denied"):
		return domain.ErrAccessDenied
	case containsAny(message, "does not exist", "not found"):
		return notFoundFor(method)
	case containsAny(message, "invalid", "required", "this field", "must", "unique"):
		return domain.ErrBadRequest
	}
	return unmatched(write)
}

// unmatched is the sentinel of a JSON-RPC error that names no known case: an upstream failure for a read, and an unknown
// outcome for a write.
func unmatched(write bool) error {
	if write {
		return domain.ErrWriteOutcomeUnknown
	}
	return domain.ErrUpstreamFailure
}

// notFoundFor names the not-found sentinel of the object a method works on. A product lookup is a collection, a plan or a
// case lookup or change is a document, and any other lookup is the generic not-found.
func notFoundFor(method string) error {
	switch method {
	case methodProductFilter:
		return domain.ErrCollectionNotFound
	case methodTestPlanFilter, methodTestCaseFilter, methodTestCaseSortkeys, methodTestCaseUpdate, methodTestPlanAddCase:
		return domain.ErrDocumentNotFound
	}
	return domain.ErrNotFound
}

// statusError maps a non-2xx HTTP reply to a sentinel, before the body is read as JSON-RPC. A 429 is retried by the
// client and never reaches here. A 5xx reply to a write has an unknown outcome (ADR 009 section 5, as corrected).
func statusError(status int, write bool) error {
	switch {
	case status >= 300 && status < 400:
		return domain.ErrRedirected
	case status == http.StatusBadRequest:
		return domain.ErrBadRequest
	case status == http.StatusUnauthorized:
		return domain.ErrKeyRejected
	case status == http.StatusForbidden:
		return domain.ErrAccessDenied
	case status >= 500:
		return unmatched(write)
	}
	return domain.ErrUnexpectedResponse
}

// containsAny reports whether text holds any of the words.
func containsAny(text string, words ...string) bool {
	for _, word := range words {
		if strings.Contains(text, word) {
			return true
		}
	}
	return false
}
