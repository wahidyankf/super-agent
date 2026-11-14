package outlinestub

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"
)

// testInstant is the clock every TLS test in this package runs on. No test reads the real clock.
var testInstant = time.Date(2026, time.November, 14, 9, 0, 0, 0, time.UTC)

func fixedClock() time.Time { return testInstant }

// memListener is a listener that holds no socket. Accept blocks until Close, and nothing is ever accepted, so a test
// can start a TLS server, read its address and certificate, and close it without any network.
type memListener struct {
	addr net.Addr
	done chan struct{}
	once sync.Once
}

func newMemListener() *memListener {
	return &memListener{addr: &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 4443}, done: make(chan struct{})}
}

func (l *memListener) Accept() (net.Conn, error) {
	<-l.done
	return nil, net.ErrClosed
}

func (l *memListener) Close() error {
	l.once.Do(func() { close(l.done) })
	return nil
}

func (l *memListener) Addr() net.Addr { return l.addr }

func (l *memListener) closed() bool {
	select {
	case <-l.done:
		return true
	default:
		return false
	}
}

// useMemListener makes the next started TLS server open l instead of a socket, and it returns to the defaults when the
// test ends. The tests of this package do not run in parallel, because the seams are package variables.
func useMemListener(t *testing.T, l *memListener) {
	t.Helper()
	oldListen, oldNow := listenTCP, tlsNow
	t.Cleanup(func() { listenTCP, tlsNow = oldListen, oldNow })
	listenTCP = func(network, address string) (net.Listener, error) {
		if network != "tcp" || address != "127.0.0.1:0" {
			return nil, fmt.Errorf("unexpected listen of %s %s", network, address)
		}
		return l, nil
	}
	tlsNow = fixedClock
}

// recordingTB is a testing.TB that records what Start asks of its test. Fatalf stops the call with a panic, as the
// real Fatalf stops the test goroutine.
type recordingTB struct {
	testing.TB
	cleanups []func()
	fatal    string
}

type fatalStop struct{}

func (tb *recordingTB) Helper() {}

func (tb *recordingTB) Cleanup(f func()) { tb.cleanups = append(tb.cleanups, f) }

func (tb *recordingTB) Fatalf(format string, args ...any) {
	tb.fatal = fmt.Sprintf(format, args...)
	panic(fatalStop{})
}

// startFatals runs Start and reports whether it stopped the test with Fatalf.
func startFatals(tb testing.TB) (stopped bool) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(fatalStop); !ok {
				panic(r)
			}
			stopped = true
		}
	}()
	Start(tb)
	return false
}

// leafCertificate parses the PEM a started stub hands to its clients.
func leafCertificate(t *testing.T, data []byte) *x509.Certificate {
	t.Helper()
	block, rest := pem.Decode(data)
	if block == nil || block.Type != "CERTIFICATE" || len(rest) != 0 {
		t.Fatalf("CAPEM is not one PEM certificate block: %q", data)
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("ParseCertificate: %v", err)
	}
	return cert
}

func TestStartedCertificateCoversEveryCatalogHost(t *testing.T) {
	useMemListener(t, newMemListener())
	stub := Start(t)
	data, err := stub.CAPEM()
	if err != nil {
		t.Fatalf("CAPEM: %v", err)
	}
	cert := leafCertificate(t, data)
	for _, host := range append(append([]string(nil), CatalogHosts...), "127.0.0.1", "::1") {
		if err := cert.VerifyHostname(host); err != nil {
			t.Errorf("the stub certificate does not cover %q: %v", host, err)
		}
	}
	if err := cert.VerifyHostname("not-a-catalog-host.example.org"); err == nil {
		t.Error("the stub certificate covers a host it was not made for")
	}
}

func TestStartedCertificateIsDatedFromTheInjectedClock(t *testing.T) {
	useMemListener(t, newMemListener())
	stub := Start(t)
	data, err := stub.CAPEM()
	if err != nil {
		t.Fatalf("CAPEM: %v", err)
	}
	cert := leafCertificate(t, data)
	if want := testInstant.Add(-time.Hour); !cert.NotBefore.Equal(want) {
		t.Errorf("NotBefore = %v, want %v", cert.NotBefore, want)
	}
	if want := testInstant.Add(365 * 24 * time.Hour); !cert.NotAfter.Equal(want) {
		t.Errorf("NotAfter = %v, want %v", cert.NotAfter, want)
	}
	if !cert.IsCA || cert.KeyUsage&x509.KeyUsageCertSign == 0 {
		t.Error("the stub certificate is not its own CA, so a client cannot trust its PEM")
	}
}

