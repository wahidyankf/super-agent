package cli

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// fakeServe stands in for the composition root's GraphQL server. It records how it was called and announces addr.
type fakeServe struct {
	calls int
	port  int
	addr  string
	err   error
}

func (f *fakeServe) serve(_ context.Context, port int, ready func(string)) error {
	f.calls++
	f.port = port
	if f.err != nil {
		return f.err
	}
	ready(f.addr)
	return nil
}

func TestServePrintsTheListeningLineOnce(t *testing.T) {
	fake := &fakeServe{addr: "127.0.0.1:4567"}

	code, out, errOut := runRetriever(t, []string{"serve"}, outlineSetup{serve: fake.serve})

	if code != 0 || out != "retriever serve listening on 127.0.0.1:4567\n" {
		t.Fatalf("want exactly the listening line and exit 0, got %q (exit %d, stderr %q)", out, code, errOut)
	}
}

func TestServeDefaultsToPortZero(t *testing.T) {
	fake := &fakeServe{addr: "127.0.0.1:1"}

	runRetriever(t, []string{"serve"}, outlineSetup{serve: fake.serve})

	if fake.calls != 1 || fake.port != 0 {
		t.Fatalf("want one call on port 0, got %d calls on port %d", fake.calls, fake.port)
	}
}

func TestServeTakesThePortOption(t *testing.T) {
	fake := &fakeServe{addr: "127.0.0.1:8123"}

	code, _, errOut := runRetriever(t, []string{"serve", "--port", "8123"}, outlineSetup{serve: fake.serve})

	if code != 0 || fake.port != 8123 {
		t.Fatalf("want port 8123 and exit 0, got port %d (exit %d, stderr %q)", fake.port, code, errOut)
	}
}

func TestServeRejectsAPortOutsideTheRange(t *testing.T) {
	for _, port := range []string{"70000", "-1"} {
		fake := &fakeServe{addr: "127.0.0.1:1"}

		code, _, errOut := runRetriever(t, []string{"serve", "--port", port}, outlineSetup{serve: fake.serve})

		if code != 2 || !strings.Contains(errOut, "must be between 0 and 65535") || fake.calls != 0 {
			t.Fatalf("port %s: want a range error and no server, got %q (exit %d, calls %d)", port, errOut, code, fake.calls)
		}
	}
}

func TestServeRejectsANonNumericPort(t *testing.T) {
	fake := &fakeServe{addr: "127.0.0.1:1"}

	code, _, errOut := runRetriever(t, []string{"serve", "--port", "abc"}, outlineSetup{serve: fake.serve})

	if code != 2 || !strings.Contains(errOut, "invalid value for --port: abc") || fake.calls != 0 {
		t.Fatalf("want an invalid-port error and no server, got %q (exit %d)", errOut, code)
	}
}

func TestServeHasNoHostOption(t *testing.T) {
	fake := &fakeServe{addr: "127.0.0.1:1"}

	code, _, errOut := runRetriever(t, []string{"serve", "--host", "0.0.0.0"}, outlineSetup{serve: fake.serve})

	if code != 2 || !strings.Contains(errOut, "unrecognized argument: --host") || fake.calls != 0 {
		t.Fatalf("the host must not be configurable, got %q (exit %d)", errOut, code)
	}
}

func TestServeReportsAFailureBeforeListeningAndExitsOneTwentyFive(t *testing.T) {
	fake := &fakeServe{err: errors.New("listen tcp 127.0.0.1:80: bind: permission denied")}

	code, out, errOut := runRetriever(t, []string{"serve", "--port", "80"}, outlineSetup{serve: fake.serve})

	if code != 125 || !strings.Contains(errOut, "permission denied") || strings.Contains(out, "listening") {
		t.Fatalf("want the failure and no listening line, got %q / %q (exit %d)", out, errOut, code)
	}
}

func TestServeWithoutWiringReportsTheSetupError(t *testing.T) {
	setup := outlineSetup{setupErr: errors.New("no repository holds the working directory")}

	code, out, errOut := runRetriever(t, []string{"serve"}, setup)

	if code != 2 || !strings.Contains(errOut, "no repository holds the working directory") || strings.Contains(out, "listening") {
		t.Fatalf("want the setup error and no server, got %q / %q (exit %d)", out, errOut, code)
	}
}

func TestServeHelpShowsThePortOption(t *testing.T) {
	code, out, _ := runRetriever(t, []string{"serve", "--help"}, outlineSetup{})

	if code != 0 || !strings.Contains(out, "--port <port>") {
		t.Fatalf("serve help must show the port option, got %q (exit %d)", out, code)
	}
}
