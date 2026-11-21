//go:build integration

package integration_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
)

// TestRequestSafeguards_RealRedirectRefusedAfterOneRequest binds a read whose source answers with a redirect over real TLS to
// the redirect error, with exactly one request received by the server.
// @scenario specs/libs/retriever-engine/behaviours/request-safeguards.feature: A redirect in the reply is refused and only one request is sent
func TestRequestSafeguards_RealRedirectRefusedAfterOneRequest(t *testing.T) {
	lb := reintgStart(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Location", "https://elsewhere.example.test/v1/docs/doc-1")
		w.WriteHeader(http.StatusFound)
	})
	eng := reintgEngine(t, reintgSpec(t, reintgObjectOp), lb, engine.AllowPrivateNetwork())
	_, err := eng.Read(context.Background(), "example-wiki.docs.one", domain.NewArgs(map[string]any{"id": "doc-1"}))
	if !errors.Is(err, domain.ErrRedirected) {
		t.Fatalf("the error is %v, want the redirect error", err)
	}
	if n := lb.hits.Load(); n != 1 {
		t.Fatalf("the server received %d requests, want exactly one", n)
	}
}
