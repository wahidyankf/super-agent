package kiwistub

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

var (
	errEntropy = errors.New("test: entropy unavailable")
	errSign    = errors.New("test: certificate not signed")
	errDisk    = errors.New("test: disk full")
	errPort    = errors.New("test: port unavailable")
)

// parseCAPEM returns the certificate of a PEM file that holds one certificate.
func parseCAPEM(t *testing.T, data []byte) *x509.Certificate {
	t.Helper()
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "CERTIFICATE" {
		t.Fatalf("CA data is not one PEM certificate: %q", data)
	}
	leaf, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("CA certificate does not parse: %v", err)
	}
	return leaf
}

// verifyHost checks that leaf is trusted by roots for host, at the fixed clock, for server authentication.
func verifyHost(leaf *x509.Certificate, roots *x509.CertPool, host string) error {
	_, err := leaf.Verify(x509.VerifyOptions{
		DNSName:     host,
		Roots:       roots,
		CurrentTime: fixedNow,
		KeyUsages:   []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	})
	return err
}

// TestStartListensOnTheLoopbackAndCertifiesTheCatalog checks the started stub: it listens on a free loopback port, its
// certificate is valid for the catalog hosts and the loopback address on the stub's clock, and it is its own CA.
func TestStartListensOnTheLoopbackAndCertifiesTheCatalog(t *testing.T) {
	var network, address string
	ln := newFakeListener(4321)
	s := New(testKey, WithClock(fixedClock), WithDataset(fixture()), WithListener(func(n, a string) (net.Listener, error) {
		network, address = n, a
		return ln, nil
	}))
	if err := s.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	<-ln.started

	if network != "tcp" || address != "127.0.0.1:0" {
		t.Fatalf("listen(%q, %q), want tcp on the loopback address with a free port", network, address)
	}
	if s.URL() != "https://127.0.0.1:4321" {
		t.Fatalf("URL = %q, want the loopback URL of the listener", s.URL())
	}

	leaf := parseCAPEM(t, s.CAPEM())
	roots := x509.NewCertPool()
	roots.AddCert(leaf)
	for _, host := range append(append([]string{}, CatalogHosts...), "127.0.0.1") {
		if err := verifyHost(leaf, roots, host); err != nil {
			t.Errorf("certificate is not valid for %s: %v", host, err)
		}
	}
	if err := verifyHost(leaf, roots, "other.example.invalid"); err == nil {
		t.Errorf("certificate is valid for a host it does not name")
	}
	if !leaf.NotBefore.Equal(fixedNow.Add(-time.Hour)) || !leaf.NotAfter.Equal(fixedNow.Add(365*24*time.Hour)) {
		t.Errorf("validity = %v to %v, want an hour before and a year after the stub's clock", leaf.NotBefore, leaf.NotAfter)
	}
	if !leaf.IsCA {
		t.Errorf("leaf is not a CA, so it cannot be its own trust root")
	}

	s.Close()
	if !ln.isClosed() {
		t.Fatalf("Close did not close the listener")
	}
}

// TestStartRefusesWhenAStepFails checks that each failing step returns its error, and that nothing is left started.
func TestStartRefusesWhenAStepFails(t *testing.T) {
	cases := []struct {
		name     string
		setup    func(s *Stub)
		wantErr  error
		listened bool
	}{
		{"the TLS key cannot be generated", func(s *Stub) {
			s.sys.genKey = func(elliptic.Curve, io.Reader) (*ecdsa.PrivateKey, error) { return nil, errEntropy }
		}, errEntropy, false},
		{"the certificate is not signed", func(s *Stub) {
			s.sys.sign = func(io.Reader, *x509.Certificate, *x509.Certificate, any, any) ([]byte, error) {
				return nil, errSign
			}
		}, errSign, false},
		{"the signed certificate does not parse", func(s *Stub) {
			s.sys.sign = func(io.Reader, *x509.Certificate, *x509.Certificate, any, any) ([]byte, error) {
				return []byte("not DER"), nil
			}
		}, nil, false},
		{"the port cannot be opened", func(s *Stub) {}, errPort, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newStub(WithDataset(fixture()))
			listens := 0
			s.sys.listen = func(string, string) (net.Listener, error) {
				listens++
				return nil, errPort
			}
			tc.setup(s)
			err := s.Start()
			if err == nil {
				t.Fatalf("Start returned no error")
			}
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Fatalf("Start error = %v, want %v", err, tc.wantErr)
			}
			wantListens := 0
			if tc.listened {
				wantListens = 1
			}
			if listens != wantListens {
				t.Fatalf("listen was called %d times, want %d", listens, wantListens)
			}
			if s.URL() != "" || s.CAPEM() != nil {
				t.Fatalf("a stub that failed to start has a URL %q or a certificate", s.URL())
			}
		})
	}
}

