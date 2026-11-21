//go:build integration

package integration_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	plugin "github.com/wahidyankf/super-agent/libs/retriever-plugin"
)

// @scenario specs/libs/retriever-plugin/behaviours/real-process.feature: A real plugin sees exactly its own credential variable and, when declared, its identity variable
func TestARealPluginSeesExactlyItsOwnCredentialAndDeclaredIdentity(t *testing.T) {
	t.Setenv("PARENT_SECRET_API_KEY", "parent-secret-3d0f")
	cfg := helperConfig(t, 10*time.Second)
	cfg.IdentityEnv = "EXAMPLE_WIKI_EMAIL"
	cfg.IdentityValue = "name@example.test"
	reply, err := plugin.Call(context.Background(), cfg, "env", nil, plugin.Seams{})
	if err != nil {
		t.Fatalf("Call error = %v, want the reply", err)
	}
	want := []string{"EXAMPLE_WIKI_API_KEY=[REDACTED]", "EXAMPLE_WIKI_EMAIL=[REDACTED]"}
	if got := envOf(t, reply.Result); !reflect.DeepEqual(got, want) {
		t.Fatalf("plugin environment = %q, want its credential and identity variables only", got)
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/real-process.feature: The output of a real plugin that echoes its environment holds no secret
func TestTheOutputOfARealPluginThatEchoesItsEnvironmentHoldsNoSecret(t *testing.T) {
	reply, err := plugin.Call(context.Background(), helperConfig(t, 10*time.Second), "env", nil, plugin.Seams{})
	if err != nil {
		t.Fatalf("Call error = %v, want the reply", err)
	}
	const secret = "credential-sentinel-9f2a"
	for _, text := range []string{string(reply.Result), reply.Stderr} {
		if strings.Contains(text, secret) {
			t.Fatalf("a plugin output holds the secret: %q", text)
		}
		if !strings.Contains(text, "[REDACTED]") {
			t.Fatalf("a plugin output reads no redaction marker: %q", text)
		}
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/real-process.feature: A real silent plugin is killed at the time limit and the run exits with 124
func TestARealSilentPluginIsKilledAtTheTimeLimitAndTheRunExits124(t *testing.T) {
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "pid")
	script := writeScript(t, dir, "plugin", "#!/bin/sh\necho $$ > '"+pidFile+"'\nsleep 300\n")
	cfg := helperConfigFor(t, script)
	cfg.Timeout = 500 * time.Millisecond
	start := time.Now()
	_, err := plugin.Call(context.Background(), cfg, "env", nil, plugin.Seams{})
	elapsed := time.Since(start)
	var timeout plugin.TimeoutError
	if !errors.As(err, &timeout) || !errors.Is(err, domain.ErrTimedOut) {
		t.Fatalf("Call error = %#v, want the timeout that the run maps to exit status 124", err)
	}
	if elapsed > 30*time.Second {
		t.Fatalf("the call took %s: the plugin was not killed at the limit", elapsed)
	}
	pid := readPID(pidFile)
	if pid <= 0 || !waitUntil(5*time.Second, func() bool { return processGone(pid) }) {
		t.Fatalf("the silent plugin %d still runs after the limit", pid)
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/real-process.feature: A real plugin whose bytes differ from the pin starts no process
func TestARealPluginWhoseBytesDifferFromThePinStartsNoProcess(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "started")
	script := writeScript(t, dir, "plugin", "#!/bin/sh\ntouch '"+marker+"'\n")
	ref, err := domain.NewPluginRef(script, strings.Repeat("f", 64))
	if err != nil {
		t.Fatalf("NewPluginRef: %v", err)
	}
	cfg := plugin.Config{Plugin: ref, CredentialEnv: "EXAMPLE_WIKI_API_KEY", CredentialValue: "credential-sentinel-9f2a", Timeout: 10 * time.Second}
	_, err = plugin.Call(context.Background(), cfg, "env", nil, plugin.Seams{})
	var pin plugin.PinMismatchError
	if !errors.As(err, &pin) {
		t.Fatalf("Call error = %#v, want the pin-mismatch error", err)
	}
	if _, statErr := os.Stat(marker); !errors.Is(statErr, fs.ErrNotExist) {
		t.Fatal("the plugin ran although its bytes differ from the pin")
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/real-process.feature: A real plugin that floods its output past 1 MiB fails the call
func TestARealPluginThatFloodsItsOutputPast1MiBFailsTheCall(t *testing.T) {
	_, err := plugin.Call(context.Background(), helperConfig(t, 10*time.Second), "flood", nil, plugin.Seams{})
	var protocol plugin.ProtocolError
	if !errors.As(err, &protocol) {
		t.Fatalf("Call error = %#v, want the frame refusal", err)
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/real-process.feature: A real plugin that exits with status 127 at once makes the run exit with 2 and never 127
func TestARealPluginThatExitsWith127AtOnceMakesTheRunExitWith2(t *testing.T) {
	dir := t.TempDir()
	// The plugin reads its request before it exits, so the write never races the exit.
	script := writeScript(t, dir, "plugin", "#!/bin/sh\nread line\nexit 127\n")
	_, err := plugin.Call(context.Background(), helperConfigFor(t, script), "env", nil, plugin.Seams{})
	var protocol plugin.ProtocolError
	if !errors.As(err, &protocol) || protocol.Rule != "the plugin answered nothing" {
		t.Fatalf("Call error = %#v, want the rule that the plugin answered nothing", err)
	}
	if protocol.ErrorCode() != plugin.ProtocolCode {
		t.Fatalf("code = %q, want the protocol code that the run maps to exit status 2", protocol.ErrorCode())
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/real-process.feature: A source served through the real fixture plugin maps its collections onto domain values
func TestASourceServedThroughTheRealFixturePluginMapsItsCollections(t *testing.T) {
	gw := gatewayFor(t, helperPath, false, &fixedCredentials{})
	got, err := gw.ListCollections(context.Background())
	if err != nil {
		t.Fatalf("ListCollections error = %v", err)
	}
	want := []domain.Collection{{ID: "c1", Name: "One"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("collections = %+v, want %+v", got, want)
	}
}
