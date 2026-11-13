package graphql

import (
	"context"
	"net/http"
	"testing"
)

// TestListenRejectsPortsOutsideTheTCPRange checks that Listen refuses an out-of-range port without binding.
func TestListenRejectsPortsOutsideTheTCPRange(t *testing.T) {
	for _, port := range []int{-1, 65536} {
		l, err := Listen(port)
		if err == nil {
			if l != nil {
				_ = l.Close()
			}
			t.Fatalf("Listen(%d) error = nil, want an error", port)
		}
	}
}

// TestServeRefusesNonLoopbackListener checks that Serve never accepts from a listener bound beyond loopback.
func TestServeRefusesNonLoopbackListener(t *testing.T) {
	for _, ip := range []string{"0.0.0.0", "::", "198.51.100.10"} {
		t.Run(ip, func(t *testing.T) {
			l := tcpListener(ip, testPort)
			handler := http.NotFoundHandler()
			if err := Serve(context.Background(), l, handler); err == nil {
				t.Fatal("Serve error = nil, want a refusal for a non-loopback listener")
			}
			if n := l.accepts.Load(); n != 0 {
				t.Fatalf("Serve accepted %d times on a non-loopback listener, want 0", n)
			}
		})
	}
}

// TestServeRefusesMissingArguments checks that Serve refuses a nil listener or a nil handler.
func TestServeRefusesMissingArguments(t *testing.T) {
	if err := Serve(context.Background(), nil, http.NotFoundHandler()); err == nil {
		t.Error("Serve with a nil listener error = nil, want an error")
	}
	if err := Serve(context.Background(), tcpListener("127.0.0.1", testPort), nil); err == nil {
		t.Error("Serve with a nil handler error = nil, want an error")
	}
}

// TestLoopbackAddrIsTheBindAddress checks that the address retriever serve binds is the loopback address for its port,
// and that a port outside the TCP range has no address at all.
func TestLoopbackAddrIsTheBindAddress(t *testing.T) {
	addr, err := LoopbackAddr(testPort)
	if err != nil {
		t.Fatalf("LoopbackAddr(%d): %v", testPort, err)
	}
	if got := addr.String(); got != allowedHost {
		t.Fatalf("LoopbackAddr(%d) = %q, want %q", testPort, got, allowedHost)
	}
	if _, err := LoopbackAddr(65536); err == nil {
		t.Fatal("LoopbackAddr(65536) error = nil, want an error")
	}
}
