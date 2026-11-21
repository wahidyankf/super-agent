//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-testkit/kiwistub"
)

const kiwiKey = "EXAMPLE_KEY"

// @scenario specs/libs/retriever-testkit/behaviours/kiwi-stub.feature: The stub listens on the loopback and certifies the catalog hosts
func TestStartedStubListensOnLoopbackAndCertifiesCatalogHosts(t *testing.T) {
	stub := kiwistub.New(kiwiKey)
	if err := stub.Start(); err != nil {
		t.Fatalf("start the stub: %v", err)
	}
	defer stub.Close()
	if !strings.HasPrefix(stub.URL(), "https://127.0.0.1:") {
		t.Fatalf("the stub URL %q is not a loopback TLS address", stub.URL())
	}
	block, _ := pem.Decode(stub.CAPEM())
	if block == nil {
		t.Fatal("the CA file holds no PEM certificate")
	}
	leaf, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("parse the certificate: %v", err)
	}
	for _, host := range append([]string{"127.0.0.1"}, kiwistub.CatalogHosts...) {
		if err := leaf.VerifyHostname(host); err != nil {
			t.Fatalf("the certificate does not cover %s: %v", host, err)
		}
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/kiwi-stub.feature: A client made from the stub trusts the stub only
func TestClientMadeFromStartedStubConnectsAndOtherTLSServerIsRefused(t *testing.T) {
	stub := kiwistub.New(kiwiKey)
	if err := stub.Start(); err != nil {
		t.Fatalf("start the stub: %v", err)
	}
	defer stub.Close()
	client, err := stub.Client()
	if err != nil {
		t.Fatalf("make the client: %v", err)
	}
	body, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "Product.filter", "params": map[string]any{"query": map[string]any{}},
	})
	if err != nil {
		t.Fatalf("encode the call: %v", err)
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, stub.URL()+kiwistub.Endpoint, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("build the request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+kiwiKey)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("the client did not connect to the stub: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("the stub answered %d, want 200", resp.StatusCode)
	}

	other := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer other.Close()
	otherResp, err := client.Get(other.URL)
	if err == nil {
		otherResp.Body.Close()
		t.Fatal("the client trusted a TLS server that is not the stub")
	}
	var unknown x509.UnknownAuthorityError
	if !errors.As(err, &unknown) {
		t.Fatalf("the other server was refused with %v, want an unknown certificate authority", err)
	}
}
