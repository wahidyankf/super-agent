//go:build integration

package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/tests/specstub"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
)

// exampleSpecPath is the bundled example spec, which the run loads from a connectors folder of the test.
const exampleSpecPath = "../../examples/connectors/example-wiki.yml"

// loopbackNetwork returns the network of the test: the stub host resolves to the loopback address, and a dial to the checked
// loopback address on the standard port reaches the stub server. The engine still checks each address before it dials.
func loopbackNetwork(s *specstub.Server) engine.Network {
	var dialer net.Dialer
	return engine.Network{
		Lookup: func(_ context.Context, host string) ([]netip.Addr, error) {
			if host != specstub.Host {
				return nil, errors.New("the test resolver knows only the stub host")
			}
			return []netip.Addr{netip.MustParseAddr("127.0.0.1")}, nil
		},
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			if address != "127.0.0.1:443" {
				return nil, errors.New("the test dialer reaches only the checked loopback address")
			}
			return dialer.DialContext(ctx, network, s.Addr())
		},
	}
}

// specRunEnv is one run of the composition root against the loopback stub: a catalog and a connectors folder in a temporary
// folder, the real host with the loopback network, and the output buffers.
type specRunEnv struct {
	server *specstub.Server
	argv   []string
	host   host
	stdout *bytes.Buffer
	stderr *bytes.Buffer
}

// newSpecRun prepares a run of the example spec. The catalog's pin is pin, or no pin when pin is empty, and the key of the
// example source is set in the process environment for the run.
func newSpecRun(t *testing.T, pin string, extra ...string) specRunEnv {
	t.Helper()
	raw, err := os.ReadFile(exampleSpecPath)
	if err != nil {
		t.Fatalf("read the example spec: %v", err)
	}
	model := mustSpec(t, string(raw))
	server := specstub.Start(t, model)

	dir := t.TempDir()
	connectors := filepath.Join(dir, "connectors")
	if err := os.Mkdir(connectors, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(connectors, "example-wiki.yml"), raw, 0o600); err != nil {
		t.Fatalf("write the spec: %v", err)
	}
	catalogPath := filepath.Join(dir, "sources.yml")
	catalogText := "version: 1\nsources:\n  - id: wiki\n    kind: example-wiki\n    endpoint: https://" + specstub.Host +
		"\n    credential_env: EXAMPLE_WIKI_API_KEY\n"
	if pin != "" {
		catalogText += "    spec_sha256: " + pin + "\n"
	}
	if err := os.WriteFile(catalogPath, []byte(catalogText), 0o600); err != nil {
		t.Fatalf("write the catalog: %v", err)
	}
	t.Setenv("EXAMPLE_WIKI_API_KEY", "example-key")

	h := realHost
	h.engine = engineSeam{clock: realClock{}, network: loopbackNetwork(server), roots: server.Roots, options: []engine.Option{engine.AllowPrivateNetwork()}}
	env := specRunEnv{server: server, host: h}
	env.stdout, env.stderr = &bytes.Buffer{}, &bytes.Buffer{}
	env.host.stdout, env.host.stderr = env.stdout, env.stderr
	env.argv = append([]string{"--catalog", catalogPath, "--env-file", filepath.Join(dir, "keys.env"),
		"--connectors-dir", connectors}, extra...)
	return env
}

// TestTheRealNetworkHasBothCalls checks that the production network holds its two calls, so the engine can reach a source.
func TestTheRealNetworkHasBothCalls(t *testing.T) {
	n := realNetwork()
	if n.Lookup == nil || n.Dial == nil {
		t.Fatalf("realNetwork() has a nil call")
	}
}

// TestAExampleSpecCommandRunsAgainstTheLoopbackStubAndExitsZero is scenario (a) of the connector specs: a catalog source
// of the example kind, and a command of the spec. The request goes over loopback HTTPS through the real dialer, the stub
// answers from the spec's own example, and the command prints the mapped document with exit 0.
func TestAExampleSpecCommandRunsAgainstTheLoopbackStubAndExitsZero(t *testing.T) {
	env := newSpecRun(t, "", "--output", "json", "example-wiki", "docs", "list")
	status := run(env.argv, func(string) string { return "" }, env.host)
	if status != 0 || !strings.Contains(env.stdout.String(), "doc-1") {
		t.Fatalf("status = %d, stdout = %q, stderr = %q; want 0 and doc-1", status, env.stdout.String(), env.stderr.String())
	}
	if env.server.Hits() != 1 {
		t.Fatalf("the stub received %d requests, want 1", env.server.Hits())
	}
}

