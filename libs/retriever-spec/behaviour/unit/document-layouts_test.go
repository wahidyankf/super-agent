package unit_test

import (
	"fmt"
	"strings"
	"testing"
)

// docLayoutOp is a single-object read at index zero whose document layout has ten metadata lines. The first nine are
// field lines, and the tenth is a verbatim line. Its body converts with markdown-from-adf, as its field does.
func docLayoutOp() string {
	var b strings.Builder
	b.WriteString(`  - id: example-wiki.docs.report
    summary: Render one document report.
    access: read
    method: GET
    path: /docs/{id}/report
    params:
      - name: id
        type: ID
        in: path
        required: true
    response:
      fields:
        title:
          pointer: /title
        key:
          pointer: /key
        body:
          pointer: /body
          convert: markdown-from-adf
`)
	for i := 0; i < 9; i++ {
		fmt.Fprintf(&b, "        f%d:\n          pointer: /f%d\n", i, i)
	}
	b.WriteString("      document:\n        title: title\n        lines:\n")
	for i := 0; i < 9; i++ {
		fmt.Fprintf(&b, "          - label: Field %d\n            field: f%d\n", i, i)
	}
	b.WriteString(`          - label: Key
            verbatim: key
        body:
          field: body
          convert: markdown-from-adf
          heading: Description
          empty: (no description)
    examples:
      - request:
          id: doc-1
        response:
          status: 200
          body:
            title: Report
            key: EX-1
            body: ""
        expect: {}
`)
	return b.String()
}

// @scenario specs/libs/retriever-spec/behaviours/document-layouts.feature: A single-object read with a title, metadata lines, and a description body is accepted
func TestSingleObjectReadWithTitleMetadataLinesAndBodyIsAccepted(t *testing.T) {
	requireAccepted(t, oneOp(docLayoutOp()))
}

// @scenario specs/libs/retriever-spec/behaviours/document-layouts.feature: A document layout on a list read is refused at the document
func TestDocumentLayoutOnListReadIsRefusedAtTheDocument(t *testing.T) {
	doc := mutate(t, oneOp(docLayoutOp()), "    response:\n      fields:\n", "    response:\n      items: /items\n      fields:\n")
	requireRefusedAt(t, doc, "/operations/0/response/document")
}

// @scenario specs/libs/retriever-spec/behaviours/document-layouts.feature: A document layout that sits beside a markdown output field is refused at the document
func TestDocumentLayoutBesideMarkdownOutputFieldIsRefusedAtTheDocument(t *testing.T) {
	doc := mutate(t, oneOp(docLayoutOp()), "      document:\n", "      markdown: body\n      document:\n")
	requireRefusedAt(t, doc, "/operations/0/response/document")
}

// @scenario specs/libs/retriever-spec/behaviours/document-layouts.feature: A metadata line whose label is outside the closed form is refused at its label
func TestMetadataLineWhoseLabelIsOutsideClosedFormIsRefusedAtItsLabel(t *testing.T) {
	doc := mutate(t, oneOp(docLayoutOp()), "          - label: Field 0\n",
		"          - label: "+strings.Repeat("L", 33)+"\n")
	requireRefusedAt(t, doc, "/operations/0/response/document/lines/0/label")
}

// @scenario specs/libs/retriever-spec/behaviours/document-layouts.feature: A join on a verbatim metadata line is refused at the join
func TestJoinOnVerbatimMetadataLineIsRefusedAtTheJoin(t *testing.T) {
	doc := mutate(t, oneOp(docLayoutOp()), "          - label: Key\n            verbatim: key\n",
		"          - label: Key\n            verbatim: key\n            join: \", \"\n")
	requireRefusedAt(t, doc, "/operations/0/response/document/lines/9/join")
}

// @scenario specs/libs/retriever-spec/behaviours/document-layouts.feature: A description body whose converter does not match its field is refused at the converter
func TestDescriptionBodyWhoseConverterDoesNotMatchItsFieldIsRefusedAtTheConverter(t *testing.T) {
	// The body names the converter of its field. Here the body names another converter than the field uses.
	doc := mutate(t, oneOp(docLayoutOp()), "        body:\n          field: body\n          convert: markdown-from-adf\n",
		"        body:\n          field: body\n          convert: markdown-from-html\n")
	requireRefusedAt(t, doc, "/operations/0/response/document/body/convert")
}

// @scenario specs/libs/retriever-spec/behaviours/document-layouts.feature: A document layout whose line names a field the operation does not declare is refused
func TestDocumentLayoutWhoseLineNamesUndeclaredFieldIsRefused(t *testing.T) {
	doc := mutate(t, oneOp(docLayoutOp()), "          - label: Field 0\n            field: f0\n",
		"          - label: Field 0\n            field: missing\n")
	r := requireRefused(t, doc)
	if !hasFindingUnder(r, "/operations/0/response/document") {
		t.Fatalf("no finding under the document of the first operation:\n%s", text(r))
	}
}
