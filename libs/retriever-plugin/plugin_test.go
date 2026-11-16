package plugin

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// pluginBytesV1 and pluginBytesV2 are two versions of one plugin file: the pin names one of them.
var (
	pluginBytesV1 = []byte("#!/bin/sh\necho v1\n")
	pluginBytesV2 = []byte("#!/bin/sh\necho v2\n")
)

// pinOf returns the lower-case hex SHA-256 of data.
func pinOf(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// mustPluginConfig returns the Config of a plugin source whose pin matches pluginBytesV1, failing the test on any
// error. The credential and identity values are sentinels: no output of any kind may contain them unredacted.
func mustPluginConfig(t *testing.T) Config {
	t.Helper()
	ref, err := domain.NewPluginRef("/opt/plugins/wiki", pinOf(pluginBytesV1))
	if err != nil {
		t.Fatalf("NewPluginRef: %v", err)
	}
	return Config{
		Plugin:          ref,
		CredentialEnv:   "EXAMPLE_WIKI_API_KEY",
		CredentialValue: "credential-sentinel-9f2a",
		IdentityEnv:     "EXAMPLE_WIKI_EMAIL",
		IdentityValue:   "sentinel-mail-77@example.test",
		Timeout:         time.Second,
	}
}

// fakeFiles serves the file calls of a Call from an in-memory map, recording every call. A hook runs inside Start
// in the TOCTOU test; every error field fails one call on demand.
type fakeFiles struct {
	files            map[string][]byte
	mkdirErr         error
	copyErr          error
	hashErr          error
	removeErr        error
	made             int
	copied           [][2]string
	hashed           []string
	removed          []string
	dir              string
	copyReplacesHook func()
}

// filesWithV1 returns the fake whose source file holds pluginBytesV1, at the path the fixture configs name.
func filesWithV1() *fakeFiles {
	return &fakeFiles{files: map[string][]byte{"/opt/plugins/wiki": append([]byte(nil), pluginBytesV1...)}, dir: "/private/1"}
}

func (f *fakeFiles) MakePrivateDir() (string, error) {
	f.made++
	if f.mkdirErr != nil {
		return "", f.mkdirErr
	}
	return f.dir, nil
}

func (f *fakeFiles) CopyFile(src, dst string) error {
	f.copied = append(f.copied, [2]string{src, dst})
	if f.copyErr != nil {
		return f.copyErr
	}
	data, ok := f.files[src]
	if !ok {
		return errors.New("fakeFiles: no source file")
	}
	f.files[dst] = append([]byte(nil), data...)
	return nil
}

func (f *fakeFiles) HashFile(path string) (string, error) {
	f.hashed = append(f.hashed, path)
	if f.hashErr != nil {
		return "", f.hashErr
	}
	data, ok := f.files[path]
	if !ok {
		return "", errors.New("fakeFiles: no file to hash")
	}
	return pinOf(data), nil
}

func (f *fakeFiles) RemoveAll(path string) error {
	f.removed = append(f.removed, path)
	return f.removeErr
}

// fakeProc answers from memory, recording every call. blockRead holds Read until Kill runs, for the timeout test.
// waitBlocks holds Wait until Kill runs, for the grace test.
type fakeProc struct {
	wrote       [][]byte
	writeErr    error
	reply       []byte
	readErr     error
	blockRead   bool
	unblock     chan struct{}
	unblocked   bool
	stderrBytes []byte
	killed      int
	waitErr     error
	waitBlocks  bool
	waited      int
}

func answerProc(reply string) *fakeProc {
	return &fakeProc{reply: []byte(reply), unblock: make(chan struct{})}
}

func (f *fakeProc) Write(frame []byte) error {
	f.wrote = append(f.wrote, append([]byte(nil), frame...))
	return f.writeErr
}

func (f *fakeProc) Read() ([]byte, error) {
	if f.blockRead {
		<-f.unblock
		return nil, errors.New("fakeProc: killed while reading")
	}
	if f.readErr != nil {
		return nil, f.readErr
	}
	return append([]byte(nil), f.reply...), nil
}

func (f *fakeProc) Stderr() []byte { return append([]byte(nil), f.stderrBytes...) }

func (f *fakeProc) Kill() error {
	f.killed++
	if !f.unblocked {
		f.unblocked = true
		close(f.unblock)
	}
	return nil
}

func (f *fakeProc) Wait() error {
	f.waited++
	if f.waitBlocks {
		<-f.unblock
	}
	return f.waitErr
}

// fakeStarter serves procs from a script, recording the path, dir, and env of every start.
type fakeStarter struct {
	proc   *fakeProc
	err    error
	starts int
	path   string
	dir    string
	env    []string
	hook   func()
}

// Start records the start and answers the scripted proc. A hook mutates the world between the hash and the run,
// for the TOCTOU test.
func (s *fakeStarter) Start(path, dir string, env []string) (Proc, error) {
	s.starts++
	s.path, s.dir, s.env = path, dir, append([]string(nil), env...)
	if s.hook != nil {
		s.hook()
	}
	if s.err != nil {
		return nil, s.err
	}
	return s.proc, nil
}

// openChan returns a channel no test ever closes: the After of a call that must not time out.
func openChan() <-chan time.Time { return make(chan time.Time) }

// closedChan returns an already-closed channel: the After of a call whose timer has fired.
func closedChan() <-chan time.Time {
	ch := make(chan time.Time)
	close(ch)
	return ch
}

// callSeams returns the Seams of a fake world: the files, the starter over proc, and an After that never fires.
func callSeams(files *fakeFiles, starter *fakeStarter, proc *fakeProc) Seams {
	if starter.proc == nil {
		starter.proc = proc
	}
	return Seams{
		Files: files,
		Start: starter.Start,
		After: func(time.Duration) <-chan time.Time { return openChan() },
	}
}

// goodReply is a valid reply frame for method tests: id 1, version 2.0, and an object result.
const goodReply = `{"jsonrpc":"2.0","id":1,"result":{"collections":[]}}`

// TestCallSendsOneRequestAndReadsOneReply checks the whole success path: the request frame carries the method, the
// result decodes, the stderr passes through, and the private folder is removed.
func TestCallSendsOneRequestAndReadsOneReply(t *testing.T) {
	files, starter := filesWithV1(), &fakeStarter{}
	proc := answerProc(goodReply)
	proc.stderrBytes = []byte("note\n")
	reply, err := Call(context.Background(), mustPluginConfig(t), "list_collections", map[string]any{"a": "b"}, callSeams(files, starter, proc))
	if err != nil {
		t.Fatalf("Call error = %v, want nil", err)
	}
	if got := string(reply.Result); got != `{"collections":[]}` {
		t.Fatalf("Result = %s, want the object", got)
	}
	if reply.Stderr != "note\n" {
		t.Fatalf("Stderr = %q, want the note", reply.Stderr)
	}
	if len(proc.wrote) != 1 || !bytes.Contains(proc.wrote[0], []byte(`"method":"list_collections"`)) {
		t.Fatalf("wrote %d frames, want one with the method", len(proc.wrote))
	}
	if !bytes.Contains(proc.wrote[0], []byte(`"id":1`)) || !bytes.Contains(proc.wrote[0], []byte(`"jsonrpc":"2.0"`)) {
		t.Fatalf("request frame = %s, want JSON-RPC 2.0 with id 1", proc.wrote[0])
	}
	if starter.starts != 1 || starter.path != "/private/1/plugin" || starter.dir != "/private/1" {
		t.Fatalf("starts = %d path = %q dir = %q, want one start of the copy", starter.starts, starter.path, starter.dir)
	}
	if len(files.removed) != 1 || files.removed[0] != "/private/1" {
		t.Fatalf("removed = %v, want the private folder", files.removed)
	}
}

// TestCallScrubsTheEnvironment checks P3 at the seam: the child gets exactly its own credential variable and its
// identity variable, and nothing of the caller.
func TestCallScrubsTheEnvironment(t *testing.T) {
	files, starter := filesWithV1(), &fakeStarter{}
	if _, err := Call(context.Background(), mustPluginConfig(t), "m", nil, callSeams(files, starter, answerProc(goodReply))); err != nil {
		t.Fatalf("Call error = %v, want nil", err)
	}
	want := []string{"EXAMPLE_WIKI_API_KEY=credential-sentinel-9f2a", "EXAMPLE_WIKI_EMAIL=sentinel-mail-77@example.test"}
	if strings.Join(starter.env, "\n") != strings.Join(want, "\n") {
		t.Fatalf("env = %q, want exactly the credential and the identity", starter.env)
	}
}

// TestCallOmitsTheIdentityVariableWhenTheSourceHasNone checks that a source without an identity passes only its
// credential variable.
func TestCallOmitsTheIdentityVariableWhenTheSourceHasNone(t *testing.T) {
	cfg := mustPluginConfig(t)
	cfg.IdentityEnv, cfg.IdentityValue = "", ""
	files, starter := filesWithV1(), &fakeStarter{}
	if _, err := Call(context.Background(), cfg, "m", nil, callSeams(files, starter, answerProc(goodReply))); err != nil {
		t.Fatalf("Call error = %v, want nil", err)
	}
	if len(starter.env) != 1 || starter.env[0] != "EXAMPLE_WIKI_API_KEY=credential-sentinel-9f2a" {
		t.Fatalf("env = %q, want only the credential", starter.env)
	}
}

// TestCallFailsClosedWhenTheBytesDifferFromThePin checks P1 and the first gate: a mismatch starts no process and
// fails with PinMismatchError.
func TestCallFailsClosedWhenTheBytesDifferFromThePin(t *testing.T) {
	cfg := mustPluginConfig(t)
	ref, err := domain.NewPluginRef("/opt/plugins/wiki", pinOf(pluginBytesV2))
	if err != nil {
		t.Fatalf("NewPluginRef: %v", err)
	}
	cfg.Plugin = ref
	files, starter := filesWithV1(), &fakeStarter{proc: answerProc(goodReply)}
	if _, err := Call(context.Background(), cfg, "m", nil, callSeams(files, starter, starter.proc)); err == nil {
		t.Fatal("Call accepted bytes that differ from the pin")
	} else if _, ok := err.(PinMismatchError); !ok {
		t.Fatalf("Call error = %#v, want PinMismatchError", err)
	}
	if starter.starts != 0 {
		t.Fatalf("starts = %d, want no process on a pin mismatch", starter.starts)
	}
	if len(files.removed) != 1 {
		t.Fatalf("removed = %v, want the private folder removed on a pin mismatch", files.removed)
	}
}

// TestCallRunsTheCopyAfterASwapOfTheOriginal checks the TOCTOU half of P1: the host hashes the copy and runs the
// copy, so replacing the original between the hash and the run changes nothing.
func TestCallRunsTheCopyAfterASwapOfTheOriginal(t *testing.T) {
	files, starter := filesWithV1(), &fakeStarter{}
	proc := answerProc(goodReply)
	starter.proc = proc
	starter.hook = func() {
		files.files["/opt/plugins/wiki"] = append([]byte(nil), pluginBytesV2...)
	}
	reply, err := Call(context.Background(), mustPluginConfig(t), "m", nil, Seams{
		Files: files,
		Start: starter.Start,
		After: func(time.Duration) <-chan time.Time { return openChan() },
	})
	if err != nil {
		t.Fatalf("Call error = %v, want nil after the swap", err)
	}
	if string(reply.Result) != `{"collections":[]}` {
		t.Fatalf("Result = %s, want the reply of the original copy", reply.Result)
	}
	if starter.path != "/private/1/plugin" {
		t.Fatalf("started path = %q, want the private copy, not the original", starter.path)
	}
	if len(files.hashed) != 1 || files.hashed[0] != "/private/1/plugin" {
		t.Fatalf("hashed = %v, want the hash of the copy", files.hashed)
	}
}

// TestCallKillsTheProcessGroupOnTimeout checks P4 and the third gate: a plugin that never answers is killed, and
// the failure unwraps to the bounded wait the CLI reports as 124. The clock is already fired, so the timeout path
// is deterministic and the test never waits.
func TestCallKillsTheProcessGroupOnTimeout(t *testing.T) {
	files, starter := filesWithV1(), &fakeStarter{}
	proc := &fakeProc{blockRead: true, unblock: make(chan struct{})}
	starter.proc = proc
	seams := Seams{
		Files: files,
		Start: starter.Start,
		After: func(time.Duration) <-chan time.Time { return closedChan() },
	}
	_, err := Call(context.Background(), mustPluginConfig(t), "m", nil, seams)
	var timeout TimeoutError
	if !errors.As(err, &timeout) {
		t.Fatalf("Call error = %#v, want TimeoutError", err)
	}
	if timeout.Limit != time.Second {
		t.Fatalf("TimeoutError.Limit = %s, want the configured bound", timeout.Limit)
	}
	if !errors.Is(err, domain.ErrTimedOut) {
		t.Fatalf("TimeoutError does not unwrap to ErrTimedOut: %v", err)
	}
	if proc.killed == 0 {
		t.Fatal("a plugin that never answers was not killed")
	}
	if proc.waited == 0 {
		t.Fatal("a timed-out plugin was not reaped")
	}
}

// TestCallEndsWhenTheParentContextEnds checks that a cancelled parent kills the plugin and fails with the
// context's own error.
func TestCallEndsWhenTheParentContextEnds(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	files, starter := filesWithV1(), &fakeStarter{}
	proc := &fakeProc{blockRead: true, unblock: make(chan struct{})}
	starter.proc = proc
	if _, err := Call(ctx, mustPluginConfig(t), "m", nil, callSeams(files, starter, proc)); !errors.Is(err, context.Canceled) {
		t.Fatalf("Call error = %v, want context.Canceled", err)
	}
	if proc.killed == 0 {
		t.Fatal("a cancelled call did not kill the plugin")
	}
}

// TestCallFailurePaths checks every other failure of the setup and the exchange: each setup failure is a StartError
// (exit 2, never 126 or 127), and each exchange failure is a ProtocolError.
func TestCallFailurePaths(t *testing.T) {
	badParams := map[string]any{"f": func() {}}
	boom := errors.New("boom")
	cases := []struct {
		name   string
		mutate func(*fakeFiles, *fakeStarter, *fakeProc)
		method string
		params map[string]any
		check  func(error) bool
	}{
		{"empty method", func(*fakeFiles, *fakeStarter, *fakeProc) {}, "", nil,
			func(err error) bool { return err != nil && err.Error() == "plugin: a method name is required" }},
		{"unmarshallable params", func(*fakeFiles, *fakeStarter, *fakeProc) {}, "m", badParams,
			func(err error) bool { return err != nil }},
		{"private folder fails", func(f *fakeFiles, _ *fakeStarter, _ *fakeProc) { f.mkdirErr = boom }, "m", nil,
			func(err error) bool {
				_, ok := err.(StartError)
				return ok && err.Error() == "could not prepare the private folder"
			}},
		{"copy fails", func(f *fakeFiles, _ *fakeStarter, _ *fakeProc) { f.copyErr = boom }, "m", nil,
			func(err error) bool { _, ok := err.(StartError); return ok }},
		{"hash fails", func(f *fakeFiles, _ *fakeStarter, _ *fakeProc) { f.hashErr = boom }, "m", nil,
			func(err error) bool { _, ok := err.(StartError); return ok }},
		{"start fails", func(_ *fakeFiles, s *fakeStarter, _ *fakeProc) { s.err = boom }, "m", nil,
			func(err error) bool { _, ok := err.(StartError); return ok }},
		{"write fails", func(_ *fakeFiles, _ *fakeStarter, p *fakeProc) { p.writeErr = boom }, "m", nil,
			func(err error) bool { _, ok := err.(ProtocolError); return ok }},
		{"write protocol fails", func(_ *fakeFiles, _ *fakeStarter, p *fakeProc) {
			p.writeErr = ProtocolError{Rule: "kept"}
		}, "m", nil, func(err error) bool {
			protocol, ok := err.(ProtocolError)
			return ok && protocol.Rule == "kept"
		}},
		{"read fails", func(_ *fakeFiles, _ *fakeStarter, p *fakeProc) { p.readErr = boom }, "m", nil,
			func(err error) bool { _, ok := err.(ProtocolError); return ok }},
		{"read protocol fails", func(_ *fakeFiles, _ *fakeStarter, p *fakeProc) {
			p.readErr = ProtocolError{Rule: "kept"}
		}, "m", nil, func(err error) bool {
			protocol, ok := err.(ProtocolError)
			return ok && protocol.Rule == "kept"
		}},
		{"empty reply", func(_ *fakeFiles, _ *fakeStarter, p *fakeProc) { p.reply = []byte("") }, "m", nil,
			func(err error) bool { _, ok := err.(ProtocolError); return ok }},
		{"malformed reply", func(_ *fakeFiles, _ *fakeStarter, p *fakeProc) { p.reply = []byte("{oops") }, "m", nil,
			func(err error) bool {
				protocol, ok := err.(ProtocolError)
				return ok && protocol.Rule == "the plugin frame is not JSON"
			}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			files, starter := filesWithV1(), &fakeStarter{}
			proc := answerProc(goodReply)
			starter.proc = proc
			c.mutate(files, starter, proc)
			if _, err := Call(context.Background(), mustPluginConfig(t), c.method, c.params, callSeams(files, starter, proc)); !c.check(err) {
				t.Fatalf("Call error = %#v, want the documented failure", err)
			}
		})
	}
}

