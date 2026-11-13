package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
)

// maxPort is the highest TCP port number.
const maxPort = 65535

// errPortRange reports a --port value outside the TCP port range.
var errPortRange = usageError{msg: fmt.Sprintf("port must be between 0 and %d", maxPort)}

// serveGraphQL runs the GraphQL server until SIGINT or SIGTERM, then shuts it down. Its one stdout line names the
// address it listens on, which is useful when --port 0 picks a free port.
func serveGraphQL(inv *invocation) error {
	rt := inv.rt
	port := 0
	if v, ok := inv.values["--port"]; ok {
		n, err := strconv.Atoi(v)
		if err != nil {
			return usageError{msg: "invalid value for --port: " + v}
		}
		if n < 0 || n > maxPort {
			return errPortRange
		}
		port = n
	}
	if rt.setupErr != nil {
		return setupError{err: rt.setupErr}
	}
	if rt.serve == nil {
		return errNotConfigured
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sig := stopOnSignal(ctx, cancel)
	listening := false
	err := rt.serve(ctx, port, func(addr string) {
		listening = true
		fmt.Fprintf(rt.out, "retriever serve listening on %s\n", addr)
	})
	cancel()
	if signalled, ok := <-sig; ok {
		// The listener shut down because a signal ended the session, so the status names the signal, as mcp does.
		return exitStatus{code: signalExitStatus(signalled)}
	}
	if err != nil && !listening {
		// The listener never came up, so the environment refused the start before any work began (exit status 125).
		return refusedError{err: fmt.Errorf("could not listen on the loopback address: %w", err)}
	}
	return err
}

// stopOnSignal cancels ctx when SIGINT or SIGTERM arrives. The returned channel yields the signal once, or closes without
// a value when ctx ends first, so the caller can tell a signal from an ordinary shutdown.
func stopOnSignal(ctx context.Context, cancel context.CancelFunc) <-chan os.Signal {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	out := make(chan os.Signal, 1)
	go func() {
		defer close(out)
		defer signal.Stop(sigs)
		select {
		case s := <-sigs:
			out <- s
			cancel()
		case <-ctx.Done():
		}
	}()
	return out
}

// signalExitStatus maps a signal onto the status retriever serve and retriever mcp report: 130 for an interrupt and 143
// for SIGTERM.
func signalExitStatus(s os.Signal) int {
	if s == syscall.SIGTERM {
		return mcpExitTerm
	}
	return mcpExitInterrupt
}
