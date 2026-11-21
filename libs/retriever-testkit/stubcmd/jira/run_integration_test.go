//go:build integration

package jira

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// realStopped returns the Real host with its stop signal already done, so a run returns after it prints. Everything else
// is real: the files, the clock, and the loopback TLS server.
func realStopped() Host {
	h := Real
	h.Notify = stopAtOnce
	return h
}

// trustingClient returns an HTTP client that trusts only the certificate with the given DER bytes.
func trustingClient(t *testing.T, der []byte) *http.Client {
	t.Helper()
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse the server certificate: %v", err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(cert)
	return &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}}}
}

// TestRealRunPrintsTheListeningLineAndWritesTheCertificate starts the stub on a loopback port with the real host and a
// done stop signal, so run returns once it has printed its address and written the CA file.
func TestRealRunPrintsTheListeningLineAndWritesTheCertificate(t *testing.T) {
	ca := filepath.Join(t.TempDir(), "ca.pem")
	var out, errOut bytes.Buffer
	args := []string{"-ca-file", ca, "-email", "fixture-reviewer@example.com", "-token", "sentinel-token-5d2b8e-do-not-print"}
	if status := Run(args, &out, &errOut, realStopped()); status != 0 {
		t.Fatalf("run = %d, stderr %q", status, errOut.String())
	}
	if !strings.HasPrefix(out.String(), "jira stub listening on https://127.0.0.1:") {
		t.Fatalf("listening line = %q", out.String())
	}
	data, err := os.ReadFile(ca)
	if err != nil {
		t.Fatalf("read CA file: %v", err)
	}
	if !strings.HasPrefix(string(data), "-----BEGIN CERTIFICATE-----") {
		t.Fatalf("CA file is not a PEM certificate")
	}
	if strings.Contains(out.String(), "sentinel-token") || strings.Contains(errOut.String(), "sentinel-token") {
		t.Fatal("the command output shows the token")
	}
}

// TestRealRecordFileHoldsEachRequestAndNeverTheToken serves one signed request over real TLS with a real record file, then reads
// the file after the stub stops.
func TestRealRecordFileHoldsEachRequestAndNeverTheToken(t *testing.T) {
	dir := t.TempDir()
	record := filepath.Join(dir, "requests.jsonl")
	const token = "sentinel-token-5d2b8e-do-not-print"
	r, err := startStub(Real, config{caFile: filepath.Join(dir, "ca.pem"), email: "a@example.com", token: token, recordFile: record})
	if err != nil {
		t.Fatalf("startStub: %v", err)
	}
	req, err := http.NewRequest(http.MethodGet, r.server.URL+"/rest/api/3/myself", nil)
	if err != nil {
		r.stop()
		t.Fatalf("NewRequest: %v", err)
	}
	req.SetBasicAuth("a@example.com", token)
	resp, err := trustingClient(t, r.server.DER).Do(req)
	if err != nil {
		r.stop()
		t.Fatalf("GET over TLS: %v", err)
	}
	_ = resp.Body.Close()
	r.stop()

	data, err := os.ReadFile(record)
	if err != nil {
		t.Fatalf("read the record file: %v", err)
	}
	if strings.Count(string(data), "\n") != 1 || !strings.Contains(string(data), `"Path":"/rest/api/3/myself"`) {
		t.Fatalf("record = %q, want one line for the request", data)
	}
	if strings.Contains(string(data), token) {
		t.Fatal("the record file holds the token")
	}
}

// TestStartServesAFaultAndWritesTheCA checks that -fault-status with -fault-path starts a real stub and writes its CA file.
func TestStartServesAFaultAndWritesTheCA(t *testing.T) {
	ca := filepath.Join(t.TempDir(), "ca.pem")
	r, err := startStub(Real, config{caFile: ca, email: "a@example.com", token: "t", faultStatus: 429, faultPath: "/rest/api/3/issue/", retryAfter: "3"})
	if err != nil {
		t.Fatalf("startStub: %v", err)
	}
	defer r.stop()
	if _, err := os.Stat(ca); err != nil {
		t.Fatalf("CA file missing: %v", err)
	}
}
