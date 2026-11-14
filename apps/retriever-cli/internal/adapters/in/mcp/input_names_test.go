package mcp

import (
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/operations"
)

// TestEveryGeneratedInputMapsBackToItsCatalogName covers finding F4: each snake_case input the model sees is advertised
// in the tool schema, and the call maps it back to the catalog name the binder checks. A name that does not map back
// would reach the binder as an unknown argument, which is refused, so the model could not use the parameter.
func TestEveryGeneratedInputMapsBackToItsCatalogName(t *testing.T) {
	for _, row := range operationTools() {
		props, ok := inputSchema(row)["properties"].(map[string]any)
		if !ok {
			t.Fatalf("%s: the input schema has no properties", row.Tool)
		}
		for _, p := range row.Params {
			wire := snakeCase(p.Name)
			if _, ok := props[wire]; !ok {
				t.Errorf("%s: the schema does not advertise %q for the catalog parameter %q", row.Tool, wire, p.Name)
			}
			args := catalogArgs(row, map[string]any{wire: "value"})
			if _, ok := args[p.Name]; !ok {
				t.Errorf("%s: the call maps %q to %v, not to the catalog name %q", row.Tool, wire, keysOf(args), p.Name)
			}
		}
	}
}

// TestCamelCaseParametersAreSnakeCaseOnTheWire pins the snake_case names of the parameters the model is most likely to
// use across tools: a group source, a status filter, a start date, and the anchor-text flag.
func TestCamelCaseParametersAreSnakeCaseOnTheWire(t *testing.T) {
	cases := map[string]string{
		"groupSource":       "group_source",
		"statusFilter":      "status_filter",
		"startDate":         "start_date",
		"includeAnchorText": "include_anchor_text",
	}
	for catalog, wire := range cases {
		if got := snakeCase(catalog); got != wire {
			t.Errorf("snakeCase(%q) = %q, want %q", catalog, got, wire)
		}
		args := catalogArgs(rowWithParam(t, catalog), map[string]any{wire: "value"})
		if _, ok := args[catalog]; !ok {
			t.Errorf("%q does not map back to the catalog name %q: got %v", wire, catalog, keysOf(args))
		}
	}
}

// rowWithParam returns the first generated tool that declares the catalog parameter, and fails the test when none does.
func rowWithParam(t *testing.T, name string) operations.Row {
	t.Helper()
	for _, row := range operationTools() {
		for _, p := range row.Params {
			if p.Name == name {
				return row
			}
		}
	}
	t.Fatalf("no generated tool declares the parameter %q", name)
	return operations.Row{}
}

// keysOf returns the keys of a map, for a failure message.
func keysOf(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
