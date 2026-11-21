package unit_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// wireIssueDocOps is the document read of one issue: a title, the metadata lines with their skip, default, valid, and join
// rules, and a description that converts from ADF under a fixed heading with an empty text.
const wireIssueDocOps = `  - id: example-wiki.issues.get
    summary: Read one issue as a document.
    access: read
    method: GET
    path: /issues/{issueKey}
    params:
      - name: issueKey
        type: IssueKey
        in: path
        required: true
    response:
      echo: { param: issueKey, pointer: /key }
      fields:
        key:
          pointer: /key
        summary:
          pointer: /fields/summary
          convert: jira-title
        projectName:
          pointer: /fields/project/name
        issuetype:
          pointer: /fields/issuetype/name
        resolution:
          pointer: /fields/resolution/name
        assignee:
          pointer: /fields/assignee/displayName
        reporter:
          pointer: /fields/reporter/displayName
        labels:
          pointer: /fields/labels
          format: strings
        parent:
          pointer: /fields/parent/key
        updated:
          pointer: /fields/updated
          convert: jira-time
        due:
          pointer: /fields/duedate
          convert: jira-date
        description:
          pointer: /fields/description
          convert: markdown-from-adf
      document:
        title: summary
        lines:
          - { label: Key, verbatim: key }
          - { label: Project, pair: { name: projectName, key: key } }
          - { label: Type, field: issuetype }
          - { label: Resolution, field: resolution, skip_empty: true }
          - { label: Assignee, field: assignee, default: Unassigned }
          - { label: Reporter, field: reporter }
          - { label: Labels, field: labels, join: ", ", skip_empty: true }
          - { label: Parent, field: parent, valid: issue_key, skip_empty: true }
          - { label: Updated, verbatim: updated }
          - { label: Due, verbatim: due, skip_empty: true }
        body: { field: description, convert: markdown-from-adf, heading: Description, empty: "(no description)" }
    examples:
      - request:
          issueKey: EXAMPLE-12
        response:
          status: 200
          body:
            key: EXAMPLE-12
            fields:
              summary: Login fails on second try
        expect:
          key: EXAMPLE-12
          summary: Login fails on second try
`

// wireBuiltOps is the issue read whose identifiers are built from its key: a project key, an issue ID, a collection ID,
// and a browse URL, next to a plain field that carries the reply's own self link.
const wireBuiltOps = `  - id: example-wiki.issues.get
    summary: Read one issue with built IDs.
    access: read
    method: GET
    path: /issues/{issueKey}
    params:
      - name: issueKey
        type: IssueKey
        in: path
        required: true
    response:
      fields:
        projectKey:
          project_key: /key
        id:
          issue_id: /key
        collectionId:
          collection_id: /key
        url:
          browse_url: /key
        self:
          pointer: /self
    examples:
      - request:
          issueKey: EXAMPLE-12
        response:
          status: 200
          body:
            key: EXAMPLE-12
            self: https://docs.example.test/rest/api/3/issue/EXAMPLE-12
        expect:
          projectKey: EXAMPLE
          id: issue-EXAMPLE-12
          collectionId: project-EXAMPLE
          url: https://docs.example.test/browse/EXAMPLE-12
`

// wireBuiltListOps is a list read whose items each carry an issue key, with the project key built from each item's key.
const wireBuiltListOps = `  - id: example-wiki.issues.list
    summary: List issues with their project keys.
    access: read
    method: GET
    path: /issues
    response:
      items: /items
      fields:
        projectKey:
          project_key: /key
        key:
          pointer: /key
    examples:
      - request: {}
        response:
          status: 200
          body:
            items:
              - key: ALPHA-1
        expect:
          key: ALPHA-1
          projectKey: ALPHA
`

// wireLinkOps is a read whose only field is a browse URL built from the key, so that the built URL alone can exceed a
// small reply cap while the raw reply still fits it.
const wireLinkOps = `  - id: example-wiki.issues.link
    summary: Read the browse link of one issue.
    access: read
    method: GET
    path: /issues/{issueKey}
    params:
      - name: issueKey
        type: IssueKey
        in: path
        required: true
    response:
      fields:
        url:
          browse_url: /key
    examples:
      - request:
          issueKey: EXAMPLE-12
        response:
          status: 200
          body:
            key: EXAMPLE-12
        expect:
          url: https://docs.example.test/browse/EXAMPLE-12
`

