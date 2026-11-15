package engine

import (
	"errors"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// treeOp is a tree of one root with one child, each with 16 characters of text in the id and the title. Mapping
// the root charges 16 characters for the id and 16 for the title, and mapping the child charges 32 more with 16
// for the copied parent id, so the tree costs 80 characters of the shared output budget.
func treeOp() spec.Operation {
	return spec.Operation{
		Response: spec.Response{
			Items: "/data",
			Fields: map[string]spec.Field{
				"id":    {Pointer: "/id", Required: true},
				"title": {Pointer: "/title"},
			},
			Tree: &spec.Tree{Children: "/children", ParentField: "parent_id", DepthField: "depth"},
		},
	}
}

// treeRoot is a reply whose root has a 16-character id and title with one child of the same shape.
func treeRoot() map[string]any {
	return map[string]any{
		"data": []any{
			map[string]any{
				"id":    "0123456789abcdef",
				"title": "0123456789abcdef",
				"children": []any{
					map[string]any{"id": "fedcba9876543210", "title": "fedcba9876543210"},
				},
			},
		},
	}
}

// TestTreeOutputSharesBudget proves the shared cap of a tree: the root with its child maps with a budget of 80,
// and a budget of 79 fails the read with reply_too_large. A budget per record would pass either record alone at
// 79, so the failure proves the cap is shared across the computed copies too.
func TestTreeOutputSharesBudget(t *testing.T) {
	budget := 80
	items, err := treeRecords(treeOp(), domain.NewArgs(nil), treeRoot(), &budget, "https://docs.example.test")
	if err != nil {
		t.Fatalf("a tree within the shared cap was refused: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
	if budget != 0 {
		t.Fatalf("budget = %d, want 0 after the shared charge of 80", budget)
	}
	budget = 79
	if _, err := treeRecords(treeOp(), domain.NewArgs(nil), treeRoot(), &budget, "https://docs.example.test"); !errors.Is(err, domain.ErrReplyTooLarge) {
		t.Fatalf("a tree past the shared cap = %v, want ErrReplyTooLarge", err)
	}
}

// TestTreeRefusesBadShapes checks the shapes that a tree refuses: an items pointer that names no list, a children
// value that is not a list, and a child that is not an object are each an unexpected response. A missing children
// value ends the branch, so a leaf without one still maps.
func TestTreeRefusesBadShapes(t *testing.T) {
	op := treeOp()
	leaf := map[string]any{"id": "0123456789abcdef", "title": "0123456789abcdef"}
	cases := []struct {
		name string
		root any
		want int
	}{
		{"no list", map[string]any{"data": map[string]any{}}, -1},
		{"a non-list child list", map[string]any{"data": []any{leafWith("children", "leaf")}}, -1},
		{"a non-object child", map[string]any{"data": []any{leafWith("children", []any{"leaf"})}}, -1},
		{"a missing required id", map[string]any{"data": []any{map[string]any{}}}, -1},
		{"a missing children value", map[string]any{"data": []any{leaf}}, 1},
		{"a null children value", map[string]any{"data": []any{leafWith("children", nil)}}, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			budget := 10000
			items, err := treeRecords(op, domain.NewArgs(nil), c.root, &budget, "https://docs.example.test")
			if c.want < 0 {
				if !errors.Is(err, domain.ErrUnexpectedResponse) {
					t.Fatalf("a tree of %s = %v, want ErrUnexpectedResponse", c.name, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("a tree of %s was refused: %v", c.name, err)
			}
			if len(items) != c.want {
				t.Fatalf("got %d items, want %d", len(items), c.want)
			}
		})
	}
}

// leafWith returns a leaf node with the named key set to the value, for the shape rows.
func leafWith(key string, value any) map[string]any {
	return map[string]any{"id": "0123456789abcdef", "title": "0123456789abcdef", key: value}
}
