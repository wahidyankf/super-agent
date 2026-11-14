//go:build integration

package cli

import (
	"context"
	"errors"
	"syscall"
	"testing"
	"time"
)

// serveSignalled returns a serve function that is ready at once and then sends sig to this process, as a terminal or a
// supervisor would. The signal is caught by the serve command, which cancels the context the server runs under. Sending a
// real signal opens an OS-level effect, so these tests carry the integration tag.
func serveSignalled(sig syscall.Signal) func(ctx context.Context, _ int, ready func(string)) error {
	return func(ctx context.Context, _ int, ready func(string)) error {
		ready("127.0.0.1:1")
		if err := syscall.Kill(syscall.Getpid(), sig); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(10 * time.Second):
			return errors.New("the signal did not reach the serve command")
		}
	}
}

// TestServeStopsOnSIGTERMWithStatus143Sent proves, with a real signal, that SIGTERM ends serve with 143, as retriever mcp does.
func TestServeStopsOnSIGTERMWithStatus143Sent(t *testing.T) {
	code, _, errOut := runRetriever(t, []string{"serve"}, outlineSetup{serve: serveSignalled(syscall.SIGTERM)})
	if code != 143 {
		t.Fatalf("SIGTERM must end serve with exit 143, got exit %d (stderr %q)", code, errOut)
	}
}

// TestServeStopsOnSIGINTWithStatus130Sent proves, with a real signal, that an interrupt ends serve with 130, as retriever mcp does.
func TestServeStopsOnSIGINTWithStatus130Sent(t *testing.T) {
	code, _, errOut := runRetriever(t, []string{"serve"}, outlineSetup{serve: serveSignalled(syscall.SIGINT)})
	if code != 130 {
		t.Fatalf("SIGINT must end serve with exit 130, got exit %d (stderr %q)", code, errOut)
	}
}
