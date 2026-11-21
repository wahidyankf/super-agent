package kiwistub

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestServeHTTPHoldHookRunsBeforeTheWait checks the hold hook over a socket path: a held request calls the hook once,
// before it waits, so a test can end the hold on its own clock. The hook here ends the request's context.
func TestServeHTTPHoldHookRunsBeforeTheWait(t *testing.T) {
	s := newStub(WithDataset(fixture()))
	s.SetFault(Fault{Never: true})
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	s.SetHoldHook(func() {
		calls++
		cancel()
	})
	req := httptest.NewRequest(http.MethodPost, Endpoint, strings.NewReader("{}")).WithContext(ctx)
	req.Header.Set("Authorization", "Bearer "+testKey)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if calls != 1 {
		t.Fatalf("the hold hook ran %d times, want once", calls)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("a held request wrote %q, want nothing", rec.Body.String())
	}
}
