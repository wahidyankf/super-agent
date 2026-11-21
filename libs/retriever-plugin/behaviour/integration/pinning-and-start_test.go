//go:build integration

package integration_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	plugin "github.com/wahidyankf/super-agent/libs/retriever-plugin"
)

// @scenario specs/libs/retriever-plugin/behaviours/pinning-and-start.feature: A plugin whose bytes differ from the pin fails the call with the pin-mismatch error before any process starts
func TestARealPluginWithOtherBytesFailsBeforeAnyProcessStarts(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "started")
	script := writeScript(t, dir, "plugin", "#!/bin/sh\ntouch '"+marker+"'\n")
	ref, err := domain.NewPluginRef(script, strings.Repeat("0", 64))
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

// @scenario specs/libs/retriever-plugin/behaviours/pinning-and-start.feature: Replacing the original plugin file after the check does not change what runs
func TestReplacingTheOriginalPluginFileDoesNotChangeWhatRuns(t *testing.T) {
	dir := t.TempDir()
	original := filepath.Join(dir, "plugin")
	helper, err := os.ReadFile(helperPath)
	if err != nil {
		t.Fatalf("read the fixture plugin: %v", err)
	}
	// The test copies the fixture before it swaps the original, so the shared fixture is never changed.
	if err := os.WriteFile(original, helper, 0o700); err != nil {
		t.Fatalf("write the original plugin: %v", err)
	}
	swapped := []byte("#!/bin/sh\necho swapped\n")
	files := &realFiles{afterCopy: func(src string) {
		_ = os.WriteFile(src, swapped, 0o700)
	}}
	reply, err := plugin.Call(context.Background(), helperConfigFor(t, original), "list_collections", nil, plugin.Seams{Files: files})
	if err != nil {
		t.Fatalf("Call error = %v, want the reply of the checked copy", err)
	}
	if got, _ := os.ReadFile(original); string(got) != string(swapped) {
		t.Fatal("the original was not replaced after the check, so the test proves nothing")
	}
	if !strings.Contains(string(reply.Result), `"c1"`) {
		t.Fatalf("result = %s, want the collections the checked copy serves", reply.Result)
	}
}

