package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/connectorspec"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/envfile"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// cannedJiraReply is one scripted answer of the canned Jira transport.
type cannedJiraReply struct {
	status int
	body   string
}

// cannedJiraCall is one request the canned Jira transport received.
type cannedJiraCall struct {
	method string
	path   string
	query  string
	body   string
	auth   string
}

// cannedJiraTransport answers the Jira paths a gateway test names, in the order the test scripts them. A path
// with no scripted reply left gets a 404 without a router message, so the engine reports a plain not found.
type cannedJiraTransport struct {
	mu      sync.Mutex
	scripts map[string][]cannedJiraReply
	calls   []cannedJiraCall
}

// RoundTrip serves the next scripted reply of the request path and records the request.
func (c *cannedJiraTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	var body []byte
	if r.Body != nil {
		body, _ = io.ReadAll(r.Body)
	}
	call := cannedJiraCall{
		method: r.Method,
		path:   r.URL.EscapedPath(),
		query:  r.URL.RawQuery,
		body:   string(body),
		auth:   r.Header.Get("Authorization"),
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls = append(c.calls, call)
	key := r.Method + " " + r.URL.EscapedPath()
	script := c.scripts[key]
	rep := cannedJiraReply{status: 404, body: `{"unexpected":true}`}
	if len(script) > 0 {
		rep = script[0]
		c.scripts[key] = script[1:]
	}
	return &http.Response{
		StatusCode: rep.status,
		Header:     http.Header{},
		Body:       io.NopCloser(bytes.NewReader([]byte(rep.body))),
	}, nil
}

// callsTo returns the recorded calls of one method and path.
func (c *cannedJiraTransport) callsTo(method, path string) []cannedJiraCall {
	c.mu.Lock()
	defer c.mu.Unlock()
	var out []cannedJiraCall
	for _, call := range c.calls {
		if call.method == method && call.path == path {
			out = append(out, call)
		}
	}
	return out
}

// jiraFixture builds the Jira gateway of a test: the bundled jira.yml model over the given transport. The
// credentials resolve the key and the e-mail address to fixed values, so no environment is read.
func jiraFixture(t *testing.T, transport http.RoundTripper) *jiraGateway {
	t.Helper()
	set, err := connectorspec.LoadBundled()
	if err != nil {
		t.Fatalf("LoadBundled: %v", err)
	}
	model := bundledJira(set)
	if model == nil {
		t.Fatal("the bundled set holds no jira spec")
	}
	src := seamSourceOf(t, "jira", "https://jira.example.test", "EXAMPLE_JIRA_API_TOKEN")
	identity, err := domain.NewIdentityRef("EXAMPLE_JIRA_EMAIL")
	if err != nil {
		t.Fatalf("NewIdentityRef: %v", err)
	}
	src = src.WithIdentity(identity)
	creds := jiraGatewayCredentials(t)
	eseam := engineSeam{clock: fakeClock{}, transport: transport}
	gw, err := buildJiraGateway(model, src, creds, seamConfig{}, eseam)
	if err != nil {
		t.Fatalf("buildJiraGateway: %v", err)
	}
	jira, ok := gw.(*jiraGateway)
	if !ok {
		t.Fatalf("built %T, want the Jira gateway", gw)
	}
	if _, ok := any(gw).(ports.SourceGateway); !ok {
		t.Fatal("the Jira gateway is not a source gateway")
	}
	if _, ok := any(gw).(ports.SourceReader); !ok {
		t.Fatal("the Jira gateway is not a source reader")
	}
	return jira
}

// jiraGatewayCredentials returns credentials whose Jira key and e-mail address are set, so a gateway built in a
// test reads both without an environment.
func jiraGatewayCredentials(t *testing.T) *envfile.Credentials {
	t.Helper()
	creds, err := envfile.NewWithFiles("/work/project/keys.env", func(name string) string {
		switch name {
		case "EXAMPLE_JIRA_API_TOKEN":
			return "synthetic-jira-token"
		case "EXAMPLE_JIRA_EMAIL":
			return "synthetic-user@example.test"
		}
		return ""
	}, envfile.Files{FS: fstest.MapFS{}, SameFile: sameName})
	if err != nil {
		t.Fatalf("envfile.NewWithFiles: %v", err)
	}
	return creds
}

// TestJiraGatewayServesWhoamiOnTheEngine proves the read routing: the whoami read runs on the engine, and the
// canned reply maps to its three fields. The request carries no query, and it authenticates with HTTP Basic, as
// the Go gateway does.
func TestJiraGatewayServesWhoamiOnTheEngine(t *testing.T) {
	transport := &cannedJiraTransport{scripts: map[string][]cannedJiraReply{
		"GET /rest/api/3/myself": {{status: 200, body: `{"displayName":"Ada","accountType":"atlassian","active":true}`}},
	}}
	gw := jiraFixture(t, transport)
	rec, err := gw.Read(context.Background(), domain.OpJiraWhoami, domain.NewArgs(nil))
	if err != nil {
		t.Fatalf("Read = %v, want nil", err)
	}
	if got := rec.Str("displayName"); got != "Ada" {
		t.Fatalf("displayName = %q, want Ada", got)
	}
	if got := rec.Str("accountType"); got != "atlassian" {
		t.Fatalf("accountType = %q, want atlassian", got)
	}
	if got, ok := rec.Get("active"); !ok || got != true {
		t.Fatalf("active = %v, want true", got)
	}
	calls := transport.callsTo("GET", "/rest/api/3/myself")
	if len(calls) != 1 {
		t.Fatalf("requests to myself = %d, want 1", len(calls))
	}
	if calls[0].query != "" {
		t.Fatalf("query = %q, want none", calls[0].query)
	}
	if !strings.HasPrefix(calls[0].auth, "Basic ") {
		t.Fatalf("authorization = %q, want HTTP Basic", calls[0].auth)
	}
}

// TestJiraGatewayServesProjectsListOnTheEngine proves the project mapping: one page of the project search maps
// to the four project fields of the catalog row.
func TestJiraGatewayServesProjectsListOnTheEngine(t *testing.T) {
	transport := &cannedJiraTransport{scripts: map[string][]cannedJiraReply{
		"GET /rest/api/3/project/search": {{status: 200, body: `{"startAt":0,"maxResults":25,"total":1,"isLast":true,` +
			`"values":[{"key":"EXAMPLE","name":"Synthetic project","projectTypeKey":"software","style":"classic"}]}`}},
	}}
	gw := jiraFixture(t, transport)
	rec, err := gw.Read(context.Background(), domain.OpJiraProjectsList,
		domain.NewArgs(map[string]any{"limit": 25}))
	if err != nil {
		t.Fatalf("Read = %v, want nil", err)
	}
	if !rec.IsList() || len(rec.Items()) != 1 {
		t.Fatalf("items = %v, want one project", rec.Items())
	}
	item := rec.Items()[0]
	for field, want := range map[string]string{
		"key": "EXAMPLE", "name": "Synthetic project", "type": "software", "style": "classic",
	} {
		if got := item.Str(field); got != want {
			t.Fatalf("%s = %q, want %q", field, got, want)
		}
	}
}

// TestJiraGatewayServesIssuesSearchOnTheEngine proves the search routing: the JQL travels as one JSON body field
// with the page size and the fixed field list and no page token, each item carries the truncated flag of the
// reply, and the gateway sets the search page from it, as the Go gateway does.
func TestJiraGatewayServesIssuesSearchOnTheEngine(t *testing.T) {
	transport := &cannedJiraTransport{scripts: map[string][]cannedJiraReply{
		"POST /rest/api/3/search/jql": {{status: 200, body: `{"issues":[{"id":"10010","key":"EXAMPLE-1",` +
			`"fields":{"summary":"First issue","status":{"name":"To Do"},"issuetype":{"name":"Task"},` +
			`"priority":{"name":"Medium"},"assignee":{"displayName":"Ann"},` +
			`"updated":"2026-11-02T09:15:00.000+0700","project":{"key":"EXAMPLE"}}}],` +
			`"isLast":false,"nextPageToken":"p1"}`}},
	}}
	gw := jiraFixture(t, transport)
	rec, err := gw.Read(context.Background(), domain.OpJiraIssuesSearch,
		domain.NewArgs(map[string]any{"jql": "project=EXAMPLE", "limit": 5}))
	if err != nil {
		t.Fatalf("Read = %v, want nil", err)
	}
	calls := transport.callsTo("POST", "/rest/api/3/search/jql")
	if len(calls) != 1 {
		t.Fatalf("requests to search/jql = %d, want 1", len(calls))
	}
	for _, want := range []string{`"jql":"project=EXAMPLE"`, `"maxResults":5`, `"summary"`, `"nextPageToken"`} {
		if want == `"nextPageToken"` {
			if strings.Contains(calls[0].body, want) {
				t.Fatalf("body = %s, want no page token on the first page", calls[0].body)
			}
			continue
		}
		if !strings.Contains(calls[0].body, want) {
			t.Fatalf("body = %s, want it to hold %s", calls[0].body, want)
		}
	}
	if !rec.IsList() || len(rec.Items()) != 1 {
		t.Fatalf("items = %v, want one issue", rec.Items())
	}
	item := rec.Items()[0]
	for field, want := range map[string]string{
		"key": "EXAMPLE-1", "summary": "First issue", "status": "To Do", "type": "Task",
		"priority": "Medium", "assignee": "Ann", "updated": "2026-11-02T02:15:00Z", "projectKey": "EXAMPLE",
	} {
		if got := item.Str(field); got != want {
			t.Fatalf("%s = %q, want %q", field, got, want)
		}
	}
	if got, ok := item.Get("truncated"); !ok || got != true {
		t.Fatalf("truncated = %v, want true", got)
	}
	page, ok := rec.Page()
	if !ok || !page.HasMore || page.NextOffset != 1 {
		t.Fatalf("page = %+v, want a continuation after one issue", page)
	}
}

// TestJiraGatewayServesIssuesGetOnTheEngine proves the document routing: one issue maps to the five document
// fields of the catalog row, and the Markdown is byte-identical to the Go gateway: the title, the
// metadata lines, and the description under its fixed heading.
func TestJiraGatewayServesIssuesGetOnTheEngine(t *testing.T) {
	transport := &cannedJiraTransport{scripts: map[string][]cannedJiraReply{
		"GET /rest/api/3/issue/EXAMPLE-12": {{status: 200, body: `{"id":"10012","key":"EXAMPLE-12",` +
			`"fields":{"summary":"Login fails on second try",` +
			`"description":{"type":"doc","version":1,"content":[{"type":"paragraph",` +
			`"content":[{"type":"text","text":"Press the button twice."}]}]},` +
			`"issuetype":{"name":"Bug"},"status":{"name":"To Do"},"priority":{"name":"High"},` +
			`"assignee":{"displayName":"Ann"},"reporter":{"displayName":"Bob"},"labels":["auth"],` +
			`"created":"2026-10-01T08:00:00.000+0700","updated":"2026-11-02T09:15:00.000+0700",` +
			`"project":{"key":"EXAMPLE","name":"Example"}}}`}},
	}}
	gw := jiraFixture(t, transport)
	rec, err := gw.Read(context.Background(), domain.OpJiraIssuesGet,
		domain.NewArgs(map[string]any{"issueKey": "EXAMPLE-12"}))
	if err != nil {
		t.Fatalf("Read = %v, want nil", err)
	}
	for field, want := range map[string]string{
		"id":           "issue-EXAMPLE-12",
		"title":        "Login fails on second try",
		"url":          "https://jira.example.test/browse/EXAMPLE-12",
		"collectionId": "project-EXAMPLE",
	} {
		if got := rec.Str(field); got != want {
			t.Fatalf("%s = %q, want %q", field, got, want)
		}
	}
	want := "# Login fails on second try\n" +
		"\n" +
		"- Key: EXAMPLE-12\n" +
		"- Project: Example (EXAMPLE)\n" +
		"- Type: Bug\n" +
		"- Status: To Do\n" +
		"- Priority: High\n" +
		"- Assignee: Ann\n" +
		"- Reporter: Bob\n" +
		"- Labels: auth\n" +
		"- Created: 2026-10-01T01:00:00Z\n" +
		"- Updated: 2026-11-02T02:15:00Z\n" +
		"\n" +
		"## Description\n" +
		"\n" +
		"Press the button twice."
	if got := rec.Str("markdown"); got != want {
		t.Fatalf("markdown = %q, want %q", got, want)
	}
}

// TestJiraGatewayServesCommentsListOnTheEngine proves the comment mapping: one page of comments maps to the five
// comment fields of the catalog row, with each body rendered to Markdown.
func TestJiraGatewayServesCommentsListOnTheEngine(t *testing.T) {
	transport := &cannedJiraTransport{scripts: map[string][]cannedJiraReply{
		"GET /rest/api/3/issue/EXAMPLE-1/comment": {{status: 200, body: `{"startAt":0,"maxResults":50,"total":1,` +
			`"comments":[{"id":"20001","author":{"displayName":"Ann"},"created":"2026-11-03T10:00:00.000+0700",` +
			`"updated":"2026-11-03T10:00:00.000+0700",` +
			`"body":{"type":"doc","version":1,"content":[{"type":"paragraph",` +
			`"content":[{"type":"text","text":"First comment."}]}]}}]}`}},
	}}
	gw := jiraFixture(t, transport)
	rec, err := gw.Read(context.Background(), domain.OpJiraCommentsList,
		domain.NewArgs(map[string]any{"issueKey": "EXAMPLE-1", "limit": 50}))
	if err != nil {
		t.Fatalf("Read = %v, want nil", err)
	}
	if !rec.IsList() || len(rec.Items()) != 1 {
		t.Fatalf("items = %v, want one comment", rec.Items())
	}
	item := rec.Items()[0]
	for field, want := range map[string]string{
		"id": "20001", "author": "Ann", "created": "2026-11-03T03:00:00Z",
		"updated": "2026-11-03T03:00:00Z", "markdown": "First comment.",
	} {
		if got := item.Str(field); got != want {
			t.Fatalf("%s = %q, want %q", field, got, want)
		}
	}
}

// TestJiraGatewayRestoresTheDocumentNotFoundSentinel proves the 404 translation: the engine reports every 404 as
// a plain not found, and the gateway restores the document sentinel of the reads whose path holds the issue key,
// as the Go gateway maps them.
func TestJiraGatewayRestoresTheDocumentNotFoundSentinel(t *testing.T) {
	for _, op := range []domain.OperationID{domain.OpJiraIssuesGet, domain.OpJiraCommentsList} {
		transport := &cannedJiraTransport{}
		gw := jiraFixture(t, transport)
		_, err := gw.Read(context.Background(), op, domain.NewArgs(map[string]any{"issueKey": "EXAMPLE-999"}))
		if !errors.Is(err, domain.ErrDocumentNotFound) {
			t.Fatalf("Read(%s) = %v, want the document-not-found sentinel", op, err)
		}
	}
	for _, op := range []domain.OperationID{
		domain.OpJiraWhoami, domain.OpJiraProjectsList, domain.OpJiraIssuesSearch,
	} {
		transport := &cannedJiraTransport{}
		gw := jiraFixture(t, transport)
		var args domain.Args
		if op == domain.OpJiraIssuesSearch {
			args = domain.NewArgs(map[string]any{"jql": "project=EXAMPLE"})
		} else {
			args = domain.NewArgs(nil)
		}
		_, err := gw.Read(context.Background(), op, args)
		if !errors.Is(err, domain.ErrNotFound) || errors.Is(err, domain.ErrDocumentNotFound) {
			t.Fatalf("Read(%s) = %v, want a plain not found", op, err)
		}
	}
}

// TestJiraGatewayRestoresBareUnreachable proves the unreachable translation: an engine failure that wraps the
// unreachable sentinel comes back bare, as the Go gateway mapped it, so the presenter never shows the dial text.
func TestJiraGatewayRestoresBareUnreachable(t *testing.T) {
	transport := roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("memnet: connection refused at 127.0.0.1:40000, no listener holds it")
	})
	gw := jiraFixture(t, transport)
	_, err := gw.Read(context.Background(), domain.OpJiraWhoami, domain.NewArgs(nil))
	if !errors.Is(err, domain.ErrUnreachable) {
		t.Fatalf("Read = %v, want unreachable", err)
	}
	if err != domain.ErrUnreachable {
		t.Fatalf("Read = %v, want the bare sentinel without the dial text", err)
	}
}