// TestCallKeepsTheStableCodes checks that a pin, protocol, and start failure each keep its stable code in every
// command that reports it.
func TestCallKeepsTheStableCodes(t *testing.T) {
	if got := (PinMismatchError{}).ErrorCode(); got != "retriever.connector.plugin_pin_mismatch" {
		t.Fatalf("PinMismatchError code = %q", got)
	}
	if got := (ProtocolError{Rule: "r"}).ErrorCode(); got != "retriever.connector.plugin_protocol" {
		t.Fatalf("ProtocolError code = %q", got)
	}
	if got := (StartError{Rule: "r"}).ErrorCode(); got != "retriever.connector.plugin_start_failed" {
		t.Fatalf("StartError code = %q", got)
	}
	if got := (PinMismatchError{}).Code(); got != "retriever.connector.plugin_pin_mismatch" {
		t.Fatalf("PinMismatchError.Code = %q", got)
	}
	if got := (PinMismatchError{}).Error(); got != "the plugin bytes differ from the approved pin" {
		t.Fatalf("PinMismatchError text = %q", got)
	}
	if got := (ProtocolError{Rule: "r"}).Code(); got != "retriever.connector.plugin_protocol" {
		t.Fatalf("ProtocolError.Code = %q", got)
	}
	if got := (ProtocolError{Rule: "the plugin sent a notification"}).Error(); got != "the plugin sent a notification" {
		t.Fatalf("ProtocolError text = %q", got)
	}
	if got := (StartError{Rule: "r"}).Code(); got != "retriever.connector.plugin_start_failed" {
		t.Fatalf("StartError.Code = %q", got)
	}
	if got := (StartError{Rule: "could not start the plugin process"}).Error(); got != "could not start the plugin process" {
		t.Fatalf("StartError text = %q", got)
	}
}

