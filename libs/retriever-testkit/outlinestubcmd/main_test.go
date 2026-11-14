//go:build integration

package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-testkit/outlinestub"
)

// postList sends one collections.list request over the server's TLS listener and returns the status and body.
func postList(t *testing.T, client *http.Client, base string) (int, string) {
	t.Helper()
	resp, err := client.Post(base+"/api/collections.list", "application/json", strings.NewReader(`{"offset":0,"limit":25}`))
	if err != nil {
		t.Fatalf("POST collections.list: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read collections.list body: %v", err)
	}
	return resp.StatusCode, string(body)
}

// postInfo sends one documents.info request for id over the server's TLS listener and returns the status and body.
func postInfo(t *testing.T, client *http.Client, base, id string) (int, string) {
	t.Helper()
	resp, err := client.Post(base+"/api/documents.info", "application/json", strings.NewReader(`{"id":"`+id+`"}`))
	if err != nil {
		t.Fatalf("POST documents.info: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read documents.info body: %v", err)
	}
	return resp.StatusCode, string(body)
}

// trustingClient returns an HTTP client that trusts only the certificate the command wrote to caFile.
func trustingClient(t *testing.T, caFile string) *http.Client {
	t.Helper()
	pemBytes, err := os.ReadFile(caFile)
	if err != nil {
		t.Fatalf("read CA file: %v", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pemBytes) {
		t.Fatal("the CA file holds no certificate")
	}
	return &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}}}
}

func TestStartServesTheCollectionsOverTLSAndWritesTheCertificate(t *testing.T) {
	caFile := filepath.Join(t.TempDir(), "ca.pem")
	server, err := start(config{
		caFile:      caFile,
		collections: []outlinestub.Collection{{ID: "c1", Name: "Engineering"}, {ID: "c2", Name: "Finance"}},
	}, system)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(server.Close)

	status, body := postList(t, trustingClient(t, caFile), server.URL)

	if status != http.StatusOK {
		t.Fatalf("collections.list status = %d, want 200", status)
	}
	if !strings.Contains(body, "Engineering") || !strings.Contains(body, "Finance") {
		t.Fatalf("collections.list body %q lacks the configured collections", body)
	}
}

func TestStartAnswersTheListWithTheConfiguredStatusWhenAFaultIsSet(t *testing.T) {
	caFile := filepath.Join(t.TempDir(), "ca.pem")
	server, err := start(config{caFile: caFile, listStatus: http.StatusUnauthorized}, system)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(server.Close)

	status, _ := postList(t, trustingClient(t, caFile), server.URL)

	if status != http.StatusUnauthorized {
		t.Fatalf("collections.list status = %d, want 401", status)
	}
}

