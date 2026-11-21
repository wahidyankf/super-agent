package unit_test

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/pem"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	outlinecmd "github.com/wahidyankf/super-agent/libs/retriever-testkit/stubcmd/outline"
)

// ocmdHost is the host of one unit run of the Outline command. Its input files are a map, its writes are recorded, its TLS
// server is a fake that keeps the handler and opens no socket, and its clock is fixedClock.
type ocmdHost struct {
	inputs   map[string][]byte
	written  map[string][]byte
	writeErr error
	serves   int
	handler  http.Handler
	stopped  bool
	notify   func(parent context.Context, signals ...os.Signal) (context.Context, context.CancelFunc)
	// served is closed by the fake Serve once the stub exists. The command calls Notify before it serves, so the stop
	// signal that Notify returns (cancel) is safe to read only after served is closed.
	served chan struct{}
	cancel context.CancelFunc
}

// newOcmdHost returns a host with no input files.
func newOcmdHost() *ocmdHost {
	return &ocmdHost{inputs: map[string][]byte{}, written: map[string][]byte{}, served: make(chan struct{})}
}

// ocmdStopNow is the signal wait of a run to its end: the context it returns is already done.
func ocmdStopNow(parent context.Context, _ ...os.Signal) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(parent)
	cancel()
	return ctx, cancel
}

// host returns the command's host of fakes.
func (f *ocmdHost) host() outlinecmd.Host {
	notify := f.notify
	if notify == nil {
		notify = ocmdStopNow
	}
	return outlinecmd.Host{
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
		Now: fixedClock,
		Serve: func(h http.Handler) (outlinecmd.Started, error) {
			f.serves++
			f.handler = h
			close(f.served)
			return outlinecmd.Started{
				URL:  "https://127.0.0.1:9443",
				DER:  []byte("fixture-certificate-der"),
				Stop: func() { f.stopped = true },
			}, nil
		},
		Notify: notify,
	}
}

// ocmdOrdered is the standard output of a run. It notes whether the CA file was already written when the first byte was
// printed, which is the order the scenario states.
type ocmdOrdered struct {
	host    *ocmdHost
	printed bytes.Buffer
	first   bool
	seen    bool
}

func (w *ocmdOrdered) Write(p []byte) (int, error) {
	if !w.seen {
		w.seen = true
		w.first = len(w.host.written) > 0
	}
	return w.printed.Write(p)
}

// ocmdRun runs the Outline command with args and host h, with both streams in memory, and returns its status and streams.
func ocmdRun(args []string, h outlinecmd.Host) (int, string, string) {
	var out, errOut bytes.Buffer
	status := outlinecmd.Run(args, &out, &errOut, h)
	return status, out.String(), errOut.String()
}

// ocmdStart runs the Outline command in a goroutine. It returns once the command waits for its stop signal, which it does
// after the stub is serving, so the test can call the handler. The returned function sends the stop signal and returns the
// status of the run.
func ocmdStart(args []string, f *ocmdHost) func() int {
	f.notify = func(parent context.Context, _ ...os.Signal) (context.Context, context.CancelFunc) {
		ctx, cancel := context.WithCancel(parent)
		f.cancel = cancel
		return ctx, cancel
	}
	done := make(chan int, 1)
	go func() {
		done <- outlinecmd.Run(args, io.Discard, io.Discard, f.host())
	}()
	<-f.served
	return func() int {
		f.cancel()
		return <-done
	}
}

