package engine_test

import (
	"context"
	"errors"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// TestAFieldThatHoldsAnObjectFailsTheRead checks a declared field whose value is an object: a field is a scalar, so the
// read fails and returns no record.
func TestAFieldThatHoldsAnObjectFailsTheRead(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpec(t))
	stub.Script("GET", "/v1/docs?limit=20", page(`{"items":[{"id":"doc-1","title":{"a":1}}],"next_cursor":null}`))
	if _, err := eng.Read(context.Background(), "example-wiki.docs.list", domain.NewArgs(nil)); !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
	}
}
