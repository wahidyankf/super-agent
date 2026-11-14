package jirastub

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"
)

// CatalogHosts are the endpoint hosts of the test catalogs. The Jira gateway checks the TLS name of its source's endpoint,
// so the certificate of a started stub is valid for each of these names.
var CatalogHosts = []string{"jira.example.test"}

// stubCertificate is the self-signed certificate that every TLS stub serves. The first caller makes it, once per process. A
// unit test replaces this variable to reach the failure paths of its callers.
var stubCertificate = sync.OnceValues(produceStubCertificate)

// listenLoopback opens the loopback listener that NewTLSServer serves on. A unit test replaces it, so no socket opens.
var listenLoopback = func() (net.Listener, error) {
	return net.Listen("tcp", "127.0.0.1:0")
}

// stubClock is the clock that dates the process certificate. A unit test replaces it, so the certificate is dated without
// reading the real clock.
var stubClock = time.Now

// produceStubCertificate makes the process certificate from the stub clock and the real entropy source.
func produceStubCertificate() (tls.Certificate, error) {
	return makeStubCertificate(stubClock(), rand.Reader, realCertificateSteps)
}

// certificateSteps are the fallible steps of makeStubCertificate. realCertificateSteps calls the standard library, and a
// unit test replaces one step to reach its failure path.
type certificateSteps struct {
	key    func(random io.Reader) (*ecdsa.PrivateKey, error)
	serial func(random io.Reader) (*big.Int, error)
	sign   func(random io.Reader, template *x509.Certificate, key *ecdsa.PrivateKey) ([]byte, error)
}

// realCertificateSteps are the standard library's steps: a P-256 key, a 62-bit serial, and a self-signed certificate.
var realCertificateSteps = certificateSteps{
	key: func(random io.Reader) (*ecdsa.PrivateKey, error) {
		return ecdsa.GenerateKey(elliptic.P256(), random)
	},
	serial: func(random io.Reader) (*big.Int, error) {
		return rand.Int(random, new(big.Int).Lsh(big.NewInt(1), 62))
	},
	sign: func(random io.Reader, template *x509.Certificate, key *ecdsa.PrivateKey) ([]byte, error) {
		return x509.CreateCertificate(random, template, template, &key.PublicKey, key)
	},
}

// makeStubCertificate returns a self-signed certificate that is valid for a year from one hour before now. It is valid for
// CatalogHosts and the loopback addresses. The certificate is its own trust root, so the PEM of the leaf is the CA file a
// client trusts.
func makeStubCertificate(now time.Time, random io.Reader, steps certificateSteps) (tls.Certificate, error) {
	key, err := steps.key(random)
	if err != nil {
		return tls.Certificate{}, err
	}
	serial, err := steps.serial(random)
	if err != nil {
		return tls.Certificate{}, err
	}
	template := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "jirastub"},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		DNSNames:              CatalogHosts,
		IPAddresses:           []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
	}
	der, err := steps.sign(random, template, key)
	if err != nil {
		return tls.Certificate{}, err
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}, nil
}

// CertificatePEM returns the PEM certificate that every TLS stub serves. A client that trusts it reaches any stub started
// in this process, and a test uses it for a seam whose server never answers.
func CertificatePEM() ([]byte, error) {
	cert, err := stubCertificate()
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]}), nil
}

// NewTLSServerWith is NewTLSServer with its listener and its certificate clock supplied by the caller. A caller that must
// not open a socket, such as a unit test, passes its own listener, and the clock that dates the certificate.
func NewTLSServerWith(h http.Handler, listen func(network, address string) (net.Listener, error), now func() time.Time) (*httptest.Server, error) {
	cert, err := makeStubCertificate(now(), rand.Reader, realCertificateSteps)
	if err != nil {
		return nil, err
	}
	listener, err := listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	srv := &httptest.Server{
		Listener: listener,
		Config:   &http.Server{Handler: h},
		TLS:      &tls.Config{Certificates: []tls.Certificate{cert}},
	}
	srv.StartTLS()
	return srv, nil
}

// NewTLSServer serves h over TLS on a loopback port with the stub certificate. The caller closes the server.
func NewTLSServer(h http.Handler) (*httptest.Server, error) {
	cert, err := stubCertificate()
	if err != nil {
		return nil, err
	}
	listener, err := listenLoopback()
	if err != nil {
		return nil, err
	}
	srv := &httptest.Server{
		Listener: listener,
		Config:   &http.Server{Handler: h},
		TLS:      &tls.Config{Certificates: []tls.Certificate{cert}},
	}
	srv.StartTLS()
	return srv, nil
}
