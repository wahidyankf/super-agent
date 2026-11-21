package unit_test

import (
	"bytes"
	"context"
	"encoding/pem"
	"errors"
	"io/fs"
	"net"
	"os"
	"strings"
	"sync"
	"testing"

	kiwicmd "github.com/wahidyankf/super-agent/libs/retriever-testkit/stubcmd/kiwi"
)

// kcmdListener is a listening socket that opens no connection. Its Close is recorded, so a test sees that the stub stopped.
type kcmdListener struct {
	closed chan struct{}
	once   sync.Once
}

func newKcmdListener() *kcmdListener {
	return &kcmdListener{closed: make(chan struct{})}
}

func (l *kcmdListener) Accept() (net.Conn, error) {
	<-l.closed
	return nil, net.ErrClosed
}

func (l *kcmdListener) Close() error {
	l.once.Do(func() { close(l.closed) })
	return nil
}

func (l *kcmdListener) Addr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 9443}
}

// kcmdStopNow is the signal wait of a run to its end: the context it returns is already done.
func kcmdStopNow(parent context.Context, _ ...os.Signal) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(parent)
	cancel()
	return ctx, cancel
}

// kcmdHost is the host of one unit run of the Kiwi command. Its environment and its input files are maps, its writes are
// recorded, its listener counts its opens, and its entropy is fixed, so the run reads no real clock, file, or entropy.
type kcmdHost struct {
	env      map[string]string
	files    map[string][]byte
	written  map[string][]byte
	writeErr error
	listens  int
	listener *kcmdListener
	notify   func(parent context.Context, signals ...os.Signal) (context.Context, context.CancelFunc)
}

// newKcmdHost returns a host with the synthetic key set and no files.
func newKcmdHost() *kcmdHost {
	return &kcmdHost{
		env:      map[string]string{"KIWISTUB_API_KEY": kiwiKey},
		files:    map[string][]byte{},
		written:  map[string][]byte{},
		listener: newKcmdListener(),
	}
}

