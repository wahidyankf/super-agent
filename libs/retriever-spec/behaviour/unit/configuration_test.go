package unit_test

import (
	"testing"
)

// @scenario specs/libs/retriever-spec/behaviours/configuration.feature: A base path or host suffix that does not fit its form is refused at that setting
func TestBasePathOrHostSuffixThatDoesNotFitItsFormIsRefusedAtThatSetting(t *testing.T) {
	cases := []struct {
		name, old, repl, path string
	}{
		{"the base path is v1, with no leading slash", "  base_path: /v1", "  base_path: v1", "/endpoint/base_path"},
		{"the base path is an https address", "  base_path: /v1", "  base_path: https://example.test/v1", "/endpoint/base_path"},
		{"the first host suffix is example.test, with no dot", "    - .example.test", "    - example.test", "/endpoint/host_suffix/0"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			requireRefusedAt(t, mutate(t, exampleWiki, c.old, c.repl), c.path)
		})
	}
}

// @scenario specs/libs/retriever-spec/behaviours/configuration.feature: An endpoint with a base path and a dotted host suffix is accepted
func TestEndpointWithBasePathAndDottedHostSuffixIsAccepted(t *testing.T) {
	requireAccepted(t, exampleWiki)
}

// @scenario specs/libs/retriever-spec/behaviours/configuration.feature: An auth type is accepted with exactly the variable names that it needs
func TestAuthTypeIsAcceptedWithExactlyTheVariableNamesItNeeds(t *testing.T) {
	const bearer = "  type: bearer\n  credential_env: EXAMPLE_WIKI_API_KEY\n"
	cases := []struct {
		name string
		doc  string
	}{
		{"bearer with the name of the key variable", exampleWiki},
		{"basic with the names of the key variable and the identity variable", mutate(t, exampleWiki, bearer,
			"  type: basic\n  credential_env: EXAMPLE_WIKI_API_KEY\n  identity_env: EXAMPLE_WIKI_EMAIL\n")},
		{"header with the name of the key variable and a header name", mutate(t, exampleWiki, bearer,
			"  type: header\n  credential_env: EXAMPLE_WIKI_API_KEY\n  header_name: X-Wiki-Key\n")},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			requireAccepted(t, c.doc)
		})
	}
}

// @scenario specs/libs/retriever-spec/behaviours/configuration.feature: An auth type that lacks or adds a variable name it does not take is refused at the auth
func TestAuthTypeThatLacksOrAddsAVariableNameItDoesNotTakeIsRefusedAtTheAuth(t *testing.T) {
	const bearer = "  type: bearer\n  credential_env: EXAMPLE_WIKI_API_KEY\n"
	cases := []struct {
		name, doc, path string
	}{
		{"basic with no identity variable", mutate(t, exampleWiki, bearer,
			"  type: basic\n  credential_env: EXAMPLE_WIKI_API_KEY\n"), "/auth/identity_env"},
		{"bearer that also names an identity variable", mutate(t, exampleWiki, bearer,
			"  type: bearer\n  credential_env: EXAMPLE_WIKI_API_KEY\n  identity_env: EXAMPLE_WIKI_EMAIL\n"), "/auth"},
		{"header with no header name", mutate(t, exampleWiki, bearer,
			"  type: header\n  credential_env: EXAMPLE_WIKI_API_KEY\n"), "/auth/header_name"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			requireRefusedAt(t, c.doc, c.path)
		})
	}
}

// @scenario specs/libs/retriever-spec/behaviours/configuration.feature: A spec that names a plugin is refused at the plugin field
func TestSpecThatNamesAPluginIsRefusedAtThePluginField(t *testing.T) {
	doc := mutate(t, exampleWiki, "operations:\n", "plugin:\n  path: ./example-plugin\noperations:\n")
	requireRefusedAt(t, doc, "/plugin")
}

// @scenario specs/libs/retriever-spec/behaviours/configuration.feature: A limit at or below its built-in cap is accepted
func TestLimitAtOrBelowItsBuiltInCapIsAccepted(t *testing.T) {
	cases := []struct {
		name, doc string
	}{
		{"the timeout of 30 seconds", mutate(t, exampleWiki, "timeout_seconds: 20", "timeout_seconds: 30")},
		{"the reply size of 10485760 bytes", mutate(t, exampleWiki, "limits:\n  timeout_seconds: 20\n",
			"limits:\n  timeout_seconds: 20\n  max_reply_bytes: 10485760\n")},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			requireAccepted(t, c.doc)
		})
	}
}

// @scenario specs/libs/retriever-spec/behaviours/configuration.feature: A limit above its built-in cap is refused at that limit
func TestLimitAboveItsBuiltInCapIsRefusedAtThatLimit(t *testing.T) {
	cases := []struct {
		name, old, repl, path string
	}{
		{"the timeout of 31", "timeout_seconds: 20", "timeout_seconds: 31", "/limits/timeout_seconds"},
		{"the reply size of 10485761", "limits:\n  timeout_seconds: 20\n",
			"limits:\n  timeout_seconds: 20\n  max_reply_bytes: 10485761\n", "/limits/max_reply_bytes"},
		{"the page cap of 21", "limits:\n  timeout_seconds: 20\n  max_pages: 5\n",
			"limits:\n  timeout_seconds: 20\n  max_pages: 21\n", "/limits/max_pages"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			requireRefusedAt(t, mutate(t, exampleWiki, c.old, c.repl), c.path)
		})
	}
}

// @scenario specs/libs/retriever-spec/behaviours/configuration.feature: An operation that turns off its command-line surface is accepted
func TestOperationThatTurnsOffItsCommandLineSurfaceIsAccepted(t *testing.T) {
	requireAccepted(t, mutate(t, exampleWiki, "    path: /docs\n    params:", "    path: /docs\n    cli: false\n    params:"))
}

// @scenario specs/libs/retriever-spec/behaviours/configuration.feature: An operation that names its command-line and MCP surfaces is accepted
func TestOperationThatNamesItsCommandLineAndMCPSurfacesIsAccepted(t *testing.T) {
	requireAccepted(t, mutate(t, exampleWiki, "    path: /docs\n    params:",
		"    path: /docs\n    cli: find-docs\n    mcp: find-docs\n    params:"))
}

// @scenario specs/libs/retriever-spec/behaviours/configuration.feature: A surface name that is not a lower-case slug is refused at the surface
func TestSurfaceNameThatIsNotALowerCaseSlugIsRefusedAtTheSurface(t *testing.T) {
	cases := []struct {
		name, key, value, path string
	}{
		{"command-line name Not A Slug", "cli", "Not A Slug", "/operations/0/cli"},
		{"MCP name Find Docs", "mcp", "Find Docs", "/operations/0/mcp"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			doc := mutate(t, exampleWiki, "    path: /docs\n    params:", "    path: /docs\n    "+c.key+": "+c.value+"\n    params:")
			requireRefusedAt(t, doc, c.path)
		})
	}
}

// @scenario specs/libs/retriever-spec/behaviours/configuration.feature: An operation that omits both surface keys shows both surfaces on in the review
func TestOperationThatOmitsBothSurfaceKeysShowsBothSurfacesOnInTheReview(t *testing.T) {
	const want = "1. example-wiki.docs.list: read, GET /v1/docs, writes no, cli on, mcp on"
	if !hasLine(reviewOf(t, exampleWiki), want) {
		t.Fatalf("the review has no line %q", want)
	}
}
