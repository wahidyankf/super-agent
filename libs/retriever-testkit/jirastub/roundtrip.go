package jirastub

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
)

// errNotAStubHost reports a request to a host that the stub does not serve. Only a loopback address and a name that ends in
// .invalid are served, so the in-process stub never answers for a real site.
var errNotAStubHost = errors.New("jirastub: the request names a host the stub does not serve")

// errConnectionDropped is what the in-process transport returns for a request that a Drop fault closed with no reply. It is
// not a timeout, so the client reports the server as unreachable.
var errConnectionDropped = errors.New("jirastub: the connection was closed before a reply")

// RoundTripper returns a transport that answers in-process through ServeHTTP, with no socket. Unit tests use it. It refuses
// a request to any host that is neither a loopback address nor a name ending in .invalid. A Hang fault makes the request time
// out at once, as a server that never answers would after the client's limit, so no test waits.
func (s *Stub) RoundTripper() http.RoundTripper {
	return inProcess{stub: s}
}

// inProcess is the RoundTripper of an in-process Stub.
type inProcess struct {
	stub *Stub
}

// RoundTrip answers one request synchronously, with the stub's handler, and returns the recorded reply.
func (p inProcess) RoundTrip(req *http.Request) (*http.Response, error) {
	if !stubHost(req.URL.Hostname()) {
		return nil, errNotAStubHost
	}
	ctx, ctl := withControl(req.Context())
	rec := httptest.NewRecorder()
	p.stub.ServeHTTP(rec, req.WithContext(ctx))
	switch {
	case ctl.hung:
		return nil, context.DeadlineExceeded
	case ctl.dropped:
		return nil, errConnectionDropped
	}
	resp := rec.Result()
	resp.Request = req
	return resp, nil
}

// stubHost reports whether host is one the in-process stub serves: a loopback IP address, or a .invalid name.
func stubHost(host string) bool {
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}
	return strings.HasSuffix(host, ".invalid")
}
