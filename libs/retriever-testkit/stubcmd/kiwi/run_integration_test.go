//go:build integration

package kiwi

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-testkit/kiwistub"
)

// realHost returns the Real host with the key variable set to fixtureKey and the stop signal already done, so a run
// returns after it prints. Everything else is real: the files, the clock, the loopback socket, and the entropy.
func realHost() Host {
	h := Real
	h.Getenv = func(name string) string {
		if name == keyVariable {
			return fixtureKey
		}
		return ""
	}
	h.Notify = stopAtOnce
	return h
}

// TestRunPrintsTheListeningLineAndWritesTheCA checks the real run: the CA file exists, and the one line names the URL.
func TestRunPrintsTheListeningLineAndWritesTheCA(t *testing.T) {
	caFile := filepath.Join(t.TempDir(), "ca.pem")
	var out bytes.Buffer
	if status := Run([]string{"-ca-file", caFile}, &out, io.Discard, realHost()); status != 0 {
		t.Fatalf("run = %d, want 0", status)
	}
	if !strings.HasPrefix(out.String(), "kiwi stub listening on https://127.0.0.1:") {
		t.Fatalf("output = %q, want the listening line", out.String())
	}
	if data, err := os.ReadFile(caFile); err != nil || !strings.Contains(string(data), "BEGIN CERTIFICATE") {
		t.Fatalf("CA file = %v, want a PEM certificate", err)
	}
}

// TestStartServesTheDatasetFileOverTLS checks the real path end to end: the dataset file is read from disk, the stub
// listens on loopback, and a TLS client that trusts the CA file gets the dataset.
func TestStartServesTheDatasetFileOverTLS(t *testing.T) {
	dir := t.TempDir()
	datasetFile := filepath.Join(dir, "dataset.json")
	if err := os.WriteFile(datasetFile, []byte(fixtureDataset), 0o600); err != nil {
		t.Fatal(err)
	}
	h := realHost()
	dataset, err := readDataset(h, datasetFile)
	if err != nil {
		t.Fatalf("readDataset: %v", err)
	}
	stub, err := start(config{caFile: filepath.Join(dir, "ca.pem"), key: fixtureKey, dataset: dataset}, h)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	defer stub.Close()

	client, err := stub.Client()
	if err != nil {
		t.Fatalf("Client: %v", err)
	}
	body := `{"jsonrpc":"2.0","method":"Product.filter","params":{"query":{}},"id":1}`
	req, _ := http.NewRequest(http.MethodPost, stub.URL()+kiwistub.Endpoint, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+fixtureKey)
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
