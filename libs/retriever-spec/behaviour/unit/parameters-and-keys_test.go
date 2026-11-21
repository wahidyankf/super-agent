package unit_test

import (
	"fmt"
	"testing"
)

// @scenario specs/libs/retriever-spec/behaviours/parameters-and-keys.feature: A query parameter with a wire key of its own is accepted
func TestQueryParameterWithWireKeyOfItsOwnIsAccepted(t *testing.T) {
	requireAccepted(t, mutate(t, exampleWiki, "        default: 20\n    pagination:",
		"        default: 20\n        query_key: maxResults\n    pagination:"))
}

// @scenario specs/libs/retriever-spec/behaviours/parameters-and-keys.feature: A wire key that a path or body parameter cannot take is refused
func TestWireKeyThatPathOrBodyParameterCannotTakeIsRefused(t *testing.T) {
	cases := []struct {
		name, old, repl string
	}{
		{"path parameter", "        in: path\n", "        in: path\n        query_key: docId\n"},
		{"body parameter", "        in: body\n", "        in: body\n        query_key: title\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			requireRefused(t, mutate(t, exampleWiki, c.old, c.repl))
		})
	}
}

// @scenario specs/libs/retriever-spec/behaviours/parameters-and-keys.feature: A wire key that repeats a name or another query key of the same operation is refused
func TestWireKeyThatRepeatsANameOrAnotherQueryKeyIsRefused(t *testing.T) {
	cases := []struct {
		name  string
		pairs []string
	}{
		{"repeats the parameter name", []string{
			"        default: 20\n    pagination:", "        default: 20\n        query_key: limit\n    pagination:",
		}},
		{"is the same as another query key", []string{
			"        help: The cursor of the next page.\n", "        help: The cursor of the next page.\n        query_key: pageSize\n",
			"        default: 20\n    pagination:", "        default: 20\n        query_key: pageSize\n    pagination:",
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			requireRefused(t, mutate(t, exampleWiki, c.pairs...))
		})
	}
}

// @scenario specs/libs/retriever-spec/behaviours/parameters-and-keys.feature: A built key field that names one source pointer is accepted
func TestBuiltKeyFieldThatNamesOneSourcePointerIsAccepted(t *testing.T) {
	requireAccepted(t, mutate(t, exampleWiki, "        title:\n          pointer: /title\n    examples:",
		"        title:\n          pointer: /title\n        projectKey:\n          project_key: /key\n    examples:"))
}

// @scenario specs/libs/retriever-spec/behaviours/parameters-and-keys.feature: A field that holds a pointer and also builds a key is refused at the field
func TestFieldThatHoldsPointerAndAlsoBuildsAKeyIsRefusedAtTheField(t *testing.T) {
	doc := mutate(t, exampleWiki, "        title:\n          pointer: /title\n    examples:",
		"        title:\n          pointer: /title\n        projectKey:\n          project_key: /key\n          pointer: /key\n    examples:")
	requireRefusedAt(t, doc, "/operations/0/response/fields/projectKey")
}

// @scenario specs/libs/retriever-spec/behaviours/parameters-and-keys.feature: A built key field that also sets a format is refused at the field
func TestBuiltKeyFieldThatAlsoSetsAFormatIsRefusedAtTheField(t *testing.T) {
	doc := mutate(t, exampleWiki, "        title:\n          pointer: /title\n    examples:",
		"        title:\n          pointer: /title\n        projectKey:\n          project_key: /key\n          format: id\n    examples:")
	requireRefusedAt(t, doc, "/operations/0/response/fields/projectKey")
}

// @scenario specs/libs/retriever-spec/behaviours/parameters-and-keys.feature: A reply field may name each closed converter
func TestReplyFieldMayNameEachClosedConverter(t *testing.T) {
	converters := []string{
		"plain", "markdown-from-adf", "hostname", "text-from-editor-data", "jira-time", "jira-date", "jira-title",
	}
	for _, converter := range converters {
		t.Run(converter, func(t *testing.T) {
			requireAccepted(t, mutate(t, exampleWiki, "        title:\n          pointer: /title\n    examples:",
				fmt.Sprintf("        title:\n          pointer: /title\n          convert: %s\n    examples:", converter)))
		})
	}
}

// @scenario specs/libs/retriever-spec/behaviours/parameters-and-keys.feature: A reply field that names a converter outside the closed list is refused at the converter
func TestReplyFieldNamingConverterOutsideClosedListIsRefusedAtTheConverter(t *testing.T) {
	doc := mutate(t, exampleWiki, "convert: markdown-from-html", "convert: unknown-converter")
	requireRefusedAt(t, doc, "/operations/1/response/fields/body/convert")
}

// @scenario specs/libs/retriever-spec/behaviours/parameters-and-keys.feature: An issue-key parameter of the issue key type is accepted
func TestIssueKeyParameterOfTheIssueKeyTypeIsAccepted(t *testing.T) {
	requireAccepted(t, mutate(t, exampleWiki, "      - name: cursor\n        type: ID", "      - name: cursor\n        type: IssueKey"))
}

// @scenario specs/libs/retriever-spec/behaviours/parameters-and-keys.feature: A parameter of a catalog kind is accepted, including icon, URL list, and event list kinds
func TestParameterOfCatalogKindIsAccepted(t *testing.T) {
	for _, kind := range []string{"ID", "Limit", "Enum", "Icon", "URLList", "EventList"} {
		t.Run(kind, func(t *testing.T) {
			repl := "      - name: cursor\n        type: " + kind
			if kind == "Enum" {
				repl += "\n        values: [open, closed]"
			}
			requireAccepted(t, mutate(t, exampleWiki, "      - name: cursor\n        type: ID", repl))
		})
	}
}

// @scenario specs/libs/retriever-spec/behaviours/parameters-and-keys.feature: An enum parameter with no closed values is refused at its values
func TestEnumParameterWithNoClosedValuesIsRefusedAtItsValues(t *testing.T) {
	doc := mutate(t, exampleWiki, "        default: 20\n    pagination:",
		"        default: 20\n      - name: state\n        type: Enum\n        in: query\n        required: false\n    pagination:")
	requireRefusedAt(t, doc, "/operations/0/params/2/values")
}
