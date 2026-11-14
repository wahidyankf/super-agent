package spec

import (
	"strings"
	"testing"
)

// The layer of a fault names the check that must refuse it. parse is the YAML reader, schema is the JSON Schema file,
// and semantic is a rule the schema cannot state, which only the Go validation checks.
const (
	layerParse    = "parse"
	layerSchema   = "schema"
	layerSemantic = "semantic"
)

// fault is one document that must be refused. The path is the JSON pointer that the finding must name. The echo is a
// value of the document that no finding may show, or "". The threats are the ADR threat ids that the row proves.
type fault struct {
	name    string
	layer   string
	threats []string
	path    string
	echo    string
	doc     func() string
}

// fx returns the bundled example spec with the given (old, repl) pairs applied.
func fx(pairs ...string) func() string {
	return func() string { return fixtureWith(pairs...) }
}

// mn returns the minimal spec with the given (old, repl) pairs applied.
func mn(pairs ...string) func() string {
	return func() string { return minimalWith(pairs...) }
}

// raw returns a document as it is.
func raw(doc string) func() string {
	return func() string { return doc }
}

// faults is the one table of refused documents. The threat-named tests, the fault test, and the agreement test with the
// schema all read it, so one corpus proves each claim.
var faults = []fault{
	// The YAML reader.
	{name: "an alias is refused", layer: layerParse, threats: []string{"S10"}, path: "/operations/1/summary",
		doc: fx("summary: Reads and writes documents of an example wiki.", "summary: &s Reads and writes documents of an example wiki.",
			"    summary: Read one document.", "    summary: *s")},
	{name: "an anchor without an alias is refused", layer: layerParse, threats: []string{"S10"}, path: "/summary",
		doc: fx("summary: Reads and writes documents of an example wiki.", "summary: &s Reads and writes documents of an example wiki.")},
	{name: "a duplicate key is refused", layer: layerParse, path: "/kind",
		doc: fx("version: 1\n", "version: 1\nkind: other\n")},
	{name: "a second YAML document is refused", layer: layerParse, threats: []string{"S10"}, path: "",
		doc: func() string { return exampleWiki + "---\nversion: 1\n" }},
	{name: "a syntax error is refused", layer: layerParse, path: "", doc: raw("version: [1\n")},
	{name: "a root that is not a mapping is refused", layer: layerParse, path: "", doc: raw("- one\n")},
	{name: "a key that is not text is refused", layer: layerParse, path: "", doc: fx("version: 1\n", "1: one\nversion: 1\n")},
	{name: "a custom tag is refused", layer: layerParse, path: "/summary",
		doc: fx("summary: Reads and writes", "summary: !custom Reads and writes")},
	{name: "a timestamp is refused", layer: layerParse, path: "/summary",
		doc: fx("summary: Reads and writes documents of an example wiki.", "summary: 2026-01-01")},
	{name: "a not-a-number value is refused", layer: layerParse, path: "/nan",
		doc: fx("version: 1\n", "version: 1\nnan: .nan\n")},
	{name: "a document over the 256 KiB cap is refused", layer: layerParse, threats: []string{"S10"}, path: "",
		doc: func() string { return exampleWiki + "# " + strings.Repeat("a", MaxBytes) }},
	{name: "a tagged mapping is refused", layer: layerParse, path: "/endpoint",
		doc: fx("endpoint:\n  base_path: /v1", "endpoint: !custom\n  base_path: /v1")},
	{name: "a tagged list is refused", layer: layerParse, path: "/endpoint/host_suffix",
		doc: fx("  host_suffix:\n", "  host_suffix: !custom\n")},
	{name: "a boolean that is not true or false is refused", layer: layerParse, path: "/operations/0/params/0/required",
		doc: fx("        required: false\n        help:", "        required: !!bool yes\n        help:")},
	{name: "an integer written with a fraction is refused", layer: layerParse, path: "/operations/0/params/1/default",
		doc: fx("        default: 20", "        default: !!int 2.5")},
	{name: "an infinite number is refused", layer: layerParse, path: "/inf",
		doc: fx("version: 1\n", "version: 1\ninf: !!float inf\n")},

	// The JSON Schema file.
	{name: "a template function in a body is refused", layer: layerSchema, threats: []string{"S1"},
		path: "/operations/2/body/title", echo: "upper(title)",
		doc: fx("      title: \"{{title}}\"", "      title: \"{{ upper(title) }}\"")},
	{name: "an expression in a path slot is refused", layer: layerSchema, threats: []string{"S1"},
		path: "/operations/1/path", echo: "{{ id }}",
		doc: fx("    path: /docs/{id}\n    params:", "    path: /docs/{{ id }}\n    params:")},
	{name: "a path slot in a text is refused", layer: layerSchema, threats: []string{"S1"},
		path: "/operations/1/path", echo: "x-{id}",
		doc: fx("    path: /docs/{id}\n    params:", "    path: /docs/x-{id}\n    params:")},
	{name: "a path param typed as a query is refused", layer: layerSchema, threats: []string{"S1"},
		path: "/operations/1/params/0/type",
		doc:  fx("        type: ID\n        in: path", "        type: Query\n        in: path")},
	{name: "a path that climbs with dots is refused", layer: layerSchema, threats: []string{"S1", "S7"},
		path: "/operations/1/path", echo: "/../{id}",
		doc: fx("    path: /docs/{id}\n    params:", "    path: /../{id}\n    params:")},
	{name: "a path with a line break is refused", layer: layerSchema, threats: []string{"S1", "S7"},
		path: "/operations/1/path",
		doc:  fx("    path: /docs/{id}\n    params:", "    path: \"/docs/{id}\\n\"\n    params:")},
	{name: "a credential in the query string of a path is refused", layer: layerSchema, threats: []string{"S8"},
		path: "/operations/0/path", echo: "api_key=x",
		doc: fx("    path: /docs\n    params:", "    path: /docs?api_key=x\n    params:")},
	{name: "a credential-named query parameter is refused", layer: layerSchema, threats: []string{"S8"},
		path: "/operations/0/params/0/name",
		doc:  fx("      - name: cursor\n        type: ID\n        in: query", "      - name: api_key\n        type: ID\n        in: query")},
	{name: "a credential name in a body slot is refused", layer: layerSchema, threats: []string{"S8"},
		path: "/operations/2/body/text", echo: "EXAMPLE_WIKI_API_KEY",
		doc: fx("      text: \"{{text}}\"", "      text: \"{{EXAMPLE_WIKI_API_KEY}}\"")},
	{name: "an Authorization header declared by the auth block is refused", layer: layerSchema, threats: []string{"S6", "S8"},
		path: "/auth/header_name",
		doc: fx("  type: bearer\n  credential_env: EXAMPLE_WIKI_API_KEY",
			"  type: header\n  credential_env: EXAMPLE_WIKI_API_KEY\n  header_name: Authorization")},
	{name: "an Authorization header on an operation is refused", layer: layerSemantic, threats: []string{"S6"},
		path: "/operations/1/headers/Authorization",
		doc: fx("    response:\n      fields:\n        title:",
			"    headers:\n      Authorization: Basic-abc\n    response:\n      fields:\n        title:")},
	{name: "a header value built from a parameter is refused", layer: layerSchema, threats: []string{"S6"},
		path: "/operations/1/headers/X-Tenant", echo: "tenant-{id}",
		doc: fx("    response:\n      fields:\n        title:",
			"    headers:\n      X-Tenant: tenant-{id}\n    response:\n      fields:\n        title:")},
	{name: "a header value with a control character is refused", layer: layerSchema, threats: []string{"S6"},
		path: "/operations/1/headers/X-Trace",
		doc: fx("    response:\n      fields:\n        title:",
			"    headers:\n      X-Trace: \"a\\u0001b\"\n    response:\n      fields:\n        title:")},
	{name: "a header value that is a number is refused", layer: layerSchema, path: "/operations/1/headers/X-Tenant",
		doc: fx("    response:\n      fields:\n        title:", "    headers:\n      X-Tenant: 5\n    response:\n      fields:\n        title:")},
	{name: "an unknown field of an operation is refused", layer: layerSchema, path: "/operations/1/verb",
		doc: fx("    summary: Read one document.\n", "    summary: Read one document.\n    verb: get\n")},
	{name: "an unknown top-level field is refused", layer: layerSchema, path: "/extra",
		doc: fx("endpoint:\n", "extra: 1\nendpoint:\n")},
	{name: "a wrong version is refused", layer: layerSchema, path: "/version",
		doc: fx("version: 1\n", "version: 2\n")},
	{name: "a version written as text is refused", layer: layerSchema, path: "/version",
		doc: fx("version: 1\n", "version: \"1\"\n")},
	{name: "an endpoint that is an https URL is refused", layer: layerSchema, path: "/endpoint/base_path",
		echo: "https://example.test/v1",
		doc:  fx("  base_path: /v1", "  base_path: https://example.test/v1")},
	{name: "an access label that does not exist is refused", layer: layerSchema, threats: []string{"S11"},
		path: "/operations/0/access",
		doc:  fx("    access: read\n    method: GET\n    path: /docs\n    params:", "    access: owner\n    method: GET\n    path: /docs\n    params:")},
	{name: "DELETE labelled read is refused", layer: layerSchema, threats: []string{"S11"},
		path: "/operations/3/method",
		doc:  fx("    access: write\n    method: DELETE", "    access: read\n    method: DELETE")},
	{name: "an unknown converter is refused", layer: layerSchema, path: "/operations/1/response/fields/body/convert",
		doc: fx("          convert: markdown-from-html", "          convert: markdown-from-pdf")},
	{name: "pagination without a stop rule is refused", layer: layerSchema, path: "/operations/0/pagination/max_pages",
		doc: fx("      style: cursor\n      max_pages: 5\n", "      style: cursor\n")},
	{name: "a page limit above its cap is refused", layer: layerSchema, path: "/limits/max_pages",
		doc: fx("  max_pages: 5\noperations:", "  max_pages: 21\noperations:")},
	{name: "a timeout above its cap is refused", layer: layerSchema, path: "/limits/timeout_seconds",
		doc: fx("  timeout_seconds: 20", "  timeout_seconds: 31")},
	{name: "a summary with braces is refused", layer: layerSchema, path: "/summary", echo: "{documents}",
		doc: fx("summary: Reads and writes documents of an example wiki.", "summary: Reads {documents} of an example wiki.")},
	{name: "a summary shaped like a token is refused", layer: layerSchema, threats: []string{"R2"}, path: "/summary",
		echo: "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdef0123456789",
		doc: fx("summary: Reads and writes documents of an example wiki.",
			"summary: Key ABCDEFGHIJKLMNOPQRSTUVWXYZabcdef0123456789 here.")},
	{name: "a bearer-shaped string in an example is refused", layer: layerSchema, threats: []string{"R2"},
		path: "/operations/1/examples/0/response/body/title", echo: "Bearer abcdefghijklmnopqrstuvwxyz0123456789",
		doc: fx("          status: 200\n          body:\n            title: First document",
			"          status: 200\n          body:\n            title: Bearer abcdefghijklmnopqrstuvwxyz0123456789")},
	{name: "an Authorization header in an example reply is refused", layer: layerSemantic, threats: []string{"R2"},
		path: "/operations/1/examples/0/response/headers/Authorization",
		doc: fx("          status: 200\n          body:\n            title: First document",
			"          status: 200\n          headers:\n            Authorization: Basic-abc\n          body:\n            title: First document")},
	{name: "a missing summary is refused", layer: layerSchema, path: "/operations/1/summary",
		doc: fx("    summary: Read one document.\n", "")},
	{name: "an operation with no example is refused", layer: layerSchema, path: "/operations/3/examples",
		doc: fx("    examples:\n      - request:\n          id: doc-1\n        response:\n          status: 204\n        expect: {}",
			"    examples: []")},
	{name: "a status outside 100 to 599 is refused", layer: layerSchema, path: "/operations/3/examples/0/response/status",
		doc: fx("          status: 204", "          status: 99")},
	{name: "an Enum parameter with no values is refused", layer: layerSchema, path: "/operations/0/params/1/values",
		doc: fx("        type: Limit", "        type: Enum")},
	{name: "a parameter kind that does not exist is refused", layer: layerSchema, path: "/operations/0/params/1/type",
		doc: fx("        type: Limit", "        type: Secret")},
	{name: "basic auth with no identity variable is refused", layer: layerSchema, path: "/auth/identity_env",
		doc: fx("  type: bearer", "  type: basic")},
	{name: "bearer auth with an identity variable is refused", layer: layerSchema, path: "/auth",
		doc: fx("  credential_env: EXAMPLE_WIKI_API_KEY", "  credential_env: EXAMPLE_WIKI_API_KEY\n  identity_env: EXAMPLE_WIKI_EMAIL")},
	{name: "a custom command name that is not a slug is refused", layer: layerSchema, path: "/operations/0/cli",
		doc: fx("    path: /docs\n    params:", "    path: /docs\n    cli: Not A Slug\n    params:")},
	{name: "a spec with no operation is refused", layer: layerSchema, path: "/operations",
		doc: raw("version: 1\nkind: example-wiki\nsummary: Reads example documents.\nendpoint: {}\nauth:\n  type: bearer\n  credential_env: EXAMPLE_WIKI_API_KEY\noperations: []\n")},

	// Rules that only the Go validation states, because a JSON Schema cannot compare two fields of one document.
	{name: "a built-in kind is refused", layer: layerSemantic, threats: []string{"S10"}, path: "/kind",
		doc: fx("kind: example-wiki", "kind: outline")},
	{name: "an operation id outside the kind is refused", layer: layerSemantic, path: "/operations/0/id",
		doc: fx("  - id: example-wiki.docs.list", "  - id: other.docs.list")},
	{name: "a duplicate operation id is refused", layer: layerSemantic, path: "/operations/1/id",
		doc: fx("  - id: example-wiki.docs.get", "  - id: example-wiki.docs.list")},
	{name: "a path slot with no declared parameter is refused", layer: layerSemantic, path: "/operations/1/path",
		doc: fx("    path: /docs/{id}\n    params:", "    path: /docs/{key}\n    params:")},
	{name: "a path parameter the path does not use is refused", layer: layerSemantic, path: "/operations/1/params/0/name",
		doc: fx("    path: /docs/{id}\n    params:", "    path: /docs\n    params:")},
	{name: "a body slot that names no body parameter is refused", layer: layerSemantic, path: "/operations/2/body/title",
		doc: fx("      title: \"{{title}}\"", "      title: \"{{nope}}\"")},
	{name: "a header slot that names no parameter is refused", layer: layerSemantic, path: "/operations/1/headers/X-Tenant",
		doc: fx("    response:\n      fields:\n        title:", "    headers:\n      X-Tenant: \"{nope}\"\n    response:\n      fields:\n        title:")},
	{name: "a body slot inside a list names no body parameter", layer: layerSemantic, path: "/operations/2/body/labels/1",
		doc: fx("      text: \"{{text}}\"", "      text: \"{{text}}\"\n      labels: [\"one\", \"{{nope}}\"]")},
	{name: "a parameter name repeated in one operation is refused", layer: layerSemantic, path: "/operations/0/params/1/name",
		doc: fx("      - name: limit\n", "      - name: cursor\n")},
	{name: "a pagination name that is no query parameter is refused", layer: layerSemantic, path: "/operations/0/pagination/cursor_param",
		doc: fx("      cursor_param: cursor", "      cursor_param: nope")},
	{name: "a resolve that names no operation is refused", layer: layerSemantic, path: "/operations/1/params/0/resolve/operation",
		doc: fx("    params:\n      - name: id\n        type: ID\n        in: path",
			"    params:\n      - name: id\n        type: ID\n        in: path\n        resolve:\n          operation: example-wiki.docs.missing\n          field: title")},
	{name: "a resolve that names a write is refused", layer: layerSemantic, path: "/operations/1/params/0/resolve/operation",
		doc: fx("    params:\n      - name: id\n        type: ID\n        in: path",
			"    params:\n      - name: id\n        type: ID\n        in: path\n        resolve:\n          operation: example-wiki.docs.create\n          field: title")},
	{name: "a resolve that names no field of its operation is refused", layer: layerSemantic,
		path: "/operations/1/params/0/resolve/field",
		doc: fx("    params:\n      - name: id\n        type: ID\n        in: path",
			"    params:\n      - name: id\n        type: ID\n        in: path\n        resolve:\n          operation: example-wiki.docs.list\n          field: nope")},
	{name: "markdown that names no field is refused", layer: layerSemantic, path: "/operations/1/response/markdown",
		doc: fx("      markdown: body", "      markdown: nope")},
	{name: "an example request that names no parameter is refused", layer: layerSemantic,
		path: "/operations/1/examples/0/request/other",
		doc:  fx("request:\n          id: doc-1\n        response:", "request:\n          other: doc-1\n        response:")},
	{name: "an example expectation that names no field is refused", layer: layerSemantic,
		path: "/operations/1/examples/0/expect/heading",
		doc:  fx("        expect:\n          title: First document\n          body:", "        expect:\n          heading: First document\n          body:")},
}

