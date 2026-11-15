package spec

import (
	"encoding/json"
	"strings"
	"testing"
)

// fallbackOps is the pair of read operations that the fallback tests append to the bundled example as operations 4
// and 5: a metadata read with a fallback to a body read, and the body read itself. Both take the same path
// parameter, so the target declares only parameters the first operation also declares. It names example hosts only.
const fallbackOps = `  - id: example-wiki.docs.meta
    summary: Read the metadata of one document.
    access: read
    method: GET
    path: /docs/{id}/meta
    params:
      - name: id
        type: ID
        in: path
        required: true
    response:
      fields:
        title:
          pointer: /title
        markdown:
          pointer: /markdown
    fallback:
      operation: example-wiki.docs.export
      when:
        empty: markdown
      fill: [markdown]
    examples:
      - request:
          id: doc-1
        response:
          status: 200
          body:
            title: First document
            markdown: Full text
        expect:
          title: First document
          markdown: Full text
  - id: example-wiki.docs.export
    summary: Read the body of one document.
    access: read
    method: GET
    path: /docs/{id}/export
    params:
      - name: id
        type: ID
        in: path
        required: true
    response:
      fields:
        markdown:
          pointer: /markdown
    examples:
      - request:
          id: doc-1
        response:
          status: 200
          body:
            markdown: Full text
        expect:
          markdown: Full text
`

// fallbackDoc returns the bundled example spec with the fallback pair appended as operations 4 and 5, so the paths
// of the rows above keep their indexes.
func fallbackDoc() string {
	return mutateAll(exampleWiki, "        expect: {}", "        expect: {}\n"+fallbackOps)
}

// fbx returns the fallback document with the given (old, repl) pairs applied, for a fault row.
func fbx(pairs ...string) func() string {
	return func() string { return mutateAll(fallbackDoc(), pairs...) }
}

// TestFallbackKeywordMapsToModel checks the E8 schema keyword: a read with a fallback block loads, and the model
// keeps the target operation, the trigger, and the fill fields. It reads the model as JSON, as the E3 format test,
// the E4 object test, the E5 first test, the E6 merge test, and the E7 tree test do, so the test fails on the verdict
// before the fix, not on the shape of the model.
func TestFallbackKeywordMapsToModel(t *testing.T) {
	r := requireValid(t, fallbackDoc())
	raw, err := json.Marshal(r.Spec.Operations[4].Fallback)
	if err != nil {
		t.Fatalf("the model does not encode: %v", err)
	}
	for _, want := range []string{`"operation":"example-wiki.docs.export"`, `"empty":"markdown"`,
		`"fill":["markdown"]`} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("the model drops %s: %s", want, raw)
		}
	}
}

// TestFallbackChainIsRefused checks the PR 14 gate: a target with a fallback of its own is refused at load, so a
// chain of two fallbacks can never run. A fallback runs at most one extra request per call.
func TestFallbackChainIsRefused(t *testing.T) {
	chained := fbx("        expect:\n          markdown: Full text\n",
		"        expect:\n          markdown: Full text\n    fallback:\n      operation: example-wiki.docs.meta\n      when:\n        empty: markdown\n      fill: [markdown]\n")()
	requireRefused(t, []byte(chained), "/operations/5/fallback/operation")
	requireValid(t, fallbackDoc())
}

// TestFallbackTargetMustBeRead checks the E8 target rule: a fallback to a write is refused at load, because the
// target is a read of the same spec. A read is never answered by a write.
func TestFallbackTargetMustBeRead(t *testing.T) {
	toWrite := fbx("      operation: example-wiki.docs.export\n",
		"      operation: example-wiki.docs.create\n")()
	requireRefused(t, []byte(toWrite), "/operations/4/fallback/operation")
	requireValid(t, fallbackDoc())
}

// TestFallbackStatusTriggerLoads checks the second closed trigger: a fallback on a list of HTTP statuses loads,
// and the model keeps the statuses. The triggers empty and status are owner proposals that the owner can still veto
// at review.
func TestFallbackStatusTriggerLoads(t *testing.T) {
	onStatus := fbx("      when:\n        empty: markdown\n", "      when:\n        status: [404]\n")()
	r := requireValid(t, onStatus)
	raw, err := json.Marshal(r.Spec.Operations[4].Fallback)
	if err != nil {
		t.Fatalf("the model does not encode: %v", err)
	}
	if !strings.Contains(string(raw), `"status":[404]`) {
		t.Fatalf("the model drops the status trigger: %s", raw)
	}
	if strings.Contains(string(raw), "empty") {
		t.Fatalf("the model keeps both triggers: %s", raw)
	}
}

// TestFallbackRefusesTwoTriggers checks that when names exactly one closed trigger: an empty and a status trigger
// together are refused at the when block.
func TestFallbackRefusesTwoTriggers(t *testing.T) {
	both := fbx("      when:\n        empty: markdown\n",
		"      when:\n        empty: markdown\n        status: [404]\n")()
	requireRefused(t, []byte(both), "/operations/4/fallback/when")
	requireValid(t, fallbackDoc())
}

// TestReviewListsFallbackReads checks the review of a fallback read: its own block names the operation, the target,
// the trigger, and the fill fields. A spec with no fallback read has no such block, so its review keeps its text.
func TestReviewListsFallbackReads(t *testing.T) {
	review := Review(requireValid(t, fallbackDoc()).Spec)
	block := "Fallback reads: 1\n1. example-wiki.docs.meta: fallback to example-wiki.docs.export " +
		"when markdown is empty, filling markdown\n"
	if !strings.Contains(review, block) {
		t.Fatalf("the review has no fallback-reads block:\n%s", review)
	}
	onStatus := Review(requireValid(t, fbx("      when:\n        empty: markdown\n",
		"      when:\n        status: [404]\n")()).Spec)
	if !strings.Contains(onStatus, "when status [404], filling markdown\n") {
		t.Fatalf("the review names no status trigger:\n%s", onStatus)
	}
	if strings.Contains(Review(requireValid(t, exampleWiki).Spec), "Fallback reads") {
		t.Fatal("a spec with no fallback read has a fallback-reads block")
	}
}
