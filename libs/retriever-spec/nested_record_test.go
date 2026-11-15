package spec

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// nestedRecordOp is a whoami read whose reply field is one level of object with two declared sub-fields. It is
// appended to the example as operation 4, so the paths of the rows above keep their indexes, as the E1 postReadOp,
// the E2 body paging ops, and the E3 requiredFieldOp do. The example carries no e-mail: the engine plants one in the
// reply instead.
const nestedRecordOp = `  - id: example-wiki.whoami.get
    summary: Read the account user.
    access: read
    method: GET
    path: /whoami
    response:
      fields:
        user:
          object:
            id:
              pointer: /data/user/id
            name:
              pointer: /data/user/name
    examples:
      - request: {}
        response:
          status: 200
          body:
            data:
              user:
                id: u-1
                name: Ada
        expect:
          user: {}
`

// nfx returns the bundled example spec with the nested-record read appended as operation 4, with the given (old,
// repl) pairs applied, for a fault row.
func nfx(pairs ...string) func() string {
	return func() string {
		return mutateAll(mutateAll(exampleWiki, "        expect: {}", "        expect: {}\n"+nestedRecordOp), pairs...)
	}
}

// nestedWithFields returns the example with the nested-record read whose object holds n sub-fields named f00 and up,
// each reading its own pointer, for the count-limit rows.
func nestedWithFields(n int) string {
	var b strings.Builder
	b.WriteString("          object:\n")
	for i := 0; i < n; i++ {
		name := fmt.Sprintf("f%02d", i)
		b.WriteString("            " + name + ":\n              pointer: /data/user/" + name + "\n")
	}
	return mutateAll(mutateAll(exampleWiki, "        expect: {}", "        expect: {}\n"+nestedRecordOp),
		"          object:\n            id:\n              pointer: /data/user/id\n            name:\n              pointer: /data/user/name", strings.TrimSuffix(b.String(), "\n"))
}

// TestObjectKeywordMapsSubFields checks the E4 schema keyword: a field with an object of two pointer sub-fields loads,
// and the model keeps the object with both pointers. It reads the model as JSON, as the E3 format test does, so the
// test fails on the verdict before the fix, not on the shape of the model.
func TestObjectKeywordMapsSubFields(t *testing.T) {
	r := requireValid(t, mutateAll(exampleWiki, "        expect: {}", "        expect: {}\n"+nestedRecordOp))
	raw, err := json.Marshal(r.Spec.Operations[4].Response.Fields)
	if err != nil {
		t.Fatalf("the model does not encode: %v", err)
	}
	for _, want := range []string{`"user":{"pointer":"","object":{"id":{"pointer":"/data/user/id"}`, `"name":{"pointer":"/data/user/name"}`} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("the model drops %s: %s", want, raw)
		}
	}
}

// TestNestedObjectCountLimit checks the E4 count limit: an object of 32 sub-fields loads, and one of 33 is refused at
// its object field.
func TestNestedObjectCountLimit(t *testing.T) {
	requireValid(t, nestedWithFields(32))
	runFaults(t, rowsWithThreat("L5"))
}

// soundObject returns a nested record field of two pointer sub-fields, which the Go rule accepts.
func soundObject() Field {
	return Field{Object: map[string]Field{"id": {Pointer: "/data/user/id"}, "name": {Pointer: "/data/user/name"}}}
}

// oversizedObject returns a nested record field of 33 pointer sub-fields, which the Go rule refuses.
func oversizedObject() Field {
	obj := map[string]Field{}
	for i := 0; i <= maxObjectFields; i++ {
		name := fmt.Sprintf("f%02d", i)
		obj[name] = Field{Pointer: "/data/user/" + name}
	}
	return Field{Object: obj}
}

// TestCheckObjectFieldsRefusesHandBuiltModels checks the Go rule on models that bypass the schema: an object field
// with a scalar keyword, an empty or oversized object, and a sub-field with its own object are refused, each at its
// field path with a fixed rule. A scalar field and a sound object pass with no finding.
func TestCheckObjectFieldsRefusesHandBuiltModels(t *testing.T) {
	cases := []struct {
		name   string
		fields map[string]Field
		path   string
		rule   string
	}{
		{"a scalar field", map[string]Field{"id": {Pointer: "/id"}}, "", ""},
		{"a sound object", map[string]Field{"user": soundObject()}, "", ""},
		{"an object field with a pointer", map[string]Field{"user": {Pointer: "/data/user", Object: soundObject().Object}},
			"/operations/0/response/fields/user", ruleObjectScalar},
		{"an object field with required", map[string]Field{"user": {Required: true, Object: soundObject().Object}},
			"/operations/0/response/fields/user", ruleObjectScalar},
		{"an empty object", map[string]Field{"user": {Object: map[string]Field{}}},
			"/operations/0/response/fields/user/object", ruleObjectCount},
		{"an oversized object", map[string]Field{"user": oversizedObject()},
			"/operations/0/response/fields/user/object", ruleObjectCount},
		{"a sub-field with its own object", map[string]Field{"user": {Object: map[string]Field{
			"id": {Object: map[string]Field{"leaf": {Pointer: "/data/user/id"}}}}}},
			"/operations/0/response/fields/user/object/id/object", ruleObjectDepth},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := &Spec{Operations: []Operation{{ID: "example-wiki.whoami.get", Response: Response{Fields: c.fields}}}}
			fs := CheckObjectFields(s)
			if c.path == "" {
				if len(fs) != 0 {
					t.Fatalf("a sound model has findings: %v", fs)
				}
				return
			}
			found := false
			for _, f := range fs {
				if f.Code != CodeSpecInvalid {
					t.Fatalf("finding code = %q; want %q", f.Code, CodeSpecInvalid)
				}
				if f.Path == c.path && f.Rule == c.rule {
					found = true
				}
			}
			if !found {
				t.Fatalf("no finding at %q with rule %q; findings: %v", c.path, c.rule, fs)
			}
		})
	}
}

// TestReviewListsNestedObjects checks the review of a read with a nested record field: its own block names the
// operation, the object field, and each sub-field. A spec with no object field has no such block, so its review keeps
// its text.
func TestReviewListsNestedObjects(t *testing.T) {
	review := Review(requireValid(t, mutateAll(exampleWiki, "        expect: {}", "        expect: {}\n"+nestedRecordOp)).Spec)
	block := "Nested objects: 1\n1. example-wiki.whoami.get: user with id, name\n"
	if !strings.Contains(review, block) {
		t.Fatalf("the review has no nested-objects block:\n%s", review)
	}
	if strings.Contains(Review(requireValid(t, exampleWiki).Spec), "Nested objects") {
		t.Fatal("a spec with no object field has a nested-objects block")
	}
}
