//go:build integration

package integration_test

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-testkit/kiwistub"
	kiwicmd "github.com/wahidyankf/super-agent/libs/retriever-testkit/stubcmd/kiwi"
)

// kiwiCommandHost returns the real host of the Kiwi command, with the key variable set to the synthetic key and the signal
// wait the test passes. Its files, clock, loopback socket, and entropy are real.
func kiwiCommandHost(notify commandNotify) kiwicmd.Host {
	h := kiwicmd.Real
	h.Getenv = func(name string) string {
		if name == "KIWISTUB_API_KEY" {
			return kiwiKey
		}
		return ""
	}
	h.Notify = notify
	return h
}

// watchedListener is a real listener that reports when it is closed, so a test waits for the stub to release its port.
type watchedListener struct {
	net.Listener
	once   sync.Once
	closed chan struct{}
}

func (l *watchedListener) Close() error {
	l.once.Do(func() { close(l.closed) })
	return l.Listener.Close()
}

// @scenario specs/libs/retriever-testkit/behaviours/kiwi-stub-command.feature: The command writes the certificate and then prints its listening address
func TestKiwiCommandWritesTheCertificateAndAnnouncesALoopbackURL(t *testing.T) {
	ca := filepath.Join(t.TempDir(), "ca.pem")
	r := startCommandRun(t, func(stdout, stderr io.Writer, notify commandNotify) int {
		return kiwicmd.Run([]string{"-ca-file", ca}, stdout, stderr, kiwiCommandHost(notify))
	})
	base, found := strings.CutPrefix(r.readLine(t), "kiwi stub listening on ")
	if !found || !strings.HasPrefix(base, "https://127.0.0.1:") {
		t.Fatalf("the announcement is not a loopback TLS address: %q", base)
	}
	data, err := os.ReadFile(ca)
	if err != nil || !bytes.Contains(data, []byte("BEGIN CERTIFICATE")) {
		t.Fatalf("the certificate file = %v, want a PEM certificate", err)
	}

	req, err := http.NewRequest(http.MethodPost, base+kiwistub.Endpoint, strings.NewReader(`{"jsonrpc":"2.0","method":"Product.filter","params":{"query":{}},"id":1}`))
	if err != nil {
		t.Fatalf("build the request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+kiwiKey)
	resp, err := commandCAClient(t, ca).Do(req)
	if err != nil {
		t.Fatalf("the trusting client could not reach the announced address: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("the stub answered %d to the synthetic key, want 200", resp.StatusCode)
	}
	if status := r.finish(t); status != 0 {
		t.Fatalf("status %d after the stop signal, want 0", status)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/kiwi-stub-command.feature: A failure after start stops the stub and prints to standard error
func TestKiwiCommandFailureAfterStartReleasesThePortAndPrintsTheError(t *testing.T) {
	watch := &watchedListener{closed: make(chan struct{})}
	h := kiwiCommandHost(stopAtOnceNotify)
	h.Listen = func(network, address string) (net.Listener, error) {
		l, err := net.Listen(network, address)
		if err != nil {
			return nil, err
		}
		watch.Listener = l
		return watch, nil
	}
	var out bytes.Buffer
	var errOut bytes.Buffer
	missing := filepath.Join(t.TempDir(), "missing", "ca.pem")

	status := kiwicmd.Run([]string{"-ca-file", missing}, &out, &errOut, h)

	if status != 1 || !strings.Contains(errOut.String(), "no such file") {
		t.Fatalf("status %d and stderr %q, want 1 with the write error", status, errOut.String())
	}
	if out.Len() != 0 {
		t.Fatalf("the command announced %q before the failure", out.String())
	}
	select {
	case <-watch.closed:
	case <-time.After(commandWait):
		t.Fatalf("the listener was not released within %v", commandWait)
	}
	if conn, err := net.Dial("tcp", watch.Addr().String()); err == nil {
		conn.Close()
		t.Fatal("the port still accepts connections after the failed run")
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/kiwi-stub-command.feature: A clean run exits with status 0
func TestKiwiCleanRunExitsWithStatusZeroOnItsStopSignalOverRealSocket(t *testing.T) {
	ca := filepath.Join(t.TempDir(), "ca.pem")
	r := startCommandRun(t, func(stdout, stderr io.Writer, notify commandNotify) int {
		return kiwicmd.Run([]string{"-ca-file", ca}, stdout, stderr, kiwiCommandHost(notify))
	})
	if line := r.readLine(t); !strings.HasPrefix(line, "kiwi stub listening on https://127.0.0.1:") {
		t.Fatalf("the announcement = %q", line)
	}
	if status := r.finish(t); status != 0 {
		t.Fatalf("status %d after the stop signal, want 0", status)
	}
}
