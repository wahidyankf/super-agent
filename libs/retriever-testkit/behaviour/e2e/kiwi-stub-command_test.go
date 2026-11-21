//go:build e2e

package e2e_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const kiwiPrefix = "kiwi stub listening on "

// @scenario specs/libs/retriever-testkit/behaviours/kiwi-stub-command.feature: The command refuses to start without a certificate file path
func TestKiwiCommandRefusesToStartWithoutCAFile(t *testing.T) {
	bin := commandBinary(t, "kiwistubcmd")
	res := runCommand(t, environment(kiwiKey), bin)
	if res.code != 1 || !strings.Contains(res.stderr, "the -ca-file flag is required") {
		t.Fatalf("status %d and stderr %q; want 1 with the -ca-file message", res.code, res.stderr)
	}
	if strings.Contains(res.stdout, "listening on") {
		t.Fatalf("the command listened: %q", res.stdout)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/kiwi-stub-command.feature: The command refuses to start when the key variable is not set
func TestKiwiCommandRefusesToStartWhenKeyVariableIsNotSet(t *testing.T) {
	bin := commandBinary(t, "kiwistubcmd")
	ca := filepath.Join(t.TempDir(), "ca.pem")
	res := runCommand(t, environment(""), bin, "-ca-file", ca)
	if res.code != 1 || !strings.Contains(res.stderr, "KIWISTUB_API_KEY must hold the synthetic key") {
		t.Fatalf("status %d and stderr %q; want 1 with the key message", res.code, res.stderr)
	}
	if strings.Contains(res.stdout, "listening on") {
		t.Fatalf("the command listened: %q", res.stdout)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/kiwi-stub-command.feature: The command refuses a flag that it does not know and names the flag
func TestKiwiCommandRefusesUnknownFlagAndNamesIt(t *testing.T) {
	bin := commandBinary(t, "kiwistubcmd")
	res := runCommand(t, environment(kiwiKey), bin, "-no-such-flag")
	if res.code != 1 || !strings.Contains(res.stderr, "-no-such-flag") {
		t.Fatalf("status %d and stderr %q; want 1 with the flag named", res.code, res.stderr)
	}
	if strings.Contains(res.stdout, "listening on") {
		t.Fatalf("the command listened: %q", res.stdout)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/kiwi-stub-command.feature: The command writes the certificate and then prints its listening address
func TestKiwiCommandWritesCertificateAndThenPrintsAddress(t *testing.T) {
	bin := commandBinary(t, "kiwistubcmd")
	ca := filepath.Join(t.TempDir(), "ca.pem")
	r := startCommand(t, environment(kiwiKey), bin, "-ca-file", ca)
	defer r.stop(t)
	base := announcedURL(t, r.announce, kiwiPrefix)
	if !strings.HasPrefix(base, "https://127.0.0.1:") {
		t.Fatalf("the announced address %q is not a loopback TLS address", base)
	}
	data, err := os.ReadFile(ca)
	if err != nil {
		t.Fatalf("the certificate file is not written before the announcement: %v", err)
	}
	if !bytes.Contains(data, []byte("BEGIN CERTIFICATE")) {
		t.Fatal("the certificate file holds no PEM certificate")
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/kiwi-stub-command.feature: A failure after start stops the stub and prints to standard error
func TestKiwiCommandFailureAfterStartExitsOneWithErrorOnStandardError(t *testing.T) {
	bin := commandBinary(t, "kiwistubcmd")
	missing := filepath.Join(t.TempDir(), "missing", "ca.pem")
	res := runCommand(t, environment(kiwiKey), bin, "-ca-file", missing)
	if res.code != 1 || !strings.Contains(res.stderr, "no such file") {
		t.Fatalf("status %d and stderr %q; want 1 with the write error on standard error", res.code, res.stderr)
	}
	if strings.Contains(res.stdout, "listening on") {
		t.Fatalf("the command announced an address before the failure: %q", res.stdout)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/kiwi-stub-command.feature: A clean run exits with status 0
func TestKiwiCleanRunExitsWithStatusZeroOnStopSignal(t *testing.T) {
	bin := commandBinary(t, "kiwistubcmd")
	ca := filepath.Join(t.TempDir(), "ca.pem")
	r := startCommand(t, environment(kiwiKey), bin, "-ca-file", ca)
	if code := r.stop(t); code != 0 {
		t.Fatalf("the clean run exited %d, want 0", code)
	}
}
