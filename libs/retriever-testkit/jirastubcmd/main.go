// Package main is a test-only command. It serves the scripted Jira stub over TLS on a loopback port, as a process, so the
// end-to-end layer can point the real retriever binary at it through the loopback seam. The command logic lives in the
// stubcmd/jira library, which a unit test runs with fakes. This shell only passes the process to it.
package main

import (
	"os"

	"github.com/wahidyankf/super-agent/libs/retriever-testkit/stubcmd/jira"
)

// exit ends the process, and host is the process's own effects. main reads both, so a test replaces them and runs main
// with fakes.
var (
	exit = os.Exit
	host = jira.Real
)

func main() {
	exit(jira.Run(os.Args[1:], os.Stdout, os.Stderr, host))
}