// wireIssueDocBody is a reply that holds every declared field of one issue.
const wireIssueDocBody = `{"key":"EXAMPLE-12","fields":{` +
	`"summary":"Login fails on second try",` +
	`"project":{"name":"Example Project"},` +
	`"issuetype":{"name":"Bug"},` +
	`"resolution":{"name":"Fixed"},` +
	`"assignee":{"displayName":"Ana"},` +
	`"reporter":{"displayName":"Budi"},` +
	`"labels":["one","two"],` +
	`"parent":{"key":"EXAMPLE-1"},` +
	`"updated":"2026-11-02T09:15:00.000+0700",` +
	`"duedate":"2026-12-01",` +
	`"description":{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"Hello"}]}]}` +
	`}}`

// wireIssueDocRead runs the document read of EXAMPLE-12 over one reply, and returns the document it renders.
func wireIssueDocRead(t *testing.T, body string) (string, error) {
	t.Helper()
	s := rebindSpec(t, wireIssueDocOps)
	rt := rebindReplies(rebindJSON(200, body))
	eng := rebindEngine(t, s, rt, &rebindClock{})
	rec, err := eng.Read(context.Background(), "example-wiki.issues.get", domain.NewArgs(map[string]any{"issueKey": "EXAMPLE-12"}))
	if err != nil {
		return "", err
	}
	return rec.Str("markdown"), nil
}

// TestDocumentValuesStayLiteralText reads a document whose summary and field values hold a heading-like text and a link.
//
// @scenario specs/libs/retriever-engine/behaviours/documents-and-built-ids.feature: Markup in a summary or a field value stays text in the document
func TestDocumentValuesStayLiteralText(t *testing.T) {
	body := `{"key":"EXAMPLE-12","fields":{"summary":"Fix [link](https://x.test)","issuetype":{"name":"**Bug**"}}}`
	doc, err := wireIssueDocRead(t, body)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	for _, want := range []string{`# Fix \[link\](https://x.test)`, `- Type: \*\*Bug\*\*`} {
		if !strings.Contains(doc, want) {
			t.Fatalf("document %q does not hold the literal text %q", doc, want)
		}
	}
}

// TestDocumentSkipsEmptyLinesOnlyWhereTheLayoutSkips reads a document whose skipped line and unskipped line are empty.
//
// @scenario specs/libs/retriever-engine/behaviours/documents-and-built-ids.feature: A metadata line with an empty value is skipped where its layout skips it and kept where it does not
func TestDocumentSkipsEmptyLinesOnlyWhereTheLayoutSkips(t *testing.T) {
	doc, err := wireIssueDocRead(t, `{"key":"EXAMPLE-12","fields":{"summary":"Login fails"}}`)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if strings.Contains(doc, "- Resolution:") {
		t.Fatalf("the skipped line is present:\n%s", doc)
	}
	if !strings.Contains(doc, "- Reporter: \n") {
		t.Fatalf("the unskipped line is not present and empty:\n%s", doc)
	}
}

// TestDocumentShowsDefaultsAndEndsWithoutALineBreak reads a document whose assignee is unset and whose description is null.
//
// @scenario specs/libs/retriever-engine/behaviours/documents-and-built-ids.feature: An unassigned value shows its default and a null description ends the document with empty text
func TestDocumentShowsDefaultsAndEndsWithoutALineBreak(t *testing.T) {
	doc, err := wireIssueDocRead(t, `{"key":"EXAMPLE-12","fields":{"summary":"Login fails","assignee":null,"description":null}}`)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if !strings.Contains(doc, "- Assignee: Unassigned\n") {
		t.Fatalf("the assignee line does not show the default:\n%s", doc)
	}
	if !strings.HasSuffix(doc, "(no description)") {
		t.Fatalf("the document does not end with the empty description:\n%q", doc)
	}
}

