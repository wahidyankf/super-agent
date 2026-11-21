package jira

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-testkit/jirastub"
)

// fixedNow is the clock of every fake run, so no unit test reads the real clock.
var fixedNow = time.Date(2026, 11, 14, 9, 0, 0, 0, time.UTC)

// Synthetic credentials of the tests. The e-mail uses the reserved example.com domain, and the token is a sentinel.
const (
	testEmail = "fixture-reviewer@example.com"
	testToken = "sentinel-token-5d2b8e-do-not-print"
)

// memFile is an in-memory record file. Close only marks it closed.
type memFile struct {
	bytes.Buffer
	closed bool
}

func (f *memFile) Close() error {
	f.closed = true
	return nil
}

// fakeServer is the TLS server seam. It keeps the handler it was given, so a test can call the stub while the command
// runs, and it opens no socket.
type fakeServer struct {
	calls   int
	err     error
	handler http.Handler
	stopped bool
	der     []byte
}

func (f *fakeServer) serve(h http.Handler) (Started, error) {
	f.calls++
	f.handler = h
	if f.err != nil {
		return Started{}, f.err
	}
	f.der = []byte("fake-certificate")
	return Started{URL: "https://127.0.0.1:1", DER: f.der, Stop: func() { f.stopped = true }}, nil
}

// fakeHost is the host of one test. Its files live in maps, and its clock is fixedNow.
type fakeHost struct {
	inputs    map[string][]byte
	written   map[string][]byte
	perms     map[string]os.FileMode
	writeErr  error
	createErr error
	record    *memFile
	server    *fakeServer
}

func newFakeHost() *fakeHost {
	return &fakeHost{
		inputs:  map[string][]byte{},
		written: map[string][]byte{},
		perms:   map[string]os.FileMode{},
		server:  &fakeServer{},
	}
}

// host returns the host of the command. Its stop signal is already done, so run returns once it has started the stub.
func (f *fakeHost) host() Host {
	return Host{
		ReadFile: func(name string) ([]byte, error) {
			data, ok := f.inputs[name]
			if !ok {
				return nil, errors.New("no fake file named " + name)
			}
			return data, nil
		},
		WriteFile: func(name string, data []byte, perm os.FileMode) error {
			if f.writeErr != nil {
				return f.writeErr
			}
			f.written[name] = data
			f.perms[name] = perm
			return nil
		},
		CreateFile: func(string) (io.WriteCloser, error) {
			if f.createErr != nil {
				return nil, f.createErr
			}
			f.record = &memFile{}
			return f.record, nil
		},
		Now:    func() time.Time { return fixedNow },
		Serve:  f.server.serve,
		Notify: stopAtOnce,
	}
}

// stopAtOnce is the signal wait of a run to its end: the context it returns is already done.
func stopAtOnce(parent context.Context, _ ...os.Signal) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(parent)
	cancel()
	return ctx, cancel
}

// live is a command running in a goroutine, with a stop signal that the test sends.
type live struct {
	out     *bytes.Buffer
	errOut  *bytes.Buffer
	signals []os.Signal
	stop    context.CancelFunc
	done    chan error
}

// startRun runs the command in a goroutine. It returns once the command waits for its stop signal, which it does after the
// stub is serving, so the test can call the stub's handler. finish sends the stop signal.
func startRun(fh *fakeHost, args ...string) *live {
	l := &live{out: &bytes.Buffer{}, errOut: &bytes.Buffer{}, done: make(chan error, 1)}
	waiting := make(chan struct{})
	h := fh.host()
	h.Notify = func(parent context.Context, signals ...os.Signal) (context.Context, context.CancelFunc) {
		ctx, cancel := context.WithCancel(parent)
		l.signals = signals
		l.stop = cancel
		close(waiting)
		return ctx, cancel
	}
	go func() {
		l.done <- run(context.Background(), args, l.out, l.errOut, h)
	}()
	<-waiting
	return l
}

// finish sends the stop signal and returns the error of the command.
func (l *live) finish() error {
	l.stop()
	return <-l.done
}

