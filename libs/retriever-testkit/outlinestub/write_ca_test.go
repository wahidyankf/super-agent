package outlinestub

import (
	"bytes"
	"errors"
	"io/fs"
	"path/filepath"
	"testing"
)

// useWriteFile replaces the CA file writer for one test, so the test writes no file.
func useWriteFile(t *testing.T, f func(name string, data []byte, perm fs.FileMode) error) {
	t.Helper()
	old := writeFile
	t.Cleanup(func() { writeFile = old })
	writeFile = f
}

func TestWriteCAFileWritesTheStubCertificateToItsOwnFileNameWithOwnerOnlyAccess(t *testing.T) {
	useMemListener(t, newMemListener())
	stub := Start(t)
	var gotName string
	var gotData []byte
	var gotPerm fs.FileMode
	useWriteFile(t, func(name string, data []byte, perm fs.FileMode) error {
		gotName, gotData, gotPerm = name, data, perm
		return nil
	})

	path, err := stub.WriteCAFile("ca-dir")
	if err != nil {
		t.Fatalf("WriteCAFile: %v", err)
	}

	if want := filepath.Join("ca-dir", "outline-stub-ca.pem"); path != want || gotName != want {
		t.Fatalf("WriteCAFile wrote %q and returned %q, want both %q", gotName, path, want)
	}
	if want, err := stub.CAPEM(); err != nil || !bytes.Equal(gotData, want) {
		t.Fatalf("the CA file holds %q, want the stub's PEM", gotData)
	}
	if gotPerm != 0o600 {
		t.Fatalf("the CA file has mode %o, want 0600", gotPerm)
	}
}

func TestWriteCAFileWrapsAWriteFailureWithItsCause(t *testing.T) {
	useMemListener(t, newMemListener())
	stub := Start(t)
	useWriteFile(t, func(string, []byte, fs.FileMode) error { return fs.ErrPermission })

	path, err := stub.WriteCAFile("ca-dir")

	if path != "" {
		t.Fatalf("WriteCAFile returned path %q after a failed write", path)
	}
	if !errors.Is(err, fs.ErrPermission) {
		t.Fatalf("WriteCAFile returned %v, want the write's cause", err)
	}
	if got, want := err.Error(), "outlinestub: write CA file: permission denied"; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}
