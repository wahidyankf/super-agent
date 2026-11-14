package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"sync"
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

// memListener is a listener that opens no socket. Accept blocks until Close.
type memListener struct {
	once sync.Once
	done chan struct{}
}

func newMemListener() *memListener {
	return &memListener{done: make(chan struct{})}
}

func (l *memListener) Accept() (net.Conn, error) {
	<-l.done
	return nil, net.ErrClosed
}

func (l *memListener) Close() error {
	l.once.Do(func() { close(l.done) })
	return nil
}

func (l *memListener) Addr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 1}
}

// isClosed reports whether Close has been called.
func (l *memListener) isClosed() bool {
	select {
	case <-l.done:
		return true
	default:
		return false
	}
}

// memFile is an in-memory record file. Close only marks it closed.
type memFile struct {
	bytes.Buffer
	closed bool
}

func (f *memFile) Close() error {
	f.closed = true
	return nil
}

// fakeServer is the TLS server seam. It starts the handler on an in-memory listener with a self-signed certificate, and it
// keeps what it was given, so a test can call the stub while the command runs.
type fakeServer struct {
	calls    int
	err      error
	handler  http.Handler
	listener *memListener
	der      []byte
}

func (f *fakeServer) serve(h http.Handler) (*httptest.Server, error) {
	f.calls++
	f.handler = h
	if f.err != nil {
		return nil, f.err
	}
	cert := selfSignedCertificate()
	f.der = cert.Certificate[0]
	f.listener = newMemListener()
	srv := &httptest.Server{
		Listener: f.listener,
		Config:   &http.Server{Handler: h},
		TLS:      &tls.Config{Certificates: []tls.Certificate{cert}},
	}
	srv.StartTLS()
	return srv, nil
}

// selfSignedCertificate makes the certificate of the fake server. Its clock is fixedNow, so it needs no real clock.
func selfSignedCertificate() tls.Certificate {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(fmt.Sprintf("generate the fake key: %v", err))
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "jirastubcmd-fake"},
		NotBefore:    fixedNow.Add(-time.Hour),
		NotAfter:     fixedNow.Add(time.Hour),
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1)},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		panic(fmt.Sprintf("sign the fake certificate: %v", err))
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
}

// fakeSystem is the fake seam set of one test. Its files live in maps, and its clock is fixedNow.
type fakeSystem struct {
	inputs    map[string][]byte
	written   map[string][]byte
	perms     map[string]os.FileMode
	writeErr  error
	createErr error
	record    *memFile
	server    *fakeServer
}

func newFakeSystem() *fakeSystem {
	return &fakeSystem{
		inputs:  map[string][]byte{},
		written: map[string][]byte{},
		perms:   map[string]os.FileMode{},
		server:  &fakeServer{},
	}
}

