package outlinestub_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-testkit/outlinestub"
)

// TestServeHTTPHoldHookRunsBeforeTheWait checks the hold hook: a held request calls the hook once, before it waits, so a
// test can end the hold on its own clock. The hook here ends the request's context, so the wait returns at once.
func TestServeHTTPHoldHookRunsBeforeTheWait(t *testing.T) {
	s := newStub()
	s.SetFault(outlinestub.Fault{Hang: true})
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	s.SetHoldHook(func() {
		calls++
		cancel()
	})
	req := httptest.NewRequest(http.MethodPost, outlineURL("auth.info"), strings.NewReader("{}")).WithContext(ctx)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if calls != 1 {
		t.Fatalf("the hold hook ran %d times, want once", calls)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("a held request wrote %q, want nothing", rec.Body.String())
	}
}
