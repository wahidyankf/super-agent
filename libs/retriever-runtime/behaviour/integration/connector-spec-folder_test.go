//go:build integration

package integration_test

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-runtime/specfolder"
)

// specSizeLimit is the most bytes one spec file may hold: 256 KiB, the limit of a spec document.
const specSizeLimit = 256 << 10

// digestOf returns the lower-case hex SHA-256 of text.
func digestOf(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

// @scenario specs/libs/retriever-runtime/behaviours/connector-spec-folder.feature: Every YAML spec file of the folder is read in name order with its digest
func TestEveryYAMLSpecFileOfTheFolderIsReadInNameOrderWithItsDigest(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"b.yml": "kind: b\n", "a.yaml": "kind: a\n"})
	files, err := specfolder.Read(os.DirFS(dir), specSizeLimit)
	if err != nil {
		t.Fatalf("Read error = %v", err)
	}
	if len(files) != 2 || files[0].Name != "a.yaml" || files[1].Name != "b.yml" {
		t.Fatalf("files = %+v, want a.yaml then b.yml", files)
	}
	if files[0].Digest != digestOf("kind: a\n") || files[1].Digest != digestOf("kind: b\n") {
		t.Fatalf("digests do not match the bytes of the files")
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/connector-spec-folder.feature: Entries that are not YAML spec files are skipped
func TestEntriesThatAreNotYAMLSpecFilesAreSkipped(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"a.yml": "kind: a\n", "notes.txt": "notes", "readme": "readme"})
	files, err := specfolder.Read(os.DirFS(dir), specSizeLimit)
	if err != nil {
		t.Fatalf("Read error = %v", err)
	}
	if len(files) != 1 || files[0].Name != "a.yml" {
		t.Fatalf("files = %+v, want only a.yml", files)
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/connector-spec-folder.feature: An empty folder has no spec files
func TestAnEmptyFolderHasNoSpecFiles(t *testing.T) {
	files, err := specfolder.Read(os.DirFS(t.TempDir()), specSizeLimit)
	if err != nil || len(files) != 0 {
		t.Fatalf("Read = %d files, error %v; want none and no error", len(files), err)
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/connector-spec-folder.feature: A spec file that is a symbolic link or a folder refuses the whole read
func TestASpecFileThatIsASymbolicLinkOrAFolderRefusesTheWholeRead(t *testing.T) {
	cases := []struct {
		name  string
		setup func(t *testing.T, dir string)
	}{
		{"symbolic link", func(t *testing.T, dir string) {
			writeFiles(t, dir, map[string]string{"real.yml": "kind: a\n"})
			if err := os.Symlink(filepath.Join(dir, "real.yml"), filepath.Join(dir, "bad.yml")); err != nil {
				t.Fatalf("symlink: %v", err)
			}
		}},
		{"folder", func(t *testing.T, dir string) {
			if err := os.Mkdir(filepath.Join(dir, "bad.yml"), 0o755); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			tc.setup(t, dir)
			files, err := specfolder.Read(os.DirFS(dir), specSizeLimit)
			wantContains(t, err, "connector spec file bad.yml: must be a regular file, not a symlink or a folder")
			if files != nil {
				t.Fatalf("files = %+v, want none", files)
			}
		})
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/connector-spec-folder.feature: A spec file over the size limit refuses the read and names the file
func TestASpecFileOverTheSizeLimitRefusesTheReadAndNamesTheFile(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"big.yml": strings.Repeat("x", specSizeLimit+1)})
	files, err := specfolder.Read(os.DirFS(dir), specSizeLimit)
	wantContains(t, err, "connector spec file big.yml: is larger than the size limit")
	if files != nil {
		t.Fatalf("files = %+v, want none", files)
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/connector-spec-folder.feature: A spec file of exactly the size limit is read
func TestASpecFileOfExactlyTheSizeLimitIsRead(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"edge.yml": strings.Repeat("x", specSizeLimit)})
	files, err := specfolder.Read(os.DirFS(dir), specSizeLimit)
	if err != nil || len(files) != 1 || files[0].Name != "edge.yml" {
		t.Fatalf("Read = %+v, error %v; want edge.yml", files, err)
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/connector-spec-folder.feature: A folder that does not exist returns the no-folder error without a path
func TestAFolderThatDoesNotExistReturnsTheNoFolderErrorWithoutAPath(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "connectors")
	_, err := specfolder.Read(os.DirFS(missing), specSizeLimit)
	wantError(t, err, "the connectors folder does not exist")
	if strings.Contains(err.Error(), missing) {
		t.Fatalf("error %q holds the path", err.Error())
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/connector-spec-folder.feature: A folder that cannot be read returns the unreadable error, no path
func TestAFolderThatCannotBeReadReturnsTheUnreadableErrorWithoutAPath(t *testing.T) {
	// A regular file where the folder should be exists, and it cannot be listed as a folder.
	file := filepath.Join(t.TempDir(), "connectors")
	writeFiles(t, filepath.Dir(file), map[string]string{"connectors": "not a folder"})
	_, err := specfolder.Read(os.DirFS(file), specSizeLimit)
	wantError(t, err, "could not read the connectors folder")
	if strings.Contains(err.Error(), file) {
		t.Fatalf("error %q holds the path", err.Error())
	}
}
