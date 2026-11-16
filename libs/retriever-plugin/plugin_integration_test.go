//go:build integration

package plugin

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// buildHelper compiles the fixture plugin and returns its path: the host runs this copy, and the tests pin it by
// hashing these bytes.
func buildHelper(t *testing.T) string {
	t.Helper()
	out := filepath.Join(t.TempDir(), "plugintest")
	cmd := exec.Command("go", "build", "-o", out, "testdata/plugin-helper.go")
	if data, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build the fixture plugin: %v\n%s", err, data)
	}
	return out
}

// pinFile returns the lower-case hex SHA-256 of the file at path.
func pinFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read the fixture plugin: %v", err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// helperConfig returns the Config of the fixture plugin at path, pinned by its real bytes.
func helperConfig(t *testing.T, path, credential, identity string) Config {
	t.Helper()
	ref, err := domain.NewPluginRef(path, pinFile(t, path))
	if err != nil {
		t.Fatalf("NewPluginRef: %v", err)
	}
	return Config{
		Plugin:          ref,
		CredentialEnv:   "EXAMPLE_WIKI_API_KEY",
		CredentialValue: credential,
		IdentityEnv:     "EXAMPLE_WIKI_EMAIL",
		IdentityValue:   identity,
		Timeout:         10 * time.Second,
	}
}

// TestIntegrationPluginSeesOnlyItsOwnCredential runs the env-printing plugin for real: the child sees exactly its
// own credential variable and, when declared, its identity variable, and nothing of the caller (P3 and the second
// gate).
func TestIntegrationPluginSeesOnlyItsOwnCredential(t *testing.T) {
	helper := buildHelper(t)
	t.Setenv("PARENT_SECRET_API_KEY", "parent-secret-3d0f")
	cfg := helperConfig(t, helper, "credential-sentinel-9f2a", "sentinel-77@example.test")
	reply, err := Call(context.Background(), cfg, "env", nil, Seams{})
	if err != nil {
		t.Fatalf("Call error = %v, want nil", err)
	}
	var decoded struct {
		Env []string `json:"env"`
	}
	if err := json.Unmarshal(reply.Result, &decoded); err != nil {
		t.Fatalf("decode the env result: %v", err)
	}
	// The reply is redacted (P6), so the values read [REDACTED]: their presence as markers proves the child
	// saw the secrets, and the exact two names prove it saw nothing else (P3).
	want := []string{
		"EXAMPLE_WIKI_API_KEY=[REDACTED]",
		"EXAMPLE_WIKI_EMAIL=[REDACTED]",
	}
	if strings.Join(decoded.Env, "\n") != strings.Join(want, "\n") {
		t.Fatalf("plugin env = %q, want only its own credential and identity, redacted", decoded.Env)
	}
	for _, entry := range decoded.Env {
		if strings.Contains(entry, "PARENT_SECRET") || strings.Contains(entry, "parent-secret-3d0f") {
			t.Fatalf("a parent credential reached the plugin: %q", entry)
		}
	}
}

// TestIntegrationPluginOutputLeaksNoSecret runs the env-printing plugin, which echoes its environment in its reply
// and on standard error: both read [REDACTED] (P6).
func TestIntegrationPluginOutputLeaksNoSecret(t *testing.T) {
	helper := buildHelper(t)
	cfg := helperConfig(t, helper, "credential-sentinel-9f2a", "")
	reply, err := Call(context.Background(), cfg, "env", nil, Seams{})
	if err != nil {
		t.Fatalf("Call error = %v, want nil", err)
	}
	for _, text := range []string{string(reply.Result), reply.Stderr} {
		if strings.Contains(text, "credential-sentinel-9f2a") {
			t.Fatalf("a plugin output holds the secret: %q", text)
		}
		if !strings.Contains(text, "[REDACTED]") {
			t.Fatalf("a plugin output holds no redaction: %q", text)
		}
	}
}

// TestIntegrationSilentPluginIsKilled runs the never-answering plugin for real: the host kills it at the limit and
// fails with the timeout the CLI reports as 124 (P4 and the third gate).
func TestIntegrationSilentPluginIsKilled(t *testing.T) {
	helper := buildHelper(t)
	cfg := helperConfig(t, helper, "credential-sentinel-9f2a", "")
	cfg.Timeout = 300 * time.Millisecond
	start := time.Now()
	_, err := Call(context.Background(), cfg, "silent", nil, Seams{})
	elapsed := time.Since(start)
	var timeout TimeoutError
	if !errors.As(err, &timeout) {
		t.Fatalf("Call error = %#v, want TimeoutError", err)
	}
	if elapsed > 30*time.Second {
		t.Fatalf("a silenced call took %s: the plugin was not killed", elapsed)
	}
}