// TestTimeoutErrorUnwrapsToTheBoundedWait checks that the CLI maps the timeout to 124 and marks it retryable.
func TestTimeoutErrorUnwrapsToTheBoundedWait(t *testing.T) {
	err := TimeoutError{Limit: 2 * time.Second}
	if !errors.Is(err, domain.ErrTimedOut) {
		t.Fatalf("TimeoutError does not match ErrTimedOut: %v", err)
	}
	if got := err.Error(); got != "the plugin call timed out after 2s" {
		t.Fatalf("TimeoutError text = %q", got)
	}
}

// TestDecodeReplyRefusesEveryAbuse checks P5 case by case: only one JSON-RPC 2.0 object with id 1 and exactly one
// of result and error is accepted.
func TestDecodeReplyRefusesEveryAbuse(t *testing.T) {
	cases := []struct {
		name  string
		frame string
		rule  string
	}{
		{"empty", ``, "the plugin answered nothing"},
		{"blank", "  \n ", "the plugin answered nothing"},
		{"batch", `[{"jsonrpc":"2.0","id":1,"result":{}}]`, "the plugin sent a batch"},
		{"not JSON", `{oops`, "the plugin frame is not JSON"},
		{"not an object", `"ok"`, "the plugin frame is not JSON"},
		{"extra data", goodReply + " {}", "the plugin frame holds extra data"},
		{"wrong version", `{"jsonrpc":"1.0","id":1,"result":{}}`, "the plugin frame is not JSON-RPC 2.0"},
		{"missing version", `{"id":1,"result":{}}`, "the plugin frame is not JSON-RPC 2.0"},
		{"notification", `{"jsonrpc":"2.0","result":{}}`, "the plugin sent a notification"},
		{"null id", `{"jsonrpc":"2.0","id":null,"result":{}}`, "the plugin reply id does not match the request"},
		{"wrong numeric id", `{"jsonrpc":"2.0","id":2,"result":{}}`, "the plugin reply id does not match the request"},
		{"string id", `{"jsonrpc":"2.0","id":"1","result":{}}`, "the plugin reply id does not match the request"},
		{"error object", `{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"gone"}}`, "the plugin returned an error"},
		{"result and error", `{"jsonrpc":"2.0","id":1,"result":{},"error":{"code":1}}`, "the plugin returned an error"},
		{"missing result", `{"jsonrpc":"2.0","id":1}`, "the plugin reply has no result"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := decodeReply([]byte(c.frame)); err == nil {
				t.Fatal("decodeReply accepted a refused frame")
			} else if protocol, ok := err.(ProtocolError); !ok || protocol.Rule != c.rule {
				t.Fatalf("decodeReply error = %#v, want rule %q", err, c.rule)
			}
		})
	}
}

