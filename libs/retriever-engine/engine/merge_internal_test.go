package engine

import (
	"errors"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// mergeOp is a merge of two lists with one text field each, for the output-cap rows. The texts are 16 characters, and
// the labels are 4 and 8 characters, so both lists together charge 44 characters of the shared output budget.
func mergeOp() spec.Operation {
	return spec.Operation{Response: spec.Response{Merge: &spec.Merge{
		LabelField: "kind",
		Lists: []spec.MergeList{
			{Items: "/users", Label: "user", Fields: map[string]spec.Field{"bio": {Pointer: "/bio"}}},
			{Items: "/docs", Label: "document", Fields: map[string]spec.Field{"bio": {Pointer: "/bio"}}},
		},
	}}}
}

// mergeRoot is a reply whose two lists each hold one item of 16 characters of text.
func mergeRoot() map[string]any {
	return map[string]any{
		"users": []any{map[string]any{"bio": "0123456789abcdef"}},
		"docs":  []any{map[string]any{"bio": "0123456789abcdef"}},
	}
}

// TestMergeOutputCap proves the shared cap of a merge: both lists together charge 44 characters of one budget, so a
// budget of 44 maps both items with their labels, and a budget of 43 fails the read with reply_too_large. A budget
// per list would pass either list alone at 43, so the failure proves the cap is shared.
func TestMergeOutputCap(t *testing.T) {
	budget := 44
	items, err := mergeRecords(mergeOp(), mergeRoot(), &budget, "https://docs.example.test")
	if err != nil {
		t.Fatalf("a merge within the shared cap was refused: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
	if got := items[0].Str("kind"); got != "user" {
		t.Fatalf("first kind = %q, want user", got)
	}
	if got := items[1].Str("kind"); got != "document" {
		t.Fatalf("second kind = %q, want document", got)
	}
	if budget != 0 {
		t.Fatalf("budget = %d, want 0 after the shared charge of 44", budget)
	}
	budget = 43
	if _, err := mergeRecords(mergeOp(), mergeRoot(), &budget, "https://docs.example.test"); !errors.Is(err, domain.ErrReplyTooLarge) {
		t.Fatalf("a merge past the shared cap = %v, want ErrReplyTooLarge", err)
	}
}

// TestMergeRefusesAMissingRequiredField checks that a list field with required true fails the merge when the reply
// holds nothing for it, and no partial list comes back.
func TestMergeRefusesAMissingRequiredField(t *testing.T) {
	op := mergeOp()
	op.Response.Merge.Lists[0].Fields = map[string]spec.Field{"id": {Pointer: "/id", Required: true}}
	root := map[string]any{
		"users": []any{map[string]any{}},
		"docs":  []any{map[string]any{"bio": "0123456789abcdef"}},
	}
	budget := 1000
	if _, err := mergeRecords(op, root, &budget, "https://docs.example.test"); !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("a merge with a missing required field = %v, want ErrUnexpectedResponse", err)
	}
}
