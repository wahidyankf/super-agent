package unit_test

import (
	"fmt"
	"strings"
	"testing"
)

// userRecordOp is a single-object read at index four whose user field is a nested record of two scalar sub-fields.
const userRecordOp = `  - id: example-wiki.docs.user
    summary: Read one user record.
    access: read
    method: GET
    path: /user
    response:
      fields:
        title:
          pointer: /title
        user:
          object:
            id:
              pointer: /user/id
            name:
              pointer: /user/name
    examples:
      - request: {}
        response:
          status: 200
          body:
            title: Profile
            user:
              id: u-1
              name: Ada
        expect: {}
`

// settingsOp is a single-object read at index four whose settings field is an open-key map of flags.
const settingsOp = `  - id: example-wiki.docs.settings
    summary: Read the notification settings.
    access: read
    method: GET
    path: /settings
    response:
      fields:
        settings:
          pointer: /settings
          bool_map: true
    examples:
      - request: {}
        response:
          status: 200
          body:
            settings:
              doc.create: true
        expect: {}
`

// indexOp is a list read at index four whose id and slug fields may each name a format.
const indexOp = `  - id: example-wiki.docs.index
    summary: List document slugs.
    access: read
    method: GET
    path: /index
    response:
      items: /items
      fields:
        id:
          pointer: /id
        slug:
          pointer: /slug
    examples:
      - request: {}
        response:
          status: 200
          body:
            items:
              - id: doc-1
                slug: first
        expect: {}
`

// @scenario specs/libs/retriever-spec/behaviours/record-fields.feature: A nested record of scalar sub-fields is accepted
func TestNestedRecordOfScalarSubFieldsIsAccepted(t *testing.T) {
	requireAccepted(t, withOps(userRecordOp))
}

// @scenario specs/libs/retriever-spec/behaviours/record-fields.feature: A nested record that holds another record is refused at the inner record
func TestNestedRecordThatHoldsAnotherRecordIsRefusedAtTheInnerRecord(t *testing.T) {
	doc := mutate(t, withOps(userRecordOp), "            id:\n              pointer: /user/id\n",
		"            id:\n              object:\n                key:\n                  pointer: /user/id/key\n")
	requireRefusedAt(t, doc, "/operations/4/response/fields/user/object/id/object")
}

// @scenario specs/libs/retriever-spec/behaviours/record-fields.feature: A nested record with 33 sub-fields is refused at the record
func TestNestedRecordWith33SubFieldsIsRefusedAtTheRecord(t *testing.T) {
	var subs strings.Builder
	for i := 0; i < 33; i++ {
		fmt.Fprintf(&subs, "            f%02d:\n              pointer: /user/f%02d\n", i, i)
	}
	doc := mutate(t, withOps(userRecordOp), "          object:\n            id:\n              pointer: /user/id\n            name:\n              pointer: /user/name\n",
		"          object:\n"+subs.String())
	requireRefusedAt(t, doc, "/operations/4/response/fields/user/object")
}

// @scenario specs/libs/retriever-spec/behaviours/record-fields.feature: A field that is both a record and a pointer is refused at the field
func TestFieldThatIsBothRecordAndPointerIsRefusedAtTheField(t *testing.T) {
	doc := mutate(t, withOps(userRecordOp), "        user:\n          object:",
		"        user:\n          pointer: /user\n          object:")
	requireRefusedAt(t, doc, "/operations/4/response/fields/user/pointer")
}

// @scenario specs/libs/retriever-spec/behaviours/record-fields.feature: An open-key map with flag values is accepted
func TestOpenKeyMapWithFlagValuesIsAccepted(t *testing.T) {
	requireAccepted(t, withOps(settingsOp))
}

// @scenario specs/libs/retriever-spec/behaviours/record-fields.feature: An open-key map whose flag keyword is false is refused
func TestOpenKeyMapWhoseFlagKeywordIsFalseIsRefused(t *testing.T) {
	requireRefused(t, mutate(t, withOps(settingsOp), "bool_map: true", "bool_map: false"))
}

// @scenario specs/libs/retriever-spec/behaviours/record-fields.feature: An open key set spelled any other way is refused
func TestOpenKeySetSpelledAnyOtherWayIsRefused(t *testing.T) {
	cases := []struct {
		name string
		repl string
	}{
		{"a star sub-field that matches any key", "          object:\n            \"*\":\n              pointer: /settings/x\n"},
		{"a schema spelling of additional properties", "          additionalProperties: true\n"},
		{"an undeclared value rule for the open keys", "          values: flag\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			doc := mutate(t, withOps(settingsOp), "          pointer: /settings\n          bool_map: true\n", c.repl)
			requireRefused(t, doc)
		})
	}
}

// @scenario specs/libs/retriever-spec/behaviours/record-fields.feature: An open-key map that also sets a scalar keyword is refused
func TestOpenKeyMapThatAlsoSetsAScalarKeywordIsRefused(t *testing.T) {
	doc := mutate(t, withOps(settingsOp), "          bool_map: true\n", "          bool_map: true\n          format: id\n")
	requireRefused(t, doc)
}

// @scenario specs/libs/retriever-spec/behaviours/record-fields.feature: A format name from the closed list is accepted on a reply field
func TestFormatNameFromClosedListIsAcceptedOnReplyField(t *testing.T) {
	for _, format := range []string{"id", "uuid", "share-id", "strings"} {
		t.Run(format, func(t *testing.T) {
			doc := mutate(t, withOps(indexOp), "        id:\n          pointer: /id\n        slug:",
				"        id:\n          pointer: /id\n          format: "+format+"\n        slug:")
			requireAccepted(t, doc)
		})
	}
}

// @scenario specs/libs/retriever-spec/behaviours/record-fields.feature: A format name outside the closed list is refused at the format of the field
func TestFormatNameOutsideClosedListIsRefusedAtTheFormatOfTheField(t *testing.T) {
	// The id field of the first operation of the bundled example has the same text, so the id row anchors on the slug
	// field that follows it, which only the appended index operation holds.
	cases := []struct {
		name, old, repl, path string
	}{
		{
			name: "id with pattern",
			old:  "        id:\n          pointer: /id\n        slug:",
			repl: "        id:\n          pointer: /id\n          format: pattern\n        slug:",
			path: "/operations/4/response/fields/id/format",
		},
		{
			name: "id with ID",
			old:  "        id:\n          pointer: /id\n        slug:",
			repl: "        id:\n          pointer: /id\n          format: ID\n        slug:",
			path: "/operations/4/response/fields/id/format",
		},
		{
			name: "slug with share_id",
			old:  "        slug:\n          pointer: /slug\n",
			repl: "        slug:\n          pointer: /slug\n          format: share_id\n",
			path: "/operations/4/response/fields/slug/format",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			requireRefusedAt(t, mutate(t, withOps(indexOp), c.old, c.repl), c.path)
		})
	}
}
