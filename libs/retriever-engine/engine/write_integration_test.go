//go:build integration

package engine_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
)

// TestS3_AWriteTimeoutAfterTheRequestLeftIsNeverSentAgainOnARealConnection checks the unknown outcome on a real connection.
// The server reads nothing and answers nothing until the test ends, and the spec lowers the time limit to one second. The
// write reports an unknown outcome with exit status 2, and the server receives exactly one request.
func TestS3_AWriteTimeoutAfterTheRequestLeftIsNeverSentAgainOnARealConnection(t *testing.T) {
	release := make(chan struct{})
	lb := startLoopback(t, func(http.ResponseWriter, *http.Request) {
		<-release
	})
	t.Cleanup(func() { close(release) })
	s := loadSpecFrom(t, replaceIn(t, wikiYAML, "timeout_seconds: 20", "timeout_seconds: 1"))
	eng := realEngine(t, s, lb, engine.AllowPrivateNetwork())
	_, err := eng.Write(context.Background(), "example-wiki.docs.create", domain.NewArgs(createArgs))
	if !errors.Is(err, domain.ErrWriteOutcomeUnknown) {
		t.Fatalf("err = %v, want ErrWriteOutcomeUnknown", err)
	}
	if got := engine.ExitStatus(err); got != engine.ExitFailed {
		t.Fatalf("exit status = %d, want %d", got, engine.ExitFailed)
	}
	if n := lb.hits.Load(); n != 1 {
		t.Fatalf("the timed out write reached the server %d times, want 1: it is never sent again", n)
	}
}

// TestWriteOverRealTLSIsSentOnceAndMapped checks a write that the real server accepts: one request over TLS, with the key
// in the authorization header, and the declared field mapped.
func TestWriteOverRealTLSIsSentOnceAndMapped(t *testing.T) {
	var auth string
	lb := startLoopback(t, func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"doc-2"}`))
	})
	eng := realEngine(t, loadSpec(t), lb, engine.AllowPrivateNetwork())
	rec, err := eng.Write(context.Background(), "example-wiki.docs.create", domain.NewArgs(createArgs))
	if err != nil {
		t.Fatalf("Write over TLS: %v", err)
	}
	if got := rec.Str("id"); got != "doc-2" {
		t.Fatalf("id = %q, want doc-2", got)
	}
	if lb.hits.Load() != 1 || auth != "Bearer "+fixtureKey {
		t.Fatalf("hits = %d, authorization = %q; want one request with the bearer key", lb.hits.Load(), auth)
	}
}

// TestS5_AWriteHandshakeFailureOnARealConnectionIsKnown checks a real handshake that fails. The client does not trust the
// test certificate, so the handshake fails before any request leaves. The error is the unreachable error, not an unknown
// outcome, and the server receives nothing.
func TestS5_AWriteHandshakeFailureOnARealConnectionIsKnown(t *testing.T) {
	lb := startLoopback(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	eng, err := engine.New(loadSpec(t), testSource(t), engine.Ports{
		Credentials: &fakeCredentials{},
		Clock:       realClock{},
		Network:     lb.realNetwork(),
	}, engine.AllowPrivateNetwork())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = eng.Write(context.Background(), "example-wiki.docs.create", domain.NewArgs(createArgs))
	if !errors.Is(err, domain.ErrUnreachable) {
		t.Fatalf("err = %v, want ErrUnreachable", err)
	}
	if errors.Is(err, domain.ErrWriteOutcomeUnknown) {
		t.Fatal("a handshake failure reported an unknown outcome, but no request was sent")
	}
	if n := lb.hits.Load(); n != 0 {
		t.Fatalf("the server received %d requests after a failed handshake, want 0", n)
	}
}