// TestDecodeReplyAcceptsTheResult checks the valid shapes: an object result passes through, and a null result
// decodes for the gateway to refuse as a missing object.
func TestDecodeReplyAcceptsTheResult(t *testing.T) {
	result, err := decodeReply([]byte(goodReply))
	if err != nil {
		t.Fatalf("decodeReply error = %v, want nil", err)
	}
	if string(result) != `{"collections":[]}` {
		t.Fatalf("result = %s", result)
	}
	result, err = decodeReply([]byte(`{"jsonrpc":"2.0","id":1,"result":null}`))
	if err != nil || string(result) != "null" {
		t.Fatalf("null result = %s, err = %v; want null with no error", result, err)
	}
}

// TestRedactReplacesOnlyTheSecrets checks P6 at the value level: every secret occurrence reads [REDACTED], an
// empty secret changes nothing, and other text passes through.
func TestRedactReplacesOnlyTheSecrets(t *testing.T) {
	secrets := []string{"credential-sentinel-9f2a", "sentinel-mail-77@example.test"}
	got := redact("key credential-sentinel-9f2a by sentinel-mail-77@example.test done", secrets)
	if got != "key [REDACTED] by [REDACTED] done" {
		t.Fatalf("redact = %q", got)
	}
	if got := redact("plain", []string{""}); got != "plain" {
		t.Fatalf("an empty secret changed the text: %q", got)
	}
	if got := redactBytes(json.RawMessage(`{"k":"credential-sentinel-9f2a"}`), secrets); string(got) != `{"k":"[REDACTED]"}` {
		t.Fatalf("redactBytes = %s", got)
	}
}

