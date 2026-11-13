//go:build integration

package catalog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// loadRealFile loads the catalog file at path from the real file system.
func loadRealFile(t *testing.T, path string) (domain.Catalog, error) {
	t.Helper()
	l, err := New(path)
	if err != nil {
		t.Fatalf("New(%q): %v", path, err)
	}
	return l.Load()
}

func TestSymlinkAndFolderAreRefused(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "elsewhere.yml")
	if err := os.WriteFile(target, []byte(oneSource), 0o600); err != nil {
		t.Fatalf("write target: %v", err)
	}
	link := filepath.Join(dir, "sources.yml")
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	if _, err := loadRealFile(t, link); err == nil || !strings.HasPrefix(err.Error(), catalogPrefix) {
		t.Fatalf("symlink error = %v", err)
	}

	folder := filepath.Join(t.TempDir(), "sources.yml")
	if err := os.Mkdir(folder, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if _, err := loadRealFile(t, folder); err == nil || !strings.Contains(err.Error(), "must be a regular file") {
		t.Fatalf("folder error = %v", err)
	}
}

func TestUnreadableCatalogIsAnError(t *testing.T) {
	file := filepath.Join(t.TempDir(), "sources.yml")
	if err := os.WriteFile(file, []byte(oneSource), 0o600); err != nil {
		t.Fatalf("write catalog: %v", err)
	}
	if err := os.Chmod(file, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(file, 0o600) })
	if _, err := loadRealFile(t, file); err == nil || !strings.HasPrefix(err.Error(), catalogPrefix) {
		t.Fatalf("unreadable catalog error = %v", err)
	}
}