// rowsWithThreat returns the fault rows that prove the given threat id.
func rowsWithThreat(id string) []fault {
	var rows []fault
	for _, f := range faults {
		for _, t := range f.threats {
			if t == id {
				rows = append(rows, f)
				break
			}
		}
	}
	return rows
}

// runFaults refuses every row of the list, each with the stable code, its path, and no echoed value.
func runFaults(t *testing.T, rows []fault) {
	t.Helper()
	if len(rows) == 0 {
		t.Fatal("no fault row proves this threat")
	}
	for _, f := range rows {
		t.Run(f.name, func(t *testing.T) {
			requireRefused(t, []byte(f.doc()), f.path, f.echo)
		})
	}
}

// TestFaultsAreRefusedWithTheirFieldPath refuses each fault of the table with the stable code and the field path that
// names it, and it shows no value of the document.
func TestFaultsAreRefusedWithTheirFieldPath(t *testing.T) {
	runFaults(t, faults)
}

// TestS1_TemplateCodeAndTypedSlotsAreRefused proves S1: a template holds only typed slots, and no function, expression,
// or text interpolation reaches a request. The engine checks a parameter value that looks like a slot, in its own PR.
func TestS1_TemplateCodeAndTypedSlotsAreRefused(t *testing.T) {
	runFaults(t, rowsWithThreat("S1"))
}

