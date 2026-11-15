package engine_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine/replay"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// treeDoc is a documents read whose response flattens the tree at /data depth first, computing the parent and the
// depth of each node. The id is required, because the parent of a node is the id of its parent node; the top nodes
// take their parent from the document_id parameter. It names example hosts only. The max_depth is left unset, so
// the read keeps the default of 64 levels.
const treeDoc = `version: 1
kind: example-wiki
summary: Read the documents of a collection as a tree.
endpoint:
  base_path: /v1
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
operations:
  - id: example-wiki.docs.tree
    summary: Read the documents of a collection as a tree.
    access: read
    method: GET
    path: /tree
    params:
      - name: document_id
        type: ID
        in: query
        required: false
    response:
      items: /data
      fields:
        id:
          pointer: /id
          required: true
        title:
          pointer: /title
      tree:
        children: /children
        parent_field: parent_id
        depth_field: depth
        root_parent_param: document_id
    examples:
      - request: {}
        response:
          status: 200
          body:
            data:
              - id: doc-1
                title: Root
                children:
                  - id: doc-2
                    title: Child
        expect:
          id: doc-1
          title: Root
          parent_id: null
          depth: 0
`

// plainDoc is a documents read with no tree block, for the rows that prove only a tree reply gets the higher decode
// cap. It names example hosts only.
const plainDoc = `version: 1
kind: example-wiki
summary: List documents.
endpoint:
  base_path: /v1
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
operations:
  - id: example-wiki.docs.flat
    summary: List documents.
    access: read
    method: GET
    path: /flat
    response:
      items: /data
      fields:
        id:
          pointer: /id
        title:
          pointer: /title
    examples:
      - request: {}
        response:
          status: 200
          body:
            data:
              - id: doc-1
                title: First document
        expect:
          id: doc-1
          title: First document
`

// buildChain returns a reply whose data holds one chain of n nodes, each with one child, so the tree is n levels
// deep. The leaf carries no children key, like a real reply. Each tree level takes two JSON levels, an array and an
// object, so n nodes nest 3+2*(n-1) levels deep under the data envelope.
func buildChain(n int) string {
	var b strings.Builder
	b.WriteString(`{"data":[`)
	for k := 0; k < n; k++ {
		b.WriteString(`{"id":"n-`)
		b.WriteString(itoa(k))
		b.WriteString(`","title":"T `)
		b.WriteString(itoa(k))
		b.WriteString(`"`)
		if k == n-1 {
			b.WriteString(`}`)
		} else {
			b.WriteString(`,"children":[`)
		}
	}
	for k := 0; k < n-1; k++ {
		b.WriteString(`]}`)
	}
	b.WriteString(`]}`)
	return b.String()
}

// itoa returns the decimal text of a small non-negative number, without importing a converter for a test helper.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [8]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// deepArrays returns a reply whose data holds one object wrapped in n arrays, so a reply without a tree nests past
// the S9 cap of 64 levels.
func deepArrays(n int) string {
	var b strings.Builder
	b.WriteString(`{"data":`)
	for k := 0; k < n; k++ {
		b.WriteString(`[`)
	}
	b.WriteString(`{"id":"x","title":"X"}`)
	for k := 0; k < n; k++ {
		b.WriteString(`]`)
	}
	b.WriteString(`}`)
	return b.String()
}

// treeItems reads the tree operation of the doc with the given arguments, and fails the test on any refusal. The
// query names the arguments the read sends, because the stub matches a request by its method and target.
func treeItems(t *testing.T, doc string, args map[string]any, query, body string) []domain.Record {
	t.Helper()
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, doc))
	target := "/v1/tree"
	if query != "" {
		target += "?" + query
	}
	stub.Script("GET", target, page(body))
	rec, err := eng.Read(context.Background(), "example-wiki.docs.tree", domain.NewArgs(args))
	if err != nil {
		t.Fatalf("a tree read was refused: %v", err)
	}
	if !rec.IsList() {
		t.Fatal("a tree read returned no list")
	}
	return rec.Items()
}

