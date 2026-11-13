package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/ports"
)

// fakeMCP stands in for the composition root's MCP server. It reads the whole input, writes one message, and records
// what it saw. A non-nil err is returned after the write.
type fakeMCP struct {
	read      string
	calls     int
	err       error
	ceiling   domain.Access
	resources []string
}

func (f *fakeMCP) serve(_ context.Context, ceiling domain.Access, resources []string, in io.Reader, out io.Writer) error {
	f.calls++
	f.ceiling = ceiling
	f.resources = resources
	data, err := io.ReadAll(in)
	if err != nil {
		return err
	}
	f.read = string(data)
	if _, err := io.WriteString(out, `{"jsonrpc":"2.0","id":1,"result":{}}`+"\n"); err != nil {
		return err
	}
	return f.err
}

// runMCPCLI runs argv with the given standard input and MCP server and returns the exit code and the two streams.
func runMCPCLI(t *testing.T, argv []string, stdin string, setupErr error, mcp MCPFunc) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := Run(argv, Deps{
		Ports:    ports.Ports{FS: stubFS{}},
		Stdin:    strings.NewReader(stdin),
		Stdout:   &stdout,
		Stderr:   &stderr,
		Getenv:   func(string) string { return "" },
		Now:      func() time.Time { return fixedNow },
		SetupErr: setupErr,
		MCP:      mcp,
	})
	return code, stdout.String(), stderr.String()
}

// TestMCPWritesOnlyTheServerToStandardOutput checks that the server gets the process input, and that the answer is the only
// text on standard output.
func TestMCPWritesOnlyTheServerToStandardOutput(t *testing.T) {
	fake := &fakeMCP{}
	code, out, errOut := runMCPCLI(t, []string{"mcp"}, "request\n", nil, fake.serve)
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
	if fake.read != "request\n" {
		t.Fatalf("server read %q, want the process input", fake.read)
	}
	if out != `{"jsonrpc":"2.0","id":1,"result":{}}`+"\n" {
		t.Fatalf("standard output is %q, want only the server's answer", out)
	}
}

// TestMCPCeilingDefaultsToRead checks that a session with no --max-access flag runs at the read ceiling (ADR 007 section
// 3.3), and that each named level is passed to the server as it is named.
func TestMCPCeilingDefaultsToRead(t *testing.T) {
	cases := []struct {
		name string
		argv []string
		want domain.Access
	}{
		{name: "no flag", argv: []string{"mcp"}, want: domain.AccessRead},
		{name: "read", argv: []string{"mcp", "--max-access", "read"}, want: domain.AccessRead},
		{name: "write", argv: []string{"mcp", "--max-access", "write"}, want: domain.AccessWrite},
		{name: "admin", argv: []string{"mcp", "--max-access", "admin"}, want: domain.AccessAdmin},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fake := &fakeMCP{}
			code, _, errOut := runMCPCLI(t, tc.argv, "", nil, fake.serve)
			if code != 0 {
				t.Fatalf("exit %d, stderr %q", code, errOut)
			}
			if fake.ceiling != tc.want {
				t.Fatalf("ceiling = %v, want %v", fake.ceiling, tc.want)
			}
		})
	}
}

// TestMCPInvalidMaxAccessExitsTwoBeforeAnyOutput checks that a --max-access value other than read, write, or admin is a
// usage error with status 2: the server never starts and standard output stays empty.
func TestMCPInvalidMaxAccessExitsTwoBeforeAnyOutput(t *testing.T) {
	for _, raw := range []string{"root", "Read", ""} {
		fake := &fakeMCP{}
		code, out, errOut := runMCPCLI(t, []string{"mcp", "--max-access", raw}, "", nil, fake.serve)
		if code != 2 || out != "" || !strings.Contains(errOut, "invalid value for --max-access") {
			t.Fatalf("--max-access %q: exit %d, stdout %q, stderr %q", raw, code, out, errOut)
		}
		if fake.calls != 0 {
			t.Fatalf("--max-access %q: the server started", raw)
		}
	}
}

// TestMCPSetupErrorExitsTwoBeforeTheServerStarts checks that a setup error ends the command with status 2, writes the
// reason to standard error only, and never starts the server.
func TestMCPSetupErrorExitsTwoBeforeTheServerStarts(t *testing.T) {
	fake := &fakeMCP{}
	code, out, errOut := runMCPCLI(t, []string{"mcp"}, "", errors.New("sources catalog is invalid"), fake.serve)
	if code != 2 || out != "" || !strings.Contains(errOut, "sources catalog is invalid") {
		t.Fatalf("exit %d, stdout %q, stderr %q", code, out, errOut)
	}
	if fake.calls != 0 {
		t.Fatal("the server started after a setup error")
	}
}

// TestMCPToolsDefaultsToEveryResource checks that a session with no --tools flag passes no resource list, so the server
// offers every tool the ceiling allows", tool count and context cost).
func TestMCPToolsDefaultsToEveryResource(t *testing.T) {
	fake := &fakeMCP{}
	code, _, errOut := runMCPCLI(t, []string{"mcp"}, "", nil, fake.serve)
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
	if fake.resources != nil {
		t.Fatalf("resources = %v, want none (every resource)", fake.resources)
	}
}

