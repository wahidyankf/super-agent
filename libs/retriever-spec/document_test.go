package spec

import (
	"strings"
	"testing"
)

// docLayoutDoc is a read of one issue with the document layout: the title, the closed line kinds with defaults and
// skips, and the description body with its fixed heading and empty text. It names example hosts only.
const docLayoutDoc = `version: 1
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
              project: { name: Example Org }
              issuetype: { name: Bug }
              assignee: { displayName: Fixture Reviewer }
              labels: [login, beta]
              parent: { key: EXAMPLE-1 }
              updated: 2026-11-04T16:30:00.000+0700
              description: Fixed it.
        expect:
          key: EXAMPLE-12
          summary: Login fails on second try
`

// TestDocumentLayoutValidates checks that a read of one object may declare the document layout: the title, the
// closed line kinds, and the description body.
func TestDocumentLayoutValidates(t *testing.T) {
	if r := Validate([]byte(docLayoutDoc)); !r.Valid() {
		t.Fatalf("a read with a document layout is refused: %v", r.Err())
	}
}

// TestDocumentLayoutOnListIsRefused checks that only a read of one object declares a document: a list response
// takes none.
func TestDocumentLayoutOnListIsRefused(t *testing.T) {
	doc := mutateAll(docLayoutDoc,
		"      echo: { param: issueKey, pointer: /key }\n      fields:",
		"      echo: { param: issueKey, pointer: /key }\n      items: /issues\n      fields:")
	r := Validate([]byte(doc))
	if r.Valid() {
		t.Fatal("a document on a list response is valid")
	}
	if !hasFinding(r, "/operations/0/response/document") {
		t.Fatalf("no finding names the document:\n%s", allText(r))
	}
}

// TestDocumentLayoutOnWriteIsRefused checks that only a read declares a document: a write of one object takes none.
func TestDocumentLayoutOnWriteIsRefused(t *testing.T) {
	doc := mutateAll(docLayoutDoc, "    access: read\n    method: GET", "    access: write\n    method: POST")
	r := Validate([]byte(doc))
	if r.Valid() {
		t.Fatal("a document on a write is valid")
	}
	if !hasFinding(r, "/operations/0/response/document") {
		t.Fatalf("no finding names the document:\n%s", allText(r))
	}
}

// TestDocumentLayoutWithMarkdownIsRefused checks that a read names one body only: markdown with a document is
// refused.
func TestDocumentLayoutWithMarkdownIsRefused(t *testing.T) {
	doc := mutateAll(docLayoutDoc, "      document:\n", "      markdown: summary\n      document:\n")
	r := Validate([]byte(doc))
	if r.Valid() {
		t.Fatal("markdown with a document is valid")
	}
	if !hasFinding(r, "/operations/0/response/document") {
		t.Fatalf("no finding names the document:\n%s", allText(r))
	}
}

// TestDocumentLayoutTitleNamesNoFieldIsRefused checks that the title names a field of the operation.
func TestDocumentLayoutTitleNamesNoFieldIsRefused(t *testing.T) {
	doc := mutateAll(docLayoutDoc, "        title: summary", "        title: nope")
	r := Validate([]byte(doc))
	if r.Valid() {
		t.Fatal("a document title naming no field is valid")
	}
	if !hasFinding(r, "/operations/0/response/document/title") {
		t.Fatalf("no finding names the document title:\n%s", allText(r))
	}
}

// TestDocumentLayoutLineFieldNamesNoFieldIsRefused checks that a field line names a field of the operation.
func TestDocumentLayoutLineFieldNamesNoFieldIsRefused(t *testing.T) {
	doc := mutateAll(docLayoutDoc, "- { label: Type, field: issuetype }", "- { label: Type, field: nope }")
	r := Validate([]byte(doc))
	if r.Valid() {
		t.Fatal("a document line naming no field is valid")
	}
	if !hasFinding(r, "/operations/0/response/document/lines/2/field") {
		t.Fatalf("no finding names the line field:\n%s", allText(r))
	}
}

