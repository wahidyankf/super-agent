//go:build integration

package integration_test

import (
	"context"
	"errors"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// @scenario specs/libs/retriever-connector-kiwi/behaviours/tls-connections.feature: A read over a real TLS connection reaches the server with the endpoint host and an accepted Bearer key
func TestTLSReadReachesServerWithEndpointHostAndAcceptedKey(t *testing.T) {
	srv := serveKiwi(t, newStub(sampleDataset()), nil)
	gw := gatewayAt(t, srv, staticCreds{})
	_, err := gw.ListCollections(context.Background())
	must(t, err)
	seen := srv.Log.all()
	if len(seen) != 1 {
		t.Fatalf("%d requests reached the server, want 1", len(seen))
	}
	if seen[0].ServerName != testHost || seen[0].Host != testHost || !seen[0].Secure {
		t.Errorf("the TLS name is %q and the Host header is %q, want %q for both over TLS", seen[0].ServerName, seen[0].Host, testHost)
	}
	reqs := srv.Stub.Requests()
	if len(reqs) != 1 || !reqs[0].AuthOK {
		t.Errorf("the server did not accept the Bearer key")
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/tls-connections.feature: A client that does not trust the server certificate sends no request and no key
func TestTLSUntrustedClientSendsNoRequestAndNoKey(t *testing.T) {
	srv := serveKiwi(t, newStub(sampleDataset()), nil)
	gw := untrustingGateway(t, srv.Base)
	_, err := gw.ListCollections(context.Background())
	if !errors.Is(err, domain.ErrUnreachable) {
		t.Errorf("the read failed with %v, want the unreachable error", err)
	}
	if n := srv.Log.count(); n != 0 {
		t.Errorf("%d requests reached the server, want none", n)
	}
	if n := len(srv.Stub.Requests()); n != 0 {
		t.Errorf("the server recorded %d requests with a key, want none", n)
	}
}