// TestIntegrationPinMismatchStartsNoProcess pins other bytes for the real fixture plugin: the mismatch fails
// before any process starts (P1 and the first gate).
func TestIntegrationPinMismatchStartsNoProcess(t *testing.T) {
	helper := buildHelper(t)
	ref, err := domain.NewPluginRef(helper, strings.Repeat("0", 64))
	if err != nil {
		t.Fatalf("NewPluginRef: %v", err)
	}
	cfg := Config{Plugin: ref, CredentialEnv: "EXAMPLE_WIKI_API_KEY", CredentialValue: "v", Timeout: 10 * time.Second}
	if _, err := Call(context.Background(), cfg, "env", nil, Seams{}); err == nil {
		t.Fatal("Call accepted bytes that differ from the pin")
	} else if _, ok := err.(PinMismatchError); !ok {
		t.Fatalf("Call error = %#v, want PinMismatchError", err)
	}
}

// TestIntegrationFloodedFrameIsRefused runs the flooding plugin for real: a frame past 1 MiB fails the call.
func TestIntegrationFloodedFrameIsRefused(t *testing.T) {
	helper := buildHelper(t)
	cfg := helperConfig(t, helper, "credential-sentinel-9f2a", "")
	if _, err := Call(context.Background(), cfg, "flood", nil, Seams{}); err == nil {
		t.Fatal("Call accepted a frame past 1 MiB")
	} else if protocol, ok := err.(ProtocolError); !ok || protocol.Rule != "the plugin frame is larger than 1 MiB" {
		t.Fatalf("Call error = %#v, want the frame refusal", err)
	}
}

// TestIntegrationDeadChildExitsTwo runs a plugin that exits 127 at once: the child started, so the host reports
// that it answered nothing, and the run exits 2, never 127.
func TestIntegrationDeadChildExitsTwo(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "plugin")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nexit 127\n"), 0o700); err != nil {
		t.Fatalf("write the dying plugin: %v", err)
	}
	ref, err := domain.NewPluginRef(script, pinFile(t, script))
	if err != nil {
		t.Fatalf("NewPluginRef: %v", err)
	}
	cfg := Config{Plugin: ref, CredentialEnv: "EXAMPLE_WIKI_API_KEY", CredentialValue: "v", Timeout: 10 * time.Second}
	if _, err := Call(context.Background(), cfg, "env", nil, Seams{}); err == nil {
		t.Fatal("Call accepted a plugin that died at once")
	} else if _, ok := err.(ProtocolError); !ok {
		t.Fatalf("Call error = %#v, want ProtocolError, never the child status", err)
	}
}

// TestIntegrationGatewayServesASource runs a source through the real fixture plugin: the gateway maps the served
// collections onto domain values.
func TestIntegrationGatewayServesASource(t *testing.T) {
	helper := buildHelper(t)
	ref, err := domain.NewPluginRef(helper, pinFile(t, helper))
	if err != nil {
		t.Fatalf("NewPluginRef: %v", err)
	}
	id, err := domain.NewSourceID("docs")
	if err != nil {
		t.Fatalf("NewSourceID: %v", err)
	}
	endpoint, err := domain.NewEndpoint("https://docs.example.test")
	if err != nil {
		t.Fatalf("NewEndpoint: %v", err)
	}
	credential, err := domain.NewCredentialRef("EXAMPLE_OUTLINE_API_KEY")
	if err != nil {
		t.Fatalf("NewCredentialRef: %v", err)
	}
	tags, err := domain.NewTagSet()
	if err != nil {
		t.Fatalf("NewTagSet: %v", err)
	}
	source := domain.NewSource(id, domain.KindOutline, tags, endpoint, credential).WithPlugin(ref)
	creds, err := newIntegrationCredentials("EXAMPLE_OUTLINE_API_KEY", "example-key")
	if err != nil {
		t.Fatalf("credentials: %v", err)
	}
	gw := &Gateway{Source: source, Credentials: creds, Identities: creds, Timeout: 10 * time.Second}
	collections, err := gw.ListCollections(context.Background())
	if err != nil {
		t.Fatalf("ListCollections error = %v, want nil", err)
	}
	if len(collections) != 1 || collections[0].ID != "c1" || collections[0].Name != "One" {
		t.Fatalf("collections = %+v, want the served collection", collections)
	}
}

// integrationCredentials resolves one credential from memory.
type integrationCredentials struct {
	name  string
	value string
}

func newIntegrationCredentials(name, value string) (*integrationCredentials, error) {
	if name == "" || value == "" {
		return nil, errors.New("a credential needs a name and a value")
	}
	return &integrationCredentials{name: name, value: value}, nil
}

func (c *integrationCredentials) APIKey(ref domain.CredentialRef) (domain.ApiKey, error) {
	if ref.Name() != c.name {
		return domain.ApiKey{}, domain.MissingKeyError{Variable: ref.Name()}
	}
	return domain.NewApiKey(ref.Name(), c.value)
}

func (c *integrationCredentials) Identity(ref domain.IdentityRef) (domain.Identity, error) {
	return domain.Identity{}, domain.MissingKeyError{Variable: ref.Name()}
}
