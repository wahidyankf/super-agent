package jirastub

import (
	"crypto/ecdsa"
	"crypto/x509"
	"errors"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

// unitCertificateInstant is the fixed clock of the certificate tests of this file.
var unitCertificateInstant = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

// TestNewTLSServerWithReportsAListenerThatCannotOpen checks that a listener that fails to open is reported, and that no
// server is returned with it.
func TestNewTLSServerWithReportsAListenerThatCannotOpen(t *testing.T) {
	errListen := errors.New("listener refused")
	srv, err := NewTLSServerWith(http.NotFoundHandler(),
		func(string, string) (net.Listener, error) { return nil, errListen },
		func() time.Time { return unitCertificateInstant })
	if !errors.Is(err, errListen) {
		t.Fatalf("NewTLSServerWith error = %v, want the listener error", err)
	}
	if srv != nil {
		t.Fatal("NewTLSServerWith returned a server with the listener error")
	}
}

// TestNewTLSServerWithServesOnTheGivenListenerAndDatesFromTheGivenClock checks that the server runs on the listener it was
// given, and that its certificate is dated from the clock it was given. The listener opens no socket.
func TestNewTLSServerWithServesOnTheGivenListenerAndDatesFromTheGivenClock(t *testing.T) {
	l := newPipeListener()
	srv, err := NewTLSServerWith(http.NotFoundHandler(),
		func(string, string) (net.Listener, error) { return l, nil },
		func() time.Time { return unitCertificateInstant })
	if err != nil {
		t.Fatalf("NewTLSServerWith: %v", err)
	}
	defer srv.Close()
	if got, want := srv.Certificate().NotBefore, unitCertificateInstant.Add(-time.Hour); !got.Equal(want) {
		t.Fatalf("certificate NotBefore = %v, want %v from the given clock", got, want)
	}
}

// TestProduceStubCertificateDatesFromTheStubClock checks that the process certificate is dated from the stub clock.
func TestProduceStubCertificateDatesFromTheStubClock(t *testing.T) {
	saved := stubClock
	stubClock = func() time.Time { return unitCertificateInstant }
	t.Cleanup(func() { stubClock = saved })
	cert, err := produceStubCertificate()
	if err != nil {
		t.Fatalf("produceStubCertificate: %v", err)
	}
	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatalf("ParseCertificate: %v", err)
	}
	if want := unitCertificateInstant.Add(-time.Hour); !leaf.NotBefore.Equal(want) {
		t.Fatalf("NotBefore = %v, want %v from the stub clock", leaf.NotBefore, want)
	}
}

// TestNewTLSServerWithMakesTheCertificateBeforeItOpensTheListener checks that a certificate that cannot be made is reported
// and that no listener opens for it.
func TestNewTLSServerWithMakesTheCertificateBeforeItOpensTheListener(t *testing.T) {
	saved := realCertificateSteps
	realCertificateSteps.key = func(io.Reader) (*ecdsa.PrivateKey, error) { return nil, errors.New("no key") }
	t.Cleanup(func() { realCertificateSteps = saved })
	opened := false
	srv, err := NewTLSServerWith(http.NotFoundHandler(),
		func(string, string) (net.Listener, error) { opened = true; return newPipeListener(), nil },
		func() time.Time { return unitCertificateInstant })
	if err == nil || srv != nil {
		t.Fatalf("NewTLSServerWith = %v, %v; want an error and no server", srv, err)
	}
	if opened {
		t.Fatal("a listener opened for a certificate that could not be made")
	}
}
