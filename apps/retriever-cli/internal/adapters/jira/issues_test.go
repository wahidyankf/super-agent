package jira

import (
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// issueText returns the Markdown of one issue of the stub, through the read operation.
func issueText(t *testing.T, gw *Gateway, key string) domain.Record {
	t.Helper()
	rec, err := readOp(gw, domain.OpJiraIssuesGet, map[string]any{"issueKey": key})
	if err != nil {
		t.Fatalf("issues get %s: %v", key, err)
	}
	return rec
}

// TestAnIssueIsReadAsItsDocumentFields proves the document fields: the ID, the title, the URL on the endpoint, the collection,
// and the Markdown. The URL is built from the endpoint, not from a reply.
func TestAnIssueIsReadAsItsDocumentFields(t *testing.T) {
	r := newRig(t)
	rec := issueText(t, r.gw, "EXAMPLE-12")
	want := []string{"id", "title", "url", "markdown", "collectionId"}
	if got := fieldNames(rec); !reflect.DeepEqual(got, want) {
		t.Fatalf("fields = %v, want %v", got, want)
	}
	if rec.Str("id") != "issue-EXAMPLE-12" || rec.Str("title") != "Login fails on second try" {
		t.Fatalf("id %q title %q", rec.Str("id"), rec.Str("title"))
	}
	if rec.Str("url") != "https://"+testHost+"/browse/EXAMPLE-12" {
		t.Fatalf("url = %q", rec.Str("url"))
	}
	if rec.Str("collectionId") != "project-EXAMPLE" {
		t.Fatalf("collectionId = %q", rec.Str("collectionId"))
	}
}

// TestTheIssueMarkdownHasItsMetadataAndDescription pins the document layout for an issue with every field.
func TestTheIssueMarkdownHasItsMetadataAndDescription(t *testing.T) {
	r := newRig(t)
	md := issueText(t, r.gw, "EXAMPLE-12").Str("markdown")
	head := "# Login fails on second try\n\n" +
		"- Key: EXAMPLE-12\n" +
		"- Project: Example Org (EXAMPLE)\n" +
		"- Type: Bug\n" +
		"- Status: To Do\n" +
		"- Priority: Medium\n" +
		"- Assignee: Fixture Reviewer\n" +
		"- Reporter: Fixture Author\n" +
		"- Labels: login, beta\n" +
		"- Parent: EXAMPLE-1\n" +
		"- Created: 2026-11-02T02:15:00Z\n" +
		"- Updated: 2026-11-04T09:30:00Z\n" +
		"- Due: 2026-12-01\n\n" +
		"## Description\n\n"
	if !strings.HasPrefix(md, head) {
		t.Fatalf("markdown does not start with the metadata:\n%s", md)
	}
	body := strings.TrimPrefix(md, head)
	for _, want := range []string{"#### Steps", "- Open the login page", "[runbook](<https://wiki.example.com/runbook>)", "[attachment: screenshot.png]"} {
		if !strings.Contains(body, want) {
			t.Fatalf("description misses %q:\n%s", want, body)
		}
	}
}

// TestTheIssueMarkdownHoldsNoAccountIDNoMediaIDAndNoEmail proves the privacy rules: a mention's account ID, a media ID, and any
// e-mail address stay out of the document.
func TestTheIssueMarkdownHoldsNoAccountIDNoMediaIDAndNoEmail(t *testing.T) {
	r := newRig(t)
	md := issueText(t, r.gw, "EXAMPLE-12").Str("markdown")
	for _, secret := range []string{"5b10ac8d82e05b22cc7d4ef5", "media-secret-1", "fixture-reviewer@example.com", sentinelEmail, sentinelToken} {
		if strings.Contains(md, secret) {
			t.Fatalf("the document holds %q", secret)
		}
	}
}

// TestAnIssueWithoutADescriptionSaysSo proves a null description is written as "(no description)", and an unassigned issue
// shows Unassigned. Empty optional fields are left out.
func TestAnIssueWithoutADescriptionSaysSo(t *testing.T) {
	r := newRig(t)
	md := issueText(t, r.gw, "EXAMPLE-13").Str("markdown")
	if !strings.HasSuffix(md, "## Description\n\n(no description)") {
		t.Fatalf("markdown does not end with the no-description line:\n%s", md)
	}
	for _, want := range []string{"- Assignee: Unassigned", "- Resolution: Done"} {
		if !strings.Contains(md, want) {
			t.Fatalf("markdown misses %q", want)
		}
	}
	for _, absent := range []string{"- Parent:", "- Labels:", "- Due:"} {
		if strings.Contains(md, absent) {
			t.Fatalf("markdown holds %q for an issue that has none", absent)
		}
	}
}

// TestHostileTextIsEscapedAndSafeLinksAreKept proves a hostile summary and description cannot make Markdown: the heading and
// the text are escaped, a javascript link is dropped, and a control character is gone.
func TestHostileTextIsEscapedAndSafeLinksAreKept(t *testing.T) {
	r := newRig(t)
	rec := issueText(t, r.gw, "EXAMPLE-14")
	md := rec.Str("markdown")
	if strings.Contains(strings.SplitN(md, "\n", 2)[0], "<b>") {
		t.Fatalf("the heading holds raw HTML: %q", md)
	}
	if !strings.Contains(md, `\<b\>summary\</b\>`) {
		t.Fatalf("the summary is not escaped:\n%s", md)
	}
	if strings.Contains(md, "](<javascript:") || !strings.Contains(md, "click here") {
		t.Fatalf("the javascript link was not dropped to its text:\n%s", md)
	}
	if strings.ContainsRune(md, '\x1b') || strings.ContainsRune(md, '\u202e') || strings.ContainsRune(md, '\u200b') {
		t.Fatal("a control or direction character reached the document")
	}
}

// TestTheTitleIsOneLineWithoutControlCharacters proves the title field is plain text, with line breaks turned into spaces.
func TestTheTitleIsOneLineWithoutControlCharacters(t *testing.T) {
	body := `{"id":"10012","key":"EXAMPLE-12","fields":{"summary":"first\nsecond\u0007 line","issuetype":{"name":"Bug"},"status":{"name":"To Do"},"created":"","updated":""}}`
	p := &probe{script: func(int, *http.Request) (*http.Response, error) {
		return reply(http.StatusOK, body), nil
	}}
	gw := probeGateway(t, p, newFakeClock(), newSecrets())
	rec := issueText(t, gw, "EXAMPLE-12")
	if rec.Str("title") != "first second line" {
		t.Fatalf("title = %q, want one plain line", rec.Str("title"))
	}
}

// TestAnIssueRequestAsksOnlyTheFixedFieldsAndNoExpansion proves the query holds the fixed field list, and no expand member.
func TestAnIssueRequestAsksOnlyTheFixedFieldsAndNoExpansion(t *testing.T) {
	r := newRig(t)
	issueText(t, r.gw, "EXAMPLE-12")
	reqs := r.stub.Requests()
	if len(reqs) != 1 || reqs[0].Path != "/rest/api/3/issue/EXAMPLE-12" || reqs[0].Method != http.MethodGet {
		t.Fatalf("requests %+v, want one GET of the issue", reqs)
	}
	q, err := url.ParseQuery(reqs[0].Query)
	if err != nil {
		t.Fatalf("ParseQuery: %v", err)
	}
	if q.Get("fields") != issueFieldsList || q.Has("expand") || q.Has("properties") {
		t.Fatalf("query %v, want the fixed fields and no expand", q)
	}
}

// TestAMissingIssueIsDocumentNotFound proves the server's 404 for an issue is the document sentinel.
func TestAMissingIssueIsDocumentNotFound(t *testing.T) {
	r := newRig(t)
	_, err := readOp(r.gw, domain.OpJiraIssuesGet, map[string]any{"issueKey": "EXAMPLE-999"})
	if !errors.Is(err, domain.ErrDocumentNotFound) {
		t.Fatalf("err = %v, want ErrDocumentNotFound", err)
	}
}

// TestAMalformedIssueKeyIsRefusedBeforeAnyRequest proves a lower-case key, a path, or a key with no number is refused by the
// catalog, with no request.
func TestAMalformedIssueKeyIsRefusedBeforeAnyRequest(t *testing.T) {
	for _, key := range []string{"example-12", "EXAMPLE", "EXAMPLE-12/../x", "EXAMPLE-0"} {
		r := newRig(t)
		_, err := readOp(r.gw, domain.OpJiraIssuesGet, map[string]any{"issueKey": key})
		if !errors.Is(err, domain.ErrInvalidDocumentID) {
			t.Fatalf("key %q: err = %v, want ErrInvalidDocumentID", key, err)
		}
		if n := len(r.stub.Requests()); n != 0 {
			t.Fatalf("key %q sent %d requests", key, n)
		}
	}
}

// TestAReplyThatIsNotTheRequestedIssueIsUnexpected proves the reply must name the issue asked for, and must hold its ID,
// its fields, and its summary.
func TestAReplyThatIsNotTheRequestedIssueIsUnexpected(t *testing.T) {
	bodies := map[string]string{
		"another key":   `{"id":"1","key":"EXAMPLE-99","fields":{"summary":"s"}}`,
		"no id":         `{"key":"EXAMPLE-12","fields":{"summary":"s"}}`,
		"no fields":     `{"id":"1","key":"EXAMPLE-12"}`,
		"no summary":    `{"id":"1","key":"EXAMPLE-12","fields":{}}`,
		"lower-case id": `{"id":"1","key":"example-12","fields":{"summary":"s"}}`,
	}
	for name, body := range bodies {
		p := &probe{script: func(int, *http.Request) (*http.Response, error) {
			return reply(http.StatusOK, body), nil
		}}
		gw := probeGateway(t, p, newFakeClock(), newSecrets())
		if _, err := readOp(gw, domain.OpJiraIssuesGet, map[string]any{"issueKey": "EXAMPLE-12"}); !errors.Is(err, domain.ErrUnexpectedResponse) {
			t.Fatalf("%s: err = %v, want ErrUnexpectedResponse", name, err)
		}
	}
}

// TestCommentsListPagesByOffsetAndLimit proves the page request: startAt is the offset, maxResults is the limit, and the
// comments are the fixture's two, in order, with their author, dates, and Markdown.
func TestCommentsListPagesByOffsetAndLimit(t *testing.T) {
	r := newRig(t)
	rec, err := readOp(r.gw, domain.OpJiraCommentsList, map[string]any{"issueKey": "EXAMPLE-12", "limit": 1, "offset": 1})
	if err != nil {
		t.Fatalf("comments: %v", err)
	}
	reqs := r.stub.Requests()
	if len(reqs) != 1 || reqs[0].Path != "/rest/api/3/issue/EXAMPLE-12/comment" {
		t.Fatalf("requests %+v, want one GET of the comments", reqs)
	}
	q := queryOf(t, reqs[0].Query)
	if q.Get("startAt") != "1" || q.Get("maxResults") != "1" || q.Has("orderBy") || q.Has("expand") {
		t.Fatalf("query %v, want startAt 1 and maxResults 1 with no order or expansion", q)
	}
	items := rec.Items()
	if len(items) != 1 || items[0].Str("id") != "20002" || items[0].Str("author") != "Fixture Reviewer" {
		t.Fatalf("items %+v, want the second comment", items)
	}
	if items[0].Str("created") != "2026-11-03T01:30:00Z" || items[0].Str("updated") != "2026-11-03T02:00:00Z" {
		t.Fatalf("created %q updated %q", items[0].Str("created"), items[0].Str("updated"))
	}
	if !strings.Contains(items[0].Str("markdown"), "Fix is in review.") || !strings.Contains(items[0].Str("markdown"), `\- not a list`) {
		t.Fatalf("markdown %q: the text is missing or its list marker is not escaped", items[0].Str("markdown"))
	}
	if got := fieldNames(items[0]); !reflect.DeepEqual(got, []string{"id", "author", "created", "updated", "markdown"}) {
		t.Fatalf("comment fields = %v", got)
	}
}

// TestCommentsOfAMissingIssueAreDocumentNotFound proves the comments of an issue that does not exist are the document sentinel.
func TestCommentsOfAMissingIssueAreDocumentNotFound(t *testing.T) {
	r := newRig(t)
	_, err := readOp(r.gw, domain.OpJiraCommentsList, map[string]any{"issueKey": "EXAMPLE-999"})
	if !errors.Is(err, domain.ErrDocumentNotFound) {
		t.Fatalf("err = %v, want ErrDocumentNotFound", err)
	}
}

// TestAnIssueWithNoCommentsIsAnEmptyList proves an issue with no comments answers an empty list, not an error.
func TestAnIssueWithNoCommentsIsAnEmptyList(t *testing.T) {
	r := newRig(t)
	rec, err := readOp(r.gw, domain.OpJiraCommentsList, map[string]any{"issueKey": "EXAMPLE-13"})
	if err != nil {
		t.Fatalf("comments: %v", err)
	}
	if !rec.IsList() || len(rec.Items()) != 0 {
		t.Fatalf("result %+v, want an empty list", rec)
	}
}

// TestACommentsReplyNeedsItsMembers proves a comment without an ID, or a reply without the comments list, is unexpected.
func TestACommentsReplyNeedsItsMembers(t *testing.T) {
	bodies := []string{
		`{"startAt":0,"maxResults":50,"total":1}`,
		`{"comments":[{"author":{"displayName":"A"},"created":"2026-11-03T01:30:00.000+0000"}]}`,
		`{"comments":[{"id":"x y","body":null}]}`,
	}
	for _, body := range bodies {
		p := &probe{script: func(int, *http.Request) (*http.Response, error) {
			return reply(http.StatusOK, body), nil
		}}
		gw := probeGateway(t, p, newFakeClock(), newSecrets())
		if _, err := readOp(gw, domain.OpJiraCommentsList, map[string]any{"issueKey": "EXAMPLE-12"}); !errors.Is(err, domain.ErrUnexpectedResponse) {
			t.Fatalf("reply %s: err = %v, want ErrUnexpectedResponse", body, err)
		}
	}
}

// TestAnOperationOutsideTheJiraRowsIsRefused proves a read of another kind, such as an Outline operation, is refused before any
// request, and a write of the Jira kind does not exist.
func TestAnOperationOutsideTheJiraRowsIsRefused(t *testing.T) {
	r := newRig(t)
	for _, op := range []domain.OperationID{domain.OpCollectionsList, domain.OpDocumentsGet} {
		if _, err := readOp(r.gw, op, nil); !errors.Is(err, domain.ErrUnsupportedOperation) {
			t.Fatalf("%s: err = %v, want ErrUnsupportedOperation", op, err)
		}
	}
	if n := len(r.stub.Requests()); n != 0 {
		t.Fatalf("requests = %d, want none", n)
	}
}
