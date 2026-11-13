package markdownfiles

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// errFakeFault is the fault a fake folder step returns. The real folder is never touched by a unit test.
var errFakeFault = errors.New("injected folder fault")

// fakeTemp is a temporary export file held in memory. Each step records what it did, and a step fails when its fault is set.
type fakeTemp struct {
	name     string
	written  []byte
	mode     os.FileMode
	closed   bool
	writeErr error
	chmodErr error
	closeErr error
}

func (f *fakeTemp) Write(p []byte) (int, error) {
	if f.writeErr != nil {
		return 0, f.writeErr
	}
	f.written = append(f.written, p...)
	return len(p), nil
}

func (f *fakeTemp) Name() string { return f.name }

func (f *fakeTemp) Chmod(mode os.FileMode) error {
	if f.chmodErr != nil {
		return f.chmodErr
	}
	f.mode = mode
	return nil
}

func (f *fakeTemp) Close() error {
	f.closed = true
	return f.closeErr
}

// folderRecord lists the folder calls a write made, in order.
type folderRecord struct {
	creates []string
	mkdirs  []string
	renames [][2]string
	removed []string
}

// folderFaults are the faults a fake folder returns from one step.
type folderFaults struct {
	createErr error
	mkdirErr  error
	renameErr error
}

// fakeSink returns a sink over an in-memory folder. Its temporary file is temp, and every folder call is recorded in rec.
func fakeSink(temp *fakeTemp, rec *folderRecord, faults folderFaults) *Sink {
	return &Sink{
		dir: "/exports",
		create: func(dir, pattern string) (tempFile, error) {
			rec.creates = append(rec.creates, dir+"/"+pattern)
			if faults.createErr != nil {
				return nil, faults.createErr
			}
			return temp, nil
		},
		fsys: folderOps{
			mkdirAll: func(path string, _ os.FileMode) error {
				rec.mkdirs = append(rec.mkdirs, path)
				return faults.mkdirErr
			},
			rename: func(oldpath, newpath string) error {
				rec.renames = append(rec.renames, [2]string{oldpath, newpath})
				return faults.renameErr
			},
			remove: func(name string) error {
				rec.removed = append(rec.removed, name)
				return nil
			},
		},
	}
}

// TestWriteStoresTheExportAndReturnsItsPath checks the happy path with fakes: the folder is created, the bytes go to the
// temporary file, the mode is set, the file is closed, and it is renamed onto the target name.
func TestWriteStoresTheExportAndReturnsItsPath(t *testing.T) {
	temp := &fakeTemp{name: "/exports/.export-1.tmp"}
	var rec folderRecord
	got, err := fakeSink(temp, &rec, folderFaults{}).Write(context.Background(), "doc.md", []byte("# Hi\n"))
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}
	target := filepath.Join("/exports", "doc.md")
	if got != target {
		t.Fatalf("location = %q, want %q", got, target)
	}
	if string(temp.written) != "# Hi\n" || temp.mode != fileMode || !temp.closed {
		t.Fatalf("temporary file = %q, mode %v, closed %v; want the bytes, the export mode, and a close", temp.written, temp.mode, temp.closed)
	}
	if len(rec.mkdirs) != 1 || rec.mkdirs[0] != "/exports" {
		t.Fatalf("folder creations = %v, want one for /exports", rec.mkdirs)
	}
	if len(rec.renames) != 1 || rec.renames[0] != [2]string{temp.name, target} {
		t.Fatalf("renames = %v, want the temporary file onto the target", rec.renames)
	}
	if len(rec.removed) != 0 {
		t.Fatalf("a successful write removed %v", rec.removed)
	}
}

// TestWriteRefusesEveryNameThatIsNotAPlainFileName checks that a refused name touches no folder: no folder is created and
// no temporary file is made.
func TestWriteRefusesEveryNameThatIsNotAPlainFileName(t *testing.T) {
	names := []string{"", ".", "..", "../escape.md", "sub/doc.md", `sub\doc.md`, "doc\x00.md", "doc.txt", ".md"}
	for _, name := range names {
		t.Run(strings.ReplaceAll(name, "\x00", "NUL"), func(t *testing.T) {
			var rec folderRecord
			_, err := fakeSink(&fakeTemp{}, &rec, folderFaults{}).Write(context.Background(), name, []byte("x"))
			if !errors.Is(err, ErrBadName) {
				t.Fatalf("Write(%q) error = %v, want ErrBadName", name, err)
			}
			if len(rec.mkdirs) != 0 || len(rec.creates) != 0 {
				t.Fatalf("a refused name %q reached the folder: mkdirs %v, creates %v", name, rec.mkdirs, rec.creates)
			}
		})
	}
}

