package spec

import (
	"encoding/json"
	"reflect"
	"sort"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/operations"
)

// schemaDocument decodes the embedded schema as a generic JSON object.
func schemaDocument(t *testing.T) map[string]any {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(Schema(), &doc); err != nil {
		t.Fatalf("the embedded schema is not JSON: %v", err)
	}
	return doc
}

// TestSchemaIsADraft202012Document checks the embedded schema: it is a JSON object that declares the 2020-12 dialect and
// holds its named parts under $defs.
func TestSchemaIsADraft202012Document(t *testing.T) {
	doc := schemaDocument(t)
	if doc["$schema"] != "https://json-schema.org/draft/2020-12/schema" {
		t.Fatalf("$schema = %v; want the 2020-12 dialect", doc["$schema"])
	}
	if _, ok := doc["$defs"].(map[string]any); !ok {
		t.Fatal("the schema has no $defs")
	}
}

// TestSchemaVersionIsTheConstantOne checks that the version rule of the schema is the constant of this module.
func TestSchemaVersionIsTheConstantOne(t *testing.T) {
	doc := schemaDocument(t)
	props := doc["properties"].(map[string]any)
	version := props["version"].(map[string]any)
	if got, ok := version["const"].(float64); !ok || int(got) != Version {
		t.Fatalf("version const = %v; want %d", version["const"], Version)
	}
}

// TestSchemaReturnsACopy checks that changing the returned bytes does not change what the next call returns.
func TestSchemaReturnsACopy(t *testing.T) {
	first := Schema()
	want := append([]byte(nil), first...)
	for i := range first {
		first[i] = 'x'
	}
	if string(Schema()) != string(want) {
		t.Fatal("a change to the returned schema reached the next call")
	}
}

// specParamKinds is the list of parameter kinds that a spec may name. A kind that would take a secret, a collection, or
// an event name stays out. The list must match the enum of the schema, and every name must be a catalog kind.
var specParamKinds = []operations.Kind{
	operations.KindID, operations.KindUUID, operations.KindShareID, operations.KindQuery, operations.KindLimit,
	operations.KindOffset, operations.KindInt, operations.KindBool, operations.KindName, operations.KindText,
	operations.KindEnum, operations.KindEnumList, operations.KindIDList, operations.KindUUIDList, operations.KindTime,
	operations.KindURL, operations.KindIndex, operations.KindColor, operations.KindScore, operations.KindReason,
	operations.KindEmail,
}

// TestSchemaParamTypesAreTheCatalogKinds checks that the parameter types of the schema are the parameter kinds of the
// operation catalog, so every adapter validates a value of a spec the same way.
func TestSchemaParamTypesAreTheCatalogKinds(t *testing.T) {
	doc := schemaDocument(t)
	defs := doc["$defs"].(map[string]any)
	paramType, ok := defs["param_type"].(map[string]any)
	if !ok {
		t.Fatal("the schema has no param_type definition")
	}
	enum, ok := paramType["enum"].([]any)
	if !ok {
		t.Fatal("param_type has no enum")
	}
	var got []string
	for _, v := range enum {
		got = append(got, v.(string))
	}
	var want []string
	for _, k := range specParamKinds {
		want = append(want, string(k))
	}
	sort.Strings(got)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("param_type enum = %v; want the catalog kinds %v", got, want)
	}
}

// TestRuleTextFallsBackForAnUnnamedKeyword checks that a failing keyword with no fixed text still gets a rule that
// names no value.
func TestRuleTextFallsBackForAnUnnamedKeyword(t *testing.T) {
	if got := ruleFor("no-such-keyword"); got != fallbackRule {
		t.Fatalf("ruleFor(unknown) = %q; want the fallback", got)
	}
	if got := ruleFor("pattern"); got == fallbackRule || got == "" {
		t.Fatalf("ruleFor(pattern) = %q; want a fixed rule text", got)
	}
}
