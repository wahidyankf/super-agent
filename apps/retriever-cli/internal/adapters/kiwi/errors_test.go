package kiwi

import (
	"errors"
	"net/http"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// TestRPCErrorMapsEachCodeAndMessage is the table of error codes, read by code and by message text. The message is
// matched and never shown, so each row names the sentinel the presenter prints.
func TestRPCErrorMapsEachCodeAndMessage(t *testing.T) {
	cases := []struct {
		name    string
		method  string
		code    int
		message string
		write   bool
		want    error
	}{
		{"invalid request", methodProductFilter, -32600, "Invalid Request", false, domain.ErrBadRequest},
		{"method not found", methodProductFilter, -32601, "Method not found", false, domain.ErrBadRequest},
		{"invalid params", methodProductFilter, -32602, "Invalid params", false, domain.ErrBadRequest},
		{"request code on a write", methodTestCaseUpdate, -32602, "Something is wrong.", true, domain.ErrBadRequest},
		{"authentication", methodProductFilter, -32603, "Authentication credentials were not provided", false, domain.ErrKeyRejected},
		{"login required", methodProductFilter, -32603, "login required for this call", false, domain.ErrKeyRejected},
		{"permission", methodProductFilter, -32603, "You do not have permission to read products", false, domain.ErrAccessDenied},
		{"not allowed", methodProductFilter, -32603, "This action is not allowed", false, domain.ErrAccessDenied},
		{"forbidden", methodProductFilter, -32603, "Access is forbidden for this user", false, domain.ErrAccessDenied},
		{"permission in upper case", methodProductFilter, -32603, "PERMISSION denied", false, domain.ErrAccessDenied},
		{"product not found", methodProductFilter, -32603, "Product not found", false, domain.ErrCollectionNotFound},
		{"case not found", methodTestCaseFilter, -32603, "TestCase matching query does not exist", false, domain.ErrDocumentNotFound},
		{"plan not found", methodTestPlanFilter, -32603, "Plan does not exist", false, domain.ErrDocumentNotFound},
		{"other not found", methodCategoryFilter, -32603, "Category not found", false, domain.ErrNotFound},
		{"plan missing on add-case", methodTestPlanAddCase, -32603, "Plan does not exist", true, domain.ErrDocumentNotFound},
		{"case missing on update", methodTestCaseUpdate, -32603, "TestCase matching query does not exist", true, domain.ErrDocumentNotFound},
		{"validation", methodTestCaseCreate, -32603, "This field is required: summary.", true, domain.ErrBadRequest},
		{"unique", methodTestCaseCreate, -32603, "Name must be unique.", true, domain.ErrBadRequest},
		{"unmatched read", methodProductFilter, -32603, "Something unexpected happened", false, domain.ErrUpstreamFailure},
		{"other code on a read", methodProductFilter, -32000, "Custom server error", false, domain.ErrUpstreamFailure},
		{"unmatched write", methodTestCaseCreate, -32603, "Something unexpected happened", true, domain.ErrWriteOutcomeUnknown},
		{"other code on a write", methodTestPlanAddCase, -32000, "Custom server error", true, domain.ErrWriteOutcomeUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := rpcError(tc.method, tc.code, tc.message, tc.write)
			if !errors.Is(err, tc.want) {
				t.Fatalf("rpcError(%s, %d, %q) = %v, want %v", tc.method, tc.code, tc.message, err, tc.want)
			}
		})
	}
}

// TestStatusErrorMapsEachHTTPStatus checks the HTTP status rules that come before the JSON-RPC envelope is read.
func TestStatusErrorMapsEachHTTPStatus(t *testing.T) {
	cases := []struct {
		status int
		write  bool
		want   error
	}{
		{http.StatusFound, false, domain.ErrRedirected},
		{http.StatusMovedPermanently, true, domain.ErrRedirected},
		{http.StatusBadRequest, false, domain.ErrBadRequest},
		{http.StatusUnauthorized, false, domain.ErrKeyRejected},
		{http.StatusForbidden, false, domain.ErrAccessDenied},
		{http.StatusNotFound, false, domain.ErrUnexpectedResponse},
		{http.StatusMethodNotAllowed, false, domain.ErrUnexpectedResponse},
		{http.StatusTeapot, false, domain.ErrUnexpectedResponse},
		{http.StatusInternalServerError, false, domain.ErrUpstreamFailure},
		{http.StatusServiceUnavailable, false, domain.ErrUpstreamFailure},
		{http.StatusInternalServerError, true, domain.ErrWriteOutcomeUnknown},
		{http.StatusServiceUnavailable, true, domain.ErrWriteOutcomeUnknown},
		{http.StatusUnauthorized, true, domain.ErrKeyRejected},
		{http.StatusForbidden, true, domain.ErrAccessDenied},
	}
	for _, tc := range cases {
		err := statusError(tc.status, tc.write)
		if !errors.Is(err, tc.want) {
			t.Errorf("statusError(%d, write=%v) = %v, want %v", tc.status, tc.write, err, tc.want)
		}
	}
}

// TestKindOfEveryFailureIsKiwi checks that the boundary wraps each failure in the Kiwi kind, so the presenters name Kiwi.
func TestKindOfEveryFailureIsKiwi(t *testing.T) {
	wrapped := kindErr(domain.ErrKeyRejected)
	var kindError domain.KindError
	if !errors.As(wrapped, &kindError) || kindError.Kind != domain.KindKiwi {
		t.Fatalf("kindErr = %#v, want a KindError of kiwi", wrapped)
	}
	if !errors.Is(wrapped, domain.ErrKeyRejected) {
		t.Fatalf("kindErr does not match ErrKeyRejected")
	}
	if kindErr(nil) != nil {
		t.Fatalf("kindErr(nil) is not nil")
	}
}