// TestDocumentOfAFullObjectHoldsEveryLineInLayoutOrder reads an issue that holds every declared field.
//
// @scenario specs/libs/retriever-engine/behaviours/documents-and-built-ids.feature: A full document of an object with every field renders the title, each metadata line, and the converted description
func TestDocumentOfAFullObjectHoldsEveryLineInLayoutOrder(t *testing.T) {
	doc, err := wireIssueDocRead(t, wireIssueDocBody)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	want := "# Login fails on second try\n\n" +
		"- Key: EXAMPLE-12\n" +
		"- Project: Example Project (EXAMPLE-12)\n" +
		"- Type: Bug\n" +
		"- Resolution: Fixed\n" +
		"- Assignee: Ana\n" +
		"- Reporter: Budi\n" +
		"- Labels: one, two\n" +
		"- Parent: EXAMPLE-1\n" +
		"- Updated: 2026-11-02T02:15:00Z\n" +
		"- Due: 2026-12-01\n" +
		"\n## Description\n\nHello"
	if doc != want {
		t.Fatalf("document =\n%s\nwant\n%s", doc, want)
	}
}

// TestDocumentLayoutOnAListIsRefused hand-builds a list read that carries a document layout, and builds the engine.
//
// @scenario specs/libs/retriever-engine/behaviours/documents-and-built-ids.feature: A document layout on a list read is refused before any request
func TestDocumentLayoutOnAListIsRefused(t *testing.T) {
	layout := rebindSpec(t, wireIssueDocOps).Operations[0].Response.Document
	base := rebindSpec(t, wireBuiltListOps)
	model := *base
	ops := append([]spec.Operation(nil), base.Operations...)
	ops[0].Response.Document = layout
	model.Operations = ops
	rt := rebindReplies(rebindJSON(200, "{}"))
	eng, err := engine.New(&model, rebindSource(t), wirePorts(&rebindClock{}, rt))
	var inv *spec.InvalidError
	if !errors.As(err, &inv) {
		t.Fatalf("err = %v, want the spec refusal", err)
	}
	if eng != nil {
		t.Fatal("an engine was returned for a document layout on a list")
	}
	if n := len(rt.seen()); n != 0 {
		t.Fatalf("the refused model sent %d requests, want 0", n)
	}
}

// TestBrowseURLIsBuiltFromTheEndpointOrigin plants another address in the reply's own URL field.
//
// @scenario specs/libs/retriever-engine/behaviours/documents-and-built-ids.feature: A browse URL is built from the endpoint origin and ignores a URL that the reply holds
func TestBrowseURLIsBuiltFromTheEndpointOrigin(t *testing.T) {
	const planted = "https://attacker.example/evil"
	s := rebindSpec(t, wireBuiltOps)
	rt := rebindReplies(rebindJSON(200, `{"key":"EXAMPLE-12","self":"`+planted+`"}`))
	eng := rebindEngine(t, s, rt, &rebindClock{})
	rec, err := eng.Read(context.Background(), "example-wiki.issues.get", domain.NewArgs(map[string]any{"issueKey": "EXAMPLE-12"}))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got := rec.Str("url"); got != "https://docs.example.test/browse/EXAMPLE-12" {
		t.Fatalf("url = %q, want the endpoint-origin browse URL", got)
	}
}

// TestBuiltIDKeyOutsideTheShapeFailsNamingTheField replies with a key that is outside the issue-key shape.
//
// @scenario specs/libs/retriever-engine/behaviours/documents-and-built-ids.feature: A reply key outside the issue-key shape fails the read and the error names the field, never the value
func TestBuiltIDKeyOutsideTheShapeFailsNamingTheField(t *testing.T) {
	const planted = "not-a-key"
	s := rebindSpec(t, wireBuiltOps)
	rt := rebindReplies(rebindJSON(200, `{"key":"`+planted+`"}`))
	eng := rebindEngine(t, s, rt, &rebindClock{})
	_, err := eng.Read(context.Background(), "example-wiki.issues.get", domain.NewArgs(map[string]any{"issueKey": "EXAMPLE-12"}))
	if !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
	}
	if !strings.Contains(err.Error(), "collectionId") {
		t.Fatalf("the failure does not name the field: %q", err.Error())
	}
	if strings.Contains(err.Error(), planted) {
		t.Fatalf("the failure echoes the value: %q", err.Error())
	}
}

