package mcp

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/operations"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/sources"
)

// TestTheToolsComeFromTheComposedCatalog checks that the MCP tools are the rows of the service's operation catalog: a spec
// row that names a tool becomes a tool after the built-in tools, and a spec row with no tool does not.
func TestTheToolsComeFromTheComposedCatalog(t *testing.T) {
	kinds, err := domain.NewKindSet(
		domain.KindDefinition{Name: "outline", Display: "Outline"},
		domain.KindDefinition{Name: "kiwi", Display: "Kiwi"},
		domain.KindDefinition{Name: "jira", Display: "Jira", UsesIdentity: true},
		domain.KindDefinition{Name: "example-wiki", Display: "Example wiki"},
	)
	if err != nil {
		t.Fatalf("NewKindSet: %v", err)
	}
	specRows := []operations.Row{
		{ID: "example-wiki.docs.list", CLI: "example-wiki docs list", Tool: "example_wiki_docs_list", Effect: domain.EffectRead, ListKey: "items", Fields: []string{"id"}},
		{ID: "example-wiki.docs.delete", CLI: "example-wiki docs delete", Effect: domain.EffectDestructive},
	}
	cat, err := operations.NewCatalog(kinds, append(operations.Rows(), specRows...))
	if err != nil {
		t.Fatalf("NewCatalog: %v", err)
	}
	svc := sources.Service{Operations: &cat}

	var names []string
	for _, row := range operationTools(svc) {
		names = append(names, row.Tool)
	}
	last := names[len(names)-1]
	if last != "example_wiki_docs_list" {
		t.Fatalf("last tool = %q, want example_wiki_docs_list", last)
	}
	for _, name := range names {
		if name == "" {
			t.Fatalf("a tool with an empty name is listed: %v", names)
		}
	}
	if len(names) != len(operationTools(sources.Service{}))+1 {
		t.Fatalf("tools = %d, want the built-in tools and one spec tool", len(names))
	}
}

// TestToolsJSONRendersTheNamedToolsInOrder checks the canonical JSON of the
// conformance report: one object per row in order, with the name, and exactly
// one trailing newline.
func TestToolsJSONRendersTheNamedToolsInOrder(t *testing.T) {
	rows := []operations.Row{
		{ID: "example-wiki.docs.list", CLI: "example-wiki docs list", Tool: "example_wiki_docs_list", Summary: "List documents.", Effect: domain.EffectRead, ListKey: "items", Fields: []string{"id"}},
		{ID: "example-wiki.docs.get", CLI: "example-wiki docs get", Tool: "example_wiki_docs_get", Summary: "Get one document.", Effect: domain.EffectRead, Fields: []string{"id", "title"}},
	}
	text := string(ToolsJSON(rows))
	if !strings.HasSuffix(text, "]\n") || strings.HasSuffix(text, "\n\n") {
		t.Fatalf("ToolsJSON does not end with exactly one newline: %q", text)
	}
	first := strings.Index(text, "example_wiki_docs_list")
	second := strings.Index(text, "example_wiki_docs_get")
	if first < 0 || second < 0 || first > second {
		t.Fatalf("ToolsJSON = %q, want both tools in order", text)
	}
	var decoded []map[string]any
	if err := json.Unmarshal([]byte(text), &decoded); err != nil {
		t.Fatalf("ToolsJSON is not JSON: %v", err)
	}
	if len(decoded) != 2 {
		t.Fatalf("ToolsJSON holds %d tools, want 2", len(decoded))
	}
}
