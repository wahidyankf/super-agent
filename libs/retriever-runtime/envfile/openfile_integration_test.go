//go:build integration

package envfile

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// newReal builds Credentials over the real files, the way the composition root does: os.DirFS of the env file's folder
// and os.SameFile. The integration tests use it; no untagged test reads the real folder.
func newReal(path string) (*Credentials, error) {
	return NewWithFiles(path, os.Getenv, Files{FS: os.DirFS(filepath.Dir(path)), SameFile: os.SameFile})
}

// TestAMissingRealFolderIsAMissingKey checks the real files on a path that does not exist: the key is reported as
// missing, with no error from the file calls and no value.
func TestAMissingRealFolderIsAMissingKey(t *testing.T) {
	c, err := newReal("/nonexistent-retriever-test-folder/.env.local")
	if err != nil {
		t.Fatalf("newReal error = %v", err)
	}
	_, err = c.APIKey(mustCredentialRef(t, "OUTLINE_API_KEY"))
	var missing domain.MissingKeyError
	if !errors.As(err, &missing) {
		t.Fatalf("APIKey error = %v, want a MissingKeyError", err)
	}
}
