package kiwistub

import (
	"bytes"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"time"
)

// CatalogHosts are the endpoint hosts of the test catalogs. The adapter checks the TLS name of its source's endpoint, so
// the certificate of a started stub is valid for each of these names and for the loopback addresses.
var CatalogHosts = []string{"kiwi.example.test", "kiwi-archive.example.test"}

// errNotStarted reports a call that needs a started TLS server.
var errNotStarted = errors.New("kiwistub: the stub is not started")

// errDropped is the transport failure of a Drop fault seen in process, as a closed connection is seen over TLS.
var errDropped = errors.New("kiwistub: the connection was dropped")

// WithListener sets the function that opens the listening socket of Start. The default is net.Listen, and Start asks it
// for the loopback address only. A test of another package passes a fake listener, so it runs a command without a socket.
func WithListener(listen func(network, address string) (net.Listener, error)) Option {
	return func(s *Stub) { s.sys.listen = listen }
}

// makeCertificate returns a self-signed certificate valid for CatalogHosts and the loopback addresses, with its parsed
// leaf. Its validity starts an hour before the stub's clock and lasts a year. The certificate is its own trust root, so
// the leaf is the CA that a client trusts. The serial is fixed because the stub makes one certificate with its own key,
// and only that stub trusts it.
func (s *Stub) makeCertificate() (tls.Certificate, *x509.Certificate, error) {
	key, err := s.sys.genKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, nil, err
	}
	now := s.sys.now()
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "kiwistub"},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		DNSNames:              CatalogHosts,
		IPAddresses:           []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
	}
	der, err := s.sys.sign(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, nil, err
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		return tls.Certificate{}, nil, err
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key, Leaf: leaf}, leaf, nil
}

// Start serves the stub over TLS on a loopback port. The caller calls Close. It returns an error when the certificate
// cannot be made or the port cannot be opened.
func (s *Stub) Start() error {
	cert, leaf, err := s.makeCertificate()
	if err != nil {
		return fmt.Errorf("make the TLS certificate: %w", err)
	}
	ln, err := s.sys.listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("listen on loopback: %w", err)
	}
	server := &http.Server{
		Handler:   s,
		TLSConfig: &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12},
	}
	s.server, s.leaf, s.url = server, leaf, "https://"+ln.Addr().String()
	go func() {
		// Serve returns when Close stops the server, or when the listener fails. A failed listener surfaces to the
		// client as a transport error, so the error is not kept here.
		_ = server.Serve(tls.NewListener(ln, server.TLSConfig))
	}()
	return nil
}

// URL returns the base URL of a started stub, such as https://127.0.0.1:43123.
func (s *Stub) URL() string {
	return s.url
}

// CAPEM returns the PEM certificate a client must trust to reach a started stub. It is nil before Start.
func (s *Stub) CAPEM() []byte {
	if s.leaf == nil {
		return nil
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: s.leaf.Raw})
}

// WriteCAFile writes the CA PEM to path, so a process can trust the stub.
func (s *Stub) WriteCAFile(path string) error {
	pemBytes := s.CAPEM()
	if pemBytes == nil {
		return errNotStarted
	}
	return s.sys.writeFile(path, pemBytes, 0o600)
}

// Client returns an HTTP client that trusts the stub's certificate and uses no proxy. It is for tests that connect to a
// started stub; the adapter copies it and replaces its redirect policy.
func (s *Stub) Client() (*http.Client, error) {
	if s.leaf == nil {
		return nil, errNotStarted
	}
	pool := x509.NewCertPool()
	pool.AddCert(s.leaf)
	return &http.Client{Transport: &http.Transport{
		Proxy:           nil,
		TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12},
	}}, nil
}

// Close stops a started stub. It is a no-op for a stub that was never started.
func (s *Stub) Close() {
	if s.server != nil {
		_ = s.server.Close()
	}
}

// ServeHTTP answers one request of the TLS server.
func (s *Stub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	s.begin()
	defer s.end()
	rep := s.answerWire(wire{
		verb: r.Method, path: r.URL.Path, host: r.Host, auth: r.Header.Get("Authorization"),
		contentType: r.Header.Get("Content-Type"), secure: r.TLS != nil, body: body,
	})
	if rep.hold {
		<-r.Context().Done()
		return
	}
	if rep.drop {
		if hijacker, ok := w.(http.Hijacker); ok {
			if conn, _, err := hijacker.Hijack(); err == nil {
				_ = conn.Close()
			}
		}
		return
	}
	for key, values := range rep.header {
		for _, v := range values {
			w.Header().Add(key, v)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(rep.status)
	_, _ = w.Write(rep.body)
}

// RoundTripper returns a transport that answers in process, with no socket. Tests build the adapter's client from it, so
// the adapter's own checks run against the stub's replies.
func (s *Stub) RoundTripper() http.RoundTripper {
	return inProcess{s: s}
}

// inProcess is the in-process transport of a stub.
type inProcess struct {
	s *Stub
}

// RoundTrip answers one request. A request that the stub never answers ends when its context ends.
func (p inProcess) RoundTrip(req *http.Request) (*http.Response, error) {
	var body []byte
	if req.Body != nil {
		body, _ = io.ReadAll(req.Body)
		_ = req.Body.Close()
	}
	p.s.begin()
	defer p.s.end()
	rep := p.s.answerWire(wire{
		verb: req.Method, path: req.URL.Path, host: req.Host, auth: req.Header.Get("Authorization"),
		contentType: req.Header.Get("Content-Type"), secure: req.URL.Scheme == "https", body: body,
	})
	if rep.hold {
		<-req.Context().Done()
		return nil, req.Context().Err()
	}
	if rep.drop {
		return nil, errDropped
	}
	header := rep.header
	if header == nil {
		header = http.Header{}
	}
	header.Set("Content-Type", "application/json")
	return &http.Response{
		Status:        http.StatusText(rep.status),
		StatusCode:    rep.status,
		Header:        header,
		Body:          io.NopCloser(bytes.NewReader(rep.body)),
		ContentLength: int64(len(rep.body)),
		Request:       req,
	}, nil
}
