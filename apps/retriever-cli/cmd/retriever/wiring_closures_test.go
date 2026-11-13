package main

import (
	"bytes"
	"context"
	"errors"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/configpaths"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/in/cli"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/outline"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/sources"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
)

// seamFixture is a loopback HTTPS server and the PEM file that trusts it. A test turns a seam on with these two values.
type seamFixture struct {
	base string
	ca   string
}

// mapEnv reads the variables of vars, and every other variable as unset.
func mapEnv(vars map[string]string) func(string) string {
	return func(name string) string { return vars[name] }
}

// TestReadSeamRefusesAKindWithNoSeam checks that a kind with no seam, here the zero kind, is an error and never a silent
// default.
func TestReadSeamRefusesAKindWithNoSeam(t *testing.T) {
	if _, err := readSeam(domain.SourceKind{}, emptyEnv); err == nil {
		t.Fatal("readSeam for a kind with no seam returned no error")
	}
}

// TestCheckSeamsRejectsAnOnSeamWithABadBase checks that a seam that is on is validated per kind at setup.
func TestCheckSeamsRejectsAnOnSeamWithABadBase(t *testing.T) {
	for _, kind := range seamKinds {
		seams := seamSet{kind: seamConfig{on: true, base: "http://127.0.0.1:1"}}
		if err := checkSeams(seams); err == nil {
			t.Fatalf("checkSeams accepted an http base for the %s seam", kind)
		}
	}
}

// client returns an HTTP client that trusts the fixture's certificate, as the seam does.
func (f seamFixture) client(t *testing.T) *http.Client {
	t.Helper()
	seam, err := outline.TestSeamFromEnv(mapEnv(map[string]string{outline.BaseURLEnv: f.base, outline.CAFileEnv: f.ca}))
	if err != nil {
		t.Fatal(err)
	}
	return seam.Client
}

// TestGraphQLServeReportsAnInvalidPort checks that a port outside the range fails before anything is bound, and that
// the ready callback is not called.
func TestGraphQLServeReportsAnInvalidPort(t *testing.T) {
	serve := graphqlServe(sources.Service{})
	called := false
	err := serve(t.Context(), -1, func(string) { called = true })
	if err == nil || called {
		t.Fatalf("serve on port -1 = %v, ready called = %v; want an error and no ready call", err, called)
	}
}

// TestGraphQLServeReportsABusyPort checks that a port another process holds fails the serve command.
func TestGraphQLServeReportsABusyPort(t *testing.T) {
	held, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = held.Close() })
	port := held.Addr().(*net.TCPAddr).Port
	serve := graphqlServe(sources.Service{})
	if err := serve(t.Context(), port, func(string) {}); err == nil {
		t.Fatal("serve on a busy port returned no error")
	}
}

// TestGraphQLServeStartsOnLoopbackAndStopsWhenCancelled checks that the serve function binds a loopback address, reports
// it, and returns cleanly once its context ends.
func TestGraphQLServeStartsOnLoopbackAndStopsWhenCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var addr string
	serve := graphqlServe(sources.Service{})
	err := serve(ctx, 0, func(bound string) {
		addr = bound
		cancel()
	})
	if err != nil {
		t.Fatalf("serve: %v", err)
	}
	if !strings.HasPrefix(addr, "127.0.0.1:") {
		t.Fatalf("the bound address = %q, want a loopback address", addr)
	}
}

// TestMCPServeWritesNothingToStandardOutputWithoutARequest checks that the MCP function ends when its input ends and writes
// nothing to the protocol channel on its own.
func TestMCPServeWritesNothingToStandardOutputWithoutARequest(t *testing.T) {
	var out bytes.Buffer
	serve := mcpServe(sources.Service{}, &bytes.Buffer{})
	_ = serve(t.Context(), domain.AccessRead, nil, strings.NewReader(""), &out)
	if out.Len() != 0 {
		t.Fatalf("the MCP session wrote %q without a request", out.String())
	}
}

// TestAttachSourcesRefusesWhenTheWorkingDirectoryCannotBeRead checks that a path rule the environment cannot apply is
// recorded as the setup error, so only the sources commands fail.
func TestAttachSourcesRefusesWhenTheWorkingDirectoryCannotBeRead(t *testing.T) {
	var deps cli.Deps
	getwd := func() (string, error) { return "", os.ErrNotExist }
	err := attachSourcesFrom(&deps, cli.Globals{}, seamSet{}, emptyEnv, getwd)
	if err == nil || !errors.Is(err, configpaths.ErrRefused) {
		t.Fatalf("attachSourcesFrom error = %v, want configpaths.ErrRefused", err)
	}
}