func TestStartAnswersDocumentInfoWithTheConfiguredStatusWhenAFaultIsSet(t *testing.T) {
	caFile := filepath.Join(t.TempDir(), "ca.pem")
	server, err := start(config{caFile: caFile, documentStatus: http.StatusNotFound}, system)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(server.Close)

	resp, err := trustingClient(t, caFile).Post(server.URL+"/api/documents.info", "application/json", strings.NewReader(`{"id":"doc-2"}`))
	if err != nil {
		t.Fatalf("POST documents.info: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("documents.info status = %d, want 404", resp.StatusCode)
	}
}

func TestStartRefusesTwoFaultsAtOnce(t *testing.T) {
	_, err := start(config{caFile: filepath.Join(t.TempDir(), "ca.pem"), listStatus: 500, documentStatus: 404}, system)
	if !errors.Is(err, errTwoFaults) {
		t.Fatalf("start with two faults returned %v, want errTwoFaults", err)
	}
}

func TestStartRefusesACAFileThatCannotBeWritten(t *testing.T) {
	caFile := filepath.Join(t.TempDir(), "missing-dir", "ca.pem")
	server, err := start(config{caFile: caFile}, system)
	if server != nil {
		server.Close()
	}
	if err == nil {
		t.Fatal("start with an unwritable CA file returned no error")
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("start returned %v, want a not-exist error from writing the CA file", err)
	}
	var pathErr *fs.PathError
	if !errors.As(err, &pathErr) || pathErr.Path != caFile {
		t.Fatalf("start returned %v, want a PathError naming %q", err, caFile)
	}
}

// listeningPrefix starts the line the command prints once it listens.
const listeningPrefix = "outline stub listening on "

// stubWait bounds every wait on a run, so a hang fails the test instead of stalling the suite.
const stubWait = 5 * time.Second

// stubRun is one run of the command in its own goroutine, with its output read through a pipe.
type stubRun struct {
	cancel context.CancelFunc
	done   chan int
	output *bufio.Reader
}

// startStub runs the command with args in a goroutine. The run stops when the test cancels it, which the test does
// when it ends. Everything else is the real effects: real files, a real loopback port, and the real clock.
func startStub(t *testing.T, args ...string) *stubRun {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	reader, writer := io.Pipe()
	d := system
	d.notify = func(context.Context, ...os.Signal) (context.Context, context.CancelFunc) { return ctx, cancel }
	done := make(chan int, 1)
	go func() {
		code := run(args, stdio{out: writer, err: io.Discard}, d)
		writer.Close()
		done <- code
	}()
	return &stubRun{cancel: cancel, done: done, output: bufio.NewReader(reader)}
}

// readLine returns the first line the command prints, failing the test if none arrives in time.
func (s *stubRun) readLine(t *testing.T) string {
	t.Helper()
	type read struct {
		line string
		err  error
	}
	got := make(chan read, 1)
	go func() {
		line, err := s.output.ReadString('\n')
		got <- read{line, err}
	}()
	select {
	case r := <-got:
		if r.err != nil {
			t.Fatalf("read the command's first line: %v", r.err)
		}
		return strings.TrimSuffix(r.line, "\n")
	case <-time.After(stubWait):
		t.Fatalf("the command printed no line within %v", stubWait)
		return ""
	}
}

// wait returns the exit code run returned, failing the test if run does not return in time.
func (s *stubRun) wait(t *testing.T) int {
	t.Helper()
	select {
	case code := <-s.done:
		return code
	case <-time.After(stubWait):
		t.Fatalf("the command did not return within %v", stubWait)
		return -1
	}
}

// listeningURL returns the base URL in the listening line, failing the test if the line lacks the prefix.
func listeningURL(t *testing.T, line string) string {
	t.Helper()
	rawURL, found := strings.CutPrefix(line, listeningPrefix)
	if !found {
		t.Fatalf("first line %q lacks the prefix %q", line, listeningPrefix)
	}
	return rawURL
}

func TestRunStopsWhenItsContextIsCancelled(t *testing.T) {
	caFile := filepath.Join(t.TempDir(), "ca.pem")
	stub := startStub(t, "-ca-file", caFile, "-collection", "c1=Engineering")
	base := listeningURL(t, stub.readLine(t))

	stub.cancel()

	if code := stub.wait(t); code != 0 {
		t.Fatalf("run after its context was cancelled exited %d, want 0", code)
	}
	resp, err := trustingClient(t, caFile).Post(base+"/api/collections.list", "application/json", strings.NewReader(`{"offset":0,"limit":25}`))
	if err == nil {
		resp.Body.Close()
		t.Fatal("the stub still answered after run returned")
	}
}

func TestRunAnnouncesALoopbackAddressAndPort(t *testing.T) {
	caFile := filepath.Join(t.TempDir(), "ca.pem")
	stub := startStub(t, "-ca-file", caFile)
	rawURL := listeningURL(t, stub.readLine(t))

	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("listening URL %q does not parse: %v", rawURL, err)
	}
	if parsed.Scheme != "https" {
		t.Fatalf("listening URL %q has scheme %q, want https", rawURL, parsed.Scheme)
	}
	ip := net.ParseIP(parsed.Hostname())
	if ip == nil || !ip.IsLoopback() {
		t.Fatalf("listening URL %q does not name a loopback address", rawURL)
	}
	port, err := strconv.Atoi(parsed.Port())
	if err != nil || port < 1 || port > 65535 {
		t.Fatalf("listening URL %q does not carry a valid port", rawURL)
	}
}

// writeSetupDocument writes one document, doc-2, as a documents file in dir and returns its path.
func writeSetupDocument(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "documents.json")
	content := `[{"ID":"doc-2","CollectionID":"c1","ParentID":"","Title":"Setup","URL":"/doc/doc-2","Markdown":"Run the installer."}]`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write documents file: %v", err)
	}
	return path
}

