package jirastub

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io"
	"math/big"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

// fixedNow is the clock of every internal test, so no test reads the real clock.
var fixedNow = time.Date(2026, 11, 14, 9, 0, 0, 0, time.UTC)

// errBoom is the failure that an injected step or seam returns.
var errBoom = errors.New("injected failure")

// withStubCertificate replaces the process certificate until the test ends.
func withStubCertificate(t *testing.T, fn func() (tls.Certificate, error)) {
	t.Helper()
	saved := stubCertificate
	stubCertificate = fn
	t.Cleanup(func() { stubCertificate = saved })
}

// withListener replaces the loopback listener seam until the test ends.
func withListener(t *testing.T, fn func() (net.Listener, error)) {
	t.Helper()
	saved := listenLoopback
	listenLoopback = fn
	t.Cleanup(func() { listenLoopback = saved })
}

// cannedCertificate returns the certificate that makeStubCertificate makes for fixedNow.
func cannedCertificate(t *testing.T) tls.Certificate {
	t.Helper()
	cert, err := makeStubCertificate(fixedNow, rand.Reader, realCertificateSteps)
	if err != nil {
		t.Fatalf("makeStubCertificate: %v", err)
	}
	return cert
}

// pipeListener is a listener that opens no socket. Each dial makes an in-memory connection, and Accept hands its server end
// to the server.
type pipeListener struct {
	conns chan net.Conn
	done  chan struct{}
	once  sync.Once
}

func newPipeListener() *pipeListener {
	return &pipeListener{conns: make(chan net.Conn), done: make(chan struct{})}
}

func (l *pipeListener) Accept() (net.Conn, error) {
	select {
	case c := <-l.conns:
		return c, nil
	case <-l.done:
		return nil, net.ErrClosed
	}
}

func (l *pipeListener) Close() error {
	l.once.Do(func() { close(l.done) })
	return nil
}

func (l *pipeListener) Addr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 1}
}

// dial is the dialer of the test transport. It returns the client end of a new in-memory connection.
func (l *pipeListener) dial(context.Context, string, string) (net.Conn, error) {
	client, server := net.Pipe()
	select {
	case l.conns <- server:
		return client, nil
	case <-l.done:
		_ = client.Close()
		_ = server.Close()
		return nil, net.ErrClosed
	}
}

func TestTheStubCertificateCoversCatalogHostsAndLoopbackForAYear(t *testing.T) {
	cert := cannedCertificate(t)
	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatalf("parse the certificate: %v", err)
	}
	if !leaf.NotBefore.Equal(fixedNow.Add(-time.Hour)) || !leaf.NotAfter.Equal(fixedNow.Add(365*24*time.Hour)) {
		t.Fatalf("validity %v to %v, want one hour before and a year after the clock", leaf.NotBefore, leaf.NotAfter)
	}
	roots := x509.NewCertPool()
	roots.AddCert(leaf)
	for _, host := range append(append([]string(nil), CatalogHosts...), "127.0.0.1", "::1") {
		if _, err := leaf.Verify(x509.VerifyOptions{
			Roots: roots, CurrentTime: fixedNow, DNSName: host, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		}); err != nil {
			t.Errorf("certificate does not verify for %s: %v", host, err)
		}
	}
}

func TestMakeStubCertificateReportsTheStepThatFails(t *testing.T) {
	tests := []struct {
		name   string
		injure func(*certificateSteps)
	}{
		{name: "the key step", injure: func(s *certificateSteps) {
			s.key = func(io.Reader) (*ecdsa.PrivateKey, error) { return nil, errBoom }
		}},
		{name: "the serial step", injure: func(s *certificateSteps) {
			s.serial = func(io.Reader) (*big.Int, error) { return nil, errBoom }
		}},
		{name: "the signing step", injure: func(s *certificateSteps) {
			s.sign = func(io.Reader, *x509.Certificate, *ecdsa.PrivateKey) ([]byte, error) { return nil, errBoom }
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps := realCertificateSteps
			tt.injure(&steps)
			cert, err := makeStubCertificate(fixedNow, rand.Reader, steps)
			if !errors.Is(err, errBoom) {
				t.Fatalf("error = %v, want the injected failure", err)
			}
			if cert.Certificate != nil || cert.PrivateKey != nil {
				t.Fatal("a failed step still returned a certificate")
			}
		})
	}
}

