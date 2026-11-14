package sources

import (
	"context"
	"errors"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// TestAdministerRefusesAKindWithNoAdminPort checks that a change to a source with no admin port is refused before any
// request, with the unsupported-source error.
func TestAdministerRefusesAKindWithNoAdminPort(t *testing.T) {
	svc, _ := adminService(t, domain.AccessAdmin, &fakeReader{}, nil)

	_, err := Administer(context.Background(), svc, domain.OpDocumentsArchive, strPtr("outline"),
		map[string]any{"documentId": "doc-1"})
	if !errors.Is(err, domain.ErrUnsupportedSource) {
		t.Fatalf("Administer without an admin port: %v, want ErrUnsupportedSource", err)
	}
}

// TestAdministerReturnsTheAdminPortError checks that a change the admin port refuses keeps the port's error, and that the
// port was asked exactly once.
func TestAdministerReturnsTheAdminPortError(t *testing.T) {
	admin := &fakeAdmin{err: domain.ErrDocumentNotFound}
	svc, _ := adminService(t, domain.AccessAdmin, &fakeReader{}, admin)

	_, err := Administer(context.Background(), svc, domain.OpDocumentsArchive, strPtr("outline"),
		map[string]any{"documentId": "doc-1"})
	if !errors.Is(err, domain.ErrDocumentNotFound) {
		t.Fatalf("Administer error = %v; want ErrDocumentNotFound", err)
	}
	if len(admin.ops) != 1 {
		t.Fatalf("the admin port was called %d times; want once", len(admin.ops))
	}
}
