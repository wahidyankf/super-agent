// Package plugin starts one pinned out-of-process connector plugin and speaks JSON-RPC 2.0 with it over its
// standard input and output (ADR 001, 003-plugin-and-supply-chain-threats).
//
// A plugin is the only path on which code runs (boundary B4), so the host holds it tightly:
//
//   - The catalog names the plugin by an absolute path and a SHA-256 pin. Call copies the file into a fresh private
//     folder, hashes the copy, and runs the copy only when the hash equals the pin, so a swap of the original after
//     the check changes nothing (P1). A mismatch starts no process and fails with PinMismatchError.
//   - A plugin is opt-in per source: only a catalog source with a plugin block routes to the Gateway, and the spec
//     schema has no plugin field, so a spec can never enable one (P2).
//   - The child gets an environment of exactly its own credential variable and, when the source declares one, its
//     identity variable, and it starts in an empty private folder as its working directory (P3).
//   - One process serves one call of one source. Each call has a time limit, each frame is at most MaxFrameBytes, and
//     on a timeout the host kills the process group and fails with TimeoutError, which the CLI reports as exit 124
//     (P4). A failed start fails with StartError and exit 2, never 126 or 127.
//   - The host writes one request with id 1 and accepts only the reply with that id: a notification, a wrong id, a
//     batch, a malformed frame, and a JSON-RPC error object each fail with ProtocolError (P5).
//   - The host redacts the plugin's credential values from the reply and from the captured standard error, and it
//     caps standard error at MaxStderrBytes (P6).
//
// The host does not sandbox a plugin: it may use the network and the user's files. The pin and the opt-in are the
// controls. Review a plugin as you review any program you run.
//
// The file system, the process table, and the clock reach the code through Seams: a unit test supplies fakes and
// starts no process, and only the integration tests run a real plugin. Production is the nil Seams: the real file
// calls, the real process start, and time.After.
package plugin

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

const (
	// MaxFrameBytes caps one JSON-RPC frame the host reads: a frame past one mebibyte fails the call, so a
	// flooding plugin cannot grow the host's memory (P4).
	MaxFrameBytes = 1 << 20
	// MaxStderrBytes caps the standard error the host keeps: the first 64 KiB are kept and redacted, the rest is
	// dropped, so a noisy plugin cannot grow the host's memory (P6).
	MaxStderrBytes = 64 << 10
	// DefaultTimeout bounds one plugin call when the configuration names no limit (P4).
	DefaultTimeout = 30 * time.Second
	// exitGrace bounds the wait for a plugin to exit after it answered: the host closes the input, waits this
	// long, then kills the process group, so an answered call never leaks a process.
	exitGrace = 5 * time.Second
	// redacted is the only text a secret is ever replaced with.
	redacted = "[REDACTED]"
)

const (
	// PinMismatchCode is the stable code of a plugin whose bytes differ from the approved pin (P1). The CLI keeps
	// this code in every command that reports it.
	PinMismatchCode = "retriever.connector.plugin_pin_mismatch"
	// ProtocolCode is the stable code of a plugin reply the host refuses: a wrong id, a notification, a malformed
	// frame, an oversize frame, or a JSON-RPC error object (P5).
	ProtocolCode = "retriever.connector.plugin_protocol"
	// StartFailedCode is the stable code of a plugin the host could not prepare or start. A failed start exits 2,
	// never 126 or 127.
	StartFailedCode = "retriever.connector.plugin_start_failed"
)

// PinMismatchError reports a plugin whose bytes differ from the approved pin. The host started no process. Code
// returns PinMismatchCode, the stable code of the failure.
type PinMismatchError struct{}

// Error returns the rule that broke, in fixed words.
func (PinMismatchError) Error() string { return "the plugin bytes differ from the approved pin" }

// Code returns PinMismatchCode.
func (PinMismatchError) Code() string { return PinMismatchCode }

// ErrorCode returns PinMismatchCode too, so the CLI reports the pin failure with its own stable code, whatever
// command runs.
func (PinMismatchError) ErrorCode() string { return PinMismatchCode }