// system returns the seams of the command. Its signal wait is already done, so run returns once it has started the stub.
func (f *fakeSystem) system() system {
	return system{
		readFile: func(name string) ([]byte, error) {
			data, ok := f.inputs[name]
			if !ok {
				return nil, fmt.Errorf("no fake file named %s", name)
			}
			return data, nil
		},
		writeFile: func(name string, data []byte, perm os.FileMode) error {
			if f.writeErr != nil {
				return f.writeErr
			}
			f.written[name] = data
			f.perms[name] = perm
			return nil
		},
		createFile: func(string) (io.WriteCloser, error) {
			if f.createErr != nil {
				return nil, f.createErr
			}
			f.record = &memFile{}
			return f.record, nil
		},
		now:   func() time.Time { return fixedNow },
		serve: f.server.serve,
		notify: func(parent context.Context, _ ...os.Signal) (context.Context, context.CancelFunc) {
			ctx, cancel := context.WithCancel(parent)
			cancel()
			return ctx, cancel
		},
	}
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
func startRun(fs *fakeSystem, args ...string) *live {
	l := &live{out: &bytes.Buffer{}, errOut: &bytes.Buffer{}, done: make(chan error, 1)}
	waiting := make(chan struct{})
	sys := fs.system()
	sys.notify = func(parent context.Context, signals ...os.Signal) (context.Context, context.CancelFunc) {
		ctx, cancel := context.WithCancel(parent)
		l.signals = signals
		l.stop = cancel
		close(waiting)
		return ctx, cancel
	}
	go func() {
		l.done <- run(context.Background(), args, l.out, l.errOut, sys)
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

// runToEnd runs the command with args and the fake system, which stops at once, and returns its output and error.
func runToEnd(fs *fakeSystem, args ...string) (*bytes.Buffer, *bytes.Buffer, error) {
	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
	err := run(context.Background(), args, out, errOut, fs.system())
	return out, errOut, err
}

func TestRunPrintsTheListeningLineAndWritesTheCertificate(t *testing.T) {
	fs := newFakeSystem()
	out, errOut, err := runToEnd(fs, caArgs()...)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if got, want := out.String(), "jira stub listening on https://127.0.0.1:1\n"; got != want {
		t.Fatalf("listening line = %q, want %q", got, want)
	}
	if strings.Contains(out.String()+errOut.String(), testToken) {
		t.Fatal("the command output shows the token")
	}
	block, _ := pem.Decode(fs.written["ca.pem"])
	if block == nil || block.Type != "CERTIFICATE" {
		t.Fatalf("CA file is not a PEM certificate: %q", fs.written["ca.pem"])
	}
	if !bytes.Equal(block.Bytes, fs.server.der) {
		t.Fatal("the CA file does not hold the certificate the server serves")
	}
	if perm := fs.perms["ca.pem"]; perm != 0o600 {
		t.Fatalf("CA file mode = %o, want 600", perm)
	}
	if fs.record != nil {
		t.Fatal("a run without -record opened a record file")
	}
	if !fs.server.listener.isClosed() {
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
			fs := newFakeSystem()
			out, _, err := runToEnd(fs, tt.args...)
			if !errors.Is(err, tt.want) {
				t.Fatalf("run error = %v, want %v", err, tt.want)
			}
			if out.Len() != 0 || fs.server.calls != 0 {
				t.Fatalf("a refused run printed %q or started a server", out.String())
			}
		})
	}
}

func TestRunRefusesAnUnknownFlagWithItsDiagnosticOnErrOut(t *testing.T) {
	fs := newFakeSystem()
	out, errOut, err := runToEnd(fs, "-nope")
	if err == nil {
		t.Fatal("an unknown flag was accepted")
	}
	if !strings.Contains(errOut.String(), "flag provided but not defined: -nope") {
		t.Fatalf("diagnostic = %q", errOut.String())
	}
	if out.Len() != 0 || fs.server.calls != 0 {
		t.Fatal("a refused run started a server")
	}
}

func TestRunAcceptsTheCredentialsOfAConfigFileInPlaceOfTheFlags(t *testing.T) {
	fs := newFakeSystem()
	data, err := json.Marshal(jirastub.Config{Email: "config@example.test", Token: "config-token"})
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	fs.inputs["stub.json"] = data
	l := startRun(fs, "-ca-file", "ca.pem", "-config", "stub.json")
	if rec := call(fs.server.handler, http.MethodGet, "/rest/api/3/myself", basicAuth("config@example.test", "config-token")); rec.Code != http.StatusOK {
		t.Fatalf("the config credentials were refused: %d", rec.Code)
	}
	if rec := call(fs.server.handler, http.MethodGet, "/rest/api/3/myself", basicAuth(testEmail, testToken)); rec.Code != http.StatusUnauthorized {
		t.Fatalf("a flag credential was accepted under -config: %d", rec.Code)
	}
	if err := l.finish(); err != nil {
		t.Fatalf("run: %v", err)
	}
}

func TestRunServesTheDatasetAndTheFaultOfAConfigFile(t *testing.T) {
	fs := newFakeSystem()
	data, err := json.Marshal(jirastub.Config{
		Email: "config@example.test", Token: "config-token",
		Fault: &jirastub.Fault{Status: http.StatusServiceUnavailable}, FaultLeft: 1,
	})
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	fs.inputs["stub.json"] = data
	l := startRun(fs, "-ca-file", "ca.pem", "-config", "stub.json")
	auth := basicAuth("config@example.test", "config-token")
	first := call(fs.server.handler, http.MethodGet, "/rest/api/3/myself", auth)
	second := call(fs.server.handler, http.MethodGet, "/rest/api/3/myself", auth)
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
			fs := newFakeSystem()
			if !tt.noFile {
				fs.inputs["stub.json"] = tt.input
			}
			_, _, err := runToEnd(fs, "-ca-file", "ca.pem", "-config", "stub.json")
			if err == nil || !strings.Contains(err.Error(), tt.wantMsg) {
				t.Fatalf("run error = %v, want it to mention %q", err, tt.wantMsg)
			}
			if fs.server.calls != 0 {
				t.Fatal("a config that did not load still started a server")
			}
		})
	}
}

