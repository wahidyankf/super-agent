package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-plugin"
)

// errNoPluginProcess is the failure of any plugin call the unit-test host did not set up: the test would otherwise
// start a real process.
var errNoPluginProcess = errors.New("the unit-test host starts no plugin")

// failingPluginFiles refuses every file call of a plugin call.
type failingPluginFiles struct{}

func (failingPluginFiles) MakePrivateDir() (string, error) { return "", errNoPluginProcess }

func (failingPluginFiles) CopyFile(string, string) error { return errNoPluginProcess }

func (failingPluginFiles) HashFile(string) (string, error) { return "", errNoPluginProcess }

func (failingPluginFiles) RemoveAll(string) error { return nil }

// failingPluginStart refuses every process start of a plugin call.
func failingPluginStart(string, string, []string) (plugin.Proc, error) {
	return nil, errNoPluginProcess
}

// pluginBytesV1 are the bytes of the fixture plugin file the fake files serve.
var pluginBytesV1 = []byte("#!/bin/sh\necho v1\n")

// pinHexOf returns the lower-case hex SHA-256 of data.
func pinHexOf(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// memoryPluginFiles serves one plugin file with pluginBytesV1 at path, recording every hash and removal.
type memoryPluginFiles struct {
	path   string
	hashed []string
}

func (f memoryPluginFiles) MakePrivateDir() (string, error) { return "/private/1", nil }

func (f memoryPluginFiles) CopyFile(src, dst string) error {
	if src != f.path {
		return errors.New("memoryPluginFiles: unknown source")
	}
	return nil
}

func (f *memoryPluginFiles) HashFile(path string) (string, error) {
	f.hashed = append(f.hashed, path)
	return pinHexOf(pluginBytesV1), nil
}

func (f memoryPluginFiles) RemoveAll(string) error { return nil }

// recordingStarter counts every process start and answers the scripted proc or error.
type recordingStarter struct {
	calls int
	proc  plugin.Proc
	err   error
	panic string
}

func (s *recordingStarter) start(_, _ string, _ []string) (plugin.Proc, error) {
	s.calls++
	if s.panic != "" {
		panic(s.panic)
	}
	if s.err != nil {
		return nil, s.err
	}
	return s.proc, nil
}

// blockingProc never answers: Read waits for Kill, for the timeout tests.
type blockingProc struct {
	unblock chan struct{}
	killed  int
	waited  int
}

func (p *blockingProc) Write([]byte) error { return nil }

func (p *blockingProc) Read() ([]byte, error) {
	<-p.unblock
	return nil, errors.New("blockingProc: killed while reading")
}

func (p *blockingProc) Stderr() []byte { return nil }

func (p *blockingProc) Kill() error {
	p.killed++
	close(p.unblock)
	return nil
}

func (p *blockingProc) Wait() error {
	p.waited++
	return nil
}

// replyProc answers one fixed frame.
type replyProc struct{ frame string }

func (p replyProc) Write([]byte) error { return nil }

func (p replyProc) Read() ([]byte, error) { return []byte(p.frame), nil }

func (p replyProc) Stderr() []byte { return nil }

func (p replyProc) Kill() error { return nil }

func (p replyProc) Wait() error { return nil }

// pluginCatalogYAML returns a catalog with one outline source served by the plugin at path with pin.
func pluginCatalogYAML(path, pin string) string {
	return `version: 1
sources:
  - id: docs
    kind: outline
    endpoint: https://docs.example.test
    credential_env: EXAMPLE_OUTLINE_API_KEY
    plugin:
      path: ` + path + `
      sha256: ` + pin + `
`
}

// pluginHost returns the unit-test host whose catalog, env file, and plugin seams the test sets: the plugin file
// holds pluginBytesV1, and starter is the process seam.
func pluginHost(t *testing.T, catalog string, starter *recordingStarter, after func(time.Duration) <-chan time.Time) host {
	t.Helper()
	root := fstest.MapFS{
		"sources.yml": &fstest.MapFile{Data: []byte(catalog)},
		"keys.env":    &fstest.MapFile{Data: []byte("EXAMPLE_OUTLINE_API_KEY=example-key\n")},
	}
	h := hostOver(func() (string, error) { return "/work/project", nil }, root, fstest.MapFS{})
	files := &memoryPluginFiles{path: "/opt/plugins/wiki"}
	h.plugin = plugin.Seams{Files: files, Start: starter.start, After: after}
	return h
}

// openChan returns a channel no test closes: the After of a call that must not time out.
func openChan() <-chan time.Time { return make(chan time.Time) }

// closedChan returns an already-closed channel: the After of a call whose timer has fired.
func closedChan() <-chan time.Time {
	ch := make(chan time.Time)
	close(ch)
	return ch
}

// runCollections runs the outline collections command over the plugin host and returns the status and the streams.
func runCollections(t *testing.T, h host, output string) (int, string, string) {
	t.Helper()
	argv := []string{"--output", output, "--catalog", "/work/project/sources.yml", "--env-file", "/work/project/keys.env",
		"outline", "collections"}
	status := run(argv, func(string) string { return "" }, h)
	return status, h.stdout.(*bytes.Buffer).String(), h.stderr.(*bytes.Buffer).String()
}

// TestPluginSourceResolvesToAPluginGateway checks the routing: a source with a plugin block is served by the plugin
// gateway, whatever its kind.
func TestPluginSourceResolvesToAPluginGateway(t *testing.T) {
	starter := &recordingStarter{proc: replyProc{frame: `{"jsonrpc":"2.0","id":1,"result":{"collections":[]}}`}}
	h := pluginHost(t, pluginCatalogYAML("/opt/plugins/wiki", pinHexOf(pluginBytesV1)), starter,
		func(time.Duration) <-chan time.Time { return openChan() })
	w, err := wireSources(examplePaths, seamSet{}, h, registeredKinds())
	if err != nil {
		t.Fatalf("wireSources error = %v, want nil", err)
	}
	src := w.service.Catalog.All()[0]
	if !src.HasPlugin() {
		t.Fatal("the wired source must have a plugin")
	}
	gw, err := w.service.Gateways.For(src)
	if err != nil {
		t.Fatalf("For(plugin source) error = %v, want nil", err)
	}
	if _, ok := gw.(*plugin.Gateway); !ok {
		t.Fatalf("gateway = %T, want *plugin.Gateway", gw)
	}
	if starter.calls != 0 {
		t.Fatalf("starts = %d, want no process at build time", starter.calls)
	}
}

// TestRunWithoutAPluginBlockStartsNoProcess checks P2 end to end: a catalog with no plugin block runs a source
// command with no process start.
func TestRunWithoutAPluginBlockStartsNoProcess(t *testing.T) {
	starter := &recordingStarter{}
	h := memoryHost()
	h.plugin = plugin.Seams{Files: failingPluginFiles{}, Start: starter.start,
		After: func(time.Duration) <-chan time.Time { return openChan() }}
	status := run([]string{"--catalog", "/work/project/sources.yml", "--env-file", "/work/project/keys.env",
		"sources", "list"}, func(string) string { return "" }, h)
	if status != 0 {
		t.Fatalf("sources list status = %d, want 0", status)
	}
	if starter.calls != 0 {
		t.Fatalf("starts = %d, want no process without a plugin block", starter.calls)
	}
}

// TestPluginPinMismatchStopsTheSource checks the first gate end to end: a plugin whose bytes differ from its pin
// fails the source command with plugin_pin_mismatch and exit 2, and no process starts.
func TestPluginPinMismatchStopsTheSource(t *testing.T) {
	starter := &recordingStarter{}
	h := pluginHost(t, pluginCatalogYAML("/opt/plugins/wiki", strings.Repeat("f", 64)), starter,
		func(time.Duration) <-chan time.Time { return openChan() })
	status, out, errOut := runCollections(t, h, "json")
	if status != 2 {
		t.Fatalf("status = %d, want 2", status)
	}
	if out != "" {
		t.Fatalf("stdout = %q, want nothing on a failed run", out)
	}
	if !strings.Contains(errOut, `"code":"retriever.connector.plugin_pin_mismatch"`) {
		t.Fatalf("stderr = %q, want the pin code", errOut)
	}
	if !strings.Contains(errOut, `"exitCode":2`) {
		t.Fatalf("stderr = %q, want exit code 2", errOut)
	}
	if starter.calls != 0 {
		t.Fatalf("starts = %d, want no process on a pin mismatch", starter.calls)
	}
}

// TestSilentPluginExits124 checks the third gate end to end: a plugin that never answers is killed, and the source
// command exits 124 with nothing on standard output.
func TestSilentPluginExits124(t *testing.T) {
	starter := &recordingStarter{proc: &blockingProc{unblock: make(chan struct{})}}
	h := pluginHost(t, pluginCatalogYAML("/opt/plugins/wiki", pinHexOf(pluginBytesV1)), starter,
		func(time.Duration) <-chan time.Time { return closedChan() })
	status, out, errOut := runCollections(t, h, "json")
	if status != 124 {
		t.Fatalf("status = %d, want 124 (stderr %q)", status, errOut)
	}
	if out != "" {
		t.Fatalf("stdout = %q, want nothing on a failed run", out)
	}
	proc := starter.proc.(*blockingProc)
	if proc.killed == 0 {
		t.Fatal("a silent plugin was not killed")
	}
}

// TestFailingPluginStartExitsTwo checks that a plugin that cannot start fails with plugin_start_failed and exit 2,
// never 126 or 127: the child's own statuses never surface.
func TestFailingPluginStartExitsTwo(t *testing.T) {
	starter := &recordingStarter{err: errors.New("no exec")}
	h := pluginHost(t, pluginCatalogYAML("/opt/plugins/wiki", pinHexOf(pluginBytesV1)), starter,
		func(time.Duration) <-chan time.Time { return openChan() })
	status, out, errOut := runCollections(t, h, "json")
	if status != 2 {
		t.Fatalf("status = %d, want 2 (stderr %q)", status, errOut)
	}
	if out != "" {
		t.Fatalf("stdout = %q, want nothing on a failed run", out)
	}
	if !strings.Contains(errOut, `"code":"retriever.connector.plugin_start_failed"`) {
		t.Fatalf("stderr = %q, want the start code, not 126 or 127", errOut)
	}
}

// TestRefusedPluginReplyKeepsItsCode checks that a reply with the wrong id fails the source command with
// plugin_protocol and exit 2.
func TestRefusedPluginReplyKeepsItsCode(t *testing.T) {
	starter := &recordingStarter{proc: replyProc{frame: `{"jsonrpc":"2.0","id":2,"result":{}}`}}
	h := pluginHost(t, pluginCatalogYAML("/opt/plugins/wiki", pinHexOf(pluginBytesV1)), starter,
		func(time.Duration) <-chan time.Time { return openChan() })
	status, out, errOut := runCollections(t, h, "json")
	if status != 2 {
		t.Fatalf("status = %d, want 2 (stderr %q)", status, errOut)
	}
	if out != "" {
		t.Fatalf("stdout = %q, want nothing on a failed run", out)
	}
	if !strings.Contains(errOut, `"code":"retriever.connector.plugin_protocol"`) {
		t.Fatalf("stderr = %q, want the protocol code", errOut)
	}
}

// TestPanickingPluginStartExitsTwo checks that a panic in the plugin path is reported as the internal failure with
// exit 2: the runtime default of crashing is overridden for the new path too.
func TestPanickingPluginStartExitsTwo(t *testing.T) {
	starter := &recordingStarter{panic: "boom"}
	h := pluginHost(t, pluginCatalogYAML("/opt/plugins/wiki", pinHexOf(pluginBytesV1)), starter,
		func(time.Duration) <-chan time.Time { return openChan() })
	status, out, errOut := runCollections(t, h, "text")
	if status != 2 {
		t.Fatalf("status = %d, want 2 (stderr %q)", status, errOut)
	}
	if out != "" {
		t.Fatalf("stdout = %q, want nothing on a failed run", out)
	}
	if !strings.Contains(errOut, "retriever.panic") {
		t.Fatalf("stderr = %q, want the panic code", errOut)
	}
}

// TestPluginGatewayServesACollectionList checks the success path through the whole run: a plugin that answers with
// collections prints them with exit 0.
func TestPluginGatewayServesACollectionList(t *testing.T) {
	starter := &recordingStarter{proc: replyProc{frame: `{"jsonrpc":"2.0","id":1,"result":{"items":[{"id":"c1","name":"One"}]}}`}}
	h := pluginHost(t, pluginCatalogYAML("/opt/plugins/wiki", pinHexOf(pluginBytesV1)), starter,
		func(time.Duration) <-chan time.Time { return openChan() })
	status, out, errOut := runCollections(t, h, "text")
	if status != 0 {
		t.Fatalf("status = %d, want 0 (stderr %q)", status, errOut)
	}
	if !strings.Contains(out, "c1") || !strings.Contains(out, "One") {
		t.Fatalf("stdout = %q, want the plugin collection", out)
	}
	if starter.calls != 1 {
		t.Fatalf("starts = %d, want one process for one call", starter.calls)
	}
}

// TestEveryKindWithoutAPluginKeepsItsBuilder checks that the wrapped builders still serve sources with no plugin:
// the plugin routing changes nothing for them.
func TestEveryKindWithoutAPluginKeepsItsBuilder(t *testing.T) {
	h := memoryHost()
	w, err := wireSources(examplePaths, seamSet{}, h, registeredKinds())
	if err != nil {
		t.Fatalf("wireSources error = %v, want nil", err)
	}
	for _, src := range w.service.Catalog.All() {
		gw, err := w.service.Gateways.For(src)
		if err != nil {
			t.Fatalf("gateway for %s: %v", src.ID(), err)
		}
		if _, ok := gw.(*plugin.Gateway); ok {
			t.Fatalf("source %s without a plugin block must not use the plugin gateway", src.ID())
		}
	}
}
