// Package kiwi is the Kiwi stub command as a library. Run parses the command line, starts the scripted Kiwi stub over TLS on
// a loopback port, writes its certificate, prints its address, and waits until a stop signal arrives. The kiwistubcmd
// process is a call to Run with the real Host. A test passes fakes for the environment, the clock, the files, the listening
// socket, the entropy, and the signals, so it runs the command without touching the machine.
//
// The Bearer key the stub accepts comes from the KIWISTUB_API_KEY variable, which the test sets to the same synthetic value
// it gives retriever. The command prints one line, "kiwi stub listening on <url>", once it listens.
package kiwi

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-testkit/kiwistub"
)

// keyVariable names the variable that holds the synthetic Bearer key the stub accepts. Its text names the variable only.
const keyVariable = "KIWISTUB_API_KEY"

// errNoCAFile reports a run that names no -ca-file.
var errNoCAFile = errors.New("the -ca-file flag is required")

// errNoKey reports a run whose key variable is empty.
var errNoKey = errors.New(keyVariable + " must hold the synthetic key")

// Host is everything the command reads from and writes to, apart from its two output streams. Real is the host of a
// process; a test passes fakes for every field it needs.
type Host struct {
	Getenv    func(name string) string
	Now       func() time.Time
	ReadFile  func(name string) ([]byte, error)
	WriteFile func(name string, data []byte, perm fs.FileMode) error
	Listen    func(network, address string) (net.Listener, error)
	Random    io.Reader
	Notify    func(parent context.Context, signals ...os.Signal) (context.Context, context.CancelFunc)
}

// Real is the host of a process: the real environment, clock, files, loopback socket, entropy, and stop signals.
var Real = Host{
	Getenv:    os.Getenv,
	Now:       time.Now,
	ReadFile:  os.ReadFile,
	WriteFile: os.WriteFile,
	Listen:    net.Listen,
	Random:    rand.Reader,
	Notify:    signal.NotifyContext,
}

// config is what the command needs to start the stub.
type config struct {
	caFile      string
	key         string
	dataset     kiwistub.Dataset
	faultStatus int
}

// readDataset reads the dataset the stub holds from a JSON file. The file has the field names of kiwistub.Dataset.
func readDataset(h Host, path string) (kiwistub.Dataset, error) {
	data, err := h.ReadFile(path)
	if err != nil {
		return kiwistub.Dataset{}, fmt.Errorf("read the dataset file: %w", err)
	}
	var d kiwistub.Dataset
	if err := json.Unmarshal(data, &d); err != nil {
		return kiwistub.Dataset{}, fmt.Errorf("parse the dataset file: %w", err)
	}
	return d, nil
}

// start serves the stub over TLS and writes its certificate to cfg.caFile. A faultStatus answers every request with that
// HTTP status. The caller closes the returned stub.
func start(cfg config, h Host) (*kiwistub.Stub, error) {
	stub := kiwistub.New(cfg.key,
		kiwistub.WithDataset(cfg.dataset),
		kiwistub.WithClock(h.Now),
		kiwistub.WithListener(h.Listen),
		kiwistub.WithRandom(h.Random),
	)
	if cfg.faultStatus > 0 {
		stub.SetFault(kiwistub.Fault{Status: cfg.faultStatus})
	}
	if err := stub.Start(); err != nil {
		return nil, fmt.Errorf("start the Kiwi stub: %w", err)
	}
	if err := h.WriteFile(cfg.caFile, stub.CAPEM(), 0o600); err != nil {
		stub.Close()
		return nil, fmt.Errorf("write the CA file: %w", err)
	}
	return stub, nil
}

// run parses the flags, starts the stub, prints its address to stdout, and waits until ctx is done or a stop signal
// arrives. Flag diagnostics go to stderr.
func run(ctx context.Context, args []string, stdout, stderr io.Writer, h Host) error {
	flags := flag.NewFlagSet("kiwistubcmd", flag.ContinueOnError)
	flags.SetOutput(stderr)
	caFile := flags.String("ca-file", "", "where to write the PEM certificate of the stub")
	datasetFile := flags.String("dataset-file", "", "a JSON file of the dataset the stub holds")
	faultStatus := flags.Int("fault-status", 0, "answer every request with this HTTP status; 0 replays normally")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *caFile == "" {
		return errNoCAFile
	}
	key := h.Getenv(keyVariable)
	if key == "" {
		return errNoKey
	}
	dataset := kiwistub.Dataset{}
	if *datasetFile != "" {
		d, err := readDataset(h, *datasetFile)
		if err != nil {
			return err
		}
		dataset = d
	}

	stub, err := start(config{caFile: *caFile, key: key, dataset: dataset, faultStatus: *faultStatus}, h)
	if err != nil {
		return err
	}
	defer stub.Close()

	ctx, stop := h.Notify(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Fprintf(stdout, "kiwi stub listening on %s\n", stub.URL())

	<-ctx.Done()
	return nil
}

// Run runs the command with args and returns the process status: 0 for a clean run, and 1 for a failed one, after the
// error has been printed to stderr. The command waits for its stop signal, as the process does.
func Run(args []string, stdout, stderr io.Writer, h Host) int {
	return exitStatus(stderr, run(context.Background(), args, stdout, stderr, h))
}

// exitStatus reports a failed run on stderr, and returns the process status: 0 for a clean run, 1 for a failed one.
func exitStatus(stderr io.Writer, err error) int {
	if err == nil {
		return 0
	}
	fmt.Fprintln(stderr, err)
	return 1
}
