package unit_test

import (
	"errors"
	"strings"
	"testing"

	spec "github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// tokenShaped is a credential-shaped run that a finding must never show.
const tokenShaped = "ghp_" + "aB3dE5fG7hJ9kL1mN3pQ5rS7tU9vW1xY3zA5"

// docsDeleteExample is the example of the fourth operation. Removing it leaves that operation with no example.
const docsDeleteExample = "    examples:\n      - request:\n          id: doc-1\n        response:\n          status: 204\n        expect: {}\n"

// @scenario specs/libs/retriever-spec/behaviours/refusals-and-findings.feature: A spec that breaks one rule is refused with no model and a finding at the broken field
func TestSpecThatBreaksOneRuleIsRefusedAtTheBrokenField(t *testing.T) {
	cases := []struct {
		name string
		doc  string
		path string
	}{
		{"a summary with braces", mutate(t, exampleWiki,
			"summary: Reads and writes documents of an example wiki.", "summary: Reads {documents} of an example wiki."), "/summary"},
		{"an operation with no example", mutate(t, exampleWiki, docsDeleteExample, ""), "/operations/3/examples"},
		{"an unknown parameter kind", mutate(t, exampleWiki, "      - name: limit\n        type: Limit", "      - name: limit\n        type: Bogus"),
			"/operations/0/params/1/type"},
		{"an operation id outside its kind", mutate(t, exampleWiki, "  - id: example-wiki.docs.list", "  - id: other.docs.list"), "/operations/0/id"},
		{"a path slot with no declared parameter", mutate(t, exampleWiki, "    path: /docs/{id}\n    params:", "    path: /docs/{doc}\n    params:"),
			"/operations/1/path"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			requireRefusedAt(t, c.doc, c.path)
		})
	}
}

// @scenario specs/libs/retriever-spec/behaviours/refusals-and-findings.feature: A spec that breaks two rules lists both findings in path order
func TestSpecThatBreaksTwoRulesListsBothFindingsInPathOrder(t *testing.T) {
	doc := mutate(t, exampleWiki,
		"summary: Reads and writes documents of an example wiki.", "summary: Reads {documents} of an example wiki.",
		docsDeleteExample, "")
	r := requireRefused(t, doc)
	examples, summary := -1, -1
	for i, f := range r.Findings {
		switch f.Path {
		case "/operations/3/examples":
			examples = i
		case "/summary":
			summary = i
		}
	}
	if examples < 0 || summary < 0 {
		t.Fatalf("want findings at /operations/3/examples and /summary; findings:\n%s", text(r))
	}
	if examples > summary {
		t.Fatalf("/operations/3/examples comes after /summary; findings:\n%s", text(r))
	}
}

// @scenario specs/libs/retriever-spec/behaviours/refusals-and-findings.feature: The error of a refused spec matches the spec error sentinel
func TestErrorOfRefusedSpecMatchesTheSpecErrorSentinel(t *testing.T) {
	doc := mutate(t, exampleWiki, "    path: /docs/{id}\n    params:", "    path: /docs/{doc}\n    params:")
	r := requireRefused(t, doc)
	if !errors.Is(r.Err(), spec.ErrInvalid) {
		t.Fatalf("Err() = %v; want a match with ErrInvalid", r.Err())
	}
}

// @scenario specs/libs/retriever-spec/behaviours/refusals-and-findings.feature: A document over the size cap is refused with a finding that names the whole document
func TestDocumentOverTheSizeCapNamesTheWholeDocument(t *testing.T) {
	doc := exampleWiki + "#" + strings.Repeat("x", spec.MaxBytes)
	r := requireRefusedAt(t, doc, "")
	if got := r.Findings[0].String(); !strings.Contains(got, "(document)") {
		t.Fatalf("finding = %q; want it to show (document)", got)
	}
}

// @scenario specs/libs/retriever-spec/behaviours/refusals-and-findings.feature: A field name that is not plain text is withheld from the finding
func TestFieldNameThatIsNotPlainTextIsWithheldFromTheFinding(t *testing.T) {
	const token = "Q9vR7sLmT2pW4kZ8.hN3aB6cD1eF5gJ0"
	doc := mutate(t, exampleWiki, "kind: example-wiki\n", "kind: example-wiki\n\""+token+"\": x\n")
	r := requireRefused(t, doc)
	if !strings.Contains(text(r), "(name withheld)") {
		t.Fatalf("no finding shows the placeholder (name withheld):\n%s", text(r))
	}
	if strings.Contains(text(r), token) {
		t.Fatalf("a finding shows the field name token:\n%s", text(r))
	}
}

