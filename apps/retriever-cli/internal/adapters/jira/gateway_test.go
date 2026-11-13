package jira

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// projectBody returns one page of the project search, holding the given keys.
func projectBody(keys []string, isLast bool) string {
	values := make([]string, 0, len(keys))
	for _, k := range keys {
		values = append(values, fmt.Sprintf(`{"key":%q,"name":"Name %s","projectTypeKey":"software","style":"classic"}`, k, k))
	}
	return fmt.Sprintf(`{"values":[%s],"isLast":%t}`, strings.Join(values, ","), isLast)
}

// projectKeys returns n distinct project keys that pass the pattern.
func projectKeys(prefix string, n int) []string {
	keys := make([]string, 0, n)
	for i := 0; i < n; i++ {
		keys = append(keys, fmt.Sprintf("%s%04d", prefix, i))
	}
	return keys
}

// issueJSON returns one issue of a search page. An empty parent is left out.
func issueJSON(key, parent string) string {
	fields := `"summary":"Summary of ` + key + `"`
	if parent != "" {
		fields += `,"parent":{"key":"` + parent + `"}`
	}
	return `{"id":"1","key":"` + key + `","fields":{` + fields + `}}`
}

// treePage returns one page of issues of a tree search, with a token or the last-page flag.
func treePage(issues []string, token string, isLast bool) string {
	body := `{"issues":[` + strings.Join(issues, ",") + `],"isLast":` + fmt.Sprint(isLast)
	if token != "" {
		body += `,"nextPageToken":"` + token + `"`
	}
	return body + `}`
}

// numberedIssues returns n issues with keys PREFIX-1 to PREFIX-n.
func numberedIssues(prefix string, n int) []string {
	out := make([]string, 0, n)
	for i := 1; i <= n; i++ {
		out = append(out, issueJSON(fmt.Sprintf("%s-%d", prefix, i), ""))
	}
	return out
}

// collectionID and documentID build the typed IDs of the tests, failing the test on a value the domain refuses.
func collectionID(t *testing.T, raw string) domain.CollectionID {
	t.Helper()
	id, err := domain.NewCollectionID(raw)
	if err != nil {
		t.Fatalf("NewCollectionID(%q): %v", raw, err)
	}
	return id
}

func documentID(t *testing.T, raw string) domain.DocumentID {
	t.Helper()
	id, err := domain.NewDocumentID(raw)
	if err != nil {
		t.Fatalf("NewDocumentID(%q): %v", raw, err)
	}
	return id
}

