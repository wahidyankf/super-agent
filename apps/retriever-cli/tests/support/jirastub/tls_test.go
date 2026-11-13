//go:build integration

package jirastub_test

import (
	"crypto/tls"
	"crypto/x509"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/tests/support/jirastub"
)

// TestStartedCertificateCoversEveryCatalogHost checks that the TLS certificate names the catalog host and the loopback
// addresses, so a gateway with the stub's CA accepts the connection.
func TestStartedCertificateCoversEveryCatalogHost(t *testing.T) {
	srv, err := jirastub.NewTLSServer(newStub(t))
	if err != nil {
		t.Fatalf("NewTLSServer: %v", err)
	}
	defer srv.Close()
	leaf := srv.Certificate()
	for _, host := range append(append([]string(nil), jirastub.CatalogHosts...), "127.0.0.1", "::1") {
		if err := leaf.VerifyHostname(host); err != nil {
			t.Errorf("certificate does not cover %s: %v", host, err)
		}
	}
}

// TestLoopbackTLSAnswersBasicAuth proves the stub answers over TLS with the stub's CA, and checks the Basic header on the wire.
func TestLoopbackTLSAnswersBasicAuth(t *testing.T) {
	stub := newStub(t)
	srv, err := jirastub.NewTLSServer(stub)
	if err != nil {
		t.Fatalf("NewTLSServer: %v", err)
	}
	defer srv.Close()

	pool := x509.NewCertPool()
	pool.AddCert(srv.Certificate())
	client := &http.Client{
		Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}},
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/rest/api/3/myself", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.SetBasicAuth(testEmail, testToken)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "Fixture Reviewer") {
		t.Fatalf("status %d, body %s", resp.StatusCode, body)
	}
	if reqs := stub.Requests(); len(reqs) != 1 || !reqs[0].AuthOK {
		t.Fatalf("recorded requests = %+v, want one authenticated request", reqs)
	}
}