func TestRunAppliesTheFaultFlagsToTheirPathOnly(t *testing.T) {
	fs := newFakeSystem()
	args := append(caArgs(), "-fault-status", "429", "-fault-path", "/rest/api/3/issue/", "-retry-after", "3")
	l := startRun(fs, args...)
	auth := basicAuth(testEmail, testToken)
	faulted := call(fs.server.handler, http.MethodGet, "/rest/api/3/issue/"+jirastub.IssueKey, auth)
	if faulted.Code != http.StatusTooManyRequests || faulted.Header().Get("Retry-After") != "3" {
		t.Fatalf("issue reply = %d, Retry-After %q; want 429 and 3", faulted.Code, faulted.Header().Get("Retry-After"))
	}
	other := call(fs.server.handler, http.MethodGet, "/rest/api/3/myself", auth)
	if other.Code != http.StatusOK {
		t.Fatalf("myself under an issue fault = %d, want 200", other.Code)
	}
	if err := l.finish(); err != nil {
		t.Fatalf("run: %v", err)
	}
}

func TestRunRecordsEachRequestAsOneJSONLineWithoutTheSecret(t *testing.T) {
	fs := newFakeSystem()
	l := startRun(fs, append(caArgs(), "-record", "requests.jsonl")...)
	call(fs.server.handler, http.MethodGet, "/rest/api/3/myself", basicAuth(testEmail, testToken))
	if err := l.finish(); err != nil {
		t.Fatalf("run: %v", err)
	}

	if fs.record == nil {
		t.Fatal("-record opened no record file")
	}
	content := fs.record.String()
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
	if !fs.record.closed {
		t.Fatal("the run left the record file open")
	}
}

func TestRunReportsARecordFileThatCannotOpen(t *testing.T) {
	fs := newFakeSystem()
	fs.createErr = errors.New("record directory is missing")
	_, _, err := runToEnd(fs, append(caArgs(), "-record", "requests.jsonl")...)
	if !errors.Is(err, fs.createErr) || !strings.Contains(err.Error(), "open the record file") {
		t.Fatalf("run error = %v", err)
	}
	if fs.server.calls != 0 {
		t.Fatal("a run with no record file still started a server")
	}
}

func TestRunReleasesTheRecordWhenTheServerCannotStart(t *testing.T) {
	fs := newFakeSystem()
	fs.server.err = errors.New("no loopback port")
	_, _, err := runToEnd(fs, append(caArgs(), "-record", "requests.jsonl")...)
	if !errors.Is(err, fs.server.err) || !strings.Contains(err.Error(), "start the TLS server") {
		t.Fatalf("run error = %v", err)
	}
	if fs.record == nil || !fs.record.closed {
		t.Fatal("the record file was left open after the server failed")
	}
}

func TestRunStopsTheServerWhenTheCAFileCannotBeWritten(t *testing.T) {
	fs := newFakeSystem()
	fs.writeErr = errors.New("CA directory is read-only")
	_, _, err := runToEnd(fs, append(caArgs(), "-record", "requests.jsonl")...)
	if !errors.Is(err, fs.writeErr) || !strings.Contains(err.Error(), "write the CA file") {
		t.Fatalf("run error = %v", err)
	}
	if !fs.server.listener.isClosed() {
		t.Fatal("the server was left running after the CA file failed")
	}
	if fs.record == nil || !fs.record.closed {
		t.Fatal("the record file was left open after the CA file failed")
	}
}

func TestRunKeepsServingUntilItsStopSignalArrives(t *testing.T) {
	fs := newFakeSystem()
	l := startRun(fs, caArgs()...)
	if rec := call(fs.server.handler, http.MethodGet, "/rest/api/3/myself", basicAuth(testEmail, testToken)); rec.Code != http.StatusOK {
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
	if !fs.server.listener.isClosed() {
		t.Fatal("the stop signal left the server open")
	}
	if !strings.HasPrefix(l.out.String(), "jira stub listening on ") {
		t.Fatalf("listening line = %q", l.out.String())
	}
}

func TestRunMainReturnsZeroOnSuccessAndOneAfterPrintingTheError(t *testing.T) {
	fs := newFakeSystem()
	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
	if status := runMain(caArgs(), out, errOut, fs.system()); status != 0 {
		t.Fatalf("success status = %d, want 0 (stderr %q)", status, errOut.String())
	}
	if !strings.HasPrefix(out.String(), "jira stub listening on ") {
		t.Fatalf("listening line = %q", out.String())
	}

	out.Reset()
	errOut.Reset()
	if status := runMain([]string{"-nope"}, out, errOut, fs.system()); status != 1 {
		t.Fatalf("failure status = %d, want 1", status)
	}
	if !strings.Contains(errOut.String(), "flag provided but not defined") {
		t.Fatalf("failure did not print its error: %q", errOut.String())
	}
}