// roundTripperFunc is a function that round-trips one request.
type roundTripperFunc func(*http.Request) (*http.Response, error)

// RoundTrip sends one request through the function.
func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// TestJiraGatewayHidesNothingItDoesNotMap proves the error passthrough: a failure the gateway does not translate
// reaches the caller unchanged, with no sentinel restored.
func TestJiraGatewayHidesNothingItDoesNotMap(t *testing.T) {
	transport := &cannedJiraTransport{}
	gw := jiraFixture(t, transport)
	if _, err := gw.Read(context.Background(), domain.OpJiraWhoami, domain.NewArgs(nil)); !errors.Is(err,
		domain.ErrNotFound) || errors.Is(err, domain.ErrDocumentNotFound) {
		t.Fatalf("Read(whoami) = %v, want a plain not found", err)
	}
	if err := translateJiraNotFound(domain.OpJiraWhoami, nil); err != nil {
		t.Fatalf("translateJiraNotFound(nil) = %v, want nil", err)
	}
	if err := restoreJiraError(domain.OpJiraWhoami, errBoom); !errors.Is(err, errBoom) {
		t.Fatalf("restoreJiraError(boom) = %v, want the failure unchanged", err)
	}
}

// TestJiraGatewayHardensASeamWithoutTLSSettings proves the transport hardening: a seam transport with no TLS
// settings gains the endpoint host as its server name, and a transport that is not an HTTP transport passes
// through untouched.
func TestJiraGatewayHardensASeamWithoutTLSSettings(t *testing.T) {
	bare := &http.Transport{}
	hardened, ok := hardenSeamTransport(bare, "jira.example.test").(*http.Transport)
	if !ok {
		t.Fatal("hardenSeamTransport did not return the HTTP transport")
	}
	if hardened == bare {
		t.Fatal("hardenSeamTransport returned the caller's transport instead of a copy")
	}
	if hardened.TLSClientConfig == nil || hardened.TLSClientConfig.ServerName != "jira.example.test" {
		t.Fatalf("server name = %+v, want jira.example.test", hardened.TLSClientConfig)
	}
	if bare.TLSClientConfig != nil && bare.TLSClientConfig.ServerName != "" {
		t.Fatalf("server name of the caller = %q, want it unchanged", bare.TLSClientConfig.ServerName)
	}
	custom := roundTripperFunc(func(*http.Request) (*http.Response, error) { return nil, errBoom })
	kept, ok := hardenSeamTransport(custom, "jira.example.test").(roundTripperFunc)
	if !ok {
		t.Fatal("hardenSeamTransport changed the type of a transport with no TLS settings of its own")
	}
	if _, err := kept(nil); !errors.Is(err, errBoom) {
		t.Fatalf("hardened custom transport = %v, want the failure unchanged", err)
	}
	configured := &http.Transport{TLSClientConfig: &tls.Config{ServerName: "stub.example.test"}}
	renamed, ok := hardenSeamTransport(configured, "jira.example.test").(*http.Transport)
	if !ok {
		t.Fatal("hardenSeamTransport did not return the HTTP transport")
	}
	if renamed.TLSClientConfig.ServerName != "jira.example.test" {
		t.Fatalf("server name = %q, want jira.example.test", renamed.TLSClientConfig.ServerName)
	}
	if configured.TLSClientConfig.ServerName != "stub.example.test" {
		t.Fatalf("server name of the caller = %q, want it unchanged", configured.TLSClientConfig.ServerName)
	}
}