// ProtocolError reports a plugin reply the host refuses. Rule states the broken rule in fixed words and never a
// value, because a frame may echo a secret. ErrorCode returns ProtocolCode.
type ProtocolError struct {
	Rule string
}

// Error returns the rule that broke.
func (e ProtocolError) Error() string { return e.Rule }

// Code returns ProtocolCode.
func (ProtocolError) Code() string { return ProtocolCode }

// ErrorCode returns ProtocolCode too, so the CLI reports the protocol failure with its own stable code, whatever
// command runs.
func (ProtocolError) ErrorCode() string { return ProtocolCode }

// StartError reports a plugin the host could not prepare or start: the private folder, the copy, or the process.
// Rule states the broken rule in fixed words and never a path. ErrorCode returns StartFailedCode, and the CLI exits
// 2, never 126 or 127: the child's own statuses never surface.
type StartError struct {
	Rule string
}

// Error returns the rule that broke.
func (e StartError) Error() string { return e.Rule }

// Code returns StartFailedCode.
func (StartError) Code() string { return StartFailedCode }

// ErrorCode returns StartFailedCode too, so the CLI reports the start failure with its own stable code, whatever
// command runs.
func (StartError) ErrorCode() string { return StartFailedCode }

// TimeoutError reports a plugin call that got no reply in time: the host killed the process group. Limit is the
// bound the call ran under. Unwrap returns domain.ErrTimedOut, so the CLI exits 124 and marks the failure
// retryable, like any other bounded wait.
type TimeoutError struct {
	Limit time.Duration
}

// Error returns the bound the call ran under, in fixed words.
func (e TimeoutError) Error() string {
	return fmt.Sprintf("the plugin call timed out after %s", e.Limit)
}

// Unwrap returns domain.ErrTimedOut, so errors.Is finds the bounded wait under it.
func (e TimeoutError) Unwrap() error { return domain.ErrTimedOut }

// Config is one pinned plugin invocation: the plugin the catalog names, the secret values the host resolved for
// its source, and the bound of the call. CredentialValue is the secret the credential variable holds, and
// IdentityValue the address the identity variable holds, when the source declares one. A zero Timeout reads as
// DefaultTimeout. The values never reach an error or a log: replies and standard error are redacted with them.
type Config struct {
	Plugin          domain.PluginRef
	CredentialEnv   string
	CredentialValue string
	IdentityEnv     string
	IdentityValue   string
	Timeout         time.Duration
}

// environ returns the whole environment of the child: exactly its own credential variable and, when the source
// declares one, its identity variable (P3). The process environment is never inherited, so a sibling credential
// cannot leak in.
func (c Config) environ() []string {
	env := []string{c.CredentialEnv + "=" + c.CredentialValue}
	if c.IdentityEnv != "" {
		env = append(env, c.IdentityEnv+"="+c.IdentityValue)
	}
	return env
}

// secrets returns the values the host redacts from the reply and the captured standard error: the credential and,
// when set, the identity. An empty value redacts nothing.
func (c Config) secrets() []string {
	secrets := []string{c.CredentialValue}
	if c.IdentityValue != "" {
		secrets = append(secrets, c.IdentityValue)
	}
	return secrets
}

// limit returns the bound of the call: the configured timeout, or DefaultTimeout when none is set.
func (c Config) limit() time.Duration {
	if c.Timeout <= 0 {
		return DefaultTimeout
	}
	return c.Timeout
}

// Request is one JSON-RPC 2.0 call the host sends: Method with Params as its params. The host always sends id 1:
// one process serves one call, so one id is enough.
type Request struct {
	Method string
	Params map[string]any
}

// Reply is what a plugin call returned: the decoded result frame and the captured standard error, both redacted.
// Stderr is capped at MaxStderrBytes.
type Reply struct {
	Result json.RawMessage
	Stderr string
}

// Files are the file calls a Call makes: a fresh private folder per call, the copy of the plugin file into it, the
// hash of the copy, and the removal of the folder. The production calls read and write the real disk; a unit test
// passes an in-memory map, so the pin check runs with no real file.
type Files interface {
	// MakePrivateDir returns a fresh empty folder only this call uses. The caller removes it.
	MakePrivateDir() (string, error)
	// CopyFile copies src to dst.
	CopyFile(src, dst string) error
	// HashFile returns the SHA-256 of the file at path, in lower-case hexadecimal.
	HashFile(path string) (string, error)
	// RemoveAll removes the private folder and everything in it.
	RemoveAll(path string) error
}

