package spec

import (
	"strings"
	"testing"
)

// pageEndDoc is an offset read with the page-end rule: the isLast flag decides last, and a page holding more than the
// limit is cut to it. It names example hosts only.
const pageEndDoc = `version: 1
kind: example-wiki
summary: Lists projects with the page-end rule.
endpoint:
  base_path: /v1
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
limits:
  timeout_seconds: 20
  max_pages: 20
operations:
  - id: example-wiki.projects.list
    summary: List projects.
    access: read
    method: GET
    path: /projects
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
      - name: query
        type: Query
        in: query
        required: false
    pagination:
      style: offset
      max_pages: 5
      limit_param: limit
      offset_param: offset
      last_pointer: /isLast
      cut_to_limit: limit
    response:
      items: /values
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
            values:
              - key: EXAMPLE-1
              - key: EXAMPLE-2
            isLast: false
        expect:
          key: EXAMPLE-1
`

// truncatedDoc is a cursor read with the truncated flag: each item tells whether more follows. It names example hosts
// only.
const truncatedDoc = `version: 1
kind: example-wiki
summary: Searches issues with the truncated flag.
endpoint:
  base_path: /v1
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
limits:
  timeout_seconds: 20
  max_pages: 20
operations:
  - id: example-wiki.issues.search
    summary: Search issues.
    access: read
    method: POST
    read_over_post: true
    path: /search
    params:
      - name: limit
        type: Limit
        in: query
        required: false
        default: 2
      - name: pageToken
        type: ID
        in: query
        required: false
      - name: jql
        type: Query
        in: body
        required: true
    body:
      jql: "{{jql}}"
    pagination:
      style: cursor
      max_pages: 1
      cursor_param: pageToken
      next_pointer: /nextPageToken
      last_pointer: /isLast
    response:
      items: /issues
      truncated:
        token: /nextPageToken
        last: /isLast
      fields:
        key:
          pointer: /key
    examples:
      - request:
          limit: 2
          jql: project = EXAMPLE
        response:
          status: 200
          body:
            issues:
              - key: EXAMPLE-1
            nextPageToken: token-1
            isLast: false
        expect:
          key: EXAMPLE-1
          truncated: true
`

// TestPageEndRuleValidates checks that an offset read may declare the page-end rule: the last_pointer flag and the
// cut_to_limit parameter. The keywords are new, so the document loads.
func TestPageEndRuleValidates(t *testing.T) {
	r := Validate([]byte(pageEndDoc))
	if !r.Valid() {
		t.Fatalf("a spec with the page-end rule is refused: %v", r.Err())
	}
}

// TestTruncatedRuleValidates checks that a cursor read declaring last_pointer may declare the truncated block, and
// that an example may expect the computed truncated field.
func TestTruncatedRuleValidates(t *testing.T) {
	r := Validate([]byte(truncatedDoc))
	if !r.Valid() {
		t.Fatalf("a spec with the truncated block is refused: %v", r.Err())
	}
}

// TestTruncatedWithoutLastPointerIsRefused checks that the truncated block needs a cursor rule declaring last_pointer:
// a truncated block on an offset read is refused naming the truncated block.
func TestTruncatedWithoutLastPointerIsRefused(t *testing.T) {
	r := Validate([]byte(truncatedOffsetDoc()))
	if !r.Valid() && hasPath(r, "/operations/0/response/truncated") {
		return
	}
	t.Fatalf("a truncated block on an offset rule is accepted: valid=%v findings=%v", r.Valid(), r.Findings)
}

// truncatedOffsetDoc is the example spec whose first operation is an offset read carrying a truncated block, which no
// cursor rule backs. It names example hosts only.
func truncatedOffsetDoc() string {
	return `version: 1
kind: example-wiki
summary: Lists documents with a truncated block on an offset rule.
endpoint:
  base_path: /v1
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
limits:
  timeout_seconds: 20
  max_pages: 20
operations:
  - id: example-wiki.docs.page
    summary: List one page of documents.
    access: read
    method: GET
    path: /docs/page
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
      max_pages: 5
      limit_param: limit
      offset_param: offset
    response:
      items: /items
      truncated:
        token: /next_cursor
        last: /is_last
      fields:
        id:
          pointer: /id
    examples:
      - request:
          limit: 2
          offset: 0
        response:
          status: 200
          body:
            items:
              - id: doc-1
        expect:
          id: doc-1
`
}

// hasPath reports whether any finding names the given path.
func hasPath(r Result, path string) bool {
	for _, f := range r.Findings {
		if f.Path == path {
			return true
		}
	}
	return false
}

// TestCutToLimitNamingNoParameterIsRefused checks that cut_to_limit names a parameter of the operation: naming none is
// refused naming the cut_to_limit field.
func TestCutToLimitNamingNoParameterIsRefused(t *testing.T) {
	doc := cutDoc("cut_to_limit: absent")
	r := Validate([]byte(doc))
	if r.Valid() || !hasPath(r, "/operations/0/pagination/cut_to_limit") {
		t.Fatalf("cut_to_limit naming no parameter is accepted: valid=%v findings=%v", r.Valid(), r.Findings)
	}
}

// TestCutToLimitNamingAQueryTextIsRefused checks that cut_to_limit names a Limit or Int parameter: naming a Query
// parameter is refused naming the cut_to_limit field.
func TestCutToLimitNamingAQueryTextIsRefused(t *testing.T) {
	doc := cutDoc("cut_to_limit: query")
	r := Validate([]byte(doc))
	if r.Valid() || !hasPath(r, "/operations/0/pagination/cut_to_limit") {
		t.Fatalf("cut_to_limit naming a Query parameter is accepted: valid=%v findings=%v", r.Valid(), r.Findings)
	}
}

// TestLastPointerOnPageAllIsRefused checks that the last flag needs an offset or cursor walk: a page-all rule
// declaring last_pointer is refused naming the last_pointer field.
func TestLastPointerOnPageAllIsRefused(t *testing.T) {
	doc := `version: 1
kind: example-wiki
summary: Reads every document with a last flag on a page-all rule.
endpoint:
  base_path: /v1
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
limits:
  timeout_seconds: 20
  max_pages: 20
operations:
  - id: example-wiki.docs.all
    summary: Read every document.
    access: read
    method: GET
    path: /docs/all
    pagination:
      style: page-all
      max_pages: 5
      page_size: 2
      last_pointer: /isLast
    response:
      items: /items
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
	r := Validate([]byte(doc))
	if r.Valid() || !hasPath(r, "/operations/0/pagination/last_pointer") {
		t.Fatalf("last_pointer on a page-all rule is accepted: valid=%v findings=%v", r.Valid(), r.Findings)
	}
}

// cutDoc returns the page-end document with its cut_to_limit line replaced by the given line.
func cutDoc(line string) string {
	out := pageEndDoc
	from := "      cut_to_limit: limit"
	i := strings.Index(out, from)
	if i < 0 {
		return out
	}
	return out[:i] + "      " + line + out[i+len(from):]
}