func TestCertificatePEMEncodesTheProcessCertificateOrReportsItsError(t *testing.T) {
	cert := cannedCertificate(t)
	withStubCertificate(t, func() (tls.Certificate, error) { return cert, nil })
	data, err := CertificatePEM()
	block, _ := pem.Decode(data)
	if err != nil || block == nil || block.Type != "CERTIFICATE" || !bytes.Equal(block.Bytes, cert.Certificate[0]) {
		t.Fatalf("CertificatePEM = %q, %v; want the PEM of the process certificate", data, err)
	}

	withStubCertificate(t, func() (tls.Certificate, error) { return tls.Certificate{}, errBoom })
	if _, err := CertificatePEM(); !errors.Is(err, errBoom) {
		t.Fatalf("CertificatePEM error = %v, want the injected failure", err)
	}
}

func TestNewTLSServerServesTheStubOverTLSOnTheListenerItIsGiven(t *testing.T) {
	cert := cannedCertificate(t)
	withStubCertificate(t, func() (tls.Certificate, error) { return cert, nil })
	ln := newPipeListener()
	withListener(t, func() (net.Listener, error) { return ln, nil })
	stub := New("fixture-reviewer@example.com", "sentinel-token-5d2b8e-do-not-print")
	stub.SetClock(func() time.Time { return fixedNow })

	srv, err := NewTLSServer(stub)
	if err != nil {
		t.Fatalf("NewTLSServer: %v", err)
	}
	defer srv.Close()

	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatalf("parse the certificate: %v", err)
	}
	roots := x509.NewCertPool()
	roots.AddCert(leaf)
	transport := &http.Transport{
		DialContext: ln.dial,
		TLSClientConfig: &tls.Config{
			RootCAs: roots, MinVersion: tls.VersionTLS12, Time: func() time.Time { return fixedNow },
		},
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport}

	resp, err := client.Get(srv.URL + "/rest/api/3/myself")
	if err != nil {
		t.Fatalf("unauthenticated GET over TLS: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d, want 401", resp.StatusCode)
	}

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/rest/api/3/myself", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.SetBasicAuth("fixture-reviewer@example.com", "sentinel-token-5d2b8e-do-not-print")
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("signed GET over TLS: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "Fixture Reviewer") {
		t.Fatalf("signed status %d, body %s; want the signed-in account", resp.StatusCode, body)
	}

	recorded := stub.Requests()
	if len(recorded) != 2 || !recorded[0].Secure || recorded[0].AuthOK || !recorded[1].AuthOK || !recorded[1].Secure {
		t.Fatalf("recorded %+v, want two TLS requests, the second signed", recorded)
	}
}

func TestNewTLSServerReportsAFailedCertificateOrListener(t *testing.T) {
	cert := cannedCertificate(t)
	tests := []struct {
		name        string
		certificate func() (tls.Certificate, error)
		listen      func() (net.Listener, error)
		wantListen  bool
	}{
		{
			name:        "a certificate that fails stops before any listener opens",
			certificate: func() (tls.Certificate, error) { return tls.Certificate{}, errBoom },
		},
		{
			name:        "a listener that fails is reported",
			certificate: func() (tls.Certificate, error) { return cert, nil },
			listen:      func() (net.Listener, error) { return nil, errBoom },
			wantListen:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withStubCertificate(t, tt.certificate)
			listened := false
			withListener(t, func() (net.Listener, error) {
				listened = true
				if tt.listen != nil {
					return tt.listen()
				}
				return nil, errBoom
			})
			srv, err := NewTLSServer(New("a@example.test", "t"))
			if !errors.Is(err, errBoom) || srv != nil {
				t.Fatalf("NewTLSServer = %v, %v; want the injected failure and no server", srv, err)
			}
			if listened != tt.wantListen {
				t.Fatalf("listener requested = %v, want %v", listened, tt.wantListen)
			}
		})
	}
}
