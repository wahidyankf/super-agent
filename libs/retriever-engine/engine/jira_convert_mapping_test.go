package engine_test

import (
	"context"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// jiraYAML is a read whose reply holds Jira timestamps, due dates, and summaries, mapped with the Jira text
// converters. It names example hosts only.
const jiraYAML = `version: 1
kind: example-wiki
summary: Reads issues with Jira text converters.
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
    path: /issues/{id}
    params:
      - name: id
        type: ID
        in: path
        required: true
    response:
      fields:
        updated:
          pointer: /fields/updated
          convert: jira-time
        due:
          pointer: /fields/duedate
          convert: jira-date
        summary:
          pointer: /fields/summary
          convert: jira-title
        meta:
          object:
            notes:
              pointer: /fields/notes
              convert: jira-title
    examples:
      - request:
          id: doc-1
        response:
          status: 200
          body:
            fields:
              updated: "2026-11-02T09:15:00.000+0700"
              duedate: "2026-12-01"
              summary: First issue
        expect:
          updated: "2026-11-02T02:15:00Z"
          due: "2026-12-01"
          summary: First issue
`

// TestAJiraReadMapsTimeDateAndTitle checks that a read naming the Jira converters is served: the timestamp answers
// in UTC, the date passes through, and the title stays one line, in a plain field and in a nested record.
func TestAJiraReadMapsTimeDateAndTitle(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, jiraYAML))
	stub.Script("GET", "/v1/issues/doc-1", page(`{"fields":{"updated":"2026-11-02T09:15:00.000+0700","duedate":"2026-12-01","summary":"First issue","notes":"a note"}}`))
	rec, err := eng.Read(context.Background(), "example-wiki.issues.get", domain.NewArgs(map[string]any{"id": "doc-1"}))
	if err != nil {
		t.Fatalf("Read() error = %v, want nil", err)
	}
	if got := rec.Str("updated"); got != "2026-11-02T02:15:00Z" {
		t.Fatalf("updated = %q, want the UTC instant", got)
	}
	if got := rec.Str("due"); got != "2026-12-01" {
		t.Fatalf("due = %q, want the calendar day", got)
	}
	if got := rec.Str("summary"); got != "First issue" {
		t.Fatalf("summary = %q, want the one-line title", got)
	}
	meta, ok := rec.Get("meta")
	if !ok {
		t.Fatal("meta is missing")
	}
	metaRec, ok := meta.(domain.Record)
	if !ok {
		t.Fatalf("meta = %#v, want a record", meta)
	}
	if got := metaRec.Str("notes"); got != "a note" {
		t.Fatalf("meta.notes = %q, want the converted title", got)
	}
	if n := len(stub.Requests()); n != 1 {
		t.Fatalf("a served converter sent %d requests, want 1", n)
	}
}

// TestAJiraTitleStripsABidiOverride checks that a bidi override in a title reaches no output: the field holds the
// visible text only.
func TestAJiraTitleStripsABidiOverride(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, jiraYAML))
	stub.Script("GET", "/v1/issues/doc-1", page(`{"fields":{"updated":"2026-11-02T09:15:00.000+0700","duedate":"2026-12-01","summary":"a‮b"}}`))
	rec, err := eng.Read(context.Background(), "example-wiki.issues.get", domain.NewArgs(map[string]any{"id": "doc-1"}))
	if err != nil {
		t.Fatalf("Read() error = %v, want nil", err)
	}
	if got := rec.Str("summary"); got != "ab" {
		t.Fatalf("summary = %q, want no bidi override", got)
	}
}

// TestAJiraBadDateMapsToEmptyWithTheFieldPresent checks that a broken date keeps its row: the field is present and
// empty instead of failing the read.
func TestAJiraBadDateMapsToEmptyWithTheFieldPresent(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, jiraYAML))
	stub.Script("GET", "/v1/issues/doc-1", page(`{"fields":{"updated":"yesterday","duedate":"2026","summary":"First issue"}}`))
	rec, err := eng.Read(context.Background(), "example-wiki.issues.get", domain.NewArgs(map[string]any{"id": "doc-1"}))
	if err != nil {
		t.Fatalf("Read() error = %v, want nil", err)
	}
	got, ok := rec.Get("updated")
	if !ok || got != "" {
		t.Fatalf("updated = (%#v, %v), want (\"\", true)", got, ok)
	}
	due, ok := rec.Get("due")
	if !ok || due != "" {
		t.Fatalf("due = (%#v, %v), want (\"\", true)", due, ok)
	}
}
