package spec

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// mergeReadOp is a mention search whose response merges two lists with constant labels. It is appended to the example
// as operation 4, so the paths of the rows above keep their indexes, as the E1 postReadOp, the E2 body paging ops,
// the E3 requiredFieldOp, the E4 nestedRecordOp, and the E5 firstReadOp do. There is no top-level fields entry: each
// list names its own fields, and the label field is a constant of the list.
const mergeReadOp = `  - id: example-wiki.mention.search
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

// mergeDoc returns the bundled example spec with the merge read appended as operation 4.
func mergeDoc() string {
	return mutateAll(exampleWiki, "        expect: {}", "        expect: {}\n"+mergeReadOp)
}

// mfx returns the bundled example spec with the merge read appended as operation 4, with the given (old, repl)
// pairs applied, for a fault row.
func mfx(pairs ...string) func() string {
	return func() string { return mutateAll(mergeDoc(), pairs...) }
}

// mergeWithLists returns the example with a merge read of n lists named list-a and up, each reading its own pointer
// with one id field, for the count-limit rows.
func mergeWithLists(n int) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		name := fmt.Sprintf("list-%c", 'a'+i)
		b.WriteString("          - items: /data/" + name + "\n            label: " + name + "\n            fields:\n              id:\n                pointer: /id\n")
	}
	op := `  - id: example-wiki.mention.search
    summary: Search mentions across users and documents.
    access: read
    method: GET
    path: /mention
    response:
      merge:
        label_field: kind
        lists:
` + strings.TrimSuffix(b.String(), "\n") + `
    examples:
      - request: {}
        response:
          status: 200
          body:
            data: {}
        expect:
          id: u-1
          kind: user
