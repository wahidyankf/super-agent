package jirastub_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-testkit/jirastub"
)

func TestConfigRoundTripsTheWholeStubThroughJSON(t *testing.T) {
	source := newStub(t)
	source.SetProjects(jirastub.NumberedProjects(2))
	source.SetIssues([]jirastub.Issue{
		jirastub.NewIssue("EXAMPLE-40", "Plain", nil),
		jirastub.NewIssue("EXAMPLE-41", "Raw", json.RawMessage(`plain text`)),
		jirastub.NewIssue("EXAMPLE-42", "Structured", json.RawMessage(`{"type":"doc","version":1,"content":[]}`)),
	})
	source.SetComments(map[string][]jirastub.Comment{
		jirastub.IssueKey: {{ID: "60001", Author: "Fixture Author", Body: json.RawMessage(`{"type":"doc","version":1,"content":[]}`)}},
	})
	source.SetAccount(jirastub.Account{AccountID: "acc-1", DisplayName: "Fixture Reviewer", AccountType: "atlassian", Active: true})

	data, err := json.Marshal(source.Config())
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	var c jirastub.Config
	if err := json.Unmarshal(data, &c); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	target := newStub(t)
	target.ApplyConfig(c)

	if email, token := target.Credentials(); email != testEmail || token != testToken {
		t.Fatalf("credentials = %q, %q; want the source pair", email, token)
	}
	if !reflect.DeepEqual(target.Projects(), source.Projects()) {
		t.Fatalf("projects = %v, want %v", target.Projects(), source.Projects())
	}
	if !reflect.DeepEqual(target.Issues(), source.Issues()) {
		t.Fatalf("issues differ after the round trip:\n got %+v\nwant %+v", target.Issues(), source.Issues())
	}
	if !reflect.DeepEqual(target.Comments(), source.Comments()) {
		t.Fatalf("comments = %v, want %v", target.Comments(), source.Comments())
	}
	auth := basicHeader(testEmail, testToken)
	for _, path := range []string{"/rest/api/3/myself", "/rest/api/3/issue/EXAMPLE-41", "/rest/api/3/issue/" + jirastub.IssueKey + "/comment"} {
		wantStatus, wantBody := call(t, source, http.MethodGet, path, "", auth)
		gotStatus, gotBody := call(t, target, http.MethodGet, path, "", auth)
		if gotStatus != wantStatus || string(gotBody) != string(wantBody) {
			t.Fatalf("%s: round-tripped stub answered %d %s, source answered %d %s", path, gotStatus, gotBody, wantStatus, wantBody)
		}
	}
}

func TestConfigMovesOnlyNotJSONDescriptionsOutOfTheIssues(t *testing.T) {
	s := newStub(t)
	s.SetIssues([]jirastub.Issue{
		jirastub.NewIssue("EXAMPLE-40", "Null", nil),
		jirastub.NewIssue("EXAMPLE-41", "Raw", json.RawMessage(`plain text`)),
		{ID: "40001", Key: "EXAMPLE-42", Summary: "No description"},
	})
	c := s.Config()
	if !reflect.DeepEqual(c.RawDescriptions, map[string]string{"EXAMPLE-41": "plain text"}) {
		t.Fatalf("raw descriptions = %v, want only EXAMPLE-41", c.RawDescriptions)
	}
	if c.Issues[1].Description != nil {
		t.Fatalf("the raw description stayed in the issue: %q", c.Issues[1].Description)
	}
	if string(c.Issues[0].Description) != "null" || c.Issues[2].Description != nil {
		t.Fatalf("null and absent descriptions = %q and %q, want null and nil", c.Issues[0].Description, c.Issues[2].Description)
	}
}

func TestConfigCarriesTheFaultCountAndTheCookie(t *testing.T) {
	source := newStub(t)
	source.SetCookie("JSESSIONID")
	source.SetFaultFirst(1, jirastub.Fault{Status: http.StatusServiceUnavailable})
	c := source.Config()
	if c.Cookie != "JSESSIONID=synthetic-session; Path=/" || c.Fault == nil || c.Fault.Status != http.StatusServiceUnavailable || c.FaultLeft != 1 {
		t.Fatalf("config cookie %q, fault %+v, left %d; want the cookie, the 503 fault, and 1 reply", c.Cookie, c.Fault, c.FaultLeft)
	}

	target := newStub(t)
	target.ApplyConfig(c)
	auth := basicHeader(testEmail, testToken)
	first := httptest.NewRecorder()
	target.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "https://jira.example.test/rest/api/3/myself", nil))
	second := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "https://jira.example.test/rest/api/3/myself", nil)
	req.Header.Set("Authorization", auth)
	target.ServeHTTP(second, req)
	if first.Code != http.StatusServiceUnavailable || second.Code != http.StatusOK {
		t.Fatalf("statuses = %d then %d, want 503 then 200", first.Code, second.Code)
	}
	if second.Header().Get("Set-Cookie") != "JSESSIONID=synthetic-session; Path=/" {
		t.Fatalf("the applied cookie is not sent: %q", second.Header().Get("Set-Cookie"))
	}
}

