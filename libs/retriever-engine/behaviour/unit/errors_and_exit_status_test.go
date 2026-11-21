package unit_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
)

// TestFailureCarriesTheCodeOfItsKindAndReason reads a document that the source answers with not found.
//
// @scenario specs/libs/retriever-engine/behaviours/errors-and-exit-status.feature: A failure carries a stable code that names its source kind and its reason
func TestFailureCarriesTheCodeOfItsKindAndReason(t *testing.T) {
	model := rebindSpec(t, wireNetOps)
	rt := rebindReplies(rebindJSON(http.StatusNotFound, `{}`))
	_, err := wireRun(t, model, rt, "example-wiki.docs.get", wireNetDocArgs())
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	if got, want := engine.Code(rebindKind, err), "retriever.example-wiki.not_found"; got != want {
		t.Fatalf("code = %q, want %q", got, want)
	}
}

// TestRefusedNetworkCarriesTheConnectorCode reads a host whose address the network guard refuses.
//
// @scenario specs/libs/retriever-engine/behaviours/errors-and-exit-status.feature: A refused network connection carries a code that names the connector area
func TestRefusedNetworkCarriesTheConnectorCode(t *testing.T) {
	model := rebindSpec(t, wireNetOps)
	n := &wireNetwork{answers: wireAnswers(ipv4Text(10, 0, 0, 1))}
	_, err := wireNetEngine(t, model, n).Read(context.Background(), "example-wiki.docs.get", wireNetDocArgs())
	if !errors.Is(err, engine.ErrNetworkRefused) {
		t.Fatalf("err = %v, want the network refusal", err)
	}
	code := engine.Code(rebindKind, err)
	if !strings.HasPrefix(code, "retriever.connector.") {
		t.Fatalf("code = %q, want the connector area", code)
	}
	if want := "retriever.connector.network_refused"; code != want {
		t.Fatalf("code = %q, want %q", code, want)
	}
}

// TestExitStatusOfEachFailureKind runs one real failure of each kind through a read, and reads its exit status.
//
// @scenario specs/libs/retriever-engine/behaviours/errors-and-exit-status.feature: A failure of kind <outcome> maps to exit status <status>
func TestExitStatusOfEachFailureKind(t *testing.T) {
	cases := []struct {
		outcome string
		reply   rebindReply
		status  int
	}{
		{"a timeout", rebindReply{err: context.DeadlineExceeded}, 124},
		{"a rate-limit wait", rebindReply{status: http.StatusTooManyRequests, header: http.Header{"Retry-After": {"abc"}}}, 124},
		{"any other failure", rebindJSON(http.StatusNotFound, `{}`), 2},
	}
	model := rebindSpec(t, wireNetOps)
	for _, c := range cases {
		t.Run(c.outcome, func(t *testing.T) {
			rt := rebindReplies(c.reply)
			_, err := wireRun(t, model, rt, "example-wiki.docs.get", wireNetDocArgs())
			if err == nil {
				t.Fatal("the read did not fail")
			}
			if got := engine.ExitStatus(err); got != c.status {
				t.Fatalf("exit status = %d, want %d", got, c.status)
			}
		})
	}
	if engine.ExitStatus(nil) != 0 {
		t.Fatal("a nil error did not exit 0")
	}
}
