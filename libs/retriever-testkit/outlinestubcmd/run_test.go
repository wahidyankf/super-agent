package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-testkit/outlinestub"
)

// commandClock is the clock every run in this file uses. The client that trusts the stub reads the same clock.
var commandClock = time.Date(2026, time.November, 14, 9, 0, 0, 0, time.UTC)

func commandNow() time.Time { return commandClock }

// commandWait bounds every wait on a run, so a hang fails the test instead of stalling the suite.
const commandWait = 5 * time.Second

// listeningLine is the first line of a run that listens on the pipe listener below.
const listeningLine = "outline stub listening on https://127.0.0.1:4443"

// fakeFS is the file system a test gives the command. Reads come from its files, writes are recorded, and no disk is
// touched.
type fakeFS struct {
	files    map[string][]byte
	readErr  error
	writeErr error
	written  map[string]writtenFile
}

// writtenFile is one file the command wrote, with the mode it asked for.
type writtenFile struct {
	data []byte
	perm fs.FileMode
}

func newFakeFS() *fakeFS {
	return &fakeFS{files: map[string][]byte{}, written: map[string]writtenFile{}}
}

func (f *fakeFS) readFile(name string) ([]byte, error) {
	if f.readErr != nil {
		return nil, f.readErr
	}
	data, ok := f.files[name]
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	return data, nil
}

func (f *fakeFS) writeFile(name string, data []byte, perm fs.FileMode) error {
	if f.writeErr != nil {
		return &fs.PathError{Op: "open", Path: name, Err: f.writeErr}
	}
	f.written[name] = writtenFile{data: append([]byte(nil), data...), perm: perm}
	return nil
}

// pipeListener is a listener with no socket. A client dials it, and Accept hands the server end of an in-memory pipe
// to the server.
type pipeListener struct {
	conns chan net.Conn
	done  chan struct{}
	once  sync.Once
}

func newPipeListener() *pipeListener {
	return &pipeListener{conns: make(chan net.Conn), done: make(chan struct{})}
}

func (l *pipeListener) Accept() (net.Conn, error) {
	select {
	case conn := <-l.conns:
		return conn, nil
	case <-l.done:
		return nil, net.ErrClosed
	}
}

func (l *pipeListener) Close() error {
	l.once.Do(func() { close(l.done) })
	return nil
}

func (l *pipeListener) Addr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 4443}
}

func (l *pipeListener) closed() bool {
	select {
	case <-l.done:
		return true
	default:
		return false
	}
}

// dial returns the client end of a new in-memory connection to the listener.
func (l *pipeListener) dial(ctx context.Context, network, address string) (net.Conn, error) {
	client, server := net.Pipe()
	select {
	case l.conns <- server:
		return client, nil
	case <-l.done:
		client.Close()
		server.Close()
		return nil, net.ErrClosed
	case <-ctx.Done():
		client.Close()
		server.Close()
		return nil, ctx.Err()
	}
}

// listenSpy is the listen seam of one run. It counts the calls, and it hands out l, or it refuses with fail.
type listenSpy struct {
	calls int
	l     *pipeListener
	fail  error
}

func (s *listenSpy) listen(network, address string) (net.Listener, error) {
	s.calls++
	if s.fail != nil {
		return nil, s.fail
	}
	return s.l, nil
}

// fakeDeps is the set of effects one run gets from the test.
func fakeDeps(files *fakeFS, listen func(string, string) (net.Listener, error), ctx context.Context, cancel context.CancelFunc) deps {
	return deps{
		readFile:  files.readFile,
		writeFile: files.writeFile,
		listen:    listen,
		now:       commandNow,
		notify: func(context.Context, ...os.Signal) (context.Context, context.CancelFunc) {
			return ctx, cancel
		},
	}
}

// command is one run of the command in its own goroutine, with its output read through a pipe and its effects fakes.
type command struct {
	cancel   context.CancelFunc
	done     chan int
	stdout   *bufio.Reader
	stderr   *bytes.Buffer
	files    *fakeFS
	listener *pipeListener
}