// Starter starts the plugin copy at path, with dir as its working directory and env as its whole environment, and
// returns the running process. A nil Starter reads as the real process start.
type Starter func(path, dir string, env []string) (Proc, error)

// Proc is one running plugin process. Write sends the request frame and closes the standard input. Read returns the
// reply frame. Stderr returns the captured standard error. Kill ends the process group, and Wait reaps the process.
// The fake answers from memory, so a unit test drives every path with no real process.
type Proc interface {
	Write(frame []byte) error
	Read() ([]byte, error)
	Stderr() []byte
	Kill() error
	Wait() error
}

// Seams are the file, process, and clock calls a Call makes. The zero value is production: the real file calls,
// the real process start, and time.After. A unit test passes fakes, so no real file, process, or wait is touched.
//
// NewCommand builds the command that a start runs when Start is nil. The pipes, the start, and the group kill of that
// command go through the command it returns, so a unit test measures every wiring path with a fake command and no
// real process. A nil NewCommand reads as the production command.
type Seams struct {
	Files      Files
	Start      Starter
	NewCommand func(path, dir string, env []string) Command
	After      func(d time.Duration) <-chan time.Time
}

// withDefaults returns the seams with production in every unset slot.
func (s Seams) withDefaults() Seams {
	if s.Files == nil {
		s.Files = osFiles{}
	}
	if s.Start == nil {
		s.Start = func(path, dir string, env []string) (Proc, error) {
			if s.NewCommand == nil {
				return startProcess(path, dir, env)
			}
			return startCommand(s.NewCommand(path, dir, env))
		}
	}
	if s.After == nil {
		s.After = time.After
	}
	return s
}

// osCalls are the operating-system calls the production file calls make. Each is a direct alias, so a unit test
// stubs it with a fake and the logic above runs with no real file.
var (
	osMkdirTemp = os.MkdirTemp
	osReadFile  = os.ReadFile
	osWriteFile = os.WriteFile
	osRemoveAll = os.RemoveAll
)

// osFiles implements Files over the real disk, through osCalls. Only these methods touch real files, and a unit
// test never calls them: it stubs the calls or passes its own Files.
type osFiles struct{}

// MakePrivateDir returns a fresh folder with no group or other access, only for this call.
func (osFiles) MakePrivateDir() (string, error) { return osMkdirTemp("", "retriever-plugin-") }

// CopyFile copies the plugin file to its private copy.
func (osFiles) CopyFile(src, dst string) error {
	data, err := osReadFile(src)
	if err != nil {
		return err
	}
	return osWriteFile(dst, data, 0o700)
}

