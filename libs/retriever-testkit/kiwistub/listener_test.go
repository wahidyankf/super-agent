package kiwistub

import (
	"bufio"
	"net"
	"net/http/httptest"
	"sync"
	"testing"
)

// fakeListener is an in-memory net.Listener. It hands out no connection, so a started stub serves nothing and opens no
// socket. The started channel closes at the first Accept, so a test knows the server is serving before it closes it. The
// closed channel closes when the server closes the listener.
type fakeListener struct {
	addr      net.Addr
	started   chan struct{}
	closed    chan struct{}
	startOnce sync.Once
	closeOnce sync.Once
}

func newFakeListener(port int) *fakeListener {
	return &fakeListener{
		addr:    &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: port},
		started: make(chan struct{}),
		closed:  make(chan struct{}),
	}
}

// Accept blocks until the listener is closed, and then reports that it is closed.
func (l *fakeListener) Accept() (net.Conn, error) {
	l.startOnce.Do(func() { close(l.started) })
	<-l.closed
	return nil, net.ErrClosed
}

// Close marks the listener closed. A second Close does nothing.
func (l *fakeListener) Close() error {
	l.closeOnce.Do(func() { close(l.closed) })
	return nil
}

// Addr returns the loopback address the fake listener reports.
func (l *fakeListener) Addr() net.Addr {
	return l.addr
}

// isClosed reports whether Close has been called, without waiting.
func (l *fakeListener) isClosed() bool {
	select {
	case <-l.closed:
		return true
	default:
		return false
	}
}

// startFake starts s on a fake listener at port 4321, and waits until the server is accepting, so that Close has a
// listener to stop. The wait is a channel receive, never a timer.
func startFake(t *testing.T, s *Stub) *fakeListener {
	t.Helper()
	ln := newFakeListener(4321)
	s.sys.listen = func(string, string) (net.Listener, error) { return ln, nil }
	if err := s.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	<-ln.started
	return ln
}

// fakeConn is a connection that records Close and does nothing else.
type fakeConn struct {
	net.Conn
	closed bool
}

func (c *fakeConn) Close() error {
	c.closed = true
	return nil
}

// hijackRecorder is a response recorder that can be hijacked. When err is set, Hijack fails. A recorder without this
// type is one that cannot be hijacked.
type hijackRecorder struct {
	*httptest.ResponseRecorder
	conn *fakeConn
	err  error
}

func (w *hijackRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if w.err != nil {
		return nil, nil, w.err
	}
	return w.conn, nil, nil
}
