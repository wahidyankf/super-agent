//go:build integration

package jira

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-testkit/jirastub"
)

// tlsStub starts the stub on a TLS loopback port and returns it with the server. The server closes with the test.
func tlsStub(t *testing.T, email, token string) (*jirastub.Stub, *httptest.Server) {
	t.Helper()
	stub := jirastub.New(email, token)
	server, err := jirastub.NewTLSServer(stub)
	if err != nil {
		t.Fatalf("NewTLSServer: %v", err)
	}
	t.Cleanup(server.Close)
	return stub, server
}

// writeCA writes the certificate of server as a PEM CA file in a fresh directory, and returns its path.
func writeCA(t *testing.T, server *httptest.Server) string {
	t.Helper()
	return writePEM(t, server.Certificate().Raw)
}

// writeUnrelatedCA writes a PEM CA file for a self-signed certificate that the stub never serves. A client that trusts only
// this file must refuse the stub's certificate.
func writeUnrelatedCA(t *testing.T) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "unrelated"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("CreateCertificate: %v", err)
	}
	return writePEM(t, der)
}

// writePEM writes one DER certificate as a PEM file in a fresh directory, and returns the path.
func writePEM(t *testing.T, der []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ca.pem")
	block := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	if err := os.WriteFile(path, block, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

// seamGateway builds the gateway through the test seam, the way the composition root does: the base and the CA file come from
// the environment names, and the client trusts only that CA.
func seamGateway(t *testing.T, server *httptest.Server, secrets Secrets, clock *fakeClock) *Gateway {
	t.Helper()
	env := map[string]string{BaseURLEnv: server.URL, CAFileEnv: writeCA(t, server)}
	seam, err := seamFromEnvDisk(func(name string) string { return env[name] })
	if err != nil || !seam.Enabled {
		t.Fatalf("TestSeamFromEnv: %v (enabled %v)", err, seam.Enabled)
	}
	gw, err := NewWithBase(testSource(t), seam.BaseURL, seam.Client, secrets,
		WithMinDelay(0), WithClock(clock.Now), WithSleep(clock.Sleep))
	if err != nil {
		t.Fatalf("NewWithBase: %v", err)
	}
	return gw
}

// TestTheSeamReadsOverTLSWithTheCAFile proves the handshake succeeds against the endpoint host name with the CA file, and the
// signed-in account is read with the Basic credentials.
func TestTheSeamReadsOverTLSWithTheCAFile(t *testing.T) {
	stub, server := tlsStub(t, sentinelEmail, sentinelToken)
	gw := seamGateway(t, server, newSecrets(), newFakeClock())
	rec, err := readOp(gw, domain.OpJiraWhoami, nil)
	if err != nil {
		t.Fatalf("whoami over TLS: %v", err)
	}
	if rec.Str("displayName") != "Fixture Reviewer" {
		t.Fatalf("display name %q", rec.Str("displayName"))
	}
	reqs := stub.Requests()
	if len(reqs) != 1 || !reqs[0].AuthOK || reqs[0].Path != "/rest/api/3/myself" {
		t.Fatalf("requests %+v, want one authorized GET of myself", reqs)
	}
}

// TestAWrongCertificateIsRefusedAsUnreachable proves a CA file that does not sign the server's certificate stops the call before
// any credential is sent, and the text names Jira.
func TestAWrongCertificateIsRefusedAsUnreachable(t *testing.T) {
	stub, server := tlsStub(t, sentinelEmail, sentinelToken)
	env := map[string]string{BaseURLEnv: server.URL, CAFileEnv: writeUnrelatedCA(t)}
	seam, err := seamFromEnvDisk(func(name string) string { return env[name] })
	if err != nil {
		t.Fatalf("TestSeamFromEnv: %v", err)
	}
	gw, err := NewWithBase(testSource(t), seam.BaseURL, seam.Client, newSecrets(), WithMinDelay(0), WithSleep(noSleep))
	if err != nil {
		t.Fatalf("NewWithBase: %v", err)
	}
	_, err = readOp(gw, domain.OpJiraWhoami, nil)
	if !errors.Is(err, domain.ErrUnreachable) || !strings.Contains(err.Error(), "Jira") || strings.Contains(err.Error(), "Outline") {
		t.Fatalf("err = %v, want the Jira unreachable text", err)
	}
	if n := len(stub.Requests()); n != 0 {
		t.Fatalf("the stub saw %d requests; a failed handshake sends none", n)
	}
}

// TestTheFullReadSetRunsOverTLS runs every gateway method and read against the TLS stub. Each request is authorized and inside
// the table, the search sends its JQL as one body value, and the documents are the stub's.
func TestTheFullReadSetRunsOverTLS(t *testing.T) {
	stub, server := tlsStub(t, sentinelEmail, sentinelToken)
	gw := seamGateway(t, server, newSecrets(), newFakeClock())
	ctx := context.Background()

	collections, err := gw.ListCollections(ctx)
	if err != nil || len(collections) != 2 {
		t.Fatalf("ListCollections: %d, %v", len(collections), err)
	}
	nodes, err := gw.CollectionTree(ctx, collectionID(t, "project-EXAMPLE"))
	if err != nil || len(nodes) != 4 {
		t.Fatalf("CollectionTree: %d, %v", len(nodes), err)
	}
	doc, err := gw.Document(ctx, documentID(t, "issue-EXAMPLE-12"))
	if err != nil || doc.Title != "Login fails on second try" {
		t.Fatalf("Document: %+v, %v", doc, err)
	}
	jql := "project = \"EXAMPLE\"\nAND summary ~ \"login & retry\" ORDER BY key"
	if _, err := readOp(gw, domain.OpJiraIssuesSearch, map[string]any{"jql": jql, "limit": 10}); err != nil {
		t.Fatalf("search: %v", err)
	}
	if _, err := readOp(gw, domain.OpJiraCommentsList, map[string]any{"issueKey": "EXAMPLE-12"}); err != nil {
		t.Fatalf("comments: %v", err)
	}
	if _, err := readOp(gw, domain.OpJiraProjectsList, map[string]any{"query": "EXA"}); err != nil {
		t.Fatalf("projects: %v", err)
	}

	// One request per call: projects (ListCollections), the tree search, the issue, the search, the comments, and the projects
	// search again. A different count means a call sent an extra request or skipped one.
	reqs := stub.Requests()
	if len(reqs) != 6 {
		t.Fatalf("%d requests recorded, want 6 (one per call)", len(reqs))
	}
	searchSeen := false
	for _, req := range reqs {
		if !req.AuthOK {
			t.Fatalf("request %s %s was not authorized", req.Method, req.Path)
		}
		if !inTable(req.Method, req.Path) {
			t.Fatalf("request %s %s is outside the table", req.Method, req.Path)
		}
		if req.Path == "/rest/api/3/search/jql" && req.Body["jql"] == jql {
			searchSeen = true
		}
	}
	if !searchSeen {
		t.Fatal("the search did not send its JQL as one body value")
	}
}

// TestAFaultOverTLSIsMappedAndRetriedOnlyWhenAllowed proves the status rules over the real transport: a 401 is final, and a 429
// with Retry-After is waited for three times and then refused, with the waits measured on the fake clock.
func TestAFaultOverTLSIsMappedAndRetriedOnlyWhenAllowed(t *testing.T) {
	stub, server := tlsStub(t, sentinelEmail, sentinelToken)
	clock := newFakeClock()
	gw := seamGateway(t, server, newSecrets(), clock)

	stub.SetFault(jirastub.Fault{Status: 401})
	if _, err := readOp(gw, domain.OpJiraWhoami, nil); !errors.Is(err, domain.ErrKeyRejected) {
		t.Fatalf("401: err = %v, want ErrKeyRejected", err)
	}

	stub.SetFault(jirastub.Fault{Status: 429, RetryAfter: "1"})
	if _, err := readOp(gw, domain.OpJiraWhoami, nil); !errors.Is(err, domain.ErrSourceRateLimited) {
		t.Fatalf("429: err = %v, want ErrSourceRateLimited", err)
	}
	if waits := clock.recordedSleeps(); len(waits) != 3 || waits[0] != time.Second {
		t.Fatalf("waits = %v, want three of 1s", waits)
	}
	stub.ClearFault()
}

// TestARemovedPathAndARedirectAndAnOversizeReplyAreRefused proves the 410, redirect, and size rules hold over TLS, and the texts
// name Jira.
func TestARemovedPathAndARedirectAndAnOversizeReplyAreRefused(t *testing.T) {
	stub, server := tlsStub(t, sentinelEmail, sentinelToken)
	gw := seamGateway(t, server, newSecrets(), newFakeClock())

	stub.SetFault(jirastub.Fault{Path: "/rest/api/3/search/jql", Status: 410})
	_, err := readOp(gw, domain.OpJiraIssuesSearch, map[string]any{"jql": `project = "EXAMPLE"`})
	if !errors.Is(err, domain.ErrOperationUnavailable) || err.Error() != "this Jira server does not offer the endpoint" {
		t.Fatalf("410: err = %v", err)
	}

	stub.SetFault(jirastub.Fault{Status: 302, Location: "https://evil.invalid/login"})
	if _, err := readOp(gw, domain.OpJiraWhoami, nil); !errors.Is(err, domain.ErrRedirected) || !strings.Contains(err.Error(), "Jira") {
		t.Fatalf("302: err = %v", err)
	}

	stub.SetFault(jirastub.Fault{Oversize: true})
	if _, err := readOp(gw, domain.OpJiraWhoami, nil); !errors.Is(err, domain.ErrReplyTooLarge) {
		t.Fatalf("oversize: err = %v", err)
	}
	stub.ClearFault()
}

// TestAWrongTokenIsSentOnceAndRefused proves a refused token is one request, not a retry, and the stub records no secret.
func TestAWrongTokenIsSentOnceAndRefused(t *testing.T) {
	stub, server := tlsStub(t, sentinelEmail, "a-different-token-value")
	secrets := newSecrets()
	gw := seamGateway(t, server, secrets, newFakeClock())
	if _, err := readOp(gw, domain.OpJiraWhoami, nil); !errors.Is(err, domain.ErrKeyRejected) {
		t.Fatalf("err = %v, want ErrKeyRejected", err)
	}
	reqs := stub.Requests()
	if len(reqs) != 1 || reqs[0].AuthOK {
		t.Fatalf("requests %+v, want one request that did not match", reqs)
	}
	if got := secrets.reads(); strings.Join(got, ",") != "key,identity" {
		t.Fatalf("reads %v, want one key and one identity read", got)
	}
}

// seamFromEnvDisk reads the seam through getenv with the real CA reader. Only an integration test calls it.
func seamFromEnvDisk(getenv func(string) string) (Seam, error) {
	return TestSeamFromEnvWithFiles(getenv, os.ReadFile)
}
