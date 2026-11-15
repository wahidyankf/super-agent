package engine

import (
	"context"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// TestRunFallbackTargetWithoutATargetFailsClosed checks the defensive guard of the fallback target: an engine whose
// operations hold no target, which the loader refuses before any engine is built, still fails closed with
// unsupported operation instead of running a request it cannot render.
func TestRunFallbackTargetWithoutATargetFailsClosed(t *testing.T) {
	eng := &Engine{ops: map[domain.OperationID]spec.Operation{}}
	op := spec.Operation{
		ID:       "example-wiki.docs.meta",
		Fallback: &spec.Fallback{Operation: "example-wiki.docs.export", Fill: []string{"markdown"}},
	}
	budget := 1024
	_, err := eng.runFallbackTarget(context.Background(), op, domain.NewArgs(nil), domain.Record{}, false, &budget)
	if err != domain.ErrUnsupportedOperation {
		t.Fatalf("err = %v, want ErrUnsupportedOperation", err)
	}
}
