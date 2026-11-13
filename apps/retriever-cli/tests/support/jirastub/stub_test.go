package jirastub_test

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/tests/support/jirastub"
)

// Synthetic credentials of the tests. The e-mail uses the reserved example.com domain, and the token is a sentinel.
const (
	testEmail = "fixture-reviewer@example.com"
	testToken = "sentinel-token-5d2b8e-do-not-print"
)

// newStub returns a stub that accepts the synthetic credentials.
func newStub(t *testing.T) *jirastub.Stub {
	t.Helper()
	return jirastub.New(testEmail, testToken)
}

// basicHeader returns the Authorization header for the given credentials.
func basicHeader(email, token string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(email+":"+token))
}

// call sends one request to the stub and returns the status and body. An empty auth sends no Authorization header.
func call(t *testing.T, s *jirastub.Stub, method, path, body, auth string) (int, []byte) {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, "https://jira.example.test"+path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	return rec.Code, rec.Body.Bytes()
}

// decode decodes a JSON reply into a generic map, failing the test on error.
func decode(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("reply is not a JSON object: %v (%q)", err, data)
	}
	return out
}

func TestTheStubAcceptsOnlyTheSyntheticCredentials(t *testing.T) {
	tests := []struct {
		name   string
		auth   string
		status int
		authOK bool
	}{
		{name: "the right pair is answered", auth: basicHeader(testEmail, testToken), status: http.StatusOK, authOK: true},
		{name: "a wrong token is refused", auth: basicHeader(testEmail, "other-token"), status: http.StatusUnauthorized},
		{name: "a wrong e-mail is refused", auth: basicHeader("other@example.com", testToken), status: http.StatusUnauthorized},
		{name: "a Bearer header is refused", auth: "Bearer " + testToken, status: http.StatusUnauthorized},
		{name: "no header is refused", auth: "", status: http.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newStub(t)
			status, _ := call(t, s, http.MethodGet, "/rest/api/3/myself", "", tt.auth)
			if status != tt.status {
				t.Fatalf("status = %d, want %d", status, tt.status)
			}
			reqs := s.Requests()
			if len(reqs) != 1 || reqs[0].AuthOK != tt.authOK {
				t.Fatalf("recorded AuthOK = %+v, want %v", reqs, tt.authOK)
			}
		})
	}
}

// TestTheRecorderNeverHoldsTheSecret proves that neither the header value nor the token appears in any recorded request.
func TestTheRecorderNeverHoldsTheSecret(t *testing.T) {
	s := newStub(t)
	call(t, s, http.MethodGet, "/rest/api/3/myself", "", basicHeader(testEmail, testToken))
	call(t, s, http.MethodGet, "/rest/api/3/myself", "", basicHeader(testEmail, "wrong"))
	data, err := json.Marshal(s.Requests())
	if err != nil {
		t.Fatalf("marshal recorder: %v", err)
	}
	for _, secret := range []string{testToken, basicHeader(testEmail, testToken), base64.StdEncoding.EncodeToString([]byte(testEmail + ":" + testToken))} {
		if strings.Contains(string(data), secret) {
			t.Fatalf("recorder holds a secret: %s", data)
		}
	}
}

func TestMyselfAnswersTheSignedInAccount(t *testing.T) {
	s := newStub(t)
	status, body := call(t, s, http.MethodGet, "/rest/api/3/myself", "", basicHeader(testEmail, testToken))
	if status != http.StatusOK {
		t.Fatalf("status = %d", status)
	}
	got := decode(t, body)
	if got["displayName"] != "Fixture Reviewer" || got["accountType"] != "atlassian" || got["active"] != true {
		t.Fatalf("account = %v", got)
	}
}