// startCommand runs the command with args in a goroutine. The run stops when the test calls stop, or when the test ends.
func startCommand(t *testing.T, files *fakeFS, args ...string) *command {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	l := newPipeListener()
	spy := &listenSpy{l: l}
	reader, writer := io.Pipe()
	stderr := &bytes.Buffer{}
	done := make(chan int, 1)
	go func() {
		code := run(args, stdio{out: writer, err: stderr}, fakeDeps(files, spy.listen, ctx, cancel))
		writer.Close()
		done <- code
	}()
	return &command{cancel: cancel, done: done, stdout: bufio.NewReader(reader), stderr: stderr, files: files, listener: l}
}

// firstLine returns the first line the command prints, failing the test if none arrives in time.
func (c *command) firstLine(t *testing.T) string {
	t.Helper()
	return readFirstLine(t, c.stdout)
}

// readFirstLine returns the first line on r, failing the test if none arrives in time.
func readFirstLine(t *testing.T, r *bufio.Reader) string {
	t.Helper()
	got := make(chan string, 1)
	failed := make(chan error, 1)
	go func() {
		line, err := r.ReadString('\n')
		if err != nil {
			failed <- err
			return
		}
		got <- line
	}()
	select {
	case line := <-got:
		return strings.TrimSuffix(line, "\n")
	case err := <-failed:
		t.Fatalf("read the command's first line: %v", err)
	}
	return ""
}

// stop cancels the run and returns its exit code, failing the test if run does not return in time.
func (c *command) stop(t *testing.T) int {
	t.Helper()
	c.cancel()
	return <-c.done
}

// trustingClient returns an HTTP client that dials the pipe listener and trusts only the CA PEM. It reads the same
// clock as the stub, so the certificate's validity is checked against the injected instant and not the real clock.
func pipeTLSClient(t *testing.T, l *pipeListener, caPEM []byte) *http.Client {
	t.Helper()
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		t.Fatal("the CA file holds no certificate")
	}
	transport := &http.Transport{
		DialContext:     l.dial,
		TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12, Time: commandNow},
	}
	t.Cleanup(transport.CloseIdleConnections)
	return &http.Client{Transport: transport}
}

// postJSON sends one JSON POST and returns the status and body.
func postJSON(t *testing.T, client *http.Client, url, body string) (int, string) {
	t.Helper()
	resp, err := client.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read the body of %s: %v", url, err)
	}
	return resp.StatusCode, string(raw)
}

// caOf returns the CA file the run wrote, failing the test when it wrote none.
func caOf(t *testing.T, files *fakeFS) []byte {
	t.Helper()
	ca, ok := files.written["ca.pem"]
	if !ok {
		t.Fatal("the run wrote no CA file")
	}
	return ca.data
}

