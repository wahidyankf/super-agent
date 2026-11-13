//go:build integration

package kiwi

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/tests/support/kiwistub"
)

// startTLSStub serves the sample dataset over a real TLS listener, and returns it with a client that trusts it.
func startTLSStub(t *testing.T) (*kiwistub.Stub, *http.Client) {
	t.Helper()
	stub := kiwistub.New(sentinelKey, kiwistub.WithDataset(sampleDataset()))
	if err := stub.Start(); err != nil {
		t.Fatalf("start the TLS stub: %v", err)
	}
	t.Cleanup(stub.Close)
	client, err := stub.Client()
	if err != nil {
		t.Fatalf("stub client: %v", err)
	}
	return stub, client
}

// TestTLSReadReachesTheStubWithTheEndpointHostAndTheKey checks a read over a real TLS connection: the TLS name and the Host
// header are the endpoint host, and the Bearer key is accepted.
func TestTLSReadReachesTheStubWithTheEndpointHostAndTheKey(t *testing.T) {
	stub, client := startTLSStub(t)
	gw, err := NewWithBase(testSourceAt(t, domain.AccessRead), stub.URL(), client, &staticCreds{}, WithMinDelay(0))
	if err != nil {
		t.Fatalf("NewWithBase: %v", err)
	}
	cols, err := gw.ListCollections(context.Background())
	if err != nil {
		t.Fatalf("ListCollections over TLS: %v", err)
	}
	if len(cols) != 2 {
		t.Fatalf("collections = %d, want 2", len(cols))
	}
	reqs := stub.Requests()
	if len(reqs) != 1 || reqs[0].Host != testHost || !reqs[0].AuthOK || reqs[0].Verb != http.MethodPost {
		t.Fatalf("requests = %+v, want one authenticated POST with the endpoint host", reqs)
	}
}

// TestUntrustedCertificateSendsNoRequestAndNoKey checks that a client which does not trust the stub's certificate fails
// the handshake, so no request and no key reach the server.
func TestUntrustedCertificateSendsNoRequestAndNoKey(t *testing.T) {
	stub, _ := startTLSStub(t)
	gw, err := NewWithBase(testSourceAt(t, domain.AccessRead), stub.URL(), &http.Client{}, &staticCreds{}, WithMinDelay(0))
	if err != nil {
		t.Fatalf("NewWithBase: %v", err)
	}
	_, err = gw.ListCollections(context.Background())
	if err == nil {
		t.Fatalf("a read succeeded against an untrusted certificate")
	}
	if strings.Contains(err.Error(), sentinelKey) {
		t.Fatalf("error %q carries the key", err.Error())
	}
	if n := len(stub.Requests()); n != 0 {
		t.Fatalf("%d requests reached the server before the handshake was verified", n)
	}
}
