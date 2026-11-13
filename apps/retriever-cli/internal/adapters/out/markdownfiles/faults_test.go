package markdownfiles

import (
	"errors"
	"os"
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
