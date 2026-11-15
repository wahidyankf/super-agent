package engine_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine/replay"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// nestedRecordDoc is a whoami-style read whose reply field is one level of object with two declared sub-fields. The
// example carries no e-mail: the tests plant one in the reply instead, so the plant is never part of the spec.
const nestedRecordDoc = `version: 1
kind: example-wiki
summary: Reads the account user as a nested record.
endpoint:
  base_path: /v1
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
operations:
  - id: example-wiki.whoami.get
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

// nestedObjectModel is a hand-built model whose object holds another object. It bypasses the schema, so only the Go
// rule and the engine constructor can refuse it.
const nestedObjectModel = `{"version":1,"kind":"example-wiki","summary":"Reads the account user.","endpoint":{"base_path":"/v1","host_suffix":[".example.test"]},"auth":{"type":"bearer","credential_env":"EXAMPLE_WIKI_API_KEY"},"operations":[{"id":"example-wiki.whoami.get","summary":"Read the account user.","access":"read","method":"GET","path":"/whoami","response":{"fields":{"user":{"object":{"id":{"object":{"leaf":{"pointer":"/data/user/id"}}}}}}},"examples":[{"request":{},"response":{"status":200},"expect":{"user":{}}}]}]}`

// plainRecord returns the fields of a record as plain values, with nested records as maps, so a test can marshal the
// whole result and prove that a planted value reaches none of it.
func plainRecord(rec domain.Record) map[string]any {
	out := map[string]any{}
	for _, f := range rec.Fields() {
		if nested, ok := f.Value.(domain.Record); ok {
			out[f.Name] = plainRecord(nested)
			continue
		}
		out[f.Name] = f.Value
	}
	return out
}

// TestNestedRecordWithholdsUndeclaredFields plants an e-mail address in the reply of a nested record and proves that
// it reaches no output, no error, and no log. The record holds the two declared sub-fields only. The engine holds no
// logger (no log or slog call in the package), so there is no log line it could reach; the test proves the output by
// marshaling the whole record, and the error by reading its text.
func TestNestedRecordWithholdsUndeclaredFields(t *testing.T) {
	const planted = "withheld-person@example.test"
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, nestedRecordDoc))
	stub.Script("GET", "/v1/whoami", page(`{"data":{"user":{"id":"u-1","name":"Ada","email":"`+planted+`"}}}`))
	rec, err := eng.Read(context.Background(), "example-wiki.whoami.get", domain.NewArgs(nil))
	if err != nil {
		t.Fatalf("a reply with an undeclared field was refused: %v", err)
	}
	if got := rec.Rec("user").Str("id"); got != "u-1" {
		t.Fatalf("user.id = %q, want u-1", got)
	}
	if got := rec.Rec("user").Str("name"); got != "Ada" {
		t.Fatalf("user.name = %q, want Ada", got)
	}
	raw, marshalErr := json.Marshal(plainRecord(rec))
	if marshalErr != nil {
		t.Fatalf("the record does not marshal: %v", marshalErr)
	}
	if strings.Contains(string(raw), planted) {
		t.Fatalf("the planted e-mail reaches the output: %s", raw)
	}
	// The read above returns a nil error, so no error carries the plant either.
	for _, req := range stub.Requests() {
		if strings.Contains(string(req.Body), planted) {
			t.Fatalf("the planted e-mail reaches a request: %q", req.Body)
		}
	}
}

// TestObjectPointerToWholeObjectIsRefused checks that a sub-field pointer that reaches an object or a list is refused
// with unexpected_response, so a whole object is never copied. The refusal names the dotted path of the sub-field and
// never repeats a value of the reply.
func TestObjectPointerToWholeObjectIsRefused(t *testing.T) {
	cases := []struct {
		name    string
		pointer string
		body    string
		path    string
		secret  string
	}{
		{"a pointer to an object", "/data/user",
			`{"data":{"user":{"id":"u-1","name":"Ada"}}}`,
			"user.id", "Ada"},
		{"a pointer to a list", "/data/tags",
			`{"data":{"user":{"id":"u-1","name":"Ada"},"tags":["one","two"]}}`,
			"user.id", "one"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			doc := replaceIn(t, nestedRecordDoc, "pointer: /data/user/id", "pointer: "+c.pointer)
			eng, stub, _ := newStubEngine(t, loadSpecFrom(t, doc))
			stub.Script("GET", "/v1/whoami", page(c.body))
			_, err := eng.Read(context.Background(), "example-wiki.whoami.get", domain.NewArgs(nil))
			if !errors.Is(err, domain.ErrUnexpectedResponse) {
				t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
			}
			if !strings.Contains(err.Error(), c.path) {
				t.Fatalf("the refusal does not name the path %q: %q", c.path, err.Error())
			}
			if strings.Contains(err.Error(), c.secret) {
				t.Fatalf("the refusal echoes a value of the reply: %q", err.Error())
			}
		})
	}
}

// TestNestedSubFieldRulesFollowTheScalarRules checks required and format on a sub-field: a missing required sub-field
// and a sub-field of the wrong shape fail with unexpected_response, and each failure names the dotted path, never the
// value. A sub-field of the right shape maps.
func TestNestedSubFieldRulesFollowTheScalarRules(t *testing.T) {
	doc := replaceIn(t, nestedRecordDoc,
		"            id:\n              pointer: /data/user/id",
		"            id:\n              pointer: /data/user/id\n              required: true\n              format: id")
	cases := []struct {
		name  string
		body  string
		path  string
		value string
	}{
		{"a missing required sub-field", `{"data":{"user":{"name":"Ada"}}}`, "user.id", ""},
		{"a sub-field of the wrong shape", `{"data":{"user":{"id":"u.1","name":"Ada"}}}`, "user.id", "u.1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			eng, stub, _ := newStubEngine(t, loadSpecFrom(t, doc))
			stub.Script("GET", "/v1/whoami", page(c.body))
			rec, err := eng.Read(context.Background(), "example-wiki.whoami.get", domain.NewArgs(nil))
			if !errors.Is(err, domain.ErrUnexpectedResponse) {
				t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
			}
			if len(rec.Fields()) != 0 {
				t.Fatalf("a refused read returned a partial record: %+v", rec)
			}
			if !strings.Contains(err.Error(), c.path) {
				t.Fatalf("the refusal does not name the path %q: %q", c.path, err.Error())
			}
			if c.value != "" && strings.Contains(err.Error(), c.value) {
				t.Fatalf("the refusal echoes a value of the reply: %q", err.Error())
			}
		})
	}
	t.Run("a sub-field of the right shape", func(t *testing.T) {
		eng, stub, _ := newStubEngine(t, loadSpecFrom(t, doc))
		stub.Script("GET", "/v1/whoami", page(`{"data":{"user":{"id":"u-1","name":"Ada"}}}`))
		rec, err := eng.Read(context.Background(), "example-wiki.whoami.get", domain.NewArgs(nil))
		if err != nil {
			t.Fatalf("a sound nested reply was refused: %v", err)
		}
		if got := rec.Rec("user").Str("id"); got != "u-1" {
			t.Fatalf("user.id = %q, want u-1", got)
		}
	})
}

// TestNestedObjectDepthLimit checks that a record nests one level only: a spec whose object holds another object is
// refused at load, and an engine built from a hand-made model with a nested object is refused too. Each refusal names
// the nested object field and never a value.
func TestNestedObjectDepthLimit(t *testing.T) {
	deep := replaceIn(t, nestedRecordDoc,
		"            id:\n              pointer: /data/user/id",
		"            id:\n              object:\n                leaf:\n                  pointer: /data/user/id")
	if res := spec.Validate([]byte(deep)); res.Valid() {
		t.Fatal("a spec with an object inside an object is valid")
	} else {
		found := false
		for _, f := range res.Findings {
			if f.Path == "/operations/0/response/fields/user/object/id/object" {
				found = true
			}
		}
		if !found {
			t.Fatalf("the refusal names no nested object field: %v", res.Findings)
		}
	}

	// A hand-built model bypasses the schema, so the engine constructor itself refuses the nested object. The model
	// is decoded from JSON, so this test compiles before the model grows the object keyword and fails on the verdict.
	var s spec.Spec
	if err := json.Unmarshal([]byte(nestedObjectModel), &s); err != nil {
		t.Fatalf("the hand-built model does not decode: %v", err)
	}
	stub, err := replay.New(loadSpec(t))
	if err != nil {
		t.Fatalf("build the replay stub: %v", err)
	}
	if _, err := engine.New(&s, testSource(t), stubPorts(stub, &fakeClock{})); err == nil {
		t.Fatal("an engine with an object inside an object was built")
	}
}