// TestC2AChangedSpecFailsTheRunAndNoRequestGoesOut is the C2 control at the composition root: a catalog pin that does not
// match the spec file fails the run with the pin code, exits 2, and sends no request to the stub.
func TestC2AChangedSpecFailsTheRunAndNoRequestGoesOut(t *testing.T) {
	wrong := strings.Repeat("0", 64)
	env := newSpecRun(t, wrong, "--output", "json", "example-wiki", "docs", "list")
	status := run(env.argv, func(string) string { return "" }, env.host)
	body := env.stdout.String() + env.stderr.String()
	if status != 2 || !strings.Contains(body, "retriever.connector.spec_pin_mismatch") {
		t.Fatalf("status = %d, body = %q; want 2 and the pin code", status, body)
	}
	if env.server.Hits() != 0 {
		t.Fatalf("the stub received %d requests after a pin mismatch, want 0", env.server.Hits())
	}
}

// TestAPinThatMatchesTheSpecFileRunsTheCommand checks the other side of the pin: the digest of the bytes on disk matches, so
// the command runs and the stub answers once.
func TestAPinThatMatchesTheSpecFileRunsTheCommand(t *testing.T) {
	raw, err := os.ReadFile(exampleSpecPath)
	if err != nil {
		t.Fatalf("read the example spec: %v", err)
	}
	sum := sha256.Sum256(raw)
	env := newSpecRun(t, hex.EncodeToString(sum[:]), "--output", "json", "example-wiki", "docs", "list")
	status := run(env.argv, func(string) string { return "" }, env.host)
	if status != 0 || env.server.Hits() != 1 {
		t.Fatalf("status = %d, hits = %d, stderr = %q; want 0 and one request", status, env.server.Hits(), env.stderr.String())
	}
}

// TestAnExplicitMissingConnectorsFolderExitsTwoOnTheRealDisk checks the explicit folder on the real disk: a folder that
// --connectors-dir names and that does not exist exits 2 with the folder code, and the command runs no request.
func TestAnExplicitMissingConnectorsFolderExitsTwoOnTheRealDisk(t *testing.T) {
	h := realHost
	var stdout, stderr bytes.Buffer
	h.stdout, h.stderr = &stdout, &stderr
	missing := filepath.Join(t.TempDir(), "absent")
	status := run([]string{"--output", "json", "--catalog", filepath.Join(t.TempDir(), "sources.yml"),
		"--env-file", filepath.Join(t.TempDir(), "keys.env"), "--connectors-dir", missing, "sources", "list"},
		func(string) string { return "" }, h)
	body := stdout.String() + stderr.String()
	if status != 2 || !strings.Contains(body, "retriever.connector.folder_unavailable") || strings.Contains(body, missing) {
		t.Fatalf("status = %d, body = %q; want 2, the folder code, and no path", status, body)
	}
}

// TestADefaultConnectorsFolderThatIsAbsentIsNoError checks the default folder on the real disk: no user configuration
// folder can be named, so no default folder exists, and the sources command runs as it did before connector specs.
func TestADefaultConnectorsFolderThatIsAbsentIsNoError(t *testing.T) {
	h := realHost
	var stdout, stderr bytes.Buffer
	h.stdout, h.stderr = &stdout, &stderr
	catalogPath := filepath.Join(t.TempDir(), "sources.yml")
	catalogText := "version: 1\nsources:\n  - id: docs\n    kind: outline\n    endpoint: https://docs.example.test\n    credential_env: EXAMPLE_OUTLINE_API_KEY\n"
	if err := os.WriteFile(catalogPath, []byte(catalogText), 0o600); err != nil {
		t.Fatalf("write the catalog: %v", err)
	}
	status := run([]string{"--catalog", catalogPath, "--env-file", filepath.Join(t.TempDir(), "keys.env"), "sources", "list"},
		func(string) string { return "" }, h)
	if status != 0 {
		t.Fatalf("status = %d, stderr = %q; want 0", status, stderr.String())
	}
}