// TestListCollectionsReturnsEveryProjectAsACollection proves each project is a collection with the ID project-<KEY>.
func TestListCollectionsReturnsEveryProjectAsACollection(t *testing.T) {
	r := newRig(t)
	got, err := r.gw.ListCollections(context.Background())
	if err != nil {
		t.Fatalf("ListCollections: %v", err)
	}
	want := []domain.Collection{{ID: "project-EXAMPLE", Name: "Example Org"}, {ID: "project-OPS", Name: "Operations"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("collections = %+v, want %+v", got, want)
	}
	reqs := r.stub.Requests()
	if len(reqs) != 1 || queryOf(t, reqs[0].Query).Get("maxResults") != "50" {
		t.Fatalf("requests %+v, want one page of 50", reqs)
	}
}

// TestListCollectionsPagesUntilTheLastPage proves the list asks each page with startAt and stops at the last one.
func TestListCollectionsPagesUntilTheLastPage(t *testing.T) {
	pages := []string{projectBody(projectKeys("A", 50), false), projectBody(projectKeys("B", 3), true)}
	p := &probe{script: func(n int, _ *http.Request) (*http.Response, error) {
		return reply(http.StatusOK, pages[n-1]), nil
	}}
	gw := probeGateway(t, p, newFakeClock(), newSecrets())
	got, err := gw.ListCollections(context.Background())
	if err != nil {
		t.Fatalf("ListCollections: %v", err)
	}
	if len(got) != 53 {
		t.Fatalf("collections = %d, want 53", len(got))
	}
	reqs := p.requests()
	if len(reqs) != 2 || queryOf(t, reqs[0].query).Get("startAt") != "0" || queryOf(t, reqs[1].query).Get("startAt") != "50" {
		t.Fatalf("requests %+v, want startAt 0 then 50", reqs)
	}
}

// TestListCollectionsStopsAtTheTwentyPageCap proves a listing that never ends is refused at the 21st page, after 20 requests.
func TestListCollectionsStopsAtTheTwentyPageCap(t *testing.T) {
	p := &probe{script: func(int, *http.Request) (*http.Response, error) {
		return reply(http.StatusOK, projectBody(projectKeys("C", 50), false)), nil
	}}
	gw := probeGateway(t, p, newFakeClock(), newSecrets())
	if _, err := gw.ListCollections(context.Background()); !errors.Is(err, domain.ErrPageCapReached) {
		t.Fatalf("err = %v, want ErrPageCapReached", err)
	}
	if p.calls() != 20 {
		t.Fatalf("calls = %d, want 20", p.calls())
	}
}

// TestListCollectionsStopsOnAnEmptyPage proves an empty page ends the list without an error.
func TestListCollectionsStopsOnAnEmptyPage(t *testing.T) {
	p := &probe{script: func(int, *http.Request) (*http.Response, error) {
		return reply(http.StatusOK, projectBody(nil, false)), nil
	}}
	gw := probeGateway(t, p, newFakeClock(), newSecrets())
	got, err := gw.ListCollections(context.Background())
	if err != nil || len(got) != 0 || p.calls() != 1 {
		t.Fatalf("got %v, err %v, calls %d; want an empty list after one call", got, err, p.calls())
	}
}

// TestListCollectionsRefusesAKeyThatFailsThePattern proves a project key the server sends in a shape Retriever does not accept is
// an unexpected reply.
func TestListCollectionsRefusesAKeyThatFailsThePattern(t *testing.T) {
	p := &probe{script: func(int, *http.Request) (*http.Response, error) {
		return reply(http.StatusOK, projectBody([]string{"bad-key"}, true)), nil
	}}
	gw := probeGateway(t, p, newFakeClock(), newSecrets())
	if _, err := gw.ListCollections(context.Background()); !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
	}
}

// TestCollectionTreeListsTheIssuesWithParentsAndDepths proves the tree of a project: its issues in key order, a parent ID only
// when the parent is in the result, and a depth from the parent chain. The JQL is a constant with one checked key.
func TestCollectionTreeListsTheIssuesWithParentsAndDepths(t *testing.T) {
	r := newRig(t)
	nodes, err := r.gw.CollectionTree(context.Background(), collectionID(t, "project-EXAMPLE"))
	if err != nil {
		t.Fatalf("CollectionTree: %v", err)
	}
	if len(nodes) != 4 {
		t.Fatalf("nodes = %d, want the 4 EXAMPLE issues", len(nodes))
	}
	parent := "issue-EXAMPLE-1"
	if nodes[0].ID != "issue-EXAMPLE-1" || nodes[0].ParentID != nil || nodes[0].Depth != 0 {
		t.Fatalf("root node %+v", nodes[0])
	}
	if nodes[1].ID != "issue-EXAMPLE-12" || nodes[1].ParentID == nil || *nodes[1].ParentID != parent || nodes[1].Depth != 1 {
		t.Fatalf("child node %+v, want parent %s at depth 1", nodes[1], parent)
	}
	if nodes[1].URL != "https://"+testHost+"/browse/EXAMPLE-12" || nodes[1].Title != "Login fails on second try" {
		t.Fatalf("child URL %q title %q", nodes[1].URL, nodes[1].Title)
	}
	reqs := r.stub.Requests()
	if len(reqs) != 1 || reqs[0].Path != "/rest/api/3/search/jql" {
		t.Fatalf("requests %+v, want one search", reqs)
	}
	if reqs[0].Body["jql"] != `project = "EXAMPLE" ORDER BY key ASC` || reqs[0].Body["maxResults"] != float64(100) {
		t.Fatalf("body %v, want the constant JQL and 100 per page", reqs[0].Body)
	}
}

