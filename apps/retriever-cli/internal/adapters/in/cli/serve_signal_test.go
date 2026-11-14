package cli

import (
	"context"
	"syscall"
	"testing"
)

// signalledContext returns a context that is already cancelled with the cause of sig, as the composition root cancels the
// serve context when that signal arrives. The serve command reads the cause and maps it to an exit status, so no real
// signal is sent.
func signalledContext(sig syscall.Signal) context.Context {
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(SignalCause(sig))
	return ctx
}

// serveUntilEnded is a serve function that is ready at once and returns when its context ends, as a server does.
func serveUntilEnded(ctx context.Context, _ int, ready func(string)) error {
	ready("127.0.0.1:1")
	<-ctx.Done()
	return nil
}

// TestServeStopsOnSIGTERMWithStatus143 proves that a SIGTERM cause ends serve with 143, as retriever mcp does. The
// real signal that sets this cause is delivered by the integration test of the same file name.
func TestServeStopsOnSIGTERMWithStatus143(t *testing.T) {
	code, _, errOut := runRetriever(t, []string{"serve"}, outlineSetup{serve: serveUntilEnded, ctx: signalledContext(syscall.SIGTERM)})
	if code != 143 {
		t.Fatalf("SIGTERM must end serve with exit 143, got exit %d (stderr %q)", code, errOut)
	}
}

// TestServeStopsOnSIGINTWithStatus130 proves that an interrupt cause ends serve with 130, as retriever mcp does.
func TestServeStopsOnSIGINTWithStatus130(t *testing.T) {
	code, _, errOut := runRetriever(t, []string{"serve"}, outlineSetup{serve: serveUntilEnded, ctx: signalledContext(syscall.SIGINT)})
	if code != 130 {
		t.Fatalf("SIGINT must end serve with exit 130, got exit %d (stderr %q)", code, errOut)
	}
}

// TestServeReturnsZeroWhenItsServerEndsWithoutASignal proves that a shutdown with no signal keeps exit status 0.
func TestServeReturnsZeroWhenItsServerEndsWithoutASignal(t *testing.T) {
	serve := func(_ context.Context, _ int, ready func(string)) error {
		ready("127.0.0.1:1")
		return nil
	}
	code, _, errOut := runRetriever(t, []string{"serve"}, outlineSetup{serve: serve})
	if code != 0 {
		t.Fatalf("a shutdown without a signal must exit 0, got exit %d (stderr %q)", code, errOut)
	}
}
