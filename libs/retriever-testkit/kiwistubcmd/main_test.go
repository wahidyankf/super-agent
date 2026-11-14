package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-testkit/kiwistub"
)

const fixtureKey = "fixture-key-cmd"

var (
	errListen = errors.New("test: port unavailable")
	errWrite  = errors.New("test: disk full")
)

// fixedNow is the clock of every unit test, so no test reads the real clock.
var fixedNow = time.Date(2030, time.January, 2, 3, 4, 5, 0, time.UTC)

func fixedClock() time.Time {
	return fixedNow
}

// fakeListener is an in-memory net.Listener. It hands out no connection, and Close marks it closed, so no socket opens.
type fakeListener struct {
	addr      net.Addr
	closed    chan struct{}
	closeOnce sync.Once
}

// Accept blocks until the listener is closed, and then reports that it is closed.
func (l *fakeListener) Accept() (net.Conn, error) {
	<-l.closed
	return nil, net.ErrClosed
}

// Close marks the listener closed. A second Close does nothing.
func (l *fakeListener) Close() error {
	l.closeOnce.Do(func() { close(l.closed) })
	return nil
}

// Addr returns the loopback address the fake listener reports.
func (l *fakeListener) Addr() net.Addr {
	return l.addr
}

// fakeHost is a host of fakes: a map for the environment and one for the files, a recorder for every write, and an
// in-memory listener. Its two output streams are buffers, so a test reads what the command printed.
type fakeHost struct {
	env        map[string]string
	now        func() time.Time
	files      map[string][]byte
	written    map[string][]byte
	perms      map[string]fs.FileMode
	writeErr   error
	listenErr  error
	listened   int
	listenedOn string
	listener   *fakeListener
	stdout     bytes.Buffer
	stderr     bytes.Buffer
}

func newFakeHost() *fakeHost {
	return &fakeHost{
		env:      map[string]string{keyVariable: fixtureKey},
		now:      fixedClock,
		files:    map[string][]byte{},
		written:  map[string][]byte{},
		perms:    map[string]fs.FileMode{},
		listener: &fakeListener{addr: &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 4321}, closed: make(chan struct{})},
	}
}

// host returns the command's host of fakes.
func (f *fakeHost) host() host {
	return host{
		getenv: func(name string) string { return f.env[name] },
		now:    f.now,
		stdout: &f.stdout,
		stderr: &f.stderr,
		readFile: func(name string) ([]byte, error) {
			data, ok := f.files[name]
			if !ok {
				return nil, fs.ErrNotExist
			}
			return data, nil
		},
		writeFile: func(name string, data []byte, perm fs.FileMode) error {
			if f.writeErr != nil {
				return f.writeErr
			}
			f.written[name] = data
			f.perms[name] = perm
			return nil
		},
		listen: func(network, address string) (net.Listener, error) {
			f.listened++
			f.listenedOn = network + " " + address
			if f.listenErr != nil {
				return nil, f.listenErr
			}
			return f.listener, nil
		},
	}
}

// cancelledContext returns a context that is already done, so run returns at once after it starts the stub.
func cancelledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

// fixtureDataset is the JSON of one product, as a dataset file holds it.
const fixtureDataset = `{"Products":[{"ID":1,"Name":"Example Portal"}]}`

// TestRunNeedsTheCAFile checks that a run without -ca-file stops before it listens.
func TestRunNeedsTheCAFile(t *testing.T) {
	f := newFakeHost()
	if err := run(cancelledContext(), nil, f.host()); !errors.Is(err, errNoCAFile) {
		t.Fatalf("run without -ca-file = %v, want errNoCAFile", err)
	}
	if f.listened != 0 {
		t.Fatalf("a run without -ca-file listened %d times", f.listened)
	}
}

// TestRunNeedsTheKeyVariable checks that an empty key variable stops the run, and that the error names the variable only.
func TestRunNeedsTheKeyVariable(t *testing.T) {
	f := newFakeHost()
	f.env[keyVariable] = ""
	err := run(cancelledContext(), []string{"-ca-file", "ca.pem"}, f.host())
	if !errors.Is(err, errNoKey) || !strings.Contains(err.Error(), keyVariable) {
		t.Fatalf("run with an empty key = %v, want errNoKey naming the variable", err)
	}
	if f.listened != 0 {
		t.Fatalf("a run with no key listened %d times", f.listened)
	}
}

// TestRunReportsAFlagItDoesNotKnow checks that an unknown flag stops the run, and that the flag package reports it on
// the error stream.
func TestRunReportsAFlagItDoesNotKnow(t *testing.T) {
	f := newFakeHost()
	if err := run(cancelledContext(), []string{"-nope"}, f.host()); err == nil {
		t.Fatalf("run with an unknown flag returned no error")
	}
	if !strings.Contains(f.stderr.String(), "-nope") || f.listened != 0 {
		t.Fatalf("stderr = %q, listened = %d: want the flag named and no listen", f.stderr.String(), f.listened)
	}
}

// TestRunPrintsTheListeningLineAfterItWritesTheCA checks the output contract: the stub listens on the loopback with a
// free port, the CA file holds one PEM certificate with owner-only permissions, and the one line names the URL.
func TestRunPrintsTheListeningLineAfterItWritesTheCA(t *testing.T) {
	f := newFakeHost()
	f.files["dataset.json"] = []byte(fixtureDataset)
	args := []string{"-ca-file", "ca.pem", "-dataset-file", "dataset.json"}
	if err := run(cancelledContext(), args, f.host()); err != nil {
		t.Fatalf("run: %v", err)
	}
	if f.listenedOn != "tcp 127.0.0.1:0" {
		t.Fatalf("listened on %q, want tcp on the loopback address with a free port", f.listenedOn)
	}
	if got := f.stdout.String(); got != "kiwi stub listening on https://127.0.0.1:4321\n" {
		t.Fatalf("output = %q, want the one listening line", got)
	}
	if !strings.HasPrefix(string(f.written["ca.pem"]), "-----BEGIN CERTIFICATE-----") || f.perms["ca.pem"] != 0o600 {
		t.Fatalf("CA file = %q at mode %o, want a PEM certificate at mode 600", f.written["ca.pem"], f.perms["ca.pem"])
	}
}

