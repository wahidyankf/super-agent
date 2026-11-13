//go:build e2e

package behaviour

import (
	"os"
	"os/exec"
	"syscall"
	"testing"
)

// TestClosedStandardOutputPipeEndsTheProcessBySIGPIPE runs the built binary with its standard output connected to a pipe
// whose reader is already closed. The first write fails with EPIPE, and the Go runtime ends the process by SIGPIPE, which
// the shell reports as status 141. The test names the process's own end, not a status it made up. It runs the real binary,
// so it belongs to the end-to-end level.
func TestClosedStandardOutputPipeEndsTheProcessBySIGPIPE(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("close the reader: %v", err)
	}
	defer writer.Close()

	cmd := exec.Command(e2eBinary, "--help")
	cmd.Stdout = writer
	err = cmd.Run()
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("the process must end with an exit error after its pipe closes, got %v", err)
	}
	status, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok || !status.Signaled() || status.Signal() != syscall.SIGPIPE {
		t.Fatalf("the process must end by SIGPIPE (shell status 141), got %v", exitErr)
	}
}
