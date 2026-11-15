package engine_test

import (
	"context"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// mergePagedDoc is a mention search that merges users and documents with
// constant labels, one page at a time. It mirrors the bundled suggestions
// endpoint: a merged list read with offset paging.
const mergePagedDoc = `version: 1
kind: example-wiki
summary: Search mentions across users and documents, one page at a time.
endpoint:
  base_path: /v1
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
operations:
  - id: example-wiki.mention.search
    summary: Search mentions across users and documents, one page at a time.
    access: read
    method: GET
    path: /mention
    params:
      - name: limit
        type: Limit
        in: query
        required: false
        default: 20
      - name: offset
        type: Offset
        in: query
        required: false
        default: 0
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
    pagination:
      style: offset
      max_pages: 1
      limit_param: limit
      offset_param: offset
    examples:
      - request: {}
        response:
          status: 200
          body:
            data:
              users:
                - id: u-1
              documents: []
        expect:
          id: u-1
          kind: user
`

// TestPagedMergeMergesEachPage checks that a merged list read with paging
// maps each page with the merge lists, in declared list order and reply order.
func TestPagedMergeMergesEachPage(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, mergePagedDoc))
	stub.Script("GET", "/v1/mention?limit=20&offset=0", page(`{"data":{"users":[{"id":"u-1"}],"documents":[{"id":"d-1"}]}}`))
	rec, err := eng.Read(context.Background(), "example-wiki.mention.search", domain.NewArgs(nil))
	if err != nil {
		t.Fatalf("a paged merge was refused: %v", err)
	}
	items := rec.Items()
	if len(items) != 2 {
		t.Fatalf("got %d items, want the user and the document in list order", len(items))
	}
	if got := items[0].Str("id"); got != "u-1" {
		t.Fatalf("item 0 id = %q, want u-1", got)
	}
	if got := items[0].Str("kind"); got != "user" {
		t.Fatalf("item 0 kind = %q, want user", got)
	}
	if got := items[1].Str("id"); got != "d-1" {
		t.Fatalf("item 1 id = %q, want d-1", got)
	}
	if got := items[1].Str("kind"); got != "document" {
		t.Fatalf("item 1 kind = %q, want document", got)
	}
}
