package mcp

import (
	"context"
	"encoding/json"
	"slices"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/sources"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/sources/operations"
)

// kindSource returns one source of the given kind, with the endpoint and key name of the catalog.
func kindSource(t *testing.T, kind, id, host, credential string) domain.Source {
	t.Helper()
	sid, err := domain.NewSourceID(id)
	if err != nil {
		t.Fatal(err)
	}
	k, err := domain.NewSourceKind(kind)
	if err != nil {
		t.Fatal(err)
	}
	tags, err := domain.NewTagSet("testing")
	if err != nil {
		t.Fatal(err)
	}
	endpoint, err := domain.NewEndpoint("https://" + host)
	if err != nil {
		t.Fatal(err)
	}
	ref, err := domain.NewCredentialRef(credential)
	if err != nil {
		t.Fatal(err)
	}
	return domain.NewSource(sid, k, tags, endpoint, ref)
}

// threeKindService returns a catalog with an Outline source, a Kiwi source, and a Jira source with its identity name.
// Every gateway answers from gw.
func threeKindService(t *testing.T, gw *fakeGateway) sources.Service {
	t.Helper()
	identity, err := domain.NewIdentityRef("EXAMPLE_JIRA_EMAIL")
	if err != nil {
		t.Fatal(err)
	}
	jira := kindSource(t, "jira", "jira", "jira.example.test", "EXAMPLE_JIRA_API_TOKEN").WithIdentity(identity)
	return serviceWith(t, gw,
		mustSource(t, "outline"),
		kindSource(t, "kiwi", "kiwi", "kiwi.example.test", "EXAMPLE_KIWI_API_KEY"),
		jira,
	)
}

// TestKindsOtherThanOutlineAddNoTool checks that a catalog with Kiwi and Jira sources offers exactly the tools of an
// Outline-only catalog.
func TestKindsOtherThanOutlineAddNoTool(t *testing.T) {
	outlineOnly := serviceWith(t, &fakeGateway{}, mustSource(t, "outline"))
	all := threeKindService(t, &fakeGateway{})
	want, err := RegisteredTools(outlineOnly, nil)
	if err != nil {
		t.Fatalf("RegisteredTools(outline only): %v", err)
	}
	got, err := RegisteredTools(all, nil)
	if err != nil {
		t.Fatalf("RegisteredTools(three kinds): %v", err)
	}
	if !slices.Equal(sortedNames(got), sortedNames(want)) {
		t.Fatalf("the three-kind catalog offers %v, want the outline tools %v", sortedNames(got), sortedNames(want))
	}
}

// TestListSourcesShowsTheKiwiAndJiraSources checks that list_sources lists every source with its kind and tags.
func TestListSourcesShowsTheKiwiAndJiraSources(t *testing.T) {
	out, _, err := listSources(context.Background(), threeKindService(t, &fakeGateway{}), listSourcesIn{})
	if err != nil {
		t.Fatalf("list_sources: %v", err)
	}
	seen := map[string]string{}
	for _, row := range out.Sources {
		seen[row.ID] = row.Kind
	}
	if seen["kiwi"] != "kiwi" || seen["jira"] != "jira" || seen["outline"] != "outline" {
		t.Fatalf("list_sources kinds = %v, want outline, kiwi, and jira", seen)
	}
}

// TestOutlineToolRefusesAKiwiSource checks that an Outline tool given a Kiwi source is refused with the wrong-kind text,
// before any request.
func TestOutlineToolRefusesAKiwiSource(t *testing.T) {
	row, ok := operations.Lookup(domain.OpCollectionsList)
	if !ok {
		t.Fatal("collections.list is not in the catalog")
	}
	gw := &fakeGateway{}
	req := &sdk.CallToolRequest{Params: &sdk.CallToolParamsRaw{Arguments: json.RawMessage(`{"source":"kiwi"}`)}}
	_, _, _, public := runOperation(context.Background(), threeKindService(t, gw), row, req)
	if public == nil || public.Error() != "the source is of another kind" {
		t.Fatalf("public error = %v, want %q", public, "the source is of another kind")
	}
	if gw.calls != 0 {
		t.Fatalf("the gateway was called %d times, want none", gw.calls)
	}
}
