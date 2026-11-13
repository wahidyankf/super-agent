package sources

import (
	"context"
	"errors"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// TestAdministerReturnsTheAdminPortError checks that a change the admin port refuses keeps the port's error, and that the
// port was asked exactly once.
func TestAdministerReturnsTheAdminPortError(t *testing.T) {
	reader := &fakeReader{reply: targetDocument()}
	admin := &fakeAdmin{err: domain.ErrDocumentNotFound}
	svc, _ := adminService(t, domain.AccessAdmin, reader, admin)

	_, err := Administer(context.Background(), svc, domain.OpDocumentsArchive, strPtr("outline"),
		map[string]any{"documentId": "doc-1"}, Confirmation{Yes: true})
	if !errors.Is(err, domain.ErrDocumentNotFound) {
		t.Fatalf("Administer error = %v; want ErrDocumentNotFound", err)
	}
	if len(admin.ops) != 1 {
		t.Fatalf("the admin port was called %d times; want once", len(admin.ops))
	}
}