func TestProjectSearchPagesAndFilters(t *testing.T) {
	s := newStub(t)
	auth := basicHeader(testEmail, testToken)

	_, body := call(t, s, http.MethodGet, "/rest/api/3/project/search?startAt=0&maxResults=1", "", auth)
	first := decode(t, body)
	if values, _ := first["values"].([]any); len(values) != 1 || first["isLast"] != false {
		t.Fatalf("first page = %v", first)
	}

	_, body = call(t, s, http.MethodGet, "/rest/api/3/project/search?startAt=1&maxResults=5", "", auth)
	second := decode(t, body)
	if second["isLast"] != true {
		t.Fatalf("second page is not last: %v", second)
	}

	_, body = call(t, s, http.MethodGet, "/rest/api/3/project/search?query=ops", "", auth)
	if !strings.Contains(string(body), jirastub.OtherProject) || strings.Contains(string(body), jirastub.ProjectKey+"\"") {
		t.Fatalf("query filter did not select %s only: %s", jirastub.OtherProject, body)
	}

	_, body = call(t, s, http.MethodGet, "/rest/api/3/project/search?query=nothing-matches", "", auth)
	if !strings.Contains(string(body), `"values":[]`) {
		t.Fatalf("an empty result must hold an empty array, got %s", body)
	}
}

func TestJQLSearchFiltersByProjectAndPagesByToken(t *testing.T) {
	s := newStub(t)
	auth := basicHeader(testEmail, testToken)

	_, body := call(t, s, http.MethodPost, "/rest/api/3/search/jql",
		`{"jql":"project = \"EXAMPLE\" ORDER BY key ASC","maxResults":2}`, auth)
	page1 := decode(t, body)
	issues, _ := page1["issues"].([]any)
	if len(issues) != 2 || page1["isLast"] != false || page1["nextPageToken"] != "p2" {
		t.Fatalf("first page = %v", page1)
	}

	_, body = call(t, s, http.MethodPost, "/rest/api/3/search/jql",
		`{"jql":"project = EXAMPLE","maxResults":50,"nextPageToken":"p2"}`, auth)
	page2 := decode(t, body)
	if page2["isLast"] != true || page2["nextPageToken"] != nil {
		t.Fatalf("last page = %v", page2)
	}
	for _, is := range append(issues, page2["issues"].([]any)...) {
		key, _ := is.(map[string]any)["key"].(string)
		if !strings.HasPrefix(key, jirastub.ProjectKey+"-") {
			t.Fatalf("a row of another project came back: %s", key)
		}
	}
}

func TestJQLSearchRefusesEmptyUnboundedAndInvalidText(t *testing.T) {
	s := newStub(t)
	auth := basicHeader(testEmail, testToken)
	for _, jql := range []string{``, `order by key`, `project = EXAMPLE AND summary ~ invalid`} {
		body, _ := json.Marshal(map[string]any{"jql": jql})
		status, _ := call(t, s, http.MethodPost, "/rest/api/3/search/jql", string(body), auth)
		if status != http.StatusBadRequest {
			t.Errorf("jql %q: status = %d, want 400", jql, status)
		}
	}
	reqs := s.Requests()
	if reqs[0].Body["jql"] != "" || reqs[0].Method != http.MethodPost {
		t.Fatalf("the recorder did not keep the JQL body: %+v", reqs[0])
	}
}

func TestIssueEndpointAnswersOneIssueAndNulls(t *testing.T) {
	s := newStub(t)
	auth := basicHeader(testEmail, testToken)

	status, body := call(t, s, http.MethodGet, "/rest/api/3/issue/"+jirastub.IssueKey, "", auth)
	if status != http.StatusOK {
		t.Fatalf("issue status = %d", status)
	}
	fields := decode(t, body)["fields"].(map[string]any)
	if fields["summary"] != "Login fails on second try" || fields["parent"].(map[string]any)["key"] != jirastub.ParentKey {
		t.Fatalf("issue fields = %v", fields)
	}
	if fields["description"] == nil {
		t.Fatalf("issue %s has no description", jirastub.IssueKey)
	}

	_, body = call(t, s, http.MethodGet, "/rest/api/3/issue/"+jirastub.UnassignedKey, "", auth)
	fields = decode(t, body)["fields"].(map[string]any)
	if fields["description"] != nil || fields["assignee"] != nil {
		t.Fatalf("an empty description and assignee must be null: %v", fields)
	}

	status, _ = call(t, s, http.MethodGet, "/rest/api/3/issue/"+jirastub.MissingKey, "", auth)
	if status != http.StatusNotFound {
		t.Fatalf("missing issue status = %d, want 404", status)
	}
}