// HashFile returns the SHA-256 of the file at path, in lower-case hexadecimal.
func (osFiles) HashFile(path string) (string, error) {
	data, err := osReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// RemoveAll removes the private folder and everything in it.
func (osFiles) RemoveAll(path string) error { return osRemoveAll(path) }

// command is the subset of exec.Cmd the host drives. *exec.Cmd satisfies it through the execGroup wrapper, which
// adds the process-group kill; a unit test passes a fake, so no real process starts.
type command interface {
	StdinPipe() (io.WriteCloser, error)
	StdoutPipe() (io.ReadCloser, error)
	StderrPipe() (io.ReadCloser, error)
	Start() error
	Wait() error
	KillGroup() error
}

// Command is the process command a Seams.NewCommand factory returns: the subset of exec.Cmd the host drives, with the
// group kill. Production builds the real one, and a unit test returns a fake.
type Command = command

// execGroup is an exec.Cmd that kills its process group. The host starts every plugin as a group leader, so the
// kill ends the plugin and the work it spawned (P4).
type execGroup struct {
	*exec.Cmd
}

// killGroup ends one process group. It is a variable so a unit test stubs it with a recorder; production kills the
// real group.
var killGroup = syscall.Kill

// KillGroup ends the process group of the command with SIGKILL, which cannot be caught or ignored.
func (e *execGroup) KillGroup() error { return killGroup(-e.Process.Pid, syscall.SIGKILL) }

// newCommand builds the process for the plugin copy at path, in dir, with env as its whole environment. It is a
// variable so a unit test stubs it with a fake command; production runs the only os/exec call of the repository.
// Building the command starts no process: Start runs only in the integration tests.
var newCommand = func(path, dir string, env []string) command {
	cmd := exec.Command(path)
	cmd.Dir, cmd.Env, cmd.SysProcAttr = dir, env, &syscall.SysProcAttr{Setpgid: true}
	return &execGroup{Cmd: cmd}
}

// startProcess starts the plugin copy at path, with dir as its working directory and env as its whole
// environment. The copy runs as a process-group leader in the empty private folder (P3).
func startProcess(path, dir string, env []string) (Proc, error) {
	return startCommand(newCommand(path, dir, env))
}

// startCommand wires the pipes of cmd and starts it. A unit test passes a fake command, so every failure path is
// measured with no real process.
func startCommand(cmd command) (Proc, error) {
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, StartError{Rule: "could not open the plugin input"}
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, StartError{Rule: "could not open the plugin output"}
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, StartError{Rule: "could not open the plugin diagnostics"}
	}
	proc := newExecProc(stdin, stdout, stderr, cmd.KillGroup, cmd.Wait)
	if err := cmd.Start(); err != nil {
		return nil, StartError{Rule: "could not start the plugin process"}
	}
	return proc, nil
}

// newExecProc returns the Proc over the live ends of one process: the input writer, the output reader, the
// diagnostics reader, and the kill and wait calls. It starts the background copy of the standard error at once, so
// a plugin that floods it never blocks the reply.
func newExecProc(stdin io.WriteCloser, stdout io.Reader, stderrPipe io.Reader, kill, wait func() error) *execProc {
	p := &execProc{
		stdin:  stdin,
		stdout: bufio.NewReader(stdout),
		stderr: newCappedStderr(),
		kill:   kill,
		wait:   wait,
	}
	p.stderr.capture(stderrPipe)
	return p
}

// execProc is the Proc over a live plugin process.
type execProc struct {
	stdin  io.WriteCloser
	stdout *bufio.Reader
	stderr *cappedStderr
	kill   func() error
	wait   func() error
}

// Write sends the request frame with its newline and closes the standard input, so a plugin that reads to the end
// sees the whole request and nothing after it.
func (p *execProc) Write(frame []byte) error {
	withNewline := make([]byte, 0, len(frame)+1)
	withNewline = append(withNewline, frame...)
	withNewline = append(withNewline, '\n')
	if _, err := p.stdin.Write(withNewline); err != nil {
		return ProtocolError{Rule: "the plugin closed its input"}
	}
	if err := p.stdin.Close(); err != nil {
		return ProtocolError{Rule: "the plugin closed its input"}
	}
	return nil
}

// Read returns the reply frame: the bytes up to the first newline, or the bytes up to the end of the output when
// the plugin exits without one. A frame past MaxFrameBytes fails the call, so a flooding plugin is stopped (P4).
func (p *execProc) Read() ([]byte, error) {
	frame := make([]byte, 0, 4096)
	for {
		b, err := p.stdout.ReadByte()
		if err != nil {
			if len(frame) > 0 && errors.Is(err, io.EOF) {
				return frame, nil
			}
			if errors.Is(err, io.EOF) {
				return nil, io.EOF
			}
			return nil, ProtocolError{Rule: "could not read the plugin reply"}
		}
		if b == '\n' {
			return frame, nil
		}
		frame = append(frame, b)
		if len(frame) > MaxFrameBytes {
			return nil, ProtocolError{Rule: "the plugin frame is larger than 1 MiB"}
		}
	}
}

// Stderr returns the captured standard error. Call it after the process ended: the background copy is done then,
// because the dead process holds no pipe end open.
func (p *execProc) Stderr() []byte { return p.stderr.bytes() }

// Kill ends the process group.
func (p *execProc) Kill() error { return p.kill() }

// Wait reaps the process.
func (p *execProc) Wait() error { return p.wait() }