// TestCollectionTreeOfAnotherProject proves the tree follows the project the ID names.
func TestCollectionTreeOfAnotherProject(t *testing.T) {
	r := newRig(t)
	nodes, err := r.gw.CollectionTree(context.Background(), collectionID(t, "project-OPS"))
	if err != nil {
		t.Fatalf("CollectionTree: %v", err)
	}
	if len(nodes) != 1 || nodes[0].ID != "issue-OPS-3" {
		t.Fatalf("nodes %+v, want the one OPS issue", nodes)
	}
}

// TestCollectionTreeOfAProjectTheServerRefusesIsNotFound proves a 400 for the constant query means the project does not exist
// or is not visible.
func TestCollectionTreeOfAProjectTheServerRefusesIsNotFound(t *testing.T) {
	p := &probe{script: func(int, *http.Request) (*http.Response, error) {
		return reply(http.StatusBadRequest, `{}`), nil
	}}
	gw := probeGateway(t, p, newFakeClock(), newSecrets())
	if _, err := gw.CollectionTree(context.Background(), collectionID(t, "project-NOPE")); !errors.Is(err, domain.ErrCollectionNotFound) {
		t.Fatalf("err = %v, want ErrCollectionNotFound", err)
	}
}

// TestCollectionTreePagesWithTheToken proves the second page is asked with the token of the first, and the tree joins both.
func TestCollectionTreePagesWithTheToken(t *testing.T) {
	pages := []string{
		treePage(numberedIssues("EXAMPLE", 100), "t1", false),
		treePage([]string{issueJSON("EXAMPLE-101", "")}, "", true),
	}
	p := &probe{script: func(n int, _ *http.Request) (*http.Response, error) {
		return reply(http.StatusOK, pages[n-1]), nil
	}}
	gw := probeGateway(t, p, newFakeClock(), newSecrets())
	nodes, err := gw.CollectionTree(context.Background(), collectionID(t, "project-EXAMPLE"))
	if err != nil {
		t.Fatalf("CollectionTree: %v", err)
	}
	if len(nodes) != 101 {
		t.Fatalf("nodes = %d, want 101", len(nodes))
	}
	second := p.requests()[1].body
	var body map[string]any
	if err := json.Unmarshal([]byte(second), &body); err != nil {
		t.Fatalf("second body: %v", err)
	}
	if body["nextPageToken"] != "t1" || body["maxResults"] != float64(100) {
		t.Fatalf("second body %v, want the token t1 and 100", body)
	}
}

// TestCollectionTreeRefusesARepeatedToken proves a token the server already sent ends the walk with an unexpected reply.
func TestCollectionTreeRefusesARepeatedToken(t *testing.T) {
	p := &probe{script: func(int, *http.Request) (*http.Response, error) {
		return reply(http.StatusOK, treePage(numberedIssues("EXAMPLE", 2), "t1", false)), nil
	}}
	gw := probeGateway(t, p, newFakeClock(), newSecrets())
	if _, err := gw.CollectionTree(context.Background(), collectionID(t, "project-EXAMPLE")); !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
	}
	if p.calls() != 2 {
		t.Fatalf("calls = %d, want 2 (the repeat is seen on the second page)", p.calls())
	}
}

// TestCollectionTreeRefusesAnEmptyPageThatCarriesAToken proves an empty page that still has a token is an unexpected reply.
func TestCollectionTreeRefusesAnEmptyPageThatCarriesAToken(t *testing.T) {
	p := &probe{script: func(int, *http.Request) (*http.Response, error) {
		return reply(http.StatusOK, treePage(nil, "t9", false)), nil
	}}
	gw := probeGateway(t, p, newFakeClock(), newSecrets())
	if _, err := gw.CollectionTree(context.Background(), collectionID(t, "project-EXAMPLE")); !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
	}
	if p.calls() != 1 {
		t.Fatalf("calls = %d, want 1", p.calls())
	}
}