func TestCommentsAreListedInOrderWithPagesAndAnEmptyList(t *testing.T) {
	s := newStub(t)
	auth := basicHeader(testEmail, testToken)

	_, body := call(t, s, http.MethodGet, "/rest/api/3/issue/"+jirastub.IssueKey+"/comment", "", auth)
	all := decode(t, body)
	if all["total"] != float64(2) || len(all["comments"].([]any)) != 2 {
		t.Fatalf("comments = %v", all)
	}

	_, body = call(t, s, http.MethodGet, "/rest/api/3/issue/"+jirastub.IssueKey+"/comment?startAt=1&maxResults=1", "", auth)
	one := decode(t, body)
	if got := one["comments"].([]any); len(got) != 1 || got[0].(map[string]any)["id"] != "20002" {
		t.Fatalf("second comment page = %v", one)
	}

	_, body = call(t, s, http.MethodGet, "/rest/api/3/issue/"+jirastub.OtherKey+"/comment", "", auth)
	if !strings.Contains(string(body), `"comments":[]`) {
		t.Fatalf("an issue with no comments must hold an empty list: %s", body)
	}

	status, _ := call(t, s, http.MethodGet, "/rest/api/3/issue/"+jirastub.MissingKey+"/comment", "", auth)
	if status != http.StatusNotFound {
		t.Fatalf("comments of a missing issue = %d, want 404", status)
	}
}

func TestTheRemovedSearchPathAnswersGone(t *testing.T) {
	s := newStub(t)
	status, _ := call(t, s, http.MethodGet, "/rest/api/3/search?jql=project%3DEXAMPLE", "", basicHeader(testEmail, testToken))
	if status != http.StatusGone {
		t.Fatalf("status = %d, want 410", status)
	}
}

func TestUnknownPathsAndMethodsAreNotFound(t *testing.T) {
	s := newStub(t)
	auth := basicHeader(testEmail, testToken)
	for _, c := range []struct{ method, path string }{
		{http.MethodGet, "/rest/api/3/unknown"},
		{http.MethodPost, "/rest/api/3/myself"},
		{http.MethodGet, "/rest/api/3/search/jql"},
		{http.MethodGet, "/rest/api/3/issue/EXAMPLE-12/transitions"},
	} {
		if status, _ := call(t, s, c.method, c.path, "", auth); status != http.StatusNotFound {
			t.Errorf("%s %s = %d, want 404", c.method, c.path, status)
		}
	}
}

func TestStatusFaultsReplaceTheReplyWithAnErrorBody(t *testing.T) {
	tests := []struct {
		name   string
		fault  jirastub.Fault
		status int
	}{
		{name: "401", fault: jirastub.Fault{Status: 401}, status: 401},
		{name: "403", fault: jirastub.Fault{Status: 403}, status: 403},
		{name: "410", fault: jirastub.Fault{Status: 410}, status: 410},
		{name: "500", fault: jirastub.Fault{Status: 500}, status: 500},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newStub(t)
			s.SetFault(tt.fault)
			status, body := call(t, s, http.MethodGet, "/rest/api/3/myself", "", basicHeader(testEmail, testToken))
			if status != tt.status {
				t.Fatalf("status = %d, want %d", status, tt.status)
			}
			if _, ok := decode(t, body)["errorMessages"]; !ok {
				t.Fatalf("an error fault must carry an error body: %s", body)
			}
		})
	}
}

func TestRedirectAndRetryFaultsSetTheirHeaders(t *testing.T) {
	s := newStub(t)
	auth := basicHeader(testEmail, testToken)

	s.SetFault(jirastub.Fault{Location: "https://login.example.com/"})
	req := httptest.NewRequest(http.MethodGet, "https://jira.example.test/rest/api/3/myself", nil)
	req.Header.Set("Authorization", auth)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "https://login.example.com/" {
		t.Fatalf("redirect = %d %q", rec.Code, rec.Header().Get("Location"))
	}

	s.SetFault(jirastub.Fault{Status: 429, RetryAfter: "7"})
	req = httptest.NewRequest(http.MethodGet, "https://jira.example.test/rest/api/3/myself", nil)
	req.Header.Set("Authorization", auth)
	rec = httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") != "7" {
		t.Fatalf("429 = %d, Retry-After %q", rec.Code, rec.Header().Get("Retry-After"))
	}
}

