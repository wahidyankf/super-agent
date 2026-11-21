package unit_test

import (
	"context"
	"errors"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-testkit/kiwistub"
)

// The TLS name itself is proven at the integration level, where a real handshake checks it. Here the request's Host
// header and its Bearer key are checked on the in-memory connection.

// @scenario specs/libs/retriever-connector-kiwi/behaviours/tls-connections.feature: A read over a real TLS connection reaches the server with the endpoint host and an accepted Bearer key
func TestReadReachesServerWithEndpointHostAndAcceptedKey(t *testing.T) {
	r := newRig(t, sampleDataset())
	_, err := r.gw.ListCollections(context.Background())
	must(t, err)
	reqs := r.stub.Requests()
	if len(reqs) != 1 {
		t.Fatalf("%d requests were sent, want 1", len(reqs))
	}
	if reqs[0].Host != testHost {
		t.Errorf("the Host header is %q, want %q", reqs[0].Host, testHost)
	}
	if !reqs[0].AuthOK {
		t.Errorf("the server did not accept the Bearer key")
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/tls-connections.feature: A client that does not trust the server certificate sends no request and no key
func TestClientThatDoesNotTrustCertificateSendsNothing(t *testing.T) {
	stub := kiwistub.New(sentinelKey, kiwistub.WithDataset(sampleDataset()))
	gw := hermeticGateway(t, handshakeRT{}, staticCreds{}, newFakeClock())
	_, err := gw.ListCollections(context.Background())
	if !errors.Is(err, domain.ErrUnreachable) {
		t.Errorf("the read failed with %v, want the unreachable error", err)
	}
	if n := len(stub.Requests()); n != 0 {
		t.Errorf("%d requests reached the server, want none", n)
	}
}