// errBoom is the opaque failure of a test double, which no sentinel matches.
var errBoom = errors.New("boom")

// TestJiraGatewayRefusesAnOperationItDoesNotServe proves the operation boundary: a catalog operation of another
// kind never reaches the engine.
func TestJiraGatewayRefusesAnOperationItDoesNotServe(t *testing.T) {
	transport := &cannedJiraTransport{}
	gw := jiraFixture(t, transport)
	if _, err := gw.Read(context.Background(), "outline.collections.list",
		domain.NewArgs(nil)); !errors.Is(err, domain.ErrUnsupportedOperation) {
		t.Fatalf("Read(outline.collections.list) = %v, want unsupported", err)
	}
}

// TestBuildJiraGatewayBuildsTheEngine proves the builder: a source whose engine cannot be built stops the source,
// a seam base that is not HTTPS is refused, and a seam timeout bounds one engine request.
func TestBuildJiraGatewayBuildsTheEngine(t *testing.T) {
	set, err := connectorspec.LoadBundled()
	if err != nil {
		t.Fatalf("LoadBundled: %v", err)
	}
	model := bundledJira(set)
	if model == nil {
		t.Fatal("the bundled set holds no jira spec")
	}
	src := seamSourceOf(t, "jira", "https://jira.example.test", "EXAMPLE_JIRA_API_TOKEN")
	creds := jiraGatewayCredentials(t)
	eseam := engineSeam{clock: fakeClock{}, transport: &cannedJiraTransport{}}
	if _, err := buildJiraGateway(nil, src, creds, seamConfig{}, eseam); err == nil {
		t.Fatal("buildJiraGateway without a model succeeds")
	}
	seam := seamConfig{base: "https://127.0.0.1:1", client: fakeClient(), on: true}
	if _, err := buildJiraGateway(model, src, creds, seam, eseam); err != nil {
		t.Fatalf("buildJiraGateway with a seam = %v, want nil", err)
	}
	seam.base = "://no-scheme"
	if _, err := buildJiraGateway(model, src, creds, seam, eseam); err == nil {
		t.Fatal("buildJiraGateway with a base it cannot parse succeeds")
	}
	seam.base = "http://127.0.0.1:1"
	if _, err := buildJiraGateway(model, src, creds, seam, eseam); err == nil {
		t.Fatal("buildJiraGateway with a plain HTTP base succeeds")
	}
	timed := seamConfig{base: "https://127.0.0.1:1", client: fakeClient(), on: true, timeout: 50 * time.Millisecond}
	gw, err := buildJiraGateway(model, src, creds, timed, eseam)
	if err != nil {
		t.Fatalf("buildJiraGateway with a timeout = %v, want nil", err)
	}
	timedGateway, ok := gw.(*jiraGateway)
	if !ok {
		t.Fatalf("built %T, want the Jira gateway", gw)
	}
	if _, err := timedGateway.Read(context.Background(), domain.OpJiraWhoami, domain.NewArgs(nil)); err == nil {
		t.Fatal("Read against the refusing transport succeeds")
	}
}

