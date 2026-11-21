package unit_test

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// wireMergeOps is a mention search that merges three lists. Two lists carry the user label, and the reply holds
// them in a different order from the declared one, so a binding can see the declared order win.
const wireMergeOps = `  - id: example-wiki.mention.search
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
          - items: /data/guests
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
              guests: []
              documents: []
        expect:
          id: u-1
          kind: user
`

// wireMergeKindOps is a mention search whose reply holds its own kind field on each record. The constant label of each
// list must win over that field.
const wireMergeKindOps = `  - id: example-wiki.mention.search
    summary: Search mentions across users and documents, with the kind field of the reply.
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
                pointer: /kind
          - items: /data/documents
            label: document
            fields:
              id:
                pointer: /id
              kind:
                pointer: /kind
    examples:
      - request: {}
        response:
          status: 200
          body:
            data:
              users:
                - id: u-1
                  kind: other
              documents: []
        expect:
          id: u-1
          kind: user
`

// wireMergeConvertOps is a mention search whose user list names a converter the engine does not serve.
const wireMergeConvertOps = `  - id: example-wiki.mention.search
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
              documents: []
        expect:
          id: u-1
          kind: user
`

// wireMergePagedOps is a paged mention search that merges a user list and a document list, two items to a page.
const wireMergePagedOps = `  - id: example-wiki.mention.search
    summary: Search mentions across users and documents, one page at a time.
    access: read
    method: GET
    path: /mention
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
      max_pages: 5
      limit_param: limit
      offset_param: offset
    examples:
      - request:
          limit: 2
          offset: 0
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

// wireTreeDoc is a document tree read that reads its roots from the items and its children from the children field.
const wireTreeDoc = `version: 1
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

// wirePagedTreeOps is a tree read with offset paging, two roots to a page.
const wirePagedTreeOps = `  - id: example-wiki.docs.treepage
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

// wireFlatOps is a documents read with no tree block, which takes the ordinary decode cap.
const wireFlatOps = `  - id: example-wiki.docs.flat
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