// @scenario specs/libs/retriever-spec/behaviours/refusals-and-findings.feature: A plain unknown field name is shown in its finding path
func TestPlainUnknownFieldNameIsShownInItsFindingPath(t *testing.T) {
	requireRefusedAt(t, mutate(t, exampleWiki, "kind: example-wiki\n", "kind: example-wiki\nextra: x\n"), "/extra")
}

// @scenario specs/libs/retriever-spec/behaviours/refusals-and-findings.feature: A value shaped like a credential is refused and is never shown in a finding
func TestValueShapedLikeACredentialIsRefusedAndNeverShown(t *testing.T) {
	cases := []struct {
		name string
		doc  string
		path string
	}{
		{"a token-shaped string in the summary", mutate(t, exampleWiki,
			"summary: Reads and writes documents of an example wiki.", "summary: Reads with "+tokenShaped+" tokens."), "/summary"},
		{"a bearer-shaped string in the title of an example reply body", mutate(t, exampleWiki,
			"            title: First document\n            body: <p>Hello</p>",
			"            title: Bearer "+tokenShaped+"\n            body: <p>Hello</p>"),
			"/operations/1/examples/0/response/body/title"},
		{"an Authorization header in the headers of an example reply", mutate(t, exampleWiki,
			"          status: 200\n          body:\n            title: First document",
			"          status: 200\n          headers:\n            Authorization: Bearer "+tokenShaped+"\n          body:\n            title: First document"),
			"/operations/1/examples/0/response/headers/Authorization"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := requireRefusedAt(t, c.doc, c.path)
			if strings.Contains(text(r), tokenShaped) {
				t.Fatalf("a finding shows the credential-shaped value:\n%s", text(r))
			}
		})
	}
}

// @scenario specs/libs/retriever-spec/behaviours/refusals-and-findings.feature: A finding prints as one line with its code, its path, and its rule
func TestFindingPrintsAsOneLineWithItsCodePathAndRule(t *testing.T) {
	doc := mutate(t, exampleWiki, "summary: Reads and writes documents of an example wiki.", "summary: Reads {documents} of an example wiki.")
	r := requireRefusedAt(t, doc, "/summary")
	for _, f := range r.Findings {
		if f.Path != "/summary" {
			continue
		}
		line := f.String()
		if strings.Contains(line, "\n") || !strings.HasPrefix(line, spec.CodeSpecInvalid+": /summary: ") {
			t.Fatalf("finding line = %q; want the code, then /summary, then the rule", line)
		}
		return
	}
	t.Fatal("no finding at /summary")
}

// @scenario specs/libs/retriever-spec/behaviours/refusals-and-findings.feature: A finding for the whole document prints the document in place of a path
func TestFindingForWholeDocumentPrintsTheDocumentInPlaceOfAPath(t *testing.T) {
	r := requireRefusedAt(t, exampleWiki+"#"+strings.Repeat("x", spec.MaxBytes), "")
	line := r.Findings[0].String()
	if !strings.HasPrefix(line, spec.CodeSpecInvalid+": (document): ") {
		t.Fatalf("finding line = %q; want the code, then (document), then the rule", line)
	}
}

// @scenario specs/libs/retriever-spec/behaviours/refusals-and-findings.feature: The error of a refused spec lists each finding on its own line and holds no value of the spec
func TestErrorOfRefusedSpecListsEachFindingAndHoldsNoValue(t *testing.T) {
	doc := mutate(t, exampleWiki,
		"summary: Reads and writes documents of an example wiki.", "summary: Reads {documents} of an example wiki.",
		docsDeleteExample, "")
	r := requireRefused(t, doc)
	lines := strings.Split(r.Err().Error(), "\n")
	if len(lines) != len(r.Findings) {
		t.Fatalf("error has %d lines for %d findings:\n%s", len(lines), len(r.Findings), r.Err())
	}
	if strings.Contains(r.Err().Error(), "example-wiki.docs.list") || strings.Contains(r.Err().Error(), "Reads") {
		t.Fatalf("the error text shows a value of the spec:\n%s", r.Err())
	}
}
