package spec

import "testing"

// TestJiraConvertersValidate checks that a reply field may name each Jira text converter: jira-time rewrites a Jira
// timestamp as RFC 3339 in UTC, jira-date keeps a calendar day, and jira-title keeps one line of at most 255 runes.
// Each converter is a pure function of the reply: it sends no request and reaches no other host. A value that does
// not parse maps to the empty text, so the row is kept with a visible gap.
func TestJiraConvertersValidate(t *testing.T) {
	doc := `version: 1
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
`
	if res := Validate([]byte(doc)); !res.Valid() {
		t.Fatalf("a field with a Jira converter is refused: %v", res.Err())
	}
}