// helperConfigFor returns the call configuration of the plugin at path, pinned by its bytes before the call.
func helperConfigFor(t *testing.T, path string) plugin.Config {
	t.Helper()
	return plugin.Config{
		Plugin:          checkedRef(t, path),
		CredentialEnv:   "EXAMPLE_WIKI_API_KEY",
		CredentialValue: "credential-sentinel-9f2a",
		Timeout:         10 * time.Second,
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/pinning-and-start.feature: A setup failure before the process runs is a start error with exit status 2
func TestASetupFailureWithRealFilesIsAStartError(t *testing.T) {
	// The pipe-wiring rows fail only inside the process wiring, where no real file or folder can fail on demand. The unit
	// binding covers those rows through the command seam, and this binding covers the three that real files reach.
	cases := []struct {
		name  string
		setup func(t *testing.T, dir string) plugin.Config
	}{
		{"copying the plugin", func(t *testing.T, dir string) plugin.Config {
			ref, err := domain.NewPluginRef(filepath.Join(dir, "missing"), strings.Repeat("a", 64))
			if err != nil {
				t.Fatalf("NewPluginRef: %v", err)
			}
			return plugin.Config{Plugin: ref, CredentialEnv: "EXAMPLE_WIKI_API_KEY", CredentialValue: "v", Timeout: 10 * time.Second}
		}},
		{"creating the private folder", func(t *testing.T, dir string) plugin.Config {
			// The private folder is made in the temporary folder, so a missing one makes it fail.
			t.Setenv("TMPDIR", filepath.Join(dir, "missing"))
			return helperConfig(t, 10*time.Second)
		}},
		{"starting the process", func(t *testing.T, dir string) plugin.Config {
			// The copy is made executable, so the start fails only when the kernel cannot run the file: its interpreter is
			// missing. The copy and the pin check both pass first.
			path := filepath.Join(dir, "plugin")
			if err := os.WriteFile(path, []byte("#!/nonexistent-interpreter-9d1c/sh\nexit 0\n"), 0o600); err != nil {
				t.Fatalf("write the plugin: %v", err)
			}
			return helperConfigFor(t, path)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := tc.setup(t, t.TempDir())
			_, err := plugin.Call(context.Background(), cfg, "env", nil, plugin.Seams{})
			var start plugin.StartError
			if !errors.As(err, &start) {
				t.Fatalf("Call error = %#v, want the start error", err)
			}
			if start.ErrorCode() != plugin.StartFailedCode {
				t.Fatalf("code = %q, want %q", start.ErrorCode(), plugin.StartFailedCode)
			}
		})
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/pinning-and-start.feature: A plugin that exits at once before answering is reported as having answered nothing, with exit status 2
func TestAPluginThatExitsAtOnceIsReportedAsAnsweringNothing(t *testing.T) {
	dir := t.TempDir()
	// The plugin reads its request before it exits, so the host's write always lands. Without the read, an exit that wins
	// the race turns the write into "the plugin closed its input", which is also a protocol error with exit status 2.
	script := writeScript(t, dir, "plugin", "#!/bin/sh\nread line\nexit 127\n")
	cfg := plugin.Config{Plugin: checkedRef(t, script), CredentialEnv: "EXAMPLE_WIKI_API_KEY", CredentialValue: "v", Timeout: 10 * time.Second}
	_, err := plugin.Call(context.Background(), cfg, "env", nil, plugin.Seams{})
	var protocol plugin.ProtocolError
	if !errors.As(err, &protocol) || protocol.Rule != "the plugin answered nothing" {
		t.Fatalf("Call error = %#v, want the rule that the plugin answered nothing", err)
	}
	if protocol.ErrorCode() != plugin.ProtocolCode {
		t.Fatalf("code = %q, want %q", protocol.ErrorCode(), plugin.ProtocolCode)
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/pinning-and-start.feature: A plugin that answers but never exits is killed after the grace period
func TestAPluginThatAnswersAndNeverExitsIsKilledAfterTheGracePeriod(t *testing.T) {
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "pid")
	script := writeScript(t, dir, "plugin", "#!/bin/sh\n"+
		"echo $$ > '"+pidFile+"'\n"+
		"printf '%s\\n' '{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"ok\":true}}'\n"+
		"sleep 300\n")
	start := time.Now()
	reply, err := plugin.Call(context.Background(), helperConfigFor(t, script), "env", nil, plugin.Seams{})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("Call error = %v, want the reply the plugin wrote", err)
	}
	if string(reply.Result) != `{"ok":true}` {
		t.Fatalf("result = %s, want the reply", reply.Result)
	}
	// The host waits the grace period, five seconds, before it kills a plugin that has answered and not exited.
	if elapsed < 4*time.Second {
		t.Fatalf("the call returned after %s, before the grace period ran out", elapsed)
	}
	pid := readPID(pidFile)
	if pid <= 0 || !processGone(pid) {
		t.Fatalf("plugin process %d still runs after the call", pid)
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/pinning-and-start.feature: A plugin that exits promptly after its reply is never killed
func TestAPluginThatExitsPromptlyAfterItsReplyRunsToItsEnd(t *testing.T) {
	dir := t.TempDir()
	script := writeScript(t, dir, "plugin", "#!/bin/sh\n"+
		"read line\n"+
		"printf '%s\\n' '{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"ok\":true}}'\n"+
		"echo finished >&2\n"+
		"exit 0\n")
	start := time.Now()
	reply, err := plugin.Call(context.Background(), helperConfigFor(t, script), "env", nil, plugin.Seams{})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("Call error = %v, want the reply", err)
	}
	// The plugin's last line runs only if the host did not kill it before it exited on its own.
	if reply.Stderr != "finished\n" {
		t.Fatalf("stderr = %q, want the plugin's last line, written before its own exit", reply.Stderr)
	}
	if elapsed >= 4*time.Second {
		t.Fatalf("the call waited %s, so the host waited out the grace period for a plugin that had exited", elapsed)
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/pinning-and-start.feature: Killing a plugin ends its whole process group
func TestKillingAPluginEndsItsWholeProcessGroup(t *testing.T) {
	dir := t.TempDir()
	childPID := filepath.Join(dir, "child")
	// The plugin ignores the catchable signals, so only a kill that cannot be caught or ignored ends it and its child.
	script := writeScript(t, dir, "plugin", "#!/bin/sh\n"+
		"trap '' TERM INT HUP\n"+
		"sleep 300 &\n"+
		"echo $! > '"+childPID+"'\n"+
		"sleep 300\n")
	cfg := helperConfigFor(t, script)
	cfg.Timeout = 2 * time.Second
	_, err := plugin.Call(context.Background(), cfg, "env", nil, plugin.Seams{})
	var timeout plugin.TimeoutError
	if !errors.As(err, &timeout) {
		t.Fatalf("Call error = %#v, want the timeout that kills the plugin", err)
	}
	pid := readPID(childPID)
	if pid <= 0 {
		t.Fatal("the plugin never started its child process")
	}
	if !waitUntil(5*time.Second, func() bool { return processGone(pid) }) {
		t.Fatalf("the plugin's child process %d survived the kill of its group", pid)
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/pinning-and-start.feature: A stable code for a pin, protocol, or start failure is kept in every command that reports it
func TestAStableCodeIsKeptForEachRealFailure(t *testing.T) {
	dir := t.TempDir()
	exitsAt127 := writeScript(t, dir, "dies", "#!/bin/sh\nexit 127\n")
	cases := []struct {
		name string
		cfg  func(t *testing.T) plugin.Config
		code string
	}{
		{"pin", func(t *testing.T) plugin.Config {
			ref, err := domain.NewPluginRef(helperPath, strings.Repeat("0", 64))
			if err != nil {
				t.Fatalf("NewPluginRef: %v", err)
			}
			return plugin.Config{Plugin: ref, CredentialEnv: "EXAMPLE_WIKI_API_KEY", CredentialValue: "v", Timeout: 10 * time.Second}
		}, plugin.PinMismatchCode},
		{"protocol", func(t *testing.T) plugin.Config {
			return plugin.Config{
				Plugin: checkedRef(t, exitsAt127), CredentialEnv: "EXAMPLE_WIKI_API_KEY", CredentialValue: "v", Timeout: 10 * time.Second,
			}
		}, plugin.ProtocolCode},
		{"start", func(t *testing.T) plugin.Config {
			ref, err := domain.NewPluginRef(filepath.Join(dir, "missing"), strings.Repeat("a", 64))
			if err != nil {
				t.Fatalf("NewPluginRef: %v", err)
			}
			return plugin.Config{Plugin: ref, CredentialEnv: "EXAMPLE_WIKI_API_KEY", CredentialValue: "v", Timeout: 10 * time.Second}
		}, plugin.StartFailedCode},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := plugin.Call(context.Background(), tc.cfg(t), "env", nil, plugin.Seams{})
			coded, ok := err.(interface {
				Code() string
				ErrorCode() string
			})
			if !ok {
				t.Fatalf("error %#v carries no stable code", err)
			}
			if coded.Code() != tc.code || coded.ErrorCode() != tc.code {
				t.Fatalf("codes = %q and %q, want %q in both", coded.Code(), coded.ErrorCode(), tc.code)
			}
		})
	}
}