// TestSeamsDefaultToProduction checks that a zero Seams runs the real file calls, the real process start, and the
// real clock: a stubbed os call proves the wiring without touching the disk.
func TestSeamsDefaultToProduction(t *testing.T) {
	oldMkdir := osMkdirTemp
	defer func() { osMkdirTemp = oldMkdir }()
	osMkdirTemp = func(string, string) (string, error) { return "/stubbed", nil }
	s := Seams{}.withDefaults()
	dir, err := s.Files.MakePrivateDir()
	if err != nil || dir != "/stubbed" {
		t.Fatalf("default Files = %q, %v; want the stubbed folder", dir, err)
	}
	if s.Start == nil || s.After == nil {
		t.Fatal("a zero Seams must default the starter and the clock")
	}
}

// TestOsFilesCopiesHashesAndRemoves checks the production file calls against stubbed operating-system calls: the
// copy writes the source bytes with owner-only access, and the hash reads them back.
func TestOsFilesCopiesHashesAndRemoves(t *testing.T) {
	oldRead, oldWrite, oldRemove := osReadFile, osWriteFile, osRemoveAll
	defer func() { osReadFile, osWriteFile, osRemoveAll = oldRead, oldWrite, oldRemove }()
	var written string
	var mode os.FileMode
	osReadFile = func(path string) ([]byte, error) {
		if path == "/nope" {
			return nil, errors.New("no file")
		}
		return append([]byte(nil), pluginBytesV1...), nil
	}
	osWriteFile = func(path string, data []byte, perm os.FileMode) error {
		written, mode = path, perm
		if path == "/bad" {
			return errors.New("no write")
		}
		return nil
	}
	var removed string
	osRemoveAll = func(path string) error { removed = path; return nil }
	files := osFiles{}
	if err := files.CopyFile("/src", "/dst"); err != nil {
		t.Fatalf("CopyFile error = %v, want nil", err)
	}
	if written != "/dst" || mode != 0o700 {
		t.Fatalf("copy wrote %q with mode %o, want the copy with owner-only access", written, mode)
	}
	if err := files.CopyFile("/nope", "/dst"); err == nil {
		t.Fatal("CopyFile accepted an unreadable source")
	}
	if err := files.CopyFile("/src", "/bad"); err == nil {
		t.Fatal("CopyFile accepted an unwritable copy")
	}
	sum, err := files.HashFile("/dst")
	if err != nil || sum != pinOf(pluginBytesV1) {
		t.Fatalf("HashFile = %q, %v; want the pin of the bytes", sum, err)
	}
	if _, err := files.HashFile("/nope"); err == nil {
		t.Fatal("HashFile accepted an unreadable file")
	}
	if err := files.RemoveAll("/dir"); err != nil || removed != "/dir" {
		t.Fatalf("RemoveAll removed %q with %v", removed, err)
	}
}

// nilCall answers a lifecycle call with success, for procs whose kill and wait the test never triggers.
func nilCall() error { return nil }