// TestJiraGatewayFollowsTheSabotagedSpecModel proves the engine engagement: with the title pointer of the bundled
// model retargeted at the status name, the same canned reply renders the status as the document title. A gateway
// that still ran Go code would answer the summary instead.
func TestJiraGatewayFollowsTheSabotagedSpecModel(t *testing.T) {
	set, err := connectorspec.LoadBundled()
	if err != nil {
		t.Fatalf("LoadBundled: %v", err)
	}
	model := bundledJira(set)
	if model == nil {
		t.Fatal("the bundled set holds no jira spec")
	}
	sabotaged := *model
	operations := append([]spec.Operation(nil), model.Operations...)
	sabotaged.Operations = operations
	for i := range operations {
		if operations[i].ID == "jira.issues.read" {
			fields := make(map[string]spec.Field, len(operations[i].Response.Fields))
			for name, field := range operations[i].Response.Fields {
				fields[name] = field
			}
			title := fields["title"]
			title.Pointer = "/fields/status/name"
			fields["title"] = title
			operations[i].Response.Fields = fields
		}
	}
	transport := &cannedJiraTransport{scripts: map[string][]cannedJiraReply{
		"GET /rest/api/3/issue/EXAMPLE-12": {{status: 200, body: `{"id":"10012","key":"EXAMPLE-12",` +
			`"fields":{"summary":"Login fails","status":{"name":"To Do"}}}`}},
	}}
	src := seamSourceOf(t, "jira", "https://jira.example.test", "EXAMPLE_JIRA_API_TOKEN")
	identity, err := domain.NewIdentityRef("EXAMPLE_JIRA_EMAIL")
	if err != nil {
		t.Fatalf("NewIdentityRef: %v", err)
	}
	src = src.WithIdentity(identity)
	eng, err := buildJiraEngine(&sabotaged, src, jiraGatewayCredentials(t), seamConfig{},
		engineSeam{clock: fakeClock{}, transport: transport})
	if err != nil {
		t.Fatalf("buildJiraEngine: %v", err)
	}
	rec, err := eng.Read(context.Background(), "jira.issues.read",
		domain.NewArgs(map[string]any{"issueKey": "EXAMPLE-12"}))
	if err != nil {
		t.Fatalf("Read = %v, want nil", err)
	}
	if got := rec.Str("title"); got != "To Do" {
		t.Fatalf("title of the sabotaged model = %q, want the status name", got)
	}
}