// TestDocumentLayoutJoinOnVerbatimIsRefused checks that a join rides a field line: a verbatim line takes none.
func TestDocumentLayoutJoinOnVerbatimIsRefused(t *testing.T) {
	doc := mutateAll(docLayoutDoc,
		"- { label: Due, verbatim: due, skip_empty: true }",
		"- { label: Due, verbatim: due, join: \", \", skip_empty: true }")
	r := Validate([]byte(doc))
	if r.Valid() {
		t.Fatal("a join on a verbatim line is valid")
	}
	if !hasFinding(r, "/operations/0/response/document/lines/9/join") {
		t.Fatalf("no finding names the line join:\n%s", allText(r))
	}
}

// TestDocumentLayoutJoinOnPairIsRefused checks that a join rides a field line: a pair line takes none.
func TestDocumentLayoutJoinOnPairIsRefused(t *testing.T) {
	doc := mutateAll(docLayoutDoc,
		"- { label: Project, pair: { name: projectName, key: key } }",
		"- { label: Project, pair: { name: projectName, key: key }, join: \", \" }")
	r := Validate([]byte(doc))
	if r.Valid() {
		t.Fatal("a join on a pair line is valid")
	}
	if !hasFinding(r, "/operations/0/response/document/lines/1/join") {
		t.Fatalf("no finding names the line join:\n%s", allText(r))
	}
}

// TestDocumentLayoutJoinNeedsStringsField checks that a join renders a strings field: the fixed separator on a
// text field is refused.
func TestDocumentLayoutJoinNeedsStringsField(t *testing.T) {
	doc := mutateAll(docLayoutDoc,
		"- { label: Type, field: issuetype }",
		"- { label: Type, field: issuetype, join: \", \" }")
	r := Validate([]byte(doc))
	if r.Valid() {
		t.Fatal("a join on a text field is valid")
	}
	if !hasFinding(r, "/operations/0/response/document/lines/2/join") {
		t.Fatalf("no finding names the line join:\n%s", allText(r))
	}
}

// TestDocumentLayoutLineNeedsOneKind checks the Go finding for a hand-built line with no closed kind, or with two:
// a document never carries one, because the schema refuses the line first.
func TestDocumentLayoutLineNeedsOneKind(t *testing.T) {
	for _, line := range []DocLine{{Label: "A"}, {Label: "A", Field: "a", Verbatim: "a"}} {
		op := Operation{Access: "read", Response: Response{
			Fields:   map[string]Field{"a": {Pointer: "/a", Convert: "markdown-from-adf"}},
			Document: &Document{Title: "a", Lines: []DocLine{line}, Body: DocBody{Field: "a", Convert: "markdown-from-adf", Heading: "H", Empty: "E"}},
		}}
		fs := CheckDocument(&Spec{Operations: []Operation{op}})
		if len(fs) != 1 || fs[0].Path != "/operations/0/response/document/lines/0" || fs[0].Rule != ruleDocumentLine {
			t.Fatalf("findings = %+v; want the line-kind finding at the line", fs)
		}
	}
}

// TestDocumentLayoutValidIsTheIssueKeyCheck checks the Go finding for a hand-built line with another valid check: a
// document never carries one, because the schema only loads the issue-key check.
func TestDocumentLayoutValidIsTheIssueKeyCheck(t *testing.T) {
	op := Operation{Access: "read", Response: Response{
		Fields:   map[string]Field{"a": {Pointer: "/a", Convert: "markdown-from-adf"}},
		Document: &Document{Title: "a", Lines: []DocLine{{Label: "A", Field: "a", Valid: "other"}}, Body: DocBody{Field: "a", Convert: "markdown-from-adf", Heading: "H", Empty: "E"}},
	}}
	fs := CheckDocument(&Spec{Operations: []Operation{op}})
	if len(fs) != 1 || fs[0].Path != "/operations/0/response/document/lines/0/valid" || fs[0].Rule != ruleDocumentValid {
		t.Fatalf("findings = %+v; want the valid-check finding at the valid field", fs)
	}
}

