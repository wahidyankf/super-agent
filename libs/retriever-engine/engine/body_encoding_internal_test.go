package engine

import (
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// TestRenderRequestRefusesABodyWithNoJSONForm checks that a body which cannot be encoded fails the render, so no request
// is sent with a body that is not JSON. A spec cannot hold such a value, so the test builds the model by hand.
func TestRenderRequestRefusesABodyWithNoJSONForm(t *testing.T) {
	o := spec.Operation{Method: "POST", Path: "/docs", Body: map[string]any{"x": make(chan int)}}
	if _, err := renderRequest("/v1", o, domain.NewArgs(nil), nil); err == nil {
		t.Fatal("a body with no JSON form was rendered")
	}
}