// TestClientTrustsOnlyItsOwnStub checks the client of a started stub: it uses no proxy, requires TLS 1.2, and trusts the
// stub's certificate and no other.
func TestClientTrustsOnlyItsOwnStub(t *testing.T) {
	s := newStub(WithDataset(fixture()))
	if _, err := s.Client(); !errors.Is(err, errNotStarted) {
		t.Fatalf("Client before Start = %v, want errNotStarted", err)
	}
	ownListener := startFake(t, s)
	defer s.Close()

	client, err := s.Client()
	if err != nil {
		t.Fatalf("Client: %v", err)
	}
	transport, ok := client.Transport.(*http.Transport)
	if !ok || transport.Proxy != nil || transport.TLSClientConfig.MinVersion != tls.VersionTLS12 {
		t.Fatalf("client transport = %#v, want no proxy and TLS 1.2 or later", client.Transport)
	}
	if err := verifyHost(parseCAPEM(t, s.CAPEM()), transport.TLSClientConfig.RootCAs, "kiwi.example.test"); err != nil {
		t.Fatalf("client does not trust its own stub: %v", err)
	}

	other := newStub(WithDataset(fixture()))
	otherListener := startFake(t, other)
	defer other.Close()
	if err := verifyHost(parseCAPEM(t, other.CAPEM()), transport.TLSClientConfig.RootCAs, "kiwi.example.test"); err == nil {
		t.Fatalf("client trusts the certificate of another stub")
	}
	if ownListener.isClosed() || otherListener.isClosed() {
		t.Fatalf("a listener closed before its stub did")
	}
}

// TestWriteCAFileHandsTheCertificateToTheWriter checks the CA file: it is refused before Start, it holds the PEM of the
// stub with owner-only permissions, and a write failure is returned.
func TestWriteCAFileHandsTheCertificateToTheWriter(t *testing.T) {
	t.Run("before Start there is nothing to write", func(t *testing.T) {
		s := newStub()
		writes := 0
		s.sys.writeFile = func(string, []byte, fs.FileMode) error {
			writes++
			return nil
		}
		if err := s.WriteCAFile("ca.pem"); !errors.Is(err, errNotStarted) {
			t.Fatalf("WriteCAFile before Start = %v, want errNotStarted", err)
		}
		if writes != 0 {
			t.Fatalf("the writer was called %d times before Start", writes)
		}
	})
	t.Run("after Start the PEM is written owner-only", func(t *testing.T) {
		s := newStub()
		startFake(t, s)
		defer s.Close()
		var gotName string
		var gotData []byte
		var gotPerm fs.FileMode
		s.sys.writeFile = func(name string, data []byte, perm fs.FileMode) error {
			gotName, gotData, gotPerm = name, data, perm
			return nil
		}
		if err := s.WriteCAFile("ca.pem"); err != nil {
			t.Fatalf("WriteCAFile: %v", err)
		}
		if gotName != "ca.pem" || string(gotData) != string(s.CAPEM()) || gotPerm != 0o600 {
			t.Fatalf("write(%q, %d bytes, %o), want the CA PEM at mode 600", gotName, len(gotData), gotPerm)
		}
	})
	t.Run("a failed write is returned", func(t *testing.T) {
		s := newStub()
		startFake(t, s)
		defer s.Close()
		s.sys.writeFile = func(string, []byte, fs.FileMode) error { return errDisk }
		if err := s.WriteCAFile("ca.pem"); !errors.Is(err, errDisk) {
			t.Fatalf("WriteCAFile = %v, want the write error", err)
		}
	})
}

