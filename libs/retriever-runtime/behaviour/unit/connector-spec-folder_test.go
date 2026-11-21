package unit_test

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/wahidyankf/super-agent/libs/retriever-runtime/specfolder"
)

// specSizeLimit is the most bytes one spec file may hold: 256 KiB, the limit of a spec document.
const specSizeLimit = 256 << 10

// digestOf returns the lower-case hex SHA-256 of text.
func digestOf(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

// faultyFolder serves the folder it wraps, except for one file. That file cannot be opened when openFails is set, and it
// fails inside Read otherwise. These are the two faults of a real file that a folder cannot give on demand.
type faultyFolder struct {
	fs.FS
	name      string
	openFails bool
}

// Open refuses the faulty file when openFails is set, and returns the faulty file with a failing Read otherwise.
func (f faultyFolder) Open(name string) (fs.File, error) {
	if name != f.name {
		return f.FS.Open(name)
	}
	if f.openFails {
		return nil, &fs.PathError{Op: "open", Path: name, Err: errors.New("permission denied")}
	}
	file, err := f.FS.Open(name)
	if err != nil {
		return nil, err
	}
	return failingRead{file}, nil
}

// failingRead is a file whose Read fails, as a disk does when it cannot give the bytes.
type failingRead struct{ fs.File }

// Read always fails.
func (failingRead) Read([]byte) (int, error) { return 0, errors.New("input/output error") }

// missingFolder is a folder that does not exist: opening it reports the not-exist error.
type missingFolder struct{}

// Open reports that the folder does not exist, with the path it was asked for.
func (missingFolder) Open(name string) (fs.File, error) {
	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
}

// unreadableFolder is a folder that exists and cannot be read: opening it reports a permission error.
type unreadableFolder struct{}

// Open reports that the folder cannot be read, with the path it was asked for.
func (unreadableFolder) Open(name string) (fs.File, error) {
	return nil, &fs.PathError{Op: "open", Path: name, Err: errors.New("permission denied")}
}

// @scenario specs/libs/retriever-runtime/behaviours/connector-spec-folder.feature: Every YAML spec file of the folder is read in name order with its digest
func TestEveryYAMLSpecFileOfTheFolderIsReadInNameOrderWithItsDigest(t *testing.T) {
	folder := fileSet(map[string]string{"b.yml": "kind: b\n", "a.yaml": "kind: a\n"})
	files, err := specfolder.Read(folder, specSizeLimit)
	if err != nil {
		t.Fatalf("Read error = %v", err)
	}
	if len(files) != 2 || files[0].Name != "a.yaml" || files[1].Name != "b.yml" {
		t.Fatalf("files = %+v, want a.yaml then b.yml", files)
	}
	for _, f := range files {
		want := map[string]string{"a.yaml": "kind: a\n", "b.yml": "kind: b\n"}[f.Name]
		if string(f.Data) != want {
			t.Fatalf("%s holds %q, want its bytes", f.Name, f.Data)
		}
		if f.Digest != digestOf(want) {
			t.Fatalf("%s digest = %s, want the SHA-256 of its bytes", f.Name, f.Digest)
		}
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/connector-spec-folder.feature: Entries that are not YAML spec files are skipped
func TestEntriesThatAreNotYAMLSpecFilesAreSkipped(t *testing.T) {
	folder := fstest.MapFS{
		"a.yml":     &fstest.MapFile{Data: []byte("kind: a\n")},
		"notes.txt": &fstest.MapFile{Data: []byte("notes")},
		"readme":    &fstest.MapFile{Data: []byte("readme")},
	}
	files, err := specfolder.Read(folder, specSizeLimit)
	if err != nil {
		t.Fatalf("Read error = %v", err)
	}
	if len(files) != 1 || files[0].Name != "a.yml" {
		t.Fatalf("files = %+v, want only a.yml", files)
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/connector-spec-folder.feature: An empty folder has no spec files
func TestAnEmptyFolderHasNoSpecFiles(t *testing.T) {
	files, err := specfolder.Read(fstest.MapFS{}, specSizeLimit)
	if err != nil {
		t.Fatalf("Read error = %v, want none", err)
	}
	if len(files) != 0 {
		t.Fatalf("files = %+v, want none", files)
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/connector-spec-folder.feature: A spec file that is a symbolic link or a folder refuses the whole read
func TestASpecFileThatIsASymbolicLinkOrAFolderRefusesTheWholeRead(t *testing.T) {
	cases := []struct {
		name string
		mode fs.FileMode
	}{
		{"symbolic link", fs.ModeSymlink},
		{"folder", fs.ModeDir},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			folder := fstest.MapFS{
				"a.yml":   &fstest.MapFile{Data: []byte("kind: a\n")},
				"bad.yml": &fstest.MapFile{Mode: tc.mode},
			}
			files, err := specfolder.Read(folder, specSizeLimit)
			wantError(t, err, "connector spec file bad.yml: must be a regular file, not a symlink or a folder")
			if files != nil {
				t.Fatalf("files = %+v, want none", files)
			}
		})
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/connector-spec-folder.feature: A spec file over the size limit refuses the read and names the file
func TestASpecFileOverTheSizeLimitRefusesTheReadAndNamesTheFile(t *testing.T) {
	folder := fileSet(map[string]string{"big.yml": strings.Repeat("x", specSizeLimit+1)})
	files, err := specfolder.Read(folder, specSizeLimit)
	wantError(t, err, "connector spec file big.yml: is larger than the size limit")
	if files != nil {
		t.Fatalf("files = %+v, want none", files)
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/connector-spec-folder.feature: A spec file of exactly the size limit is read
func TestASpecFileOfExactlyTheSizeLimitIsRead(t *testing.T) {
	folder := fileSet(map[string]string{"edge.yml": strings.Repeat("x", specSizeLimit)})
	files, err := specfolder.Read(folder, specSizeLimit)
	if err != nil {
		t.Fatalf("Read error = %v, want the file", err)
	}
	if len(files) != 1 || files[0].Name != "edge.yml" {
		t.Fatalf("files = %+v, want edge.yml", files)
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/connector-spec-folder.feature: A folder that does not exist returns the no-folder error without a path
func TestAFolderThatDoesNotExistReturnsTheNoFolderErrorWithoutAPath(t *testing.T) {
	_, err := specfolder.Read(missingFolder{}, specSizeLimit)
	wantError(t, err, "the connectors folder does not exist")
	if strings.Contains(err.Error(), "/") {
		t.Fatalf("error %q holds a path", err.Error())
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/connector-spec-folder.feature: A folder that cannot be read returns the unreadable error, no path
func TestAFolderThatCannotBeReadReturnsTheUnreadableErrorWithoutAPath(t *testing.T) {
	_, err := specfolder.Read(unreadableFolder{}, specSizeLimit)
	wantError(t, err, "could not read the connectors folder")
	if strings.Contains(err.Error(), "/") {
		t.Fatalf("error %q holds a path", err.Error())
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/connector-spec-folder.feature: A spec file that cannot be opened or read refuses the whole read
func TestASpecFileThatCannotBeOpenedOrReadRefusesTheWholeRead(t *testing.T) {
	cases := []struct {
		name      string
		openFails bool
	}{
		{"cannot be opened", true},
		{"fails while it is being read", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			folder := faultyFolder{
				FS:        fileSet(map[string]string{"locked.yml": "kind: a\n"}),
				name:      "locked.yml",
				openFails: tc.openFails,
			}
			files, err := specfolder.Read(folder, specSizeLimit)
			wantError(t, err, "connector spec file locked.yml: could not be read")
			if files != nil {
				t.Fatalf("files = %+v, want none", files)
			}
		})
	}
}
