package mcp

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestListSourcesNamesTheActiveProfile checks F4 at the MCP surface: the list_sources answer names the profile that
// answered, so a model sees that no second profile was merged in.
func TestListSourcesNamesTheActiveProfile(t *testing.T) {
	svc := serviceWith(t, &fakeGateway{}, mustSource(t, "outline"))
	svc.Profile = "personal"
	var logs bytes.Buffer
	text, isErr := call(t, connect(t, svc, &logs), "list_sources", nil)
	if isErr {
		t.Fatalf("list_sources is an error: %q", text)
	}
	if !strings.Contains(text, `"profile":"personal"`) {
		t.Fatalf("list_sources answer = %q, want the personal profile", text)
	}
}

// TestListSourcesWithoutAProfileNamesNoProfile checks that a run with no profiles folder answers as before: the
// profile field stays empty.
func TestListSourcesWithoutAProfileNamesNoProfile(t *testing.T) {
	var logs bytes.Buffer
	text, isErr := call(t, connect(t, serviceWith(t, &fakeGateway{}, mustSource(t, "outline")), &logs), "list_sources", nil)
	if isErr {
		t.Fatalf("list_sources is an error: %q", text)
	}
	if !strings.Contains(text, `"profile":""`) {
		t.Fatalf("list_sources answer = %q, want an empty profile", text)
	}
}

// TestListSourcesRefusesAProfileArgument checks F2 at the MCP surface: a tool call with a profile argument is an
// unknown argument, because a run has one profile from its process and no call selects another. The SDK schema refuses
// it before the handler runs; the unit test below pins the handler's own refusal.
func TestListSourcesRefusesAProfileArgument(t *testing.T) {
	var logs bytes.Buffer
	text, isErr := call(t, connect(t, serviceWith(t, &fakeGateway{}, mustSource(t, "outline")), &logs),
		"list_sources", map[string]any{"profile": "work"})
	if !isErr || text == "" {
		t.Fatalf("list_sources with a profile argument = %q (error %v), want a refusal", text, isErr)
	}
}

// TestRefuseUnknownListArgsRefusesAProfileArgument pins the handler's own refusal: only kind and tags pass, and a
// profile argument is the unknown-input error the generated tools return.
func TestRefuseUnknownListArgsRefusesAProfileArgument(t *testing.T) {
	if err := refuseUnknownListArgs(map[string]any{"kind": "outline", "tags": []string{"wiki"}}); err != nil {
		t.Fatalf("refuseUnknownListArgs(kind, tags) = %v, want nil", err)
	}
	if err := refuseUnknownListArgs(map[string]any{"profile": "work"}); err != errUnknownInput {
		t.Fatalf("refuseUnknownListArgs(profile) = %v, want the unknown-input error", err)
	}
}

// TestCheckListArgsDecodesAndRefusesBeforeBinding covers the argument check: a declared input decodes, a malformed
// object and an unlisted input fail before anything is bound.
func TestCheckListArgsDecodesAndRefusesBeforeBinding(t *testing.T) {
	req := func(raw string) *sdk.CallToolRequest {
		return &sdk.CallToolRequest{Params: &sdk.CallToolParamsRaw{Arguments: json.RawMessage(raw)}}
	}
	args, err := checkListArgs(req(`{"kind":"outline"}`))
	if err != nil || args["kind"] != "outline" {
		t.Fatalf("checkListArgs(kind) = %v, %v; want the decoded arguments", args, err)
	}
	if _, err := checkListArgs(req(`[1]`)); err == nil {
		t.Fatal("checkListArgs(array) accepted a malformed object, want an error")
	}
	if _, err := checkListArgs(req(`{"profile":"work"}`)); err != errUnknownInput {
		t.Fatalf("checkListArgs(profile) = %v, want the unknown-input error", err)
	}
}