func TestRunPassesEachFlagToTheStub(t *testing.T) {
	t.Run("collection flags reach collections.list", func(t *testing.T) {
		caFile := filepath.Join(t.TempDir(), "ca.pem")
		stub := startStub(t, "-ca-file", caFile, "-collection", "c1=Engineering", "-collection", "c2=Finance")
		base := listeningURL(t, stub.readLine(t))

		status, body := postList(t, trustingClient(t, caFile), base)

		if status != http.StatusOK || !strings.Contains(body, "Engineering") || !strings.Contains(body, "Finance") {
			t.Fatalf("collections.list = %d %q, want 200 with both configured collections", status, body)
		}
	})

	t.Run("list-status reaches collections.list", func(t *testing.T) {
		caFile := filepath.Join(t.TempDir(), "ca.pem")
		stub := startStub(t, "-ca-file", caFile, "-list-status", "401")
		base := listeningURL(t, stub.readLine(t))

		if status, _ := postList(t, trustingClient(t, caFile), base); status != http.StatusUnauthorized {
			t.Fatalf("collections.list status = %d, want 401", status)
		}
	})

	t.Run("document-status reaches documents.info", func(t *testing.T) {
		dir := t.TempDir()
		caFile := filepath.Join(dir, "ca.pem")
		docsFile := writeSetupDocument(t, dir)
		stub := startStub(t, "-ca-file", caFile, "-documents-file", docsFile, "-document-status", "404")
		base := listeningURL(t, stub.readLine(t))

		if status, _ := postInfo(t, trustingClient(t, caFile), base, "doc-2"); status != http.StatusNotFound {
			t.Fatalf("documents.info status = %d, want 404 from the fault on a document that exists", status)
		}
	})

	t.Run("documents file reaches documents.info", func(t *testing.T) {
		dir := t.TempDir()
		caFile := filepath.Join(dir, "ca.pem")
		docsFile := writeSetupDocument(t, dir)
		stub := startStub(t, "-ca-file", caFile, "-documents-file", docsFile)
		base := listeningURL(t, stub.readLine(t))

		status, body := postInfo(t, trustingClient(t, caFile), base, "doc-2")
		var reply struct {
			Data struct {
				Title string `json:"title"`
			} `json:"data"`
		}
		if err := json.Unmarshal([]byte(body), &reply); err != nil {
			t.Fatalf("documents.info body %q is not JSON: %v", body, err)
		}
		if status != http.StatusOK || reply.Data.Title != "Setup" {
			t.Fatalf("documents.info = %d %q, want 200 with the configured title Setup", status, body)
		}
	})
}

func TestRunRefusesBadFlagsWithoutListening(t *testing.T) {
	dir := t.TempDir()
	caFile := filepath.Join(dir, "ca.pem")
	badDocuments := filepath.Join(dir, "documents.json")
	if err := os.WriteFile(badDocuments, []byte("not json"), 0o600); err != nil {
		t.Fatalf("write documents file: %v", err)
	}

	cases := []struct {
		name    string
		args    []string
		refused func(error) bool
		want    string
	}{
		{
			name:    "no CA file",
			args:    []string{},
			refused: func(err error) bool { return errors.Is(err, errNoCAFile) },
			want:    "errNoCAFile",
		},
		{
			name:    "two faults",
			args:    []string{"-ca-file", caFile, "-list-status", "500", "-document-status", "404"},
			refused: func(err error) bool { return errors.Is(err, errTwoFaults) },
			want:    "errTwoFaults",
		},
		{
			name: "documents file not JSON",
			args: []string{"-ca-file", caFile, "-documents-file", badDocuments},
			refused: func(err error) bool {
				var syntaxErr *json.SyntaxError
				return errors.As(err, &syntaxErr)
			},
			want: "a *json.SyntaxError",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			err := serve(t.Context(), tc.args, stdio{out: &out, err: io.Discard}, system)
			if !tc.refused(err) {
				t.Fatalf("run returned %v, want %s", err, tc.want)
			}
			if out.Len() != 0 {
				t.Fatalf("run printed %q before refusing the flags, want no listening line", out.String())
			}
			if _, statErr := os.Stat(caFile); !errors.Is(statErr, fs.ErrNotExist) {
				t.Fatalf("run wrote the CA file before refusing the flags: %v", statErr)
			}
		})
	}
}
