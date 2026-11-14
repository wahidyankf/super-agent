package spec

import (
	"reflect"
	"sort"
	"testing"
)

// validCorpus holds documents that every rule accepts. Each one varies a field the fault table does not touch, so the
// agreement test covers the accepted forms as well as the refused ones.
var validCorpus = []struct {
	name string
	doc  func() string
}{
	{"the minimal spec", func() string { return minimalSpec }},
	{"the example spec", func() string { return exampleWiki }},
	{"a switched-off command and a custom MCP name", fx("    path: /docs\n    params:", "    path: /docs\n    cli: false\n    mcp: docs-list\n    params:")},
	{"offset pagination with two query names", fx(
		"      style: cursor\n      max_pages: 5\n      cursor_param: cursor\n      next_pointer: /next_cursor",
		"      style: offset\n      max_pages: 5\n      limit_param: limit\n      offset_param: cursor")},
	{"basic auth with an account variable", fx("  type: bearer", "  type: basic",
		"  credential_env: EXAMPLE_WIKI_API_KEY", "  credential_env: EXAMPLE_WIKI_API_KEY\n  identity_env: EXAMPLE_WIKI_EMAIL")},
	{"header auth with a named header", fx("  type: bearer", "  type: header",
		"  credential_env: EXAMPLE_WIKI_API_KEY", "  credential_env: EXAMPLE_WIKI_API_KEY\n  header_name: X-Api-Token")},
	{"a constant header", fx("    response:\n      fields:\n        title:",
		"    headers:\n      X-Client: example-cli\n    response:\n      fields:\n        title:")},
	{"an enum parameter with its values", fx("        type: Limit\n        in: query\n        required: false\n        default: 20",
		"        type: Enum\n        in: query\n        required: false\n        values: [asc, desc]")},
	{"a scalar of each plain kind in an example", fx("            next_cursor: null",
		"            next_cursor: null\n            total: 2\n            more: false")},
	{"a header slot that names a parameter", fx("    response:\n      fields:\n        title:",
		"    headers:\n      X-Tenant: \"{id}\"\n    response:\n      fields:\n        title:")},
	{"a constant body value and a list in a body", fx("      text: \"{{text}}\"",
		"      text: \"{{text}}\"\n      format: markdown\n      labels: [\"one\", \"{{text}}\"]")},
	{"a reply size limit", fx("  max_pages: 5\noperations:", "  max_pages: 5\n  max_reply_bytes: 1048576\noperations:")},
	{"a resolve that names a read of its own field", fx("    params:\n      - name: id\n        type: ID\n        in: path",
		"    params:\n      - name: id\n        type: ID\n        in: path\n        resolve:\n          operation: example-wiki.docs.list\n          field: id")},
}

// positions returns the sorted, de-duplicated paths of a set of findings.
func positions(fs []Finding) []string {
	seen := map[string]bool{}
	var out []string
	for _, f := range fs {
		if !seen[f.Path] {
			seen[f.Path] = true
			out = append(out, f.Path)
		}
	}
	sort.Strings(out)
	return out
}

// TestSchemaAndGoAgreeOnTheCorpus runs the JSON Schema file and the Go validation over the same good and bad documents.
// A valid document passes both. A refused document is refused by the layer its row names: the YAML reader, the schema,
// or the Go rule that a schema cannot state. For a schema row, the Go findings name the same paths as the schema.
func TestSchemaAndGoAgreeOnTheCorpus(t *testing.T) {
	for _, v := range validCorpus {
		t.Run("valid/"+v.name, func(t *testing.T) {
			doc := []byte(v.doc())
			value, parseFindings := parseDocument(doc)
			if len(parseFindings) != 0 {
				t.Fatalf("the YAML reader refuses a valid document: %v", parseFindings)
			}
			if fs := checkSchema(value); len(fs) != 0 {
				t.Fatalf("the JSON Schema refuses a valid document: %v", fs)
			}
			if r := Validate(doc); !r.Valid() {
				t.Fatalf("the Go validation refuses a valid document:\n%s", allText(r))
			}
		})
	}
	for _, f := range faults {
		t.Run("refused/"+f.name, func(t *testing.T) {
			doc := []byte(f.doc())
			value, parseFindings := parseDocument(doc)
			r := Validate(doc)
			if r.Valid() {
				t.Fatal("the Go validation accepts a refused document")
			}
			switch f.layer {
			case layerParse:
				if len(parseFindings) == 0 {
					t.Fatal("the row claims the YAML reader refuses it, and the reader accepts it")
				}
			case layerSchema:
				if len(parseFindings) != 0 {
					t.Fatalf("the YAML reader refuses a schema row: %v", parseFindings)
				}
				schemaFindings := checkSchema(value)
				if len(schemaFindings) == 0 {
					t.Fatal("the row claims the JSON Schema refuses it, and the schema accepts it")
				}
				if got, want := positions(r.Findings), positions(schemaFindings); !reflect.DeepEqual(got, want) {
					t.Fatalf("the Go findings %v do not name the schema's paths %v", got, want)
				}
			case layerSemantic:
				if len(parseFindings) != 0 {
					t.Fatalf("the YAML reader refuses a semantic row: %v", parseFindings)
				}
				if fs := checkSchema(value); len(fs) != 0 {
					t.Fatalf("the JSON Schema already refuses this row; relabel it as a schema row: %v", fs)
				}
			default:
				t.Fatalf("unknown layer %q", f.layer)
			}
		})
	}
}
