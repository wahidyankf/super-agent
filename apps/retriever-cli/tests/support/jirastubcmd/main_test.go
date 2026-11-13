//go:build integration

package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRunPrintsTheListeningLineAndWritesTheCertificate starts the stub on a loopback port with a cancelled context, so run
// returns once it has printed its address and written the CA file.
func TestRunPrintsTheListeningLineAndWritesTheCertificate(t *testing.T) {
	ca := filepath.Join(t.TempDir(), "ca.pem")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var out bytes.Buffer
	args := []string{"-ca-file", ca, "-email", "fixture-reviewer@example.com", "-token", "sentinel-token-5d2b8e-do-not-print"}
	if err := run(ctx, args, &out); err != nil {
		t.Fatalf("run: %v", err)
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
	if strings.Contains(out.String(), "sentinel-token") {
		t.Fatal("the listening line shows the token")
	}
}

// TestStartAppliesAFaultAndWritesTheCA checks that -fault-status with -fault-path starts a stub and writes its CA file.
func TestStartAppliesAFaultAndWritesTheCA(t *testing.T) {
	ca := filepath.Join(t.TempDir(), "ca.pem")
	srv, err := start(config{caFile: ca, email: "a@example.com", token: "t", faultStatus: 429, faultPath: "/rest/api/3/issue/", retryAfter: "3"})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	defer srv.Close()
	if _, err := os.Stat(ca); err != nil {
		t.Fatalf("CA file missing: %v", err)
	}
}