// host returns the command's host of fakes.
func (f *kcmdHost) host() kiwicmd.Host {
	notify := f.notify
	if notify == nil {
		notify = kcmdStopNow
	}
	return kiwicmd.Host{
		Getenv: func(name string) string { return f.env[name] },
		Now:    fixedClock,
		ReadFile: func(name string) ([]byte, error) {
			data, ok := f.files[name]
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
		Listen: func(_, _ string) (net.Listener, error) {
			f.listens++
			return f.listener, nil
		},
		Random: &fixedRandom{},
		Notify: notify,
	}
}

// kcmdOrdered is the standard output of a run. It notes whether the CA file was already written when the first byte was
// printed, which is the order the scenario states.
type kcmdOrdered struct {
	host    *kcmdHost
	printed bytes.Buffer
	first   bool
	seen    bool
}

func (w *kcmdOrdered) Write(p []byte) (int, error) {
	if !w.seen {
		w.seen = true
		w.first = len(w.host.written) > 0
	}
	return w.printed.Write(p)
}

// kcmdRun runs the Kiwi command with args and host h, with both streams in memory, and returns its status and streams.
func kcmdRun(args []string, h kiwicmd.Host) (int, string, string) {
	var out, errOut bytes.Buffer
	status := kiwicmd.Run(args, &out, &errOut, h)
	return status, out.String(), errOut.String()
}

// kcmdStart runs the Kiwi command in a goroutine. It returns once the command waits for its stop signal, which it does
// after it has printed its address. The returned function sends the stop signal and returns the status of the run.
func kcmdStart(args []string, f *kcmdHost) (func() int, *bytes.Buffer, *bytes.Buffer) {
	waiting := make(chan context.CancelFunc, 1)
	f.notify = func(parent context.Context, _ ...os.Signal) (context.Context, context.CancelFunc) {
		ctx, cancel := context.WithCancel(parent)
		waiting <- cancel
		return ctx, cancel
	}
	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
	done := make(chan int, 1)
	go func() {
		done <- kiwicmd.Run(args, out, errOut, f.host())
	}()
	cancel := <-waiting
	return func() int {
		cancel()
		return <-done
	}, out, errOut
}

// @scenario specs/libs/retriever-testkit/behaviours/kiwi-stub-command.feature: The command refuses to start without a certificate file path
func TestKiwiCommandRefusesToStartWithoutCertificatePath(t *testing.T) {
	f := newKcmdHost()
	status, out, errOut := kcmdRun(nil, f.host())
	if status != 1 || !strings.Contains(errOut, "the -ca-file flag is required") {
		t.Fatalf("status %d and stderr %q, want 1 with the -ca-file message", status, errOut)
	}
	if out != "" || f.listens != 0 {
		t.Fatalf("the command printed %q and listened %d times, want no port", out, f.listens)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/kiwi-stub-command.feature: The command refuses to start when the key variable is not set
func TestKiwiCommandRefusesToStartWhenKeyVariableIsNotSet(t *testing.T) {
	f := newKcmdHost()
	delete(f.env, "KIWISTUB_API_KEY")
	status, out, errOut := kcmdRun([]string{"-ca-file", "ca.pem"}, f.host())
	if status != 1 || !strings.Contains(errOut, "KIWISTUB_API_KEY must hold the synthetic key") {
		t.Fatalf("status %d and stderr %q, want 1 with the key message", status, errOut)
	}
	if out != "" || f.listens != 0 {
		t.Fatalf("the command printed %q and listened %d times, want no port", out, f.listens)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/kiwi-stub-command.feature: The command refuses a flag that it does not know and names the flag
func TestKiwiCommandRefusesAnUnknownFlagAndNamesIt(t *testing.T) {
	f := newKcmdHost()
	status, out, errOut := kcmdRun([]string{"-no-such-flag"}, f.host())
	if status != 1 || !strings.Contains(errOut, "-no-such-flag") {
		t.Fatalf("status %d and stderr %q, want 1 with the flag named", status, errOut)
	}
	if out != "" || f.listens != 0 {
		t.Fatalf("the command printed %q and listened %d times, want no port", out, f.listens)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/kiwi-stub-command.feature: The command writes the certificate and then prints its listening address
func TestKiwiCommandWritesTheCertificateAndThenPrintsItsAddress(t *testing.T) {
	f := newKcmdHost()
	printed := &kcmdOrdered{host: f}
	var errOut bytes.Buffer
	status := kiwicmd.Run([]string{"-ca-file", "ca.pem"}, printed, &errOut, f.host())
	if status != 0 {
		t.Fatalf("status %d, stderr %q; want a clean run", status, errOut.String())
	}
	if got, want := printed.printed.String(), "kiwi stub listening on https://127.0.0.1:9443\n"; got != want {
		t.Fatalf("printed %q, want %q", got, want)
	}
	if !printed.first {
		t.Fatal("the address was printed before the certificate file was written")
	}
	block, _ := pem.Decode(f.written["ca.pem"])
	if block == nil || block.Type != "CERTIFICATE" {
		t.Fatalf("the certificate file holds no PEM certificate: %q", f.written["ca.pem"])
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/kiwi-stub-command.feature: A failure after start stops the stub and prints to standard error
func TestKiwiCommandFailureAfterStartStopsTheStubAndPrintsToStandardError(t *testing.T) {
	f := newKcmdHost()
	f.writeErr = errors.New("disk full")
	status, out, errOut := kcmdRun([]string{"-ca-file", "ca.pem"}, f.host())
	if status != 1 || !strings.Contains(errOut, "write the CA file: disk full") {
		t.Fatalf("status %d and stderr %q, want 1 with the write error", status, errOut)
	}
	// The serving goroutine closes the listener once the stub is closed, so the test waits for that close.
	<-f.listener.closed
	if out != "" {
		t.Fatalf("the command printed %q before the failure", out)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/kiwi-stub-command.feature: A clean run exits with status 0
func TestKiwiCleanRunExitsWithStatusZeroOnItsStopSignal(t *testing.T) {
	f := newKcmdHost()
	stop, out, errOut := kcmdStart([]string{"-ca-file", "ca.pem"}, f)
	status := stop()
	if status != 0 {
		t.Fatalf("status %d after the stop signal, stderr %q; want 0", status, errOut.String())
	}
	if !strings.HasPrefix(out.String(), "kiwi stub listening on ") {
		t.Fatalf("printed %q, want the listening line", out.String())
	}
	// The serving goroutine closes the listener once the stub is closed, so the test waits for that close.
	<-f.listener.closed
}
