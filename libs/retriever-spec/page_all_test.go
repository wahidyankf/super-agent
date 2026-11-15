package spec

import (
	"strings"
	"testing"
)

// pageAllDoc returns a read document with the given pagination block. The read declares no page parameters, so only
// a page-all rule can page it.
func pageAllDoc(pagination string) string {
	return `version: 1
kind: example-wiki
summary: Lists every collection of an example wiki.
endpoint:
  base_path: /v1
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
operations:
  - id: example-wiki.collections.list
    summary: List every collection.
    access: read
    method: POST
    read_over_post: true
    path: /collections
    body: {}
    pagination:
` + pagination + `    response:
      items: /data
      fields:
        id:
          pointer: /id
    examples:
      - request: {}
        response:
          status: 200
          body:
            data:
              - id: c1
        expect:
          id: c1
`
}

// TestPageAllPaginationValidates checks that a read with no page parameters takes a page-all rule: the fixed page
// size, the page cap, and where the paging values travel. The engine sends pages of the fixed size until a short
// page, and fails at the cap with a page-cap error, mirroring the Go reader.
func TestPageAllPaginationValidates(t *testing.T) {
	doc := pageAllDoc("      style: page-all\n      in: body\n      max_pages: 20\n      page_size: 25\n")
	if res := Validate([]byte(doc)); !res.Valid() {
		t.Fatalf("a page-all rule is refused: %v", res.Err())
	}
}

// TestPageAllPaginationNeedsAPageSize checks that a page-all rule without its fixed page size is refused: without a
// size the engine could not page an operation that declares no page parameters.
func TestPageAllPaginationNeedsAPageSize(t *testing.T) {
	doc := pageAllDoc("      style: page-all\n      in: body\n      max_pages: 20\n")
	res := Validate([]byte(doc))
	if res.Valid() {
		t.Fatal("a page-all rule without a page size validates")
	}
	if !strings.Contains(res.Err().Error(), "/operations/0/pagination") {
		t.Fatalf("err = %v, want the pagination path", res.Err())
	}
}

// TestPageAllPaginationNamesNoParameters checks that a page-all rule names no paging parameter: the operation
// declares none, so a limit, offset, cursor, or next pointer name has nothing to bind to. The schema refuses the
// document at the pagination rule, and the semantic rule below names the exact parameter.
func TestPageAllPaginationNamesNoParameters(t *testing.T) {
	for _, param := range []string{"limit_param: limit", "offset_param: offset", "cursor_param: cursor", "next_pointer: /next"} {
		name := strings.Split(param, ":")[0]
		doc := pageAllDoc("      style: page-all\n      in: body\n      max_pages: 20\n      page_size: 25\n      " + param + "\n")
		res := Validate([]byte(doc))
		if res.Valid() {
			t.Fatalf("a page-all rule with %s validates", name)
		}
		if !strings.Contains(res.Err().Error(), "/operations/0/pagination") {
			t.Fatalf("%s: err = %v, want the pagination path", name, res.Err())
		}
	}
}

// TestPageAllSemanticNamesEachRefusedParameter checks the Go rule behind the schema: each named paging parameter of
// a page-all rule is refused at its own path with the fixed rule, so a model built by hand meets the same refusal
// as a document.
func TestPageAllSemanticNamesEachRefusedParameter(t *testing.T) {
	op := Operation{
		ID: "example-wiki.collections.list",
		Pagination: &Pagination{
			Style: "page-all", In: "body", MaxPages: 20, PageSize: 25,
			LimitParam: "limit", OffsetParam: "offset", CursorParam: "cursor", NextPointer: "/next",
		},
	}
	fs := checkPagination(op, "/operations/0", map[string]Param{})
	want := map[string]bool{
		"/operations/0/pagination/cursor_param": false,
		"/operations/0/pagination/limit_param":  false,
		"/operations/0/pagination/next_pointer": false,
		"/operations/0/pagination/offset_param": false,
	}
	for _, f := range fs {
		if _, ok := want[f.Path]; !ok {
			t.Fatalf("finding path = %q, want one of the named parameters", f.Path)
		}
		if f.Rule != "names a paging parameter of a page-all rule" {
			t.Fatalf("finding rule = %q, want the fixed rule", f.Rule)
		}
		want[f.Path] = true
	}
	for path, seen := range want {
		if !seen {
			t.Fatalf("no finding at %q", path)
		}
	}
}

// TestPageAllSemanticRefusesAnOutOfRangePageSize checks the Go rule behind the schema: a page-all page size outside
// 1 to 100 is refused at its own path with the fixed rule.
func TestPageAllSemanticRefusesAnOutOfRangePageSize(t *testing.T) {
	op := Operation{
		ID:         "example-wiki.collections.list",
		Pagination: &Pagination{Style: "page-all", In: "body", MaxPages: 20},
	}
	fs := checkPagination(op, "/operations/0", map[string]Param{})
	if len(fs) != 1 || fs[0].Path != "/operations/0/pagination/page_size" {
		t.Fatalf("findings = %+v, want the page-size path", fs)
	}
	if fs[0].Rule != "is not a page size from 1 to 100" {
		t.Fatalf("finding rule = %q, want the fixed rule", fs[0].Rule)
	}
}