// TestMCPToolsPassesTheNamedResources checks that the --tools value reaches the server as its resource words, in the
// order named, with a repeated word kept once.
func TestMCPToolsPassesTheNamedResources(t *testing.T) {
	cases := []struct {
		name string
		argv []string
		want []string
	}{
		{name: "one resource", argv: []string{"mcp", "--tools", "documents"}, want: []string{"documents"}},
		{name: "two resources, one repeated", argv: []string{"mcp", "--tools=comments,documents,comments"}, want: []string{"comments", "documents"}},
		{name: "a resource with no tool at read", argv: []string{"mcp", "--tools", "api-keys"}, want: []string{"api-keys"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fake := &fakeMCP{}
			code, _, errOut := runMCPCLI(t, tc.argv, "", nil, fake.serve)
			if code != 0 {
				t.Fatalf("exit %d, stderr %q", code, errOut)
			}
			if !slices.Equal(fake.resources, tc.want) {
				t.Fatalf("resources = %v, want %v", fake.resources, tc.want)
			}
		})
	}
}

// TestMCPToolsUsageErrorsExitTwoBeforeAnyOutput checks that an empty value, an empty entry, and an unknown resource are
// usage errors with status 2: the server never starts, standard output stays empty, and the message names the flag and
// lists the valid resources.
func TestMCPToolsUsageErrorsExitTwoBeforeAnyOutput(t *testing.T) {
	for _, argv := range [][]string{
		{"mcp", "--tools="},
		{"mcp", "--tools", "documents,"},
		{"mcp", "--tools", ",documents"},
		{"mcp", "--tools", "root"},
	} {
		fake := &fakeMCP{}
		code, out, errOut := runMCPCLI(t, argv, "", nil, fake.serve)
		if code != 2 || out != "" || !strings.Contains(errOut, "invalid value for --tools") {
			t.Fatalf("%v: exit %d, stdout %q, stderr %q", argv, code, out, errOut)
		}
		if !strings.Contains(errOut, "webhook-subscriptions") {
			t.Fatalf("%v: stderr %q does not list the valid resources", argv, errOut)
		}
		if fake.calls != 0 {
			t.Fatalf("%v: the server started", argv)
		}
	}
}

// TestMCPHelpShowsTheToolsFlag checks that the help of the protocol command names the --tools flag, on standard error.
func TestMCPHelpShowsTheToolsFlag(t *testing.T) {
	_, helpOut, helpErr := runMCPCLI(t, []string{"mcp", "--help"}, "", nil, nil)
	if helpOut != "" || !strings.Contains(helpErr, "--tools") {
		t.Fatalf("help on stdout %q, stderr %q", helpOut, helpErr)
	}
}

// TestMCPWithoutAServerExitsTwo checks that a composition root that did not wire the server cannot start the command.
func TestMCPWithoutAServerExitsTwo(t *testing.T) {
	code, out, errOut := runMCPCLI(t, []string{"mcp"}, "", nil, nil)
	if code != 2 || out != "" || !strings.Contains(errOut, "not configured") {
		t.Fatalf("exit %d, stdout %q, stderr %q", code, out, errOut)
	}
}

// TestMCPServerFailureExitsTwo checks that a server that fails during the session is reported with status 2.
func TestMCPServerFailureExitsTwo(t *testing.T) {
	fake := &fakeMCP{err: errors.New("protocol broke")}
	code, _, errOut := runMCPCLI(t, []string{"mcp"}, "", nil, fake.serve)
	if code != 2 || !strings.Contains(errOut, "unexpected error") || strings.Contains(errOut, "protocol broke") {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
}

// TestMCPHelpGoesToStandardError checks that the help of the protocol command never reaches standard output, where the
// messages are.
func TestMCPHelpGoesToStandardError(t *testing.T) {
	fake := &fakeMCP{}
	_, helpOut, helpErr := runMCPCLI(t, []string{"mcp", "--help"}, "", nil, fake.serve)
	if helpOut != "" || !strings.Contains(helpErr, "USAGE: retriever mcp") {
		t.Fatalf("help on stdout %q, stderr %q", helpOut, helpErr)
	}
}

// TestMCPListedInTheCommandTree checks that the subcommand exists and is a protocol leaf.
func TestMCPListedInTheCommandTree(t *testing.T) {
	c := commandTree().child("mcp")
	if c == nil || c.leaf == nil || !c.leaf.protocol {
		t.Fatal("retriever mcp is missing or is not a protocol leaf")
	}
}

// TestStartupStatusIsTwoForEveryCommand checks the status of a failure that stops the process before Run: every command
// gives 2, whatever global flags come first (CLI convention 001).
func TestStartupStatusIsTwoForEveryCommand(t *testing.T) {
	for _, argv := range [][]string{{"mcp"}, {"--catalog", "sources.yml", "mcp"}, {"sources", "list"}, nil} {
		if got := StartupStatus(argv); got != 2 {
			t.Fatalf("StartupStatus(%v) = %d, want 2", argv, got)
		}
	}
}