// TestJiraGatewayCollectionTreeWalksEveryPage proves the tree walk: the gateway reads the JQL of the project key
// through the cursor walk, and maps the issues of every page to flat nodes with the parent of each child and the
// depth from 0. A server that refuses the JQL keeps the collection sentinel.
func TestJiraGatewayCollectionTreeWalksEveryPage(t *testing.T) {
	transport := &cannedJiraTransport{scripts: map[string][]cannedJiraReply{
		"POST /rest/api/3/search/jql": {
			{status: 200, body: `{"issues":[` +
				`{"id":"10010","key":"EXAMPLE-1","fields":{"summary":"First issue"}},` +
				`{"id":"10011","key":"EXAMPLE-2","fields":{"summary":"Second issue","parent":{"key":"EXAMPLE-1"}}}` +
				`],"isLast":false,"nextPageToken":"p2"}`},
			{status: 200, body: `{"issues":[` +
				`{"id":"10012","key":"EXAMPLE-3","fields":{"summary":"Third issue","parent":{"key":"EXAMPLE-2"}}}` +
				`],"isLast":true}`},
		},
	}}
	gw := jiraFixture(t, transport)
	collection, err := domain.NewCollectionID("project-EXAMPLE")
	if err != nil {
		t.Fatalf("NewCollectionID: %v", err)
	}
	nodes, err := gw.CollectionTree(context.Background(), collection)
	if err != nil {
		t.Fatalf("CollectionTree = %v, want nil", err)
	}
	if len(nodes) != 3 {
		t.Fatalf("nodes = %v, want three issues", nodes)
	}
	if nodes[0].ID != "issue-EXAMPLE-1" || nodes[0].ParentID != nil || nodes[0].Depth != 0 {
		t.Fatalf("first node = %+v, want the root at depth 0", nodes[0])
	}
	if nodes[1].ParentID == nil || *nodes[1].ParentID != "issue-EXAMPLE-1" || nodes[1].Depth != 1 {
		t.Fatalf("second node = %+v, want the child at depth 1", nodes[1])
	}
	if nodes[2].ParentID == nil || *nodes[2].ParentID != "issue-EXAMPLE-2" || nodes[2].Depth != 2 {
		t.Fatalf("third node = %+v, want the grandchild at depth 2", nodes[2])
	}
	calls := transport.callsTo("POST", "/rest/api/3/search/jql")
	if len(calls) != 2 {
		t.Fatalf("requests to search/jql = %d, want one per page", len(calls))
	}
	if !strings.Contains(calls[0].body, `"project = \"EXAMPLE\" ORDER BY key ASC"`) {
		t.Fatalf("first body = %s, want the JQL of the project key", calls[0].body)
	}
	if strings.Contains(calls[0].body, `"nextPageToken"`) {
		t.Fatalf("first body = %s, want no page token on the first page", calls[0].body)
	}
	if !strings.Contains(calls[1].body, `"nextPageToken":"p2"`) {
		t.Fatalf("second body = %s, want the token of the first reply", calls[1].body)
	}
}

