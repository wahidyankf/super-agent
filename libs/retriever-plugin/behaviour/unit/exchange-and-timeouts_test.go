package unit_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	plugin "github.com/wahidyankf/super-agent/libs/retriever-plugin"
)

// @scenario specs/libs/retriever-plugin/behaviours/exchange-and-timeouts.feature: One call sends one request and reads one reply, passes the error output through, and removes the private folder
func TestOneCallSendsOneRequestAndReadsOneReply(t *testing.T) {
	files := newMemFiles(pluginBody)
	cmd := newFakeCommand(okFrame(`{"collections":[]}`))
	cmd.stderr = strings.NewReader("warn: slow\n")
	reply, err := plugin.Call(context.Background(), callConfig(t, time.Second), "list_collections", nil, seamsWith(files, cmd, never))
	if err != nil {
		t.Fatalf("Call error = %v, want nil", err)
	}
	frames := cmd.sentFrames()
	if len(frames) != 1 {
		t.Fatalf("the host wrote %d request frames, want one", len(frames))
	}
	var req struct {
		JSONRPC string `json:"jsonrpc"`
		ID      int    `json:"id"`
		Method  string `json:"method"`
	}
	if err := json.Unmarshal(frames[0], &req); err != nil {
		t.Fatalf("request frame is not JSON: %v", err)
	}
	if req.Method != "list_collections" || req.ID != 1 || req.JSONRPC != "2.0" {
		t.Fatalf("request = %+v, want the method of the call with id 1", req)
	}
	if string(reply.Result) != `{"collections":[]}` {
		t.Fatalf("result = %s, want the decoded result", reply.Result)
	}
	if reply.Stderr != "warn: slow\n" {
		t.Fatalf("stderr = %q, want the error output passed through", reply.Stderr)
	}
	if !reflect.DeepEqual(files.removed, []string{"/private/1"}) {
		t.Fatalf("removed folders = %q, want the private folder removed", files.removed)
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/exchange-and-timeouts.feature: A reply that breaks the one-object rule is refused with a protocol error
func TestAReplyThatBreaksTheOneObjectRuleIsRefused(t *testing.T) {
	cases := []struct {
		name  string
		reply string
	}{
		{"an empty frame", "\n"},
		{"a blank frame", "   \n"},
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
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newFakeCommand(tc.reply)
			_, err := plugin.Call(context.Background(), callConfig(t, time.Second), "env", nil, seamsWith(newMemFiles(pluginBody), cmd, never))
			var protocol plugin.ProtocolError
			if !errors.As(err, &protocol) {
				t.Fatalf("Call error = %#v, want a protocol error", err)
			}
		})
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/exchange-and-timeouts.feature: A reply with an object result passes through
func TestAReplyWithAnObjectResultPassesThrough(t *testing.T) {
	cmd := newFakeCommand(okFrame(`{"a":1}`))
	reply, err := plugin.Call(context.Background(), callConfig(t, time.Second), "env", nil, seamsWith(newMemFiles(pluginBody), cmd, never))
	if err != nil {
		t.Fatalf("Call error = %v, want nil", err)
	}
	if string(reply.Result) != `{"a":1}` {
		t.Fatalf("result = %s, want the object unchanged", reply.Result)
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/exchange-and-timeouts.feature: A reply with a null result is refused by the gateway as a missing object
func TestANullResultIsRefusedByTheGatewayAsAMissingObject(t *testing.T) {
	creds := &fakeCredentials{apiKey: "credential-sentinel-9f2a"}
	gw := &plugin.Gateway{
		Source:      wikiSource(t, true, false),
		Credentials: creds,
		Identities:  creds,
		Seams:       seamsWith(newMemFiles(pluginBody), newFakeCommand(okFrame("null")), never),
		Timeout:     time.Second,
	}
	doc, err := domain.NewDocumentID("doc-1")
	if err != nil {
		t.Fatalf("NewDocumentID: %v", err)
	}
	_, err = gw.Document(context.Background(), doc)
	var protocol plugin.ProtocolError
	if !errors.As(err, &protocol) {
		t.Fatalf("error = %#v, want the gateway to refuse the null result as a protocol error", err)
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/exchange-and-timeouts.feature: A frame over 1 MiB fails the call
func TestAFrameOver1MiBFailsTheCall(t *testing.T) {
	cmd := newFakeCommand(strings.Repeat("x", plugin.MaxFrameBytes+1))
	_, err := plugin.Call(context.Background(), callConfig(t, time.Second), "env", nil, seamsWith(newMemFiles(pluginBody), cmd, never))
	var protocol plugin.ProtocolError
	if !errors.As(err, &protocol) || protocol.Rule != "the plugin frame is larger than 1 MiB" {
		t.Fatalf("Call error = %#v, want the frame refusal", err)
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/exchange-and-timeouts.feature: The error output of a plugin is kept to its first 64 KiB however it is chunked
func TestTheErrorOutputIsKeptToItsFirst64KiB(t *testing.T) {
	cmd := newFakeCommand(okFrame(`{"ok":true}`))
	long := strings.Repeat("e", 100<<10)
	cmd.stderr = iotest.OneByteReader(strings.NewReader(long))
	reply, err := plugin.Call(context.Background(), callConfig(t, time.Second), "env", nil, seamsWith(newMemFiles(pluginBody), cmd, never))
	if err != nil {
		t.Fatalf("Call error = %v, want nil", err)
	}
	if reply.Stderr != long[:64<<10] {
		t.Fatalf("kept error output holds %d bytes, want exactly the first 64 KiB", len(reply.Stderr))
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/exchange-and-timeouts.feature: A plugin that never answers is killed at the time limit and the call fails with exit status 124
func TestAPluginThatNeverAnswersIsKilledAtTheTimeLimit(t *testing.T) {
	cmd := newFakeCommand("")
	cmd.holdStdout = true
	_, err := plugin.Call(context.Background(), callConfig(t, time.Second), "env", nil, seamsWith(newMemFiles(pluginBody), cmd, fired))
	var timeout plugin.TimeoutError
	if !errors.As(err, &timeout) {
		t.Fatalf("Call error = %#v, want the timeout error", err)
	}
	if cmd.killed() != 1 {
		t.Fatalf("process group kills = %d, want the plugin killed once", cmd.killed())
	}
	if timeout.Limit != time.Second {
		t.Fatalf("limit = %s, want the configured second", timeout.Limit)
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/exchange-and-timeouts.feature: The timeout error is retryable and maps to exit status 124
func TestTheTimeoutErrorIsTheSentinelThatMapsToExitStatus124(t *testing.T) {
	cmd := newFakeCommand("")
	cmd.holdStdout = true
	_, err := plugin.Call(context.Background(), callConfig(t, time.Second), "env", nil, seamsWith(newMemFiles(pluginBody), cmd, fired))
	// The CLI marks domain.ErrTimedOut retryable and exits 124 on it. The library's part is that the timeout error
	// unwraps to that sentinel.
	if !errors.Is(err, domain.ErrTimedOut) {
		t.Fatalf("Call error = %#v, want it to unwrap to domain.ErrTimedOut", err)
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/exchange-and-timeouts.feature: A time limit that is not set takes the 30 second default
func TestATimeLimitThatIsNotSetTakesTheThirtySecondDefault(t *testing.T) {
	var waits []time.Duration
	after := func(d time.Duration) <-chan time.Time {
		waits = append(waits, d)
		return never(d)
	}
	cmd := newFakeCommand(okFrame(`{"ok":true}`))
	if _, err := plugin.Call(context.Background(), callConfig(t, 0), "env", nil, seamsWith(newMemFiles(pluginBody), cmd, after)); err != nil {
		t.Fatalf("Call error = %v, want nil", err)
	}
	if len(waits) == 0 || waits[0] != 30*time.Second {
		t.Fatalf("the first time limit = %v, want 30s", waits)
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/exchange-and-timeouts.feature: A cancelled caller ends the plugin and the call fails with the caller's own error
func TestACancelledCallerEndsThePluginAndFailsWithItsOwnError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cmd := newFakeCommand("")
	cmd.holdStdout = true
	_, err := plugin.Call(ctx, callConfig(t, time.Second), "env", nil, seamsWith(newMemFiles(pluginBody), cmd, never))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Call error = %#v, want the context's own error", err)
	}
	if cmd.killed() != 1 {
		t.Fatalf("process group kills = %d, want the plugin ended", cmd.killed())
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/exchange-and-timeouts.feature: A protocol failure from the exchange passes through unchanged, and any other failure maps onto the fixed rules
func TestAProtocolFailurePassesThroughAndAnyOtherFailureMapsOntoTheFixedRule(t *testing.T) {
	cases := []struct {
		name string
		proc fakeProc
		rule string
	}{
		{"a protocol failure on write keeps its rule", fakeProc{writeErr: plugin.ProtocolError{Rule: "the plugin closed its input"}},
			"the plugin closed its input"},
		{"a protocol failure on read keeps its rule", fakeProc{readErr: plugin.ProtocolError{Rule: "the plugin frame is not JSON"}},
			"the plugin frame is not JSON"},
		{"another failure on write maps onto the fixed rule", fakeProc{writeErr: errors.New("boom")},
			"could not write the plugin request"},
		{"another failure on read maps onto the fixed rule", fakeProc{readErr: errors.New("boom")},
			"could not read the plugin reply"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			proc := tc.proc
			seams := plugin.Seams{
				Files: newMemFiles(pluginBody),
				Start: func(string, string, []string) (plugin.Proc, error) { return &proc, nil },
				After: never,
			}
			_, err := plugin.Call(context.Background(), callConfig(t, time.Second), "env", nil, seams)
			var protocol plugin.ProtocolError
			if !errors.As(err, &protocol) || protocol.Rule != tc.rule {
				t.Fatalf("Call error = %#v, want the protocol rule %q", err, tc.rule)
			}
		})
	}
}