// TestTreeFlattensDepthFirstWithParentAndDepth plants two roots, one with a child and a grandchild, and proves the
// read returns them depth first with the id of each parent node and the depth from 0. The top nodes take their
// parent from the document_id parameter, and a read without it leaves them null.
func TestTreeFlattensDepthFirstWithParentAndDepth(t *testing.T) {
	const body = `{"data":[{"id":"a","title":"A","children":[{"id":"a1","title":"A1","children":[{"id":"a11","title":"A11"}]},{"id":"a2","title":"A2"}]},{"id":"b","title":"B"}]}`
	items := treeItems(t, treeDoc, map[string]any{"document_id": "doc-0"}, "document_id=doc-0", body)
	want := []struct {
		id     string
		parent any
		depth  int
	}{
		{"a", "doc-0", 0},
		{"a1", "a", 1},
		{"a11", "a1", 2},
		{"a2", "a", 1},
		{"b", "doc-0", 0},
	}
	if len(items) != len(want) {
		t.Fatalf("got %d items, want %d depth first", len(items), len(want))
	}
	for i, w := range want {
		if got := items[i].Str("id"); got != w.id {
			t.Fatalf("item %d id = %q, want %q", i, got, w.id)
		}
		got, _ := items[i].Get("parent_id")
		if got != w.parent {
			t.Fatalf("item %d parentId = %v, want %v", i, got, w.parent)
		}
		got, _ = items[i].Get("depth")
		if got != w.depth {
			t.Fatalf("item %d depth = %v, want %d", i, got, w.depth)
		}
	}
	top := treeItems(t, treeDoc, nil, "", body)
	for _, i := range []int{0, 4} {
		if got, _ := top[i].Get("parent_id"); got != nil {
			t.Fatalf("top item %d parentId = %v, want null without the root parent parameter", i, got)
		}
	}
}

// TestTreeDropsNodesPastMaxDepth nests a tree to 70 levels and proves the read keeps 65 levels of nodes, depths 0
// to 64, dropping each deeper node with its subtree. A max_depth of 1 keeps the root and its children only.
func TestTreeDropsNodesPastMaxDepth(t *testing.T) {
	items := treeItems(t, treeDoc, nil, "", buildChain(70))
	if len(items) != 65 {
		t.Fatalf("got %d items, want 65 levels of nodes from a tree nested to 70 levels", len(items))
	}
	for i, rec := range items {
		if got := rec.Str("id"); got != "n-"+itoa(i) {
			t.Fatalf("item %d id = %q, want %q depth first", i, got, "n-"+itoa(i))
		}
		if got, _ := rec.Get("depth"); got != i {
			t.Fatalf("item %d depth = %v, want %d", i, got, i)
		}
	}
	shallow := strings.Replace(treeDoc, "        root_parent_param: document_id\n",
		"        root_parent_param: document_id\n        max_depth: 1\n", 1)
	kept := treeItems(t, shallow, nil, "", buildChain(4))
	if len(kept) != 2 {
		t.Fatalf("got %d items, want 2 with a max_depth of 1", len(kept))
	}
}

// TestOnlyTreeRepliesGetTheHigherDecodeCap proves the decode caps: a tree reply nested past 64 JSON levels still
// decodes, while any other reply nested past 64 levels fails with an unexpected response. A tree reply nested past
// the fixed higher cap fails too, so the higher cap is fixed and not open-ended.
func TestOnlyTreeRepliesGetTheHigherDecodeCap(t *testing.T) {
	deep := treeItems(t, treeDoc, nil, "", buildChain(68))
	if len(deep) != 65 {
		t.Fatalf("got %d items, want 65 from a tree nested past 64 JSON levels", len(deep))
	}
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, plainDoc))
	stub.Script("GET", "/v1/flat", page(deepArrays(70)))
	if _, err := eng.Read(context.Background(), "example-wiki.docs.flat", domain.NewArgs(nil)); !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("a reply without a tree nested past 64 levels = %v, want ErrUnexpectedResponse", err)
	}
	eng, stub, _ = newStubEngine(t, loadSpecFrom(t, treeDoc))
	stub.Script("GET", "/v1/tree", page(buildChain(75)))
	if _, err := eng.Read(context.Background(), "example-wiki.docs.tree", domain.NewArgs(nil)); !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("a tree reply nested past the higher cap = %v, want ErrUnexpectedResponse", err)
	}
}

