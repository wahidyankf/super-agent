//go:build integration

package cli

import (
	"context"
	"errors"
	"syscall"
	"testing"
	"time"
)

// TestServeStopsOnSIGTERM sends SIGTERM to this process while serve runs. The serve command must cancel the context
// the server runs under, and then return 0. The test sends a real signal, so it belongs to the integration layer.
func TestServeStopsOnSIGTERM(t *testing.T) {
	serve := func(ctx context.Context, _ int, ready func(string)) error {
		ready("127.0.0.1:1")
		if err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(10 * time.Second):
			return errors.New("the signal did not reach the serve command")
		}
	}

	code, _, errOut := runRetriever(t, []string{"serve"}, outlineSetup{serve: serve})

	if code != 0 {
		t.Fatalf("a signal must stop serve cleanly with exit 0, got exit %d (stderr %q)", code, errOut)
	}
}
