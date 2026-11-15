package spec

import (
	"encoding/json"
	"strings"
	"testing"
)

// firstReadOp is a shares read whose response maps the first item of the list at /data. It is appended to the example
// as operation 4, so the paths of the rows above keep their indexes, as the E1 postReadOp, the E2 body paging ops,
// the E3 requiredFieldOp, and the E4 nestedRecordOp do.
const firstReadOp = `  - id: example-wiki.shares.get
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

// ffx returns the bundled example spec with the first-item read appended as operation 4, with the given (old, repl)
// pairs applied, for a fault row.
func ffx(pairs ...string) func() string {
	return func() string {
		return mutateAll(mutateAll(exampleWiki, "        expect: {}", "        expect: {}\n"+firstReadOp), pairs...)
	}
}

// TestFirstKeywordMapsToModel checks the E5 schema keyword: a response with a first pointer loads, and the model keeps
// the pointer. It reads the model as JSON, as the E3 format test and the E4 object test do, so the test fails on the
// verdict before the fix, not on the shape of the model.
func TestFirstKeywordMapsToModel(t *testing.T) {
	r := requireValid(t, mutateAll(exampleWiki, "        expect: {}", "        expect: {}\n"+firstReadOp))
	raw, err := json.Marshal(r.Spec.Operations[4].Response)
	if err != nil {
		t.Fatalf("the model does not encode: %v", err)
	}
	if !strings.Contains(string(raw), `"first":"/data"`) {
		t.Fatalf("the model drops the first pointer: %s", raw)
	}
}

// TestFirstAndItemsTogetherAreRefused checks the E5 schema rule: a response that names both items and first is
// refused at the response, because a response holds at most one of items, first, and merge.
func TestFirstAndItemsTogetherAreRefused(t *testing.T) {
	requireRefused(t, []byte(minimalWith("      items: /items\n", "      items: /items\n      first: /data\n")),
		"/operations/0/response")
}

// TestCheckResponseShapeRefusesHandBuiltModels checks the Go rule on models that bypass the schema: a response that
// names both items and first is refused at the response with a fixed rule. A response with items alone, first alone,
// or neither passes with no finding.
func TestCheckResponseShapeRefusesHandBuiltModels(t *testing.T) {
	cases := []struct {
		name     string
		response Response
		path     string
		rule     string
	}{
		{"neither", Response{}, "", ""},
		{"items alone", Response{Items: "/items"}, "", ""},
		{"first alone", Response{First: "/data"}, "", ""},
		{"both", Response{Items: "/items", First: "/data"}, "/operations/0/response", ruleResponseShape},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := &Spec{Operations: []Operation{{ID: "example-wiki.shares.get", Response: c.response}}}
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

// TestReviewListsFirstReads checks the review of a read with a first pointer: its own block names the operation and
// the list pointer as "first of /data". A spec with no first read has no such block, so its review keeps its text.
func TestReviewListsFirstReads(t *testing.T) {
	review := Review(requireValid(t, mutateAll(exampleWiki, "        expect: {}", "        expect: {}\n"+firstReadOp)).Spec)
	block := "First reads: 1\n1. example-wiki.shares.get: first of /data\n"
	if !strings.Contains(review, block) {
		t.Fatalf("the review has no first-reads block:\n%s", review)
	}
	if strings.Contains(Review(requireValid(t, exampleWiki).Spec), "First reads") {
		t.Fatal("a spec with no first read has a first-reads block")
	}
}
