package spec

import (
	"strings"
	"testing"
)

// exampleReview is the review of the bundled example spec. Each operation keeps its declaration order, and no line
// names a value of the spec: the key variable is named, and the key is never read.
const exampleReview = `connector spec example-wiki, version 1
summary: Reads and writes documents of an example wiki.
endpoint: base path /v1, hosts ending in .example.test
auth: bearer, key variable EXAMPLE_WIKI_API_KEY
limits: timeout 20s, reply 10485760 bytes (cap), at most 5 pages
operations: 4
1. example-wiki.docs.list: read, GET /v1/docs, writes no, cli on, mcp on
   List documents.
2. example-wiki.docs.get: read, GET /v1/docs/{id}, writes no, cli on, mcp on
   Read one document.
3. example-wiki.docs.create: write, POST /v1/docs, writes yes, cli on, mcp on
   Create a document.
4. example-wiki.docs.delete: write, DELETE /v1/docs/{id}, writes yes, cli on, mcp on
   Delete a document.
`

// TestReviewOfTheExampleMatchesTheGolden checks the exact review text, so its order and its words are fixed.
func TestReviewOfTheExampleMatchesTheGolden(t *testing.T) {
	r := requireValid(t, exampleWiki)
	if got := Review(r.Spec); got != exampleReview {
		t.Fatalf("review differs from the golden text.\n--- got\n%s\n--- want\n%s", got, exampleReview)
	}
}

// TestReviewIsDeterministic checks that the same document gives the same findings and the same review on every run.
func TestReviewIsDeterministic(t *testing.T) {
	first := requireValid(t, exampleWiki)
	second := requireValid(t, exampleWiki)
	if Review(first.Spec) != Review(second.Spec) {
		t.Fatal("two reviews of one document differ")
	}
	bad := []byte(fixtureWith("    path: /docs/{id}\n    params:", "    path: /docs/{key}\n    params:"))
	if allText(Validate(bad)) != allText(Validate(bad)) {
		t.Fatal("two validations of one document differ")
	}
}

// TestReviewShowsNoValueOfTheSpec checks that the review names the key variable and no value that the examples,
// the bodies, and the expectations of the spec carry.
func TestReviewShowsNoValueOfTheSpec(t *testing.T) {
	r := requireValid(t, exampleWiki)
	review := Review(r.Spec)
	if !strings.Contains(review, "EXAMPLE_WIKI_API_KEY") {
		t.Fatalf("the review must name the key variable:\n%s", review)
	}
	for _, value := range []string{"First document", "<p>Hello</p>", "doc-1", "doc-2", "Hello"} {
		if strings.Contains(review, value) {
			t.Fatalf("the review shows the example value %q:\n%s", value, review)
		}
	}
}

// TestReviewStatesEachDefaultedLimitAsACap checks the limit line: a limit the spec sets shows its value, and a limit it
// leaves out shows the cap that applies.
func TestReviewStatesEachDefaultedLimitAsACap(t *testing.T) {
	r := requireValid(t, minimalSpec)
	line := ""
	for _, l := range strings.Split(Review(r.Spec), "\n") {
		if strings.HasPrefix(l, "limits: ") {
			line = l
		}
	}
	want := "limits: timeout 30s (cap), reply 10485760 bytes (cap), at most 20 pages (cap)"
	if line != want {
		t.Fatalf("limit line = %q; want %q", line, want)
	}
}

// TestReviewNamesEachAuthKind checks the authentication line for basic and header auth, which name more variables.
func TestReviewNamesEachAuthKind(t *testing.T) {
	basic := requireValid(t, fixtureWith("  type: bearer", "  type: basic",
		"  credential_env: EXAMPLE_WIKI_API_KEY", "  credential_env: EXAMPLE_WIKI_API_KEY\n  identity_env: EXAMPLE_WIKI_EMAIL"))
	if line := authLine(Review(basic.Spec)); line != "auth: basic, key variable EXAMPLE_WIKI_API_KEY, account variable EXAMPLE_WIKI_EMAIL" {
		t.Fatalf("basic auth line = %q", line)
	}
	header := requireValid(t, fixtureWith("  type: bearer", "  type: header",
		"  credential_env: EXAMPLE_WIKI_API_KEY", "  credential_env: EXAMPLE_WIKI_API_KEY\n  header_name: X-Api-Token"))
	if line := authLine(Review(header.Spec)); line != "auth: header X-Api-Token, key variable EXAMPLE_WIKI_API_KEY" {
		t.Fatalf("header auth line = %q", line)
	}
}

// authLine returns the authentication line of a review.
func authLine(review string) string {
	for _, l := range strings.Split(review, "\n") {
		if strings.HasPrefix(l, "auth: ") {
			return l
		}
	}
	return ""
}

// TestReviewNamesEverySurfaceOfAnOperation checks the surface words: a custom name, a switched-off surface, and the
// default on, which the other tests already read.
func TestReviewNamesEverySurfaceOfAnOperation(t *testing.T) {
	r := requireValid(t, fixtureWith("    path: /docs\n    params:", "    path: /docs\n    cli: false\n    mcp: docs-list\n    params:"))
	review := Review(r.Spec)
	if !strings.Contains(review, "cli off, mcp as docs-list") {
		t.Fatalf("the review must name a switched-off command and a custom tool:\n%s", review)
	}
}

// TestReviewOfAnEndpointWithNoPrefixOrSuffix checks the endpoint line when the spec names neither a path prefix nor a
// host suffix.
func TestReviewOfAnEndpointWithNoPrefixOrSuffix(t *testing.T) {
	r := requireValid(t, minimalSpec)
	review := Review(r.Spec)
	if !strings.Contains(review, "endpoint: base path none, hosts: any host of the catalog") {
		t.Fatalf("endpoint line missing:\n%s", review)
	}
}
