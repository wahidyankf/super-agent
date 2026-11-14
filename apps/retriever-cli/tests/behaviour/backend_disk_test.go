//go:build integration || e2e

package behaviour

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/graphql"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/catalog"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/envfile"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/markdownfiles"
)

// errNoBinary is the error of a process at a level without a built binary.
var errNoBinary = errors.New("the end-to-end level has no built retriever binary")

// diskBackend keeps a scenario on the real disk. The integration level runs commands in process, with the real
// environment and the working directory moved to the scenario folder; the end-to-end level runs the built binary, with
// the environment the scenario names, and leaves the process state alone.
type diskBackend struct {
	outer  *testing.T
	inProc bool
	binary string

	root     string
	cwd      string
	savedEnv []string
}

// newDiskBackend returns a disk backend for one scenario. An in-process backend moves the working directory and the
// process environment, and restores them in stop. A binary backend runs the file at binary.
func newDiskBackend(outer *testing.T, inProcess bool, binary string) backend {
	return &diskBackend{outer: outer, inProc: inProcess, binary: binary}
}

// start creates the scenario folder. An in-process backend also saves the working directory and the environment, and
// makes the folder the working directory.
func (d *diskBackend) start() (string, error) {
	d.root = d.outer.TempDir()
	if d.inProc {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		d.cwd = cwd
		d.savedEnv = os.Environ()
		if err := os.Chdir(d.root); err != nil {
			return "", err
		}
	}
	return d.root, nil
}

// stop restores the working directory and the process environment that start saved.
func (d *diskBackend) stop() {
	if !d.inProc {
		return
	}
	if d.cwd != "" {
		_ = os.Chdir(d.cwd)
	}
	if d.savedEnv != nil {
		os.Clearenv()
		for _, kv := range d.savedEnv {
			k, v, _ := strings.Cut(kv, "=")
			_ = os.Setenv(k, v)
		}
	}
}

// inProcess reports whether the backend runs commands in this process.
func (d *diskBackend) inProcess() bool { return d.inProc }

// adoptEnv makes the process environment exactly env. The credentials adapter reads secrets with os.Getenv, so a
// variable the scenario does not set must be absent from the process, not inherited from the developer's shell.
func (d *diskBackend) adoptEnv(env map[string]string) {
	os.Clearenv()
	for k, v := range env {
		_ = os.Setenv(k, v)
	}
}

// getwd returns the process working directory.
func (d *diskBackend) getwd() (string, error) { return os.Getwd() }

// writeFile creates or replaces the file at name, with its folder. The file is private to its owner.
func (d *diskBackend) writeFile(name string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		return err
	}
	return os.WriteFile(name, data, 0o600)
}

// mkdir creates the folder at name, with its parents.
func (d *diskBackend) mkdir(name string) error { return os.MkdirAll(name, 0o755) }

// readFile returns the bytes of the file at name.
func (d *diskBackend) readFile(name string) ([]byte, error) { return os.ReadFile(name) }

// regularFile reports whether a regular file exists at name, as the composition root checks it.
func (d *diskBackend) regularFile(name string) bool {
	info, err := os.Stat(name)
	return err == nil && info.Mode().IsRegular()
}

// walk returns every regular file under root, keyed by slash path relative to root.
func (d *diskBackend) walk(root string) (map[string]string, error) {
	files := map[string]string{}
	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = string(data)
		return nil
	})
	return files, err
}

// dirFiles returns the regular files directly inside the folder at name. A folder that does not exist holds none.
func (d *diskBackend) dirFiles(name string) (map[string]string, error) {
	entries, err := os.ReadDir(name)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	files := map[string]string{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(name, e.Name()))
		if err != nil {
			return nil, err
		}
		files[e.Name()] = string(data)
	}
	return files, nil
}

// credentials reads the env file with the real environment, as the composition root does. The process environment is
// the scenario environment, because adoptEnv made it so.
func (d *diskBackend) credentials(name string, _ func(string) string) (*envfile.Credentials, error) {
	return envfile.NewWithFiles(name, os.Getenv, envfile.Files{FS: os.DirFS(filepath.Dir(name)), SameFile: os.SameFile})
}

// loadCatalog reads the catalog from the real disk, as the composition root does.
func (d *diskBackend) loadCatalog(name string) (domain.Catalog, error) {
	loader, err := catalog.NewWithFiles(name, catalog.Files{FS: os.DirFS(filepath.Dir(name)), SameFile: os.SameFile})
	if err != nil {
		return domain.Catalog{}, err
	}
	return loader.Load()
}

// sink returns the Markdown sink that writes into dir on the real disk, as the composition root builds it.
func (d *diskBackend) sink(dir string) ports.ExportSink {
	return markdownfiles.New(dir, func(dir, pattern string) (markdownfiles.TempFile, error) {
		return os.CreateTemp(dir, pattern)
	}, markdownfiles.Folder{MkdirAll: os.MkdirAll, Rename: os.Rename, Remove: os.Remove})
}

