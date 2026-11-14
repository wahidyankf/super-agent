package specfolder

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"testing"
	"testing/fstest"
)

// limit is the size limit the tests pass to Read. A test file is far smaller, so the limit only matters where a test
// says so.
const limit = 1 << 10

// digestOf returns the lower-case hex SHA-256 of data, computed in memory.
func digestOf(data string) string {
	sum := sha256.Sum256([]byte(data))
	return hex.EncodeToString(sum[:])
}

// missingFS is a folder that does not exist: every open fails with fs.ErrNotExist.
type missingFS struct{}

func (missingFS) Open(string) (fs.File, error) { return nil, fs.ErrNotExist }

// brokenFS is a folder that cannot be read for a reason other than its absence.
type brokenFS struct{}

func (brokenFS) Open(string) (fs.File, error) { return nil, errors.New("disk fault") }

func TestEverySpecFileIsReadInNameOrderWithItsDigest(t *testing.T) {
	folder := fstest.MapFS{
		"b.yml":  {Data: []byte("kind: b\n")},
		"a.yaml": {Data: []byte("kind: a\n")},
	}
	files, err := Read(folder, limit)
	if err != nil {
		t.Fatalf("Read() error = %v, want nil", err)
	}
	if len(files) != 2 {
		t.Fatalf("files = %d, want 2", len(files))
	}
	if files[0].Name != "a.yaml" || files[1].Name != "b.yml" {
		t.Fatalf("names = %q, %q; want a.yaml, b.yml", files[0].Name, files[1].Name)
	}
	if files[0].Digest != digestOf("kind: a\n") {
		t.Fatalf("Digest = %q, want the SHA-256 of the bytes", files[0].Digest)
	}
	if string(files[1].Data) != "kind: b\n" {
		t.Fatalf("Data = %q, want the bytes as read", files[1].Data)
	}
}

func TestOtherEntriesAreSkipped(t *testing.T) {
	folder := fstest.MapFS{
		"notes.txt":   {Data: []byte("not a spec")},
		"sub/x.yml":   {Data: []byte("kind: x\n")},
		"example.yml": {Data: []byte("kind: example\n")},
	}
	files, err := Read(folder, limit)
	if err != nil {
		t.Fatalf("Read() error = %v, want nil", err)
	}
	if len(files) != 1 || files[0].Name != "example.yml" {
		t.Fatalf("files = %v, want only example.yml", files)
	}
}

func TestEmptyFolderHasNoSpecFiles(t *testing.T) {
	files, err := Read(fstest.MapFS{}, limit)
	if err != nil || len(files) != 0 {
		t.Fatalf("Read() = %d files, %v; want none and no error", len(files), err)
	}
}

func TestSymlinkSpecFileIsRefusedNamingTheFile(t *testing.T) {
	folder := fstest.MapFS{"linked.yml": {Data: []byte("elsewhere"), Mode: fs.ModeSymlink}}
	_, err := Read(folder, limit)
	var fe FileError
	if !errors.As(err, &fe) {
		t.Fatalf("Read() error = %v, want a FileError", err)
	}
	if fe.Name != "linked.yml" || fe.Rule != "must be a regular file, not a symlink or a folder" {
		t.Fatalf("FileError = %+v, want the name and the symlink rule", fe)
	}
	if want := "connector spec file linked.yml: must be a regular file, not a symlink or a folder"; err.Error() != want {
		t.Fatalf("Error() = %q, want %q", err.Error(), want)
	}
}

func TestOversizeSpecFileIsRefused(t *testing.T) {
	folder := fstest.MapFS{"big.yml": {Data: []byte("12345")}}
	_, err := Read(folder, 4)
	var fe FileError
	if !errors.As(err, &fe) || fe.Rule != "is larger than the size limit" {
		t.Fatalf("Read() error = %v, want the size rule", err)
	}
}

func TestFileExactlyAtTheLimitIsRead(t *testing.T) {
	folder := fstest.MapFS{"edge.yml": {Data: []byte("1234")}}
	files, err := Read(folder, 4)
	if err != nil || len(files) != 1 {
		t.Fatalf("Read() = %d files, %v; want one file", len(files), err)
	}
}

func TestMissingFolderIsReportedWithoutAPath(t *testing.T) {
	_, err := Read(missingFS{}, limit)
	if !errors.Is(err, ErrNoFolder) {
		t.Fatalf("Read() error = %v, want ErrNoFolder", err)
	}
}

func TestUnreadableFolderIsReported(t *testing.T) {
	_, err := Read(brokenFS{}, limit)
	if !errors.Is(err, ErrFolderUnreadable) {
		t.Fatalf("Read() error = %v, want ErrFolderUnreadable", err)
	}
	if got := err.Error(); got != "could not read the connectors folder" {
		t.Fatalf("Error() = %q, want the fixed text", got)
	}
}

// unopenable lists one spec file that cannot be opened, so the read names the file and the rule.
type unopenable struct{ fstest.MapFS }

func (u unopenable) Open(name string) (fs.File, error) {
	if name == "bad.yml" {
		return nil, errors.New("disk fault")
	}
	return u.MapFS.Open(name)
}

// faultFile is an open spec file whose read or close fails.
type faultFile struct {
	readErr  error
	closeErr error
}

func (faultFile) Stat() (fs.FileInfo, error) { return nil, errors.New("not used") }
func (f faultFile) Read([]byte) (int, error) {
	if f.readErr != nil {
		return 0, f.readErr
	}
	return 0, io.EOF
}
func (f faultFile) Close() error { return f.closeErr }

// faultyFS opens bad.yml as a faultFile and every other name from the map.
type faultyFS struct {
	fstest.MapFS
	file faultFile
}

func (f faultyFS) Open(name string) (fs.File, error) {
	if name == "bad.yml" {
		return f.file, nil
	}
	return f.MapFS.Open(name)
}

func TestSpecFileThatFailsWhileBeingReadIsRefused(t *testing.T) {
	for _, file := range []faultFile{{readErr: errors.New("disk fault")}, {closeErr: errors.New("close fault")}} {
		folder := faultyFS{MapFS: fstest.MapFS{"bad.yml": {Data: []byte("x")}}, file: file}
		_, err := Read(folder, limit)
		var fe FileError
		if !errors.As(err, &fe) || fe.Name != "bad.yml" || fe.Rule != "could not be read" {
			t.Fatalf("Read() error = %v, want a could-not-be-read FileError", err)
		}
	}
}

func TestSpecFileThatCannotBeOpenedIsRefused(t *testing.T) {
	folder := unopenable{fstest.MapFS{"bad.yml": {Data: []byte("x")}}}
	_, err := Read(folder, limit)
	var fe FileError
	if !errors.As(err, &fe) || fe.Name != "bad.yml" || fe.Rule != "could not be read" {
		t.Fatalf("Read() error = %v, want a FileError for bad.yml", err)
	}
}