// TestS8_KeyNeverLeavesItsHeader proves S8: no spec puts the key in a URL, a query, or a body, and no spec names a
// header that the key must not take. The stub that records every byte is the engine's test, in its own PR.
func TestS8_KeyNeverLeavesItsHeader(t *testing.T) {
	runFaults(t, rowsWithThreat("S8"))
}

// TestS10_SpecIsOneBoundedYAMLDocument proves S10: a spec is one YAML document of at most 256 KiB, with no alias or
// anchor, and it may not take the kind of a built-in source. A document of exactly the cap is read.
func TestS10_SpecIsOneBoundedYAMLDocument(t *testing.T) {
	runFaults(t, rowsWithThreat("S10"))
	atCap := exampleWiki + "# "
	atCap += strings.Repeat("a", MaxBytes-len(atCap))
	if len(atCap) != MaxBytes {
		t.Fatalf("test document is %d bytes; want %d", len(atCap), MaxBytes)
	}
	requireValid(t, atCap)
}

// TestS11_DeleteWithReadIsRefusedAndTheReviewListsEveryOperation proves S11: a spec that lies about access fails
// validation where the label rules it out, and the review lists each operation with its label and method, so a person
// reads what the spec can do.
func TestS11_DeleteWithReadIsRefusedAndTheReviewListsEveryOperation(t *testing.T) {
	runFaults(t, rowsWithThreat("S11"))
	r := requireValid(t, exampleWiki)
	review := Review(r.Spec)
	for _, op := range r.Spec.Operations {
		if !strings.Contains(review, op.ID) || !strings.Contains(review, op.Method) || !strings.Contains(review, op.Access) {
			t.Fatalf("the review does not list operation %s with its label and method:\n%s", op.ID, review)
		}
	}
}

// TestR2_ExampleCredentialIsRefusedAndNeverShown proves R2: an example holds no authorization header and no value
// shaped like a token, and the output names the field path, never the value.
func TestR2_ExampleCredentialIsRefusedAndNeverShown(t *testing.T) {
	runFaults(t, rowsWithThreat("R2"))
}
