package engine_test

import (
	"context"
	"errors"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// mergeDoc is a mention search that merges users and documents with constant labels. Each list maps its own id field,
// and the label field kind is a constant of the list, never a pointer into the reply.
const mergeDoc = `version: 1
kind: example-wiki
summary: Search mentions across users and documents.
endpoint:
  base_path: /v1
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
operations:
  - id: example-wiki.mention.search
    summary: Search mentions across users and documents.
    access: read
    method: GET
    path: /mention
    response:
      merge:
        label_field: kind
        lists:
          - items: /data/users
            label: user
            fields:
              id:
                pointer: /id
          - items: /data/documents
            label: document
            fields:
              id:
                pointer: /id
    examples:
      - request: {}
        response:
          status: 200
          body:
            data:
              users:
                - id: u-1
              documents:
                - id: d-1
        expect:
          id: u-1
          kind: user
`

// mergeBeatsDoc is a merge whose lists also map a kind field from the reply. The constant label of each list beats
// the mapped reply field of the same name.
const mergeBeatsDoc = `version: 1
kind: example-wiki
summary: Search mentions across users and documents.
endpoint:
  base_path: /v1
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
operations:
  - id: example-wiki.mention.search
    summary: Search mentions across users and documents.
    access: read
    method: GET
    path: /mention
    response:
      merge:
        label_field: kind
        lists:
          - items: /data/users
            label: user
            fields:
              id:
                pointer: /id
              kind:
                pointer: /type
          - items: /data/documents
            label: document
            fields:
              id:
                pointer: /id
              kind:
                pointer: /type
    examples:
      - request: {}
        response:
          status: 200
          body:
            data:
              users:
                - id: u-1
              documents:
                - id: d-1
        expect:
          id: u-1
          kind: user
`

// TestMergeKeepsDeclaredOrderAndLabels plants two users and one document and proves the read returns them in
// declared list order and then in reply order, each tagged with the constant label of its own list.
func TestMergeKeepsDeclaredOrderAndLabels(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, mergeDoc))
	stub.Script("GET", "/v1/mention", page(`{"data":{"users":[{"id":"u-1"},{"id":"u-2"}],"documents":[{"id":"d-1"}]}}`))
	rec, err := eng.Read(context.Background(), "example-wiki.mention.search", domain.NewArgs(nil))
	if err != nil {
		t.Fatalf("a merge of two lists was refused: %v", err)
	}
	if !rec.IsList() {
		t.Fatal("a merge read returned no list")
	}
	items := rec.Items()
	if len(items) != 3 {
		t.Fatalf("got %d items, want 3 in declared list order and reply order", len(items))
	}
	want := []struct{ id, kind string }{
		{"u-1", "user"},
		{"u-2", "user"},
		{"d-1", "document"},
	}
	for i, w := range want {
		if got := items[i].Str("id"); got != w.id {
			t.Fatalf("item %d id = %q, want %q", i, got, w.id)
		}
		if got := items[i].Str("kind"); got != w.kind {
			t.Fatalf("item %d kind = %q, want the constant label %q", i, got, w.kind)
		}
	}
}

// mergeConvertDoc is a merge whose users list needs a Markdown converter, which the engine does not serve yet.
const mergeConvertDoc = `version: 1
kind: example-wiki
summary: Search mentions across users and documents.
endpoint:
  base_path: /v1
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
operations:
  - id: example-wiki.mention.search
    summary: Search mentions across users and documents.
    access: read
    method: GET
    path: /mention
    response:
      merge:
        label_field: kind
        lists:
          - items: /data/users
            label: user
            fields:
              id:
                pointer: /id
                convert: markdown-from-html
          - items: /data/documents
            label: document
            fields:
              id:
                pointer: /id
    examples:
      - request: {}
        response:
          status: 200
          body:
            data:
              users:
                - id: u-1
              documents:
                - id: d-1
        expect:
          id: u-1
          kind: user
`

// TestMergeConverterIsUnsupportedBeforeAnyRequest checks that a merge list needing a converter of a later pull
// request fails closed, and sends nothing.
func TestMergeConverterIsUnsupportedBeforeAnyRequest(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, mergeConvertDoc))
	if _, err := eng.Read(context.Background(), "example-wiki.mention.search", domain.NewArgs(nil)); !errors.Is(err, domain.ErrUnsupportedOperation) {
		t.Fatalf("err = %v, want ErrUnsupportedOperation", err)
	}
	if n := len(stub.Requests()); n != 0 {
		t.Fatalf("an unsupported converter sent %d requests", n)
	}
}

// TestMergeNeedsListsOfObjects checks the list shapes that a merge maps: a list pointer that names no list, and an
// item that is not an object, are both an unexpected response.
func TestMergeNeedsListsOfObjects(t *testing.T) {
	for _, body := range []string{
		`{"data":{"users":{"id":"u-1"},"documents":[]}}`,
		`{"data":{"users":["u-1"],"documents":[]}}`,
	} {
		eng, stub, _ := newStubEngine(t, loadSpecFrom(t, mergeDoc))
		stub.Script("GET", "/v1/mention", page(body))
		if _, err := eng.Read(context.Background(), "example-wiki.mention.search", domain.NewArgs(nil)); !errors.Is(err, domain.ErrUnexpectedResponse) {
			t.Fatalf("merge of %s = %v, want ErrUnexpectedResponse", body, err)
		}
	}
}

// TestMergeLabelBeatsReplyField proves the gate of PR 12: each reply carries its own kind field, and the constant
// label of the list still wins on every merged record.
func TestMergeLabelBeatsReplyField(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, mergeBeatsDoc))
	stub.Script("GET", "/v1/mention", page(`{"data":{"users":[{"id":"u-1","type":"bogus-user"}],"documents":[{"id":"d-1","type":"bogus-document"}]}}`))
	rec, err := eng.Read(context.Background(), "example-wiki.mention.search", domain.NewArgs(nil))
	if err != nil {
		t.Fatalf("a merge whose reply names the label field was refused: %v", err)
	}
	items := rec.Items()
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
	if got := items[0].Str("kind"); got != "user" {
		t.Fatalf("user kind = %q, want the constant label user, not the reply field", got)
	}
	if got := items[1].Str("kind"); got != "document" {
		t.Fatalf("document kind = %q, want the constant label document, not the reply field", got)
	}
	if got := items[0].Str("id"); got != "u-1" {
		t.Fatalf("user id = %q, want u-1", got)
	}
}
