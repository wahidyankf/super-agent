package engine_test

import (
	"context"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine/replay"
)

// docLayoutYAML is a read of one issue with the document layout: the title, the closed line kinds with defaults and
// skips, and the description body with its fixed heading and empty text. It names example hosts only.
const docLayoutYAML = `version: 1
kind: example-wiki
summary: Reads one issue as a document.
endpoint:
  base_path: /v1
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
operations:
  - id: example-wiki.issues.get
    summary: Read one issue.
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

// docMarkdownOf runs the document read against one scripted reply and returns its markdown.
func docMarkdownOf(t *testing.T, body string) string {
	t.Helper()
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, docLayoutYAML))
	stub.Script("GET", "/v1/issues/EXAMPLE-12", page(body))
	rec, err := eng.Read(context.Background(), "example-wiki.issues.get", domain.NewArgs(map[string]any{"issueKey": "EXAMPLE-12"}))
	if err != nil {
		t.Fatalf("Read() error = %v, want nil", err)
	}
	return rec.Str("markdown")
}

// TestDocLayoutPlantStaysText plants a heading and a link in the summary, the labels, and a field value, and
// finds each literal: the layout escapes every value, so planted markup stays text.
func TestDocLayoutPlantStaysText(t *testing.T) {
	md := docMarkdownOf(t, `{"key":"EXAMPLE-12","fields":{"summary":"# Hi from [evil](https://example.test/x)‮","project":{"name":"Example Org"},"issuetype":{"name":"[evil type]"},"assignee":{"displayName":"Ann"},"reporter":{"displayName":"Bob"},"labels":["# nope","[evil](https://example.test/x)"],"parent":{"key":"EXAMPLE-1"},"updated":"2026-11-04T16:30:00.000+0700","description":"Plain words."}}`)
	head := strings.SplitN(md, "\n", 2)[0]
	if head != `# \# Hi from \[evil\](https://example.test/x)` {
		t.Fatalf("the title line = %q, want the planted markup escaped", head)
	}
	for _, want := range []string{
		"- Labels: \\# nope, \\[evil\\](https://example.test/x)",
		"- Type: \\[evil type\\]",
	} {
		if !strings.Contains(md, want) {
			t.Fatalf("the document keeps no escaped plant %q:\n%s", want, md)
		}
	}
	for _, raw := range []string{"[evil](", "[evil type]", "\n# nope", "‮"} {
		if strings.Contains(md, raw) {
			t.Fatalf("planted markup reaches the document raw: %q in\n%s", raw, md)
		}
	}
}

// TestDocLayoutSkipsEmptyLines checks that an empty resolution, labels, parent, and due leave no line: the skip
// rule omits the line when the value is still empty, while a line without it is emitted even when empty.
func TestDocLayoutSkipsEmptyLines(t *testing.T) {
	md := docMarkdownOf(t, `{"key":"EXAMPLE-12","fields":{"summary":"A quiet issue","project":{},"issuetype":{"name":"Bug"},"resolution":{"name":""},"labels":[],"parent":{"key":"oops"},"updated":"bad-time","duedate":"not-a-day","description":null}}`)
	for _, absent := range []string{"- Resolution:", "- Labels:", "- Parent:", "- Due:"} {
		if strings.Contains(md, absent) {
			t.Fatalf("the document holds %q for an issue that has none:\n%s", absent, md)
		}
	}
	for _, want := range []string{"- Project: EXAMPLE-12\n", "- Reporter: \n", "- Updated: \n"} {
		if !strings.Contains(md, want) {
			t.Fatalf("the document misses %q:\n%s", want, md)
		}
	}
}

// TestDocLayoutDefaultsExact checks that the defaults are byte-identical: an unassigned issue shows Unassigned,
// and a null description ends the document with the empty text.
func TestDocLayoutDefaultsExact(t *testing.T) {
	md := docMarkdownOf(t, `{"key":"EXAMPLE-12","fields":{"summary":"A quiet issue","project":{},"issuetype":{"name":"Bug"},"updated":"2026-11-04T16:30:00.000+0700","description":null}}`)
	assignee := ""
	for _, line := range strings.Split(md, "\n") {
		if strings.HasPrefix(line, "- Assignee: ") {
			assignee = line
		}
	}
	if assignee != "- Assignee: Unassigned" {
		t.Fatalf("the assignee line = %q, want %q", assignee, "- Assignee: Unassigned")
	}
	if !strings.HasSuffix(md, "## Description\n\n(no description)") {
		t.Fatalf("the document does not end with the empty text:\n%s", md)
	}
}

// TestDocLayoutRendersAFulIssue pins the whole document of an issue with every field: the title, each metadata
// line, and the converted description, ending with no trailing line break.
func TestDocLayoutRendersAFulIssue(t *testing.T) {
	md := docMarkdownOf(t, `{"key":"EXAMPLE-12","fields":{"summary":"Login fails on second try","project":{"name":"Example Org"},"issuetype":{"name":"Bug"},"resolution":{"name":"Done"},"assignee":{"displayName":"Fixture Reviewer"},"reporter":{"displayName":"Fixture Author"},"labels":["login","beta"],"parent":{"key":"EXAMPLE-1"},"updated":"2026-11-04T16:30:00.000+0700","duedate":"2026-12-01","description":"Fixed it."}}`)
	want := "# Login fails on second try\n\n" +
		"- Key: EXAMPLE-12\n" +
		"- Project: Example Org (EXAMPLE-12)\n" +
		"- Type: Bug\n" +
		"- Resolution: Done\n" +
		"- Assignee: Fixture Reviewer\n" +
		"- Reporter: Fixture Author\n" +
		"- Labels: login, beta\n" +
		"- Parent: EXAMPLE-1\n" +
		"- Updated: 2026-11-04T09:30:00Z\n" +
		"- Due: 2026-12-01\n" +
		"\n## Description\n\nFixed it."
	if md != want {
		t.Fatalf("the document differs.\n--- got\n%s\n--- want\n%s", md, want)
	}
}

// TestDocLayoutOnListIsRefused checks that the engine refuses a hand-built model with a document on a list
// response, the same refusal that validation reports, before any request is sent.
func TestDocLayoutOnListIsRefused(t *testing.T) {
	s := loadSpecFrom(t, docLayoutYAML)
	s.Operations[0].Response.Items = "/issues"
	stub, err := replay.New(loadSpecFrom(t, docLayoutYAML))
	if err != nil {
		t.Fatalf("build the replay stub: %v", err)
	}
	if _, err := engine.New(s, testSource(t), stubPorts(stub, &fakeClock{})); err == nil {
		t.Fatal("a document on a list response is accepted")
	} else if !strings.Contains(err.Error(), "/operations/0/response/document") {
		t.Fatalf("the refusal names no document path: %q", err.Error())
	}
}
