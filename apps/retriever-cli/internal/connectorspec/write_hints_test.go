package connectorspec

import (
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/operations"
)

// postReadDoc is a spec of kind example-wiki with one read over POST, which carries the E1 marker, and one GET read.
const postReadDoc = wikiHeader + `  - id: example-wiki.docs.get
    summary: Read one document.
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
    examples:
      - request:
          id: doc-1
        response:
          status: 200
          body:
            title: First document
        expect:
          title: First document
  - id: example-wiki.docs.find
    summary: Find documents in a collection.
    access: read
    method: POST
    read_over_post: true
    path: /docs/find
    params:
      - name: collection_id
        type: ID
        in: body
        required: false
    body:
      collectionId: "{{collection_id}}"
    response:
      items: /items
      fields:
        id:
          pointer: /id
    examples:
      - request:
          collection_id: col-1
        response:
          status: 200
          body:
            items:
              - id: doc-1
        expect:
          id: doc-1
`

// rowNamed returns the catalog row of an operation of the set, and fails the test when the set has none.
func rowNamed(t *testing.T, set Set, id string) operations.Row {
	t.Helper()
	row, ok := set.Operations.Lookup(domain.OperationID(id))
	if !ok {
		t.Fatalf("the catalog has no row %s", id)
	}
	return row
}

// TestAPostReadKeepsItsRoutingAndTakesWriteHintsUntilItsSourcesPin checks the hint rule of ADR 002, E1, at the catalog. A
// POST read routes as a read, because the engine serves it as one, but its row asks for the hints of a write until every
// source of its kind pins the spec. A GET read never asks for them.
func TestAPostReadKeepsItsRoutingAndTakesWriteHintsUntilItsSourcesPin(t *testing.T) {
	set := mustLoad(t, specFile("example-wiki.yml", postReadDoc))
	post := rowNamed(t, set, "example-wiki.docs.find")
	if post.Effect != domain.EffectRead {
		t.Fatalf("the POST read routes as %v; want a read", post.Effect)
	}
	if !post.WriteHints {
		t.Fatal("an unpinned POST read asks for read-only hints")
	}
	if get := rowNamed(t, set, "example-wiki.docs.get"); get.WriteHints {
		t.Fatal("a GET read asks for write hints")
	}

	pinned := set.WithPinned(map[string]bool{"example-wiki": true})
	if rowNamed(t, pinned, "example-wiki.docs.find").WriteHints {
		t.Fatal("a POST read of a pinned spec still asks for write hints")
	}
	if got := rowNamed(t, pinned, "example-wiki.docs.find"); got.Effect != domain.EffectRead {
		t.Fatalf("pinning changed the routing of the POST read to %v", got.Effect)
	}
	if got := rowNamed(t, pinned, "example-wiki.docs.find"); got.Tool != "example_wiki_docs_find" {
		t.Fatalf("pinning changed the tool name to %s", got.Tool)
	}
	found := false
	for _, row := range pinned.CLIRows {
		if row.ID == "example-wiki.docs.find" {
			found = true
			if row.WriteHints {
				t.Fatal("the command-line row of a pinned POST read still asks for write hints")
			}
		}
	}
	if !found {
		t.Fatal("the command-line rows lost the POST read")
	}

	unpinned := set.WithPinned(map[string]bool{"example-wiki": false})
	if !rowNamed(t, unpinned, "example-wiki.docs.find").WriteHints {
		t.Fatal("an unpinned kind lost its write hints")
	}
}

// TestBuiltInRowsNeverAskForWriteHints checks that the built-in catalog keeps its rows as they were: none of them asks for
// write hints, whatever a pin says.
func TestBuiltInRowsNeverAskForWriteHints(t *testing.T) {
	set := mustLoad(t, specFile("example-wiki.yml", postReadDoc)).WithPinned(map[string]bool{"example-wiki": true})
	for _, row := range set.Operations.Rows() {
		if row.WriteHints && !strings.HasPrefix(string(row.ID), "example-wiki.") {
			t.Fatalf("built-in row %s asks for write hints", row.ID)
		}
	}
	if _, ok := set.Operations.Lookup(domain.OpCollectionsList); !ok {
		t.Fatal("the built-in catalog lost its collections list")
	}
}

// TestPinningAKindThatNamesABuiltInPrefixKeepsTheBuiltInRows checks that a spec kind which shares its first word with a
// built-in operation, such as collections for collections.list, neither copies that built-in row nor breaks the catalog.
// Only the spec's own operations are re-tagged.
func TestPinningAKindThatNamesABuiltInPrefixKeepsTheBuiltInRows(t *testing.T) {
	doc := strings.ReplaceAll(postReadDoc, "example-wiki", "collections")
	set := mustLoad(t, specFile("collections.yml", doc))
	pinned := set.WithPinned(map[string]bool{"collections": true})
	if rowNamed(t, pinned, "collections.docs.find").WriteHints {
		t.Fatal("the pinned POST read of the spec kept its write hints")
	}
	if _, ok := pinned.Operations.Lookup(domain.OpCollectionsList); !ok {
		t.Fatal("the built-in collections list is missing after the pin")
	}
	count := 0
	for _, row := range pinned.Operations.Rows() {
		if row.ID == domain.OpCollectionsList {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("the built-in collections list appears %d times; want once", count)
	}
}
