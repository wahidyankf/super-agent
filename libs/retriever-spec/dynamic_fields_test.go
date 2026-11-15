package spec

import (
	"strings"
	"testing"
)

// dynamicFieldDoc returns a read document whose nested record tries the given open key set for the notification
// settings: the reply object whose keys the wiki chooses. The language admits one open form only, the open-key map
// with closed bool values, so any other open key set is refused.
func dynamicFieldDoc(object string) string {
	return `version: 1
kind: example-wiki
summary: Reads a user with notification settings.
endpoint:
  base_path: /v1
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
operations:
  - id: example-wiki.users.notifications-subscribe
    summary: Turn on notification settings.
    access: write
    method: POST
    path: /notifications
    response:
      fields:
        id:
          pointer: /data/id
        notificationSettings:
          object:
` + object + `    examples:
      - request: {}
        response:
          status: 200
          body:
            data:
              id: u-1
        expect:
          id: u-1
`
}

// TestDynamicRecordFieldsStayRefused checks the fail-closed rule for the open key sets the spec does not declare:
// a star sub-field, the JSON Schema spellings, and an undeclared value rule are all refused. The only open record
// mechanism is the bool-map field, so a document that spells openness any other way stays refused.
func TestDynamicRecordFieldsStayRefused(t *testing.T) {
	for _, object := range []string{
		"            \"*\": { pointer: /data/notificationSettings/star }\n",
		"            patternProperties: {}\n",
		"            additionalProperties: true\n",
		"            additionalProperties: { type: boolean }\n",
		"            type: boolean\n",
	} {
		res := Validate([]byte(dynamicFieldDoc(object)))
		if res.Valid() {
			t.Fatalf("an open key set validates: %q", strings.TrimSpace(object))
		}
	}
}

// boolMapDoc returns a write document whose notificationSettings is the declared open-key map with closed bool
// values: the pointer of the reply object, with the bool-map keyword and nothing else.
func boolMapDoc() string {
	return `version: 1
kind: example-wiki
summary: Reads a user with notification settings.
endpoint:
  base_path: /v1
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
operations:
  - id: example-wiki.users.notifications-subscribe
    summary: Turn on notification settings.
    access: write
    method: POST
    path: /notifications
    response:
      fields:
        id:
          pointer: /data/id
        notificationSettings:
          pointer: /data/notificationSettings
          bool_map: true
    examples:
      - request: {}
        response:
          status: 200
          body:
            data:
              id: u-1
        expect:
          id: u-1
`
}

// TestBoolMapFormIsAccepted checks that the declared map form validates: an open-key map with closed bool values is
// the one open record form the language admits, so a document that names it loads.
func TestBoolMapFormIsAccepted(t *testing.T) {
	res := Validate([]byte(boolMapDoc()))
	if !res.Valid() {
		t.Fatalf("the bool-map form is refused: %v", res.Err())
	}
}

// TestBoolMapFalseIsRefused checks that the keyword is exactly true: a bool-map field that names false is refused,
// so a document cannot spell the open form without taking it.
func TestBoolMapFalseIsRefused(t *testing.T) {
	doc := boolMapDoc()
	res := Validate([]byte(mutateBoolMap(t, doc, "          bool_map: true", "          bool_map: false")))
	if res.Valid() {
		t.Fatal("a bool-map field with bool_map: false validates")
	}
}

// mutateBoolMap returns doc with the first occurrence of old changed to new, failing the test when old is absent, so
// a fixture edit can never silently do nothing.
func mutateBoolMap(t *testing.T, doc, old, new string) string {
	t.Helper()
	if !strings.Contains(doc, old) {
		t.Fatalf("the fixture has no %q to change", old)
	}
	return strings.Replace(doc, old, new, 1)
}

// TestReviewListsBoolMaps checks the review of an operation with an open-key map with closed bool values: its own
// block names the operation, the map field, and its pointer. A spec with no bool-map field has no such block, so
// its review keeps its text.
func TestReviewListsBoolMaps(t *testing.T) {
	review := Review(requireValid(t, boolMapDoc()).Spec)
	block := "Open bool maps: 1\n1. example-wiki.users.notifications-subscribe: notificationSettings keeps every bool at /data/notificationSettings, sorted\n"
	if !strings.Contains(review, block) {
		t.Fatalf("the review has no bool-maps block:\n%s", review)
	}
	if strings.Contains(Review(requireValid(t, exampleWiki).Spec), "Open bool maps") {
		t.Fatal("a spec with no bool-map field has a bool-maps block")
	}
}

// TestBoolMapRefusesScalarKeywords checks the Go rule on models that bypass the schema: a bool-map field with a
// format, a converter, or an object of its own is refused, and so is a bool map with no pointer, and a bool map
// nested inside an object, each at its field path with a fixed rule. A sound bool map passes with no finding.
func TestBoolMapRefusesScalarKeywords(t *testing.T) {
	sound := map[string]Field{
		"id":                   {Pointer: "/data/id"},
		"notificationSettings": {Pointer: "/data/notificationSettings", BoolMap: true},
	}
	cases := []struct {
		name   string
		fields map[string]Field
		path   string
		rule   string
	}{
		{"a sound bool map", sound, "", ""},
		{"a bool map with a format", map[string]Field{
			"id":                   sound["id"],
			"notificationSettings": {Pointer: "/data/notificationSettings", BoolMap: true, Format: "uuid"},
		}, "/operations/0/response/fields/notificationSettings", ruleBoolMapScalar},
		{"a bool map with a converter", map[string]Field{
			"id":                   sound["id"],
			"notificationSettings": {Pointer: "/data/notificationSettings", BoolMap: true, Convert: "hostname"},
		}, "/operations/0/response/fields/notificationSettings", ruleBoolMapScalar},
		{"a bool map with an object", map[string]Field{
			"id": sound["id"],
			"notificationSettings": {Pointer: "/data/notificationSettings", BoolMap: true,
				Object: map[string]Field{"a": {Pointer: "/data/notificationSettings/a"}}},
		}, "/operations/0/response/fields/notificationSettings", ruleBoolMapScalar},
		{"a bool map with no pointer", map[string]Field{
			"id":                   sound["id"],
			"notificationSettings": {BoolMap: true},
		}, "/operations/0/response/fields/notificationSettings", ruleBoolMapPointer},
		{"a bool map inside an object", map[string]Field{
			"user": {Object: map[string]Field{
				"id":                   {Pointer: "/data/user/id"},
				"notificationSettings": {Pointer: "/data/user/notificationSettings", BoolMap: true},
			}},
		}, "/operations/0/response/fields/user/object/notificationSettings/bool_map", ruleBoolMapDepth},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := &Spec{Operations: []Operation{{ID: "example-wiki.users.notifications-subscribe",
				Response: Response{Fields: c.fields}}}}
			fs := CheckObjectFields(s)
			if c.path == "" {
				if len(fs) != 0 {
					t.Fatalf("a sound bool map has findings: %v", fs)
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
