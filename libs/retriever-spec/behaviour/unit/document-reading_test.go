package unit_test

import (
	"strings"
	"testing"

	spec "github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// @scenario specs/libs/retriever-spec/behaviours/document-reading.feature: A valid spec document returns a model and no error
func TestValidSpecDocumentReturnsAModelAndNoError(t *testing.T) {
	r := requireAccepted(t, exampleWiki)
	if r.Spec.Kind != "example-wiki" {
		t.Fatalf("model kind = %q; want example-wiki", r.Spec.Kind)
	}
	if err := r.Err(); err != nil {
		t.Fatalf("Err() = %v; want nil", err)
	}
}

// @scenario specs/libs/retriever-spec/behaviours/document-reading.feature: A valid connector spec with plain text, whole numbers, and true or false flags is accepted
func TestBundledExampleWithPlainTextNumbersAndFlagsIsAccepted(t *testing.T) {
	requireAccepted(t, exampleWiki)
}

// @scenario specs/libs/retriever-spec/behaviours/document-reading.feature: A spec document of exactly 256 KiB is read
func TestSpecDocumentOfExactly256KiBIsRead(t *testing.T) {
	doc := exampleWiki + "#" + strings.Repeat("x", spec.MaxBytes-len(exampleWiki)-1)
	if len(doc) != spec.MaxBytes {
		t.Fatalf("fixture is %d bytes; want %d", len(doc), spec.MaxBytes)
	}
	r := requireAccepted(t, doc)
	if r.Spec.Kind != "example-wiki" {
		t.Fatalf("model kind = %q; want example-wiki", r.Spec.Kind)
	}
}

// @scenario specs/libs/retriever-spec/behaviours/document-reading.feature: A spec document over 256 KiB is refused for the whole document
func TestSpecDocumentOver256KiBIsRefusedForTheWholeDocument(t *testing.T) {
	doc := exampleWiki + "#" + strings.Repeat("x", spec.MaxBytes)
	r := requireRefusedAt(t, doc, "")
	if len(r.Findings) != 1 {
		t.Fatalf("findings = %d; want 1", len(r.Findings))
	}
}

// @scenario specs/libs/retriever-spec/behaviours/document-reading.feature: A YAML alias, anchor, or repeated key is refused at the field that holds it
func TestYAMLAliasAnchorOrRepeatedKeyIsRefusedAtItsField(t *testing.T) {
	cases := []struct {
		name string
		doc  string
		path string
	}{
		{
			name: "an alias where an operation summary belongs",
			doc: mutate(t, exampleWiki,
				"    summary: List documents.\n", "    summary: &first List documents.\n",
				"    summary: Read one document.\n", "    summary: *first\n"),
			path: "/operations/1/summary",
		},
		{
			name: "an anchor on the top-level summary",
			doc: mutate(t, exampleWiki,
				"summary: Reads and writes documents of an example wiki.",
				"summary: &top Reads and writes documents of an example wiki."),
			path: "/summary",
		},
		{
			name: "a second value for the top-level kind key",
			doc:  mutate(t, exampleWiki, "kind: example-wiki\n", "kind: example-wiki\nkind: example-wiki\n"),
			path: "/kind",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			requireRefusedAt(t, c.doc, c.path)
		})
	}
}

// @scenario specs/libs/retriever-spec/behaviours/document-reading.feature: A YAML tag or timestamp that the spec does not allow is refused at the field that holds it
func TestYAMLTagOrTimestampIsRefusedAtItsField(t *testing.T) {
	const summary = "summary: Reads and writes documents of an example wiki."
	cases := []struct {
		name string
		doc  string
		path string
	}{
		{"summary with a custom YAML tag", mutate(t, exampleWiki, summary, "summary: !custom Reads and writes documents of an example wiki."), "/summary"},
		{"summary with a bare date", mutate(t, exampleWiki, summary, "summary: 2026-11-22"), "/summary"},
		{"endpoint with a custom YAML tag", mutate(t, exampleWiki, "endpoint:\n  base_path: /v1", "endpoint: !custom\n  base_path: /v1"), "/endpoint"},
		{"endpoint host suffix with a custom YAML tag", mutate(t, exampleWiki, "  host_suffix:\n    - .example.test", "  host_suffix: !custom\n    - .example.test"), "/endpoint/host_suffix"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			requireRefusedAt(t, c.doc, c.path)
		})
	}
}

// @scenario specs/libs/retriever-spec/behaviours/document-reading.feature: A number or flag that is not a plain finite value is refused at its field
func TestNumberOrFlagThatIsNotPlainFiniteIsRefusedAtItsField(t *testing.T) {
	cases := []struct {
		name string
		doc  string
		path string
	}{
		{"a top-level field that is not a number", mutate(t, exampleWiki, "kind: example-wiki\n", "kind: example-wiki\nnan: .nan\n"), "/nan"},
		{"a top-level field that is an infinite number", mutate(t, exampleWiki, "kind: example-wiki\n", "kind: example-wiki\ninf: .inf\n"), "/inf"},
		{"a parameter required flag that is not true or false", mutate(t, exampleWiki,
			"        required: false\n        help: The cursor of the next page.", "        required: maybe\n        help: The cursor of the next page."),
			"/operations/0/params/0/required"},
		{"a parameter default that is not finite", mutate(t, exampleWiki,
			"        default: 20\n    pagination:", "        default: .nan\n    pagination:"),
			"/operations/0/params/1/default"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			requireRefusedAt(t, c.doc, c.path)
		})
	}
}

// @scenario specs/libs/retriever-spec/behaviours/document-reading.feature: A document whose root is a list is refused for the whole document
func TestDocumentWhoseRootIsAListIsRefusedForTheWholeDocument(t *testing.T) {
	requireRefusedAt(t, "- version: 1\n- kind: example-wiki\n", "")
}

// @scenario specs/libs/retriever-spec/behaviours/document-reading.feature: A document with a YAML syntax error is refused without repeating the broken text
func TestYAMLSyntaxErrorIsRefusedWithoutRepeatingTheBrokenText(t *testing.T) {
	doc := "version: [broken-token\n" + exampleWiki[strings.Index(exampleWiki, "kind:"):]
	r := requireRefusedAt(t, doc, "")
	if strings.Contains(text(r), "broken-token") {
		t.Fatalf("a finding repeats the broken text:\n%s", text(r))
	}
}

// @scenario specs/libs/retriever-spec/behaviours/document-reading.feature: A document with a key that is not text is refused for the whole document
func TestDocumentWithKeyThatIsNotTextIsRefusedForTheWholeDocument(t *testing.T) {
	doc := mutate(t, exampleWiki, "version: 1\n", "version: 1\n1: one\n")
	requireRefusedAt(t, doc, "")
}

// @scenario specs/libs/retriever-spec/behaviours/document-reading.feature: A document with a second YAML document after the first is refused for the whole document
func TestDocumentWithSecondYAMLDocumentIsRefusedForTheWholeDocument(t *testing.T) {
	requireRefusedAt(t, exampleWiki+"---\nversion: 1\n", "")
}

// @scenario specs/libs/retriever-spec/behaviours/document-reading.feature: An empty document is refused
func TestEmptyDocumentIsRefused(t *testing.T) {
	requireRefusedAt(t, "", "")
}
