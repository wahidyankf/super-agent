// Package jira is the Jira stub command as a library. Run parses the command line, starts the scripted Jira stub over TLS
// on a loopback port, writes the certificate a client must trust, prints its address, and waits until a stop signal
// arrives. The jirastubcmd process is a call to Run with the real Host. A test passes fakes for the files, the clock, the
// TLS server, and the signals, so it runs the command without touching the machine.
//
// The command prints one line, "jira stub listening on <url>", once it listens. The -email and -token values are synthetic
// test values; the stub accepts only that pair. A -config file (a JSON jirastub.Config) replaces the flags and holds the
// whole dataset and faults. A -record file gets one JSON line per request the stub receives, so the test can read what the
// stub saw.
package jira

import (
	"context"
	"encoding/json"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-testkit/jirastub"
)

// errNoCAFile reports a run that names no -ca-file.
var errNoCAFile = errors.New("the -ca-file flag is required")

// errNoCredentials reports a run that names no -email or no -token, and no -config file.
var errNoCredentials = errors.New("the -email and -token flags are required, or a -config file")

// Started is a TLS server the command started: the base URL it announces, the DER certificate that its clients trust, and
// the function that stops it.
type Started struct {
	URL  string
	DER  []byte
	Stop func()
}

// Host is everything the command reads from and writes to, apart from its two output streams. Real is the host of a
// process; a test passes fakes for every field it needs.
type Host struct {
	ReadFile   func(name string) ([]byte, error)
	WriteFile  func(name string, data []byte, perm fs.FileMode) error
	CreateFile func(name string) (io.WriteCloser, error) // opens a record file for appending
	Now        func() time.Time
	Serve      func(h http.Handler) (Started, error)
	Notify     func(parent context.Context, signals ...os.Signal) (context.Context, context.CancelFunc)
}

// Real is the host of a process: the real files, clock, TLS server, and stop signals.
var Real = Host{
	ReadFile:   os.ReadFile,
	WriteFile:  os.WriteFile,
	CreateFile: openRecordFile,
	Now:        time.Now,
	Serve:      serveTLS,
	Notify:     signal.NotifyContext,
}

// serveTLS starts h on a real loopback port with the stub certificate.
func serveTLS(h http.Handler) (Started, error) {
	srv, err := jirastub.NewTLSServer(h)
	if err != nil {
		return Started{}, err
	}
	return Started{URL: srv.URL, DER: srv.Certificate().Raw, Stop: srv.Close}, nil
}

// config is what the command needs to start the stub.
type config struct {
	caFile      string
	email       string
	token       string
	faultStatus int
	faultPath   string
	retryAfter  string
	configFile  string // a JSON jirastub.Config; when set, it replaces the flags above
	recordFile  string // where each received request is appended as one JSON line
}

// openRecordFile opens the record file for appending, creating it when it does not exist.
func openRecordFile(name string) (io.WriteCloser, error) {
	return os.OpenFile(name, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
}

// running is a started stub: the stub, its TLS server, and the open record file, if any.
type running struct {
	stub   *jirastub.Stub
	server Started
	record io.Closer
}

// stop releases the hanging requests first, because the server waits for them, and then stops the server and closes the
// record.
func (r running) stop() {
	if r.stub != nil {
		r.stub.Close()
	}
	if r.server.Stop != nil {
		r.server.Stop()
	}
	if r.record != nil {
		_ = r.record.Close()
	}
}

// newStub builds the stub that cfg describes: the -config file when one is named, otherwise the flags. Its clock is h.Now.
func newStub(h Host, cfg config) (*jirastub.Stub, error) {
	if cfg.configFile == "" {
		stub := jirastub.New(cfg.email, cfg.token)
		if cfg.faultStatus > 0 {
			stub.SetFault(jirastub.Fault{Path: cfg.faultPath, Status: cfg.faultStatus, RetryAfter: cfg.retryAfter})
		}
		stub.SetClock(h.Now)
		return stub, nil
	}
	data, err := h.ReadFile(cfg.configFile)
	if err != nil {
		return nil, fmt.Errorf("read the config file: %w", err)
	}
	var c jirastub.Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse the config file: %w", err)
	}
	stub := jirastub.New("", "")
	stub.ApplyConfig(c)
	stub.SetClock(h.Now)
	return stub, nil
}

// recorderFor opens the record file and returns the hook that appends each request to it, and the file to close.
func recorderFor(h Host, path string) (func(jirastub.Request), io.Closer, error) {
	f, err := h.CreateFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open the record file: %w", err)
	}
	var mu sync.Mutex
	enc := json.NewEncoder(f)
	hook := func(req jirastub.Request) {
		mu.Lock()
		defer mu.Unlock()
		_ = enc.Encode(req)
	}
	return hook, f, nil
}

// startStub serves the stub over TLS, and writes its certificate to cfg.caFile. A fault applies to every request whose path
// starts with faultPath, when faultStatus is set. The caller stops the result.
func startStub(h Host, cfg config) (running, error) {
	stub, err := newStub(h, cfg)
	if err != nil {
		return running{}, err
	}
	out := running{stub: stub}
	if cfg.recordFile != "" {
		hook, closer, err := recorderFor(h, cfg.recordFile)
		if err != nil {
			return running{}, err
		}
		stub.SetRecorder(hook)
		out.record = closer
	}
	server, err := h.Serve(stub)
	if err != nil {
		out.stop()
		return running{}, fmt.Errorf("start the TLS server: %w", err)
	}
	out.server = server
	certificate := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.DER})
	if err := h.WriteFile(cfg.caFile, certificate, 0o600); err != nil {
		out.stop()
		return running{}, fmt.Errorf("write the CA file: %w", err)
	}
	return out, nil
}

// run parses the flags, starts the stub, prints its address to stdout, and waits until ctx is done or a stop signal arrives.
// Flag diagnostics go to stderr.
func run(ctx context.Context, args []string, stdout, stderr io.Writer, h Host) error {
	flags := flag.NewFlagSet("jirastubcmd", flag.ContinueOnError)
	flags.SetOutput(stderr)
	caFile := flags.String("ca-file", "", "where to write the PEM certificate of the stub")
	email := flags.String("email", "", "the synthetic e-mail address the stub accepts")
	token := flags.String("token", "", "the synthetic token the stub accepts")
	faultStatus := flags.Int("fault-status", 0, "answer with this HTTP status instead of replaying; 0 replays normally")
	faultPath := flags.String("fault-path", "", "limit -fault-status to paths that start with this prefix")
	retryAfter := flags.String("retry-after", "", "the Retry-After header sent with a fault status of 429")
	configFile := flags.String("config", "", "a JSON jirastub.Config file that replaces -email, -token, and the fault flags")
	recordFile := flags.String("record", "", "append each received request to this file as one JSON line")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *caFile == "" {
		return errNoCAFile
	}
	if *configFile == "" && (*email == "" || *token == "") {
		return errNoCredentials
	}

	r, err := startStub(h, config{
		caFile:      *caFile,
		email:       *email,
		token:       *token,
		faultStatus: *faultStatus,
		faultPath:   *faultPath,
		retryAfter:  *retryAfter,
		configFile:  *configFile,
		recordFile:  *recordFile,
	})
	if err != nil {
		return err
	}
	defer r.stop()

	ctx, stop := h.Notify(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Fprintf(stdout, "jira stub listening on %s\n", r.server.URL)

	<-ctx.Done()
	return nil
}

// Run runs the command and returns the process status: 0 on success, and 1 after printing the error to stderr.
func Run(args []string, stdout, stderr io.Writer, h Host) int {
	if err := run(context.Background(), args, stdout, stderr, h); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