// TestProjectKeyIsTheTextBeforeTheLastHyphen reads a key and checks the project key and the collection ID built from it.
//
// @scenario specs/libs/retriever-engine/behaviours/documents-and-built-ids.feature: A project key is the text before the last hyphen of the key
func TestProjectKeyIsTheTextBeforeTheLastHyphen(t *testing.T) {
	s := rebindSpec(t, wireBuiltOps)
	rt := rebindReplies(rebindJSON(200, `{"key":"EXAMPLE-12","self":"https://docs.example.test/x"}`))
	eng := rebindEngine(t, s, rt, &rebindClock{})
	rec, err := eng.Read(context.Background(), "example-wiki.issues.get", domain.NewArgs(map[string]any{"issueKey": "EXAMPLE-12"}))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got := rec.Str("projectKey"); got != "EXAMPLE" {
		t.Fatalf("projectKey = %q, want EXAMPLE", got)
	}
	if got := rec.Str("collectionId"); got != "project-EXAMPLE" {
		t.Fatalf("collectionId = %q, want project-EXAMPLE", got)
	}
}

// TestBuiltIDKeyThatIsMissingOrNotTextFails replies with no key, and then with a key held as a number.
//
// @scenario specs/libs/retriever-engine/behaviours/documents-and-built-ids.feature: A built identifier whose key is <problem> fails the read naming the field
func TestBuiltIDKeyThatIsMissingOrNotTextFails(t *testing.T) {
	cases := []struct {
		problem string
		body    string
	}{
		{"missing", `{"self":"https://docs.example.test/x"}`},
		{"held as a number", `{"key":12,"self":"https://docs.example.test/x"}`},
	}
	for _, c := range cases {
		t.Run(c.problem, func(t *testing.T) {
			s := rebindSpec(t, wireBuiltOps)
			rt := rebindReplies(rebindJSON(200, c.body))
			eng := rebindEngine(t, s, rt, &rebindClock{})
			_, err := eng.Read(context.Background(), "example-wiki.issues.get", domain.NewArgs(map[string]any{"issueKey": "EXAMPLE-12"}))
			if !errors.Is(err, domain.ErrUnexpectedResponse) {
				t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
			}
			if !strings.Contains(err.Error(), "collectionId") {
				t.Fatalf("the failure does not name the field: %q", err.Error())
			}
		})
	}
}

// TestListReadBuildsEachItemFromItsOwnKey reads a list whose items each carry an issue key.
//
// @scenario specs/libs/retriever-engine/behaviours/documents-and-built-ids.feature: A list read builds the identifiers of each item from that item's own key
func TestListReadBuildsEachItemFromItsOwnKey(t *testing.T) {
	s := rebindSpec(t, wireBuiltListOps)
	rt := rebindReplies(rebindJSON(200, `{"items":[{"key":"ALPHA-1"},{"key":"BETA-22"}]}`))
	eng := rebindEngine(t, s, rt, &rebindClock{})
	rec, err := eng.Read(context.Background(), "example-wiki.issues.list", domain.NewArgs(nil))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	items := rec.Items()
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
	if items[0].Str("projectKey") != "ALPHA" || items[1].Str("projectKey") != "BETA" {
		t.Fatalf("project keys = %q, %q; want ALPHA, BETA", items[0].Str("projectKey"), items[1].Str("projectKey"))
	}
}

// TestBuiltURLPastTheBudgetFailsWithTheReplySizeError builds a browse URL that is longer than the reply cap, while the raw
// reply fits the cap.
//
// @scenario specs/libs/retriever-engine/behaviours/documents-and-built-ids.feature: A built URL past the shared output budget fails with the reply-size error
func TestBuiltURLPastTheBudgetFailsWithTheReplySizeError(t *testing.T) {
	const body = `{"key":"EXAMPLE-12"}`
	s := wireCapped(t, 40, wireLinkOps)
	rt := rebindReplies(rebindJSON(200, body))
	eng := rebindEngine(t, s, rt, &rebindClock{})
	_, err := eng.Read(context.Background(), "example-wiki.issues.link", domain.NewArgs(map[string]any{"issueKey": "EXAMPLE-12"}))
	if !errors.Is(err, domain.ErrReplyTooLarge) {
		t.Fatalf("err = %v, want ErrReplyTooLarge", err)
	}
}
