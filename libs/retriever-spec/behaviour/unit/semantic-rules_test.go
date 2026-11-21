package unit_test

import (
	"testing"
)

// @scenario specs/libs/retriever-spec/behaviours/semantic-rules.feature: An operation id that does not fit its kind or repeats another is refused at that id
func TestOperationIDThatDoesNotFitItsKindOrRepeatsAnotherIsRefused(t *testing.T) {
	cases := []struct {
		name string
		doc  string
		path string
	}{
		{"the first operation id starts with a kind other than the spec kind", mutate(t, exampleWiki,
			"  - id: example-wiki.docs.list", "  - id: other.docs.list"), "/operations/0/id"},
		{"the second operation id repeats the first operation id", mutate(t, exampleWiki,
			"  - id: example-wiki.docs.get", "  - id: example-wiki.docs.list"), "/operations/1/id"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			requireRefusedAt(t, c.doc, c.path)
		})
	}
}

// @scenario specs/libs/retriever-spec/behaviours/semantic-rules.feature: A path, body, or header slot that names no declared parameter is refused at the slot
func TestSlotThatNamesNoDeclaredParameterIsRefusedAtTheSlot(t *testing.T) {
	cases := []struct {
		name string
		doc  string
		path string
	}{
		{"the path of the second operation has a slot with no declared parameter", mutate(t, exampleWiki,
			"    path: /docs/{id}\n    params:", "    path: /docs/{doc}\n    params:"), "/operations/1/path"},
		{"a body slot of the third operation names no body parameter", mutate(t, exampleWiki,
			`      title: "{{title}}"`, `      title: "{{nope}}"`), "/operations/2/body/title"},
		{"a body slot inside a list names no body parameter", mutate(t, exampleWiki,
			"      text: \"{{text}}\"\n", "      text: \"{{text}}\"\n      labels:\n        - plain\n        - \"{{nope}}\"\n"),
			"/operations/2/body/labels/1"},
		{"a header slot of the second operation names no parameter", mutate(t, exampleWiki,
			"    path: /docs/{id}\n    params:", "    path: /docs/{id}\n    headers:\n      X-Tenant: \"{tenant}\"\n    params:"),
			"/operations/1/headers/X-Tenant"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			requireRefusedAt(t, c.doc, c.path)
		})
	}
}

// @scenario specs/libs/retriever-spec/behaviours/semantic-rules.feature: A declared path parameter that the path does not use is refused at that parameter
func TestDeclaredPathParameterThatThePathDoesNotUseIsRefused(t *testing.T) {
	doc := mutate(t, exampleWiki, "    path: /docs/{id}\n    params:", "    path: /docs\n    params:")
	requireRefusedAt(t, doc, "/operations/1/params/0/name")
}

// @scenario specs/libs/retriever-spec/behaviours/semantic-rules.feature: Two parameters of one operation that share a name are refused at the second name
func TestTwoParametersOfOneOperationThatShareANameAreRefusedAtTheSecond(t *testing.T) {
	doc := mutate(t, exampleWiki, "      - name: limit\n        type: Limit", "      - name: cursor\n        type: Limit")
	requireRefusedAt(t, doc, "/operations/0/params/1/name")
}

// @scenario specs/libs/retriever-spec/behaviours/semantic-rules.feature: A cursor parameter name that is not a query parameter of the operation is refused
func TestCursorParameterNameThatIsNotAQueryParameterIsRefused(t *testing.T) {
	doc := mutate(t, exampleWiki, "      cursor_param: cursor\n", "      cursor_param: missing\n")
	requireRefusedAt(t, doc, "/operations/0/pagination/cursor_param")
}

// @scenario specs/libs/retriever-spec/behaviours/semantic-rules.feature: A name lookup that does not point at a read operation of the same spec is refused
func TestNameLookupThatDoesNotPointAtAReadOfTheSameSpecIsRefused(t *testing.T) {
	cases := []struct {
		name   string
		target string
	}{
		{"an operation id that the spec does not declare", "example-wiki.docs.none"},
		{"a write operation of the same spec", "example-wiki.docs.create"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			doc := mutate(t, exampleWiki, "        required: true\n    response:\n      fields:\n        title:",
				"        required: true\n        resolve:\n          operation: "+c.target+"\n          field: id\n"+
					"    response:\n      fields:\n        title:")
			requireRefusedAt(t, doc, "/operations/1/params/0/resolve/operation")
		})
	}
}

// @scenario specs/libs/retriever-spec/behaviours/semantic-rules.feature: A markdown output that names no field of the operation is refused at the markdown field
func TestMarkdownOutputThatNamesNoFieldOfTheOperationIsRefused(t *testing.T) {
	doc := mutate(t, exampleWiki, "      markdown: body", "      markdown: nope")
	requireRefusedAt(t, doc, "/operations/1/response/markdown")
}