// TestDocumentLayoutRefusesARecordField checks the Go finding for a hand-built title on a nested record: the layout
// renders text and string lists only.
func TestDocumentLayoutRefusesARecordField(t *testing.T) {
	op := Operation{Access: "read", Response: Response{
		Fields: map[string]Field{
			"user": {Object: map[string]Field{"name": {Pointer: "/user/name"}}},
			"desc": {Pointer: "/desc", Convert: "markdown-from-adf"},
		},
		Document: &Document{Title: "user", Lines: []DocLine{{Label: "A", Field: "user"}}, Body: DocBody{Field: "desc", Convert: "markdown-from-adf", Heading: "H", Empty: "E"}},
	}}
	fs := CheckDocument(&Spec{Operations: []Operation{op}})
	if len(fs) != 2 {
		t.Fatalf("findings = %+v; want the record finding at the title and the line", fs)
	}
	for _, f := range fs {
		if f.Rule != ruleDocumentRecord {
			t.Fatalf("findings = %+v; want only the record rule", fs)
		}
	}
}

// TestDocumentLayoutJoinOnNonStringsIsRefused checks that a join renders a strings field only.
func TestDocumentLayoutJoinOnNonStringsIsRefused(t *testing.T) {
	doc := mutateAll(docLayoutDoc,
		"- { label: Labels, field: labels, join: \", \", skip_empty: true }",
		"- { label: Labels, field: labels, join: \"; \", skip_empty: true }")
	r := Validate([]byte(doc))
	if r.Valid() {
		t.Fatal("a join with another separator is valid")
	}
	if !hasFinding(r, "/operations/0/response/document/lines/6/join") {
		t.Fatalf("no finding names the line join:\n%s", allText(r))
	}
}

// TestDocumentLayoutDefaultOnPairIsRefused checks that a pair takes no default: only a field or verbatim line does.
func TestDocumentLayoutDefaultOnPairIsRefused(t *testing.T) {
	doc := mutateAll(docLayoutDoc,
		"- { label: Project, pair: { name: projectName, key: key } }",
		"- { label: Project, pair: { name: projectName, key: key }, default: None }")
	r := Validate([]byte(doc))
	if r.Valid() {
		t.Fatal("a default on a pair line is valid")
	}
	if !hasFinding(r, "/operations/0/response/document/lines/1/default") {
		t.Fatalf("no finding names the line default:\n%s", allText(r))
	}
}

// TestDocumentLayoutValidOnPairIsRefused checks that a pair takes no valid check: only a field or verbatim line does.
func TestDocumentLayoutValidOnPairIsRefused(t *testing.T) {
	doc := mutateAll(docLayoutDoc,
		"- { label: Project, pair: { name: projectName, key: key } }",
		"- { label: Project, pair: { name: projectName, key: key }, valid: issue_key }")
	r := Validate([]byte(doc))
	if r.Valid() {
		t.Fatal("a valid check on a pair line is valid")
	}
	if !hasFinding(r, "/operations/0/response/document/lines/1/valid") {
		t.Fatalf("no finding names the line valid check:\n%s", allText(r))
	}
}

// TestDocumentLayoutSkipOnPairIsRefused checks that a pair takes no skip: only a field or verbatim line does.
func TestDocumentLayoutSkipOnPairIsRefused(t *testing.T) {
	doc := mutateAll(docLayoutDoc,
		"- { label: Project, pair: { name: projectName, key: key } }",
		"- { label: Project, pair: { name: projectName, key: key }, skip_empty: true }")
	r := Validate([]byte(doc))
	if r.Valid() {
		t.Fatal("a skip on a pair line is valid")
	}
	if !hasFinding(r, "/operations/0/response/document/lines/1/skip_empty") {
		t.Fatalf("no finding names the line skip:\n%s", allText(r))
	}
}