`
	return mutateAll(exampleWiki, "        expect: {}", "        expect: {}\n"+op)
}

// TestMergeKeywordMapsToModel checks the E6 schema keyword: a response with a merge block loads, and the model keeps
// the label field with both lists, each with its items pointer and its constant label. It reads the model as JSON, as
// the E3 format test, the E4 object test, and the E5 first test do, so the test fails on the verdict before the fix,
// not on the shape of the model.
func TestMergeKeywordMapsToModel(t *testing.T) {
	r := requireValid(t, mergeDoc())
	raw, err := json.Marshal(r.Spec.Operations[4].Response)
	if err != nil {
		t.Fatalf("the model does not encode: %v", err)
	}
	for _, want := range []string{`"merge":{"label_field":"kind"`, `"items":"/data/users"`, `"label":"user"`,
		`"items":"/data/documents"`, `"label":"document"`} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("the model drops %s: %s", want, raw)
		}
	}
}

// TestMergeListCountLimit checks the E6 count limit: a merge of two lists and of eight lists loads, and a merge of
// one list and of nine lists is refused at the lists field.
func TestMergeListCountLimit(t *testing.T) {
	requireValid(t, mergeWithLists(2))
	requireValid(t, mergeWithLists(8))
	requireRefused(t, []byte(mergeWithLists(1)), "/operations/4/response/merge/lists")
	requireRefused(t, []byte(mergeWithLists(9)), "/operations/4/response/merge/lists")
}

// TestMergeLabelFormIsClosed checks the E6 label limit: a label is lowercase letters and hyphens of at most 32
// characters, and the label field is a field name. An upper-case label, a label with a digit, a label of 33
// characters, and an upper-case label field are each refused at their own field.
func TestMergeLabelFormIsClosed(t *testing.T) {
	requireRefused(t, []byte(mfx("            label: user\n", "            label: User\n")()), "/operations/4/response/merge/lists/0/label")
	requireRefused(t, []byte(mfx("            label: user\n", "            label: user-1\n")()), "/operations/4/response/merge/lists/0/label")
	requireRefused(t, []byte(mfx("            label: user\n", "            label: "+strings.Repeat("a", 33)+"\n")()),
		"/operations/4/response/merge/lists/0/label")
	requireRefused(t, []byte(mfx("        label_field: kind\n", "        label_field: Kind\n")()), "/operations/4/response/merge/label_field")
}

// TestMergeWithAnotherShapeIsRefused checks the E6 schema rule: a response that names merge with items, with first,
// or with top-level fields is refused at the response, because a response holds at most one of items, first, and
// merge, and a merge response names its fields on its lists.
func TestMergeWithAnotherShapeIsRefused(t *testing.T) {
	requireRefused(t, []byte(mfx("    response:\n      merge:", "    response:\n      items: /data\n      merge:")()),
		"/operations/4/response")
	requireRefused(t, []byte(mfx("    response:\n      merge:", "    response:\n      first: /data\n      merge:")()),
		"/operations/4/response")
	requireRefused(t, []byte(mfx("    response:\n      merge:", "    response:\n      fields:\n        id:\n          pointer: /id\n      merge:")()),
		"/operations/4/response")
}

// soundMerge returns a merge block of two lists that the Go rule accepts.
func soundMerge() *Merge {
	return &Merge{
		LabelField: "kind",
		Lists: []MergeList{
			{Items: "/data/users", Label: "user", Fields: map[string]Field{"id": {Pointer: "/id"}}},
			{Items: "/data/documents", Label: "document", Fields: map[string]Field{"id": {Pointer: "/id"}}},
		},
	}
}

// mergeWithNLists returns a merge block of n lists named list-a and up, each with one id field, for the
// count-limit rows of hand-built models.
func mergeWithNLists(n int) *Merge {
	lists := make([]MergeList, 0, n)
	for i := 0; i < n; i++ {
		name := fmt.Sprintf("list-%c", 'a'+i)
		lists = append(lists, MergeList{Items: "/data/" + name, Label: name, Fields: map[string]Field{"id": {Pointer: "/id"}}})
	}
	return &Merge{LabelField: "kind", Lists: lists}
}

// TestCheckResponseShapeRefusesMergePairs checks the Go rule on models that bypass the schema: a response that names
// merge with items or with first is refused at the response with a fixed rule. Merge alone, items alone, and first
// alone pass with no finding.
func TestCheckResponseShapeRefusesMergePairs(t *testing.T) {
	cases := []struct {
		name     string
		response Response
		path     string
	}{
		{"merge alone", Response{Merge: soundMerge()}, ""},
		{"items alone", Response{Items: "/items", Fields: map[string]Field{"id": {Pointer: "/id"}}}, ""},
		{"first alone", Response{First: "/data", Fields: map[string]Field{"id": {Pointer: "/id"}}}, ""},
		{"items with merge", Response{Items: "/items", Merge: soundMerge()}, "/operations/0/response"},
		{"first with merge", Response{First: "/data", Merge: soundMerge()}, "/operations/0/response"},
		{"items with first", Response{Items: "/items", First: "/data"}, "/operations/0/response"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := &Spec{Operations: []Operation{{ID: "example-wiki.mention.search", Response: c.response}}}
			fs := CheckResponseShape(s)
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
				if f.Path == c.path && f.Rule == ruleResponseShape {
					found = true
				}
			}
			if !found {
				t.Fatalf("no finding at %q with rule %q; findings: %v", c.path, ruleResponseShape, fs)
			}
		})
	}
}

// TestCheckMergeRefusesHandBuiltModels checks the Go rule on merge blocks that bypass the schema: one list and nine
// lists break the count, an upper-case label and a 33-character label break the label form, and an upper-case label
// field breaks the field-name form. A sound merge passes with no finding.
func TestCheckMergeRefusesHandBuiltModels(t *testing.T) {
	cases := []struct {
		name  string
		merge *Merge
		path  string
		rule  string
	}{
		{"a sound merge", soundMerge(), "", ""},
		{"no merge", nil, "", ""},
		{"one list", &Merge{LabelField: "kind", Lists: soundMerge().Lists[:1]}, "/operations/0/response/merge/lists", ruleMergeCount},
		{"nine lists", mergeWithNLists(9), "/operations/0/response/merge/lists", ruleMergeCount},
		{"an upper-case label", &Merge{LabelField: "kind", Lists: []MergeList{
			{Items: "/data/users", Label: "User", Fields: map[string]Field{"id": {Pointer: "/id"}}},
			{Items: "/data/documents", Label: "document", Fields: map[string]Field{"id": {Pointer: "/id"}}},
		}}, "/operations/0/response/merge/lists/0/label", ruleMergeLabel},
		{"a 33-character label", &Merge{LabelField: "kind", Lists: []MergeList{
			{Items: "/data/users", Label: strings.Repeat("a", 33), Fields: map[string]Field{"id": {Pointer: "/id"}}},
			{Items: "/data/documents", Label: "document", Fields: map[string]Field{"id": {Pointer: "/id"}}},
		}}, "/operations/0/response/merge/lists/0/label", ruleMergeLabel},
		{"an upper-case label field", &Merge{LabelField: "Kind", Lists: soundMerge().Lists},
			"/operations/0/response/merge/label_field", ruleMergeLabelField},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := &Spec{Operations: []Operation{{ID: "example-wiki.mention.search", Response: Response{Merge: c.merge}}}}
			fs := CheckMerge(s)
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

// TestReviewListsMergeRequiredCheckedAndNestedFields checks that the required-fields and nested-objects blocks join
// the fields of each list of a merge: a required id with a closed format and a nested owner show under the merge
// operation, so a reviewer sees the whole feature.
func TestReviewListsMergeRequiredCheckedAndNestedFields(t *testing.T) {
	doc := mfx(
		"          - items: /data/users\n            label: user\n            fields:\n              id:\n                pointer: /id\n",
		"          - items: /data/users\n            label: user\n            fields:\n              id:\n                pointer: /id\n                required: true\n                format: id\n              owner:\n                object:\n                  name:\n                    pointer: /name\n",
	)
	review := Review(requireValid(t, doc()).Spec)
	for _, block := range []string{
		"Required fields: 1\n1. example-wiki.mention.search: required id; formats id=id\n",
		"Nested objects: 1\n1. example-wiki.mention.search: owner with name\n",
	} {
		if !strings.Contains(review, block) {
			t.Fatalf("the review has no block %q:\n%s", block, review)
		}
	}
}

// TestReviewListsMergedLists checks the review of a merge read: its own block names the operation, the label field,
// and each list with its constant label and its items pointer. A spec with no merge read has no such block, so its
// review keeps its text.
func TestReviewListsMergedLists(t *testing.T) {
	review := Review(requireValid(t, mergeDoc()).Spec)
	block := "Merged lists: 1\n1. example-wiki.mention.search: merge kind from 2 lists (user of /data/users, document of /data/documents)\n"
	if !strings.Contains(review, block) {
		t.Fatalf("the review has no merged-lists block:\n%s", review)
	}
	if strings.Contains(Review(requireValid(t, exampleWiki).Spec), "Merged lists") {
		t.Fatal("a spec with no merge read has a merged-lists block")
	}
}
