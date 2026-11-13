package mcp

import (
	"slices"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/sources"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/sources/operations"
)

// readOnlyService returns a service whose one source is at read, so the registration ceiling is read.
func readOnlyService(t *testing.T) sources.Service {
	t.Helper()
	return serviceWith(t, &fakeGateway{}, mustSource(t, "outline", "read"))
}

// adminService returns a service whose one source is at admin, capped by the given ceiling.
func adminService(t *testing.T, ceiling domain.Access) sources.Service {
	t.Helper()
	return serviceWith(t, &fakeGateway{}, mustSource(t, "outline", "admin")).Capped(ceiling)
}

// sortedNames returns the names sorted, for an order-free comparison.
func sortedNames(names []string) []string {
	out := slices.Clone(names)
	slices.Sort(out)
	return out
}

// defaultToolNames is the tool list at the read ceiling with no resource list: list_sources and the 55 generated read
// tools, sorted. mcp-server.feature pins the same names at every test level, so a change here is a change to the surface.
var defaultToolNames = []string{
	"collection_info", "collection_tree", "document_children", "get_access_request",
	"get_comment", "get_document", "get_document_insights", "get_emoji",
	"get_file_operation", "get_group", "get_installation", "get_pin",
	"get_relationship", "get_revision", "get_share", "get_subscription",
	"get_template", "get_user", "list_archived_documents", "list_attachments",
	"list_collection_group_memberships", "list_collection_memberships", "list_collections", "list_comments",
	"list_deleted_documents", "list_document_group_memberships", "list_document_memberships", "list_document_users",
	"list_documents", "list_draft_documents", "list_emojis", "list_events",
	"list_file_operations", "list_group_members", "list_group_memberships", "list_groups",
	"list_notifications", "list_oauth_authentications", "list_pins", "list_reactions",
	"list_relationships", "list_revisions", "list_searches", "list_shares",
	"list_sources", "list_stars", "list_subscriptions", "list_templates",
	"list_user_memberships", "list_users", "list_viewed_documents", "list_views",
	"search_document_titles", "search_documents", "suggest_mentions", "whoami",
}

// TestDefaultToolListAtReadIsUnchanged checks that with no resource list the server offers exactly the default tools:
// 56 names, the same ones as before --tools existed.
func TestDefaultToolListAtReadIsUnchanged(t *testing.T) {
	tools, err := RegisteredTools(readOnlyService(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := sortedNames(tools); !slices.Equal(got, defaultToolNames) {
		t.Fatalf("default tool list = %v, want %v", got, defaultToolNames)
	}
}

// TestResourceListOffersOnlyTheNamedResourcesAndListSources checks the narrowing: a named resource keeps its tools,
// every other resource's tools are left out, and list_sources stays.
func TestResourceListOffersOnlyTheNamedResourcesAndListSources(t *testing.T) {
	tools, err := RegisteredTools(readOnlyService(t), []string{"comments"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"get_comment", "list_comments", "list_sources"}
	if got := sortedNames(tools); !slices.Equal(got, want) {
		t.Fatalf("tools for comments = %v, want %v", got, want)
	}
}

// TestResourceListHoldsEveryToolOnlyOnceAcrossResources checks the partition: the tools of each resource, with
// list_sources, are the default list, and no resource offers a tool of another resource.
func TestResourceListHoldsEveryToolOnlyOnceAcrossResources(t *testing.T) {
	svc := readOnlyService(t)
	all, err := RegisteredTools(svc, nil)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]int{}
	for _, resource := range operations.Resources() {
		tools, err := RegisteredTools(svc, []string{resource})
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range tools {
			if name == "list_sources" {
				continue
			}
			seen[name]++
			if !slices.ContainsFunc(operations.Rows(), func(r operations.Row) bool {
				return r.Tool == name && r.Resource() == resource
			}) {
				t.Fatalf("resource %s offers the tool %s, which belongs to another resource", resource, name)
			}
		}
	}
	for _, name := range all {
		if name != "list_sources" && seen[name] != 1 {
			t.Fatalf("tool %s is offered by %d resources, want exactly 1", name, seen[name])
		}
	}
	if len(seen) != len(all)-1 {
		t.Fatalf("the resources offer %d generated tools, want %d", len(seen), len(all)-1)
	}
}

// TestResourceListNeverOffersAToolTheCeilingRefuses checks that a named resource does not widen the ceiling: an admin
// tool of a named resource is offered only at the admin ceiling.
func TestResourceListNeverOffersAToolTheCeilingRefuses(t *testing.T) {
	cases := []struct {
		name    string
		ceiling domain.Access
		want    []string
	}{
		{name: "read ceiling", ceiling: domain.AccessRead, want: []string{"list_sources"}},
		{name: "admin ceiling", ceiling: domain.AccessAdmin, want: []string{"list_api_keys", "list_sources"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tools, err := RegisteredTools(adminService(t, tc.ceiling), []string{"api-keys"})
			if err != nil {
				t.Fatal(err)
			}
			if got := sortedNames(tools); !slices.Equal(got, tc.want) {
				t.Fatalf("tools for api-keys = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestResourceWithNoToolAtTheCeilingAddsNothing checks that a resource whose tools the ceiling refuses is accepted and
// offers only list_sources.
func TestResourceWithNoToolAtTheCeilingAddsNothing(t *testing.T) {
	tools, err := RegisteredTools(readOnlyService(t), []string{"webhook-subscriptions"})
	if err != nil {
		t.Fatal(err)
	}
	if got := sortedNames(tools); !slices.Equal(got, []string{"list_sources"}) {
		t.Fatalf("tools for webhook-subscriptions at read = %v, want list_sources only", got)
	}
}

// TestNewServerOffersOnlyTheNamedResources checks the same narrowing through the production server and an SDK client,
// so the option reaches the tools a client lists and not only the table the test calls.
func TestNewServerOffersOnlyTheNamedResources(t *testing.T) {
	server, err := NewServer(readOnlyService(t), Options{Version: "test", Resources: []string{"documents"}})
	if err != nil {
		t.Fatal(err)
	}
	got := sortedNames(listedNames(t, server))
	if len(got) != 14 || !slices.Contains(got, "get_document") || slices.Contains(got, "list_comments") {
		t.Fatalf("listed tools = %v, want the 13 document tools and list_sources", got)
	}
}