// TestRunStopsBeforeItListens checks each failure that comes before the listen: a dataset file that cannot be read, one
// that does not parse, and a listen that fails. Each returns its error, and prints nothing.
func TestRunStopsBeforeItListens(t *testing.T) {
	cases := []struct {
		name    string
		setup   func(f *fakeHost)
		args    []string
		wantErr string
	}{
		{"the dataset file is missing", func(f *fakeHost) {}, []string{"-ca-file", "ca.pem", "-dataset-file", "missing.json"}, "read the dataset file"},
		{"the dataset file does not parse", func(f *fakeHost) {
			f.files["bad.json"] = []byte("{")
		}, []string{"-ca-file", "ca.pem", "-dataset-file", "bad.json"}, "parse the dataset file"},
		{"the port cannot be opened", func(f *fakeHost) {
			f.listenErr = errListen
		}, []string{"-ca-file", "ca.pem"}, "listen on loopback"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFakeHost()
			tc.setup(f)
			err := run(cancelledContext(), tc.args, f.host())
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("run error = %v, want one that says %q", err, tc.wantErr)
			}
			if f.stdout.Len() != 0 {
				t.Fatalf("a failed run printed %q", f.stdout.String())
			}
		})
	}
}

// TestRunStopsTheStubWhenTheCAFileCannotBeWritten checks that a failed CA write returns its error, and that the listener
// the stub opened is closed.
func TestRunStopsTheStubWhenTheCAFileCannotBeWritten(t *testing.T) {
	f := newFakeHost()
	f.writeErr = errWrite
	err := run(cancelledContext(), []string{"-ca-file", "ca.pem"}, f.host())
	if !errors.Is(err, errWrite) || !strings.Contains(err.Error(), "write the CA file") {
		t.Fatalf("run error = %v, want the CA write error", err)
	}
	<-f.listener.closed
	if f.stdout.Len() != 0 {
		t.Fatalf("a run whose CA write failed printed %q", f.stdout.String())
	}
}

// TestStartServesTheDatasetAndTheFault checks the started stub: it answers from the dataset file, and a fault status
// answers every request with that status.
func TestStartServesTheDatasetAndTheFault(t *testing.T) {
	cases := []struct {
		name        string
		faultStatus int
		wantStatus  int
		wantInBody  string
	}{
		{"replays the dataset", 0, http.StatusOK, "Example Portal"},
		{"answers every request with the fault status", 503, http.StatusServiceUnavailable, "refused"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFakeHost()
			var d kiwistub.Dataset
			if err := json.Unmarshal([]byte(fixtureDataset), &d); err != nil {
				t.Fatal(err)
			}
			stub, err := start(config{caFile: "ca.pem", key: fixtureKey, dataset: d, faultStatus: tc.faultStatus}, f.host())
			if err != nil {
				t.Fatalf("start: %v", err)
			}
			defer stub.Close()

			status, body := postProducts(t, stub)
			if status != tc.wantStatus || !strings.Contains(body, tc.wantInBody) {
				t.Fatalf("reply = %d %s, want %d with %q", status, body, tc.wantStatus, tc.wantInBody)
			}
		})
	}
}

// postProducts asks a stub for its products through its in-process transport, with the command's key.
func postProducts(t *testing.T, stub *kiwistub.Stub) (int, string) {
	t.Helper()
	body := `{"jsonrpc":"2.0","method":"Product.filter","params":{"query":{}},"id":1}`
	req, err := http.NewRequest(http.MethodPost, "https://kiwi.example.test"+kiwistub.Endpoint, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+fixtureKey)
	resp, err := stub.RoundTripper().RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read the reply: %v", err)
	}
	return resp.StatusCode, string(raw)
}

// TestReadDatasetParsesTheStubFields checks the dataset file: it is the JSON of kiwistub.Dataset, and a missing or bad
// file is an error.
func TestReadDatasetParsesTheStubFields(t *testing.T) {
	f := newFakeHost()
	f.files["good.json"] = []byte(fixtureDataset)
	f.files["bad.json"] = []byte("{")
	d, err := readDataset(f.host(), "good.json")
	if err != nil || len(d.Products) != 1 || d.Products[0].Name != "Example Portal" {
		t.Fatalf("readDataset = (%+v, %v)", d, err)
	}
	if _, err := readDataset(f.host(), "bad.json"); err == nil {
		t.Fatalf("readDataset of a bad file returned no error")
	}
	if _, err := readDataset(f.host(), "missing.json"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("readDataset of a missing file = %v, want the not-exist error", err)
	}
}

// TestExitStatusReportsAFailureOnStderr checks the process status: a clean run exits 0 and prints nothing, and a failed
// run exits 1 and prints its error.
func TestExitStatusReportsAFailureOnStderr(t *testing.T) {
	var clean bytes.Buffer
	if status := exitStatus(&clean, nil); status != 0 || clean.Len() != 0 {
		t.Fatalf("clean run = status %d, stderr %q; want 0 and nothing", status, clean.String())
	}
	var failed bytes.Buffer
	if status := exitStatus(&failed, errNoCAFile); status != 1 || failed.String() != errNoCAFile.Error()+"\n" {
		t.Fatalf("failed run = status %d, stderr %q; want 1 and the error", status, failed.String())
	}
}
