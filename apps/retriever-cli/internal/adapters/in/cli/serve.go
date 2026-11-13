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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	listening := false
	err := rt.serve(ctx, port, func(addr string) {
		listening = true
		fmt.Fprintf(rt.out, "retriever serve listening on %s\n", addr)
	})
	if err != nil && !listening {
		// The listener never came up, so the environment refused the start before any work began (exit status 125).
		return refusedError{err: fmt.Errorf("could not listen on the loopback address: %w", err)}
	}
	return err
}
