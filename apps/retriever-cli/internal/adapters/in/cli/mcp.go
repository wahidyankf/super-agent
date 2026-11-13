package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/sources/operations"
)

// Exit statuses of retriever mcp, from the exit-status contract. A client that closes its input ends the server with 0.
// An interrupt ends it with 130 and SIGTERM with 143, as the shell reports a signal, but as explicit statuses, so the
// process is not killed by the signal and a blocked read on standard input cannot hold it up.
const (
	mcpExitOK        = 0
	mcpExitNotDone   = 2
	mcpExitInterrupt = 130
	mcpExitTerm      = 143
)

// mcpCeiling returns the access ceiling of the session: the --max-access value, or read when the flag is absent. A value
// other than read, write, or admin is a usage error, so the session never starts with a level it did not name.
func mcpCeiling(inv *invocation) (domain.Access, error) {
	raw, ok := inv.values["--max-access"]
	if !ok {
		return domain.AccessRead, nil
	}
	ceiling, err := domain.NewAccess(raw)
	if err != nil {
		return domain.Access{}, usageError{msg: fmt.Sprintf("invalid value for --max-access: %s (want read, write, or admin)", raw)}
	}
	return ceiling, nil
}

// mcpResources returns the resources that the --tools flag names, or nil when the flag is absent, so the server offers
// every resource. A value that names no resource, an empty value, an empty entry, and an unknown word are usage errors,
// and the message lists the valid resources.
func mcpResources(inv *invocation) ([]string, error) {
	raw, ok := inv.values["--tools"]
	if !ok {
		return nil, nil
	}
	resources, err := operations.ParseResources(raw)
	if err != nil {
		return nil, usageError{msg: fmt.Sprintf("invalid value for --tools: %v", err)}
	}
	return resources, nil
}

// runMCP serves the MCP protocol on the process's standard input and output. A usage error or a setup error ends the
// command with status 2 before anything is written to standard output. Interrupt and SIGTERM end the session with their
// statuses.
func runMCP(inv *invocation) error {
	rt := inv.rt
	ceiling, err := mcpCeiling(inv)
	if err != nil {
		return exitStatus{code: mcpExitNotDone, err: err}
	}
	resources, err := mcpResources(inv)
	if err != nil {
		return exitStatus{code: mcpExitNotDone, err: err}
	}
	if rt.setupErr != nil {
		return exitStatus{code: mcpExitNotDone, err: setupError{err: rt.setupErr}}
	}
	if rt.mcp == nil {
		return exitStatus{code: mcpExitNotDone, err: errNotConfigured}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigs)

	done := make(chan error, 1)
	go func() { done <- rt.mcp(ctx, ceiling, resources, rt.stdin, rt.stdout) }()

	select {
	case err := <-done:
		if err != nil {
			return exitStatus{code: mcpExitNotDone, err: err}
		}
		return exitStatus{code: mcpExitOK}
	case sig := <-sigs:
		cancel()
		if sig == syscall.SIGTERM {
			return exitStatus{code: mcpExitTerm}
		}
		return exitStatus{code: mcpExitInterrupt}
	}
}
