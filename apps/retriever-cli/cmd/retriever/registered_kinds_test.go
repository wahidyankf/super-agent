package main

import (
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/catalog"
)

// hostWithCatalog returns the unit-test host whose catalog file holds content in an in-memory folder.
func hostWithCatalog(content string) host {
	files := fstest.MapFS{"sources.yml": &fstest.MapFile{Data: []byte(content)}}
	h := testHost()
	h.openCatalog = func(path string, kinds domain.KindSet) (*catalog.Loader, error) {
		return catalog.NewWithFiles(path, catalog.Files{FS: files, SameFile: sameName}, kinds)
	}
	return h
}

// wikiCatalogYAML is a catalog with one source of the example wiki kind, which only a registration names.
const wikiCatalogYAML = `version: 1
sources:
  - id: wiki
    kind: example-wiki
    endpoint: https://wiki.example.test
    credential_env: EXAMPLE_WIKI_API_KEY
`

// wikiRegistration registers the example wiki kind beside the built-in kinds, as a composition root that adds a kind would.
func wikiRegistration(t *testing.T) domain.KindSet {
	t.Helper()
	set, err := domain.NewKindSet(
		domain.KindDefinition{Name: "outline", Display: "Outline"},
		domain.KindDefinition{Name: "kiwi", Display: "Kiwi"},
		domain.KindDefinition{Name: "jira", Display: "Jira", UsesIdentity: true},
		domain.KindDefinition{Name: "example-wiki", Display: "Example Wiki"},
	)
	if err != nil {
		t.Fatalf("NewKindSet: %v", err)
	}
	return set
}

// TestRegisteredKindsAreTheBuiltInKinds checks what the composition root registers: exactly the three built-in kinds, so
// the binary's catalog accepts what it accepted before.
func TestRegisteredKindsAreTheBuiltInKinds(t *testing.T) {
	if got, want := registeredKinds().Names(), []string{"jira", "kiwi", "outline"}; !slices.Equal(got, want) {
		t.Fatalf("registeredKinds().Names() = %v, want %v", got, want)
	}
}

// TestWireSourcesLoadsACatalogOfARegisteredKind checks the wiring of a registered kind: a catalog that names it loads, and
// the source keeps its kind.
func TestWireSourcesLoadsACatalogOfARegisteredKind(t *testing.T) {
	w, err := wireSources(examplePaths, seamSet{}, hostWithCatalog(wikiCatalogYAML), wikiRegistration(t))
	if err != nil {
		t.Fatalf("wireSources error = %v, want nil", err)
	}
	all := w.service.Catalog.All()
	if len(all) != 1 || all[0].Kind().String() != "example-wiki" {
		t.Fatalf("catalog = %v, want one example-wiki source", all)
	}
}

// TestWireSourcesRefusesAKindTheRootDoesNotRegister checks the other half: the binary's own kinds refuse the wiki kind,
// and the message lists the kinds the root registers, alphabetically.
func TestWireSourcesRefusesAKindTheRootDoesNotRegister(t *testing.T) {
	_, err := wireSources(examplePaths, seamSet{}, hostWithCatalog(wikiCatalogYAML), registeredKinds())
	if err == nil {
		t.Fatal("wireSources accepted a kind the composition root does not register")
	}
	if !strings.Contains(err.Error(), "known: jira, kiwi, outline") {
		t.Fatalf("wireSources error = %q, want the registered kinds listed", err.Error())
	}
}
