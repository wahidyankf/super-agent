package registry

import (
	"errors"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// TestReaderReportsABuildFailure checks that a source which cannot be built is reported by Reader, not hidden as a missing
// capability.
func TestReaderReportsABuildFailure(t *testing.T) {
	src := sourceAt(t, "outline", "docs.example.com", "A_API_KEY")
	if _, err := New(map[domain.SourceKind]Builder{}).Reader(src); !errors.Is(err, domain.ErrUnsupportedSource) {
		t.Fatalf("Reader error = %v, want ErrUnsupportedSource", err)
	}
}
