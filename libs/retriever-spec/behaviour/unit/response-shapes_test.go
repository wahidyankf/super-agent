package unit_test

import (
	"fmt"
	"strings"
	"testing"
)

// shapeOpTemplate is a list read at index four whose response block is written by the caller.
const shapeOpTemplate = `  - id: example-wiki.docs.shape
    summary: Read a shape.
    access: read
    method: GET
    path: /shape
    response:
%s    examples:
      - request: {}
        response:
          status: 200
          body:
            items:
              - id: doc-1
        expect: {}
`

// shapeOp returns the list read at index four with the given response block.
func shapeOp(responseBlock string) string {
	return fmt.Sprintf(shapeOpTemplate, responseBlock)
}

// mergeBlock returns a response block that merges count lists, each labelled with one lower-case letter.
func mergeBlock(count int, labelField string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "      merge:\n        label_field: %s\n        lists:\n", labelField)
	for i := 0; i < count; i++ {
		label := string(rune('a' + i))
		fmt.Fprintf(&b, "          - items: /%s\n            label: %s\n            fields:\n              id:\n                pointer: /id\n",
			label, label)
	}
	return b.String()
}

// treeOp is a tree read at index four: its items list holds nodes that each carry a required id.
const treeOp = `  - id: example-wiki.docs.tree
    summary: Read the document tree.
    access: read
    method: GET
    path: /tree
    response:
      items: /items
      fields:
        id:
          pointer: /id
          required: true
      tree:
        children: /children
        parent_field: parent
        depth_field: depth
    examples:
      - request: {}
        response:
          status: 200
          body:
            items:
              - id: doc-1
                children: []
        expect: {}
`

// @scenario specs/libs/retriever-spec/behaviours/response-shapes.feature: A response that names two reply shapes at once is refused at the response
func TestResponseNamingTwoReplyShapesAtOnceIsRefusedAtTheResponse(t *testing.T) {
	cases := []struct {
		name, block string
	}{
		{"items and first", "      items: /items\n      first: /items\n"},
		{"merge and items", mergeBlock(2, "source") + "      items: /items\n"},
		{"merge and first", mergeBlock(2, "source") + "      first: /items\n"},
		{"merge and top-level fields", mergeBlock(2, "source") + "      fields:\n        id:\n          pointer: /id\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			requireRefusedAt(t, withOps(shapeOp(c.block)), "/operations/4/response")
		})
	}
}

// @scenario specs/libs/retriever-spec/behaviours/response-shapes.feature: A merge of lists is accepted only with two to eight lists
func TestMergeOfListsIsAcceptedOnlyWithTwoToEightLists(t *testing.T) {
	cases := []struct {
		count   int
		verdict string
	}{
		{1, "refused"},
		{2, "accepted"},
		{8, "accepted"},
		{9, "refused"},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%d lists", c.count), func(t *testing.T) {
			r := validate(withOps(shapeOp(mergeBlock(c.count, "source"))))
			if r.Valid() != (c.verdict == "accepted") {
				t.Fatalf("the spec is valid = %v; want %s. findings:\n%s", r.Valid(), c.verdict, text(r))
			}
		})
	}
}

// @scenario specs/libs/retriever-spec/behaviours/response-shapes.feature: A merge label or label field that is not lower-case is refused at that label
func TestMergeLabelOrLabelFieldThatIsNotLowerCaseIsRefusedAtThatLabel(t *testing.T) {
	cases := []struct {
		name, old, repl, path string
	}{
		{
			name: "the label of its first list",
			old:  "            label: a\n",
			repl: "            label: Upper\n",
			path: "/operations/4/response/merge/lists/0/label",
		},
		{
			name: "the label field of the merge",
			old:  "        label_field: source\n",
			repl: "        label_field: Upper\n",
			path: "/operations/4/response/merge/label_field",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			doc := withOps(shapeOp(mergeBlock(2, "source")))
			requireRefusedAt(t, mutate(t, doc, c.old, c.repl), c.path)
		})
	}
}

// @scenario specs/libs/retriever-spec/behaviours/response-shapes.feature: A first-item read is accepted when its response names only the first list
func TestFirstItemReadIsAcceptedWhenItsResponseNamesOnlyTheFirstList(t *testing.T) {
	const firstOp = `  - id: example-wiki.docs.first
    summary: Read the first document.
    access: read
    method: GET
    path: /first
    response:
      first: /items
      fields:
        id:
          pointer: /id
    examples:
      - request: {}
        response:
          status: 200
          body:
            items:
              - id: doc-1
        expect:
          id: doc-1
`
	requireAccepted(t, oneOp(firstOp))
}

// @scenario specs/libs/retriever-spec/behaviours/response-shapes.feature: A tree read is accepted when it rides on an items list and names a required id
func TestTreeReadIsAcceptedWhenItRidesOnAnItemsListAndNamesARequiredID(t *testing.T) {
	requireAccepted(t, withOps(treeOp))
}

// @scenario specs/libs/retriever-spec/behaviours/response-shapes.feature: A tree read that breaks its shape is refused at the part that breaks it
func TestTreeReadThatBreaksItsShapeIsRefusedAtThePartThatBreaksIt(t *testing.T) {
	cases := []struct {
		name, old, repl, path string
	}{
		{"has no items list to ride on", "      items: /items\n      fields:\n        id:\n          pointer: /id\n          required: true\n",
			"      fields:\n        id:\n          pointer: /id\n          required: true\n", "/operations/4/response/items"},
		{"also names a first item", "      items: /items\n      fields:\n        id:\n          pointer: /id\n          required: true\n",
			"      items: /items\n      first: /items\n      fields:\n        id:\n          pointer: /id\n          required: true\n",
			"/operations/4/response"},
		{"sets a depth cap above 64", "        depth_field: depth\n", "        depth_field: depth\n        max_depth: 65\n",
			"/operations/4/response/tree/max_depth"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			requireRefusedAt(t, mutate(t, withOps(treeOp), c.old, c.repl), c.path)
		})
	}
}

// @scenario specs/libs/retriever-spec/behaviours/response-shapes.feature: A tree read whose reply holds no required id field is refused at the tree
func TestTreeReadWhoseReplyHoldsNoRequiredIDFieldIsRefusedAtTheTree(t *testing.T) {
	doc := mutate(t, withOps(treeOp), "        id:\n          pointer: /id\n          required: true\n", "        name:\n          pointer: /name\n")
	requireRefusedAt(t, doc, "/operations/4/response/tree")
}

// @scenario specs/libs/retriever-spec/behaviours/response-shapes.feature: A read that names the exact unavailable message of its method is accepted
func TestReadNamingExactUnavailableMessageOfItsMethodIsAccepted(t *testing.T) {
	const legacyOp = `  - id: example-wiki.docs.legacy
    summary: Read a legacy document.
    access: read
    method: GET
    path: /legacy
    response:
      unavailable_message: The method is not available here.
      fields:
        id:
          pointer: /id
    examples:
      - request: {}
        response:
          status: 200
          body:
            id: doc-1
        expect: {}
`
	requireAccepted(t, oneOp(legacyOp))
}