// TestDocumentLayoutPairKeyNamesNoFieldIsRefused checks that a pair names fields of the operation for its name and
// its key.
func TestDocumentLayoutPairKeyNamesNoFieldIsRefused(t *testing.T) {
	doc := mutateAll(docLayoutDoc, "pair: { name: projectName, key: key }", "pair: { name: projectName, key: nope }")
	r := Validate([]byte(doc))
	if r.Valid() {
		t.Fatal("a pair naming no field is valid")
	}
	if !hasFinding(r, "/operations/0/response/document/lines/1/pair/key") {
		t.Fatalf("no finding names the pair key:\n%s", allText(r))
	}
}

// TestDocumentLayoutBodyNamesNoFieldIsRefused checks that the body names a field of the operation.
func TestDocumentLayoutBodyNamesNoFieldIsRefused(t *testing.T) {
	doc := mutateAll(docLayoutDoc,
		"body: { field: description, convert: markdown-from-adf, heading: Description, empty: \"(no description)\" }",
		"body: { field: nope, convert: markdown-from-adf, heading: Description, empty: \"(no description)\" }")
	r := Validate([]byte(doc))
	if r.Valid() {
		t.Fatal("a document body naming no field is valid")
	}
	if !hasFinding(r, "/operations/0/response/document/body/field") {
		t.Fatalf("no finding names the body field:\n%s", allText(r))
	}
}

// TestDocumentLayoutBodyConverterMismatchIsRefused checks that the body convert matches the convert of its field:
// the body declares how the value was converted.
func TestDocumentLayoutBodyConverterMismatchIsRefused(t *testing.T) {
	doc := mutateAll(docLayoutDoc,
		"        description:\n          pointer: /fields/description\n          convert: markdown-from-adf",
		"        description:\n          pointer: /fields/description\n          convert: jira-title")
	r := Validate([]byte(doc))
	if r.Valid() {
		t.Fatal("a document body with a mismatched converter is valid")
	}
	if !hasFinding(r, "/operations/0/response/document/body/convert") {
		t.Fatalf("no finding names the body converter:\n%s", allText(r))
	}
}

// TestDocumentLayoutTitleOnStringsIsRefused checks that a strings field renders through a join only: the title
// takes no list.
func TestDocumentLayoutTitleOnStringsIsRefused(t *testing.T) {
	doc := mutateAll(docLayoutDoc, "        title: summary", "        title: labels")
	r := Validate([]byte(doc))
	if r.Valid() {
		t.Fatal("a document title on a strings field is valid")
	}
	if !hasFinding(r, "/operations/0/response/document/title") {
		t.Fatalf("no finding names the document title:\n%s", allText(r))
	}
}

// TestDocumentLayoutExpectMayNameMarkdown checks that an example may expect the computed markdown field: the
// layout renders it, so the key is allowed exactly then. The value is a placeholder: a rendered document always
// spans lines, which an example string cannot hold, so the engine tests pin the real text.
func TestDocumentLayoutExpectMayNameMarkdown(t *testing.T) {
	doc := mutateAll(docLayoutDoc,
		"        expect:\n          key: EXAMPLE-12",
		"        expect:\n          key: EXAMPLE-12\n          markdown: rendered")
	requireValid(t, doc)
}

// TestReviewListsDocumentLayouts checks the review block of the reads with a document layout: each names the title
// field, the line count, and the body field with its heading. A spec with no document writes no block, so the
// golden review keeps its text.
func TestReviewListsDocumentLayouts(t *testing.T) {
	review := Review(requireValid(t, docLayoutDoc).Spec)
	for _, line := range []string{
		"Document layouts: 1",
		"1. example-wiki.issues.get: title summary, 10 lines, body description as Description",
	} {
		if !strings.Contains(review, line) {
			t.Fatalf("the review has no line %q:\n%s", line, review)
		}
	}
	if strings.Contains(Review(requireValid(t, exampleWiki).Spec), "Document layouts") {
		t.Fatal("the review of a spec with no document names document layouts")
	}
}
