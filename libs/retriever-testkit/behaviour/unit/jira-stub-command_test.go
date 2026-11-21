package unit_test

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-testkit/jirastub"
	jiracmd "github.com/wahidyankf/super-agent/libs/retriever-testkit/stubcmd/jira"
)

// jcmdRecord is an in-memory record file. Its Close is recorded, so a test sees that the run closed it.
type jcmdRecord struct {
	bytes.Buffer
	closed bool
}

func (r *jcmdRecord) Close() error {
	r.closed = true
	return nil
}

// jcmdHost is the host of one unit run of the Jira command. Its input files are a map, its writes are recorded, its
// TLS server is a fake that keeps the handler and opens no socket, and its clock is fixedClock.
type jcmdHost struct {
	inputs    map[string][]byte
	written   map[string][]byte
	writeErr  error
	createErr error
	record    *jcmdRecord
	serves    int
	handler   http.Handler
	stopped   bool
	notify    func(parent context.Context, signals ...os.Signal) (context.Context, context.CancelFunc)
}

// newJcmdHost returns a host with no input files.
func newJcmdHost() *jcmdHost {
	return &jcmdHost{inputs: map[string][]byte{}, written: map[string][]byte{}}
}

// jcmdStopNow is the signal wait of a run to its end: the context it returns is already done.
func jcmdStopNow(parent context.Context, _ ...os.Signal) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(parent)
	cancel()
	return ctx, cancel
}

// host returns the command's host of fakes.
func (f *jcmdHost) host() jiracmd.Host {
	notify := f.notify
	if notify == nil {
		notify = jcmdStopNow
	}
	return jiracmd.Host{
		ReadFile: func(name string) ([]byte, error) {
			data, ok := f.inputs[name]
			if !ok {
				return nil, fs.ErrNotExist
			}
			return data, nil
		},
		WriteFile: func(name string, data []byte, _ fs.FileMode) error {
			if f.writeErr != nil {
				return f.writeErr
			}
			f.written[name] = append([]byte(nil), data...)
			return nil
		},
		CreateFile: func(string) (io.WriteCloser, error) {
			if f.createErr != nil {
				return nil, f.createErr
			}
			f.record = &jcmdRecord{}
			return f.record, nil
		},
		Now: fixedClock,
		Serve: func(h http.Handler) (jiracmd.Started, error) {
			f.serves++
			f.handler = h
			return jiracmd.Started{
				URL:  "https://127.0.0.1:9443",
				DER:  []byte("fixture-certificate-der"),
				Stop: func() { f.stopped = true },
			}, nil
		},
		Notify: notify,
	}
}

// jcmdOrdered is the standard output of a run. It notes whether the CA file was already written when the first byte was
// printed, which is the order the scenario states.
type jcmdOrdered struct {
	host    *jcmdHost
	printed bytes.Buffer
	first   bool
	seen    bool
}

func (w *jcmdOrdered) Write(p []byte) (int, error) {
	if !w.seen {
		w.seen = true
		w.first = len(w.host.written) > 0
	}
	return w.printed.Write(p)
}

// jcmdRun runs the Jira command with args and host h, with both streams in memory, and returns its status and streams.
func jcmdRun(args []string, h jiracmd.Host) (int, string, string) {
	var out, errOut bytes.Buffer
	status := jiracmd.Run(args, &out, &errOut, h)
	return status, out.String(), errOut.String()
}

// jcmdStart runs the Jira command in a goroutine. It returns once the command waits for its stop signal, which it does
// after the stub is serving, so the test can call the handler. The returned function sends the stop signal and returns the
// status of the run.
func jcmdStart(args []string, f *jcmdHost) (func() int, *bytes.Buffer) {
	waiting := make(chan context.CancelFunc, 1)
	f.notify = func(parent context.Context, _ ...os.Signal) (context.Context, context.CancelFunc) {
		ctx, cancel := context.WithCancel(parent)
		waiting <- cancel
		return ctx, cancel
	}
	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
	done := make(chan int, 1)
	go func() {
		done <- jiracmd.Run(args, out, errOut, f.host())
	}()
	cancel := <-waiting
	return func() int {
		cancel()
		return <-done
	}, out
}

// jcmdCall sends one request to the handler in process, with the Authorization header auth when it is set.
func jcmdCall(h http.Handler, method, path, auth string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "https://jira.example.test"+path, nil)
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// jcmdCAArgs are the flags of a run that has its credentials and its CA file.
func jcmdCAArgs() []string {
	return []string{"-ca-file", "ca.pem", "-email", jiraEmail, "-token", jiraToken}
}