// TestRoundTripDropsAndHoldsLikeANetwork checks the two transport failures of the in-process client: a dropped
// connection is a transport error, and a request that is never answered ends with its context.
func TestRoundTripDropsAndHoldsLikeANetwork(t *testing.T) {
	cases := []struct {
		name    string
		fault   Fault
		ctx     func() (context.Context, context.CancelFunc)
		wantErr error
	}{
		{"a dropped connection", Fault{Drop: true}, func() (context.Context, context.CancelFunc) {
			return context.WithCancel(context.Background())
		}, errDropped},
		{"a request never answered", Fault{Never: true}, func() (context.Context, context.CancelFunc) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			return ctx, cancel
		}, context.Canceled},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newStub(WithDataset(fixture()))
			s.SetFault(tc.fault)
			ctx, cancel := tc.ctx()
			defer cancel()
			req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://kiwi.example.test"+Endpoint, strings.NewReader("{}"))
			req.Header.Set("Authorization", "Bearer "+testKey)
			if _, err := s.RoundTripper().RoundTrip(req); !errors.Is(err, tc.wantErr) {
				t.Fatalf("RoundTrip error = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

// TestRoundTripAnswersARequestWithoutABody checks a request that carries no body: the stub answers it as a verb it does
// not serve, so the reply is HTTP 405.
func TestRoundTripAnswersARequestWithoutABody(t *testing.T) {
	s := newStub(WithDataset(fixture()))
	req, _ := http.NewRequest(http.MethodGet, "https://kiwi.example.test"+Endpoint, nil)
	resp, err := s.RoundTripper().RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusMethodNotAllowed || !strings.Contains(string(raw), "method not allowed") {
		t.Fatalf("reply = %d %s, want 405 with the method-not-allowed body", resp.StatusCode, raw)
	}
}

// TestServeHTTPWritesTheReplyToTheHandler checks the handler of the TLS server: a reply is written with its headers, the
// record says whether the request came over TLS, and a redirect keeps its location.
func TestServeHTTPWritesTheReplyToTheHandler(t *testing.T) {
	body := `{"jsonrpc":"2.0","method":"Product.filter","params":{"query":{}},"id":7}`
	t.Run("a TLS request is answered and recorded as secure", func(t *testing.T) {
		s := newStub(WithDataset(fixture()))
		req := httptest.NewRequest(http.MethodPost, "https://kiwi.example.test"+Endpoint, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+testKey)
		rec := httptest.NewRecorder()
		s.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "application/json" {
			t.Fatalf("reply = %d %q, want 200 JSON", rec.Code, rec.Header().Get("Content-Type"))
		}
		if !strings.Contains(rec.Body.String(), "Example Portal") {
			t.Fatalf("reply body = %s, want the dataset", rec.Body.String())
		}
		got := s.Requests()[0]
		if !got.Secure || got.Host != "kiwi.example.test" || got.Status != http.StatusOK {
			t.Fatalf("recorded = %+v, want a secure request answered 200", got)
		}
	})
	t.Run("a plain request is recorded as not secure", func(t *testing.T) {
		s := newStub(WithDataset(fixture()))
		req := httptest.NewRequest(http.MethodPost, "http://kiwi.example.test"+Endpoint, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+testKey)
		s.ServeHTTP(httptest.NewRecorder(), req)
		if s.Requests()[0].Secure {
			t.Fatalf("a plain request was recorded as secure")
		}
	})
	t.Run("a redirect keeps its location header", func(t *testing.T) {
		s := newStub(WithDataset(fixture()))
		s.SetFault(Fault{Redirect: true, Location: "https://example.invalid/login"})
		req := httptest.NewRequest(http.MethodPost, "https://kiwi.example.test"+Endpoint, strings.NewReader(body))
		rec := httptest.NewRecorder()
		s.ServeHTTP(rec, req)
		if rec.Code != http.StatusFound || rec.Header().Get("Location") != "https://example.invalid/login" {
			t.Fatalf("reply = %d Location %q, want 302 to the named location", rec.Code, rec.Header().Get("Location"))
		}
	})
}

// TestServeHTTPHoldsAndDropsLikeANetwork checks the two faults of the handler: a held request ends with its context, and a
// dropped one closes its connection when the writer can be hijacked.
func TestServeHTTPHoldsAndDropsLikeANetwork(t *testing.T) {
	body := `{"jsonrpc":"2.0","method":"Product.filter","params":{"query":{}},"id":7}`
	newRequest := func() *http.Request {
		return httptest.NewRequest(http.MethodPost, "https://kiwi.example.test"+Endpoint, strings.NewReader(body))
	}
	t.Run("a held request returns when its context ends", func(t *testing.T) {
		s := newStub(WithDataset(fixture()))
		s.SetFault(Fault{Never: true})
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		rec := httptest.NewRecorder()
		s.ServeHTTP(rec, newRequest().WithContext(ctx))
		if rec.Body.Len() != 0 {
			t.Fatalf("a held request wrote %q", rec.Body.String())
		}
	})
	t.Run("a dropped request closes a hijackable connection", func(t *testing.T) {
		s := newStub(WithDataset(fixture()))
		s.SetFault(Fault{Drop: true})
		conn := &fakeConn{}
		w := &hijackRecorder{ResponseRecorder: httptest.NewRecorder(), conn: conn}
		s.ServeHTTP(w, newRequest())
		if !conn.closed || w.Body.Len() != 0 {
			t.Fatalf("closed = %v, body %q: want the connection closed and no reply", conn.closed, w.Body.String())
		}
	})
	t.Run("a dropped request whose hijack fails writes nothing", func(t *testing.T) {
		s := newStub(WithDataset(fixture()))
		s.SetFault(Fault{Drop: true})
		conn := &fakeConn{}
		w := &hijackRecorder{ResponseRecorder: httptest.NewRecorder(), conn: conn, err: errDisk}
		s.ServeHTTP(w, newRequest())
		if conn.closed || w.Body.Len() != 0 {
			t.Fatalf("closed = %v, body %q: want no close and no reply", conn.closed, w.Body.String())
		}
	})
	t.Run("a dropped request on a writer that cannot be hijacked writes nothing", func(t *testing.T) {
		s := newStub(WithDataset(fixture()))
		s.SetFault(Fault{Drop: true})
		rec := httptest.NewRecorder()
		s.ServeHTTP(rec, newRequest())
		if rec.Body.Len() != 0 {
			t.Fatalf("a dropped request wrote %q", rec.Body.String())
		}
	})
}
