//go:build e2e

package e2e_test

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const (
	jiraEmail = "name@example.test"
	jiraToken = "EXAMPLE_TOKEN"
	kiwiKey   = "EXAMPLE_KEY"
	keyEnv    = "KIWISTUB_API_KEY"
)

// commandBinary builds one command package of the testkit module into a temporary directory and returns the binary path.
// The test runs in behaviour/e2e, so the module root is two levels up.
func commandBinary(t *testing.T, pkg string) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), pkg)
	cmd := exec.Command("go", "build", "-o", bin, "./"+pkg)
	cmd.Dir = filepath.Join("..", "..")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build %s: %v: %s", pkg, err, out)
	}
	return bin
}

// result is the outcome of a command that ran to its end.
type result struct {
	code   int
	stdout string
	stderr string
}

// runCommand runs bin with args and env, and waits for it to end. A nil env inherits the environment of the test.
func runCommand(t *testing.T, env []string, bin string, args ...string) result {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Env = env
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		code = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("run %s: %v", bin, err)
	}
	return result{code: code, stdout: stdout.String(), stderr: stderr.String()}
}

// running is a command that is listening: its announcement line is the first line of its standard output.
type running struct {
	cmd      *exec.Cmd
	announce string
	stderr   *bytes.Buffer
	ended    bool
}

// startCommand starts bin with args and env, and reads the line that announces its address.
func startCommand(t *testing.T, env []string, bin string, args ...string) *running {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Env = env
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("pipe the output of %s: %v", bin, err)
	}
	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start %s: %v", bin, err)
	}
	r := &running{cmd: cmd, stderr: stderr}
	t.Cleanup(func() {
		if !r.ended {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	})
	line, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil {
		t.Fatalf("read the announcement of %s: %v; stderr: %s", bin, err, stderr.String())
	}
	r.announce = strings.TrimSpace(line)
	return r
}

// stop sends the interrupt that a user sends, waits for the command to return, and gives its exit status.
func (r *running) stop(t *testing.T) int {
	t.Helper()
	if err := r.cmd.Process.Signal(os.Interrupt); err != nil {
		t.Fatalf("interrupt the command: %v", err)
	}
	err := r.cmd.Wait()
	r.ended = true
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	if err != nil {
		t.Fatalf("wait for the command: %v", err)
	}
	return 0
}

// announcedURL returns the URL of an announcement such as "kiwi stub listening on https://127.0.0.1:43123".
func announcedURL(t *testing.T, announce, prefix string) string {
	t.Helper()
	if !strings.HasPrefix(announce, prefix) {
		t.Fatalf("the announcement %q does not start with %q", announce, prefix)
	}
	return strings.TrimPrefix(announce, prefix)
}

// tlsClient returns an HTTP client that trusts the PEM certificate in caPath, and nothing else.
func tlsClient(t *testing.T, caPath string) *http.Client {
	t.Helper()
	data, err := os.ReadFile(caPath)
	if err != nil {
		t.Fatalf("read the CA file: %v", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(data) {
		t.Fatal("the CA file holds no PEM certificate")
	}
	return &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12},
	}}
}

// environment returns the environment of the test with the stub key set to key, or unset when key is empty.
func environment(key string) []string {
	var env []string
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, keyEnv+"=") {
			env = append(env, kv)
		}
	}
	if key != "" {
		env = append(env, fmt.Sprintf("%s=%s", keyEnv, key))
	}
	return env
}