// wireChain returns a reply whose data holds one chain of n tree nodes, each the child of the one before. A tree of n nodes
// nests 3+2*(n-1) JSON levels deep under the data envelope.
func wireChain(n int) string {
	var b strings.Builder
	b.WriteString(`{"data":[`)
	for k := 0; k < n; k++ {
		b.WriteString(`{"id":"n-` + strconv.Itoa(k) + `","title":"T ` + strconv.Itoa(k) + `"`)
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

// wireArrays returns a reply whose data is n arrays deep around one object, which nests n+2 JSON levels.
func wireArrays(n int) string {
	return `{"data":` + strings.Repeat("[", n) + `{"id":"x","title":"X"}` + strings.Repeat("]", n) + `}`
}

// wireTreeIDs reads one tree reply through a model and returns its nodes in order.
func wireTreeIDs(t *testing.T, model *spec.Spec, op domain.OperationID, body string) []domain.Record {
	t.Helper()
	rt := rebindReplies(rebindJSON(200, body))
	rec, err := wireRun(t, model, rt, op, domain.NewArgs(nil))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	return rec.Items()
}

// TestMergeKeepsDeclaredListOrderAndLabelsEachRecord reads a merge of two user lists and one document list, where the reply
// holds the lists in another order.
//
// @scenario specs/libs/retriever-engine/behaviours/merges-and-trees.feature: A merge keeps the declared order of its lists and tags each record with its list label
func TestMergeKeepsDeclaredListOrderAndLabelsEachRecord(t *testing.T) {
	model := rebindDoc(t, rebindHeader+wireMergeOps)
	rt := rebindReplies(rebindJSON(200, `{"data":{"documents":[{"id":"d-1"}],"guests":[{"id":"g-1"}],"users":[{"id":"u-1"}]}}`))
	rec, err := wireRun(t, model, rt, "example-wiki.mention.search", domain.NewArgs(nil))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	items := rec.Items()
	want := []struct{ id, kind string }{{"u-1", "user"}, {"g-1", "user"}, {"d-1", "document"}}
	if len(items) != len(want) {
		t.Fatalf("got %d records, want %d", len(items), len(want))
	}
	for i, w := range want {
		if got := items[i].Str("id"); got != w.id {
			t.Fatalf("record %d id = %q, want %q in declared list order", i, got, w.id)
		}
		if got := items[i].Str("kind"); got != w.kind {
			t.Fatalf("record %d kind = %q, want its list label %q", i, got, w.kind)
		}
	}
}

// TestMergeLabelWinsOverTheReplyKindField reads a merge whose reply holds a kind field on each record.
//
// @scenario specs/libs/retriever-engine/behaviours/merges-and-trees.feature: A merge label wins over a kind field that the reply holds on each record
func TestMergeLabelWinsOverTheReplyKindField(t *testing.T) {
	model := rebindDoc(t, rebindHeader+wireMergeKindOps)
	rt := rebindReplies(rebindJSON(200, `{"data":{"users":[{"id":"u-1","kind":"other"}],"documents":[{"id":"d-1","kind":"other"}]}}`))
	rec, err := wireRun(t, model, rt, "example-wiki.mention.search", domain.NewArgs(nil))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	for i, want := range []string{"user", "document"} {
		if got := rec.Items()[i].Str("kind"); got != want {
			t.Fatalf("record %d kind = %q, want the constant label %q", i, got, want)
		}
	}
}

// TestMergeListThatIsNotAListOfObjectsIsUnexpected reads a merge whose user list is a pointer that names no list, or whose
// item is not an object.
//
// @scenario specs/libs/retriever-engine/behaviours/merges-and-trees.feature: A merge whose list is <problem> fails the read as unexpected
func TestMergeListThatIsNotAListOfObjectsIsUnexpected(t *testing.T) {
	cases := []struct {
		problem string
		body    string
	}{
		{"a pointer that names no list", `{"data":{"users":{"id":"u-1"},"documents":[]}}`},
		{"a list whose item is not an object", `{"data":{"users":["u-1"],"documents":[]}}`},
	}
	for _, c := range cases {
		t.Run(c.problem, func(t *testing.T) {
			model := rebindSpec(t, wireMergeOps)
			rt := rebindReplies(rebindJSON(200, c.body))
			_, err := wireRun(t, model, rt, "example-wiki.mention.search", domain.NewArgs(nil))
			if !errors.Is(err, domain.ErrUnexpectedResponse) {
				t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
			}
		})
	}
}

// TestMergeNeedingAnUnservedConverterFailsBeforeAnyRequest reads a merge whose user list names a converter the engine
// does not serve.
//
// @scenario specs/libs/retriever-engine/behaviours/merges-and-trees.feature: A merge that needs a converter the engine does not serve fails before any request
func TestMergeNeedingAnUnservedConverterFailsBeforeAnyRequest(t *testing.T) {
	model := rebindDoc(t, rebindHeader+wireMergeConvertOps)
	rt := rebindReplies(rebindJSON(200, `{"data":{"users":[{"id":"u-1"}],"documents":[]}}`))
	_, err := wireRun(t, model, rt, "example-wiki.mention.search", domain.NewArgs(nil))
	if !errors.Is(err, domain.ErrUnsupportedOperation) {
		t.Fatalf("err = %v, want ErrUnsupportedOperation", err)
	}
	if n := len(rt.seen()); n != 0 {
		t.Fatalf("an unsupported merge sent %d requests, want none", n)
	}
}

// TestPagedMergeMapsEachPageWithTheMergeLists serves two pages of a merge, each mapped with the merge lists.
//
// @scenario specs/libs/retriever-engine/behaviours/merges-and-trees.feature: A paged merge maps each page with the merge lists
func TestPagedMergeMapsEachPageWithTheMergeLists(t *testing.T) {
	model := rebindDoc(t, rebindHeader+wireMergePagedOps)
	rt := rebindFunc(func(_ int, req rebindRequest) rebindReply {
		switch req.url.RawQuery {
		case "limit=2&offset=0":
			return rebindJSON(200, `{"data":{"users":[{"id":"u-1"},{"id":"u-2"}],"documents":[]}}`)
		case "limit=2&offset=2":
			return rebindJSON(200, `{"data":{"users":[],"documents":[{"id":"d-1"}]}}`)
		default:
			return rebindJSON(200, `{"data":{"users":[],"documents":[]}}`)
		}
	})
	rec, err := wireRun(t, model, rt, "example-wiki.mention.search", domain.NewArgs(nil))
	if err != nil {
		t.Fatalf("a paged merge was refused: %v", err)
	}
	items := rec.Items()
	want := []struct{ id, kind string }{{"u-1", "user"}, {"u-2", "user"}, {"d-1", "document"}}
	if len(items) != len(want) {
		t.Fatalf("got %d records across two pages, want %d", len(items), len(want))
	}
	for i, w := range want {
		if items[i].Str("id") != w.id || items[i].Str("kind") != w.kind {
			t.Fatalf("record %d = %q/%q, want %q/%q", i, items[i].Str("id"), items[i].Str("kind"), w.id, w.kind)
		}
	}
}

// TestTreeReturnsNodesDepthFirstWithParentAndDepth reads two roots, one of which holds a child and a grandchild.
//
// @scenario specs/libs/retriever-engine/behaviours/merges-and-trees.feature: A tree read returns its nodes depth first with each parent ID and depth
func TestTreeReturnsNodesDepthFirstWithParentAndDepth(t *testing.T) {
	model := rebindDoc(t, wireTreeDoc)
	body := `{"data":[{"id":"a","children":[{"id":"a1","children":[{"id":"a11"}]},{"id":"a2"}]},{"id":"b"}]}`
	nodes := wireTreeIDs(t, model, "example-wiki.docs.tree", body)
	want := []struct {
		id     string
		parent any
		depth  int
	}{
		{"a", nil, 0},
		{"a1", "a", 1},
		{"a11", "a1", 2},
		{"a2", "a", 1},
		{"b", nil, 0},
	}
	if len(nodes) != len(want) {
		t.Fatalf("got %d nodes, want %d", len(nodes), len(want))
	}
	for i, w := range want {
		if got := nodes[i].Str("id"); got != w.id {
			t.Fatalf("node %d id = %q, want %q depth first", i, got, w.id)
		}
		if got, _ := nodes[i].Get("parent_id"); got != w.parent {
			t.Fatalf("node %d parent_id = %v, want %v", i, got, w.parent)
		}
		if got, _ := nodes[i].Get("depth"); got != w.depth {
			t.Fatalf("node %d depth = %v, want %d", i, got, w.depth)
		}
	}
}

// TestTreeKeepsSixtyFiveLevelsAndDropsDeeperNodes reads a chain of 70 nodes with the default max depth.
//
// @scenario specs/libs/retriever-engine/behaviours/merges-and-trees.feature: A tree nested past its max depth keeps the levels up to the cap and drops each deeper node
func TestTreeKeepsSixtyFiveLevelsAndDropsDeeperNodes(t *testing.T) {
	model := rebindDoc(t, wireTreeDoc)
	nodes := wireTreeIDs(t, model, "example-wiki.docs.tree", wireChain(70))
	if len(nodes) != 65 {
		t.Fatalf("got %d nodes, want 65 levels from a chain of 70", len(nodes))
	}
	for i, rec := range nodes {
		if got := rec.Str("id"); got != "n-"+strconv.Itoa(i) {
			t.Fatalf("node %d id = %q, want n-%d", i, got, i)
		}
		if got, _ := rec.Get("depth"); got != i {
			t.Fatalf("node %d depth = %v, want %d", i, got, i)
		}
	}
}

// TestTreeWithMaxDepthOneKeepsRootsAndChildren reads the same tree with a max depth of one.
//
// @scenario specs/libs/retriever-engine/behaviours/merges-and-trees.feature: A tree read with a max depth of one keeps the root and its children only
func TestTreeWithMaxDepthOneKeepsRootsAndChildren(t *testing.T) {
	model := rebindDoc(t, wireReplace(t, wireTreeDoc, "        root_parent_param: document_id\n",
		"        root_parent_param: document_id\n        max_depth: 1\n"))
	body := `{"data":[{"id":"a","children":[{"id":"a1","children":[{"id":"a11"}]},{"id":"a2"}]},{"id":"b"}]}`
	nodes := wireTreeIDs(t, model, "example-wiki.docs.tree", body)
	var ids []string
	for _, n := range nodes {
		ids = append(ids, n.Str("id"))
	}
	if strings.Join(ids, ",") != "a,a1,a2,b" {
		t.Fatalf("ids = %v, want the roots and their direct children only", ids)
	}
}

// TestTreeDecodesPastSixtyFourLevelsWhileOtherRepliesFail reads a tree nested past 64 JSON levels, and a flat reply nested
// as deep.
//
// @scenario specs/libs/retriever-engine/behaviours/merges-and-trees.feature: A tree reply nested past 64 JSON levels decodes, while any other reply nested that deep fails
func TestTreeDecodesPastSixtyFourLevelsWhileOtherRepliesFail(t *testing.T) {
	tree := rebindDoc(t, wireTreeDoc)
	nodes := wireTreeIDs(t, tree, "example-wiki.docs.tree", wireChain(50))
	if len(nodes) != 50 {
		t.Fatalf("got %d nodes from a tree nested past 64 levels, want 50", len(nodes))
	}

	flat := rebindSpec(t, wireFlatOps)
	rt := rebindReplies(rebindJSON(200, wireArrays(101)))
	_, err := wireRun(t, flat, rt, "example-wiki.docs.flat", domain.NewArgs(nil))
	if !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("a flat reply nested past 64 levels = %v, want ErrUnexpectedResponse", err)
	}
}

// TestTreeModelWithoutItsIDIsRefusedBeforeAnyRequest builds the engine from a tree model whose id field is not required.
//
// @scenario specs/libs/retriever-engine/behaviours/merges-and-trees.feature: A hand-built tree model without its required ID field is refused before any request
func TestTreeModelWithoutItsIDIsRefusedBeforeAnyRequest(t *testing.T) {
	model := rebindDoc(t, wireTreeDoc)
	model.Operations[0].Response.Fields["id"] = spec.Field{Pointer: "/id"}
	rt := rebindReplies(rebindJSON(200, `{"data":[]}`))
	if _, err := engine.New(model, rebindSource(t), wirePorts(&rebindClock{}, rt)); err == nil {
		t.Fatal("a tree model without a required id was accepted")
	}
	if n := len(rt.seen()); n != 0 {
		t.Fatalf("the refused model sent %d requests, want none", n)
	}
}

// TestPagedTreeFlattensEachPageFromDepthZero serves a first page of two roots, one with a child, and a short second page.
//
// @scenario specs/libs/retriever-engine/behaviours/merges-and-trees.feature: A paged tree read flattens each page on its own with depths from zero on every page
func TestPagedTreeFlattensEachPageFromDepthZero(t *testing.T) {
	model := rebindDoc(t, rebindHeader+wirePagedTreeOps)
	rt := rebindFunc(func(_ int, req rebindRequest) rebindReply {
		switch req.url.RawQuery {
		case "limit=2&offset=0":
			return rebindJSON(200, `{"data":[{"id":"a","title":"A","children":[{"id":"a1","title":"A1"}]},{"id":"b","title":"B"}]}`)
		case "limit=2&offset=3":
			return rebindJSON(200, `{"data":[{"id":"c","title":"C"}]}`)
		default:
			return rebindJSON(200, `{"data":[]}`)
		}
	})
	rec, err := wireRun(t, model, rt, "example-wiki.docs.treepage", domain.NewArgs(nil))
	if err != nil {
		t.Fatalf("a paged tree read was refused: %v", err)
	}
	want := []struct {
		id    string
		depth int
	}{{"a", 0}, {"a1", 1}, {"b", 0}, {"c", 0}}
	items := rec.Items()
	if len(items) != len(want) {
		t.Fatalf("got %d items, want %d across two pages", len(items), len(want))
	}
	for i, w := range want {
		if got := items[i].Str("id"); got != w.id {
			t.Fatalf("item %d id = %q, want %q", i, got, w.id)
		}
		if got, _ := items[i].Get("depth"); got != w.depth {
			t.Fatalf("item %d depth = %v, want %d", i, got, w.depth)
		}
	}
}
