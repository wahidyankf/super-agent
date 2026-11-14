package replay

import (
	"errors"
	"net/http"
	"testing"
)

// TestEncodeBodyIsEmptyForNoBody checks that an example with no body encodes to nothing, and that a body encodes to its
// JSON text.
func TestEncodeBodyIsEmptyForNoBody(t *testing.T) {
	if got, err := encodeBody(nil); err != nil || got != nil {
		t.Fatalf("encodeBody(nil) = %q, %v; want nothing", got, err)
	}
	got, err := encodeBody(map[string]any{"id": "doc-1"})
	if err != nil || string(got) != `{"id":"doc-1"}` {
		t.Fatalf("encodeBody = %q, %v; want the JSON text", got, err)
	}
	if _, err := encodeBody(make(chan int)); err == nil {
		t.Fatal("a body that cannot be encoded was accepted")
	}
}

// TestNextStepRepeatsTheLastStep checks the order of a script: each step is used once, and the last one stays.
func TestNextStepRepeatsTheLastStep(t *testing.T) {
	s := &Stub{scripts: map[string][]step{}}
	boom := errors.New("scripted")
	s.scripts["GET /a"] = []step{{reply: Reply{Status: 500}}, {err: boom}}
	if st, ok := s.nextStep("GET /a"); !ok || st.reply.Status != 500 {
		t.Fatalf("first step = %+v, %v; want status 500", st, ok)
	}
	for i := 0; i < 2; i++ {
		if st, ok := s.nextStep("GET /a"); !ok || !errors.Is(st.err, boom) {
			t.Fatalf("step %d = %+v, %v; want the last step to repeat", i, st, ok)
		}
	}
	if _, ok := s.nextStep("GET /none"); ok {
		t.Fatal("a key with no script reported a step")
	}
}

// TestResponseCopiesTheCannedHeaders checks that a reply with no header still has a header map, and that the reply
// keeps the canned status and body.
func TestResponseCopiesTheCannedHeaders(t *testing.T) {
	resp := response(nil, Reply{Status: http.StatusCreated, Body: []byte("x")})
	if resp.StatusCode != http.StatusCreated || resp.Header == nil || resp.ContentLength != 1 {
		t.Fatalf("response = %d, header %v, length %d", resp.StatusCode, resp.Header, resp.ContentLength)
	}
}