// basicAuth returns the Authorization header for the given credentials.
func basicAuth(email, token string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(email+":"+token))
}

// call sends one request to the handler in process, and returns the recorded reply.
func call(h http.Handler, method, path, auth string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "https://jira.example.test"+path, nil)
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// caArgs are the flags every run needs, with the fake CA file name.
func caArgs() []string {
	return []string{"-ca-file", "ca.pem", "-email", testEmail, "-token", testToken}
}

// runToEnd runs the command with args and the fake host, which stops at once, and returns its output and error.
func runToEnd(fh *fakeHost, args ...string) (*bytes.Buffer, *bytes.Buffer, error) {
	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
	err := run(context.Background(), args, out, errOut, fh.host())
	return out, errOut, err
}

func TestRunPrintsTheListeningLineAndWritesTheCertificate(t *testing.T) {
	fh := newFakeHost()
	out, errOut, err := runToEnd(fh, caArgs()...)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if got, want := out.String(), "jira stub listening on https://127.0.0.1:1\n"; got != want {
		t.Fatalf("listening line = %q, want %q", got, want)
	}
	if strings.Contains(out.String()+errOut.String(), testToken) {
		t.Fatal("the command output shows the token")
	}
	block, _ := pem.Decode(fh.written["ca.pem"])
	if block == nil || block.Type != "CERTIFICATE" {
		t.Fatalf("CA file is not a PEM certificate: %q", fh.written["ca.pem"])
	}
	if !bytes.Equal(block.Bytes, fh.server.der) {
		t.Fatal("the CA file does not hold the certificate the server serves")
	}
	if perm := fh.perms["ca.pem"]; perm != 0o600 {
		t.Fatalf("CA file mode = %o, want 600", perm)
	}
	if fh.record != nil {
		t.Fatal("a run without -record opened a record file")
	}
	if !fh.server.stopped {
		t.Fatal("the run left the server open")
	}
}

func TestRunRefusesMissingFlagsBeforeStartingAnything(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want error
	}{
		{name: "no CA file", args: []string{"-email", testEmail, "-token", testToken}, want: errNoCAFile},
		{name: "no e-mail", args: []string{"-ca-file", "ca.pem", "-token", testToken}, want: errNoCredentials},
		{name: "no token", args: []string{"-ca-file", "ca.pem", "-email", testEmail}, want: errNoCredentials},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fh := newFakeHost()
			out, _, err := runToEnd(fh, tt.args...)
			if !errors.Is(err, tt.want) {
				t.Fatalf("run error = %v, want %v", err, tt.want)
			}
			if out.Len() != 0 || fh.server.calls != 0 {
				t.Fatalf("a refused run printed %q or started a server", out.String())
			}
		})
	}
}

func TestRunRefusesAnUnknownFlagWithItsDiagnosticOnErrOut(t *testing.T) {
	fh := newFakeHost()
	out, errOut, err := runToEnd(fh, "-nope")
	if err == nil {
		t.Fatal("an unknown flag was accepted")
	}
	if !strings.Contains(errOut.String(), "flag provided but not defined: -nope") {
		t.Fatalf("diagnostic = %q", errOut.String())
	}
	if out.Len() != 0 || fh.server.calls != 0 {
		t.Fatal("a refused run started a server")
	}
}

func TestRunAcceptsTheCredentialsOfAConfigFileInPlaceOfTheFlags(t *testing.T) {
	fh := newFakeHost()
	data, err := json.Marshal(jirastub.Config{Email: "config@example.test", Token: "config-token"})
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	fh.inputs["stub.json"] = data
	l := startRun(fh, "-ca-file", "ca.pem", "-config", "stub.json")
	if rec := call(fh.server.handler, http.MethodGet, "/rest/api/3/myself", basicAuth("config@example.test", "config-token")); rec.Code != http.StatusOK {
		t.Fatalf("the config credentials were refused: %d", rec.Code)
	}
	if rec := call(fh.server.handler, http.MethodGet, "/rest/api/3/myself", basicAuth(testEmail, testToken)); rec.Code != http.StatusUnauthorized {
		t.Fatalf("a flag credential was accepted under -config: %d", rec.Code)
	}
	if err := l.finish(); err != nil {
		t.Fatalf("run: %v", err)
	}
}