func TestStartedStubReportsItsListenerAddressAndTrustsItsCertificate(t *testing.T) {
	useMemListener(t, newMemListener())
	stub := Start(t)
	if got, want := stub.URL(), "https://127.0.0.1:4443"; got != want {
		t.Fatalf("URL = %q, want %q", got, want)
	}
	client := stub.Client()
	if client == nil {
		t.Fatal("a started stub has no client")
	}
	transport, ok := client.Transport.(*http.Transport)
	if !ok || transport.TLSClientConfig == nil || transport.TLSClientConfig.RootCAs == nil {
		t.Fatal("the client of a started stub does not trust a root CA")
	}
}

func TestStartedStubClosesItsListenerWhenTheTestEnds(t *testing.T) {
	l := newMemListener()
	useMemListener(t, l)
	tb := &recordingTB{TB: t}
	stub := Start(tb)
	if l.closed() {
		t.Fatal("the listener closed before the test ended")
	}
	if len(tb.cleanups) != 1 {
		t.Fatalf("Start registered %d cleanups, want 1", len(tb.cleanups))
	}
	tb.cleanups[0]()
	if !l.closed() {
		t.Fatal("the listener is still open after the test's cleanup")
	}
	stub.Close()
}

func TestStartFailsTheTestWhenTheListenerCannotOpen(t *testing.T) {
	errListen := errors.New("no port for the stub")
	oldListen, oldNow := listenTCP, tlsNow
	t.Cleanup(func() { listenTCP, tlsNow = oldListen, oldNow })
	listenTCP = func(string, string) (net.Listener, error) { return nil, errListen }
	tlsNow = fixedClock

	tb := &recordingTB{TB: t}
	if !startFatals(tb) {
		t.Fatal("Start returned without failing the test")
	}
	if want := "outlinestub: start the TLS server: no port for the stub"; tb.fatal != want {
		t.Fatalf("Fatalf message = %q, want %q", tb.fatal, want)
	}
	if len(tb.cleanups) != 0 {
		t.Fatalf("Start registered a cleanup for a stub that never started")
	}
}

func TestNewTLSServerUsesTheSeamsOfThePackage(t *testing.T) {
	l := newMemListener()
	useMemListener(t, l)
	srv, err := NewTLSServer(http.NotFoundHandler())
	if err != nil {
		t.Fatalf("NewTLSServer: %v", err)
	}
	defer srv.Close()
	if got, want := srv.URL, "https://127.0.0.1:4443"; got != want {
		t.Fatalf("server URL = %q, want %q", got, want)
	}
	if got, want := leafCertificate(t, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw})).NotAfter, testInstant.Add(365*24*time.Hour); !got.Equal(want) {
		t.Fatalf("NewTLSServer dated its certificate to %v, want %v from the package clock", got, want)
	}
}

func TestNewTLSServerWithReportsAListenerFailureWithoutServing(t *testing.T) {
	errListen := errors.New("listener refused")
	srv, err := NewTLSServerWith(http.NotFoundHandler(),
		func(string, string) (net.Listener, error) { return nil, errListen }, fixedClock)
	if !errors.Is(err, errListen) {
		t.Fatalf("NewTLSServerWith returned %v, want the listener error", err)
	}
	if srv != nil {
		t.Fatal("NewTLSServerWith returned a server after its listener failed")
	}
}

