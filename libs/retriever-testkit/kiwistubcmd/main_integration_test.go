//go:build integration

package main

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-testkit/kiwistub"
)

// realHost returns the host of a real process: the real environment, the real files, and the real loopback socket.
func realHost(stdout io.Writer) host {
	return host{
		getenv:    os.Getenv,
		now:       time.Now,
		stdout:    stdout,
		stderr:    io.Discard,
		readFile:  os.ReadFile,
		writeFile: os.WriteFile,
		listen:    net.Listen,
	}
}

// TestRunPrintsTheListeningLineAndWritesTheCA checks the real run: the CA file exists, and the one line names the URL.
// The context is cancelled first, so the run returns after it prints.
func TestRunPrintsTheListeningLineAndWritesTheCA(t *testing.T) {
	t.Setenv(keyVariable, fixtureKey)
	caFile := filepath.Join(t.TempDir(), "ca.pem")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var out bytes.Buffer
	if err := run(ctx, []string{"-ca-file", caFile}, realHost(&out)); err != nil {
		t.Fatalf("run: %v", err)
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
	h := realHost(io.Discard)
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
