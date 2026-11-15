package spec

import (
	"encoding/json"
	"reflect"
	"sort"
	"testing"
)

// specFieldNames is the closed list of the field names of connector spec version 1, as ADR 001 and ADR 002 name them.
// Each name belongs to a feature: the ADR 001 fields, and the E1 field read_over_post of ADR 002. A new field needs a
// feature of an ADR and a line here, so the language cannot grow by habit (threat L11).
var specFieldNames = []string{
	// ADR 001: the document and its parts.
	"access", "auth", "body", "cli", "convert", "credential_env", "cursor_param", "default", "endpoint", "errors", "examples",
	"expect", "field", "fields", "format", "header_name", "headers", "help", "identity_env", "in", "items", "kind",
	"limit_param", "limits", "markdown", "max", "max_pages", "max_reply_bytes", "mcp", "method", "min", "name",
	"next_pointer", "offset_param", "operation", "operations", "params", "pagination", "path", "pointer", "request",
	"required", "resolve", "response", "role", "status", "style", "summary", "timeout_seconds", "type", "values", "version",
	"base_path", "host_suffix", "id",
	// ADR 002, E1: POST reads.
	"read_over_post",
	// ADR 002, E3: required and validated reply fields.
	"format",
	// ADR 002, E4: nested records with an allow-list.
	"object",
	// ADR 002, E5: the first item or not found.
	"first",
}

// collectPropertyNames gathers the keys of every "properties" object of a decoded schema, at any depth.
func collectPropertyNames(v any, out map[string]bool) {
	switch x := v.(type) {
	case map[string]any:
		for key, child := range x {
			if key == "properties" {
				if props, ok := child.(map[string]any); ok {
					for name := range props {
						out[name] = true
					}
				}
			}
			collectPropertyNames(child, out)
		}
	case []any:
		for _, child := range x {
			collectPropertyNames(child, out)
		}
	}
}

// TestSchemaKeywordLock fails when the schema names a field that no feature of the ADRs names, or when a named field is
// missing from the schema. The list is closed on both sides, so adding a field is one reviewed change to this list.
func TestSchemaKeywordLock(t *testing.T) {
	var doc any
	if err := json.Unmarshal(Schema(), &doc); err != nil {
		t.Fatalf("the embedded schema is not JSON: %v", err)
	}
	got := map[string]bool{}
	collectPropertyNames(doc, got)
	var names []string
	for name := range got {
		names = append(names, name)
	}
	sort.Strings(names)
	want := append([]string(nil), specFieldNames...)
	sort.Strings(want)
	want = uniqueSorted(want)
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("the schema field names differ from the closed list.\nschema: %v\nlist:   %v", names, want)
	}
}

// uniqueSorted drops the repeated names of a sorted list.
func uniqueSorted(in []string) []string {
	var out []string
	for i, s := range in {
		if i == 0 || s != in[i-1] {
			out = append(out, s)
		}
	}
	return out
}

// TestExistingSpecsStillValidate runs the current corpus through the validation: every valid document of the corpus,
// including the bundled example, stays valid, and each refused document of the table stays refused. A change to the
// language that moves one of these verdicts fails here first.
func TestExistingSpecsStillValidate(t *testing.T) {
	for _, v := range validCorpus {
		t.Run("valid/"+v.name, func(t *testing.T) {
			requireValid(t, v.doc())
		})
	}
	requireValid(t, exampleWiki)
	for _, f := range faults {
		t.Run("refused/"+f.name, func(t *testing.T) {
			if Validate([]byte(f.doc())).Valid() {
				t.Fatal("a refused document of the table is valid")
			}
		})
	}
}
