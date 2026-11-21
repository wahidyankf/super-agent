package unit_test

import (
	"strings"
	"testing"

	spec "github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// firstOp is a first-item read: its response maps the first item of the items list.
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

// listRequiredOp is a list read whose id field is required.
const listRequiredOp = `  - id: example-wiki.docs.required
    summary: List documents with a required id.
    access: read
    method: GET
    path: /required
    response:
      items: /items
      fields:
        id:
          pointer: /id
          required: true
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

// issuePageOp is an offset read with the page-end rule whose path names an issue key.
const issuePageOp = `  - id: example-wiki.issues.list
    summary: List the issues of one project.
    access: read
    method: GET
    path: /issues/{key}
    params:
      - name: key
        type: IssueKey
        in: path
        required: true
      - name: limit
        type: Limit
        in: query
        required: false
        default: 20
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
      last_pointer: /isLast
      cut_to_limit: limit
    response:
      items: /items
      fields:
        id:
          pointer: /id
    examples:
      - request:
          key: EX-1
        response:
          status: 200
          body:
            items:
              - id: doc-1
            isLast: true
        expect: {}
`

// reviewOf returns the lines of the review of a document that the user loader accepts.
func reviewOf(t *testing.T, doc string) []string {
	t.Helper()
	r := requireAccepted(t, doc)
	return strings.Split(spec.Review(r.Spec), "\n")
}

// hasLine reports whether a review holds a line that equals want.
func hasLine(lines []string, want string) bool {
	for _, line := range lines {
		if line == want {
			return true
		}
	}
	return false
}

// lineWith returns the first line of a review that holds the text, or the empty text.
func lineWith(lines []string, text string) string {
	for _, line := range lines {
		if strings.Contains(line, text) {
			return line
		}
	}
	return ""
}

// @scenario specs/libs/retriever-spec/behaviours/review-text.feature: The review of the bundled example opens with the name, the summary, the endpoint, and the auth
func TestReviewOfBundledExampleOpensWithNameSummaryEndpointAndAuth(t *testing.T) {
	lines := reviewOf(t, exampleWiki)
	if lines[0] != "connector spec example-wiki, version 1" {
		t.Fatalf("first line = %q; want the name and version", lines[0])
	}
	endpoint := lineWith(lines, "endpoint")
	if !strings.Contains(endpoint, "/v1") || !strings.Contains(endpoint, ".example.test") {
		t.Fatalf("endpoint line = %q; want the base path /v1 and the host suffix .example.test", endpoint)
	}
	auth := lineWith(lines, "auth")
	if !strings.Contains(auth, "bearer") || !strings.Contains(auth, "EXAMPLE_WIKI_API_KEY") {
		t.Fatalf("auth line = %q; want bearer and the variable EXAMPLE_WIKI_API_KEY", auth)
	}
}

// @scenario specs/libs/retriever-spec/behaviours/review-text.feature: The review lists each operation with its label, method, path, writes flag, and surfaces
func TestReviewListsEachOperationWithLabelMethodPathWritesAndSurfaces(t *testing.T) {
	lines := reviewOf(t, exampleWiki)
	for _, want := range []string{
		"1. example-wiki.docs.list: read, GET /v1/docs, writes no, cli on, mcp on",
		"3. example-wiki.docs.create: write, POST /v1/docs, writes yes, cli on, mcp on",
	} {
		if !hasLine(lines, want) {
			t.Fatalf("the review has no line %q:\n%s", want, strings.Join(lines, "\n"))
		}
	}
	count := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "1. ") || strings.HasPrefix(line, "2. ") || strings.HasPrefix(line, "3. ") ||
			strings.HasPrefix(line, "4. ") || strings.HasPrefix(line, "5. ") {
			if strings.Contains(line, "example-wiki.") {
				count++
			}
		}
	}
	if count != 4 {
		t.Fatalf("the review lists %d operations; want 4:\n%s", count, strings.Join(lines, "\n"))
	}
}

// @scenario specs/libs/retriever-spec/behaviours/review-text.feature: The review of a spec with a POST read adds a block that names its body slots
func TestReviewOfSpecWithPostReadAddsBlockNamingItsBodySlots(t *testing.T) {
	const postCollection = `  - id: example-wiki.docs.search
    summary: Search documents by collection.
    access: read
    method: POST
    read_over_post: true
    path: /search
    body:
      collection_id: "{{collection_id}}"
    params:
      - name: collection_id
        type: Text
        in: body
        required: true
    response:
      items: /items
      fields:
        id:
          pointer: /id
    examples:
      - request:
          collection_id: col-1
        response:
          status: 200
          body:
            items:
              - id: doc-1
        expect: {}
`
	review := strings.Join(reviewOf(t, oneOp(postCollection)), "\n")
	for _, want := range []string{"POST reads", "example-wiki.docs.search", "/v1/search", "collection_id"} {
		if !strings.Contains(review, want) {
			t.Fatalf("the review lacks %q:\n%s", want, review)
		}
	}
}

// @scenario specs/libs/retriever-spec/behaviours/review-text.feature: The review of a spec with no POST read has no POST reads block
func TestReviewOfSpecWithNoPostReadHasNoPostReadsBlock(t *testing.T) {
	if review := strings.Join(reviewOf(t, exampleWiki), "\n"); strings.Contains(review, "POST reads") {
		t.Fatalf("the review has a POST reads block:\n%s", review)
	}
}

// @scenario specs/libs/retriever-spec/behaviours/review-text.feature: The review of a spec shows no value that the spec holds
func TestReviewOfSpecShowsNoValueThatTheSpecHolds(t *testing.T) {
	const distinctive = "Distinctive reply text 7431"
	doc := strings.ReplaceAll(exampleWiki, "First document", distinctive)
	review := strings.Join(reviewOf(t, doc), "\n")
	if strings.Contains(review, distinctive) {
		t.Fatalf("the review shows a value of the spec:\n%s", review)
	}
	if !strings.Contains(review, "EXAMPLE_WIKI_API_KEY") {
		t.Fatalf("the review does not name the variable by its name:\n%s", review)
	}
}

// @scenario specs/libs/retriever-spec/behaviours/review-text.feature: The review of the same spec is the same text on every call
func TestReviewOfSameSpecIsTheSameTextOnEveryCall(t *testing.T) {
	s := requireAccepted(t, exampleWiki).Spec
	if first, second := spec.Review(s), spec.Review(s); first != second {
		t.Fatal("two reviews of the same spec differ")
	}
}

// @scenario specs/libs/retriever-spec/behaviours/review-text.feature: The review states a limit the spec does not set as a cap
func TestReviewStatesLimitThatSpecDoesNotSetAsCap(t *testing.T) {
	lines := reviewOf(t, exampleWiki)
	if lineWith(lines, "timeout 20s") == "" {
		t.Fatalf("no limits line reads timeout 20s:\n%s", strings.Join(lines, "\n"))
	}
	if lineWith(lines, "reply 10485760 bytes (cap)") == "" {
		t.Fatalf("no limits line reads reply 10485760 bytes (cap):\n%s", strings.Join(lines, "\n"))
	}
}

// @scenario specs/libs/retriever-spec/behaviours/review-text.feature: The review states the reply size limit the spec sets
func TestReviewStatesReplySizeLimitTheSpecSets(t *testing.T) {
	doc := mutate(t, exampleWiki, "limits:\n  timeout_seconds: 20\n", "limits:\n  timeout_seconds: 20\n  max_reply_bytes: 1048576\n")
	if lineWith(reviewOf(t, doc), "reply 1048576 bytes") == "" {
		t.Fatal("the review does not state the reply size as 1048576 bytes")
	}
}

// shapeCase is one reply shape of the review: a valid document with one operation that uses the shape, the operation
// ID, and the word the review uses for the shape in the block of that operation.
type shapeCase struct {
	shape, doc, opID, keyword string
	bundled                   bool
}

// builtOp is a list read whose reply field builds a project key from an issue key.
const builtOp = `  - id: example-wiki.docs.built
    summary: List projects built from issue keys.
    access: read
    method: GET
    path: /built
    response:
      items: /items
      fields:
        projectKey:
          project_key: /key
    examples:
      - request: {}
        response:
          status: 200
          body:
            items:
              - key: EX-1
        expect: {}
`

// queryKeyOp is a list read whose limit parameter travels under a wire key of its own.
const queryKeyOp = `  - id: example-wiki.docs.aliased
    summary: List documents with a wire alias.
    access: read
    method: GET
    path: /aliased
    params:
      - name: limit
        type: Limit
        in: query
        required: false
        query_key: maxResults
        default: 20
    response:
      items: /items
      fields:
        id:
          pointer: /id
    examples:
      - request:
          limit: 20
        response:
          status: 200
          body:
            items:
              - id: doc-1
        expect: {}
`

// reviewShapeCases are the reply shapes of the review. Each names one operation, and the review must name the operation
// and the shape in one block.
func reviewShapeCases() []shapeCase {
	return []shapeCase{
		{shape: "tree read", doc: oneOp(treeOp), opID: "example-wiki.docs.tree", keyword: "tree"},
		{shape: "merged list", doc: oneOp(shapeOp(mergeBlock(2, "source"))), opID: "example-wiki.docs.shape", keyword: "merge"},
		{shape: "first-item read", doc: oneOp(firstOp), opID: "example-wiki.docs.first", keyword: "first"},
		{shape: "document layout", doc: oneOp(docLayoutOp()), opID: "example-wiki.docs.report", keyword: "document"},
		{shape: "nested record", doc: oneOp(userRecordOp), opID: "example-wiki.docs.user", keyword: "object"},
		{shape: "open-key flag map", doc: oneOp(settingsOp), opID: "example-wiki.docs.settings", keyword: "bool"},
		{shape: "built IDs", doc: oneOp(builtOp), opID: "example-wiki.docs.built", keyword: "built"},
		{shape: "query key alias", doc: oneOp(queryKeyOp), opID: "example-wiki.docs.aliased", keyword: "maxResults"},
		{shape: "key echo", doc: oneOp(echoOp), opID: "example-wiki.docs.fetch", keyword: "echo"},
		{shape: "fallback read", doc: withOps(meta("example-wiki.docs.full", "{status: [404]}", "[title]"), fullOp),
			opID: "example-wiki.docs.meta", keyword: "fallback"},
		{shape: "issue key and page end", doc: oneOp(issuePageOp), opID: "example-wiki.issues.list", keyword: "page"},
		{shape: "required reply fields", doc: oneOp(listRequiredOp), opID: "example-wiki.docs.required", keyword: "required"},
		{shape: "body paging", doc: oneOp(bodyOffsetOp), opID: "example-wiki.docs.page", keyword: "body"},
		{shape: "POST read", doc: oneOp(postReadOp), opID: "example-wiki.docs.search", keyword: "POST"},
		{shape: "bundled mapping", doc: bundledJiraStub, opID: "jira.whoami", keyword: "catalog", bundled: true},
	}
}

// @scenario specs/libs/retriever-spec/behaviours/review-text.feature: The review names each reply shape and rule that an operation uses
func TestReviewNamesEachReplyShapeAndRuleThatAnOperationUses(t *testing.T) {
	for _, c := range reviewShapeCases() {
		t.Run(c.shape, func(t *testing.T) {
			var lines []string
			if c.bundled {
				r := spec.ParseBundled([]byte(c.doc))
				if r.Spec == nil {
					t.Fatalf("the bundled stub is refused:\n%s", text(r))
				}
				lines = strings.Split(spec.Review(r.Spec), "\n")
			} else {
				lines = reviewOf(t, c.doc)
			}
			review := strings.Join(lines, "\n")
			if !strings.Contains(review, c.opID) || !strings.Contains(review, c.keyword) {
				t.Fatalf("the review has no block that names %s and its %s:\n%s", c.opID, c.shape, review)
			}
		})
	}
}