func TestRunRefusesEachBadStartWithItsMessageAndListensNothing(t *testing.T) {
	cases := []struct {
		name        string
		files       func() *fakeFS
		args        []string
		listenFail  error
		wantMessage string
		wantListens int
	}{
		{
			name:        "no CA file",
			files:       newFakeFS,
			args:        []string{},
			wantMessage: "the -ca-file flag is required",
		},
		{
			name:        "a flag the command does not have",
			files:       newFakeFS,
			args:        []string{"-no-such-flag", "-ca-file", "ca.pem"},
			wantMessage: "flag provided but not defined: -no-such-flag",
		},
		{
			name:        "a collection with no name",
			files:       newFakeFS,
			args:        []string{"-ca-file", "ca.pem", "-collection", "c1="},
			wantMessage: `-collection wants id=name, got "c1="`,
		},
		{
			name:        "two faults at once",
			files:       newFakeFS,
			args:        []string{"-ca-file", "ca.pem", "-list-status", "500", "-document-status", "404"},
			wantMessage: errTwoFaults.Error(),
		},
		{
			name: "a documents file that cannot be read",
			files: func() *fakeFS {
				f := newFakeFS()
				f.readErr = errors.New("disk offline")
				return f
			},
			args:        []string{"-ca-file", "ca.pem", "-documents-file", "docs.json"},
			wantMessage: "read the documents file: disk offline",
		},
		{
			name: "a documents file that is not JSON",
			files: func() *fakeFS {
				f := newFakeFS()
				f.files["docs.json"] = []byte("not json")
				return f
			},
			args:        []string{"-ca-file", "ca.pem", "-documents-file", "docs.json"},
			wantMessage: "parse the documents file",
		},
		{
			name:        "a listener that cannot open",
			files:       newFakeFS,
			args:        []string{"-ca-file", "ca.pem"},
			listenFail:  errors.New("no loopback port"),
			wantMessage: "start the TLS stub: no loopback port",
			wantListens: 1,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			files := tc.files()
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			spy := &listenSpy{l: newPipeListener(), fail: tc.listenFail}
			var stdout, stderr bytes.Buffer

			code := run(tc.args, stdio{out: &stdout, err: &stderr}, fakeDeps(files, spy.listen, ctx, cancel))

			if code != 1 {
				t.Fatalf("exit code = %d, want 1", code)
			}
			if !strings.Contains(stderr.String(), tc.wantMessage) {
				t.Fatalf("stderr = %q, want it to hold %q", stderr.String(), tc.wantMessage)
			}
			if stdout.Len() != 0 {
				t.Fatalf("stdout = %q, want no listening line before a refusal", stdout.String())
			}
			if spy.calls != tc.wantListens {
				t.Fatalf("the command listened %d times, want %d", spy.calls, tc.wantListens)
			}
			if len(files.written) != 0 {
				t.Fatalf("the command wrote %v before refusing, want no file", files.written)
			}
		})
	}
}

func TestRunClosesItsListenerWhenTheCAFileCannotBeWritten(t *testing.T) {
	files := newFakeFS()
	files.writeErr = fs.ErrPermission
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	spy := &listenSpy{l: newPipeListener()}
	var stdout, stderr bytes.Buffer

	code := run([]string{"-ca-file", "ca.pem"}, stdio{out: &stdout, err: &stderr}, fakeDeps(files, spy.listen, ctx, cancel))

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if want := "write the CA file: open ca.pem: permission denied"; !strings.Contains(stderr.String(), want) {
		t.Fatalf("stderr = %q, want it to hold %q", stderr.String(), want)
	}
	if !spy.l.closed() {
		t.Fatal("the listener is still open after the CA file could not be written")
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want no listening line for a stub that never served", stdout.String())
	}
}

func TestRunListensWritesTheCAFileAndStopsWhenItsContextEnds(t *testing.T) {
	files := newFakeFS()
	c := startCommand(t, files, "-ca-file", "ca.pem", "-collection", "c1=Engineering")

	if line := c.firstLine(t); line != listeningLine {
		t.Fatalf("first line = %q, want %q", line, listeningLine)
	}
	ca := files.written["ca.pem"]
	if ca.perm != 0o600 {
		t.Fatalf("the CA file has mode %o, want 0600", ca.perm)
	}
	block, _ := pem.Decode(ca.data)
	if block == nil || block.Type != "CERTIFICATE" {
		t.Fatalf("the CA file is not one PEM certificate: %q", ca.data)
	}

	if code := c.stop(t); code != 0 {
		t.Fatalf("exit code after the stop = %d, want 0", code)
	}
	if !c.listener.closed() {
		t.Fatal("the listener is still open after the command stopped")
	}
	if c.stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want nothing on a clean stop", c.stderr.String())
	}
}

