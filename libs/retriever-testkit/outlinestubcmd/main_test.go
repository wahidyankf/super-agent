package main

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-testkit/stubcmd/outline"
)

// fakeHost is the host of the test: its TLS server is a fake that opens no socket, its files are not written, and its stop
// signal is already done, so a run returns once it has printed its address.
func fakeHost() outline.Host {
	return outline.Host{
		ReadFile:  func(string) ([]byte, error) { return nil, os.ErrNotExist },
		WriteFile: func(string, []byte, os.FileMode) error { return nil },
		Now:       func() time.Time { return time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC) },
		Serve: func(http.Handler) (outline.Started, error) {
			return outline.Started{URL: "https://127.0.0.1:4443", DER: []byte("fixture-der"), Stop: func() {}}, nil
		},
		Notify: func(parent context.Context, _ ...os.Signal) (context.Context, context.CancelFunc) {
			ctx, cancel := context.WithCancel(parent)
			cancel()
			return ctx, cancel
		},
	}
}

// TestMainEndsTheProcessWithTheStatusOfTheRun checks both statuses main can end with: 0 for a run that serves and stops,
// and 1 for a run that refuses its flags before it listens.
func TestMainEndsTheProcessWithTheStatusOfTheRun(t *testing.T) {
	savedExit, savedHost, savedArgs := exit, host, os.Args
	t.Cleanup(func() { exit, host, os.Args = savedExit, savedHost, savedArgs })
	var codes []int
	exit = func(code int) { codes = append(codes, code) }
	host = fakeHost()

	os.Args = []string{"outlinestubcmd", "-ca-file", "ca.pem"}
	main()
	os.Args = []string{"outlinestubcmd"}
	main()

	if len(codes) != 2 || codes[0] != 0 || codes[1] != 1 {
		t.Fatalf("main ended the process with %v, want a clean run (0) then a refused run (1)", codes)
	}
}
