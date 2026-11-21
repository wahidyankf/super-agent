package unit_test

import (
	"context"
	"errors"
	"testing"
	"time"

	plugin "github.com/wahidyankf/super-agent/libs/retriever-plugin"
)

// @scenario specs/libs/retriever-plugin/behaviours/pinning-and-start.feature: A plugin whose bytes differ from the pin fails the call with the pin-mismatch error before any process starts
func TestPinMismatchFailsBeforeAnyProcessStarts(t *testing.T) {
	files := newMemFiles([]byte("#!/bin/sh\necho v2\n"))
	cmd := newFakeCommand(okFrame(`{"ok":true}`))
	_, err := plugin.Call(context.Background(), callConfig(t, time.Second), "env", nil, seamsWith(files, cmd, never))
	var pin plugin.PinMismatchError
	if !errors.As(err, &pin) {
		t.Fatalf("Call error = %#v, want the pin-mismatch error", err)
	}
	if cmd.builds != 0 {
		t.Fatalf("a process was built %d time(s) for bytes that differ from the pin", cmd.builds)
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/pinning-and-start.feature: Replacing the original plugin file after the check does not change what runs
func TestReplacingTheOriginalAfterTheCheckChangesNothing(t *testing.T) {
	files := newMemFiles(pluginBody)
	files.onCopy = func(f *memFiles) {
		f.files[pluginPath] = []byte("#!/bin/sh\necho swapped\n")
	}
	cmd := newFakeCommand(okFrame(`{"ok":true}`))
	if _, err := plugin.Call(context.Background(), callConfig(t, time.Second), "env", nil, seamsWith(files, cmd, never)); err != nil {
		t.Fatalf("Call error = %v, want nil", err)
	}
	if cmd.path != copyPath {
		t.Fatalf("the process ran %q, want the checked copy %q", cmd.path, copyPath)
	}
	if string(files.files[copyPath]) != string(pluginBody) {
		t.Fatalf("the checked copy holds %q, want the pinned bytes", files.files[copyPath])
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/pinning-and-start.feature: A setup failure before the process runs is a start error with exit status 2
func TestASetupFailureBeforeTheProcessRunsIsAStartError(t *testing.T) {
	cases := []struct {
		name  string
		setup func(f *memFiles, c *fakeCommand)
	}{
		{"copying the plugin", func(f *memFiles, _ *fakeCommand) { f.copyErr = errors.New("boom") }},
		{"creating the private folder", func(f *memFiles, _ *fakeCommand) { f.mkdirErr = errors.New("boom") }},
		{"wiring the input pipe", func(_ *memFiles, c *fakeCommand) { c.stdinErr = errors.New("boom") }},
		{"wiring the output pipe", func(_ *memFiles, c *fakeCommand) { c.stdoutErr = errors.New("boom") }},
		{"wiring the error pipe", func(_ *memFiles, c *fakeCommand) { c.stderrErr = errors.New("boom") }},
		{"starting the process", func(_ *memFiles, c *fakeCommand) { c.startErr = errors.New("boom") }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			files := newMemFiles(pluginBody)
			cmd := newFakeCommand(okFrame(`{"ok":true}`))
			tc.setup(files, cmd)
			_, err := plugin.Call(context.Background(), callConfig(t, time.Second), "env", nil, seamsWith(files, cmd, never))
			var start plugin.StartError
			if !errors.As(err, &start) {
				t.Fatalf("Call error = %#v, want the start error", err)
			}
			// The stable code of a start error is the one the CLI maps to exit status 2, never 126 or 127.
			if start.ErrorCode() != plugin.StartFailedCode {
				t.Fatalf("code = %q, want %q", start.ErrorCode(), plugin.StartFailedCode)
			}
		})
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/pinning-and-start.feature: A plugin that exits at once before answering is reported as having answered nothing, with exit status 2
func TestAPluginThatExitsAtOnceIsReportedAsAnsweringNothing(t *testing.T) {
	files := newMemFiles(pluginBody)
	cmd := newFakeCommand("")
	_, err := plugin.Call(context.Background(), callConfig(t, time.Second), "env", nil, seamsWith(files, cmd, never))
	var protocol plugin.ProtocolError
	if !errors.As(err, &protocol) || protocol.Rule != "the plugin answered nothing" {
		t.Fatalf("Call error = %#v, want the rule that the plugin answered nothing", err)
	}
	// The protocol code is the one the CLI maps to exit status 2, and no child status ever surfaces as 127.
	if protocol.ErrorCode() != plugin.ProtocolCode {
		t.Fatalf("code = %q, want %q", protocol.ErrorCode(), plugin.ProtocolCode)
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/pinning-and-start.feature: A plugin that answers but never exits is killed after the grace period
func TestAPluginThatNeverExitsIsKilledAfterTheGracePeriod(t *testing.T) {
	files := newMemFiles(pluginBody)
	cmd := newFakeCommand(okFrame(`{"ok":true}`))
	cmd.holdWait = true
	reply, err := plugin.Call(context.Background(), callConfig(t, time.Second), "env", nil, seamsWith(files, cmd, graceFires))
	if err != nil {
		t.Fatalf("Call error = %v, want the reply", err)
	}
	if string(reply.Result) != `{"ok":true}` {
		t.Fatalf("result = %s, want the reply that the plugin wrote", reply.Result)
	}
	if cmd.killed() != 1 {
		t.Fatalf("process group kills = %d, want the plugin killed once after the grace period", cmd.killed())
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/pinning-and-start.feature: A plugin that exits promptly after its reply is never killed
func TestAPluginThatExitsPromptlyIsNeverKilled(t *testing.T) {
	files := newMemFiles(pluginBody)
	cmd := newFakeCommand(okFrame(`{"ok":true}`))
	if _, err := plugin.Call(context.Background(), callConfig(t, time.Second), "env", nil, seamsWith(files, cmd, never)); err != nil {
		t.Fatalf("Call error = %v, want nil", err)
	}
	if cmd.killed() != 0 {
		t.Fatalf("process group kills = %d, want none for a plugin that exits", cmd.killed())
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/pinning-and-start.feature: Killing a plugin ends its whole process group
func TestKillingAPluginEndsItsProcessGroup(t *testing.T) {
	files := newMemFiles(pluginBody)
	cmd := newFakeCommand("")
	cmd.holdStdout = true
	_, err := plugin.Call(context.Background(), callConfig(t, time.Second), "env", nil, seamsWith(files, cmd, fired))
	var timeout plugin.TimeoutError
	if !errors.As(err, &timeout) {
		t.Fatalf("Call error = %#v, want the timeout that kills the plugin", err)
	}
	// The host ends the process through the group command, which reaches every process the plugin started.
	if cmd.killed() != 1 {
		t.Fatalf("process group kills = %d, want one group kill", cmd.killed())
	}
}

// @scenario specs/libs/retriever-plugin/behaviours/pinning-and-start.feature: A stable code for a pin, protocol, or start failure is kept in every command that reports it
func TestAStableCodeIsKeptForEachFailure(t *testing.T) {
	cases := []struct {
		name    string
		failure func(t *testing.T) error
		code    string
	}{
		{"pin", func(t *testing.T) error {
			files := newMemFiles([]byte("#!/bin/sh\necho v2\n"))
			_, err := plugin.Call(context.Background(), callConfig(t, time.Second), "env", nil, seamsWith(files, newFakeCommand(""), never))
			return err
		}, plugin.PinMismatchCode},
		{"protocol", func(t *testing.T) error {
			files := newMemFiles(pluginBody)
			_, err := plugin.Call(context.Background(), callConfig(t, time.Second), "env", nil, seamsWith(files, newFakeCommand(""), never))
			return err
		}, plugin.ProtocolCode},
		{"start", func(t *testing.T) error {
			files := newMemFiles(pluginBody)
			files.copyErr = errors.New("boom")
			_, err := plugin.Call(context.Background(), callConfig(t, time.Second), "env", nil, seamsWith(files, newFakeCommand(""), never))
			return err
		}, plugin.StartFailedCode},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.failure(t)
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
