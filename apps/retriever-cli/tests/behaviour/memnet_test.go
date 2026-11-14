package behaviour

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

// memNet is the network of the unit level. A listener opened on a loopback address is registered here, and a dial to that
// address is joined to it through an in-memory connection. No socket opens. An address no listener holds is refused, as a closed
// port is, and a busy address fails to listen, as a socket does. Every server and client of a unit scenario uses it, so no
// scenario reaches the real network.
type memNet struct {
	now func() time.Time // the clock that dates the TLS checks of the clients, so no real clock is read

	mu        sync.Mutex
	nextPort  int
	listeners map[string]*memListener
}

// newMemNet returns an empty network whose first free port is 40000, with the given clock.
func newMemNet(now func() time.Time) *memNet {
	return &memNet{now: now, nextPort: 40000, listeners: map[string]*memListener{}}
}

// open opens a listener on a loopback address. A port of 0 takes the next free port. Only the loopback host is accepted.
func (n *memNet) open(network, address string) (net.Listener, error) {
	if network != "tcp" {
		return nil, fmt.Errorf("memnet: the network %q is not tcp", network)
	}
	host, portText, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	if host != "127.0.0.1" {
		return nil, fmt.Errorf("memnet: %s is not a loopback address", host)
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 0 || port > 65535 {
		return nil, fmt.Errorf("memnet: port %q is outside the TCP range", portText)
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	if port == 0 {
		port = n.nextPort
		n.nextPort++
	}
	key := net.JoinHostPort(host, strconv.Itoa(port))
	if _, busy := n.listeners[key]; busy {
		return nil, fmt.Errorf("memnet: listen %s: address already in use", key)
	}
	l := &memListener{addr: key, owner: n, conns: make(chan net.Conn), done: make(chan struct{})}
	n.listeners[key] = l
	return l, nil
}

// listenPort opens the loopback listener of a port, as retriever serve does. A port outside the TCP range is refused
// before anything opens.
func (n *memNet) listenPort(port int) (net.Listener, error) {
	if port < 0 || port > 65535 {
		return nil, errors.New("port must be between 0 and 65535")
	}
	return n.open("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
}

// DialContext joins a dial to the listener at address. A context that ends first stops the dial with its error.
func (n *memNet) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	n.mu.Lock()
	l, ok := n.listeners[address]
	n.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("memnet: connection refused at %s, no listener holds it", address)
	}
	return l.dial(ctx)
}

// forget removes a closed listener from the network.
func (n *memNet) forget(addr string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	delete(n.listeners, addr)
}

// client returns an HTTP client that dials only through this network.
func (n *memNet) client() *http.Client {
	return &http.Client{Transport: &http.Transport{DialContext: n.DialContext}}
}

// seamClient returns the client of a seam with its dialer replaced by this network. The transport keeps its TLS settings,
// which trust the seam's CA file, and its TLS checks read the scenario clock. A client that is not an HTTP transport
// gets no trust, so its handshake fails loudly rather than reaching a real host.
func (n *memNet) seamClient(c *http.Client) *http.Client {
	tr, ok := c.Transport.(*http.Transport)
	if !ok {
		return n.client()
	}
	clone := tr.Clone()
	clone.DialContext = n.DialContext
	if clone.TLSClientConfig == nil {
		clone.TLSClientConfig = &tls.Config{}
	}
	if n.now != nil {
		clone.TLSClientConfig.Time = n.now
	}
	return &http.Client{Transport: clone, Timeout: c.Timeout}
}

// memListener is a listener of memNet. Its Accept returns the server end of each dialled pipe.
type memListener struct {
	addr  string
	owner *memNet
	conns chan net.Conn
	done  chan struct{}
	once  sync.Once
}

// Accept returns the server end of the next dialled pipe, or net.ErrClosed once the listener is closed.
func (l *memListener) Accept() (net.Conn, error) {
	select {
	case c := <-l.conns:
		return c, nil
	case <-l.done:
		return nil, net.ErrClosed
	}
}

// Close ends Accept and removes the listener from its network. A second Close does nothing.
func (l *memListener) Close() error {
	l.once.Do(func() {
		close(l.done)
		l.owner.forget(l.addr)
	})
	return nil
}

// Addr returns the loopback address of the listener.
func (l *memListener) Addr() net.Addr {
	host, portText, _ := net.SplitHostPort(l.addr)
	port, _ := strconv.Atoi(portText)
	return &net.TCPAddr{IP: net.ParseIP(host), Port: port}
}

// dial hands the server end of a new pipe to Accept and returns the client end.
func (l *memListener) dial(ctx context.Context) (net.Conn, error) {
	client, server := newBufConnPair()
	select {
	case l.conns <- server:
		return client, nil
	case <-l.done:
		_ = client.Close()
		_ = server.Close()
		return nil, net.ErrClosed
	case <-ctx.Done():
		_ = client.Close()
		_ = server.Close()
		return nil, ctx.Err()
	}
}

// bufPipe is the shared state of an in-memory duplex connection. Each end queues what it writes for its peer, so a write
// never waits for a read. A socket's send buffer behaves the same way, and a TLS handshake, where both ends write before
// either reads, needs it: a synchronous net.Pipe deadlocks there.
type bufPipe struct {
	mu     sync.Mutex
	cond   *sync.Cond
	q      [2]bytes.Buffer // q[i] holds the bytes end i reads next
	closed [2]bool         // closed[i] reports that end i was closed
	past   [2]bool         // past[i] reports that end i has a read deadline already passed
}

// bufEpoch separates a deadline that has passed from one that is still ahead. net/http sets a read deadline of
// time.Unix(1, 0) to stop a pending read, and a later deadline is not timed, because an in-memory end has no timer.
var bufEpoch = time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)

