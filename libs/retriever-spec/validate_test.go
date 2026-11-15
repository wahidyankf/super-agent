package spec

import (
	"strings"
	"testing"
)

// TestValidMinimalSpecHasNoFindingAndItsReviewListsEachOperation proves the first acceptance of PR 2: a valid minimal
// spec has no finding, and the review lists each operation of it.
func TestValidMinimalSpecHasNoFindingAndItsReviewListsEachOperation(t *testing.T) {
	r := requireValid(t, minimalSpec)
	if len(r.Findings) != 0 {
		t.Fatalf("findings = %v; want none", r.Findings)
	}
	if r.Err() != nil {
		t.Fatalf("Err() = %v; want nil for a valid spec", r.Err())
	}
	if r.Spec.Kind != "example-wiki" || len(r.Spec.Operations) != 1 {
		t.Fatalf("model = kind %q with %d operations; want example-wiki with 1", r.Spec.Kind, len(r.Spec.Operations))
	}
	review := Review(r.Spec)
	for _, op := range r.Spec.Operations {
		if !strings.Contains(review, op.ID) {
			t.Fatalf("the review does not list %s:\n%s", op.ID, review)
		}
	}
}

// TestValidBundledExampleHasNoFinding checks that the example spec of the tests passes every rule, and that its model
// keeps each field the document names.
func TestValidBundledExampleHasNoFinding(t *testing.T) {
	r := requireValid(t, exampleWiki)
	s := r.Spec
	if s.Version != Version || s.Kind != "example-wiki" {
		t.Fatalf("version %d kind %q; want %d example-wiki", s.Version, s.Kind, Version)
	}
	if s.Endpoint.BasePath != "/v1" || len(s.Endpoint.HostSuffix) != 1 || s.Endpoint.HostSuffix[0] != ".example.test" {
		t.Fatalf("endpoint = %+v; want base /v1 and host suffix .example.test", s.Endpoint)
	}
	if s.Auth.Type != "bearer" || s.Auth.CredentialEnv != "EXAMPLE_WIKI_API_KEY" {
		t.Fatalf("auth = %+v; want bearer with EXAMPLE_WIKI_API_KEY", s.Auth)
	}
	if s.Limits.TimeoutSeconds != 20 || s.Limits.MaxPages != 5 || s.Limits.MaxReplyBytes != 0 {
		t.Fatalf("limits = %+v; want timeout 20, pages 5, and no reply limit", s.Limits)
	}
	if len(s.Operations) != 4 {
		t.Fatalf("%d operations; want 4", len(s.Operations))
	}
	list := s.Operations[0]
	if list.Pagination == nil || list.Pagination.Style != "cursor" || list.Pagination.NextPointer != "/next_cursor" {
		t.Fatalf("pagination = %+v; want the cursor read", list.Pagination)
	}
	if !list.CLI.Enabled || !list.MCP.Enabled {
		t.Fatal("cli and mcp must be on by default")
	}
	if get := s.Operations[1]; get.Response.Markdown != "body" || get.Response.Fields["body"].Convert != "markdown-from-html" {
		t.Fatalf("the get read = %+v; want markdown from body with markdown-from-html", get.Response)
	}
	if del := s.Operations[3]; del.Access != "write" || del.Method != "DELETE" {
		t.Fatalf("the delete = access %q method %q; want write DELETE", del.Access, del.Method)
	}
}

// TestC1_ReviewOfTheFixtureListsEveryOperationWithLabelMethodAndPath proves C1: the review of a fixture spec lists every
// operation with its label, method, and path, so a person can read what a hostile spec would do before they run it.
func TestC1_ReviewOfTheFixtureListsEveryOperationWithLabelMethodAndPath(t *testing.T) {
	r := requireValid(t, exampleWiki)
	review := Review(r.Spec)
	for _, op := range r.Spec.Operations {
		path := r.Spec.Endpoint.BasePath + op.Path
		line := op.ID + ": " + op.Access + ", " + op.Method + " " + path
		if !strings.Contains(review, line) {
			t.Fatalf("the review has no line %q for the operation:\n%s", line, review)
		}
	}
}

// TestAFixtureWithOneFaultFailsOnlyAtItsField checks that one fault adds one finding, at the field that holds it, and
// that the rest of the fixture stays valid.
func TestAFixtureWithOneFaultFailsOnlyAtItsField(t *testing.T) {
	r := Validate([]byte(fixtureWith("  type: bearer", "  type: header\n  header_name: Authorization")))
	if r.Valid() {
		t.Fatal("a header named Authorization must fail")
	}
	if len(r.Findings) != 1 || r.Findings[0].Path != "/auth/header_name" {
		t.Fatalf("findings = %v; want one finding at /auth/header_name", r.Findings)
	}
}

// TestACanonicalUUIDInAnExampleIsNotATokenRun proves the R2 exemption PR21
// needs: a canonical UUID in an example is an ID, not a token-shaped run, so
// a UUID operation can carry a replayable example. A 32-character run that is
// not a UUID, and a UUID with text around it, stay refused.
func TestACanonicalUUIDInAnExampleIsNotATokenRun(t *testing.T) {
	const uuid = "123e4567-e89b-12d3-a456-426614174000"
	withIDs := mutateAll(minimalSpec, "- id: doc-1", "- id: "+uuid, "id: doc-1", "id: "+uuid)
	if r := Validate([]byte(withIDs)); !r.Valid() {
		t.Fatalf("findings = %v; want none for a canonical UUID", r.Findings)
	}
	const run = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdef"
	withRun := mutateAll(minimalSpec, "- id: doc-1", "- id: "+run, "id: doc-1", "id: "+run)
	if r := Validate([]byte(withRun)); r.Valid() || !hasFinding(r, "/operations/0/examples/0/response/body/items/0/id") {
		t.Fatalf("findings = %v; want a refusal of the token-shaped run", r.Findings)
	}
	withAffix := mutateAll(minimalSpec, "- id: doc-1", "- id: id-"+uuid, "id: doc-1", "id: id-"+uuid)
	if r := Validate([]byte(withAffix)); r.Valid() {
		t.Fatalf("a UUID with text around it must stay refused")
	}
}
