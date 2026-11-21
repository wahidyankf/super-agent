package unit_test

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-testkit/jirastub"
)

const (
	jiraEmail  = "name@example.test"
	jiraToken  = "EXAMPLE_TOKEN"
	jiraBase   = "https://jira.invalid"
	jiraMyself = "/rest/api/3/myself"
	jiraSearch = "/rest/api/3/search/jql"
)

// fixedNow is the clock of every unit test of the Jira stub, so no test reads the process clock.
var fixedNow = time.Date(2026, time.November, 22, 9, 0, 0, 0, time.UTC)

func fixedClock() time.Time {
	return fixedNow
}

// fixedRandom is the entropy source of the certificate seam. It yields a fixed byte sequence, so the seam makes the same
// certificate on each run.
type fixedRandom struct {
	state byte
}

func (r *fixedRandom) Read(p []byte) (int, error) {
	for i := range p {
		r.state = r.state*31 + 7
		p[i] = r.state
	}
	return len(p), nil
}

// basicHeader returns the HTTP Basic header of an account and a token.
func basicHeader(email, token string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(email+":"+token))
}

// newJiraStub returns a stub with the synthetic credentials and the fixed clock.
func newJiraStub(t *testing.T) *jirastub.Stub {
	t.Helper()
	stub := jirastub.New(jiraEmail, jiraToken)
	stub.SetClock(fixedClock)
	return stub
}

// sendJira makes one call through the in-process transport of stub, with auth as the Authorization header when it is set.
func sendJira(stub *jirastub.Stub, method, path, auth, body string) (*http.Response, []byte, error) {
	client := &http.Client{Transport: stub.RoundTripper()}
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req, err := http.NewRequestWithContext(context.Background(), method, jiraBase+path, reader)
	if err != nil {
		return nil, nil, err
	}
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	return resp, data, err
}

