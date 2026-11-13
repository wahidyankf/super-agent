// Package main is a test-only command. It serves the scripted Jira stub over TLS on a loopback port, as a process, so the
// end-to-end layer can point the real retriever binary at it through the loopback seam. It is not part of the retriever binary,
// and nothing in retriever-cli depends on it.
//
// The command prints one line, "jira stub listening on <url>", once it listens. It writes the PEM certificate that the client
// must trust to -ca-file before that line. The -email and -token values are synthetic test values; the stub accepts only that
// pair. A -config file (a JSON jirastub.Config) replaces the flags and holds the whole dataset and faults. A -record file gets
// one JSON line per request the stub receives, so the test can read what the process saw. It runs until it receives SIGINT
// or SIGTERM.
package main

import (
	"context"
	"encoding/json"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/tests/support/jirastub"
)

// errNoCAFile reports a run that names no -ca-file.
var errNoCAFile = errors.New("the -ca-file flag is required")

// errNoCredentials reports a run that names no -email or no -token, and no -config file.
var errNoCredentials = errors.New("the -email and -token flags are required, or a -config file")

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

// running is a started stub: the stub, its TLS server, and the open record file, if any.
type running struct {
	stub   *jirastub.Stub
	server *httptest.Server
	record io.Closer
}

// stop releases the hanging requests first, because the server waits for them, and then closes the server and the record.
func (r running) stop() {
	if r.stub != nil {
		r.stub.Close()
	}
	if r.server != nil {
		r.server.Close()
	}
	if r.record != nil {
		_ = r.record.Close()
	}
}

// newStub builds the stub that cfg describes: the -config file when one is named, otherwise the flags.
func newStub(cfg config) (*jirastub.Stub, error) {
	if cfg.configFile == "" {
		stub := jirastub.New(cfg.email, cfg.token)
		if cfg.faultStatus > 0 {
			stub.SetFault(jirastub.Fault{Path: cfg.faultPath, Status: cfg.faultStatus, RetryAfter: cfg.retryAfter})
		}
		return stub, nil
	}
	data, err := os.ReadFile(cfg.configFile)
	if err != nil {
		return nil, fmt.Errorf("read the config file: %w", err)
	}
	var c jirastub.Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse the config file: %w", err)
	}
	stub := jirastub.New("", "")
	stub.ApplyConfig(c)
	return stub, nil
}

// recorderFor opens the record file and returns the hook that appends each request to it, and the file to close.
func recorderFor(path string) (func(jirastub.Request), io.Closer, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
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

// startStub serves the stub over TLS on a loopback port, and writes its certificate to cfg.caFile. A fault applies to every
// request whose path starts with faultPath, when faultStatus is set. The caller stops the result.
func startStub(cfg config) (running, error) {
	stub, err := newStub(cfg)
	if err != nil {
		return running{}, err
	}
	out := running{stub: stub}
	if cfg.recordFile != "" {
		hook, closer, err := recorderFor(cfg.recordFile)
		if err != nil {
			return running{}, err
		}
		stub.SetRecorder(hook)
		out.record = closer
	}
	server, err := jirastub.NewTLSServer(stub)
	if err != nil {
		out.stop()
		return running{}, fmt.Errorf("make the TLS certificate: %w", err)
	}
	out.server = server
	certificate := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw})
	if err := os.WriteFile(cfg.caFile, certificate, 0o600); err != nil {
		out.stop()
		return running{}, fmt.Errorf("write the CA file: %w", err)
	}
	return out, nil
}

// start serves the stub as startStub does, for a caller that needs only the server. The caller closes the returned server.
func start(cfg config) (*httptest.Server, error) {
	r, err := startStub(cfg)
	if err != nil {
		return nil, err
	}
	return r.server, nil
}

// run parses the flags, starts the stub, prints its address, and waits until ctx is done or a stop signal arrives.
func run(ctx context.Context, args []string, out io.Writer) error {
	flags := flag.NewFlagSet("jirastubcmd", flag.ContinueOnError)
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

	r, err := startStub(config{
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

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Fprintf(out, "jira stub listening on %s\n", r.server.URL)

	<-ctx.Done()
	return nil
}

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
