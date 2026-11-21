package unit_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	plugin "github.com/wahidyankf/super-agent/libs/retriever-plugin"
)

// @scenario specs/libs/retriever-plugin/behaviours/real-process.feature: A real plugin sees exactly its own credential variable and, when declared, its identity variable
func TestARealPluginSeesOnlyItsOwnCredentialAndIdentityVariables(t *testing.T) {
	cmd := newFakeCommand(okFrame(`{"env":[]}`))
	cfg := callConfig(t, time.Second)
	cfg.IdentityEnv = "EXAMPLE_WIKI_EMAIL"
	cfg.IdentityValue = "name@example.test"
	if _, err := plugin.Call(context.Background(), cfg, "env", nil, seamsWith(newMemFiles(pluginBody), cmd, never)); err != nil {
		t.Fatalf("Call error = %v, want nil", err)
	}
	want := []string{"EXAMPLE_WIKI_API_KEY=credential-sentinel-9f2a", "EXAMPLE_WIKI_EMAIL=name@example.test"}
	if !reflect.DeepEqual(cmd.env, want) {
		t.Fatalf("plugin environment = %q, want its credential and identity variables only", cmd.env)
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/real-process.feature: The output of a real plugin that echoes its environment holds no secret
func TestTheOutputOfAPluginThatEchoesItsEnvironmentHoldsNoSecret(t *testing.T) {
	const secret = "credential-sentinel-9f2a"
	cmd := newFakeCommand(okFrame(`{"env":["EXAMPLE_WIKI_API_KEY=` + secret + `"]}`))
	cmd.stderr = strings.NewReader("EXAMPLE_WIKI_API_KEY=" + secret + "\n")
	got, err := plugin.Call(context.Background(), callConfig(t, time.Second), "env", nil, seamsWith(newMemFiles(pluginBody), cmd, never))
	if err != nil {
		t.Fatalf("Call error = %v, want the reply", err)
	}
	if strings.Contains(string(got.Result), secret) || strings.Contains(got.Stderr, secret) {
		t.Fatalf("the output holds the secret: result %s, stderr %q", got.Result, got.Stderr)
	}
	if !strings.Contains(string(got.Result), "[REDACTED]") || !strings.Contains(got.Stderr, "[REDACTED]") {
		t.Fatalf("the output reads no redaction marker: result %s, stderr %q", got.Result, got.Stderr)
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/real-process.feature: A real silent plugin is killed at the time limit and the run exits with 124
func TestARealSilentPluginIsKilledAtTheTimeLimit(t *testing.T) {
	cmd := newFakeCommand("")
	cmd.holdStdout = true
	_, err := plugin.Call(context.Background(), callConfig(t, time.Second), "silent", nil, seamsWith(newMemFiles(pluginBody), cmd, fired))
	var timeout plugin.TimeoutError
	if !errors.As(err, &timeout) || !errors.Is(err, domain.ErrTimedOut) {
		t.Fatalf("Call error = %#v, want the timeout that the run maps to 124", err)
	}
	if cmd.killed() != 1 {
		t.Fatalf("process group kills = %d, want the plugin killed", cmd.killed())
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/real-process.feature: A real plugin whose bytes differ from the pin starts no process
func TestARealPluginWhoseBytesDifferFromThePinStartsNoProcess(t *testing.T) {
	cmd := newFakeCommand(okFrame(`{"env":[]}`))
	_, err := plugin.Call(context.Background(), callConfig(t, time.Second), "env", nil,
		seamsWith(newMemFiles([]byte("#!/bin/sh\necho other\n")), cmd, never))
	var pin plugin.PinMismatchError
	if !errors.As(err, &pin) {
		t.Fatalf("Call error = %#v, want the pin-mismatch error", err)
	}
	if cmd.builds != 0 {
		t.Fatalf("a process was built %d time(s) for other bytes", cmd.builds)
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/real-process.feature: A real plugin that floods its output past 1 MiB fails the call
func TestARealPluginThatFloodsItsOutputPast1MiBFailsTheCall(t *testing.T) {
	cmd := newFakeCommand(strings.Repeat("x", plugin.MaxFrameBytes+1))
	_, err := plugin.Call(context.Background(), callConfig(t, time.Second), "flood", nil, seamsWith(newMemFiles(pluginBody), cmd, never))
	var protocol plugin.ProtocolError
	if !errors.As(err, &protocol) {
		t.Fatalf("Call error = %#v, want the frame refusal", err)
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/real-process.feature: A real plugin that exits with status 127 at once makes the run exit with 2 and never 127
func TestARealPluginThatExitsWith127AtOnceAnswersNothing(t *testing.T) {
	cmd := newFakeCommand("")
	_, err := plugin.Call(context.Background(), callConfig(t, time.Second), "env", nil, seamsWith(newMemFiles(pluginBody), cmd, never))
	var protocol plugin.ProtocolError
	if !errors.As(err, &protocol) || protocol.Rule != "the plugin answered nothing" {
		t.Fatalf("Call error = %#v, want the rule that the plugin answered nothing", err)
	}
	if protocol.ErrorCode() != plugin.ProtocolCode {
		t.Fatalf("code = %q, want the protocol code that the run maps to exit status 2", protocol.ErrorCode())
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/real-process.feature: A source served through the real fixture plugin maps its collections onto domain values
func TestASourceServedThroughThePluginMapsItsCollectionsOntoDomainValues(t *testing.T) {
	gw, _ := gatewayOver(t, false, newMemFiles(pluginBody), never,
		answering(okFrame(`{"collections":[{"id":"c1","name":"One"}]}`)))
	got, err := gw.ListCollections(context.Background())
	if err != nil {
		t.Fatalf("ListCollections error = %v", err)
	}
	want := []domain.Collection{{ID: "c1", Name: "One"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("collections = %+v, want %+v", got, want)
	}
}