func TestOversizeNonJSONAndMissingMemberFaults(t *testing.T) {
	auth := basicHeader(testEmail, testToken)

	s := newStub(t)
	s.SetFault(jirastub.Fault{Oversize: true})
	_, body := call(t, s, http.MethodGet, "/rest/api/3/myself", "", auth)
	if len(body) <= 10<<20 {
		t.Fatalf("oversize reply is %d bytes, want more than 10 MiB", len(body))
	}

	s.SetFault(jirastub.Fault{NonJSON: true})
	_, body = call(t, s, http.MethodGet, "/rest/api/3/myself", "", auth)
	if json.Valid(body) {
		t.Fatalf("non-JSON fault sent JSON: %s", body)
	}

	s.SetFault(jirastub.Fault{MissingMember: true})
	_, body = call(t, s, http.MethodGet, "/rest/api/3/issue/"+jirastub.IssueKey, "", auth)
	if _, ok := decode(t, body)["fields"]; ok {
		t.Fatalf("missing-member fault still sends fields: %s", body)
	}
}

func TestTokenFaultsAndDepthBombSendTheirShapes(t *testing.T) {
	auth := basicHeader(testEmail, testToken)
	s := newStub(t)

	s.SetFault(jirastub.Fault{RepeatToken: true})
	_, body := call(t, s, http.MethodPost, "/rest/api/3/search/jql", `{"jql":"project = EXAMPLE"}`, auth)
	if got := decode(t, body)["nextPageToken"]; got != "p0" {
		t.Fatalf("repeated token = %v, want p0", got)
	}

	s.SetFault(jirastub.Fault{EmptyPageWithToken: true})
	_, body = call(t, s, http.MethodPost, "/rest/api/3/search/jql", `{"jql":"project = EXAMPLE"}`, auth)
	page := decode(t, body)
	if issues := page["issues"].([]any); len(issues) != 0 || page["nextPageToken"] == nil {
		t.Fatalf("empty page with token = %v", page)
	}

	s.SetFault(jirastub.Fault{DepthBomb: true})
	_, body = call(t, s, http.MethodGet, "/rest/api/3/issue/"+jirastub.IssueKey, "", auth)
	if !json.Valid(body) || !strings.Contains(string(body), `"type":"panel"`) {
		t.Fatalf("depth bomb reply is not a nested ADF document")
	}
}

func TestAFaultWithAPathAppliesOnlyToThatPath(t *testing.T) {
	s := newStub(t)
	auth := basicHeader(testEmail, testToken)
	s.SetFault(jirastub.Fault{Path: "/rest/api/3/issue/", Status: 503})

	if status, _ := call(t, s, http.MethodGet, "/rest/api/3/myself", "", auth); status != http.StatusOK {
		t.Fatalf("myself under an issue fault = %d, want 200", status)
	}
	if status, _ := call(t, s, http.MethodGet, "/rest/api/3/issue/"+jirastub.IssueKey, "", auth); status != http.StatusServiceUnavailable {
		t.Fatalf("issue under an issue fault = %d, want 503", status)
	}

	s.ClearFault()
	if status, _ := call(t, s, http.MethodGet, "/rest/api/3/issue/"+jirastub.IssueKey, "", auth); status != http.StatusOK {
		t.Fatalf("issue after ClearFault = %d, want 200", status)
	}
}

func TestTheRecorderKeepsOrderMethodPathQueryAndStatus(t *testing.T) {
	s := newStub(t)
	auth := basicHeader(testEmail, testToken)
	call(t, s, http.MethodGet, "/rest/api/3/project/search?query=ops", "", auth)
	call(t, s, http.MethodGet, "/rest/api/3/issue/"+jirastub.MissingKey, "", auth)
	reqs := s.Requests()
	if len(reqs) != 2 {
		t.Fatalf("recorded %d requests, want 2", len(reqs))
	}
	if reqs[0].Path != "/rest/api/3/project/search" || reqs[0].Query != "query=ops" || reqs[0].Status != http.StatusOK {
		t.Fatalf("first record = %+v", reqs[0])
	}
	if reqs[1].Status != http.StatusNotFound || !reqs[1].AuthOK {
		t.Fatalf("second record = %+v", reqs[1])
	}
}