// ocmdPost sends one JSON POST to the handler in process, and returns the status and body of the reply.
func ocmdPost(h http.Handler, method, body string) (int, []byte) {
	req := httptest.NewRequest(http.MethodPost, "https://127.0.0.1/api/"+method, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code, rec.Body.Bytes()
}

// @scenario specs/libs/retriever-testkit/behaviours/outline-stub-command.feature: The command refuses to start without a certificate file path
func TestOutlineCommandRefusesToStartWithoutCertificatePath(t *testing.T) {
	f := newOcmdHost()
	status, out, errOut := ocmdRun(nil, f.host())
	if status != 1 || !strings.Contains(errOut, "the -ca-file flag is required") {
		t.Fatalf("status %d and stderr %q, want 1 with the -ca-file message", status, errOut)
	}
	if out != "" || f.serves != 0 {
		t.Fatalf("the command printed %q and served %d times, want no port", out, f.serves)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/outline-stub-command.feature: The command refuses both status faults at once
func TestOutlineCommandRefusesBothStatusFaultsAtOnce(t *testing.T) {
	f := newOcmdHost()
	args := []string{"-ca-file", "ca.pem", "-list-status", "500", "-document-status", "503"}
	status, out, errOut := ocmdRun(args, f.host())
	if status != 1 || !strings.Contains(errOut, "only one of -list-status and -document-status can be set") {
		t.Fatalf("status %d and stderr %q, want 1 with the two-faults message", status, errOut)
	}
	if out != "" || f.serves != 0 {
		t.Fatalf("the command printed %q and served %d times, want no port", out, f.serves)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/outline-stub-command.feature: A collection flag is read as id=name in the order given
func TestOutlineCollectionFlagIsReadAsIDAndName(t *testing.T) {
	cases := []struct {
		name  string
		value string
		want  string
	}{
		{name: "c1=Eng", value: "c1=Eng", want: "Eng"},
		{name: "c1=", value: "c1=", want: "-collection wants id=name"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newOcmdHost()
			args := []string{"-ca-file", "ca.pem", "-collection", tc.value}
			if tc.value != "c1=Eng" {
				status, _, errOut := ocmdRun(args, f.host())
				if status != 1 || !strings.Contains(errOut, tc.want) {
					t.Fatalf("status %d and stderr %q, want 1 with an error that holds %q", status, errOut, tc.want)
				}
				return
			}
			stop := ocmdStart(args, f)
			code, body := ocmdPost(f.handler, "collections.info", `{"id":"c1"}`)
			if status := stop(); status != 0 {
				t.Fatalf("status %d after the stop signal, want 0", status)
			}
			var reply struct {
				Data struct {
					Name string `json:"name"`
				} `json:"data"`
			}
			if err := json.Unmarshal(body, &reply); err != nil || code != http.StatusOK {
				t.Fatalf("collections.info = %d %s, want a 200 reply", code, body)
			}
			if reply.Data.Name != tc.want {
				t.Fatalf("the collection c1 is named %q, want %q", reply.Data.Name, tc.want)
			}
		})
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/outline-stub-command.feature: The command announces its loopback address after it writes the certificate
func TestOutlineCommandAnnouncesItsLoopbackAddressAfterWritingTheCertificate(t *testing.T) {
	f := newOcmdHost()
	printed := &ocmdOrdered{host: f}
	var errOut bytes.Buffer
	status := outlinecmd.Run([]string{"-ca-file", "ca.pem"}, printed, &errOut, f.host())
	if status != 0 {
		t.Fatalf("status %d, stderr %q; want a clean run", status, errOut.String())
	}
	if got, want := printed.printed.String(), "outline stub listening on https://127.0.0.1:9443\n"; got != want {
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

// @scenario specs/libs/retriever-testkit/behaviours/outline-stub-command.feature: The command stops answering when its context ends
func TestOutlineCommandStopsTheStubWhenItsContextEnds(t *testing.T) {
	f := newOcmdHost()
	stop := ocmdStart([]string{"-ca-file", "ca.pem"}, f)
	if code, _ := ocmdPost(f.handler, "collections.list", `{"offset":0,"limit":1}`); code != http.StatusOK {
		t.Fatalf("collections.list before the stop = %d, want 200", code)
	}

	if status := stop(); status != 0 {
		t.Fatalf("status %d after the context ended, want 0", status)
	}
	if !f.stopped {
		t.Fatal("the command returned but the stub was not stopped")
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/outline-stub-command.feature: A documents file that is not JSON is refused with its parse error
func TestOutlineCommandRefusesADocumentsFileThatIsNotJSON(t *testing.T) {
	f := newOcmdHost()
	f.inputs["documents.json"] = []byte("not json")
	status, _, errOut := ocmdRun([]string{"-ca-file", "ca.pem", "-documents-file", "documents.json"}, f.host())
	if status != 1 || !strings.HasPrefix(errOut, "parse the documents file") {
		t.Fatalf("status %d and stderr %q, want 1 with an error that starts with the parse message", status, errOut)
	}
	if f.serves != 0 {
		t.Fatal("a documents file that did not parse still started the stub")
	}
}