// TestCollectionTreeStopsAtFiveRequests proves a walk that never ends is refused after five requests, which hold 500 issues.
func TestCollectionTreeStopsAtFiveRequests(t *testing.T) {
	p := &probe{script: func(n int, _ *http.Request) (*http.Response, error) {
		return reply(http.StatusOK, treePage(numberedIssues("EXAMPLE", 100), fmt.Sprintf("t%d", n), false)), nil
	}}
	gw := probeGateway(t, p, newFakeClock(), newSecrets())
	if _, err := gw.CollectionTree(context.Background(), collectionID(t, "project-EXAMPLE")); !errors.Is(err, domain.ErrPageCapReached) {
		t.Fatalf("err = %v, want ErrPageCapReached", err)
	}
	if p.calls() != 5 {
		t.Fatalf("calls = %d, want 5", p.calls())
	}
}

// TestCollectionTreeRefusesACycleOfParents proves two issues that name each other as parent cannot make the depth loop.
func TestCollectionTreeRefusesACycleOfParents(t *testing.T) {
	issues := []string{issueJSON("EXAMPLE-1", "EXAMPLE-2"), issueJSON("EXAMPLE-2", "EXAMPLE-1")}
	p := &probe{script: func(int, *http.Request) (*http.Response, error) {
		return reply(http.StatusOK, treePage(issues, "", true)), nil
	}}
	gw := probeGateway(t, p, newFakeClock(), newSecrets())
	if _, err := gw.CollectionTree(context.Background(), collectionID(t, "project-EXAMPLE")); !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
	}
}

// TestCollectionTreeNeedsAProjectID proves an ID that is not project-<KEY> is refused before any request.
func TestCollectionTreeNeedsAProjectID(t *testing.T) {
	for _, raw := range []string{"project-example", "issue-EXAMPLE-1", "EXAMPLE"} {
		r := newRig(t)
		if _, err := r.gw.CollectionTree(context.Background(), collectionID(t, raw)); !errors.Is(err, domain.ErrInvalidCollectionID) {
			t.Fatalf("%s: err = %v, want ErrInvalidCollectionID", raw, err)
		}
		if n := len(r.stub.Requests()); n != 0 {
			t.Fatalf("%s sent %d requests", raw, n)
		}
	}
}

// TestDocumentReadsOneIssue proves a document is the issue, with its ID, title, URL, Markdown, and collection.
func TestDocumentReadsOneIssue(t *testing.T) {
	r := newRig(t)
	doc, err := r.gw.Document(context.Background(), documentID(t, "issue-EXAMPLE-12"))
	if err != nil {
		t.Fatalf("Document: %v", err)
	}
	if doc.ID != "issue-EXAMPLE-12" || doc.Title != "Login fails on second try" || doc.CollectionID != "project-EXAMPLE" {
		t.Fatalf("document %+v", doc)
	}
	if doc.URL != "https://"+testHost+"/browse/EXAMPLE-12" || !strings.HasPrefix(doc.Markdown, "# Login fails on second try\n\n") {
		t.Fatalf("URL %q, markdown starts %q", doc.URL, doc.Markdown[:min(40, len(doc.Markdown))])
	}
}

// TestDocumentRefusesAnIDThatIsNotAnIssue proves a project ID, a lower-case key, or a bare key is refused before any request.
func TestDocumentRefusesAnIDThatIsNotAnIssue(t *testing.T) {
	for _, raw := range []string{"project-EXAMPLE", "issue-example-12", "EXAMPLE-12", "issue-EXAMPLE-12x", "issue-EXAMPLE-0"} {
		r := newRig(t)
		if _, err := r.gw.Document(context.Background(), documentID(t, raw)); !errors.Is(err, domain.ErrInvalidDocumentID) {
			t.Fatalf("%s: err = %v, want ErrInvalidDocumentID", raw, err)
		}
		if n := len(r.stub.Requests()); n != 0 {
			t.Fatalf("%s sent %d requests", raw, n)
		}
	}
}

