package unit_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
)

// TestLoopbackReadIsRefusedWithoutTheCatalogSetting reads a host that resolves to the loopback address, with the catalog
// setting off. The dialer must never be run.
//
// @scenario specs/libs/retriever-engine/behaviours/loopback-connections.feature: A read to a loopback test server is refused without the catalog setting and the server receives no request
func TestLoopbackReadIsRefusedWithoutTheCatalogSetting(t *testing.T) {
	model := rebindSpec(t, wireNetOps)
	n := &wireNetwork{answers: wireAnswers("127.0.0.1")}
	_, err := wireNetEngine(t, model, n).Read(context.Background(), "example-wiki.docs.get", wireNetDocArgs())
	if !errors.Is(err, engine.ErrNetworkRefused) {
		t.Fatalf("err = %v, want the network refusal", err)
	}
	if len(n.dialed()) != 0 {
		t.Fatalf("the loopback address was dialed: %v", n.dialed())
	}
}

// TestLoopbackReadSendsTheKeyOverTLS reads with the catalog setting allowing private networks, and checks the request
// that leaves: its scheme is https, and the key is in the bearer header.
//
// @scenario specs/libs/retriever-engine/behaviours/loopback-connections.feature: A read to a loopback test server connects over TLS with the catalog setting and sends the key in the authorization header
func TestLoopbackReadSendsTheKeyOverTLS(t *testing.T) {
	model := rebindSpec(t, wireNetOps)
	rt := rebindReplies(rebindJSON(200, `{"title":"First document"}`))
	rec, err := rebindEngine(t, model, rt, &rebindClock{}, engine.AllowPrivateNetwork()).Read(context.Background(), "example-wiki.docs.get", wireNetDocArgs())
	if err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	if got := rec.Str("title"); got != "First document" {
		t.Fatalf("title = %q, want the reply", got)
	}
	seen := rt.seen()
	if len(seen) != 1 {
		t.Fatalf("got %d requests, want one", len(seen))
	}
	if seen[0].url.Scheme != "https" {
		t.Fatalf("the request scheme is %q, want https", seen[0].url.Scheme)
	}
	if want := "Bearer " + rebindKey; seen[0].header.Get("Authorization") != want {
		t.Fatalf("authorization header = %q, want the key in the bearer form", seen[0].header.Get("Authorization"))
	}
}

// TestLoopbackRedirectIsRefusedAfterOneRequest answers the read with a real 302 redirect.
//
// @scenario specs/libs/retriever-engine/behaviours/loopback-connections.feature: A real 302 reply to a read is refused and the server receives one request
func TestLoopbackRedirectIsRefusedAfterOneRequest(t *testing.T) {
	model := rebindSpec(t, wireNetOps)
	rt := rebindReplies(rebindReply{status: http.StatusFound, header: http.Header{"Location": {"https://docs.example.test/v1/elsewhere"}}})
	_, err := wireRun(t, model, rt, "example-wiki.docs.get", wireNetDocArgs())
	if !errors.Is(err, domain.ErrRedirected) {
		t.Fatalf("err = %v, want ErrRedirected", err)
	}
	if n := len(rt.seen()); n != 1 {
		t.Fatalf("got %d requests, want exactly one", n)
	}
}

// TestLoopbackStallExitsWithTheTimeoutStatusAfterOneRequest answers the read with a stall, under a one-second time limit
// whose deadline has already passed.
//
// @scenario specs/libs/retriever-engine/behaviours/loopback-connections.feature: A real stall hits the spec time limit and exits with the timeout status without a second request
func TestLoopbackStallExitsWithTheTimeoutStatusAfterOneRequest(t *testing.T) {
	model := rebindDoc(t, wireReplace(t, rebindHeader, "timeout_seconds: 20", "timeout_seconds: 1")+wireNetOps)
	rt := rebindReplies(rebindReply{stall: true})
	eng := rebindEngine(t, model, rt, &rebindClock{expired: true})
	_, err := eng.Read(context.Background(), "example-wiki.docs.get", wireNetDocArgs())
	if got := engine.ExitStatus(err); got != engine.ExitTimedOut {
		t.Fatalf("exit status = %d (err %v), want the timeout status %d", got, err, engine.ExitTimedOut)
	}
	if n := len(rt.seen()); n != 1 {
		t.Fatalf("got %d requests, want exactly one", n)
	}
}

