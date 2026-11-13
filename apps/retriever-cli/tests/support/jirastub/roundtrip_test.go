package jirastub_test

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

// TestTheRoundTripperAnswersStubHostsAndRefusesAnyOtherHost proves the in-process transport serves a .invalid name and a
// loopback address, and that it never answers a name that could be a real site.
func TestTheRoundTripperAnswersStubHostsAndRefusesAnyOtherHost(t *testing.T) {
	client := &http.Client{Transport: newStub(t).RoundTripper()}

	for _, host := range []string{"jira-stub.invalid", "127.0.0.1:8443", "[::1]:8443"} {
		resp, err := client.Get("https://" + host + "/rest/api/3/myself")
		if err != nil {
			t.Fatalf("host %s: %v", host, err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("host %s: status %d, want 401 for a request with no credentials", host, resp.StatusCode)
		}
	}

	for _, host := range []string{"jira.example.test", "api.atlassian.com", "example.com"} {
		if _, err := client.Get("https://" + host + "/rest/api/3/myself"); err == nil {
			t.Fatalf("host %s: the in-process stub answered a host it must refuse", host)
		}
	}
}

// TestTheRoundTripperAnswersWithTheAccountAndRecordsWithoutTheSecret proves a signed request gets the account, and the
// recorder keeps the method, path, and query, but never the token.
func TestTheRoundTripperAnswersWithTheAccountAndRecordsWithoutTheSecret(t *testing.T) {
	stub := newStub(t)
	client := &http.Client{Transport: stub.RoundTripper()}

	req, err := http.NewRequest(http.MethodGet, "https://jira-stub.invalid/rest/api/3/myself?x=1", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.SetBasicAuth(testEmail, testToken)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "displayName") {
		t.Fatalf("status %d, body %q: want the signed-in account", resp.StatusCode, body)
	}

	recorded := stub.Requests()
	if len(recorded) != 1 {
		t.Fatalf("recorded %d requests, want 1", len(recorded))
	}
	got := recorded[0]
	if got.Method != http.MethodGet || got.Path != "/rest/api/3/myself" || got.Query != "x=1" || !got.AuthOK {
		t.Fatalf("recorded %+v, want GET /rest/api/3/myself?x=1 with the matching credentials", got)
	}
	if strings.Contains(got.Path+got.Query, testToken) {
		t.Fatal("the recorder holds the token")
	}
}
