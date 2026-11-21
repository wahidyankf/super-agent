package unit_test

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	plugin "github.com/wahidyankf/super-agent/libs/retriever-plugin"
)

// pluginPath is the absolute path the catalog names for the plugin, and pluginBody is the content that the pin checks.
const pluginPath = "/opt/plugins/wiki"

var pluginBody = []byte("#!/bin/sh\necho v1\n")

// copyPath is the private copy that Call runs: the first private folder, then the name plugin.
const copyPath = "/private/1/plugin"

// okFrame is one reply frame with the result object given, followed by the newline that ends a frame.
func okFrame(result string) string {
	return `{"jsonrpc":"2.0","id":1,"result":` + result + "}\n"
}

// pinOf returns the lower-case hex SHA-256 of data, the pin a catalog names for those bytes.
func pinOf(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// checkedRef returns the reference to the plugin whose pin matches pluginBody.
func checkedRef(t *testing.T) domain.PluginRef {
	t.Helper()
	ref, err := domain.NewPluginRef(pluginPath, pinOf(pluginBody))
	if err != nil {
		t.Fatalf("NewPluginRef: %v", err)
	}
	return ref
}

// callConfig returns the configuration of one call to the checked plugin, with the credential sentinel and no identity.
// The sentinel is never a real secret: it only has to be recognisable in the redacted output.
func callConfig(t *testing.T, timeout time.Duration) plugin.Config {
	t.Helper()
	return plugin.Config{
		Plugin:          checkedRef(t),
		CredentialEnv:   "EXAMPLE_WIKI_API_KEY",
		CredentialValue: "credential-sentinel-9f2a",
		Timeout:         timeout,
	}
}

// memFiles is an in-memory plugin.Files. It holds the plugin file at pluginPath and records each private folder it
// removes. Each failure is one field, and onCopy runs after the copy is written, before the check.
type memFiles struct {
	files     map[string][]byte
	dir       string
	mkdirErr  error
	copyErr   error
	hashErr   error
	removeErr error
	onCopy    func(f *memFiles)
	removed   []string
}

// newMemFiles returns the fake with the plugin file holding body, and one private folder named /private/1.
func newMemFiles(body []byte) *memFiles {
	return &memFiles{files: map[string][]byte{pluginPath: append([]byte(nil), body...)}, dir: "/private/1"}
}

func (f *memFiles) MakePrivateDir() (string, error) {
	if f.mkdirErr != nil {
		return "", f.mkdirErr
	}
	return f.dir, nil
}

func (f *memFiles) CopyFile(src, dst string) error {
	if f.copyErr != nil {
		return f.copyErr
	}
	data, ok := f.files[src]
	if !ok {
		return errors.New("no such plugin file")
	}
	f.files[dst] = append([]byte(nil), data...)
	if f.onCopy != nil {
		f.onCopy(f)
	}
	return nil
}

func (f *memFiles) HashFile(path string) (string, error) {
	if f.hashErr != nil {
		return "", f.hashErr
	}
	data, ok := f.files[path]
	if !ok {
		return "", errors.New("no file to hash")
	}
	return pinOf(data), nil
}

func (f *memFiles) RemoveAll(path string) error {
	f.removed = append(f.removed, path)
	return f.removeErr
}

// fakeCommand is a plugin.Command that starts no process. Its pipes carry the bytes the test gives, and each failure is
// one field. holdStdout keeps the output silent and holdWait keeps the process running, until KillGroup ends it. Every
// start and kill is recorded, so a test reads what the host did.
type fakeCommand struct {
	mu sync.Mutex

	stdinErr  error
	stdoutErr error
	stderrErr error
	startErr  error
	writeErr  error
	closeErr  error

	stdout     io.Reader
	stderr     io.Reader
	holdStdout bool
	holdWait   bool

	builds     int
	path, dir  string
	env        []string
	frames     [][]byte
	groupKills int

	released    chan struct{}
	releaseOnce sync.Once
}

// newFakeCommand returns a command whose process writes stdout and nothing on its error output.
func newFakeCommand(stdout string) *fakeCommand {
	return &fakeCommand{stdout: strings.NewReader(stdout), stderr: strings.NewReader(""), released: make(chan struct{})}
}

// record notes one build of the command, with its path, working folder, and environment.
func (c *fakeCommand) record(path, dir string, env []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.builds++
	c.path, c.dir, c.env = path, dir, append([]string(nil), env...)
}

// killed reports how many times the process group was ended.
func (c *fakeCommand) killed() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.groupKills
}

