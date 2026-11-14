//go:build integration

package outline

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-testkit/outlinestub"
)

// TestATLSHandshakeFailureOnAWriteIsUnreachable covers finding m2. It opens a real loopback socket, so it carries the integration tag. A certificate the client does not trust fails the TLS
// handshake before any byte of the request is sent, so the write's outcome is known: the wiki never saw it, and the
// error is the same unreachable error a dial failure gives, not ErrWriteOutcomeUnknown.
func TestATLSHandshakeFailureOnAWriteIsUnreachable(t *testing.T) {
	stub := outlinestub.New(sampleOptions()...)
	server, err := outlinestub.NewTLSServer(stub)
	if err != nil {
		t.Fatalf("NewTLSServer: %v", err)
	}
	defer server.Close()
	// The client uses the system roots, which do not hold the stub certificate, so the handshake fails.
	client := &http.Client{}
	gw, err := NewWithBase(testSourceAt(t, testHost), server.URL, client, &staticCreds{}, WithMinDelay(0))
	if err != nil {
		t.Fatalf("NewWithBase: %v", err)
	}
	_, err = gw.Write(context.Background(), domain.OpDocumentsCreate, domain.NewArgs(map[string]any{"title": "Plan"}))
	if !errors.Is(err, domain.ErrUnreachable) {
		t.Fatalf("error = %v, want ErrUnreachable", err)
	}
	if errors.Is(err, domain.ErrWriteOutcomeUnknown) {
		t.Fatal("a handshake failure reported an unknown outcome, but no request was sent")
	}
	if n := len(stub.Requests()); n != 0 {
		t.Fatalf("requests received = %d, want 0", n)
	}
}