// TestDocumentOfAMissingIssueIsNotFound proves an issue the server does not hold is the document sentinel.
func TestDocumentOfAMissingIssueIsNotFound(t *testing.T) {
	r := newRig(t)
	if _, err := r.gw.Document(context.Background(), documentID(t, "issue-EXAMPLE-999")); !errors.Is(err, domain.ErrDocumentNotFound) {
		t.Fatalf("err = %v, want ErrDocumentNotFound", err)
	}
}

// TestEveryRequestTheGatewayMakesIsInTheTable runs every gateway method and every read, then checks each request the stub
// received against the closed table of ADR 010 section 5.
func TestEveryRequestTheGatewayMakesIsInTheTable(t *testing.T) {
	r := newRig(t)
	ctx := context.Background()
	if _, err := r.gw.ListCollections(ctx); err != nil {
		t.Fatalf("ListCollections: %v", err)
	}
	if _, err := r.gw.CollectionTree(ctx, collectionID(t, "project-EXAMPLE")); err != nil {
		t.Fatalf("CollectionTree: %v", err)
	}
	if _, err := r.gw.Document(ctx, documentID(t, "issue-EXAMPLE-12")); err != nil {
		t.Fatalf("Document: %v", err)
	}
	reads := []struct {
		op   domain.OperationID
		args map[string]any
	}{
		{domain.OpJiraWhoami, nil},
		{domain.OpJiraProjectsList, nil},
		{domain.OpJiraIssuesSearch, map[string]any{"jql": `project = "EXAMPLE"`}},
		{domain.OpJiraIssuesGet, map[string]any{"issueKey": "EXAMPLE-12"}},
		{domain.OpJiraCommentsList, map[string]any{"issueKey": "EXAMPLE-12"}},
	}
	for _, read := range reads {
		if _, err := readOp(r.gw, read.op, read.args); err != nil {
			t.Fatalf("%s: %v", read.op, err)
		}
	}
	reqs := r.stub.Requests()
	if len(reqs) < 8 {
		t.Fatalf("only %d requests recorded", len(reqs))
	}
	for _, req := range reqs {
		if !inTable(req.Method, req.Path) {
			t.Fatalf("the gateway sent %s %s, which is not in the table", req.Method, req.Path)
		}
	}
}

// inTable reports whether a method and a concrete path are one row of the table, with the key placeholder matched by any
// issue key.
func inTable(method, path string) bool {
	for _, ep := range allowed {
		if ep.method != method {
			continue
		}
		if ep.path == path {
			return true
		}
		if strings.Contains(ep.path, keyPlaceholder) {
			prefix, suffix, _ := strings.Cut(ep.path, keyPlaceholder)
			rest, ok := strings.CutPrefix(path, prefix)
			if ok && strings.HasSuffix(rest, suffix) && !strings.Contains(strings.TrimSuffix(rest, suffix), "/") {
				return true
			}
		}
	}
	return false
}

// TestTheGatewayErrorsNameJira proves the boundary errors of the generic methods read as Jira.
func TestTheGatewayErrorsNameJira(t *testing.T) {
	p := &probe{script: func(int, *http.Request) (*http.Response, error) {
		return reply(http.StatusOK, projectBody(projectKeys("D", 50), false)), nil
	}}
	gw := probeGateway(t, p, newFakeClock(), newSecrets())
	_, err := gw.ListCollections(context.Background())
	if err == nil || err.Error() != "Jira listing reached the page cap of 20 pages" {
		t.Fatalf("text = %v, want the Jira page-cap text", err)
	}
	if strings.Contains(err.Error(), "Outline") {
		t.Fatalf("the Jira error says Outline: %v", err)
	}
}