func TestRunAnswersOverTLSWithTheCollectionsItWasGiven(t *testing.T) {
	files := newFakeFS()
	c := startCommand(t, files, "-ca-file", "ca.pem", "-collection", "c1=Engineering", "-collection", "c2=Finance")
	c.firstLine(t)
	client := pipeTLSClient(t, c.listener, caOf(t, files))

	status, body := postJSON(t, client, "https://127.0.0.1:4443/api/collections.list", `{"offset":0,"limit":25}`)

	if status != http.StatusOK {
		t.Fatalf("collections.list status = %d, want 200", status)
	}
	if !strings.Contains(body, "Engineering") || !strings.Contains(body, "Finance") {
		t.Fatalf("collections.list body lacks the configured collections: %s", body)
	}
	if code := c.stop(t); code != 0 {
		t.Fatalf("exit code after the stop = %d, want 0", code)
	}
}

func TestRunStopsAnsweringWhenItsContextEnds(t *testing.T) {
	files := newFakeFS()
	c := startCommand(t, files, "-ca-file", "ca.pem")
	c.firstLine(t)
	client := pipeTLSClient(t, c.listener, caOf(t, files))
	if status, _ := postJSON(t, client, "https://127.0.0.1:4443/api/auth.info", `{}`); status != http.StatusOK {
		t.Fatalf("auth.info before the stop = %d, want 200", status)
	}

	c.stop(t)

	resp, err := client.Post("https://127.0.0.1:4443/api/auth.info", "application/json", strings.NewReader(`{}`))
	if err == nil {
		resp.Body.Close()
		t.Fatal("the stub still answered after the command stopped")
	}
}

func TestRunPassesTheListStatusToCollectionsList(t *testing.T) {
	files := newFakeFS()
	c := startCommand(t, files, "-ca-file", "ca.pem", "-list-status", "401")
	c.firstLine(t)
	client := pipeTLSClient(t, c.listener, caOf(t, files))

	if status, _ := postJSON(t, client, "https://127.0.0.1:4443/api/collections.list", `{"offset":0,"limit":25}`); status != http.StatusUnauthorized {
		t.Fatalf("collections.list status = %d, want 401 from the -list-status fault", status)
	}
	c.stop(t)
}

func TestRunPassesTheDocumentStatusAndTheDocumentsFileToDocumentsInfo(t *testing.T) {
	cases := []struct {
		name       string
		args       []string
		wantStatus int
		wantTitle  string
	}{
		{
			name:       "the document status answers a document that exists",
			args:       []string{"-document-status", "404"},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "a document with no fault is served from the documents file",
			args:       nil,
			wantStatus: http.StatusOK,
			wantTitle:  "Setup",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			files := newFakeFS()
			files.files["docs.json"] = []byte(`[{"ID":"doc-2","CollectionID":"c1","ParentID":"","Title":"Setup","URL":"/doc/doc-2","Markdown":"Run the installer."}]`)
			args := append([]string{"-ca-file", "ca.pem", "-documents-file", "docs.json"}, tc.args...)
			c := startCommand(t, files, args...)
			c.firstLine(t)
			client := pipeTLSClient(t, c.listener, caOf(t, files))

			status, body := postJSON(t, client, "https://127.0.0.1:4443/api/documents.info", `{"id":"doc-2"}`)

			if status != tc.wantStatus {
				t.Fatalf("documents.info status = %d, want %d: %s", status, tc.wantStatus, body)
			}
			if tc.wantTitle != "" {
				var reply struct {
					Data struct {
						Title string `json:"title"`
					} `json:"data"`
				}
				if err := json.Unmarshal([]byte(body), &reply); err != nil {
					t.Fatalf("documents.info body is not JSON: %v", err)
				}
				if reply.Data.Title != tc.wantTitle {
					t.Fatalf("title = %q, want %q", reply.Data.Title, tc.wantTitle)
				}
			}
			c.stop(t)
		})
	}
}

