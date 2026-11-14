package graphql

import (
	"context"
	"net"
	"testing"

	usecases "github.com/wahidyankf/super-agent/libs/retriever-runtime/sources"
)

// unixListener is a listener whose address is not TCP. The GraphQL handler refuses such a listener, so the serve path
// must close it and report the refusal. It never accepts a connection.
type unixListener struct{}

func (unixListener) Accept() (net.Conn, error) { return nil, net.ErrClosed }
func (unixListener) Close() error              { return nil }
func (unixListener) Addr() net.Addr            { return &net.UnixAddr{Name: "synthetic", Net: "unix"} }

// TestServeOnRefusesAListenerThatIsNotTCP checks that a listener whose address is not TCP is refused, and that the refusal
// is reported before the ready callback runs.
func TestServeOnRefusesAListenerThatIsNotTCP(t *testing.T) {
	ready := false
	err := serveOn(context.Background(), unixListener{}, usecases.Service{}, nil, func(string) { ready = true })
	if err == nil {
		t.Fatal("serveOn accepted a listener whose address is not TCP")
	}
	if ready {
		t.Fatal("ready ran for a listener that was refused")
	}
}