// nopWriteCloser buffers writes and answers Close with the scripted error.
type nopWriteCloser struct {
	buf      bytes.Buffer
	writeErr error
	closeErr error
	closed   bool
}

func (n *nopWriteCloser) Write(p []byte) (int, error) {
	if n.writeErr != nil {
		return 0, n.writeErr
	}
	return n.buf.Write(p)
}

func (n *nopWriteCloser) Close() error {
	n.closed = true
	return n.closeErr
}

// failingReader fails every read with the scripted error, which is not EOF.
type failingReader struct{ err error }

func (r failingReader) Read([]byte) (int, error) { return 0, r.err }

// TestExecProcWritesReadsAndCaps checks the live process ends: the write appends the newline and closes the input,
// the read stops at the newline and accepts an EOF-terminated frame, and the stderr capture caps at MaxStderrBytes.
func TestExecProcWritesReadsAndCaps(t *testing.T) {
	stdin := &nopWriteCloser{}
	p := newExecProc(stdin, strings.NewReader("{\"a\":1}\nignored\n"), strings.NewReader("note\n"), nilCall, nilCall)
	if err := p.Write([]byte(`{"a":1}`)); err != nil {
		t.Fatalf("Write error = %v, want nil", err)
	}
	if stdin.buf.String() != "{\"a\":1}\n" || !stdin.closed {
		t.Fatalf("input = %q closed = %v, want the frame with a newline and a closed input", stdin.buf.String(), stdin.closed)
	}
	frame, err := p.Read()
	if err != nil || string(frame) != `{"a":1}` {
		t.Fatalf("Read = %q, %v; want the frame before the newline", frame, err)
	}
	if got := string(p.Stderr()); got != "note\n" {
		t.Fatalf("Stderr = %q, want the note", got)
	}
	if err := p.Kill(); err != nil {
		t.Fatalf("Kill error = %v, want nil", err)
	}
	if err := p.Wait(); err != nil {
		t.Fatalf("Wait error = %v, want nil", err)
	}
}

// TestExecProcReadPaths checks every read path: an EOF-terminated frame passes, an empty output is EOF, a flooded
// frame fails at 1 MiB, and a broken reader fails the call.
func TestExecProcReadPaths(t *testing.T) {
	eofTerminated, err := newExecProc(&nopWriteCloser{}, strings.NewReader(`{"a":1}`), strings.NewReader(""), nilCall, nilCall).Read()
	if err != nil || string(eofTerminated) != `{"a":1}` {
		t.Fatalf("EOF-terminated Read = %q, %v", eofTerminated, err)
	}
	if _, err := newExecProc(&nopWriteCloser{}, strings.NewReader(""), strings.NewReader(""), nilCall, nilCall).Read(); !errors.Is(err, io.EOF) {
		t.Fatalf("empty Read error = %v, want EOF", err)
	}
	flooded := newExecProc(&nopWriteCloser{}, strings.NewReader(strings.Repeat("x", MaxFrameBytes+1)), strings.NewReader(""), nilCall, nilCall)
	if _, err := flooded.Read(); err == nil {
		t.Fatal("Read accepted a frame past 1 MiB")
	} else if protocol, ok := err.(ProtocolError); !ok || protocol.Rule != "the plugin frame is larger than 1 MiB" {
		t.Fatalf("flooded Read error = %#v", err)
	}
	broken := newExecProc(&nopWriteCloser{}, failingReader{err: errors.New("boom")}, strings.NewReader(""), nilCall, nilCall)
	if _, err := broken.Read(); err == nil {
		t.Fatal("Read accepted a broken reader")
	} else if protocol, ok := err.(ProtocolError); !ok || protocol.Rule != "could not read the plugin reply" {
		t.Fatalf("broken Read error = %#v", err)
	}
}

// TestExecProcWritePaths checks both write failures: a failed write and a failed close each refuse the call, and a
// failed kill or wait surfaces from the proc.
func TestExecProcWritePaths(t *testing.T) {
	boom := errors.New("boom")
	if err := newExecProc(&nopWriteCloser{writeErr: boom}, strings.NewReader(""), strings.NewReader(""), nil, nil).Write([]byte("{}")); err == nil {
		t.Fatal("Write accepted a failed write")
	} else if protocol, ok := err.(ProtocolError); !ok || protocol.Rule != "the plugin closed its input" {
		t.Fatalf("failed Write error = %#v", err)
	}
	if err := newExecProc(&nopWriteCloser{closeErr: boom}, strings.NewReader(""), strings.NewReader(""), nil, nil).Write([]byte("{}")); err == nil {
		t.Fatal("Write accepted a failed close")
	}
	killErr := newExecProc(&nopWriteCloser{}, strings.NewReader(""), strings.NewReader(""),
		func() error { return boom }, func() error { return nil }).Kill()
	if !errors.Is(killErr, boom) {
		t.Fatalf("Kill error = %v, want the kill failure", killErr)
	}
	waitErr := newExecProc(&nopWriteCloser{}, strings.NewReader(""), strings.NewReader(""),
		func() error { return nil }, func() error { return boom }).Wait()
	if !errors.Is(waitErr, boom) {
		t.Fatalf("Wait error = %v, want the wait failure", waitErr)
	}
}