func TestReadDocumentsReadsEveryFieldOfEachDocument(t *testing.T) {
	files := newFakeFS()
	files.files["docs.json"] = []byte(`[{"ID":"doc-2","CollectionID":"c1","ParentID":"doc-1","Title":"Setup","URL":"/doc/doc-2","Markdown":"Run the installer."}]`)

	docs, err := readDocuments(files.readFile, "docs.json")
	if err != nil {
		t.Fatalf("readDocuments: %v", err)
	}

	want := outlinestub.Document{ID: "doc-2", CollectionID: "c1", ParentID: "doc-1", Title: "Setup", URL: "/doc/doc-2", Markdown: "Run the installer."}
	if len(docs) != 1 || docs[0] != want {
		t.Fatalf("documents = %v, want [%v]", docs, want)
	}
}

func TestReadDocumentsWrapsAReadFailureWithItsCause(t *testing.T) {
	files := newFakeFS()
	files.readErr = fs.ErrPermission

	docs, err := readDocuments(files.readFile, "docs.json")

	if docs != nil {
		t.Fatalf("readDocuments returned %v with a read error", docs)
	}
	if !errors.Is(err, fs.ErrPermission) || !strings.HasPrefix(err.Error(), "read the documents file: ") {
		t.Fatalf("readDocuments returned %v, want the read cause under its context", err)
	}
}

// installProcess replaces the process of main for one test, and it puts the real one back when the test ends.
func installProcess(t *testing.T, p process) {
	t.Helper()
	saved := proc
	t.Cleanup(func() { proc = saved })
	proc = p
}

func TestMainRunsTheProcessCommandAndExitsWithItsCode(t *testing.T) {
	files := newFakeFS()
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	spy := &listenSpy{l: newPipeListener()}
	reader, writer := io.Pipe()
	exits := make(chan int, 1)
	installProcess(t, process{
		args:    []string{"-ca-file", "ca.pem", "-collection", "c1=Engineering"},
		std:     stdio{out: writer, err: io.Discard},
		exit:    func(code int) { exits <- code },
		effects: fakeDeps(files, spy.listen, ctx, cancel),
	})

	go main()
	if line := readFirstLine(t, bufio.NewReader(reader)); line != listeningLine {
		t.Fatalf("first line = %q, want %q", line, listeningLine)
	}
	cancel()

	if code := <-exits; code != 0 {
		t.Fatalf("main exited %d, want 0 after a stop signal", code)
	}
	writer.Close()
	if !spy.l.closed() {
		t.Fatal("the listener is still open after main exited")
	}
	if _, ok := files.written["ca.pem"]; !ok {
		t.Fatal("main wrote no CA file for the process arguments")
	}
}

func TestMainExitsWithOneAndPrintsARefusalWhenTheArgumentsAreBad(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	var stdout, stderr bytes.Buffer
	var codes []int
	installProcess(t, process{
		args:    nil,
		std:     stdio{out: &stdout, err: &stderr},
		exit:    func(code int) { codes = append(codes, code) },
		effects: fakeDeps(newFakeFS(), (&listenSpy{l: newPipeListener()}).listen, ctx, cancel),
	})

	main()

	if len(codes) != 1 || codes[0] != 1 {
		t.Fatalf("main exit codes = %v, want [1]", codes)
	}
	if got, want := stderr.String(), errNoCAFile.Error()+"\n"; got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want nothing for a refused run", stdout.String())
	}
}

func TestStartRefusesTwoFaultsBeforeItListens(t *testing.T) {
	spy := &listenSpy{l: newPipeListener()}
	files := newFakeFS()

	server, err := start(config{caFile: "ca.pem", listStatus: 500, documentStatus: 404},
		deps{readFile: files.readFile, writeFile: files.writeFile, listen: spy.listen, now: commandNow})

	if server != nil || !errors.Is(err, errTwoFaults) {
		t.Fatalf("start returned %v and %v, want errTwoFaults and no server", server, err)
	}
	if spy.calls != 0 {
		t.Fatalf("start listened %d times for two faults, want 0", spy.calls)
	}
}