// bufConn is one end of a bufPipe.
type bufConn struct {
	p    *bufPipe
	side int
}

// newBufConnPair returns the two ends of a new in-memory connection.
func newBufConnPair() (net.Conn, net.Conn) {
	p := &bufPipe{}
	p.cond = sync.NewCond(&p.mu)
	return &bufConn{p: p, side: 0}, &bufConn{p: p, side: 1}
}

// Read returns queued bytes and waits while none are queued. A read whose deadline has passed fails with a timeout, and
// so does a read on an end that was closed. It returns io.EOF once the peer has closed with nothing left.
func (c *bufConn) Read(b []byte) (int, error) {
	c.p.mu.Lock()
	defer c.p.mu.Unlock()
	for c.p.q[c.side].Len() == 0 && !c.p.closed[c.side] && !c.p.closed[1-c.side] && !c.p.past[c.side] {
		c.p.cond.Wait()
	}
	if c.p.closed[c.side] {
		return 0, net.ErrClosed
	}
	if c.p.past[c.side] {
		return 0, os.ErrDeadlineExceeded
	}
	if c.p.q[c.side].Len() == 0 {
		return 0, io.EOF
	}
	return c.p.q[c.side].Read(b)
}

// Write queues b for the peer. It waits for nothing, and it fails once either end is closed.
func (c *bufConn) Write(b []byte) (int, error) {
	c.p.mu.Lock()
	defer c.p.mu.Unlock()
	if c.p.closed[c.side] {
		return 0, net.ErrClosed
	}
	if c.p.closed[1-c.side] {
		return 0, io.ErrClosedPipe
	}
	n, _ := c.p.q[1-c.side].Write(b)
	c.p.cond.Broadcast()
	return n, nil
}

// Close ends this end. A read waiting on it returns, and the peer sees io.EOF after its queued bytes.
func (c *bufConn) Close() error {
	c.p.mu.Lock()
	defer c.p.mu.Unlock()
	c.p.closed[c.side] = true
	c.p.cond.Broadcast()
	return nil
}

// LocalAddr returns a loopback address. No socket backs it.
func (c *bufConn) LocalAddr() net.Addr { return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)} }

// RemoteAddr returns a loopback address. No socket backs it.
func (c *bufConn) RemoteAddr() net.Addr { return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)} }

// SetDeadline sets the read deadline; writes never wait, so the write deadline does nothing. See SetReadDeadline.
func (c *bufConn) SetDeadline(t time.Time) error { return c.SetReadDeadline(t) }

// SetReadDeadline makes a deadline that has passed fail the reads of this end, and wakes the reads that wait on it. A zero
// deadline clears it. A deadline that is still ahead is not timed, so a stuck read ends only at the go test timeout.
func (c *bufConn) SetReadDeadline(t time.Time) error {
	c.p.mu.Lock()
	defer c.p.mu.Unlock()
	c.p.past[c.side] = !t.IsZero() && t.Before(bufEpoch)
	c.p.cond.Broadcast()
	return nil
}

// SetWriteDeadline does nothing, because a write never waits on the peer.
func (c *bufConn) SetWriteDeadline(time.Time) error { return nil }