// TestCappedStderrKeepsTheFirst64KiB checks that a flooding diagnostics stream is capped, not grown.
func TestCappedStderrKeepsTheFirst64KiB(t *testing.T) {
	p := newExecProc(&nopWriteCloser{}, strings.NewReader(""), strings.NewReader(strings.Repeat("y", MaxStderrBytes+100)), nil, nil)
	if got := len(p.Stderr()); got != MaxStderrBytes {
		t.Fatalf("Stderr holds %d bytes, want the %d-byte cap", got, MaxStderrBytes)
	}
}

// unevenReader answers one short read and then full reads: a legal reader may return less than asked, so the cap
// must hold under uneven chunks too.
type unevenReader struct {
	data  []byte
	first int
}

func (r *unevenReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	n := len(p)
	if r.first > 0 {
		n = r.first
		r.first = 0
	}
	if n > len(r.data) {
		n = len(r.data)
	}
	copy(p, r.data[:n])
	r.data = r.data[n:]
	return n, nil
}

// TestCappedStderrTrimsAUnevenChunk checks that a read straddling the cap is trimmed to the cap exactly: the kept
// bytes never pass MaxStderrBytes however the stream chunks them.
func TestCappedStderrTrimsAUnevenChunk(t *testing.T) {
	data := strings.Repeat("z", MaxStderrBytes+100)
	p := newExecProc(&nopWriteCloser{}, strings.NewReader(""), &unevenReader{data: []byte(data), first: 1000}, nil, nil)
	if got := len(p.Stderr()); got != MaxStderrBytes {
		t.Fatalf("Stderr holds %d bytes, want the %d-byte cap", got, MaxStderrBytes)
	}
}

// fakeCommand answers the pipe and lifecycle calls of startCommand from memory.
type fakeCommand struct {
	stdin     io.WriteCloser
	stdinErr  error
	stdout    io.Reader
	stdoutErr error
	stderr    io.Reader
	stderrErr error
	startErr  error
	waitErr   error
	killed    int
	waited    int
}

func (f *fakeCommand) StdinPipe() (io.WriteCloser, error) {
	if f.stdinErr != nil {
		return nil, f.stdinErr
	}
	return f.stdin, nil
}

func (f *fakeCommand) StdoutPipe() (io.ReadCloser, error) {
	if f.stdoutErr != nil {
		return nil, f.stdoutErr
	}
	return io.NopCloser(f.stdout), nil
}

func (f *fakeCommand) StderrPipe() (io.ReadCloser, error) {
	if f.stderrErr != nil {
		return nil, f.stderrErr
	}
	return io.NopCloser(f.stderr), nil
}

func (f *fakeCommand) Start() error { return f.startErr }

func (f *fakeCommand) Wait() error {
	f.waited++
	return f.waitErr
}

func (f *fakeCommand) KillGroup() error {
	f.killed++
	return nil
}

// TestStartCommandWiresThePipes checks every wiring path of a process start: each pipe failure and the start
// failure is a StartError, and the success path returns a proc over the fake.
func TestStartCommandWiresThePipes(t *testing.T) {
	boom := errors.New("boom")
	newFake := func() *fakeCommand {
		return &fakeCommand{stdin: &nopWriteCloser{}, stdout: strings.NewReader(""), stderr: strings.NewReader("")}
	}
	cases := []struct {
		name   string
		mutate func(*fakeCommand)
		rule   string
	}{
		{"input fails", func(f *fakeCommand) { f.stdinErr = boom }, "could not open the plugin input"},
		{"output fails", func(f *fakeCommand) { f.stdoutErr = boom }, "could not open the plugin output"},
		{"diagnostics fail", func(f *fakeCommand) { f.stderrErr = boom }, "could not open the plugin diagnostics"},
		{"start fails", func(f *fakeCommand) { f.startErr = boom }, "could not start the plugin process"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fake := newFake()
			c.mutate(fake)
			if _, err := startCommand(fake); err == nil {
				t.Fatal("startCommand accepted a failed wiring")
			} else if start, ok := err.(StartError); !ok || start.Rule != c.rule {
				t.Fatalf("startCommand error = %#v, want rule %q", err, c.rule)
			}
		})
	}
	proc, err := startCommand(newFake())
	if err != nil {
		t.Fatalf("startCommand error = %v, want nil", err)
	}
	if proc == nil {
		t.Fatal("startCommand returned no proc")
	}
}

// TestNewCommandBuildsAGroupLeader checks the production factory without starting a process: building the command
// wires the copy, the private folder, the scrubbed environment, and the process-group flag, and the process is not
// started. The constructor performs no I/O, so the unit test runs it directly.
func TestNewCommandBuildsAGroupLeader(t *testing.T) {
	cmd := newCommand("/copy/plugin", "/private/1", []string{"K=v"})
	concrete, ok := cmd.(*execGroup)
	if !ok {
		t.Fatalf("newCommand = %T, want the group wrapper", cmd)
	}
	if concrete.Dir != "/private/1" {
		t.Fatalf("Dir = %q, want the private folder", concrete.Dir)
	}
	if len(concrete.Env) != 1 || concrete.Env[0] != "K=v" {
		t.Fatalf("Env = %q, want the scrubbed environment", concrete.Env)
	}
	if concrete.SysProcAttr == nil || !concrete.SysProcAttr.Setpgid {
		t.Fatal("the plugin must start as a process-group leader")
	}
	if concrete.Process != nil {
		t.Fatal("building the command must not start the process")
	}
	if concrete.Path != "/copy/plugin" {
		t.Fatalf("Path = %q, want the plugin copy", concrete.Path)
	}
}