// sentFrames returns a copy of the frames the host wrote to the process input.
func (c *fakeCommand) sentFrames() [][]byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([][]byte, len(c.frames))
	for i, frame := range c.frames {
		out[i] = append([]byte(nil), frame...)
	}
	return out
}

func (c *fakeCommand) StdinPipe() (io.WriteCloser, error) {
	if c.stdinErr != nil {
		return nil, c.stdinErr
	}
	return &fakeStdin{c: c}, nil
}

func (c *fakeCommand) StdoutPipe() (io.ReadCloser, error) {
	if c.stdoutErr != nil {
		return nil, c.stdoutErr
	}
	return io.NopCloser(gated{c: c, r: c.stdout}), nil
}

func (c *fakeCommand) StderrPipe() (io.ReadCloser, error) {
	if c.stderrErr != nil {
		return nil, c.stderrErr
	}
	return io.NopCloser(c.stderr), nil
}

func (c *fakeCommand) Start() error { return c.startErr }

func (c *fakeCommand) Wait() error {
	if c.holdWait {
		<-c.released
	}
	return nil
}

// KillGroup ends the fake process: it counts the kill and releases every call that waits on the process.
func (c *fakeCommand) KillGroup() error {
	c.mu.Lock()
	c.groupKills++
	c.mu.Unlock()
	c.releaseOnce.Do(func() { close(c.released) })
	return nil
}

// fakeStdin records each write to the process input, and fails with writeErr or closeErr when one is set.
type fakeStdin struct {
	c *fakeCommand
}

func (s *fakeStdin) Write(p []byte) (int, error) {
	if s.c.writeErr != nil {
		return 0, s.c.writeErr
	}
	s.c.mu.Lock()
	s.c.frames = append(s.c.frames, append([]byte(nil), p...))
	s.c.mu.Unlock()
	return len(p), nil
}

func (s *fakeStdin) Close() error { return s.c.closeErr }

// gated is the process output. While the process is held silent, a read waits for KillGroup and then ends.
type gated struct {
	c *fakeCommand
	r io.Reader
}

func (g gated) Read(p []byte) (int, error) {
	if g.c.holdStdout {
		<-g.c.released
		return 0, io.EOF
	}
	return g.r.Read(p)
}

// fakeProc is a plugin.Proc that answers from memory, for the scenarios whose failures sit above the pipes.
type fakeProc struct {
	writeErr error
	reply    []byte
	readErr  error
}

func (p *fakeProc) Write(frame []byte) error { return p.writeErr }

func (p *fakeProc) Read() ([]byte, error) {
	if p.readErr != nil {
		return nil, p.readErr
	}
	return append([]byte(nil), p.reply...), nil
}

func (p *fakeProc) Stderr() []byte { return nil }
func (p *fakeProc) Kill() error    { return nil }
func (p *fakeProc) Wait() error    { return nil }

// never is a timer that does not fire. A call whose reply arrives first never reaches its limit.
func never(time.Duration) <-chan time.Time { return make(chan time.Time) }

// fired is a timer that has already fired. A call that gets no reply reaches its limit at once.
func fired(time.Duration) <-chan time.Time {
	c := make(chan time.Time, 1)
	c <- time.Time{}
	return c
}