// cappedStderr keeps the first MaxStderrBytes of the standard error of one process. A background copy fills it
// while the call runs, so a plugin that floods its diagnostics never blocks the reply (P4).
type cappedStderr struct {
	mu  sync.Mutex
	buf bytes.Buffer
	wg  sync.WaitGroup
}

// newCappedStderr returns the empty capture.
func newCappedStderr() *cappedStderr { return &cappedStderr{} }

// capture copies r in the background until it ends, keeping only the first MaxStderrBytes.
func (c *cappedStderr) capture(r io.Reader) {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		chunk := make([]byte, 4096)
		for {
			n, err := r.Read(chunk)
			if n > 0 {
				c.mu.Lock()
				if remaining := MaxStderrBytes - c.buf.Len(); remaining > 0 {
					if n > remaining {
						n = remaining
					}
					c.buf.Write(chunk[:n])
				}
				c.mu.Unlock()
			}
			if err != nil {
				return
			}
		}
	}()
}

// bytes returns the kept bytes, after the copy is done.
func (c *cappedStderr) bytes() []byte {
	c.wg.Wait()
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]byte(nil), c.buf.Bytes()...)
}

// Call runs one JSON-RPC 2.0 call against the pinned plugin: it copies the file into a fresh private folder, hashes
// the copy, and runs the copy only when the hash equals the pin; then it sends one request and reads one reply
// under the time limit. The private folder is removed before Call returns, on every path.
//
// A pin mismatch starts no process and fails with PinMismatchError (P1). A timeout kills the process group and fails
// with TimeoutError (P4). A refused reply fails with ProtocolError (P5). A failure to prepare or start the process
// fails with StartError and exits 2, never 126 or 127. The parent context ending fails with its own error. The
// result and the captured standard error are redacted with the secret values (P6).
func Call(ctx context.Context, cfg Config, method string, params map[string]any, s Seams) (Reply, error) {
	s = s.withDefaults()
	frame, err := encodeRequest(method, params)
	if err != nil {
		return Reply{}, err
	}
	dir, err := s.Files.MakePrivateDir()
	if err != nil {
		return Reply{}, StartError{Rule: "could not prepare the private folder"}
	}
	defer func() { _ = s.Files.RemoveAll(dir) }()
	copyPath := filepath.Join(dir, "plugin")
	if err := s.Files.CopyFile(cfg.Plugin.Path(), copyPath); err != nil {
		return Reply{}, StartError{Rule: "could not copy the plugin file"}
	}
	// The hash is taken over the copy, so a swap of the original after the check changes nothing (P1).
	sum, err := s.Files.HashFile(copyPath)
	if err != nil {
		return Reply{}, StartError{Rule: "could not hash the plugin copy"}
	}
	if sum != cfg.Plugin.SHA256() {
		return Reply{}, PinMismatchError{}
	}
	proc, err := s.Start(copyPath, dir, cfg.environ())
	if err != nil {
		return Reply{}, StartError{Rule: "could not start the plugin process"}
	}
	type outcome struct {
		reply []byte
		err   error
	}
	done := make(chan outcome, 1)
	go func() {
		if err := proc.Write(frame); err != nil {
			done <- outcome{err: writeError(err)}
			return
		}
		reply, err := proc.Read()
		if err != nil {
			done <- outcome{err: readError(err)}
			return
		}
		done <- outcome{reply: reply}
	}()
	timer := s.After(cfg.limit())
	select {
	case out := <-done:
		waitForExit(proc, s)
		stderr := redact(string(proc.Stderr()), cfg.secrets())
		if out.err != nil {
			return Reply{Stderr: stderr}, out.err
		}
		result, err := decodeReply(out.reply)
		if err != nil {
			return Reply{Stderr: stderr}, err
		}
		return Reply{Result: redactBytes(result, cfg.secrets()), Stderr: stderr}, nil
	case <-timer:
		_ = proc.Kill()
		_ = proc.Wait()
		stderr := redact(string(proc.Stderr()), cfg.secrets())
		return Reply{Stderr: stderr}, TimeoutError{Limit: cfg.limit()}
	case <-ctx.Done():
		_ = proc.Kill()
		_ = proc.Wait()
		stderr := redact(string(proc.Stderr()), cfg.secrets())
		return Reply{Stderr: stderr}, ctx.Err()
	}
}

