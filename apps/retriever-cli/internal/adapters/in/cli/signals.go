package cli

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"
)

// signalCause is the cancellation cause that names the signal which ended a session.
type signalCause struct {
	sig os.Signal
}

// Error names the signal, so a log line that carries the cause reads as a signal and not as a failure.
func (c signalCause) Error() string { return "received " + c.sig.String() }

// SignalCause returns the cancellation cause for sig. A caller that owns the context in Deps cancels it with this cause
// to end serve or mcp as if the process had received sig.
func SignalCause(sig os.Signal) error { return signalCause{sig: sig} }

// withSignals returns a context that ends when parent ends, or when SIGINT or SIGTERM arrives. A signal cancels the
// context with its SignalCause. The returned stop function cancels the context and releases the signal handler.
func withSignals(parent context.Context) (context.Context, func()) {
	ctx, cancel := context.WithCancelCause(parent)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	go func() {
		select {
		case s := <-sigs:
			cancel(SignalCause(s))
		case <-ctx.Done():
		}
		signal.Stop(sigs)
	}()
	return ctx, func() { cancel(nil) }
}

// endedBySignal reports the status of a session that a signal ended: 130 for an interrupt and 143 for SIGTERM. The
// second result is false when no signal ended the context.
func endedBySignal(ctx context.Context) (int, bool) {
	var c signalCause
	if errors.As(context.Cause(ctx), &c) {
		return signalExitStatus(c.sig), true
	}
	return 0, false
}

// signalExitStatus maps a signal onto the status retriever serve and retriever mcp report: 130 for an interrupt and 143
// for SIGTERM. The statuses are explicit, so the process is not killed by the signal (CLI convention 001).
func signalExitStatus(s os.Signal) int {
	if s == syscall.SIGTERM {
		return mcpExitTerm
	}
	return mcpExitInterrupt
}
