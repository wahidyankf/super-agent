package spec

import (
	"strings"
	"testing"
)

// queryKeyDoc is an offset read whose page parameters travel under wire aliases: limit as maxResults and offset as
// startAt, the Jira names. The catalog names stay the parameter names everywhere else. It names example hosts only.
const queryKeyDoc = `version: 1
kind: example-wiki
summary: Searches issues with wire aliases for the page parameters.
endpoint:
  base_path: /v1
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
operations:
  - id: example-wiki.issues.search
    summary: Search issues.
    access: read
    method: GET
    path: /issues/search
    params:
      - name: limit
        type: Limit
        in: query
        required: false
        default: 2
        query_key: maxResults
      - name: offset
        type: Offset
        in: query
        required: false
        default: 0
        query_key: startAt
      - name: query
        type: Query
        in: query
        required: false
        query_key: jql
    pagination:
      style: offset
      max_pages: 5
      limit_param: limit
      offset_param: offset
    response:
      items: /issues
      fields:
        key:
          pointer: /key
    examples:
      - request:
          limit: 2
          offset: 0
        response:
          status: 200
          body:
            issues:
              - key: EXAMPLE-1
        expect:
          key: EXAMPLE-1
`

// TestQueryKeyValidates checks that a query parameter may name the wire key it travels under: the limit rides as
// maxResults, the offset as startAt, and the filter as jql. The catalog name stays the parameter name.
func TestQueryKeyValidates(t *testing.T) {
	if r := Validate([]byte(queryKeyDoc)); !r.Valid() {
		t.Fatalf("a spec with wire aliases is refused: %v", r.Err())
	}
}

// TestQueryKeyOnAPathParameterIsRefused checks that the load refuses an alias on a path parameter: only an in:
// query parameter travels in the query, so only one takes query_key.
func TestQueryKeyOnAPathParameterIsRefused(t *testing.T) {
	doc := mutateAll(queryKeyDoc,
		"      - name: limit\n        type: Limit\n        in: query\n        required: false\n        default: 2",
		"      - name: limit\n        type: ID\n        in: path\n        required: true")
	r := Validate([]byte(doc))
	if r.Valid() {
		t.Fatal("an alias on a path parameter is valid")
	}
	if !hasFinding(r, "/operations/0/params/0/query_key") {
		t.Fatalf("no finding names the alias:\n%s", allText(r))
	}
}

// TestQueryKeyOnABodyParameterIsRefused checks that the load refuses an alias on a body parameter: a body key keeps
// the parameter name, so only an in: query parameter takes query_key.
func TestQueryKeyOnABodyParameterIsRefused(t *testing.T) {
	doc := mutateAll(queryKeyDoc,
		"      - name: query\n        type: Query\n        in: query\n        required: false\n        query_key: jql",
		"      - name: query\n        type: Query\n        in: body\n        required: false\n        query_key: jql")
	r := Validate([]byte(doc))
	if r.Valid() {
		t.Fatal("an alias on a body parameter is valid")
	}
	if !hasFinding(r, "/operations/0/params/2/query_key") {
		t.Fatalf("no finding names the alias:\n%s", allText(r))
	}
}

// TestQueryKeyEqualToTheParamNameIsRefused checks that the load refuses an alias that repeats the parameter name: an
// alias that changes nothing is pointless noise.
func TestQueryKeyEqualToTheParamNameIsRefused(t *testing.T) {
	doc := mutateAll(queryKeyDoc, "        query_key: jql", "        query_key: query")
	r := Validate([]byte(doc))
	if r.Valid() {
		t.Fatal("an alias that repeats the parameter name is valid")
	}
	if !hasFinding(r, "/operations/0/params/2/query_key") {
		t.Fatalf("no finding names the alias:\n%s", allText(r))
	}
}

// TestQueryKeyCollidingWithAnotherQueryKeyIsRefused checks that the load refuses an alias that another query key of
// the same operation already takes: two pairs would ride under one wire key.
func TestQueryKeyCollidingWithAnotherQueryKeyIsRefused(t *testing.T) {
	doc := mutateAll(queryKeyDoc, "        query_key: startAt", "        query_key: maxResults")
	r := Validate([]byte(doc))
	if r.Valid() {
		t.Fatal("two query keys sharing one wire key are valid")
	}
	if !hasFinding(r, "/operations/0/params/1/query_key") {
		t.Fatalf("no finding names the alias:\n%s", allText(r))
	}
}

// TestQueryKeyWithABadShapeIsRefused checks that the load refuses an alias outside the closed shape: the alias starts
// with a letter and holds letters, digits, and underscores only, so a hyphenated key never loads.
func TestQueryKeyWithABadShapeIsRefused(t *testing.T) {
	doc := mutateAll(queryKeyDoc, "        query_key: maxResults", "        query_key: max-results")
	r := Validate([]byte(doc))
	if r.Valid() {
		t.Fatal("an alias with a hyphen is valid")
	}
	if !hasFinding(r, "/operations/0/params/0/query_key") {
		t.Fatalf("no finding names the alias:\n%s", allText(r))
	}
}

// TestQueryKeyOfAHandBuiltModelIsRefused checks that the Go rule refuses what the schema cannot see in a model built
// by hand: an alias on a path parameter, an alias repeating the name, and a collision never reach the engine.
func TestQueryKeyOfAHandBuiltModelIsRefused(t *testing.T) {
	cases := []struct {
		name string
		edit func(*Spec)
		path string
	}{
		{"an alias on a path parameter", func(s *Spec) {
			s.Operations[0].Params[0].In = "path"
		}, "/operations/0/params/0/query_key"},
		{"an alias outside the closed shape", func(s *Spec) {
			s.Operations[0].Params[0].QueryKey = "max-results"
		}, "/operations/0/params/0/query_key"},
		{"an alias repeating the name", func(s *Spec) {
			s.Operations[0].Params[0].QueryKey = "limit"
		}, "/operations/0/params/0/query_key"},
		{"an alias colliding with another wire key", func(s *Spec) {
			s.Operations[0].Params[1].QueryKey = "maxResults"
		}, "/operations/0/params/1/query_key"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := requireValid(t, queryKeyDoc)
			c.edit(r.Spec)
			if fs := CheckQueryKeys(r.Spec); len(fs) == 0 {
				t.Fatal("a hand-built alias model passes the Go rule")
			} else if !hasFinding(Result{Findings: fs}, c.path) {
				t.Fatalf("no finding names %q: %v", c.path, fs)
			}
		})
	}
}

// TestReviewListsQueryKeys checks the review block of the reads with wire aliases: each names the parameter and the
// wire key it travels under. A spec with no alias writes no block, so the golden review keeps its text.
func TestReviewListsQueryKeys(t *testing.T) {
	review := Review(requireValid(t, queryKeyDoc).Spec)
	for _, line := range []string{
		"Query keys: 1",
		"1. example-wiki.issues.search: limit as maxResults, offset as startAt, query as jql",
	} {
		if !strings.Contains(review, line) {
			t.Fatalf("the review has no line %q:\n%s", line, review)
		}
	}
	if strings.Contains(Review(requireValid(t, exampleWiki).Spec), "Query keys") {
		t.Fatal("the review of a spec with no alias names query keys")
	}
}