func TestRunServesTheDatasetAndTheFaultOfAConfigFile(t *testing.T) {
	fh := newFakeHost()
	data, err := json.Marshal(jirastub.Config{
		Email: "config@example.test", Token: "config-token",
		Fault: &jirastub.Fault{Status: http.StatusServiceUnavailable}, FaultLeft: 1,
	})
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	fh.inputs["stub.json"] = data
	l := startRun(fh, "-ca-file", "ca.pem", "-config", "stub.json")
	auth := basicAuth("config@example.test", "config-token")
	first := call(fh.server.handler, http.MethodGet, "/rest/api/3/myself", auth)
	second := call(fh.server.handler, http.MethodGet, "/rest/api/3/myself", auth)
	if first.Code != http.StatusServiceUnavailable || second.Code != http.StatusOK {
		t.Fatalf("statuses = %d then %d, want 503 then 200 (one faulted reply)", first.Code, second.Code)
	}
	if err := l.finish(); err != nil {
		t.Fatalf("run: %v", err)
	}
}

func TestRunReportsAConfigFileThatCannotBeReadOrParsed(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		noFile  bool
		wantMsg string
	}{
		{name: "missing file", noFile: true, wantMsg: "read the config file"},
		{name: "not JSON", input: []byte("{not json"), wantMsg: "parse the config file"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fh := newFakeHost()
			if !tt.noFile {
				fh.inputs["stub.json"] = tt.input
			}
			_, _, err := runToEnd(fh, "-ca-file", "ca.pem", "-config", "stub.json")
			if err == nil || !strings.Contains(err.Error(), tt.wantMsg) {
				t.Fatalf("run error = %v, want it to mention %q", err, tt.wantMsg)
			}
			if fh.server.calls != 0 {
				t.Fatal("a config that did not load still started a server")
			}
		})
	}
}

func TestRunAppliesTheFaultFlagsToTheirPathOnly(t *testing.T) {
	fh := newFakeHost()
	args := append(caArgs(), "-fault-status", "429", "-fault-path", "/rest/api/3/issue/", "-retry-after", "3")
	l := startRun(fh, args...)
	auth := basicAuth(testEmail, testToken)
	faulted := call(fh.server.handler, http.MethodGet, "/rest/api/3/issue/"+jirastub.IssueKey, auth)
	if faulted.Code != http.StatusTooManyRequests || faulted.Header().Get("Retry-After") != "3" {
		t.Fatalf("issue reply = %d, Retry-After %q; want 429 and 3", faulted.Code, faulted.Header().Get("Retry-After"))
	}
	other := call(fh.server.handler, http.MethodGet, "/rest/api/3/myself", auth)
	if other.Code != http.StatusOK {
		t.Fatalf("myself under an issue fault = %d, want 200", other.Code)
	}
	if err := l.finish(); err != nil {
		t.Fatalf("run: %v", err)
	}
}

func TestRunRecordsEachRequestAsOneJSONLineWithoutTheSecret(t *testing.T) {
	fh := newFakeHost()
	l := startRun(fh, append(caArgs(), "-record", "requests.jsonl")...)
	call(fh.server.handler, http.MethodGet, "/rest/api/3/myself", basicAuth(testEmail, testToken))
	if err := l.finish(); err != nil {
		t.Fatalf("run: %v", err)
	}

	if fh.record == nil {
		t.Fatal("-record opened no record file")
	}
	content := fh.record.String()
	if strings.Count(content, "\n") != 1 {
		t.Fatalf("record holds %d lines, want 1: %q", strings.Count(content, "\n"), content)
	}
	var got jirastub.Request
	if err := json.Unmarshal([]byte(content), &got); err != nil {
		t.Fatalf("record line is not a request: %v", err)
	}
	if got.Method != http.MethodGet || got.Path != "/rest/api/3/myself" || !got.AuthOK {
		t.Fatalf("recorded %+v, want an authenticated GET of myself", got)
	}
	if !got.At.Equal(fixedNow) {
		t.Fatalf("recorded arrival %v, want the injected clock %v", got.At, fixedNow)
	}
	if strings.Contains(content, testToken) || strings.Contains(content, basicAuth(testEmail, testToken)) {
		t.Fatal("the record holds the secret")
	}
	if !fh.record.closed {
		t.Fatal("the run left the record file open")
	}
}

