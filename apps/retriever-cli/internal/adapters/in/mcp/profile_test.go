package mcp

import (
	"slices"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/operations"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/sources"
)

// outlineService returns a service whose one source is a plain outline source.
func outlineService(t *testing.T) sources.Service {
	t.Helper()
	return serviceWith(t, &fakeGateway{}, mustSource(t, "outline"))
}

// sortedNames returns the names sorted, for an order-free comparison.
func sortedNames(names []string) []string {
	out := slices.Clone(names)
	slices.Sort(out)
	return out
}

// TestDefaultToolListOffersEveryTool checks that with no resource list the server offers every tool the catalog and
// the hand-written table name, and nothing else. The server offers every tool of the table.
func TestDefaultToolListOffersEveryTool(t *testing.T) {
	want := []string{}
	for _, row := range operations.Rows() {
		if row.Tool != "" {
			want = append(want, row.Tool)
		}
	}
	for _, h := range handWrittenTools {
		want = append(want, h.name)
	}
	tools, err := RegisteredTools(outlineService(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := sortedNames(tools); !slices.Equal(got, sortedNames(want)) {
		t.Fatalf("default tool list has %d names, want the %d catalog and hand-written tools", len(got), len(want))
	}
}

// TestResourceListOffersOnlyTheNamedResourcesAndListSources checks the narrowing: a named resource keeps its tools,
// every other resource's tools are left out, and list_sources stays.
func TestResourceListOffersOnlyTheNamedResourcesAndListSources(t *testing.T) {
	tools, err := RegisteredTools(outlineService(t), []string{"comments"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"add_comment_reaction", "create_comment", "delete_comment", "get_comment", "list_comments", "list_sources",
		"remove_comment_reaction", "resolve_comment", "unresolve_comment", "update_comment",
	}
	if got := sortedNames(tools); !slices.Equal(got, want) {
		t.Fatalf("tools for comments = %v, want %v", got, want)
	}
}

// TestResourceListHoldsEveryToolOnlyOnceAcrossResources checks the partition: the tools of each resource, with
// list_sources, are the default list, and no resource offers a tool of another resource.
func TestResourceListHoldsEveryToolOnlyOnceAcrossResources(t *testing.T) {
	svc := outlineService(t)
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

// TestNewServerOffersOnlyTheNamedResources checks the same narrowing through the production server and an SDK client,
// so the option reaches the tools a client lists and not only the table the test calls.
func TestNewServerOffersOnlyTheNamedResources(t *testing.T) {
	server, err := NewServer(outlineService(t), Options{Version: "test", Resources: []string{"documents"}})
	if err != nil {
		t.Fatal(err)
	}
	got := sortedNames(listedNames(t, server))
	if !slices.Contains(got, "get_document") || slices.Contains(got, "list_comments") || !slices.Contains(got, "list_sources") {
		t.Fatalf("listed tools = %v, want the document tools and list_sources", got)
	}
}
