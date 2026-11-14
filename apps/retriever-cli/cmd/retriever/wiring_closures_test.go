package main

import (
	"bytes"
	"context"
	"errors"
	"net"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/graphql"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/in/cli"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/configpaths"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/sources"
)

// seamFixture is a loopback HTTPS server and the PEM file that trusts it. A test turns a seam on with these two values.
// The integration files start the server and write the file; this file never does.
type seamFixture struct {
	base string
	ca   string
}

// mapEnv reads the variables of vars, and every other variable as unset.
func mapEnv(vars map[string]string) func(string) string {
	return func(name string) string { return vars[name] }
}

// memListener is a loopback listener that never accepts a connection. Serve starts on it and stops when its context
// ends, so the serve path runs with no socket.
type memListener struct {
	done chan struct{}
	once sync.Once
}

// Accept waits until the listener is closed, and then reports that it is closed.
func (l *memListener) Accept() (net.Conn, error) {
	<-l.done
	return nil, net.ErrClosed
}

// Close ends Accept. A second Close does nothing.
func (l *memListener) Close() error {
	l.once.Do(func() { close(l.done) })
	return nil
}

// Addr returns a fixed loopback address, which Serve accepts as a loopback listener.
func (l *memListener) Addr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 40123}
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

// TestGraphQLServeReportsAnInvalidPort checks that a port outside the range fails before anything is bound, and that
// the ready callback is not called. The real listen function refuses the port before it opens a socket.
func TestGraphQLServeReportsAnInvalidPort(t *testing.T) {
	serve := graphqlServe(sources.Service{}, &bytes.Buffer{}, graphql.Listen)
	called := false
	err := serve(t.Context(), -1, func(string) { called = true })
	if err == nil || called {
		t.Fatalf("serve on port -1 = %v, ready called = %v; want an error and no ready call", err, called)
	}
}

// TestGraphQLServeStartsOnLoopbackAndStopsWhenCancelled checks that the serve function asks its listen function for the
// requested port, reports the address that listener is bound to, and returns cleanly once its context ends. The listener
// is in memory, so no socket opens; the real loopback bind is checked in the integration tests.
func TestGraphQLServeStartsOnLoopbackAndStopsWhenCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var requested int
	var addr string
	listener := &memListener{done: make(chan struct{})}
	listen := func(port int) (net.Listener, error) {
		requested = port
		return listener, nil
	}
	serve := graphqlServe(sources.Service{}, &bytes.Buffer{}, listen)
	err := serve(ctx, 0, func(bound string) {
		addr = bound
		cancel()
	})
	if err != nil {
		t.Fatalf("serve: %v", err)
	}
	if requested != 0 {
		t.Fatalf("the listen function was asked for port %d, want 0", requested)
	}
	if addr != "127.0.0.1:40123" {
		t.Fatalf("the bound address = %q, want the listener's loopback address", addr)
	}
}

// TestMCPServeWritesNothingToStandardOutputWithoutARequest checks that the MCP function ends when its input ends and writes
// nothing to the protocol channel on its own.
func TestMCPServeWritesNothingToStandardOutputWithoutARequest(t *testing.T) {
	var out bytes.Buffer
	serve := mcpServe(sources.Service{}, &bytes.Buffer{})
	_ = serve(t.Context(), nil, strings.NewReader(""), &out)
	if out.Len() != 0 {
		t.Fatalf("the MCP session wrote %q without a request", out.String())
	}
}

// TestAttachSourcesRefusesWhenTheWorkingDirectoryCannotBeRead checks that a path rule the environment cannot apply is
// recorded as the setup error, so only the sources commands fail.
func TestAttachSourcesRefusesWhenTheWorkingDirectoryCannotBeRead(t *testing.T) {
	var deps cli.Deps
	getwd := func() (string, error) { return "", os.ErrNotExist }
	err := attachSourcesFrom(&deps, cli.Globals{}, seamSet{}, emptyEnv, testHostWithGetwd(getwd))
	if err == nil || !errors.Is(err, configpaths.ErrRefused) {
		t.Fatalf("attachSourcesFrom error = %v, want configpaths.ErrRefused", err)
	}
}