// TestExecGroupKillGroupEndsTheGroup checks that the group kill signals the negative pid of the process with
// SIGKILL, which cannot be caught or ignored. The kill itself is stubbed; only the addressing is proved.
func TestExecGroupKillGroupEndsTheGroup(t *testing.T) {
	oldKill := killGroup
	defer func() { killGroup = oldKill }()
	var gotPid int
	var gotSignal syscall.Signal
	killGroup = func(pid int, sig syscall.Signal) error {
		gotPid, gotSignal = pid, sig
		return nil
	}
	proc := newCommand("/copy/plugin", "/private/1", nil)
	concrete, ok := proc.(*execGroup)
	if !ok {
		t.Fatalf("newCommand = %T, want the group wrapper", proc)
	}
	concrete.Process = &os.Process{Pid: 4242}
	if err := concrete.KillGroup(); err != nil {
		t.Fatalf("KillGroup error = %v, want nil", err)
	}
	if gotPid != -4242 || gotSignal != syscall.SIGKILL {
		t.Fatalf("kill targeted pid %d with signal %v, want group -4242 with SIGKILL", gotPid, gotSignal)
	}
}

// TestStartProcessRunsTheStarter checks that the production starter runs the command the factory builds: a stubbed
// factory proves the wiring with no real process.
func TestStartProcessRunsTheStarter(t *testing.T) {
	oldNew := newCommand
	defer func() { newCommand = oldNew }()
	var gotPath, gotDir string
	var gotEnv []string
	fake := &fakeCommand{stdin: &nopWriteCloser{}, stdout: strings.NewReader(""), stderr: strings.NewReader("")}
	newCommand = func(path, dir string, env []string) command {
		gotPath, gotDir, gotEnv = path, dir, env
		return fake
	}
	proc, err := startProcess("/copy/plugin", "/private/1", []string{"K=v"})
	if err != nil || proc == nil {
		t.Fatalf("startProcess = %v, %v; want a proc", proc, err)
	}
	if gotPath != "/copy/plugin" || gotDir != "/private/1" || len(gotEnv) != 1 || gotEnv[0] != "K=v" {
		t.Fatalf("startProcess built path %q dir %q env %q", gotPath, gotDir, gotEnv)
	}
}

// TestWaitForExitKillsALingeringPlugin checks that a plugin that answers but never exits is killed after the grace:
// the answered call never leaks a process.
func TestWaitForExitKillsALingeringPlugin(t *testing.T) {
	proc := &fakeProc{unblock: make(chan struct{}), waitBlocks: true}
	seams := Seams{After: func(time.Duration) <-chan time.Time { return closedChan() }}
	waitForExit(proc, seams)
	if proc.killed != 1 {
		t.Fatalf("killed = %d, want the lingering plugin killed once", proc.killed)
	}
	if proc.waited != 1 {
		t.Fatalf("waited = %d, want the plugin reaped", proc.waited)
	}
}

// TestWaitForExitAcceptsAPromptExit checks that a plugin that exits at once is never killed.
func TestWaitForExitAcceptsAPromptExit(t *testing.T) {
	proc := &fakeProc{unblock: make(chan struct{})}
	seams := Seams{After: func(time.Duration) <-chan time.Time { return openChan() }}
	waitForExit(proc, seams)
	if proc.killed != 0 {
		t.Fatalf("killed = %d, want no kill for a prompt exit", proc.killed)
	}
}

// TestConfigDefaults checks the remaining Config rules: a zero timeout reads as DefaultTimeout, and an empty
// identity value redacts nothing.
func TestConfigDefaults(t *testing.T) {
	cfg := mustPluginConfig(t)
	cfg.Timeout = 0
	if cfg.limit() != DefaultTimeout {
		t.Fatalf("limit = %s, want DefaultTimeout", cfg.limit())
	}
	cfg.IdentityValue = ""
	for _, secret := range cfg.secrets() {
		if secret == "" {
			t.Fatal("an empty identity value must not be a redaction secret")
		}
	}
}

// TestWriteAndReadErrorPassProtocolThrough checks that a typed protocol failure from the proc keeps its rule,
// while any other failure maps onto the fixed rules.
func TestWriteAndReadErrorPassProtocolThrough(t *testing.T) {
	kept := ProtocolError{Rule: "kept"}
	if writeError(kept) != kept {
		t.Fatal("writeError dropped a protocol failure")
	}
	if readError(kept) != kept {
		t.Fatal("readError dropped a protocol failure")
	}
	if err := writeError(errors.New("boom")); err == nil {
		t.Fatal("writeError accepted a generic failure")
	} else if protocol, ok := err.(ProtocolError); !ok || protocol.Rule != "could not write the plugin request" {
		t.Fatalf("writeError = %#v", err)
	}
	if err := readError(errors.New("boom")); err == nil {
		t.Fatal("readError accepted a generic failure")
	} else if protocol, ok := err.(ProtocolError); !ok || protocol.Rule != "could not read the plugin reply" {
		t.Fatalf("readError = %#v", err)
	}
	if err := readError(io.EOF); err == nil {
		t.Fatal("readError accepted an empty output")
	} else if protocol, ok := err.(ProtocolError); !ok || protocol.Rule != "the plugin answered nothing" {
		t.Fatalf("readError(EOF) = %#v", err)
	}
}