// @scenario specs/libs/retriever-testkit/behaviours/jira-stub-command.feature: The command refuses to start without a certificate file path
func TestJiraCommandRefusesToStartWithoutCertificatePath(t *testing.T) {
	f := newJcmdHost()
	status, out, errOut := jcmdRun([]string{"-email", jiraEmail, "-token", jiraToken}, f.host())
	if status != 1 || !strings.Contains(errOut, "the -ca-file flag is required") {
		t.Fatalf("status %d and stderr %q, want 1 with the -ca-file message", status, errOut)
	}
	if out != "" || f.serves != 0 {
		t.Fatalf("the command printed %q and served %d times, want no port", out, f.serves)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/jira-stub-command.feature: The command refuses to start with no account, no token, and no config file
func TestJiraCommandRefusesToStartWithNoAccountTokenOrConfigFile(t *testing.T) {
	f := newJcmdHost()
	status, out, errOut := jcmdRun([]string{"-ca-file", "ca.pem"}, f.host())
	want := "the -email and -token flags are required, or a -config file"
	if status != 1 || !strings.Contains(errOut, want) {
		t.Fatalf("status %d and stderr %q, want 1 with %q", status, errOut, want)
	}
	if out != "" || f.serves != 0 {
		t.Fatalf("the command printed %q and served %d times, want no port", out, f.serves)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/jira-stub-command.feature: The command reports a flag it does not know on its diagnostic output
func TestJiraCommandReportsAFlagItDoesNotKnowOnItsErrorOutput(t *testing.T) {
	f := newJcmdHost()
	status, out, errOut := jcmdRun([]string{"-no-such-flag"}, f.host())
	if status != 1 || !strings.Contains(errOut, "flag provided but not defined: -no-such-flag") {
		t.Fatalf("status %d and stderr %q, want 1 with the diagnostic", status, errOut)
	}
	if out != "" {
		t.Fatalf("the diagnostic reached the standard output: %q", out)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/jira-stub-command.feature: A config file supplies the account and the token in place of the flags
func TestJiraCommandTakesTheAccountAndTokenFromAConfigFile(t *testing.T) {
	f := newJcmdHost()
	data, err := json.Marshal(jirastub.Config{Email: "config@example.test", Token: "config-token"})
	if err != nil {
		t.Fatalf("marshal the config: %v", err)
	}
	f.inputs["stub.json"] = data
	stop, _ := jcmdStart([]string{"-ca-file", "ca.pem", "-config", "stub.json"}, f)

	rec := jcmdCall(f.handler, http.MethodGet, "/rest/api/3/myself", basicHeader("config@example.test", "config-token"))

	if rec.Code != http.StatusOK {
		t.Fatalf("the stub answered %d to the config account, want 200", rec.Code)
	}
	if status := stop(); status != 0 {
		t.Fatalf("status %d after the stop signal, want 0", status)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/jira-stub-command.feature: A config file that cannot be read or parsed is refused with its error
func TestJiraCommandRefusesAConfigFileThatCannotBeReadOrParsed(t *testing.T) {
	cases := []struct {
		name    string
		input   []byte
		noFile  bool
		message string
	}{
		{name: "cannot be read", noFile: true, message: "read the config file"},
		{name: "is not valid JSON", input: []byte("{not json"), message: "parse the config file"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newJcmdHost()
			if !tc.noFile {
				f.inputs["stub.json"] = tc.input
			}
			status, _, errOut := jcmdRun([]string{"-ca-file", "ca.pem", "-config", "stub.json"}, f.host())
			if status != 1 || !strings.HasPrefix(errOut, tc.message) {
				t.Fatalf("status %d and stderr %q, want 1 with an error that starts with %q", status, errOut, tc.message)
			}
			if f.serves != 0 {
				t.Fatal("a config that did not load still started the stub")
			}
		})
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/jira-stub-command.feature: The record file holds one JSON line for each request and never the token
func TestJiraCommandRecordHoldsOneJSONLineAndNeverTheToken(t *testing.T) {
	f := newJcmdHost()
	stop, _ := jcmdStart(append(jcmdCAArgs(), "-record", "requests.jsonl"), f)

	jcmdCall(f.handler, http.MethodGet, "/rest/api/3/myself", basicHeader(jiraEmail, jiraToken))
	if status := stop(); status != 0 {
		t.Fatalf("status %d after the stop signal, want 0", status)
	}

	if f.record == nil || !f.record.closed {
		t.Fatal("the record file is missing or was left open")
	}
	line := f.record.String()
	if strings.Count(line, "\n") != 1 {
		t.Fatalf("the record holds %d lines, want one: %q", strings.Count(line, "\n"), line)
	}
	var got jirastub.Request
	if err := json.Unmarshal([]byte(line), &got); err != nil {
		t.Fatalf("the record line is not a request: %v", err)
	}
	if got.Method != http.MethodGet || got.Path != "/rest/api/3/myself" || !got.AuthOK {
		t.Fatalf("recorded %+v, want an authenticated GET of myself", got)
	}
	if strings.Contains(line, jiraToken) {
		t.Fatal("the record holds the token")
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/jira-stub-command.feature: The command writes the certificate, then announces its address
func TestJiraCommandWritesTheCertificateThenAnnouncesItsAddress(t *testing.T) {
	f := newJcmdHost()
	printed := &jcmdOrdered{host: f}
	var errOut bytes.Buffer
	status := jiracmd.Run(jcmdCAArgs(), printed, &errOut, f.host())
	if status != 0 {
		t.Fatalf("status %d, stderr %q; want a clean run", status, errOut.String())
	}
	if got, want := printed.printed.String(), "jira stub listening on https://127.0.0.1:9443\n"; got != want {
		t.Fatalf("printed %q, want %q", got, want)
	}
	if !printed.first {
		t.Fatal("the address was announced before the certificate file was written")
	}
	block, _ := pem.Decode(f.written["ca.pem"])
	if block == nil || block.Type != "CERTIFICATE" {
		t.Fatal("the certificate file holds no PEM certificate")
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/jira-stub-command.feature: A clean run exits 0, and a failed run prints its error and exits 1
func TestJiraCommandCleanRunExitsZeroAndFailedRunPrintsItsErrorAndExitsOne(t *testing.T) {
	clean := newJcmdHost()
	if status, _, errOut := jcmdRun(jcmdCAArgs(), clean.host()); status != 0 {
		t.Fatalf("clean run status %d, stderr %q; want 0", status, errOut)
	}

	failed := newJcmdHost()
	failed.writeErr = errors.New("CA directory is read-only")
	status, _, errOut := jcmdRun(jcmdCAArgs(), failed.host())
	if status != 1 || !strings.Contains(errOut, "write the CA file: CA directory is read-only") {
		t.Fatalf("failed run status %d and stderr %q, want 1 with the write error", status, errOut)
	}
	if !failed.stopped {
		t.Fatal("the failed run left the stub running")
	}
}