// TestJiraGatewayCollectionTreeKeepsTheCollectionSentinel proves the refusal mapping of the tree walk: a JQL the
// server refuses ends with the collection sentinel, and an ID outside the project shape never sends a request.
func TestJiraGatewayCollectionTreeKeepsTheCollectionSentinel(t *testing.T) {
	transport := &cannedJiraTransport{scripts: map[string][]cannedJiraReply{
		"POST /rest/api/3/search/jql": {{status: 400, body: `{"unexpected":true}`}},
	}}
	gw := jiraFixture(t, transport)
	collection, err := domain.NewCollectionID("project-EXAMPLE")
	if err != nil {
		t.Fatalf("NewCollectionID: %v", err)
	}
	if _, err := gw.CollectionTree(context.Background(), collection); !errors.Is(err, domain.ErrCollectionNotFound) {
		t.Fatalf("CollectionTree on a refused JQL = %v, want the collection sentinel", err)
	}
	before := len(transport.callsTo("POST", "/rest/api/3/search/jql"))
	bad, err := domain.NewCollectionID("issue-EXAMPLE-1")
	if err != nil {
		t.Fatalf("NewCollectionID: %v", err)
	}
	if _, err := gw.CollectionTree(context.Background(), bad); err == nil {
		t.Fatal("CollectionTree on a document ID succeeds")
	}
	if got := len(transport.callsTo("POST", "/rest/api/3/search/jql")); got != before {
		t.Fatalf("requests after the refused ID = %d, want none more", got)
	}
	failing := jiraFixture(t, &cannedJiraTransport{scripts: map[string][]cannedJiraReply{
		"POST /rest/api/3/search/jql": {{status: 500, body: `{"unexpected":true}`}},
	}})
	if _, err := failing.CollectionTree(context.Background(), collection); !errors.Is(err, domain.ErrUpstreamFailure) {
		t.Fatalf("CollectionTree on an upstream failure = %v, want it unchanged", err)
	}
}

