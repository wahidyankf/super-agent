package markdownfiles

import (
	"context"
	"errors"
	"os"
	"testing"
)

// errInjected is the fault a test makes one step of a write return.
var errInjected = errors.New("injected fault")

// failingTemp is a real temporary file whose write, mode, or close step returns a fault when one is set. Every other step
// runs on the real file, so the temporary file exists on disk until the write removes it.
type failingTemp struct {
	*os.File
	writeErr error
	chmodErr error
	closeErr error
}

func (f failingTemp) Write(p []byte) (int, error) {
	if f.writeErr != nil {
		return 0, f.writeErr
	}
	return f.File.Write(p)
}

func (f failingTemp) Chmod(mode os.FileMode) error {
	if f.chmodErr != nil {
		return f.chmodErr
	}
	return f.File.Chmod(mode)
}

// Close always closes the real file, so no descriptor leaks, and then returns the fault when one is set.
func (f failingTemp) Close() error {
	err := f.File.Close()
	if f.closeErr != nil {
		return f.closeErr
	}
	return err
}

// faultyCreate returns a creator that makes a real temporary file and wraps it with the given faults.
func faultyCreate(writeErr, chmodErr, closeErr error) createTempFunc {
	return func(dir, pattern string) (tempFile, error) {
		f, err := os.CreateTemp(dir, pattern)
		if err != nil {
			return nil, err
		}
		return failingTemp{File: f, writeErr: writeErr, chmodErr: chmodErr, closeErr: closeErr}, nil
	}
}

// TestAFailedStepFailsTheWriteAndLeavesNothingBehind checks that a fault in the write, the mode change, or the close is
// returned to the caller, and that the temporary file is removed, so the folder holds no export and no leftover file.
func TestAFailedStepFailsTheWriteAndLeavesNothingBehind(t *testing.T) {
	tests := []struct {
		name     string
		writeErr error
		chmodErr error
		closeErr error
	}{
		{name: "a failed write is returned and the temporary file is removed", writeErr: errInjected},
		{name: "a failed mode change is returned and the temporary file is removed", chmodErr: errInjected},
		{name: "a failed close is returned and the temporary file is removed", closeErr: errInjected},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			sink := &Sink{dir: dir, create: faultyCreate(tt.writeErr, tt.chmodErr, tt.closeErr)}
			_, err := sink.Write(context.Background(), "doc.md", []byte("body"))
			if !errors.Is(err, errInjected) {
				t.Fatalf("Write error = %v, want the injected fault", err)
			}
			entries, readErr := os.ReadDir(dir)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if len(entries) != 0 {
				t.Fatalf("folder holds %d entries after a failed write, want none", len(entries))
			}
		})
	}
}

// TestAFailedTemporaryFileCreateIsReturned checks that a folder the temporary file cannot be made in stops the write
// with the creation fault, before any file is written.
func TestAFailedTemporaryFileCreateIsReturned(t *testing.T) {
	sink := &Sink{dir: t.TempDir(), create: func(string, string) (tempFile, error) { return nil, errInjected }}
	if _, err := sink.Write(context.Background(), "doc.md", []byte("body")); !errors.Is(err, errInjected) {
		t.Fatalf("Write error = %v, want the injected creation fault", err)
	}
}
