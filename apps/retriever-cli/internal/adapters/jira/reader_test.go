package jira

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// readOp runs one operation of the gateway with the given raw arguments, as the use case does after binding.
func readOp(gw *Gateway, op domain.OperationID, values map[string]any) (domain.Record, error) {
	return gw.Read(context.Background(), op, domain.NewArgs(values))
}

// fieldNames returns the names of an object record's fields, in order.
func fieldNames(rec domain.Record) []string {
	var names []string
	for _, f := range rec.Fields() {
		names = append(names, f.Name)
	}
	return names
}

// itemNames returns the names of an item's fields, in order.
func itemNames(item domain.Record) []string {
	return fieldNames(item)
}

// queryOf parses the query string of a recorded request.
func queryOf(t *testing.T, raw string) url.Values {
	t.Helper()
	q, err := url.ParseQuery(raw)
	if err != nil {
		t.Fatalf("ParseQuery(%q): %v", raw, err)
	}
	return q
}

// TestWhoamiShowsOnlyTheThreeFields proves the account record holds the display name, the account type, and the active flag,
// and never the e-mail address or the account ID that the stub also sends.
func TestWhoamiShowsOnlyTheThreeFields(t *testing.T) {
	r := newRig(t)
	rec, err := readOp(r.gw, domain.OpJiraWhoami, nil)
	if err != nil {
		t.Fatalf("whoami: %v", err)
	}
	want := []string{"displayName", "accountType", "active"}
	if got := fieldNames(rec); !reflect.DeepEqual(got, want) {
		t.Fatalf("fields = %v, want %v", got, want)
	}
	if rec.Str("displayName") != "Fixture Reviewer" || rec.Str("accountType") != "atlassian" {
		t.Fatalf("display name %q and type %q", rec.Str("displayName"), rec.Str("accountType"))
	}
	if active, _ := rec.Get("active"); active != true {
		t.Fatalf("active = %v, want true", active)
	}
}

// TestWhoamiNeedsItsRequiredMembers proves a reply without the display name or the active flag is an unexpected reply.
func TestWhoamiNeedsItsRequiredMembers(t *testing.T) {
	for _, body := range []string{`{"accountType":"atlassian","active":true}`, `{"displayName":"A","accountType":"atlassian"}`, `null`} {
		p := &probe{script: func(int, *http.Request) (*http.Response, error) {
			return reply(http.StatusOK, body), nil
		}}
		gw := probeGateway(t, p, newFakeClock(), newSecrets())
		if _, err := readOp(gw, domain.OpJiraWhoami, nil); !errors.Is(err, domain.ErrUnexpectedResponse) {
			t.Fatalf("reply %s: err = %v, want ErrUnexpectedResponse", body, err)
		}
	}
}

// TestProjectsListSendsOnePageWithAnEncodedQuery proves one request is sent for the page asked, with the query encoded.
func TestProjectsListSendsOnePageWithAnEncodedQuery(t *testing.T) {
	r := newRig(t)
	rec, err := readOp(r.gw, domain.OpJiraProjectsList, map[string]any{"query": "EXA", "limit": 2, "offset": 0})
	if err != nil {
		t.Fatalf("projects list: %v", err)
	}
	reqs := r.stub.Requests()
	if len(reqs) != 1 || reqs[0].Path != "/rest/api/3/project/search" {
		t.Fatalf("requests %+v, want one GET of the project search", reqs)
	}
	q := queryOf(t, reqs[0].Query)
	if q.Get("startAt") != "0" || q.Get("maxResults") != "2" || q.Get("query") != "EXA" {
		t.Fatalf("query %v, want startAt 0, maxResults 2, query EXA", q)
	}
	items := rec.Items()
	if len(items) != 1 || items[0].Str("key") != "EXAMPLE" || items[0].Str("name") != "Example Org" {
		t.Fatalf("items %+v, want the one matching project", items)
	}
	if got := itemNames(items[0]); !reflect.DeepEqual(got, []string{"key", "name", "type", "style"}) {
		t.Fatalf("item fields = %v", got)
	}
	if items[0].Str("type") != "software" || items[0].Str("style") != "classic" {
		t.Fatalf("type %q style %q", items[0].Str("type"), items[0].Str("style"))
	}
}

