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

// kindSource returns one source of the given kind at the given access, with the endpoint and key name of the catalog.
func kindSource(t *testing.T, kind, id, host, credential, access string) domain.Source {
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
	acc, err := domain.NewAccess(access)
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
	return domain.NewSource(sid, k, tags, acc, endpoint, ref)
}

// threeKindService returns a catalog with an Outline source at admin, a Kiwi source at write, and a Jira source at read
// with its identity name. Every gateway answers from gw.
func threeKindService(t *testing.T, gw *fakeGateway) sources.Service {
	t.Helper()
	identity, err := domain.NewIdentityRef("EXAMPLE_JIRA_EMAIL")
	if err != nil {
		t.Fatal(err)
	}
	jira := kindSource(t, "jira", "jira", "jira.example.test", "EXAMPLE_JIRA_API_TOKEN", "read").WithIdentity(identity)
	return serviceWith(t, gw,
		mustSource(t, "outline", "admin"),
		kindSource(t, "kiwi", "kiwi", "kiwi.example.test", "EXAMPLE_KIWI_API_KEY", "write"),
		jira,
	)
}

// TestKindsOtherThanOutlineAddNoToolAtAnyCeiling checks that a catalog with Kiwi and Jira sources offers exactly the
// tools of an Outline-only catalog, at every ceiling.
func TestKindsOtherThanOutlineAddNoToolAtAnyCeiling(t *testing.T) {
	for _, ceiling := range []domain.Access{domain.AccessRead, domain.AccessWrite, domain.AccessAdmin} {
		outlineOnly := serviceWith(t, &fakeGateway{}, mustSource(t, "outline", "admin")).Capped(ceiling)
		all := threeKindService(t, &fakeGateway{}).Capped(ceiling)
		want, err := RegisteredTools(outlineOnly, nil)
		if err != nil {
			t.Fatalf("RegisteredTools(outline only, %s): %v", ceiling, err)
		}
		got, err := RegisteredTools(all, nil)
		if err != nil {
			t.Fatalf("RegisteredTools(three kinds, %s): %v", ceiling, err)
		}
		if !slices.Equal(sortedNames(got), sortedNames(want)) {
			t.Fatalf("at %s the three-kind catalog offers %v, want the outline tools %v", ceiling, sortedNames(got), sortedNames(want))
		}
	}
}

// TestAKiwiWriteSourceDoesNotRaiseTheToolCeiling checks the case the ceiling must get right: an Outline source at read
// beside a Kiwi source at write. At every ceiling the server offers exactly the tools of the Outline source alone, so the
// Kiwi write adds no Outline write tool.
func TestAKiwiWriteSourceDoesNotRaiseTheToolCeiling(t *testing.T) {
	for _, ceiling := range []domain.Access{domain.AccessRead, domain.AccessWrite, domain.AccessAdmin} {
		outlineOnly := serviceWith(t, &fakeGateway{}, mustSource(t, "outline", "read")).Capped(ceiling)
		withKiwi := serviceWith(t, &fakeGateway{},
			mustSource(t, "outline", "read"),
			kindSource(t, "kiwi", "kiwi", "kiwi.example.test", "EXAMPLE_KIWI_API_KEY", "write"),
		).Capped(ceiling)
		want, err := RegisteredTools(outlineOnly, nil)
		if err != nil {
			t.Fatalf("RegisteredTools(outline only, %s): %v", ceiling, err)
		}
		got, err := RegisteredTools(withKiwi, nil)
		if err != nil {
			t.Fatalf("RegisteredTools(outline and kiwi, %s): %v", ceiling, err)
		}
		if !slices.Equal(sortedNames(got), sortedNames(want)) {
			t.Fatalf("at %s the outline and kiwi catalog offers %d tools, want the %d outline tools", ceiling, len(got), len(want))
		}
	}
}

// TestListSourcesShowsTheKiwiAndJiraSources checks that list_sources lists every source with its kind, access, and tags.
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
