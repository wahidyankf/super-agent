// Package main is a test-only command. It serves the scripted Kiwi stub over TLS on a loopback port, as a process, so the
// end-to-end layer can point the real retriever binary at it through the RETRIEVER_KIWI_TEST_* seam. It is not part of the
// retriever binary, and nothing in retriever-cli depends on it.
//
// The Bearer key the stub accepts comes from the KIWISTUB_API_KEY variable, which the test sets to the same synthetic value
// it gives retriever. The command prints one line, "kiwi stub listening on <url>", once it listens. It writes the PEM
// certificate the client must trust to -ca-file before that line. It runs until it receives SIGINT or SIGTERM.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/tests/support/kiwistub"
)

// keyVariable names the variable that holds the synthetic Bearer key the stub accepts. Its text names the variable only.
const keyVariable = "KIWISTUB_API_KEY"

// errNoCAFile reports a run that names no -ca-file.
var errNoCAFile = errors.New("the -ca-file flag is required")

// errNoKey reports a run whose key variable is empty.
var errNoKey = errors.New(keyVariable + " must hold the synthetic key")

// config is what the command needs to start the stub.
type config struct {
	caFile      string
	key         string
	dataset     kiwistub.Dataset
	faultStatus int
}

// readDataset reads the dataset the stub holds from a JSON file. The file has the field names of kiwistub.Dataset.
func readDataset(path string) (kiwistub.Dataset, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return kiwistub.Dataset{}, fmt.Errorf("read the dataset file: %w", err)
	}
	var d kiwistub.Dataset
	if err := json.Unmarshal(data, &d); err != nil {
		return kiwistub.Dataset{}, fmt.Errorf("parse the dataset file: %w", err)
	}
	return d, nil
}

// start serves the stub over TLS and writes its certificate to cfg.caFile. A -fault-status answers every request with
// that HTTP status. The caller closes the returned stub.
func start(cfg config) (*kiwistub.Stub, error) {
	stub := kiwistub.New(cfg.key, kiwistub.WithDataset(cfg.dataset))
	if cfg.faultStatus > 0 {
		stub.SetFault(kiwistub.Fault{Status: cfg.faultStatus})
	}
	if err := stub.Start(); err != nil {
		return nil, fmt.Errorf("make the TLS certificate: %w", err)
	}
	if err := stub.WriteCAFile(cfg.caFile); err != nil {
		stub.Close()
		return nil, fmt.Errorf("write the CA file: %w", err)
	}
	return stub, nil
}

// run parses the flags, starts the stub, prints its address, and waits until ctx is done or a stop signal arrives.
func run(ctx context.Context, args []string, out io.Writer) error {
	flags := flag.NewFlagSet("kiwistubcmd", flag.ContinueOnError)
	caFile := flags.String("ca-file", "", "where to write the PEM certificate of the stub")
	datasetFile := flags.String("dataset-file", "", "a JSON file of the dataset the stub holds")
	faultStatus := flags.Int("fault-status", 0, "answer every request with this HTTP status; 0 replays normally")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *caFile == "" {
		return errNoCAFile
	}
	key := os.Getenv(keyVariable)
	if key == "" {
		return errNoKey
	}
	dataset := kiwistub.Dataset{}
	if *datasetFile != "" {
		d, err := readDataset(*datasetFile)
		if err != nil {
			return err
		}
		dataset = d
	}

	stub, err := start(config{caFile: *caFile, key: key, dataset: dataset, faultStatus: *faultStatus})
	if err != nil {
		return err
	}
	defer stub.Close()

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Fprintf(out, "kiwi stub listening on %s\n", stub.URL())

	<-ctx.Done()
	return nil
}

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
