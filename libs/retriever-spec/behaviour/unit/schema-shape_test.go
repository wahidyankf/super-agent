package unit_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/operations"
	spec "github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// schemaDocument decodes the schema of the spec as a generic JSON object.
func schemaDocument(t *testing.T) map[string]any {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(spec.Schema(), &doc); err != nil {
		t.Fatalf("the schema is not JSON: %v", err)
	}
	return doc
}

// findEnum returns the first enum array in a decoded JSON value that holds the given text, or nil.
func findEnum(v any, holds string) []any {
	switch x := v.(type) {
	case map[string]any:
		if enum, ok := x["enum"].([]any); ok {
			for _, item := range enum {
				if item == holds {
					return enum
				}
			}
		}
		for _, child := range x {
			if found := findEnum(child, holds); found != nil {
				return found
			}
		}
	case []any:
		for _, child := range x {
			if found := findEnum(child, holds); found != nil {
				return found
			}
		}
	}
	return nil
}

// @scenario specs/libs/retriever-spec/behaviours/schema-shape.feature: A document with a wrong or unknown top-level or operation field is refused at that field
func TestDocumentWithWrongOrUnknownFieldIsRefusedAtThatField(t *testing.T) {
	cases := []struct {
		name string
		doc  string
		path string
	}{
		{"the version is 2", mutate(t, exampleWiki, "version: 1\n", "version: 2\n"), "/version"},
		{"the version is written as text", mutate(t, exampleWiki, "version: 1\n", "version: \"1\"\n"), "/version"},
		{"an unknown top-level field is added", mutate(t, exampleWiki, "kind: example-wiki\n", "extra: x\nkind: example-wiki\n"), "/extra"},
		{"an unknown field is added to the second operation", mutate(t, exampleWiki,
			"    summary: Read one document.\n", "    summary: Read one document.\n    verb: GET\n"), "/operations/1/verb"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			requireRefusedAt(t, c.doc, c.path)
		})
	}
}

// @scenario specs/libs/retriever-spec/behaviours/schema-shape.feature: An operation that lacks a required part is refused at the part it lacks
func TestOperationThatLacksARequiredPartIsRefusedAtThePartItLacks(t *testing.T) {
	cases := []struct {
		name string
		doc  string
		path string
	}{
		{"the second operation has no summary", mutate(t, exampleWiki, "    summary: Read one document.\n", ""), "/operations/1/summary"},
		{"the fourth operation has no example", mutate(t, exampleWiki, docsDeleteExample, ""), "/operations/3/examples"},
		{"the spec has no operation at all", strings.Replace(header(), "operations:\n", "operations: []\n", 1), "/operations"},
		{"a header of the second operation is a number, not text", mutate(t, exampleWiki,
			"    path: /docs/{id}\n    params:", "    path: /docs/{id}\n    headers:\n      X-Tenant: 5\n    params:"),
			"/operations/1/headers/X-Tenant"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			requireRefusedAt(t, c.doc, c.path)
		})
	}
}

// @scenario specs/libs/retriever-spec/behaviours/schema-shape.feature: A value outside its closed list is refused at the field that holds it
func TestValueOutsideItsClosedListIsRefusedAtTheField(t *testing.T) {
	cases := []struct {
		name string
		doc  string
		path string
	}{
		{"a response field uses a converter the spec does not know", mutate(t, exampleWiki,
			"convert: markdown-from-html", "convert: nope-convert"), "/operations/1/response/fields/body/convert"},
		{"an enum parameter has no values", mutate(t, exampleWiki,
			"      - name: limit\n        type: Limit\n        in: query\n        required: false\n        default: 20",
			"      - name: limit\n        type: Enum\n        in: query\n        required: false\n        values: []"),
			"/operations/0/params/1/values"},
		{"a parameter kind is not in the operation catalog", mutate(t, exampleWiki,
			"      - name: limit\n        type: Limit", "      - name: limit\n        type: Bogus"), "/operations/0/params/1/type"},
		{"an example reply has a status of 99, below 100", mutate(t, exampleWiki,
			"status: 204", "status: 99"), "/operations/3/examples/0/response/status"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			requireRefusedAt(t, c.doc, c.path)
		})
	}
}