// TestJiraGatewayTreeNodesRefuseAParentCycle proves the depth guard: a parent chain that never ends is an
// unexpected response, as the Go gateway mapped it.
func TestJiraGatewayTreeNodesRefuseAParentCycle(t *testing.T) {
	item := func(key, parent string) domain.Record {
		fields := []domain.Field{
			{Name: "key", Value: key},
			{Name: "title", Value: key},
			{Name: "url", Value: "https://jira.example.test/browse/" + key},
		}
		if parent != "" {
			fields = append(fields, domain.Field{Name: "parent", Value: parent})
		}
		return domain.NewRecord(fields...)
	}
	cyclic := domain.NewList([]domain.Record{item("EXAMPLE-1", "EXAMPLE-2"), item("EXAMPLE-2", "EXAMPLE-1")})
	if _, err := jiraTreeNodes(cyclic); !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("jiraTreeNodes on a parent cycle = %v, want unexpected", err)
	}
}

// TestJiraGatewayListCollectionsPrefixesEveryProject proves the collection walk: the engine walks every page of
// the project search, and the gateway prefixes each raw project key, as the Go gateway does.
func TestJiraGatewayListCollectionsPrefixesEveryProject(t *testing.T) {
	transport := &cannedJiraTransport{scripts: map[string][]cannedJiraReply{
		"GET /rest/api/3/project/search": {
			{status: 200, body: `{"startAt":0,"maxResults":50,"total":3,"isLast":false,` +
				`"values":[{"key":"EXAMPLE","name":"Synthetic project"}]}`},
			{status: 200, body: `{"startAt":1,"maxResults":50,"total":3,"isLast":true,` +
				`"values":[{"key":"OTHER","name":"Second project"}]}`},
		},
	}}
	gw := jiraFixture(t, transport)
	collections, err := gw.ListCollections(context.Background())
	if err != nil {
		t.Fatalf("ListCollections = %v, want nil", err)
	}
	if len(collections) != 2 || collections[0].ID != "project-EXAMPLE" || collections[1].ID != "project-OTHER" {
		t.Fatalf("collections = %+v, want both projects prefixed", collections)
	}
	if collections[0].Name != "Synthetic project" || collections[1].Name != "Second project" {
		t.Fatalf("collections = %+v, want both project names", collections)
	}
	calls := transport.callsTo("GET", "/rest/api/3/project/search")
	if len(calls) != 2 {
		t.Fatalf("requests to project/search = %d, want one per page", len(calls))
	}
	failing := jiraFixture(t, &cannedJiraTransport{scripts: map[string][]cannedJiraReply{
		"GET /rest/api/3/project/search": {{status: 500, body: `{"unexpected":true}`}},
	}})
	if _, err := failing.ListCollections(context.Background()); !errors.Is(err, domain.ErrUpstreamFailure) {
		t.Fatalf("ListCollections on an upstream failure = %v, want it unchanged", err)
	}
	hostile := jiraFixture(t, &cannedJiraTransport{scripts: map[string][]cannedJiraReply{
		"GET /rest/api/3/project/search": {{status: 200, body: `{"isLast":true,` +
			`"values":[{"key":"not a key","name":"Hostile project"}]}`}},
	}})
	if _, err := hostile.ListCollections(context.Background()); !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("ListCollections on a hostile project key = %v, want unexpected", err)
	}
}

