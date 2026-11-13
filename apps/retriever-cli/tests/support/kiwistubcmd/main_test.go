//go:build integration

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/tests/support/kiwistub"
)

// TestRunNeedsTheCAFile checks that a run without -ca-file stops before it listens.
func TestRunNeedsTheCAFile(t *testing.T) {
	t.Setenv(keyVariable, "fixture-key-cmd")
	if err := run(context.Background(), nil, &bytes.Buffer{}); err != errNoCAFile {
		t.Fatalf("run without -ca-file = %v, want errNoCAFile", err)
	}
}

// TestRunNeedsTheKeyVariable checks that an empty key variable stops the run, and that the error names the variable only.
func TestRunNeedsTheKeyVariable(t *testing.T) {
	t.Setenv(keyVariable, "")
	err := run(context.Background(), []string{"-ca-file", filepath.Join(t.TempDir(), "ca.pem")}, &bytes.Buffer{})
	if err != errNoKey || !strings.Contains(err.Error(), keyVariable) {
		t.Fatalf("run with an empty key = %v, want errNoKey", err)
	}
}

// TestRunPrintsTheListeningLineAndWritesTheCA checks the output contract: the CA file exists, and the one line names the
// URL. The context is cancelled first, so the run returns after it prints.
func TestRunPrintsTheListeningLineAndWritesTheCA(t *testing.T) {
	t.Setenv(keyVariable, "fixture-key-cmd")
	caFile := filepath.Join(t.TempDir(), "ca.pem")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var out bytes.Buffer
	if err := run(ctx, []string{"-ca-file", caFile}, &out); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.HasPrefix(out.String(), "kiwi stub listening on https://127.0.0.1:") {
		t.Fatalf("output = %q, want the listening line", out.String())
	}
	if data, err := os.ReadFile(caFile); err != nil || !strings.Contains(string(data), "BEGIN CERTIFICATE") {
		t.Fatalf("CA file = %v, want a PEM certificate", err)
	}
}

// TestReadDatasetParsesTheStubFields checks the dataset file: it is the JSON of kiwistub.Dataset, and a bad file is an error.
func TestReadDatasetParsesTheStubFields(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "good.json")
	data, err := json.Marshal(kiwistub.Dataset{Products: []kiwistub.Product{{ID: 1, Name: "Example Portal"}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(good, data, 0o600); err != nil {
		t.Fatal(err)
	}
	d, err := readDataset(good)
	if err != nil || len(d.Products) != 1 || d.Products[0].Name != "Example Portal" {
		t.Fatalf("readDataset = (%+v, %v)", d, err)
	}
	bad := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(bad, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readDataset(bad); err == nil {
		t.Fatalf("readDataset of a bad file returned no error")
	}
	if _, err := readDataset(filepath.Join(dir, "missing.json")); err == nil {
		t.Fatalf("readDataset of a missing file returned no error")
	}
}
