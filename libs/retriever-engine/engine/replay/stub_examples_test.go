package replay_test

import (
	"io"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine/replay"
)

// TestStubKeepsTheFirstExampleOfAKey checks that two read examples with the same request answer with the first one, so
// the stub never depends on the order of a second example.
func TestStubKeepsTheFirstExampleOfAKey(t *testing.T) {
	s := fixture(t)
	first := s.Operations[0].Examples[0]
	second := first
	second.Response.Body = map[string]any{"items": []any{}}
	s.Operations[0].Examples = append(s.Operations[0].Examples, second)
	stub, err := replay.New(s)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	resp, err := get(t, stub, "/v1/docs?limit=20")
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `"doc-1"`) {
		t.Fatalf("body = %s, want the first example", body)
	}
}

// TestStubRefusesAnExampleBodyThatWillNotEncode checks that an example whose body has no JSON form fails New, so the stub
// never answers with a body it cannot send.
func TestStubRefusesAnExampleBodyThatWillNotEncode(t *testing.T) {
	s := fixture(t)
	s.Operations[0].Examples[0].Response.Body = make(chan int)
	if _, err := replay.New(s); err == nil {
		t.Fatal("New accepted an example body with no JSON form")
	}
}
