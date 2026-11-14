//go:build integration

// Package specstub is the loopback HTTPS form of the generic stub. It answers each request with the replay stub of the
// engine, so the canned exchanges of a connector spec drive an in-process test and a test over a real socket with the
// same replies. It opens a real loopback listener, so it is built only under the integration tag.
package specstub

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
	"sync/atomic"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine/replay"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// Host is the DNS name of the stub. Its certificate is valid for this name and for the loopback address, so a client that
// checks the name of its endpoint accepts it.
const Host = "wiki.example.test"

// Server is a loopback HTTPS server of one spec's exchanges. Its Roots pool trusts its certificate.
type Server struct {
	// Roots is the certificate pool that trusts the server.
	Roots *x509.CertPool
	srv   *httptest.Server
	stub  *replay.Stub
	hits  atomic.Int32
}

// Start starts a server that answers the examples of model. It registers the shutdown with t, so the caller needs no
// cleanup. A model that the replay stub refuses fails the test.
func Start(t testing.TB, model *spec.Spec) *Server {
	t.Helper()
	stub, err := replay.New(model)
	if err != nil {
		t.Fatalf("replay.New: %v", err)
	}
	cert, roots := certificate(t)
	s := &Server{Roots: roots, stub: stub}
	s.srv = httptest.NewUnstartedServer(http.HandlerFunc(s.serve))
	s.srv.TLS = &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}
	s.srv.StartTLS()
	t.Cleanup(s.srv.Close)
	return s
}

// Addr returns the loopback address and port the server listens on.
func (s *Server) Addr() string {
	return s.srv.Listener.Addr().String()
}

// Hits returns how many requests the server has received, including the ones it refused.
func (s *Server) Hits() int {
	return int(s.hits.Load())
}

// serve answers one request from the replay stub. A request with no exchange gets a 404, and the stub records it.
func (s *Server) serve(w http.ResponseWriter, r *http.Request) {
	s.hits.Add(1)
	resp, err := s.stub.RoundTrip(r)
	if err != nil {
		http.Error(w, "no canned exchange", http.StatusNotFound)
		return
	}
	defer resp.Body.Close()
	for name, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

// certificate makes a self-signed certificate for Host and the loopback address, and the pool that trusts it.
func certificate(t testing.TB) (tls.Certificate, *x509.CertPool) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: Host},
		DNSNames:     []string{Host},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1)},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("CreateCertificate: %v", err)
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("ParseCertificate: %v", err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(leaf)
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}, pool
}
