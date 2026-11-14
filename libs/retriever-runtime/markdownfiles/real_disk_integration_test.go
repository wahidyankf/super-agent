//go:build integration

package markdownfiles

import (
	"os"
)

// realSink returns a sink over the real folder, with the calls of package os. The composition root passes the same calls
// in the app; the integration tests need them here, and no untagged test reads the real folder.
func realSink(dir string) *Sink {
	return New(dir, osCreateTemp, Folder{MkdirAll: os.MkdirAll, Rename: os.Rename, Remove: os.Remove})
}

// osCreateTemp is the real temporary file creator. It returns a nil interface, not a typed nil, when the file cannot be
// made.
func osCreateTemp(dir, pattern string) (TempFile, error) {
	f, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return nil, err
	}
	return f, nil
}
