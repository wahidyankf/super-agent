//go:build integration

package graphql

import (
	"context"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// TestListenBindsLoopbackOnly checks that a real listener is bound to 127.0.0.1 and nothing wider.
func TestListenBindsLoopbackOnly(t *testing.T) {
	l, err := Listen(0)
	if err != nil {
		t.Fatalf("Listen(0): %v", err)
	}
	defer l.Close()
	tcp, ok := l.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("address type = %T, want *net.TCPAddr", l.Addr())
	}
	if !tcp.IP.Equal(net.IPv4(127, 0, 0, 1)) {
		t.Fatalf("listener IP = %v, want 127.0.0.1", tcp.IP)
	}
}

// liveRequest is one request to the running server.
type liveRequest struct {
	method, host, origin, contentType string
}

// doLive sends one request to the running server and returns the status, the headers, and the body.
func doLive(t *testing.T, ctx context.Context, client *http.Client, base string, r liveRequest) (*http.Response, string) {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, r.method, base, strings.NewReader(`{"query":"{ collections { id name } }"}`))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if r.host != "" {
		req.Host = r.host
	}
	if r.origin != "" {
		req.Header.Set("Origin", r.origin)
	}
	if r.contentType != "" {
		req.Header.Set("Content-Type", r.contentType)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", r.method, base, err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp, string(data)
}

// TestServeRoundTrip runs the real server on loopback and checks the Host allowlist, the request form,
// and graceful shutdown on context cancel.
func TestServeRoundTrip(t *testing.T) {
	gw := &countingGateway{collections: []domain.Collection{{ID: "col-1", Name: "Engineering"}}}
	l, err := Listen(0)
	if err != nil {
		t.Fatalf("Listen(0): %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	handler, err := NewHandler(
		Deps{Service: oneOutlineSources(gw), Logger: discardLogger()},
		WithListener(l),
	)
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	served := make(chan error, 1)
	go func() { served <- Serve(ctx, l, handler) }()

	client := &http.Client{Timeout: 10 * time.Second}
	defer client.CloseIdleConnections()
	base := "http://" + net.JoinHostPort("127.0.0.1", strconv.Itoa(port)) + "/graphql"

	t.Run("an allowed host answers the query", func(t *testing.T) {
		resp, body := doLive(t, ctx, client, base, liveRequest{method: http.MethodPost, contentType: "application/json"})
		if resp.StatusCode != http.StatusOK || !strings.Contains(body, "Engineering") {
			t.Fatalf("status %d body %s", resp.StatusCode, body)
		}
		if got := gw.calls.Load(); got != 1 {
			t.Fatalf("gateway calls = %d, want 1", got)
		}
	})
	t.Run("a DNS-rebinding Host is refused", func(t *testing.T) {
		resp, _ := doLive(t, ctx, client, base, liveRequest{
			method: http.MethodPost, host: "evil.example.com:" + strconv.Itoa(port), contentType: "application/json",
		})
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", resp.StatusCode)
		}
	})
	t.Run("a browser Origin is refused with no CORS header", func(t *testing.T) {
		resp, _ := doLive(t, ctx, client, base, liveRequest{
			method: http.MethodPost, origin: "https://evil.example.com", contentType: "application/json",
		})
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", resp.StatusCode)
		}
		for name := range resp.Header {
			if strings.HasPrefix(strings.ToLower(name), "access-control-") {
				t.Fatalf("CORS header %q on a refusal", name)
			}
		}
	})
	t.Run("GET is refused", func(t *testing.T) {
		resp, _ := doLive(t, ctx, client, base, liveRequest{method: http.MethodGet, contentType: "application/json"})
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want 405", resp.StatusCode)
		}
	})

	cancel()
	select {
	case err := <-served:
		if err != nil {
			t.Fatalf("Serve after cancel: %v, want nil", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Serve did not return after the context was cancelled")
	}
}