// TestJiraGatewayDocumentMapsTheGetRead proves the document gateway: one issue reads through the get operation,
// and maps to the five document fields. A missing issue keeps the document sentinel, and an ID outside the issue
// shape never sends a request.
func TestJiraGatewayDocumentMapsTheGetRead(t *testing.T) {
	transport := &cannedJiraTransport{scripts: map[string][]cannedJiraReply{
		"GET /rest/api/3/issue/EXAMPLE-1": {{status: 200, body: `{"id":"10010","key":"EXAMPLE-1",` +
			`"fields":{"summary":"Synthetic issue"}}`}},
	}}
	gw := jiraFixture(t, transport)
	document, err := domain.NewDocumentID("issue-EXAMPLE-1")
	if err != nil {
		t.Fatalf("NewDocumentID: %v", err)
	}
	doc, err := gw.Document(context.Background(), document)
	if err != nil {
		t.Fatalf("Document = %v, want nil", err)
	}
	if doc.ID != "issue-EXAMPLE-1" || doc.Title != "Synthetic issue" {
		t.Fatalf("document = %+v, want the id and title of the issue", doc)
	}
	if doc.URL != "https://jira.example.test/browse/EXAMPLE-1" || doc.CollectionID != "project-EXAMPLE" {
		t.Fatalf("document = %+v, want the built URL and collection", doc)
	}
	missing, err := domain.NewDocumentID("issue-EXAMPLE-999")
	if err != nil {
		t.Fatalf("NewDocumentID: %v", err)
	}
	if _, err := gw.Document(context.Background(), missing); !errors.Is(err, domain.ErrDocumentNotFound) {
		t.Fatalf("Document on a missing issue = %v, want the document sentinel", err)
	}
	bad, err := domain.NewDocumentID("project-EXAMPLE")
	if err != nil {
		t.Fatalf("NewDocumentID: %v", err)
	}
	if _, err := gw.Document(context.Background(), bad); err == nil {
		t.Fatal("Document on a collection ID succeeds")
	}
}
