//go:build integration

package kiwistub

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestStubServesOverTLSAndWritesItsCA checks the started server: a TLS client that trusts the CA file gets the reply. It
// opens a real loopback listener and writes a file, so it is an integration test.
func TestStubServesOverTLSAndWritesItsCA(t *testing.T) {
	s := New(testKey, WithDataset(fixture()))
	if err := s.WriteCAFile(filepath.Join(t.TempDir(), "ca.pem")); err == nil {
		t.Fatalf("WriteCAFile before Start returned no error")
	}
	if err := s.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer s.Close()
	if !strings.HasPrefix(s.URL(), "https://127.0.0.1:") {
		t.Fatalf("URL = %q, want a loopback HTTPS URL", s.URL())
	}
	caFile := filepath.Join(t.TempDir(), "ca.pem")
	if err := s.WriteCAFile(caFile); err != nil {
		t.Fatalf("WriteCAFile: %v", err)
	}
	if _, err := os.Stat(caFile); err != nil {
		t.Fatalf("CA file missing: %v", err)
	}
	client, err := s.Client()
	if err != nil {
		t.Fatalf("Client: %v", err)
	}
	body := `{"jsonrpc":"2.0","method":"Product.filter","params":{"query":{}},"id":7}`
	req, _ := http.NewRequest(http.MethodPost, s.URL()+Endpoint, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+testKey)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("TLS POST: %v", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(raw), "Example Portal") {
		t.Fatalf("TLS reply = %d %s", resp.StatusCode, raw)
	}
}
