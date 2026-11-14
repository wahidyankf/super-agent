package outlinestub

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"time"
)

// CatalogHosts are the endpoint hosts of the test catalogs. A gateway checks the TLS name of its source's endpoint,
// so the certificate of a started stub is valid for each of these names.
var CatalogHosts = []string{"docs.example.test", "wiki.engineering.example.com"}

// The seams of a started TLS server: the listener it serves on and the clock that dates its certificate. The defaults
// open a real loopback port and read the real clock. A unit test replaces them, so it opens no socket.
var (
	listenTCP = net.Listen
	tlsNow    = time.Now
)

// generateKey makes the P-256 key of the stub certificate. It is a variable so a unit test can fail the key step: the
// standard library's key generation does not always read the entropy source it is given.
var generateKey = func(random io.Reader) (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(elliptic.P256(), random)
}

// makeStubCertificate returns a self-signed certificate that is valid for CatalogHosts and the loopback addresses. It is
// dated from an hour before now for a year. The certificate is its own trust root, so the PEM of the leaf is the CA file
// a client trusts. random supplies the key and the serial number.
func makeStubCertificate(now time.Time, random io.Reader) (tls.Certificate, error) {
	key, err := generateKey(random)
	if err != nil {
		return tls.Certificate{}, err
	}
	serial, err := rand.Int(random, new(big.Int).Lsh(big.NewInt(1), 62))
	if err != nil {
		return tls.Certificate{}, err
	}
	template := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "outlinestub"},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		DNSNames:              CatalogHosts,
		IPAddresses:           []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
	}
	der, err := x509.CreateCertificate(random, template, template, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, err
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}, nil
}

// NewTLSServer serves h over TLS on a loopback port with the stub certificate. The caller closes the server. It is
// the TLS server of Start, and of the engines that have no testing.TB.
func NewTLSServer(h http.Handler) (*httptest.Server, error) {
	return NewTLSServerWith(h, listenTCP, tlsNow)
}

// NewTLSServerWith is NewTLSServer with its effects supplied by the caller: listen opens the listener, and now dates
// the certificate. A caller that must not open a socket, such as a unit test, passes its own listener.
func NewTLSServerWith(h http.Handler, listen func(network, address string) (net.Listener, error), now func() time.Time) (*httptest.Server, error) {
	return newTLSServer(h, listen, now, rand.Reader)
}

// newTLSServer is NewTLSServerWith with the entropy source of the certificate supplied too. The certificate is made
// before the listener opens, so a certificate that cannot be made never leaves a socket behind.
func newTLSServer(h http.Handler, listen func(network, address string) (net.Listener, error), now func() time.Time, random io.Reader) (*httptest.Server, error) {
	cert, err := makeStubCertificate(now(), random)
	if err != nil {
		return nil, err
	}
	ln, err := listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	srv := &httptest.Server{
		Listener: ln,
		Config:   &http.Server{Handler: h},
		TLS:      &tls.Config{Certificates: []tls.Certificate{cert}},
	}
	srv.StartTLS()
	return srv, nil
}