// spawnServe starts the built binary as retriever serve, with the scenario environment, in the scenario folder.
func (d *diskBackend) spawnServe(w *world, argv []string) (*serveSession, error) {
	if d.binary == "" {
		return nil, errNoBinary
	}
	cmd := exec.Command(d.binary, argv...)
	cmd.Dir = w.workdir()
	cmd.Env = w.envList()
	cmd.Stderr = io.Discard
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start retriever serve: %w", err)
	}
	s := &serveSession{
		signal: func(sig os.Signal) error { return cmd.Process.Signal(sig) },
		end: endState{
			exited: make(chan int, 1),
			kill:   func() { _ = cmd.Process.Kill() },
		},
	}
	go func() { s.end.exited <- exitCodeOf(cmd.Wait()) }()
	w.procKill = append(w.procKill, s.end.kill)
	lines := make(chan string, 16)
	go readLines(stdout, lines)
	addr, err := awaitServeLine(lines)
	if err != nil {
		return nil, err
	}
	s.addr = addr
	return s, nil
}

// spawnMCP starts the built binary as retriever mcp, with the scenario environment, in the scenario folder.
func (d *diskBackend) spawnMCP(w *world, argv []string) (*mcpSession, error) {
	if d.binary == "" {
		return nil, errNoBinary
	}
	cmd := exec.Command(d.binary, argv...)
	cmd.Dir = w.workdir()
	cmd.Env = w.envList()
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start retriever mcp: %w", err)
	}
	m := &mcpSession{
		stdin:  stdin,
		lines:  make(chan string, 64),
		signal: func(sig os.Signal) error { return cmd.Process.Signal(sig) },
		end: endState{
			exited: make(chan int, 1),
			kill:   func() { _ = cmd.Process.Kill() },
		},
	}
	go func() { m.end.exited <- exitCodeOf(cmd.Wait()) }()
	go readLines(stdout, m.lines)
	w.procKill = append(w.procKill, func() {
		_ = m.stdin.Close()
		m.end.kill()
	})
	return m, nil
}

// runBinary runs the built binary once, with the scenario environment, in the scenario folder, and records its result.
// A standard output with no reader ends the process by SIGPIPE, as a closed pipe does.
func (d *diskBackend) runBinary(w *world, argv []string) error {
	if d.binary == "" {
		return errNoBinary
	}
	cmd := exec.Command(d.binary, argv...)
	cmd.Dir = w.workdir()
	cmd.Env = w.envList()
	cmd.Stdin = strings.NewReader("")
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	if w.stdoutClosed {
		reader, writer, err := os.Pipe()
		if err != nil {
			return err
		}
		_ = reader.Close()
		defer writer.Close()
		cmd.Stdout = writer
	}
	cmd.Stderr = &errb
	err := cmd.Run()
	var exitErr *exec.ExitError
	if err != nil && !errors.As(err, &exitErr) {
		return fmt.Errorf("start retriever: %w", err)
	}
	w.record(exitCodeOf(err), &out, &errb)
	return nil
}

// exitCodeOf returns the status of a finished process: its exit code, or 128+N when signal N ended it, as the shell
// reports it.
func exitCodeOf(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return -1
	}
	if st, ok := exitErr.Sys().(syscall.WaitStatus); ok && st.Signaled() {
		return 128 + int(st.Signal())
	}
	return exitErr.ExitCode()
}

// init arms the reply deadline of the disk levels: a real timer that fails a scenario whose process never answers. The
// unit level keeps the nil deadline of process_test.go and arms no timer.
func init() {
	replyDeadline = func() <-chan time.Time { return time.After(replyTimeout) }
}

// listenPort opens the real loopback listener of a port, as retriever serve does.
func (d *diskBackend) listenPort(port int) (net.Listener, error) { return graphql.Listen(port) }

// listenTCP opens a real listener for a stub.
func (d *diskBackend) listenTCP(network, address string) (net.Listener, error) {
	return net.Listen(network, address)
}

// seamClient returns the seam's client unchanged: the disk levels dial the real loopback stubs.
func (d *diskBackend) seamClient(c *http.Client) *http.Client { return c }

// httpClient returns a client with the reply timeout, which queries the real served endpoint.
func (d *diskBackend) httpClient() *http.Client { return &http.Client{Timeout: replyTimeout} }

// clockOf returns the scenario clock for an in-process level. The end-to-end level runs the built binary against stubs in
// this process, so its stubs date their certificates from the real clock the binary also reads.
func (d *diskBackend) clockOf(c *fakeClock) func() time.Time {
	if d.inProc {
		return c.Now
	}
	return time.Now
}