func TestConfigReportsNoFaultWhenNoneIsSet(t *testing.T) {
	if c := newStub(t).Config(); c.Fault != nil {
		t.Fatalf("a stub with no fault reports %+v", c.Fault)
	}
}

func TestApplyConfigGivesTheFaultCountItsMeaning(t *testing.T) {
	tests := []struct {
		name         string
		fault        *jirastub.Fault
		left         int
		wantStatuses []int
	}{
		{name: "no fault clears the replay", wantStatuses: []int{200, 200, 200}},
		{name: "a zero count faults every reply", fault: &jirastub.Fault{Status: 500}, wantStatuses: []int{500, 500, 500}},
		{name: "a count faults that many replies", fault: &jirastub.Fault{Status: 500}, left: 2, wantStatuses: []int{500, 500, 200}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newStub(t)
			s.SetFault(jirastub.Fault{Status: 418})
			s.ApplyConfig(jirastub.Config{Email: testEmail, Token: testToken, Fault: tt.fault, FaultLeft: tt.left})
			auth := basicHeader(testEmail, testToken)
			var got []int
			for range tt.wantStatuses {
				status, _ := call(t, s, http.MethodGet, "/rest/api/3/myself", "", auth)
				got = append(got, status)
			}
			if !reflect.DeepEqual(got, tt.wantStatuses) {
				t.Fatalf("statuses = %v, want %v", got, tt.wantStatuses)
			}
		})
	}
}

func TestNewIssueNumbersItsIDFromTheKey(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		desc     json.RawMessage
		wantID   string
		wantDesc string
	}{
		{name: "a key number sets the ID", key: "EXAMPLE-12", wantID: "20012", wantDesc: "null"},
		{name: "a description is kept as given", key: "OPS-3", desc: json.RawMessage(`{"type":"doc"}`), wantID: "20003", wantDesc: `{"type":"doc"}`},
		{name: "a key with no number gets the base ID", key: "LOOSE", wantID: "20000", wantDesc: "null"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := jirastub.NewIssue(tt.key, "Summary", tt.desc)
			if is.ID != tt.wantID || is.Key != tt.key || is.Summary != "Summary" || is.Type != "Task" || is.Status != "To Do" {
				t.Fatalf("issue = %+v", is)
			}
			if is.Priority != "Medium" || is.Reporter != "Fixture Author" || is.Labels == nil || len(is.Labels) != 0 {
				t.Fatalf("defaults = priority %q, reporter %q, labels %#v", is.Priority, is.Reporter, is.Labels)
			}
			if string(is.Description) != tt.wantDesc {
				t.Fatalf("description = %s, want %s", is.Description, tt.wantDesc)
			}
		})
	}
}

func TestNumberedFixturesCountFromOne(t *testing.T) {
	issues := jirastub.NumberedIssues("EXAMPLE", 3)
	if len(issues) != 3 {
		t.Fatalf("numbered issues = %d, want 3", len(issues))
	}
	for i, is := range issues {
		n := i + 1
		wantKey, wantID, wantSummary := fmt.Sprintf("EXAMPLE-%d", n), fmt.Sprintf("%d", 20000+n), fmt.Sprintf("Numbered issue %d", n)
		if is.Key != wantKey || is.ID != wantID || is.Summary != wantSummary {
			t.Fatalf("issue %d = %+v, want %s %s %q", n, is, wantKey, wantID, wantSummary)
		}
	}
	if got := jirastub.NumberedIssues("EXAMPLE", 0); len(got) != 0 {
		t.Fatalf("zero numbered issues = %d", len(got))
	}

	projects := jirastub.NumberedProjects(2)
	if len(projects) != 2 || projects[0].Key != "PRJ1" || projects[1].ID != "30002" || projects[1].Name != "Numbered project 2" {
		t.Fatalf("numbered projects = %+v", projects)
	}
	if projects[0].Type != "software" || projects[0].Style != "classic" {
		t.Fatalf("numbered project shape = %+v", projects[0])
	}
	if got := jirastub.NumberedProjects(0); len(got) != 0 {
		t.Fatalf("zero numbered projects = %d", len(got))
	}
}

func TestSetCredentialsReplacesThePairTheStubAccepts(t *testing.T) {
	s := newStub(t)
	s.SetCredentials("next@example.test", "next-token")
	if email, token := s.Credentials(); email != "next@example.test" || token != "next-token" {
		t.Fatalf("credentials = %q, %q", email, token)
	}
	if status, _ := call(t, s, http.MethodGet, "/rest/api/3/myself", "", basicHeader(testEmail, testToken)); status != http.StatusUnauthorized {
		t.Fatalf("the replaced pair is still accepted: %d", status)
	}
	if status, _ := call(t, s, http.MethodGet, "/rest/api/3/myself", "", basicHeader("next@example.test", "next-token")); status != http.StatusOK {
		t.Fatalf("the new pair is refused: %d", status)
	}
}
