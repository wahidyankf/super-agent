//go:build integration

package integration_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/catalog"
)

// loaderFor returns the catalog loader of the file sources.yml in dir, over the real folder and the real file metadata.
func loaderFor(t *testing.T, dir string, kinds domain.KindSet) *catalog.Loader {
	t.Helper()
	l, err := catalog.NewWithFiles(filepath.Join(dir, "sources.yml"),
		catalog.Files{FS: os.DirFS(dir), SameFile: os.SameFile}, kinds)
	if err != nil {
		t.Fatalf("NewWithFiles: %v", err)
	}
	return l
}

// @scenario specs/libs/retriever-runtime/behaviours/catalog-loading.feature: A catalog file that is absent names where to set the path
func TestACatalogFileThatIsAbsentNamesWhereToSetThePath(t *testing.T) {
	dir := t.TempDir()
	cat, err := loaderFor(t, dir, domain.BuiltinKinds()).Load()
	if err == nil || !strings.Contains(err.Error(), "not found: set --catalog or RETRIEVER_CATALOG, or create ./sources.yml") {
		t.Fatalf("Load error = %v, want the not-found rule", err)
	}
	if len(cat.All()) != 0 {
		t.Fatalf("catalog holds sources, want none")
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/catalog-loading.feature: A catalog file that is present is the whole catalog
func TestACatalogFileThatIsPresentIsTheWholeCatalog(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"sources.yml": `version: 1
sources:
  - id: docs
    kind: outline
    endpoint: https://docs.example.test
    credential_env: EXAMPLE_OUTLINE_API_KEY
  - id: wiki
    kind: outline
    endpoint: https://wiki.example.test
    credential_env: EXAMPLE_WIKI_API_KEY
`})
	cat, err := loaderFor(t, dir, domain.BuiltinKinds()).Load()
	if err != nil {
		t.Fatalf("Load error = %v", err)
	}
	if len(cat.All()) != 2 {
		t.Fatalf("catalog holds %d sources, want docs and wiki only", len(cat.All()))
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/catalog-loading.feature: A catalog path that is a symbolic link or a folder is refused
func TestACatalogPathThatIsASymbolicLinkOrAFolderIsRefused(t *testing.T) {
	cases := []struct {
		name  string
		setup func(t *testing.T, dir string)
	}{
		{"symbolic link", func(t *testing.T, dir string) {
			writeFiles(t, dir, map[string]string{"real.yml": outlineDocsYAML})
			if err := os.Symlink(filepath.Join(dir, "real.yml"), filepath.Join(dir, "sources.yml")); err != nil {
				t.Fatalf("symlink: %v", err)
			}
		}},
		{"folder", func(t *testing.T, dir string) {
			if err := os.Mkdir(filepath.Join(dir, "sources.yml"), 0o755); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			tc.setup(t, dir)
			_, err := loaderFor(t, dir, domain.BuiltinKinds()).Load()
			wantContains(t, err, "must be a regular file, not a symlink or a folder")
		})
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/catalog-loading.feature: A catalog at the size limit is read and one byte over is refused
func TestACatalogAtTheSizeLimitIsReadAndOneByteOverIsRefused(t *testing.T) {
	cases := []struct {
		name string
		size int
		ok   bool
	}{
		{"65536 bytes", 65536, true},
		{"65537 bytes", 65537, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, map[string]string{"sources.yml": padToSize(outlineDocsYAML, tc.size)})
			cat, err := loaderFor(t, dir, domain.BuiltinKinds()).Load()
			if !tc.ok {
				wantContains(t, err, "must be at most 64 KiB")
				return
			}
			if err != nil || len(cat.All()) != 1 {
				t.Fatalf("Load = %d sources, error %v; want the catalog loaded", len(cat.All()), err)
			}
		})
	}
}

// wantContains fails the test unless err is non-nil and its text holds the rule.
func wantContains(t *testing.T, err error, rule string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), rule) {
		t.Fatalf("error = %v, want a rule holding %q", err, rule)
	}
}