// TestLoopbackWriteTimeoutAfterTheRequestLeftIsUnknownAndSentOnce times out a write whose request reached the transport.
//
// @scenario specs/libs/retriever-engine/behaviours/loopback-connections.feature: A write whose connection times out after the request reached a local server is an unknown outcome and is sent once
func TestLoopbackWriteTimeoutAfterTheRequestLeftIsUnknownAndSentOnce(t *testing.T) {
	model := rebindDoc(t, wireReplace(t, rebindHeader, "timeout_seconds: 20", "timeout_seconds: 1")+wireNetOps)
	rt := rebindReplies(rebindReply{err: context.DeadlineExceeded})
	eng := rebindEngine(t, model, rt, &rebindClock{expired: true})
	_, err := eng.Write(context.Background(), "example-wiki.docs.create", domain.NewArgs(map[string]any{"title": "First document"}))
	if !errors.Is(err, domain.ErrWriteOutcomeUnknown) {
		t.Fatalf("err = %v, want ErrWriteOutcomeUnknown", err)
	}
	if n := len(rt.seen()); n != 1 {
		t.Fatalf("got %d requests, want exactly one", n)
	}
}

// TestLoopbackWriteIsSentOnceWithTheKeyAndMapsItsField answers the write with the created document.
//
// @scenario specs/libs/retriever-engine/behaviours/loopback-connections.feature: A write over TLS to a local server is sent once, carries the key in the authorization header, and maps its declared field
func TestLoopbackWriteIsSentOnceWithTheKeyAndMapsItsField(t *testing.T) {
	model := rebindSpec(t, wireNetOps)
	rt := rebindReplies(rebindJSON(http.StatusCreated, `{"id":"doc-1"}`))
	rec, err := rebindEngine(t, model, rt, &rebindClock{}, engine.AllowPrivateNetwork()).Write(context.Background(),
		"example-wiki.docs.create", domain.NewArgs(map[string]any{"title": "First document"}))
	if err != nil {
		t.Fatalf("the write fails: %v", err)
	}
	seen := rt.seen()
	if len(seen) != 1 {
		t.Fatalf("got %d requests, want exactly one", len(seen))
	}
	if want := "Bearer " + rebindKey; seen[0].header.Get("Authorization") != want {
		t.Fatalf("authorization header = %q, want the key in the bearer form", seen[0].header.Get("Authorization"))
	}
	if got := rec.Str("id"); got != "doc-1" {
		t.Fatalf("id = %q, want the declared field", got)
	}
}

// TestLoopbackHandshakeFailureIsUnreachableAndNothingIsWritten fails the dial of a write, which is where a handshake that
// fails before the connection is made ends.
//
// @scenario specs/libs/retriever-engine/behaviours/loopback-connections.feature: A TLS handshake that fails against a local server is a known failure and the write is not sent
func TestLoopbackHandshakeFailureIsUnreachableAndNothingIsWritten(t *testing.T) {
	model := rebindSpec(t, wireNetOps)
	n := &wireNetwork{answers: wireAnswers("93.184.216.34"), dialErr: errors.New("the certificate is not trusted")}
	_, err := wireNetEngine(t, model, n).Write(context.Background(), "example-wiki.docs.create", domain.NewArgs(map[string]any{"title": "First document"}))
	if !errors.Is(err, domain.ErrUnreachable) {
		t.Fatalf("err = %v, want ErrUnreachable", err)
	}
	if errors.Is(err, domain.ErrWriteOutcomeUnknown) {
		t.Fatal("a failed handshake was reported as an unknown outcome")
	}
	if len(n.dialed()) != 1 {
		t.Fatalf("dials = %v, want the one failed attempt", n.dialed())
	}
}
