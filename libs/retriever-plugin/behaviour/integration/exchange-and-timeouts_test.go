//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
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

// @scenario specs/libs/retriever-plugin/behaviours/exchange-and-timeouts.feature: One call sends one request and reads one reply, passes the error output through, and removes the private folder
func TestOneRealCallSendsOneRequestPassesErrorOutputAndRemovesItsFolder(t *testing.T) {
	files := &realFiles{}
	cfg := helperConfig(t, 10*time.Second)
	reply, err := plugin.Call(context.Background(), cfg, "list_collections", nil, plugin.Seams{Files: files})
	if err != nil {
		t.Fatalf("Call error = %v, want the reply", err)
	}
	// Only the list_collections method answers with collections, so the result proves the request carried the method.
	if !strings.Contains(string(reply.Result), `"c1"`) {
		t.Fatalf("result = %s, want the collections of the method that was asked", reply.Result)
	}
	envReply, err := plugin.Call(context.Background(), cfg, "env", nil, plugin.Seams{Files: files})
	if err != nil {
		t.Fatalf("Call error = %v, want the reply", err)
	}
	if !strings.Contains(envReply.Stderr, "EXAMPLE_WIKI_API_KEY=[REDACTED]") {
		t.Fatalf("stderr = %q, want the plugin's error output passed through", envReply.Stderr)
	}
	if len(files.made) != 2 {
		t.Fatalf("private folders = %d, want one per call", len(files.made))
	}
	for _, dir := range files.made {
		if _, err := os.Stat(dir); !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("private folder %s still exists after the call", dir)
		}
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/exchange-and-timeouts.feature: A reply that breaks the one-object rule is refused with a protocol error
func TestARealPluginReplyThatBreaksTheOneObjectRuleIsRefused(t *testing.T) {
	replies := []struct {
		name  string
		reply string
	}{
		{"an empty frame or a blank frame", "\n"},
		{"a batch of replies", `[{"jsonrpc":"2.0","id":1,"result":{}}]` + "\n"},
		{"text that is not JSON", "not json\n"},
		{"a JSON value that is not an object", `"text"` + "\n"},
		{"a reply followed by extra data", `{"jsonrpc":"2.0","id":1,"result":{}} {}` + "\n"},
		{"a reply with the version 1.0", `{"jsonrpc":"1.0","id":1,"result":{}}` + "\n"},
		{"a reply with no version", `{"id":1,"result":{}}` + "\n"},
		{"a notification with no id", `{"jsonrpc":"2.0","result":{}}` + "\n"},
		{"a reply whose id is null", `{"jsonrpc":"2.0","id":null,"result":{}}` + "\n"},
		{"a reply whose id is 2", `{"jsonrpc":"2.0","id":2,"result":{}}` + "\n"},
		{"a reply whose id is the text 1", `{"jsonrpc":"2.0","id":"1","result":{}}` + "\n"},
		{"a reply that holds an error object", `{"jsonrpc":"2.0","id":1,"error":{"code":-1,"message":"x"}}` + "\n"},
		{"a reply that holds both a result and an error", `{"jsonrpc":"2.0","id":1,"result":{},"error":{"code":-1}}` + "\n"},
		{"a reply that holds no result", `{"jsonrpc":"2.0","id":1}` + "\n"},
	}
	for _, tc := range replies {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			replyFile := filepath.Join(dir, "reply.txt")
			if err := os.WriteFile(replyFile, []byte(tc.reply), 0o600); err != nil {
				t.Fatalf("write the reply: %v", err)
			}
			script := writeScript(t, dir, "plugin", "#!/bin/sh\ncat '"+replyFile+"'\n")
			_, err := plugin.Call(context.Background(), helperConfigFor(t, script), "env", nil, plugin.Seams{})
			var protocol plugin.ProtocolError
			if !errors.As(err, &protocol) {
				t.Fatalf("Call error = %#v, want a protocol error", err)
			}
		})
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/exchange-and-timeouts.feature: A reply with an object result passes through
func TestARealObjectResultPassesThroughUnchanged(t *testing.T) {
	reply, err := plugin.Call(context.Background(), helperConfig(t, 10*time.Second), "list_collections", nil, plugin.Seams{})
	if err != nil {
		t.Fatalf("Call error = %v, want the reply", err)
	}
	var got, want map[string]any
	if err := json.Unmarshal(reply.Result, &got); err != nil {
		t.Fatalf("result is not an object: %v", err)
	}
	if err := json.Unmarshal([]byte(`{"collections":[{"id":"c1","name":"One"}]}`), &want); err != nil {
		t.Fatalf("expected object: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("result = %s, want the object the plugin wrote", reply.Result)
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/exchange-and-timeouts.feature: A reply with a null result is refused by the gateway as a missing object
func TestARealNullResultIsRefusedByTheGateway(t *testing.T) {
	dir := t.TempDir()
	script := writeScript(t, dir, "plugin", "#!/bin/sh\nread line\nprintf '%s\\n' '{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":null}'\n")
	gw := gatewayFor(t, script, false, &fixedCredentials{})
	doc, err := domain.NewDocumentID("doc-1")
	if err != nil {
		t.Fatalf("NewDocumentID: %v", err)
	}
	_, err = gw.Document(context.Background(), doc)
	var protocol plugin.ProtocolError
	if !errors.As(err, &protocol) {
		t.Fatalf("error = %#v, want the null result refused as a protocol error", err)
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/exchange-and-timeouts.feature: A frame over 1 MiB fails the call
func TestAFloodingPluginFailsTheCallWithAFrameOver1MiB(t *testing.T) {
	_, err := plugin.Call(context.Background(), helperConfig(t, 10*time.Second), "flood", nil, plugin.Seams{})
	var protocol plugin.ProtocolError
	if !errors.As(err, &protocol) || protocol.Rule != "the plugin frame is larger than 1 MiB" {
		t.Fatalf("Call error = %#v, want the frame refusal", err)
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/exchange-and-timeouts.feature: The error output of a plugin is kept to its first 64 KiB however it is chunked
func TestTheErrorOutputOfARealPluginIsKeptToItsFirst64KiB(t *testing.T) {
	dir := t.TempDir()
	long := filepath.Join(dir, "long.txt")
	if err := os.WriteFile(long, []byte(strings.Repeat("e", 100<<10)), 0o600); err != nil {
		t.Fatalf("write the long output: %v", err)
	}
	script := writeScript(t, dir, "plugin", "#!/bin/sh\nread line\ncat '"+long+"' >&2\n"+
		"printf '%s\\n' '{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"ok\":true}}'\n")
	reply, err := plugin.Call(context.Background(), helperConfigFor(t, script), "env", nil, plugin.Seams{})
	if err != nil {
		t.Fatalf("Call error = %v, want the reply", err)
	}
	if len(reply.Stderr) != 64<<10 || reply.Stderr != strings.Repeat("e", 64<<10) {
		t.Fatalf("kept error output holds %d bytes, want exactly the first 64 KiB", len(reply.Stderr))
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/exchange-and-timeouts.feature: A plugin that never answers is killed at the time limit and the call fails with exit status 124
func TestASilentRealPluginIsKilledAtTheTimeLimit(t *testing.T) {
	start := time.Now()
	_, err := plugin.Call(context.Background(), helperConfig(t, 300*time.Millisecond), "silent", nil, plugin.Seams{})
	elapsed := time.Since(start)
	var timeout plugin.TimeoutError
	if !errors.As(err, &timeout) || !errors.Is(err, domain.ErrTimedOut) {
		t.Fatalf("Call error = %#v, want the timeout that unwraps to domain.ErrTimedOut", err)
	}
	if elapsed > 30*time.Second {
		t.Fatalf("a silenced call took %s: the plugin was not killed at the limit", elapsed)
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/exchange-and-timeouts.feature: A cancelled caller ends the plugin and the call fails with the caller's own error
func TestACancelledCallerEndsARealPluginAndFailsWithItsOwnError(t *testing.T) {
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "pid")
	script := writeScript(t, dir, "plugin", "#!/bin/sh\necho $$ > '"+pidFile+"'\nsleep 300\n")
	cfg := helperConfigFor(t, script)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := plugin.Call(ctx, cfg, "env", nil, plugin.Seams{})
		done <- err
	}()
	if !waitUntil(10*time.Second, func() bool { return readPID(pidFile) > 0 }) {
		t.Fatal("the plugin never started")
	}
	pid := readPID(pidFile)
	cancel()
	var err error
	select {
	case err = <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("the call did not end after its caller cancelled")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Call error = %#v, want the caller's context.Canceled", err)
	}
	if !waitUntil(5*time.Second, func() bool { return processGone(pid) }) {
		t.Fatalf("the plugin process %d still runs after the caller cancelled", pid)
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/exchange-and-timeouts.feature: A protocol failure from the exchange passes through unchanged, and any other failure maps onto the fixed rules
func TestAProtocolFailurePassesThroughAndAnotherFailureMapsOntoTheFixedRule(t *testing.T) {
	dir := t.TempDir()
	wrongID := writeScript(t, dir, "wrong-id", "#!/bin/sh\nread line\n"+
		"printf '%s\\n' '{\"jsonrpc\":\"2.0\",\"id\":2,\"result\":{}}'\n")
	silentExit := writeScript(t, dir, "silent-exit", "#!/bin/sh\nread line\nexit 0\n")

	_, err := plugin.Call(context.Background(), helperConfigFor(t, wrongID), "env", nil, plugin.Seams{})
	var protocol plugin.ProtocolError
	if !errors.As(err, &protocol) || protocol.Rule != "the plugin reply id does not match the request" {
		t.Fatalf("a protocol failure = %#v, want its own rule unchanged", err)
	}
	_, err = plugin.Call(context.Background(), helperConfigFor(t, silentExit), "env", nil, plugin.Seams{})
	if !errors.As(err, &protocol) || protocol.Rule != "the plugin answered nothing" {
		t.Fatalf("an exit with no reply = %#v, want the fixed protocol rule", err)
	}
}
