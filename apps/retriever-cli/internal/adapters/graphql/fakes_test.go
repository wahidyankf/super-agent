package graphql

import (
	"context"
	"log/slog"
	"net"
	"sync/atomic"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
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

// discardLogger returns a logger whose records go nowhere, for tests that do not read logs.
func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}