// TestProjectsListDefaultsAndAnEmptyResult proves the default page is 25, with no query member when none is given, and an empty
// match is an empty list.
func TestProjectsListDefaultsAndAnEmptyResult(t *testing.T) {
	r := newRig(t)
	rec, err := readOp(r.gw, domain.OpJiraProjectsList, nil)
	if err != nil {
		t.Fatalf("default projects list: %v", err)
	}
	q := queryOf(t, r.stub.Requests()[0].Query)
	if q.Get("maxResults") != "25" || q.Has("query") {
		t.Fatalf("query %v, want maxResults 25 and no query member", q)
	}
	if len(rec.Items()) != 2 {
		t.Fatalf("items = %d, want both projects", len(rec.Items()))
	}

	empty, err := readOp(r.gw, domain.OpJiraProjectsList, map[string]any{"query": "zzz-no-match"})
	if err != nil {
		t.Fatalf("empty projects list: %v", err)
	}
	if !empty.IsList() || len(empty.Items()) != 0 {
		t.Fatalf("empty result = %+v, want an empty list", empty)
	}
}

// TestProjectsListRefusesABadLimitBeforeAnyRequest proves a limit outside 1 to 50 is refused without a request.
func TestProjectsListRefusesABadLimitBeforeAnyRequest(t *testing.T) {
	for _, limit := range []int{0, 51} {
		r := newRig(t)
		if _, err := readOp(r.gw, domain.OpJiraProjectsList, map[string]any{"limit": limit}); err == nil {
			t.Fatalf("limit %d was accepted", limit)
		}
		if n := len(r.stub.Requests()); n != 0 {
			t.Fatalf("limit %d sent %d requests, want none", limit, n)
		}
	}
}

// TestSearchSendsTheJQLOnlyInTheBody proves the JQL is one JSON string in the body, with its quotes, its line break, and its
// ampersand kept as they are, and never in the path or the query.
func TestSearchSendsTheJQLOnlyInTheBody(t *testing.T) {
	r := newRig(t)
	jql := "project = \"EXAMPLE\"\nAND summary ~ \"a & b\" ORDER BY key"
	rec, err := readOp(r.gw, domain.OpJiraIssuesSearch, map[string]any{"jql": jql, "limit": 2})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	reqs := r.stub.Requests()
	if len(reqs) != 1 {
		t.Fatalf("requests = %d, want exactly one", len(reqs))
	}
	req := reqs[0]
	if req.Method != http.MethodPost || req.Path != "/rest/api/3/search/jql" || req.Query != "" {
		t.Fatalf("request %s %s?%s, want POST /rest/api/3/search/jql with no query", req.Method, req.Path, req.Query)
	}
	if strings.Contains(req.Path+req.Query, "EXAMPLE") {
		t.Fatal("the JQL reached the path or the query")
	}
	if req.Body["jql"] != jql {
		t.Fatalf("body jql = %q, want the JQL exactly", req.Body["jql"])
	}
	if req.Body["maxResults"] != float64(2) {
		t.Fatalf("maxResults = %v, want 2", req.Body["maxResults"])
	}
	if _, ok := req.Body["expand"]; ok {
		t.Fatal("the body sent expand")
	}
	if _, ok := req.Body["properties"]; ok {
		t.Fatal("the body sent properties")
	}
	wantFields := []any{"summary", "status", "issuetype", "priority", "assignee", "updated", "project", "parent"}
	if !reflect.DeepEqual(req.Body["fields"], wantFields) {
		t.Fatalf("fields = %v, want the fixed list", req.Body["fields"])
	}
	if len(rec.Items()) != 2 {
		t.Fatalf("items = %d, want the first 2 of 4", len(rec.Items()))
	}
}

