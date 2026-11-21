package unit_test

import (
	"fmt"
	"testing"
)

// fallbackMetaTemplate is a single-object read at index four with one fallback. The three values are the target
// operation, the closed trigger as a flow mapping, and the fill list as a flow sequence.
const fallbackMetaTemplate = `  - id: example-wiki.docs.meta
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
        body:
          pointer: /body
    fallback:
      operation: %s
      when: %s
      fill: %s
    examples:
      - request:
          id: doc-1
        response:
          status: 200
          body:
            title: First document
        expect: {}
`

// fullOp is a single-object read with the same parameter as the metadata read, so it is a valid fallback target.
const fullOp = `  - id: example-wiki.docs.full
    summary: Read the full text of one document.
    access: read
    method: GET
    path: /docs/{id}/full
    params:
      - name: id
        type: ID
        in: path
        required: true
    response:
      fields:
        title:
          pointer: /title
        body:
          pointer: /body
    examples:
      - request:
          id: doc-1
        response:
          status: 200
          body:
            title: First document
            body: Hello
        expect: {}
`

// chainOp is a single-object read that declares a fallback of its own, so it may not be the target of another.
const chainOp = `  - id: example-wiki.docs.chain
    summary: Read a document through a chain.
    access: read
    method: GET
    path: /docs/{id}/chain
    params:
      - name: id
        type: ID
        in: path
        required: true
    response:
      fields:
        title:
          pointer: /title
        body:
          pointer: /body
    fallback:
      operation: example-wiki.docs.get
      when: {status: [404]}
      fill: [body]
    examples:
      - request:
          id: doc-1
        response:
          status: 200
          body:
            title: First document
            body: Hello
        expect: {}
`

// meta returns the metadata read at index four with the given target, trigger, and fill.
func meta(target, when, fill string) string {
	return fmt.Sprintf(fallbackMetaTemplate, target, when, fill)
}

// @scenario specs/libs/retriever-spec/behaviours/fallbacks.feature: A fallback with one closed trigger and a fill list is accepted
func TestFallbackWithOneClosedTriggerAndFillListIsAccepted(t *testing.T) {
	cases := []struct {
		name, when string
	}{
		{"the markdown field is empty", "{empty: body}"},
		{"the first reply has status 404", "{status: [404]}"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			requireAccepted(t, withOps(meta("example-wiki.docs.full", c.when, "[title]"), fullOp))
		})
	}
}

// @scenario specs/libs/retriever-spec/behaviours/fallbacks.feature: A fallback that names two triggers is refused at its trigger
func TestFallbackNamingTwoTriggersIsRefusedAtItsTrigger(t *testing.T) {
	doc := withOps(meta("example-wiki.docs.full", "{empty: body, status: [404]}", "[title]"), fullOp)
	requireRefusedAt(t, doc, "/operations/4/fallback/when")
}

// @scenario specs/libs/retriever-spec/behaviours/fallbacks.feature: A fallback that fills no field is refused at its fill list
func TestFallbackThatFillsNoFieldIsRefusedAtItsFillList(t *testing.T) {
	doc := withOps(meta("example-wiki.docs.full", "{status: [404]}", "[]"), fullOp)
	requireRefusedAt(t, doc, "/operations/4/fallback/fill")
}

// @scenario specs/libs/retriever-spec/behaviours/fallbacks.feature: A fallback trigger that lists a status outside the HTTP range is refused at the status
func TestFallbackTriggerListingStatusOutsideHTTPRangeIsRefusedAtTheStatus(t *testing.T) {
	doc := withOps(meta("example-wiki.docs.full", "{status: [99]}", "[title]"), fullOp)
	requireRefusedAt(t, doc, "/operations/4/fallback/when/status/0")
}

// @scenario specs/libs/retriever-spec/behaviours/fallbacks.feature: A fallback whose own target has a fallback is refused as a chain
func TestFallbackWhoseOwnTargetHasAFallbackIsRefusedAsAChain(t *testing.T) {
	doc := withOps(meta("example-wiki.docs.chain", "{status: [404]}", "[title]"), chainOp)
	requireRefusedAt(t, doc, "/operations/4/fallback/operation")
}

// @scenario specs/libs/retriever-spec/behaviours/fallbacks.feature: A fallback whose target is not a read of the same spec is refused at the target
func TestFallbackWhoseTargetIsNotAReadOfTheSameSpecIsRefusedAtTheTarget(t *testing.T) {
	// The target of this row declares a parameter that the metadata read does not declare, so the metadata read has no
	// path parameter here.
	paramlessMeta := mutate(t, withOps(meta("example-wiki.docs.get", "{status: [404]}", "[title]")),
		"    path: /docs/{id}/meta\n    params:\n      - name: id\n        type: ID\n        in: path\n        required: true\n",
		"    path: /docs/meta\n")
	cases := []struct {
		name, doc string
	}{
		{"a write operation", withOps(meta("example-wiki.docs.delete", "{status: [404]}", "[title]"))},
		{"an operation id that the spec does not declare", withOps(meta("example-wiki.docs.none", "{status: [404]}", "[title]"))},
		{"a read that declares a parameter the first read does not declare", paramlessMeta},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			requireRefusedAt(t, c.doc, "/operations/4/fallback/operation")
		})
	}
}

// @scenario specs/libs/retriever-spec/behaviours/fallbacks.feature: A field that names a second pointer for the same value is accepted
func TestFieldThatNamesASecondPointerForTheSameValueIsAccepted(t *testing.T) {
	doc := mutate(t, exampleWiki, "        id:\n          pointer: /id\n        title:",
		"        id:\n          pointer: /data/id\n          fallback_pointer: /id\n        title:")
	requireAccepted(t, doc)
}
