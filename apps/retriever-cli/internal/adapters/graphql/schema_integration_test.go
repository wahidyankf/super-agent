//go:build integration

package graphql

import (
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"os"
	"slices"
	"strings"
	"testing"
)

// schemaSource reads schema.graphqls as the code generator reads it, and parses it.
func schemaSource(t *testing.T) (string, *ast.Schema) {
	t.Helper()
	data, err := os.ReadFile("schema.graphqls")
	if err != nil {
		t.Fatalf("read schema.graphqls: %v", err)
	}
	sch, gerr := gqlparser.LoadSchema(&ast.Source{Name: "schema.graphqls", Input: string(data)})
	if gerr != nil {
		t.Fatalf("LoadSchema: %v", gerr)
	}
	return string(data), sch
}

// TestSchemaHasNoSourceDefault checks that no operation names an outline default for its source argument. A default
// would pick a source silently when two outline sources exist, so an omitted argument must reach the default rule.
func TestSchemaHasNoSourceDefault(t *testing.T) {
	text, sch := schemaSource(t)
	if strings.Contains(text, `= "outline"`) {
		t.Errorf(`schema.graphqls holds a "= \"outline\"" default`)
	}
	for _, def := range []*ast.Definition{sch.Query} {
		for _, field := range def.Fields {
			arg := field.Arguments.ForName("source")
			if arg != nil && arg.DefaultValue != nil {
				t.Errorf("%s.%s has a default for source: %s", def.Name, field.Name, arg.DefaultValue.String())
			}
		}
	}
}

// TestServedSchemaHasNoMutation checks that the served API is read-only. Exports write files on the local machine
// through the CLI's export sink, so the served API declares no mutation at all.
func TestServedSchemaHasNoMutation(t *testing.T) {
	_, sch := schemaSource(t)
	if sch.Mutation != nil {
		t.Fatalf("schema declares a Mutation type with %d fields; the served API must be read-only", len(sch.Mutation.Fields))
	}
}

// TestSourceTypeExposesOnlyIDKindAndTags checks that the Source type has no endpoint and no credential name.
func TestSourceTypeExposesOnlyIDKindAndTags(t *testing.T) {
	_, sch := schemaSource(t)
	def := sch.Types["Source"]
	if def == nil {
		t.Fatal("schema has no Source type")
	}
	var got []string
	for _, field := range def.Fields {
		got = append(got, field.Name)
	}
	if want := []string{"id", "kind", "tags"}; !slices.Equal(got, want) {
		t.Fatalf("Source fields = %v, want %v", got, want)
	}
}

// TestSourcesQueryTakesAKindAndAListOfTags checks the signature of the sources query: an optional kind, an optional
// list of tags, and a non-null list of Source.
func TestSourcesQueryTakesAKindAndAListOfTags(t *testing.T) {
	_, sch := schemaSource(t)
	field := sch.Query.Fields.ForName("sources")
	if field == nil {
		t.Fatal("schema has no sources query")
	}
	if got := field.Type.String(); got != "[Source!]!" {
		t.Errorf("sources returns %s, want [Source!]!", got)
	}
	wantArgs := map[string]string{"kind": "String", "tags": "[String!]"}
	for name, typ := range wantArgs {
		arg := field.Arguments.ForName(name)
		if arg == nil {
			t.Errorf("sources has no %s argument", name)
			continue
		}
		if got := arg.Type.String(); got != typ {
			t.Errorf("sources %s is %s, want %s", name, got, typ)
		}
	}
}
