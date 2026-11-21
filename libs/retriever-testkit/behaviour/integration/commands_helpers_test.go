//go:build integration

package integration_test

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// commandWait bounds every wait on a command run, so a hang fails the test instead of stalling the suite.
const commandWait = 5 * time.Second

// commandNotify is the signal wait that a command takes. A test passes its own, so the run ends when the test says so.
type commandNotify = func(parent context.Context, signals ...os.Signal) (context.Context, context.CancelFunc)

// stopAtOnceNotify is a signal wait that is already done, so a run returns once it has started the stub and printed.
func stopAtOnceNotify(parent context.Context, _ ...os.Signal) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(parent)
	cancel()
	return ctx, cancel
}

// commandRun is one run of a stub command in its own goroutine. Its standard output is read through a pipe, and its
// standard error is kept in memory. The test ends the run with its stop signal.
type commandRun struct {
	pipe   *io.PipeReader
	stdout *bufio.Reader
	stderr *bytes.Buffer
	stop   context.CancelFunc
	done   chan int
}

// startCommandRun runs body in a goroutine. body gets the standard output, the standard error, and the signal wait to pass
// to its host. startCommandRun returns once body has taken its signal wait, which it does after the stub has started.
func startCommandRun(t *testing.T, body func(stdout, stderr io.Writer, notify commandNotify) int) *commandRun {
	t.Helper()
	pr, pw := io.Pipe()
	r := &commandRun{pipe: pr, stdout: bufio.NewReader(pr), stderr: &bytes.Buffer{}, done: make(chan int, 1)}
	waiting := make(chan struct{})
	notify := func(parent context.Context, _ ...os.Signal) (context.Context, context.CancelFunc) {
		ctx, cancel := context.WithCancel(parent)
		r.stop = cancel
		close(waiting)
		return ctx, cancel
	}
	go func() {
		code := body(pw, r.stderr, notify)
		pw.Close()
		r.done <- code
	}()
	select {
	case <-waiting:
	case code := <-r.done:
		t.Fatalf("the command returned %d before it waited for its stop signal: %s", code, r.stderr.String())
	case <-time.After(commandWait):
		t.Fatalf("the command did not start within %v", commandWait)
	}
	return r
}

// readLine returns the first line the command printed, failing the test if none arrives in time.
func (r *commandRun) readLine(t *testing.T) string {
	t.Helper()
	type read struct {
		line string
		err  error
	}
	got := make(chan read, 1)
	go func() {
		line, err := r.stdout.ReadString('\n')
		got <- read{line, err}
	}()
	select {
	case res := <-got:
		if res.err != nil {
			t.Fatalf("read the command's first line: %v", res.err)
		}
		return strings.TrimSuffix(res.line, "\n")
	case <-time.After(commandWait):
		t.Fatalf("the command printed no line within %v", commandWait)
		return ""
	}
}

// finish ends the run with its stop signal, and returns its status. What the run still prints is drained first, so the
// run can reach its wait.
func (r *commandRun) finish(t *testing.T) int {
	t.Helper()
	go func() { _, _ = io.Copy(io.Discard, r.pipe) }()
	r.stop()
	select {
	case code := <-r.done:
		return code
	case <-time.After(commandWait):
		t.Fatalf("the command did not return within %v after its stop signal", commandWait)
		return -1
	}
}

// commandCAClient returns an HTTP client that trusts only the PEM certificate in caPath, and opens no reused connection,
// so a request after the stub stops cannot reach it.
func commandCAClient(t *testing.T, caPath string) *http.Client {
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
		TLSClientConfig:   &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12},
		DisableKeepAlives: true,
	}}
}