func TestRunReportsARecordFileThatCannotOpen(t *testing.T) {
	fh := newFakeHost()
	fh.createErr = errors.New("record directory is missing")
	_, _, err := runToEnd(fh, append(caArgs(), "-record", "requests.jsonl")...)
	if !errors.Is(err, fh.createErr) || !strings.Contains(err.Error(), "open the record file") {
		t.Fatalf("run error = %v", err)
	}
	if fh.server.calls != 0 {
		t.Fatal("a run with no record file still started a server")
	}
}

func TestRunReleasesTheRecordWhenTheServerCannotStart(t *testing.T) {
	fh := newFakeHost()
	fh.server.err = errors.New("no loopback port")
	_, _, err := runToEnd(fh, append(caArgs(), "-record", "requests.jsonl")...)
	if !errors.Is(err, fh.server.err) || !strings.Contains(err.Error(), "start the TLS server") {
		t.Fatalf("run error = %v", err)
	}
	if fh.record == nil || !fh.record.closed {
		t.Fatal("the record file was left open after the server failed")
	}
}

func TestRunStopsTheServerWhenTheCAFileCannotBeWritten(t *testing.T) {
	fh := newFakeHost()
	fh.writeErr = errors.New("CA directory is read-only")
	_, _, err := runToEnd(fh, append(caArgs(), "-record", "requests.jsonl")...)
	if !errors.Is(err, fh.writeErr) || !strings.Contains(err.Error(), "write the CA file") {
		t.Fatalf("run error = %v", err)
	}
	if !fh.server.stopped {
		t.Fatal("the server was left running after the CA file failed")
	}
	if fh.record == nil || !fh.record.closed {
		t.Fatal("the record file was left open after the CA file failed")
	}
}

func TestRunKeepsServingUntilItsStopSignalArrives(t *testing.T) {
	fh := newFakeHost()
	l := startRun(fh, caArgs()...)
	if rec := call(fh.server.handler, http.MethodGet, "/rest/api/3/myself", basicAuth(testEmail, testToken)); rec.Code != http.StatusOK {
		t.Fatalf("the stub stopped before the stop signal: %d", rec.Code)
	}
	waitsFor := func(want os.Signal) bool {
		return slices.ContainsFunc(l.signals, func(s os.Signal) bool { return s == want })
	}
	if !waitsFor(os.Interrupt) || !waitsFor(syscall.SIGTERM) {
		t.Fatalf("the run waits for %v, want SIGINT and SIGTERM", l.signals)
	}
	if err := l.finish(); err != nil {
		t.Fatalf("run after the stop signal: %v", err)
	}
	if !fh.server.stopped {
		t.Fatal("the stop signal left the server open")
	}
	if !strings.HasPrefix(l.out.String(), "jira stub listening on ") {
		t.Fatalf("listening line = %q", l.out.String())
	}
}

func TestRunMainReturnsZeroOnSuccessAndOneAfterPrintingTheError(t *testing.T) {
	fh := newFakeHost()
	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
	if status := Run(caArgs(), out, errOut, fh.host()); status != 0 {
		t.Fatalf("success status = %d, want 0 (stderr %q)", status, errOut.String())
	}
	if !strings.HasPrefix(out.String(), "jira stub listening on ") {
		t.Fatalf("listening line = %q", out.String())
	}

	out.Reset()
	errOut.Reset()
	if status := Run([]string{"-nope"}, out, errOut, fh.host()); status != 1 {
		t.Fatalf("failure status = %d, want 1", status)
	}
	if !strings.Contains(errOut.String(), "flag provided but not defined") {
		t.Fatalf("failure did not print its error: %q", errOut.String())
	}
}
