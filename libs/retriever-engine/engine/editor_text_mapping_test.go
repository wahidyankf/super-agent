package engine_test

import (
	"context"
	_ "embed"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// editorTextYAML is a comment read and a comment write whose text field converts editor data to plain text. It names
// example hosts only.
const editorTextYAML = `version: 1
kind: example-wiki
summary: Reads and writes comments as plain text.
endpoint:
  base_path: /v1
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
operations:
  - id: example-wiki.comments.get
    summary: Read one comment.
    access: read
    method: POST
    read_over_post: true
    path: /comments/info
    params:
      - name: id
        type: ID
        in: body
        required: true
    body:
      id: "{{id}}"
    response:
      fields:
        id:
          pointer: /data/id
          required: true
        text:
          pointer: /data/data
          fallback_pointer: /data/text
          convert: text-from-editor-data
    examples:
      - request:
          id: c1
        response:
          status: 200
          body:
            data:
              id: c1
              text: Nice
        expect:
          id: c1
          text: Nice
  - id: example-wiki.comments.update
    summary: Change one comment.
    access: write
    method: POST
    path: /comments/update
    params:
      - name: id
        type: ID
        in: body
        required: true
      - name: text
        type: Text
        in: body
        required: true
    body:
      id: "{{id}}"
      text: "{{text}}"
    response:
      fields:
        id:
          pointer: /data/id
          required: true
        text:
          pointer: /data/data
          fallback_pointer: /data/text
          convert: text-from-editor-data
    examples:
      - request:
          id: c1
          text: Nice
        response:
          status: 200
          body:
            data:
              id: c1
              text: Nice
        expect:
          id: c1
          text: Nice
`

// TestEditorConverterMapsEditorDataThroughARead replays a comment that carries editor data: the text field reads the
// plain words of the data, and the editor JSON never reaches the record.
func TestEditorConverterMapsEditorDataThroughARead(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, editorTextYAML))
	stub.Script("POST", "/v1/comments/info", page(`{"data":{"id":"c1","data":{"type":"doc","content":[`+
		`{"type":"paragraph","content":[{"type":"text","text":"Hello"}]},`+
		`{"type":"paragraph","content":[{"type":"text","text":"World"}]}]}}}`))
	rec, err := eng.Read(context.Background(), "example-wiki.comments.get", domain.NewArgs(map[string]any{"id": "c1"}))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got := rec.Str("text"); got != "Hello\nWorld" {
		t.Fatalf("text = %q, want the plain words of the editor data", got)
	}
}

// TestEditorConverterKeepsRawTextWithoutEditorData replays a comment that carries no editor data: the fallback
// pointer maps the raw text unchanged, so a comment without editor JSON reads as before.
func TestEditorConverterKeepsRawTextWithoutEditorData(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, editorTextYAML))
	stub.Script("POST", "/v1/comments/info", page(`{"data":{"id":"c1","text":"Nice"}}`))
	rec, err := eng.Read(context.Background(), "example-wiki.comments.get", domain.NewArgs(map[string]any{"id": "c1"}))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got := rec.Str("text"); got != "Nice" {
		t.Fatalf("text = %q, want the raw text", got)
	}
}

// TestEditorConverterIsServedForAWrite checks that a write with the editor-data converter is served, not refused as
// an unserved converter: the converter is a pure function, so it needs no network and no refusal.
func TestEditorConverterIsServedForAWrite(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, editorTextYAML))
	stub.Script("POST", "/v1/comments/update", page(`{"data":{"id":"c1","text":"Nice"}}`))
	rec, err := eng.Write(context.Background(), "example-wiki.comments.update",
		domain.NewArgs(map[string]any{"id": "c1", "text": "Nice"}))
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if got := rec.Str("text"); got != "Nice" {
		t.Fatalf("text = %q, want the raw text", got)
	}
	if n := len(stub.Requests()); n != 1 {
		t.Fatalf("requests = %d, want the one write", n)
	}
}
