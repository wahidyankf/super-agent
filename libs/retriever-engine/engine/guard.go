package engine

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"net/netip"
	"strings"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// sharedSpace is the carrier-grade NAT range, which is not public and is not in the standard private ranges.
var sharedSpace = netip.MustParsePrefix("100.64.0.0/10")

// guardedTransport returns the transport that every request of the engine uses. Each connection is dialed through
// guardDial, so no request can reach an address that the check refused. The transport never reads a proxy from the
// environment, and it requires TLS 1.2 or later.
func guardedTransport(n Network, allowPrivate bool, roots *x509.CertPool) *http.Transport {
	return &http.Transport{
		// Proxy stays nil, so the environment cannot route a request around the dial check.
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			return guardDial(ctx, network, address, n, allowPrivate)
		},
		TLSClientConfig:   &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: roots},
		DisableKeepAlives: true,
	}
}

// guardDial connects to the host of address. It refuses an IP literal and a local name before any lookup. It looks up
// the host once, checks every address it gets, and dials the first address that connects. It never dials the host
// name, so the address it connects to is the address it checked.
func guardDial(ctx context.Context, network, address string, n Network, allowPrivate bool) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, ErrNetworkRefused
	}
	if _, err := netip.ParseAddr(host); err == nil {
		return nil, ErrNetworkRefused
	}
	if isLocalName(host) {
		return nil, ErrNetworkRefused
	}
	addrs, err := n.Lookup(ctx, host)
	if err != nil {
		return nil, err
	}
	if len(addrs) == 0 {
		return nil, domain.ErrUnreachable
	}
	for _, a := range addrs {
		if err := checkAddress(a, allowPrivate); err != nil {
			return nil, err
		}
	}
	var last error
	for _, a := range addrs {
		conn, err := n.Dial(ctx, network, net.JoinHostPort(a.Unmap().String(), port))
		if err == nil {
			return conn, nil
		}
		last = err
	}
	return nil, last
}

// isLocalName reports whether a host is localhost or a name under it. Such a name is refused even with the catalog
// setting, because it names this machine.
func isLocalName(host string) bool {
	h := strings.ToLower(strings.TrimSuffix(host, "."))
	return h == "localhost" || strings.HasSuffix(h, ".localhost")
}

// checkAddress reports whether the engine may dial an address. An unspecified address and a multicast address are
// always refused. A private, loopback, link-local, or shared-space address is refused unless the catalog allows
// private networks.
func checkAddress(a netip.Addr, allowPrivate bool) error {
	if !a.IsValid() {
		return ErrNetworkRefused
	}
	a = a.Unmap()
	if a.IsUnspecified() || a.IsMulticast() {
		return ErrNetworkRefused
	}
	if allowPrivate {
		return nil
	}
	if a.IsLoopback() || a.IsPrivate() || a.IsLinkLocalUnicast() || sharedSpace.Contains(a) {
		return ErrNetworkRefused
	}
	return nil
}