// graceFires fires only the wait that follows an answer, which the library bounds at five seconds. The reply then
// wins the race with the call limit, so the grace timer decides whether the process is killed.
func graceFires(d time.Duration) <-chan time.Time {
	if d == 5*time.Second {
		return fired(d)
	}
	return never(d)
}

// seamsWith returns the seams of one call: files serves the file calls, cmd is the one process the call starts, and
// after is the timer. The process is built through the command seam, so the real pipes and the real frame reader run
// over the fake.
func seamsWith(files *memFiles, cmd *fakeCommand, after func(time.Duration) <-chan time.Time) plugin.Seams {
	return plugin.Seams{
		Files: files,
		NewCommand: func(path, dir string, env []string) plugin.Command {
			cmd.record(path, dir, env)
			return cmd
		},
		After: after,
	}
}

// fakeCredentials is a ports.Credentials and a ports.Identities. Each lookup is counted, and a value set to an error
// fails the lookup with it.
type fakeCredentials struct {
	apiKey    string
	apiErr    error
	apiReads  int
	address   string
	identErr  error
	identRead int
}

func (c *fakeCredentials) APIKey(ref domain.CredentialRef) (domain.ApiKey, error) {
	c.apiReads++
	if c.apiErr != nil {
		return domain.ApiKey{}, c.apiErr
	}
	return domain.NewApiKey(ref.Name(), c.apiKey)
}

func (c *fakeCredentials) Identity(ref domain.IdentityRef) (domain.Identity, error) {
	c.identRead++
	if c.identErr != nil {
		return domain.Identity{}, c.identErr
	}
	return domain.NewIdentity(ref.Name(), c.address)
}

// wikiSource returns the outline source "docs" of the catalog, with the checked plugin when withPlugin is set and the
// identity name when withIdentity is set.
func wikiSource(t *testing.T, withPlugin, withIdentity bool) domain.Source {
	t.Helper()
	id, err := domain.NewSourceID("docs")
	if err != nil {
		t.Fatalf("NewSourceID: %v", err)
	}
	endpoint, err := domain.NewEndpoint("https://docs.example.test")
	if err != nil {
		t.Fatalf("NewEndpoint: %v", err)
	}
	credential, err := domain.NewCredentialRef("EXAMPLE_WIKI_API_KEY")
	if err != nil {
		t.Fatalf("NewCredentialRef: %v", err)
	}
	tags, err := domain.NewTagSet()
	if err != nil {
		t.Fatalf("NewTagSet: %v", err)
	}
	src := domain.NewSource(id, domain.KindOutline, tags, endpoint, credential)
	if withIdentity {
		identity, err := domain.NewIdentityRef("EXAMPLE_WIKI_EMAIL")
		if err != nil {
			t.Fatalf("NewIdentityRef: %v", err)
		}
		src = src.WithIdentity(identity)
	}
	if withPlugin {
		src = src.WithPlugin(checkedRef(t))
	}
	return src
}

// gatewayOver returns a gateway over the checked plugin of source "docs". The file calls run on files, the timer is
// after, and each call starts a fresh process from build. The processes are kept in order, so a test reads what each
// call sent.
func gatewayOver(
	t *testing.T,
	withIdentity bool,
	files *memFiles,
	after func(time.Duration) <-chan time.Time,
	build func() *fakeCommand,
) (*plugin.Gateway, *[]*fakeCommand) {
	t.Helper()
	started := &[]*fakeCommand{}
	creds := &fakeCredentials{apiKey: "credential-sentinel-9f2a", address: "name@example.test"}
	gw := &plugin.Gateway{
		Source:      wikiSource(t, true, withIdentity),
		Credentials: creds,
		Identities:  creds,
		Seams: plugin.Seams{
			Files: files,
			NewCommand: func(path, dir string, env []string) plugin.Command {
				cmd := build()
				cmd.record(path, dir, env)
				*started = append(*started, cmd)
				return cmd
			},
			After: after,
		},
		Timeout: time.Second,
	}
	return gw, started
}