// @scenario specs/libs/retriever-spec/behaviours/schema-shape.feature: An auth block whose fields do not match its type is refused
func TestAuthBlockWhoseFieldsDoNotMatchItsTypeIsRefused(t *testing.T) {
	const bearer = "auth:\n  type: bearer\n  credential_env: EXAMPLE_WIKI_API_KEY\n"
	cases := []struct {
		name string
		doc  string
		path string
	}{
		{"basic with no identity variable", mutate(t, exampleWiki, bearer,
			"auth:\n  type: basic\n  credential_env: EXAMPLE_WIKI_API_KEY\n"), "/auth/identity_env"},
		{"bearer that also names an identity variable", mutate(t, exampleWiki, bearer,
			"auth:\n  type: bearer\n  credential_env: EXAMPLE_WIKI_API_KEY\n  identity_env: EXAMPLE_WIKI_EMAIL\n"), "/auth"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			requireRefusedAt(t, c.doc, c.path)
		})
	}
}

// @scenario specs/libs/retriever-spec/behaviours/schema-shape.feature: A paged read without a stop rule, or with a cap above the limit, is refused
func TestPagedReadWithoutStopRuleOrWithCapAboveLimitIsRefused(t *testing.T) {
	cases := []struct {
		name string
		doc  string
		path string
	}{
		{"the first operation pages with no maximum page count", mutate(t, exampleWiki,
			"      max_pages: 5\n      cursor_param", "      cursor_param"), "/operations/0/pagination/max_pages"},
		{"a limit asks for 21 pages, above the cap of 20", mutate(t, exampleWiki,
			"limits:\n  timeout_seconds: 20\n  max_pages: 5\n", "limits:\n  timeout_seconds: 20\n  max_pages: 21\n"), "/limits/max_pages"},
		{"a limit asks for a timeout above 30 seconds", mutate(t, exampleWiki,
			"timeout_seconds: 20", "timeout_seconds: 31"), "/limits/timeout_seconds"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			requireRefusedAt(t, c.doc, c.path)
		})
	}
}

// @scenario specs/libs/retriever-spec/behaviours/schema-shape.feature: A command-line name that is not a lower-case slug is refused at the operation
func TestCommandLineNameThatIsNotALowerCaseSlugIsRefused(t *testing.T) {
	doc := mutate(t, exampleWiki, "    path: /docs\n    params:", "    path: /docs\n    cli: Not A Slug\n    params:")
	requireRefusedAt(t, doc, "/operations/0/cli")
}

// @scenario specs/libs/retriever-spec/behaviours/schema-shape.feature: The schema of the spec is a JSON Schema document of draft 2020-12
func TestSchemaIsAJSONSchemaDocumentOfDraft202012(t *testing.T) {
	doc := schemaDocument(t)
	if got := doc["$schema"]; got != "https://json-schema.org/draft/2020-12/schema" {
		t.Fatalf("$schema = %v; want the draft 2020-12 dialect", got)
	}
}

// @scenario specs/libs/retriever-spec/behaviours/schema-shape.feature: The schema of the spec fixes the version to the number 1
func TestSchemaFixesTheVersionToTheNumberOne(t *testing.T) {
	doc := schemaDocument(t)
	props, _ := doc["properties"].(map[string]any)
	version, _ := props["version"].(map[string]any)
	if version["const"] != float64(1) {
		t.Fatalf("version schema = %v; want const 1", version)
	}
	requireAccepted(t, exampleWiki)
	requireRefusedAt(t, mutate(t, exampleWiki, "version: 1\n", "version: 2\n"), "/version")
}

// @scenario specs/libs/retriever-spec/behaviours/schema-shape.feature: Changing the bytes of one schema copy does not change what the next read returns
func TestChangingOneSchemaCopyDoesNotChangeTheNextRead(t *testing.T) {
	original := string(spec.Schema())
	held := spec.Schema()
	for i := range held {
		held[i] = 'X'
	}
	if again := string(spec.Schema()); again != original {
		t.Fatal("the next read of the schema returns the bytes that a caller overwrote")
	}
}

// @scenario specs/libs/retriever-spec/behaviours/schema-shape.feature: The schema lists the parameter kinds of the operation catalog
func TestSchemaListsTheParameterKindsOfTheOperationCatalog(t *testing.T) {
	// CollectionID is held only by the parameter kind list. The path parameter types hold no such kind, so the search
	// cannot land on that smaller list whatever order the map walk visits it in.
	enum := findEnum(schemaDocument(t), "CollectionID")
	if enum == nil {
		t.Fatal("the schema holds no parameter kind list")
	}
	allowed := make(map[string]bool, len(enum))
	for _, item := range enum {
		allowed[item.(string)] = true
	}
	// The catalog exports its rows but no list of its kinds, so this checks every kind that a row uses.
	for _, row := range operations.BuiltinCatalog().Rows() {
		for _, p := range row.Params {
			if !allowed[string(p.Kind)] {
				t.Fatalf("catalog kind %q is not allowed by the schema", p.Kind)
			}
		}
	}
}