// waitForExit waits for the plugin to exit after it answered, closing the input first through the finished Write.
// The wait is bounded by exitGrace: a plugin that lingers is killed, so an answered call never leaks a process.
func waitForExit(proc Proc, s Seams) {
	exited := make(chan error, 1)
	go func() { exited <- proc.Wait() }()
	select {
	case <-exited:
	case <-s.After(exitGrace):
		_ = proc.Kill()
		<-exited
	}
}

// writeError maps a failed request write onto a protocol failure: the process started, so a closed input means the
// plugin went away instead of answering.
func writeError(err error) error {
	var protocol ProtocolError
	if errors.As(err, &protocol) {
		return err
	}
	return ProtocolError{Rule: "could not write the plugin request"}
}

// readError maps a failed reply read onto a protocol failure: an empty output means the plugin answered nothing.
func readError(err error) error {
	var protocol ProtocolError
	if errors.As(err, &protocol) {
		return err
	}
	if errors.Is(err, io.EOF) {
		return ProtocolError{Rule: "the plugin answered nothing"}
	}
	return ProtocolError{Rule: "could not read the plugin reply"}
}

// encodeRequest returns the JSON-RPC 2.0 request frame for one call, always with id 1: one process serves one call.
func encodeRequest(method string, params map[string]any) ([]byte, error) {
	if method == "" {
		return nil, errors.New("plugin: a method name is required")
	}
	if params == nil {
		params = map[string]any{}
	}
	return json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
}

// rpcWire is the shape Call accepts: a single JSON-RPC 2.0 reply object. The id stays raw so the check is exact,
// and the result stays raw so the caller maps it.
type rpcWire struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   json.RawMessage `json:"error"`
}

// decodeReply checks one reply frame and returns its result: exactly one JSON-RPC 2.0 object with the id the host
// sent and exactly one of result and error. Anything else is a protocol failure (P5): a batch, a notification, a
// wrong id, a malformed frame, extra data, a missing result, and a JSON-RPC error object each refuse the call.
func decodeReply(frame []byte) (json.RawMessage, error) {
	trimmed := bytes.TrimSpace(frame)
	if len(trimmed) == 0 {
		return nil, ProtocolError{Rule: "the plugin answered nothing"}
	}
	if trimmed[0] == '[' {
		return nil, ProtocolError{Rule: "the plugin sent a batch"}
	}
	var wire rpcWire
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	if err := decoder.Decode(&wire); err != nil {
		return nil, ProtocolError{Rule: "the plugin frame is not JSON"}
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, ProtocolError{Rule: "the plugin frame holds extra data"}
	}
	if wire.JSONRPC != "2.0" {
		return nil, ProtocolError{Rule: "the plugin frame is not JSON-RPC 2.0"}
	}
	if wire.ID == nil {
		return nil, ProtocolError{Rule: "the plugin sent a notification"}
	}
	if string(bytes.TrimSpace(wire.ID)) != "1" {
		return nil, ProtocolError{Rule: "the plugin reply id does not match the request"}
	}
	if wire.Error != nil {
		return nil, ProtocolError{Rule: "the plugin returned an error"}
	}
	if wire.Result == nil {
		return nil, ProtocolError{Rule: "the plugin reply has no result"}
	}
	return wire.Result, nil
}

// redact replaces every occurrence of each non-empty secret in text with [REDACTED] (P6). An empty secret changes
// nothing, so an unset identity cannot blank the text.
func redact(text string, secrets []string) string {
	for _, secret := range secrets {
		if secret == "" {
			continue
		}
		text = strings.ReplaceAll(text, secret, redacted)
	}
	return text
}

// redactBytes redacts the secrets in a JSON result frame. The frame stays valid JSON: only the secret bytes change.
func redactBytes(frame json.RawMessage, secrets []string) json.RawMessage {
	return json.RawMessage(redact(string(frame), secrets))
}

// sortedKeys returns the keys of fields in order, so a mapped record reads the same way on every run.
func sortedKeys(fields map[string]any) []string {
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
