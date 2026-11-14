//go:build e2e

package behaviour

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestExampleCatalogSourcesListMatchesTheGolden checks the document that the built binary prints for the example catalog
// with the three built-in kinds. The golden file holds the document the binary printed before the kind set opened, and the
// comparison is byte for byte, so a registration that changed the built-in output would fail here.
func TestExampleCatalogSourcesListMatchesTheGolden(t *testing.T) {
	catalog, err := filepath.Abs("../../examples/sources.example.yml")
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}
	// The env file is absent on purpose: the test reads no credential, and the catalog lists sources without a key.
	missingEnv := filepath.Join(t.TempDir(), "missing-env-file")
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(e2eBinary, "--catalog", catalog, "--env-file", missingEnv, "sources", "list")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("sources list: %v, stderr %q", err, stderr.String())
	}
	want, err := os.ReadFile("testdata/sources-list-example.golden")
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if !bytes.Equal(stdout.Bytes(), want) {
		t.Fatalf("sources list of the example catalog = %q, want the golden %q", stdout.String(), string(want))
	}
}