// TestSearchItemsCarryTheFixedFieldsAndTruncation proves each item holds the row's fields, that a search which holds more
// issues marks every item truncated, and that the page says so too.
func TestSearchItemsCarryTheFixedFieldsAndTruncation(t *testing.T) {
	r := newRig(t)
	rec, err := readOp(r.gw, domain.OpJiraIssuesSearch, map[string]any{"jql": `project = "EXAMPLE"`, "limit": 2})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	first := rec.Items()[0]
	want := []string{"key", "summary", "status", "type", "priority", "assignee", "updated", "projectKey", "truncated"}
	if got := itemNames(first); !reflect.DeepEqual(got, want) {
		t.Fatalf("item fields = %v, want %v", got, want)
	}
	if first.Str("key") != "EXAMPLE-1" || first.Str("summary") != "Epic: login rework" || first.Str("status") != "In Progress" {
		t.Fatalf("first item %+v", first)
	}
	if first.Str("updated") != "2026-11-03T03:00:00Z" || first.Str("projectKey") != "EXAMPLE" || first.Str("assignee") != "Fixture Reviewer" {
		t.Fatalf("updated %q project %q assignee %q", first.Str("updated"), first.Str("projectKey"), first.Str("assignee"))
	}
	if v, _ := first.Get("truncated"); v != true {
		t.Fatalf("truncated = %v, want true: the stub holds more issues after this page", v)
	}
	if page, ok := rec.Page(); !ok || !page.HasMore {
		t.Fatalf("page = %+v, want HasMore set", page)
	}
}

// TestSearchIsNotTruncatedWhenEverythingFits proves a page that holds every match is not marked truncated, and an unassigned
// issue has an empty assignee.
func TestSearchIsNotTruncatedWhenEverythingFits(t *testing.T) {
	r := newRig(t)
	rec, err := readOp(r.gw, domain.OpJiraIssuesSearch, map[string]any{"jql": `project = "EXAMPLE"`, "limit": 100})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(rec.Items()) != 4 {
		t.Fatalf("items = %d, want all 4 EXAMPLE issues", len(rec.Items()))
	}
	for _, item := range rec.Items() {
		if v, _ := item.Get("truncated"); v != false {
			t.Fatalf("item %s truncated = %v, want false", item.Str("key"), v)
		}
	}
	if unassigned := rec.Items()[2]; unassigned.Str("key") != "EXAMPLE-13" || unassigned.Str("assignee") != "" {
		t.Fatalf("unassigned item %+v", unassigned)
	}
	if page, _ := rec.Page(); page.HasMore {
		t.Fatal("a full result set marked the page as having more")
	}
}

// TestSearchTrimsAPageThatHoldsMoreThanAsked proves a reply with more issues than the limit is cut to the limit, and is
// marked truncated, so the caller knows more matched.
func TestSearchTrimsAPageThatHoldsMoreThanAsked(t *testing.T) {
	issue := `{"key":"EXAMPLE-%d","fields":{"summary":"s","issuetype":{"name":"Task"}}}`
	body := `{"issues":[` + strings.Join([]string{
		strings.Replace(issue, "%d", "1", 1), strings.Replace(issue, "%d", "2", 1), strings.Replace(issue, "%d", "3", 1),
	}, ",") + `],"isLast":true}`
	p := &probe{script: func(int, *http.Request) (*http.Response, error) {
		return reply(http.StatusOK, body), nil
	}}
	gw := probeGateway(t, p, newFakeClock(), newSecrets())
	rec, err := readOp(gw, domain.OpJiraIssuesSearch, map[string]any{"jql": `project = "EXAMPLE"`, "limit": 2})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(rec.Items()) != 2 {
		t.Fatalf("items = %d, want the first 2", len(rec.Items()))
	}
	if v, _ := rec.Items()[0].Get("truncated"); v != true {
		t.Fatal("a trimmed page is not marked truncated")
	}
}

// TestSearchTruncationNeedsATokenAndNoLastFlag proves truncation follows the paging rule: a token that is present
// while isLast is not true.
func TestSearchTruncationNeedsATokenAndNoLastFlag(t *testing.T) {
	issue := `{"key":"EXAMPLE-1","fields":{"summary":"s"}}`
	cases := []struct {
		name      string
		body      string
		truncated bool
	}{
		{"token and not last", `{"issues":[` + issue + `],"isLast":false,"nextPageToken":"p1"}`, true},
		{"token but last", `{"issues":[` + issue + `],"isLast":true,"nextPageToken":"p1"}`, false},
		{"no token", `{"issues":[` + issue + `]}`, false},
	}
	for _, c := range cases {
		p := &probe{script: func(int, *http.Request) (*http.Response, error) {
			return reply(http.StatusOK, c.body), nil
		}}
		gw := probeGateway(t, p, newFakeClock(), newSecrets())
		rec, err := readOp(gw, domain.OpJiraIssuesSearch, map[string]any{"jql": `project = "EXAMPLE"`, "limit": 5})
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if v, _ := rec.Items()[0].Get("truncated"); v != c.truncated {
			t.Fatalf("%s: truncated = %v, want %v", c.name, v, c.truncated)
		}
	}
}

