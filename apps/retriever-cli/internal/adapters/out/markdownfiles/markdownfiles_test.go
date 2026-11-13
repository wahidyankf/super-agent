package markdownfiles

import (
	"context"
	"testing"
)

// TestWriteRefusesAnEmptyFolder checks that a sink with no folder refuses to write, rather than writing to the working
// directory.
func TestWriteRefusesAnEmptyFolder(t *testing.T) {
	if _, err := New("").Write(context.Background(), "doc.md", []byte("x")); err == nil {
		t.Fatal("Write with no folder succeeded; want a refusal")
	}
}
