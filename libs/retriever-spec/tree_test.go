package spec

import (
	"encoding/json"
	"strings"
	"testing"
)

// treeReadOp is a documents read whose response flattens the tree at /data depth first, computing the parent and the
// depth of each node. It is appended to the example as operation 4, so the paths of the rows above keep their
// indexes, as the E1 postReadOp, the E2 body paging ops, the E3 requiredFieldOp, the E4 nestedRecordOp, the E5
// firstReadOp, and the E6 mergeReadOp do. The id is required, because the parent of a node is the id of its parent
// node; the top nodes take their parent from the document_id parameter.
const treeReadOp = `  - id: example-wiki.docs.tree
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
        max_depth: 64
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

// treeDoc returns the bundled example spec with the tree read appended as operation 4.
func treeDoc() string {
	return mutateAll(exampleWiki, "        expect: {}", "        expect: {}\n"+treeReadOp)
}

// tfx returns the bundled example spec with the tree read appended as operation 4, with the given (old, repl) pairs
// applied, for a fault row.
func tfx(pairs ...string) func() string {
	return func() string { return mutateAll(treeDoc(), pairs...) }
}

// TestTreeKeywordMapsToModel checks the E7 schema keyword: a response with a tree block loads, and the model keeps
// the children pointer, the computed field names, the root parent parameter, and the depth cap. It reads the model
// as JSON, as the E3 format test, the E4 object test, the E5 first test, and the E6 merge test do, so the test fails
// on the verdict before the fix, not on the shape of the model.
func TestTreeKeywordMapsToModel(t *testing.T) {
	r := requireValid(t, treeDoc())
	raw, err := json.Marshal(r.Spec.Operations[4].Response)
	if err != nil {
		t.Fatalf("the model does not encode: %v", err)
	}
	for _, want := range []string{`"children":"/children"`, `"parent_field":"parent_id"`, `"depth_field":"depth"`,
		`"root_parent_param":"document_id"`, `"max_depth":64`} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("the model drops %s: %s", want, raw)
		}
	}
}

// TestTreeRefusesMaxDepthAbove64 checks the E7 depth cap: a max_depth above 64 is refused at the max_depth field.
// A max_depth of 64 loads.
func TestTreeRefusesMaxDepthAbove64(t *testing.T) {
	requireRefused(t, []byte(tfx("        max_depth: 64\n", "        max_depth: 65\n")()),
		"/operations/4/response/tree/max_depth")
	requireValid(t, treeDoc())
}

// TestTreeNeedsRequiredId checks the E7 id rule: a tree read without a required id field is refused at the tree,
// because the parent of a node is the id of its parent node. A tree whose id is optional, and a tree with no id
// field at all, are both refused; a tree with a required id loads.
func TestTreeNeedsRequiredId(t *testing.T) {
	requireRefused(t, []byte(tfx("          required: true\n", "")()), "/operations/4/response/tree")
	requireRefused(t, []byte(tfx("        id:\n          pointer: /id\n          required: true\n", "")()),
		"/operations/4/response/tree")
	requireValid(t, treeDoc())
}

// TestTreeNeedsItems checks the E7 shape rule: a response with a tree block but no items list is refused, because
// tree rides on items. A response holds at most one of items, first, and merge, so a tree with first or with merge
// is refused too.
func TestTreeNeedsItems(t *testing.T) {
	requireRefused(t, []byte(tfx("      items: /data\n", "")()), "/operations/4/response/items")
	requireRefused(t, []byte(tfx("      items: /data\n", "      items: /data\n      first: /data\n")()),
		"/operations/4/response")
}

// TestReviewListsTreeReads checks the review of a tree read: its own block names the operation, the items pointer,
// the children pointer, the computed fields, the root parent parameter when the read names one, and the depth cap. A
// spec with no tree read has no such block, so its review keeps its text. A tree without max_depth reviews the
// default cap of 64.
func TestReviewListsTreeReads(t *testing.T) {
	review := Review(requireValid(t, treeDoc()).Spec)
	block := "Tree reads: 1\n1. example-wiki.docs.tree: tree of /data with children /children, parent parent_id, " +
		"depth depth, roots under document_id, at most depth 64\n"
	if !strings.Contains(review, block) {
		t.Fatalf("the review has no tree-reads block:\n%s", review)
	}
	defaulted := Review(requireValid(t, tfx("        max_depth: 64\n", "")()).Spec)
	if !strings.Contains(defaulted, "at most depth 64\n") {
		t.Fatalf("a tree without max_depth reviews no default cap of 64:\n%s", defaulted)
	}
	if strings.Contains(Review(requireValid(t, exampleWiki).Spec), "Tree reads") {
		t.Fatal("a spec with no tree read has a tree-reads block")
	}
}

// TestCheckTreeRefusesHandBuiltModels checks the Go rule on tree blocks that bypass the schema: a tree without
// items, a max_depth above 64, a tree with no id field, and a tree with an optional id are each refused at their own
// path with a fixed rule. A sound tree and a response with no tree pass with no finding.
func TestCheckTreeRefusesHandBuiltModels(t *testing.T) {
	soundTree := func() *Tree {
		return &Tree{Children: "/children", ParentField: "parent_id", DepthField: "depth"}
	}
	soundFields := func() map[string]Field {
		return map[string]Field{"id": {Pointer: "/id", Required: true}}
	}
	cases := []struct {
		name     string
		response Response
		path     string
		rule     string
	}{
		{"a sound tree", Response{Items: "/data", Fields: soundFields(), Tree: soundTree()}, "", ""},
		{"no tree", Response{Items: "/items", Fields: map[string]Field{"id": {Pointer: "/id"}}}, "", ""},
		{"a tree without items", Response{Fields: soundFields(), Tree: soundTree()},
			"/operations/0/response/tree", ruleTreeNeedsItems},
		{"a max_depth above 64", Response{Items: "/data", Fields: soundFields(),
			Tree: &Tree{Children: "/children", ParentField: "parent_id", DepthField: "depth", MaxDepth: 65}},
			"/operations/0/response/tree/max_depth", ruleTreeMaxDepth},
		{"a tree without an id field", Response{Items: "/data",
			Fields: map[string]Field{"title": {Pointer: "/title"}}, Tree: soundTree()},
			"/operations/0/response/tree", ruleTreeNeedsID},
		{"a tree with an optional id", Response{Items: "/data",
			Fields: map[string]Field{"id": {Pointer: "/id"}}, Tree: soundTree()},
			"/operations/0/response/tree", ruleTreeNeedsID},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := &Spec{Operations: []Operation{{ID: "example-wiki.docs.tree", Response: c.response}}}
			fs := CheckTree(s)
			if c.path == "" {
				if len(fs) != 0 {
					t.Fatalf("a sound model has findings: %v", fs)
				}
				return
			}
			found := false
			for _, f := range fs {
				if f.Code != CodeSpecInvalid {
					t.Fatalf("finding code = %q; want %q", f.Code, CodeSpecInvalid)
				}
				if f.Path == c.path && f.Rule == c.rule {
					found = true
				}
			}
			if !found {
				t.Fatalf("no finding at %q with rule %q; findings: %v", c.path, c.rule, fs)
			}
		})
	}
}
