package engine_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// adfYAML is a read and a write whose replies hold ADF documents, mapped with
// the markdown-from-adf converter to Markdown. It names example hosts only.
const adfYAML = `version: 1
kind: example-wiki
summary: Reads with ADF-converted body fields of an example wiki.
endpoint:
  base_path: /v1
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
operations:
  - id: example-wiki.docs.adf
    summary: Get one document with its ADF body as Markdown.
    access: read
    method: GET
    path: /docs/{id}
    params:
      - name: id
        type: ID
        in: path
        required: true
    response:
      fields:
        title:
          pointer: /title
        body:
          pointer: /body
          convert: markdown-from-adf
        meta:
          object:
            notes:
              pointer: /meta/notes
              convert: markdown-from-adf
      markdown: body
      errors:
        "404": not_found
    examples:
      - request:
          id: doc-1
        response:
          status: 200
          body:
            title: First document
            body:
              type: doc
              content:
                - type: paragraph
                  content:
                    - type: text
                      text: Hello
        expect:
          title: First document
  - id: example-wiki.docs.adfrevise
    summary: Revise one document, reading its ADF body as Markdown.
    access: write
    method: POST
    path: /docs/{id}/revise
    params:
      - name: id
        type: ID
        in: path
        required: true
      - name: title
        type: Name
        in: body
        required: true
    response:
      fields:
        id:
          pointer: /id
        body:
          pointer: /body
          convert: markdown-from-adf
    examples:
      - request:
          id: doc-1
          title: Second title
        response:
          status: 200
          body:
            id: doc-1
            body:
              type: doc
              content:
                - type: paragraph
                  content:
                    - type: text
                      text: Revised
        expect:
          id: doc-1
`

// adfDocBody returns a reply whose ADF fields hold one paragraph each.
func adfDocBody(body, notes string) string {
	para := func(text string) string {
		return `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":` + fmt.Sprintf("%q", text) + `}]}]}`
	}
	return `{"title":"First document","body":` + para(body) + `,"meta":{"notes":` + para(notes) + `}}`
}

// TestAnADFReadMapsTheDocumentToMarkdown checks that a read naming
// markdown-from-adf is served: the reply ADF maps to Markdown, in a plain
// field and in a nested record, and the request is sent.
func TestAnADFReadMapsTheDocumentToMarkdown(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, adfYAML))
	stub.Script("GET", "/v1/docs/doc-1", page(adfDocBody("Hello", "Hi")))
	rec, err := eng.Read(context.Background(), "example-wiki.docs.adf", domain.NewArgs(map[string]any{"id": "doc-1"}))
	if err != nil {
		t.Fatalf("Read() error = %v, want nil", err)
	}
	if got := rec.Str("body"); got != "Hello" {
		t.Fatalf("body = %q, want %q", got, "Hello")
	}
	meta, ok := rec.Get("meta")
	if !ok {
		t.Fatal("meta is missing")
	}
	metaRec, ok := meta.(domain.Record)
	if !ok {
		t.Fatalf("meta = %#v, want a record", meta)
	}
	if got := metaRec.Str("notes"); got != "Hi" {
		t.Fatalf("meta.notes = %q, want %q", got, "Hi")
	}
	if n := len(stub.Requests()); n != 1 {
		t.Fatalf("a served converter sent %d requests, want 1", n)
	}
}

// TestAnADFWriteMapsTheDocumentToMarkdown checks that a write naming
// markdown-from-adf is served too: the reply ADF maps to Markdown.
func TestAnADFWriteMapsTheDocumentToMarkdown(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, adfYAML))
	stub.Script("POST", "/v1/docs/doc-1/revise", page(`{"id":"doc-1","body":{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"Revised"}]}]}}`))
	rec, err := eng.Write(context.Background(), "example-wiki.docs.adfrevise", domain.NewArgs(map[string]any{"id": "doc-1", "title": "Second title"}))
	if err != nil {
		t.Fatalf("Write() error = %v, want nil", err)
	}
	if got := rec.Str("body"); got != "Revised" {
		t.Fatalf("body = %q, want %q", got, "Revised")
	}
}

// TestAnADFMissingFieldMapsToNull checks that a reply without the ADF value
// maps the optional field to null, like any other missing optional field.
func TestAnADFMissingFieldMapsToNull(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, adfYAML))
	stub.Script("GET", "/v1/docs/doc-1", page(`{"title":"First document"}`))
	rec, err := eng.Read(context.Background(), "example-wiki.docs.adf", domain.NewArgs(map[string]any{"id": "doc-1"}))
	if err != nil {
		t.Fatalf("Read() error = %v, want nil", err)
	}
	if got, _ := rec.Get("body"); got != nil {
		t.Fatalf("body = %#v, want nil", got)
	}
}

// TestAnADFNonDocumentFailsUnexpected checks that a reply value that is not
// an ADF document fails the read, naming no value.
func TestAnADFNonDocumentFailsUnexpected(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, adfYAML))
	stub.Script("GET", "/v1/docs/doc-1", page(`{"title":"First document","body":7}`))
	_, err := eng.Read(context.Background(), "example-wiki.docs.adf", domain.NewArgs(map[string]any{"id": "doc-1"}))
	if !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
	}
}
