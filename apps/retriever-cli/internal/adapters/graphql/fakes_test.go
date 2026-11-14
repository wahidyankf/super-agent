package graphql

import (
	"context"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// countingGateway is an in-memory ports.SourceGateway that counts every upstream call.
// The counter is atomic, so it stays correct when a served request runs on another goroutine.
type countingGateway struct {
	collections []domain.Collection
	tree        []domain.TreeNode
	doc         domain.SourceDocument
	err         error
	calls       atomic.Int32
}

func (g *countingGateway) ListCollections(context.Context) ([]domain.Collection, error) {
	g.calls.Add(1)
	return g.collections, g.err
}

func (g *countingGateway) CollectionTree(context.Context, domain.CollectionID) ([]domain.TreeNode, error) {
	g.calls.Add(1)
	return g.tree, g.err
}

func (g *countingGateway) Document(context.Context, domain.DocumentID) (domain.SourceDocument, error) {
	g.calls.Add(1)
	return g.doc, g.err
}

// fakeListener is a net.Listener with a fixed address and no socket. Unit tests use it to
// reach the served port without binding one. It counts Accept calls, so a test can prove
// that Serve never started to accept connections.
type fakeListener struct {
	addr    net.Addr
	accepts atomic.Int32
}

func (l *fakeListener) Accept() (net.Conn, error) {
	l.accepts.Add(1)
	return nil, net.ErrClosed
}

func (l *fakeListener) Close() error   { return nil }
func (l *fakeListener) Addr() net.Addr { return l.addr }

// tcpListener returns a fake listener on the given IP and port.
func tcpListener(ip string, port int) *fakeListener {
	return &fakeListener{addr: &net.TCPAddr{IP: net.ParseIP(ip), Port: port}}
}

// pipeListener is a loopback net.Listener whose connections are in-memory pipes. Its dial hands the server end of a new
// pipe to Accept and returns the client end, so a test serves and queries HTTP with no socket.
type pipeListener struct {
	conns chan net.Conn
	done  chan struct{}
	once  sync.Once
}

// newPipeListener returns an open pipe listener.
func newPipeListener() *pipeListener {
	return &pipeListener{conns: make(chan net.Conn), done: make(chan struct{})}
}

// Accept returns the server end of the next dialled pipe, or net.ErrClosed after Close.
func (l *pipeListener) Accept() (net.Conn, error) {
	select {
	case c := <-l.conns:
		return c, nil
	case <-l.done:
		return nil, net.ErrClosed
	}
}

// Close ends Accept. A second Close does nothing.
func (l *pipeListener) Close() error {
	l.once.Do(func() { close(l.done) })
	return nil
}

// Addr returns the loopback address the listener reports, so Serve accepts it as a loopback listener.
func (l *pipeListener) Addr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP(loopbackIP), Port: testPort}
}

// dial returns the client end of a new pipe whose server end Accept receives.
func (l *pipeListener) dial(context.Context, string, string) (net.Conn, error) {
	client, server := net.Pipe()
	select {
	case l.conns <- server:
		return client, nil
	case <-l.done:
		return nil, net.ErrClosed
	}
}

// discardLogger returns a logger whose records go nowhere, for tests that do not read logs.
func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}