func TestNewTLSServerOpensNoListenerWhenTheCertificateCannotBeMade(t *testing.T) {
	cases := []struct {
		name    string
		now     time.Time
		random  io.Reader
		wantErr error
	}{
		{name: "the entropy source runs dry before the serial number", now: testInstant,
			random: &budgetReader{}, wantErr: errEntropy},
		{name: "the clock is past the year 9999, which a certificate cannot encode", now: time.Date(9999, time.December, 31, 12, 0, 0, 0, time.UTC),
			random: rand.Reader},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			listened := false
			listen := func(string, string) (net.Listener, error) {
				listened = true
				return newMemListener(), nil
			}

			srv, err := newTLSServer(http.NotFoundHandler(), listen, func() time.Time { return tc.now }, tc.random)

			if err == nil {
				t.Fatal("newTLSServer made a server with no certificate")
			}
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Fatalf("newTLSServer returned %v, want %v", err, tc.wantErr)
			}
			if srv != nil {
				t.Fatal("newTLSServer returned a server with the error")
			}
			if listened {
				t.Fatal("newTLSServer opened a listener before the certificate was made")
			}
		})
	}
}

func TestNewTLSServerWithListensOnLoopbackPortZeroOnly(t *testing.T) {
	var gotNetwork, gotAddress string
	l := newMemListener()
	srv, err := NewTLSServerWith(http.NotFoundHandler(), func(network, address string) (net.Listener, error) {
		gotNetwork, gotAddress = network, address
		return l, nil
	}, fixedClock)
	if err != nil {
		t.Fatalf("NewTLSServerWith: %v", err)
	}
	defer srv.Close()
	if gotNetwork != "tcp" || gotAddress != "127.0.0.1:0" {
		t.Fatalf("listened on %s %s, want tcp 127.0.0.1:0", gotNetwork, gotAddress)
	}
}

// budgetReader is an entropy source that fails once its budget of bytes is spent.
type budgetReader struct {
	left int
}

var errEntropy = errors.New("entropy source failed")

func (r *budgetReader) Read(p []byte) (int, error) {
	if r.left <= 0 {
		return 0, errEntropy
	}
	n := min(len(p), r.left)
	for i := range n {
		p[i] = byte(i + 1)
	}
	r.left -= n
	return n, nil
}

func TestMakeStubCertificateReportsEveryFailureOfItsEntropySource(t *testing.T) {
	// The key, the serial number, and the signature each draw on the source. Every budget that stops one of them must
	// come back as the source's own error, and the first budget that lets all three finish must yield a certificate.
	failures := 0
	for budget := 0; ; budget++ {
		cert, err := makeStubCertificate(testInstant, &budgetReader{left: budget})
		if err == nil {
			if len(cert.Certificate) != 1 || cert.PrivateKey == nil {
				t.Fatalf("budget %d made a certificate with no leaf or no key", budget)
			}
			if failures == 0 {
				t.Fatal("no budget failed, so the entropy source is never read")
			}
			return
		}
		if !errors.Is(err, errEntropy) {
			t.Fatalf("budget %d returned %v, want the entropy source's error", budget, err)
		}
		failures++
		if budget > 4096 {
			t.Fatal("no budget let the certificate finish")
		}
	}
}

// TestStartWithAListenerAndACertificateClockDatesTheCertificateFromThatClock checks the two options of Start that a unit
// test uses: the listener it opens and the clock that dates its certificate. No socket opens.
func TestStartWithAListenerAndACertificateClockDatesTheCertificateFromThatClock(t *testing.T) {
	listen := func(string, string) (net.Listener, error) { return newMemListener(), nil }
	stub := Start(t, WithListener(listen), WithCertificateClock(fixedClock))
	data, err := stub.CAPEM()
	if err != nil {
		t.Fatalf("CAPEM: %v", err)
	}
	if got, want := leafCertificate(t, data).NotBefore, testInstant.Add(-time.Hour); !got.Equal(want) {
		t.Fatalf("NotBefore = %v, want %v from the certificate clock", got, want)
	}
}

// TestMakeStubCertificateReportsAKeyThatCannotBeMade checks that a failed key step is reported as its own error.
func TestMakeStubCertificateReportsAKeyThatCannotBeMade(t *testing.T) {
	saved := generateKey
	errKey := errors.New("no key")
	generateKey = func(io.Reader) (*ecdsa.PrivateKey, error) { return nil, errKey }
	t.Cleanup(func() { generateKey = saved })
	if _, err := makeStubCertificate(testInstant, rand.Reader); !errors.Is(err, errKey) {
		t.Fatalf("makeStubCertificate error = %v, want the key error", err)
	}
}
