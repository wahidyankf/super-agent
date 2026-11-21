//go:build integration

package integration_test

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-testkit/jirastub"
)

const (
	jiraEmail    = "name@example.test"
	jiraToken    = "EXAMPLE_TOKEN"
	jiraMyself   = "/rest/api/3/myself"
	jiraLoopback = "127.0.0.1"
)

// jiraGet makes one GET of path against the started TLS server of srv, under ctx, with the Basic header of the account.
func jiraGet(ctx context.Context, t *testing.T, srv *httptest.Server, path string) (*http.Response, error) {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+path, nil)
	if err != nil {
		t.Fatalf("build the request: %v", err)
	}
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(jiraEmail+":"+jiraToken)))
	return srv.Client().Do(req)
}

// @scenario specs/libs/retriever-testkit/behaviours/jira-stub.feature: The stub certificate covers the catalog hosts and loopback for a year
func TestStubCertificateOverRealTLSCoversCatalogHostsForYear(t *testing.T) {
	stub := jirastub.New(jiraEmail, jiraToken)
	srv, err := jirastub.NewTLSServer(stub)
	if err != nil {
		t.Fatalf("start the TLS server: %v", err)
	}
	defer srv.Close()
	leaf := srv.Certificate()
	for _, host := range append([]string{jiraLoopback}, jirastub.CatalogHosts...) {
		if err := leaf.VerifyHostname(host); err != nil {
			t.Fatalf("the certificate does not cover %s: %v", host, err)
		}
	}
	const year = 365 * 24 * time.Hour
	validity := leaf.NotAfter.Sub(leaf.NotBefore)
	if validity < year || validity > year+24*time.Hour {
		t.Fatalf("the certificate is valid for %v, want one year", validity)
	}
	if left := time.Until(leaf.NotAfter); left < year-time.Hour || left > year+time.Hour {
		t.Fatalf("the certificate ends in %v, want about one year from now", left)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/jira-stub.feature: A hang fault times the client out at once, a drop fault closes it
func TestHangFaultTimesClientOutOverRealConnection(t *testing.T) {
	stub := jirastub.New(jiraEmail, jiraToken)
	stub.SetFault(jirastub.Fault{Path: jiraMyself, Hang: true})
	srv, err := jirastub.NewTLSServer(stub)
	if err != nil {
		t.Fatalf("start the TLS server: %v", err)
	}
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	resp, err := jiraGet(ctx, t, srv, jiraMyself)
	if err == nil {
		resp.Body.Close()
		t.Fatalf("the client got a reply with status %d from a hang fault", resp.StatusCode)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("the client gave up with %v, want its timeout", err)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/jira-stub.feature: A drop fault closes the connection with no reply
func TestDropFaultClosesConnectionWithNoReplyOverRealConnection(t *testing.T) {
	stub := jirastub.New(jiraEmail, jiraToken)
	stub.SetFault(jirastub.Fault{Path: jiraMyself, Drop: true})
	srv, err := jirastub.NewTLSServer(stub)
	if err != nil {
		t.Fatalf("start the TLS server: %v", err)
	}
	defer srv.Close()
	resp, err := jiraGet(context.Background(), t, srv, jiraMyself)
	if err == nil {
		resp.Body.Close()
		t.Fatalf("the client got a reply with status %d from a drop fault", resp.StatusCode)
	}
	message := strings.ToLower(err.Error())
	if !strings.Contains(message, "eof") && !strings.Contains(message, "reset") && !strings.Contains(message, "closed") {
		t.Fatalf("the client error %q does not report a closed connection", err)
	}
}
