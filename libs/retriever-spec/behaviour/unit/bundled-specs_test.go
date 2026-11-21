package unit_test

import (
	"testing"

	spec "github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// bundledValidate runs the bundled loader on a document.
func bundledValidate(doc string) spec.Result {
	return spec.ParseBundled([]byte(doc))
}

// requireBundledRefusedAt fails the test unless the bundled loader refuses the document with a finding at exactly path.
func requireBundledRefusedAt(t *testing.T, doc string, path string) spec.Result {
	t.Helper()
	r := bundledValidate(doc)
	if r.Valid() || r.Spec != nil {
		t.Fatalf("the bundled spec is not refused; want a finding at %q", path)
	}
	if !hasFinding(r, path) {
		t.Fatalf("no finding at %q; findings:\n%s", path, text(r))
	}
	return r
}

// @scenario specs/libs/retriever-spec/behaviours/bundled-specs.feature: The bundled loader accepts a bundled spec whose kind is the built-in source kind jira
func TestBundledLoaderAcceptsBundledSpecOfBuiltInKindJira(t *testing.T) {
	r := bundledValidate(bundledJiraStub)
	if !r.Valid() || r.Spec == nil {
		t.Fatalf("the bundled loader refuses the bundled jira spec:\n%s", text(r))
	}
	if r.Spec.Kind != "jira" || len(r.Spec.Operations) != 3 {
		t.Fatalf("model = kind %q with %d operations; want jira with 3", r.Spec.Kind, len(r.Spec.Operations))
	}
}

// @scenario specs/libs/retriever-spec/behaviours/bundled-specs.feature: The user loader refuses the same bundled jira spec at its kind
func TestUserLoaderRefusesTheSameBundledJiraSpecAtItsKind(t *testing.T) {
	requireRefusedAt(t, bundledJiraStub, "/kind")
}

// @scenario specs/libs/retriever-spec/behaviours/bundled-specs.feature: The user loader refuses a catalog_id on an operation of a user spec
func TestUserLoaderRefusesCatalogIDOnOperationOfUserSpec(t *testing.T) {
	doc := mutate(t, exampleWiki, "    summary: List documents.\n", "    summary: List documents.\n    catalog_id: example-wiki.docs.list\n")
	requireRefusedAt(t, doc, "/operations/0/catalog_id")
}

// @scenario specs/libs/retriever-spec/behaviours/bundled-specs.feature: A bundled operation whose parameters match its catalog row is accepted
func TestBundledOperationWhoseParametersMatchItsCatalogRowIsAccepted(t *testing.T) {
	r := bundledValidate(bundledJiraStub)
	if !r.Valid() {
		t.Fatalf("the bundled loader refuses a matching bundled operation:\n%s", text(r))
	}
}

// @scenario specs/libs/retriever-spec/behaviours/bundled-specs.feature: A bundled operation that differs from its catalog row is refused at the field that differs
func TestBundledOperationThatDiffersFromItsCatalogRowIsRefusedAtTheFieldThatDiffers(t *testing.T) {
	cases := []struct {
		name, doc, path string
	}{
		{"a parameter of another type", mutate(t, bundledJiraStub,
			"      - name: limit\n        type: Int", "      - name: limit\n        type: Text"), "/operations/1/params/1/type"},
		{"a parameter that flips its required flag", mutate(t, bundledJiraStub,
			"      - name: limit\n        type: Int\n        in: query\n        required: false",
			"      - name: limit\n        type: Int\n        in: query\n        required: true"), "/operations/1/params/1/required"},
		{"a parameter name the row does not name", mutate(t, bundledJiraStub,
			"      - name: jql\n        type: Query", "      - name: jqlx\n        type: Query"), "/operations/2/params/0/name"},
		{"a row parameter the operation omits", mutate(t, bundledJiraStub,
			"      - name: offset\n        type: Offset\n        in: query\n        required: false\n        default: 0\n", ""),
			"/operations/1/params"},
		{"a catalog_id of another built-in kind", mutate(t, bundledJiraStub,
			"    catalog_id: jira.projects.list", "    catalog_id: kiwi.products.list"), "/operations/1/catalog_id"},
		{"a catalog_id that matches no catalog row", mutate(t, bundledJiraStub,
			"    catalog_id: jira.projects.list", "    catalog_id: jira.no-such-operation"), "/operations/1/catalog_id"},
		{"a catalog_id that another operation claims", mutate(t, bundledJiraStub,
			"    catalog_id: jira.issues.search", "    catalog_id: jira.whoami"), "/operations/2/catalog_id"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			requireBundledRefusedAt(t, c.doc, c.path)
		})
	}
}

// @scenario specs/libs/retriever-spec/behaviours/bundled-specs.feature: A bundled operation with no catalog row that turns off both surfaces is accepted
func TestBundledOperationWithNoCatalogRowThatTurnsOffBothSurfacesIsAccepted(t *testing.T) {
	r := bundledValidate(bundledHiddenStub)
	if !r.Valid() {
		t.Fatalf("the bundled loader refuses a hidden operation with both surfaces off:\n%s", text(r))
	}
}

// @scenario specs/libs/retriever-spec/behaviours/bundled-specs.feature: A bundled operation with no catalog row that leaves one surface on is refused at that surface
func TestBundledOperationWithNoCatalogRowThatLeavesOneSurfaceOnIsRefusedAtThatSurface(t *testing.T) {
	cases := []struct {
		name, repl, path string
	}{
		{"command-line", "    cli: true\n    mcp: false", "/operations/0/cli"},
		{"MCP", "    cli: false\n    mcp: true", "/operations/0/mcp"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			doc := mutate(t, bundledHiddenStub, "    cli: false\n    mcp: false", c.repl)
			requireBundledRefusedAt(t, doc, c.path)
		})
	}
}
