package engine_test

import (
	"context"
	"errors"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// firstDoc is a shares read that maps the first item of the list at /data. Pointers of fields read relative to the
// item, so /id names the id of the first share.
const firstDoc = `version: 1
kind: example-wiki
summary: Reads the first share of a list.
endpoint:
  base_path: /v1
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
operations:
  - id: example-wiki.shares.get
    summary: Read the first share.
    access: read
    method: GET
    path: /shares
    response:
      first: /data
      fields:
        id:
          pointer: /id
    examples:
      - request: {}
        response:
          status: 200
          body:
            data:
              - id: share-1
        expect:
          id: share-1
`

// TestFirstTakesIndexZeroOnly plants two shares in the reply and proves the read maps item 0 only: the id of the
// first share, never the second.
func TestFirstTakesIndexZeroOnly(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, firstDoc))
	stub.Script("GET", "/v1/shares", page(`{"data":[{"id":"share-1"},{"id":"share-2"}]}`))
	rec, err := eng.Read(context.Background(), "example-wiki.shares.get", domain.NewArgs(nil))
	if err != nil {
		t.Fatalf("a reply with two items was refused: %v", err)
	}
	if got := rec.Str("id"); got != "share-1" {
		t.Fatalf("id = %q, want the first item share-1", got)
	}
}

// TestFirstOfEmptyListIsNotFound proves the gate of PR 11: an empty list ends with retriever.<kind>.not_found, and no
// partial record comes back with the failure.
func TestFirstOfEmptyListIsNotFound(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, firstDoc))
	stub.Script("GET", "/v1/shares", page(`{"data":[]}`))
	rec, err := eng.Read(context.Background(), "example-wiki.shares.get", domain.NewArgs(nil))
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("an empty list = %v, want ErrNotFound", err)
	}
	if len(rec.Fields()) != 0 || rec.IsList() {
		t.Fatalf("a not_found read returns a partial record: %+v", rec)
	}
}

// TestFirstOfNonListIsUnexpected proves a value that is not a list is an unexpected response: an object and a text
// at the first pointer both fail the read.
func TestFirstOfNonListIsUnexpected(t *testing.T) {
	for _, body := range []string{`{"data":{"id":"share-1"}}`, `{"data":"share-1"}`} {
		eng, stub, _ := newStubEngine(t, loadSpecFrom(t, firstDoc))
		stub.Script("GET", "/v1/shares", page(body))
		if _, err := eng.Read(context.Background(), "example-wiki.shares.get", domain.NewArgs(nil)); !errors.Is(err, domain.ErrUnexpectedResponse) {
			t.Fatalf("first of %s = %v, want ErrUnexpectedResponse", body, err)
		}
	}
}