// @scenario specs/libs/retriever-testkit/behaviours/jira-stub.feature: The stub certificate covers the catalog hosts and loopback for a year
func TestStubCertificateCoversCatalogHostsAndLoopbackForYear(t *testing.T) {
	cert, err := jirastub.NewCertificate(fixedNow, &fixedRandom{})
	if err != nil {
		t.Fatalf("make the certificate: %v", err)
	}
	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatalf("parse the certificate: %v", err)
	}
	for _, host := range append([]string{"127.0.0.1", "::1"}, jirastub.CatalogHosts...) {
		if err := leaf.VerifyHostname(host); err != nil {
			t.Fatalf("the certificate does not cover %s: %v", host, err)
		}
	}
	if !leaf.NotBefore.Equal(fixedNow.Add(-time.Hour)) {
		t.Fatalf("the certificate starts at %v", leaf.NotBefore)
	}
	if !leaf.NotAfter.Equal(fixedNow.Add(365 * 24 * time.Hour)) {
		t.Fatalf("the certificate ends at %v, not one year after %v", leaf.NotAfter, fixedNow)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/jira-stub.feature: A call with a wrong account or token, or no Basic header, is refused
func TestCallWithWrongAccountOrTokenOrNoBasicHeaderIsRefused(t *testing.T) {
	cases := []struct {
		header string
		auth   string
	}{
		{"Basic with the other token", basicHeader(jiraEmail, "OTHER_TOKEN")},
		{"Basic with the other account", basicHeader("other@example.test", jiraToken)},
		{"Bearer with the token", "Bearer " + jiraToken},
		{"no header", ""},
	}
	for _, c := range cases {
		t.Run(c.header, func(t *testing.T) {
			resp, _, err := sendJira(newJiraStub(t), http.MethodGet, jiraMyself, c.auth, "")
			if err != nil {
				t.Fatalf("call: %v", err)
			}
			if resp.StatusCode != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", resp.StatusCode)
			}
		})
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/jira-stub.feature: A path the stub does not offer is answered with HTTP 404
func TestPathStubDoesNotOfferIsAnsweredWith404(t *testing.T) {
	resp, _, err := sendJira(newJiraStub(t), http.MethodGet, "/rest/api/3/unknown", basicHeader(jiraEmail, jiraToken), "")
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/jira-stub.feature: A JQL search returns the page size asked for and the project of the clause
func TestJQLSearchReturnsPageSizeAndProjectOfClause(t *testing.T) {
	cases := []struct {
		jql     string
		size    int
		project string
	}{
		{"project = EXAMPLE", 2, "EXAMPLE"},
		{`project = "EXAMPLE"`, 50, "EXAMPLE"},
	}
	for _, c := range cases {
		t.Run(c.jql, func(t *testing.T) {
			stub := newJiraStub(t)
			stub.SetIssues(jirastub.NumberedIssues("EXAMPLE", 5))
			body := fmt.Sprintf(`{"jql":%q,"maxResults":%d}`, c.jql, c.size)
			_, data, err := sendJira(stub, http.MethodPost, jiraSearch, basicHeader(jiraEmail, jiraToken), body)
			if err != nil {
				t.Fatalf("search: %v", err)
			}
			var reply struct {
				Issues []struct {
					Key string `json:"key"`
				} `json:"issues"`
			}
			if err := json.Unmarshal(data, &reply); err != nil {
				t.Fatalf("decode the page: %v", err)
			}
			if len(reply.Issues) == 0 || len(reply.Issues) > c.size {
				t.Fatalf("the page holds %d issues, want between 1 and %d", len(reply.Issues), c.size)
			}
			for _, issue := range reply.Issues {
				if !strings.HasPrefix(issue.Key, c.project+"-") {
					t.Fatalf("issue %s is not of project %s", issue.Key, c.project)
				}
			}
		})
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/jira-stub.feature: A JQL search answers the page that its next page token names
func TestJQLSearchAnswersPageNamedByNextPageToken(t *testing.T) {
	stub := newJiraStub(t)
	stub.SetIssues(jirastub.NumberedIssues("EXAMPLE", 5))
	body := `{"jql":"project = EXAMPLE","maxResults":2,"nextPageToken":"p2"}`
	_, data, err := sendJira(stub, http.MethodPost, jiraSearch, basicHeader(jiraEmail, jiraToken), body)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	var reply struct {
		Issues []struct {
			Key string `json:"key"`
		} `json:"issues"`
	}
	if err := json.Unmarshal(data, &reply); err != nil {
		t.Fatalf("decode the page: %v", err)
	}
	if len(reply.Issues) == 0 || reply.Issues[0].Key != "EXAMPLE-3" {
		t.Fatalf("the page does not start at the second page: %+v", reply.Issues)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/jira-stub.feature: A JQL search with an unknown next page token answers the first page
func TestJQLSearchWithUnknownNextPageTokenAnswersFirstPage(t *testing.T) {
	stub := newJiraStub(t)
	stub.SetIssues(jirastub.NumberedIssues("EXAMPLE", 5))
	body := `{"jql":"project = EXAMPLE","maxResults":2,"nextPageToken":"bad"}`
	_, data, err := sendJira(stub, http.MethodPost, jiraSearch, basicHeader(jiraEmail, jiraToken), body)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	var reply struct {
		Issues []struct {
			Key string `json:"key"`
		} `json:"issues"`
	}
	if err := json.Unmarshal(data, &reply); err != nil {
		t.Fatalf("decode the page: %v", err)
	}
	if len(reply.Issues) == 0 || reply.Issues[0].Key != "EXAMPLE-1" {
		t.Fatalf("the page does not start at the first issue: %+v", reply.Issues)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/jira-stub.feature: A JQL search reads only a JSON object body within its size cap
func TestJQLSearchReadsOnlyJSONObjectBody(t *testing.T) {
	stub := newJiraStub(t)
	stub.SetIssues(jirastub.NumberedIssues("EXAMPLE", 5))
	resp, data, err := sendJira(stub, http.MethodPost, jiraSearch, basicHeader(jiraEmail, jiraToken), `[{"jql":"project = EXAMPLE"}]`)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if resp.StatusCode == http.StatusOK || strings.Contains(string(data), `"issues"`) {
		t.Fatalf("a JSON list body was answered with a page: status %d, body %s", resp.StatusCode, data)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/jira-stub.feature: The comments list pages from startAt and falls back on a bad value
func TestCommentsListPagesFromStartAtAndFallsBack(t *testing.T) {
	cases := []struct {
		query  string
		offset int
	}{
		{"startAt=1", 1},
		{"startAt=not-a-number", 0},
	}
	for _, c := range cases {
		t.Run(c.query, func(t *testing.T) {
			stub := newJiraStub(t)
			stub.SetIssues([]jirastub.Issue{jirastub.NewIssue("EXAMPLE-1", "Login", nil)})
			comment := func(id string) jirastub.Comment {
				return jirastub.Comment{ID: id, Author: "Fixture Author", Created: "2026-11-05T11:00:00.000+0700",
					Updated: "2026-11-05T11:00:00.000+0700", Body: json.RawMessage(`{"type":"doc"}`)}
			}
			stub.SetComments(map[string][]jirastub.Comment{"EXAMPLE-1": {comment("c1"), comment("c2"), comment("c3")}})
			_, data, err := sendJira(stub, http.MethodGet, "/rest/api/3/issue/EXAMPLE-1/comment?"+c.query, basicHeader(jiraEmail, jiraToken), "")
			if err != nil {
				t.Fatalf("list comments: %v", err)
			}
			var reply struct {
				Comments []struct {
					ID string `json:"id"`
				} `json:"comments"`
			}
			if err := json.Unmarshal(data, &reply); err != nil {
				t.Fatalf("decode the comments: %v", err)
			}
			want := fmt.Sprintf("c%d", c.offset+1)
			if len(reply.Comments) == 0 || reply.Comments[0].ID != want {
				t.Fatalf("the first comment is not %s: %+v", want, reply.Comments)
			}
		})
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/jira-stub.feature: An issue answers each field it holds
func TestIssueAnswersEachFieldItHolds(t *testing.T) {
	stub := newJiraStub(t)
	stub.SetIssues([]jirastub.Issue{jirastub.NewIssue("EXAMPLE-1", "Login", json.RawMessage(`"Steps"`))})
	_, data, err := sendJira(stub, http.MethodGet, "/rest/api/3/issue/EXAMPLE-1", basicHeader(jiraEmail, jiraToken), "")
	if err != nil {
		t.Fatalf("read the issue: %v", err)
	}
	var reply struct {
		Fields struct {
			Summary     string `json:"summary"`
			Description string `json:"description"`
		} `json:"fields"`
	}
	if err := json.Unmarshal(data, &reply); err != nil {
		t.Fatalf("decode the issue: %v", err)
	}
	if reply.Fields.Summary != "Login" || reply.Fields.Description != "Steps" {
		t.Fatalf("the reply holds summary %q and description %q", reply.Fields.Summary, reply.Fields.Description)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/jira-stub.feature: An issue whose description is not JSON is sent as it is
func TestIssueWhoseDescriptionIsNotJSONIsSentAsItIs(t *testing.T) {
	stub := newJiraStub(t)
	stub.SetIssues([]jirastub.Issue{jirastub.NewIssue("EXAMPLE-2", "Notes", json.RawMessage("not json"))})
	resp, data, err := sendJira(stub, http.MethodGet, "/rest/api/3/issue/EXAMPLE-2", basicHeader(jiraEmail, jiraToken), "")
	if err != nil {
		t.Fatalf("read the issue: %v", err)
	}
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(data), "not json") {
		t.Fatalf("status %d, body %s: the description is not sent as text", resp.StatusCode, data)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/jira-stub.feature: A comment whose body is not JSON fails the reply with HTTP 500
func TestCommentWhoseBodyIsNotJSONFailsReplyWith500(t *testing.T) {
	stub := newJiraStub(t)
	stub.SetIssues([]jirastub.Issue{jirastub.NewIssue("EXAMPLE-1", "Login", nil)})
	stub.SetComments(map[string][]jirastub.Comment{"EXAMPLE-1": {{ID: "c1", Body: json.RawMessage("not json")}}})
	resp, _, err := sendJira(stub, http.MethodGet, "/rest/api/3/issue/EXAMPLE-1/comment", basicHeader(jiraEmail, jiraToken), "")
	if err != nil {
		t.Fatalf("list comments: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", resp.StatusCode)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/jira-stub.feature: A hang fault times the client out at once, a drop fault closes it
func TestHangFaultTimesClientOutAtOnce(t *testing.T) {
	stub := newJiraStub(t)
	stub.SetFault(jirastub.Fault{Path: jiraMyself, Hang: true})
	resp, _, err := sendJira(stub, http.MethodGet, jiraMyself, basicHeader(jiraEmail, jiraToken), "")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want the client to give up", err)
	}
	if resp != nil {
		t.Fatalf("a reply arrived for a hung request: status %d", resp.StatusCode)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/jira-stub.feature: A drop fault closes the connection with no reply
func TestDropFaultClosesConnectionWithNoReply(t *testing.T) {
	stub := newJiraStub(t)
	stub.SetFault(jirastub.Fault{Path: jiraMyself, Drop: true})
	resp, _, err := sendJira(stub, http.MethodGet, jiraMyself, basicHeader(jiraEmail, jiraToken), "")
	if err == nil || !strings.Contains(err.Error(), "closed") {
		t.Fatalf("error = %v, want a closed connection", err)
	}
	if resp != nil {
		t.Fatalf("a reply arrived for a dropped request: status %d", resp.StatusCode)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/jira-stub.feature: A fault set for the next matching call answers that call only
func TestFaultSetForNextMatchingCallAnswersThatCallOnly(t *testing.T) {
	stub := newJiraStub(t)
	stub.SetFaultFirst(1, jirastub.Fault{Path: jiraMyself, Status: http.StatusInternalServerError})
	auth := basicHeader(jiraEmail, jiraToken)
	first, _, err := sendJira(stub, http.MethodGet, jiraMyself, auth, "")
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	second, _, err := sendJira(stub, http.MethodGet, jiraMyself, auth, "")
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if first.StatusCode != http.StatusInternalServerError || second.StatusCode != http.StatusOK {
		t.Fatalf("statuses %d and %d, want 500 then 200", first.StatusCode, second.StatusCode)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/jira-stub.feature: A cookie set on the stub is sent on every reply until it is cleared
func TestCookieIsSentOnEveryReplyUntilCleared(t *testing.T) {
	stub := newJiraStub(t)
	stub.SetCookie("EXAMPLE_SESSION")
	auth := basicHeader(jiraEmail, jiraToken)
	var cookies []string
	for range 2 {
		resp, _, err := sendJira(stub, http.MethodGet, jiraMyself, auth, "")
		if err != nil {
			t.Fatalf("call: %v", err)
		}
		cookies = append(cookies, resp.Header.Get("Set-Cookie"))
	}
	stub.SetCookie("")
	resp, _, err := sendJira(stub, http.MethodGet, jiraMyself, auth, "")
	if err != nil {
		t.Fatalf("call after clearing: %v", err)
	}
	if cookies[0] == "" || cookies[1] == "" {
		t.Fatalf("the first two replies carry no cookie: %q", cookies)
	}
	if got := resp.Header.Get("Set-Cookie"); got != "" {
		t.Fatalf("the reply after clearing carries cookie %q", got)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/jira-stub.feature: The request record keeps transport facts, never the header value
func TestRequestRecordKeepsTransportFactsNeverHeaderValue(t *testing.T) {
	stub := newJiraStub(t)
	auth := basicHeader(jiraEmail, jiraToken)
	if _, _, err := sendJira(stub, http.MethodGet, jiraMyself, auth, ""); err != nil {
		t.Fatalf("call: %v", err)
	}
	records := stub.Requests()
	if len(records) != 1 {
		t.Fatalf("the record holds %d requests, want 1", len(records))
	}
	rec := records[0]
	if rec.Method != http.MethodGet || rec.Path != jiraMyself || rec.Status != http.StatusOK {
		t.Fatalf("the record holds method %q, path %q, status %d", rec.Method, rec.Path, rec.Status)
	}
	data, err := json.Marshal(rec)
	if err != nil {
		t.Fatalf("encode the record: %v", err)
	}
	if strings.Contains(string(data), jiraToken) || strings.Contains(string(data), auth) {
		t.Fatalf("the record holds the header value: %s", data)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/jira-stub.feature: The stub configuration reads back from JSON unchanged
func TestStubConfigurationReadsBackFromJSONUnchanged(t *testing.T) {
	stub := newJiraStub(t)
	stub.SetIssues([]jirastub.Issue{jirastub.NewIssue("EXAMPLE-1", "Login", json.RawMessage(`"Steps"`))})
	stub.SetFault(jirastub.Fault{Path: jiraMyself, Status: http.StatusTooManyRequests, RetryAfter: "1"})
	first, err := json.Marshal(stub.Config())
	if err != nil {
		t.Fatalf("write the configuration: %v", err)
	}
	var c jirastub.Config
	if err := json.Unmarshal(first, &c); err != nil {
		t.Fatalf("read the configuration: %v", err)
	}
	reloaded := jirastub.New("", "")
	reloaded.ApplyConfig(c)
	second, err := json.Marshal(reloaded.Config())
	if err != nil {
		t.Fatalf("write the reloaded configuration: %v", err)
	}
	if string(first) != string(second) {
		t.Fatalf("the configuration changed:\n%s\n%s", first, second)
	}
}