// TestSearchRefusesABadJQLWithTheGenericBadRequest proves Atlassian's 400 becomes the generic bad-request sentinel, and the
// text never shows the JQL.
func TestSearchRefusesABadJQLWithTheGenericBadRequest(t *testing.T) {
	r := newRig(t)
	_, err := readOp(r.gw, domain.OpJiraIssuesSearch, map[string]any{"jql": "project = EXAMPLE invalid", "limit": 5})
	if !errors.Is(err, domain.ErrBadRequest) {
		t.Fatalf("err = %v, want ErrBadRequest", err)
	}
	if strings.Contains(err.Error(), "EXAMPLE") {
		t.Fatalf("the error shows the JQL: %q", err.Error())
	}
}

// TestSearchRefusesAnUnboundedJQLFromTheServer proves a JQL the server refuses as unbounded is a bad request, not a result.
func TestSearchRefusesAnUnboundedJQLFromTheServer(t *testing.T) {
	r := newRig(t)
	if _, err := readOp(r.gw, domain.OpJiraIssuesSearch, map[string]any{"jql": "summary ~ \"login\"", "limit": 5}); !errors.Is(err, domain.ErrBadRequest) {
		t.Fatalf("err = %v, want ErrBadRequest", err)
	}
}

// TestSearchNeedsItsJQLBeforeAnyRequest proves a search without a JQL is refused before the network.
func TestSearchNeedsItsJQLBeforeAnyRequest(t *testing.T) {
	r := newRig(t)
	if _, err := readOp(r.gw, domain.OpJiraIssuesSearch, map[string]any{"limit": 5}); err == nil {
		t.Fatal("a search with no JQL was accepted")
	}
	if n := len(r.stub.Requests()); n != 0 {
		t.Fatalf("requests = %d, want none", n)
	}
}

// TestSearchReplyShapeIsChecked proves the strict members: the issues list, the issue key pattern, and the fields object with
// a summary. Each malformed reply is an unexpected reply.
func TestSearchReplyShapeIsChecked(t *testing.T) {
	bodies := map[string]string{
		"no issues member":    `{"isLast":true}`,
		"lower-case key":      `{"issues":[{"key":"example-1","fields":{"summary":"s"}}],"isLast":true}`,
		"no fields object":    `{"issues":[{"key":"EXAMPLE-1"}],"isLast":true}`,
		"no summary":          `{"issues":[{"key":"EXAMPLE-1","fields":{}}],"isLast":true}`,
		"empty page, token":   `{"issues":[],"isLast":false,"nextPageToken":"p9"}`,
		"non-object item":     `{"issues":["EXAMPLE-1"],"isLast":true}`,
		"null body":           `null`,
		"not an object reply": `[]`,
	}
	for name, body := range bodies {
		p := &probe{script: func(int, *http.Request) (*http.Response, error) {
			return reply(http.StatusOK, body), nil
		}}
		gw := probeGateway(t, p, newFakeClock(), newSecrets())
		if _, err := readOp(gw, domain.OpJiraIssuesSearch, map[string]any{"jql": `project = "EXAMPLE"`, "limit": 5}); !errors.Is(err, domain.ErrUnexpectedResponse) {
			t.Fatalf("%s: err = %v, want ErrUnexpectedResponse", name, err)
		}
	}
}

// TestUnknownMembersAreIgnored proves a member Retriever does not read changes nothing: the reply still decodes.
func TestUnknownMembersAreIgnored(t *testing.T) {
	body := `{"futureMember":{"x":1},"issues":[{"id":"1","key":"EXAMPLE-1","newThing":[1,2],"fields":{"summary":"s","futureField":"v"}}],"isLast":true}`
	p := &probe{script: func(int, *http.Request) (*http.Response, error) {
		return reply(http.StatusOK, body), nil
	}}
	gw := probeGateway(t, p, newFakeClock(), newSecrets())
	rec, err := readOp(gw, domain.OpJiraIssuesSearch, map[string]any{"jql": `project = "EXAMPLE"`, "limit": 5})
	if err != nil {
		t.Fatalf("search with unknown members: %v", err)
	}
	if len(rec.Items()) != 1 || rec.Items()[0].Str("key") != "EXAMPLE-1" {
		t.Fatalf("items %+v", rec.Items())
	}
}