// TestTreeRefusesAModelWithoutRequiredId checks that the constructor closes the hand-built gap: a tree model without
// a required id field is refused before any request is sent.
func TestTreeRefusesAModelWithoutRequiredId(t *testing.T) {
	s := loadSpecFrom(t, treeDoc)
	stub, err := replay.New(s)
	if err != nil {
		t.Fatalf("build the replay stub: %v", err)
	}
	s.Operations[0].Response.Fields["id"] = spec.Field{Pointer: "/id"}
	if _, err := engine.New(s, testSource(t), stubPorts(stub, &fakeClock{})); err == nil {
		t.Fatal("a tree model without a required id was accepted")
	}
}

// pagedTreeDoc is a tree read with offset paging in the query. Each page holds its own tree, so each page flattens
// depth first with depths from 0. It names example hosts only.
const pagedTreeDoc = `version: 1
kind: example-wiki
summary: Read one page of a document tree.
endpoint:
  base_path: /v1
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
operations:
  - id: example-wiki.docs.treepage
    summary: Read one page of a document tree.
    access: read
    method: GET
    path: /treepage
    params:
      - name: limit
        type: Limit
        in: query
        required: false
        default: 2
      - name: offset
        type: Offset
        in: query
        required: false
        default: 0
    pagination:
      style: offset
      max_pages: 20
      limit_param: limit
      offset_param: offset
    response:
      items: /data
      fields:
        id:
          pointer: /id
          required: true
        title:
          pointer: /title
      tree:
        children: /children
        parent_field: parent_id
        depth_field: depth
    examples:
      - request:
          limit: 2
          offset: 0
        response:
          status: 200
          body:
            data:
              - id: doc-1
                title: First document
        expect:
          id: doc-1
          title: First document
          depth: 0
`

// TestTreePagesFlattenPerPage proves a paged tree read flattens each page depth first, with depths from 0 on every
// page. The first page holds two roots with one child, and the short second page stops the read.
func TestTreePagesFlattenPerPage(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, pagedTreeDoc))
	stub.Script("GET", "/v1/treepage?limit=2&offset=0",
		page(`{"data":[{"id":"a","title":"A","children":[{"id":"a1","title":"A1"}]},{"id":"b","title":"B"}]}`))
	stub.Script("GET", "/v1/treepage?limit=2&offset=3", page(`{"data":[{"id":"c","title":"C"}]}`))
	rec, err := eng.Read(context.Background(), "example-wiki.docs.treepage", domain.NewArgs(nil))
	if err != nil {
		t.Fatalf("a paged tree read was refused: %v", err)
	}
	items := rec.Items()
	want := []struct {
		id     string
		parent any
		depth  int
	}{
		{"a", nil, 0},
		{"a1", "a", 1},
		{"b", nil, 0},
		{"c", nil, 0},
	}
	if len(items) != len(want) {
		t.Fatalf("got %d items, want %d across two pages", len(items), len(want))
	}
	for i, w := range want {
		if got := items[i].Str("id"); got != w.id {
			t.Fatalf("item %d id = %q, want %q", i, got, w.id)
		}
		if got, _ := items[i].Get("parent_id"); got != w.parent {
			t.Fatalf("item %d parent_id = %v, want %v", i, got, w.parent)
		}
		if got, _ := items[i].Get("depth"); got != w.depth {
			t.Fatalf("item %d depth = %v, want %d", i, got, w.depth)
		}
	}
}