// TestWriteRefusesAnEmptyFolder checks that a sink with no folder refuses to write, before any folder call.
func TestFakeWriteRefusesAnEmptyFolder(t *testing.T) {
	var rec folderRecord
	sink := fakeSink(&fakeTemp{}, &rec, folderFaults{})
	sink.dir = ""
	if _, err := sink.Write(context.Background(), "doc.md", []byte("x")); !errors.Is(err, ErrNoFolder) {
		t.Fatalf("Write error = %v, want ErrNoFolder", err)
	}
	if len(rec.mkdirs) != 0 {
		t.Fatalf("a sink with no folder created %v", rec.mkdirs)
	}
}

// TestWriteReportsAFolderThatCannotBeCreated checks that a folder the process cannot create stops the write with the
// fault, before any temporary file is made.
func TestWriteReportsAFolderThatCannotBeCreated(t *testing.T) {
	var rec folderRecord
	_, err := fakeSink(&fakeTemp{}, &rec, folderFaults{mkdirErr: errFakeFault}).Write(context.Background(), "doc.md", []byte("x"))
	if !errors.Is(err, errFakeFault) {
		t.Fatalf("Write error = %v, want the folder fault", err)
	}
	if len(rec.creates) != 0 {
		t.Fatalf("a folder that could not be created still made %v", rec.creates)
	}
}

// TestAFailedStepRemovesTheTemporaryFileAndPlacesNothing checks that a fault in the write, the mode change, or the close
// is returned, and that the temporary file is removed and never renamed onto the target.
func TestAFailedStepRemovesTheTemporaryFileAndPlacesNothing(t *testing.T) {
	tests := []struct {
		name string
		temp fakeTemp
	}{
		{name: "a failed write", temp: fakeTemp{writeErr: errFakeFault}},
		{name: "a failed mode change", temp: fakeTemp{chmodErr: errFakeFault}},
		{name: "a failed close", temp: fakeTemp{closeErr: errFakeFault}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			temp := tt.temp
			temp.name = "/exports/.export-2.tmp"
			var rec folderRecord
			_, err := fakeSink(&temp, &rec, folderFaults{}).Write(context.Background(), "doc.md", []byte("body"))
			if !errors.Is(err, errFakeFault) {
				t.Fatalf("Write error = %v, want the injected fault", err)
			}
			if len(rec.removed) != 1 || rec.removed[0] != temp.name {
				t.Fatalf("removed = %v, want the temporary file", rec.removed)
			}
			if len(rec.renames) != 0 {
				t.Fatalf("a failed write was renamed onto the target: %v", rec.renames)
			}
		})
	}
}

// TestAFailedTemporaryFileCreateIsReturned checks that a failure to make the temporary file stops the write with the
// creation fault, before any file is placed or removed.
func TestFakeAFailedTemporaryFileCreateIsReturned(t *testing.T) {
	var rec folderRecord
	_, err := fakeSink(&fakeTemp{}, &rec, folderFaults{createErr: errFakeFault}).Write(context.Background(), "doc.md", []byte("x"))
	if !errors.Is(err, errFakeFault) {
		t.Fatalf("Write error = %v, want the creation fault", err)
	}
	if len(rec.renames) != 0 || len(rec.removed) != 0 {
		t.Fatalf("a failed creation renamed %v or removed %v", rec.renames, rec.removed)
	}
}

// TestAFailedRenameRemovesTheTemporaryFile checks that a rename that fails, such as onto a folder of the same name, returns
// the fault and removes the temporary file.
func TestAFailedRenameRemovesTheTemporaryFile(t *testing.T) {
	temp := &fakeTemp{name: "/exports/.export-3.tmp"}
	var rec folderRecord
	_, err := fakeSink(temp, &rec, folderFaults{renameErr: errFakeFault}).Write(context.Background(), "doc.md", []byte("x"))
	if !errors.Is(err, errFakeFault) {
		t.Fatalf("Write error = %v, want the rename fault", err)
	}
	if len(rec.removed) != 1 || rec.removed[0] != temp.name {
		t.Fatalf("removed = %v, want the temporary file after a failed rename", rec.removed)
	}
}
