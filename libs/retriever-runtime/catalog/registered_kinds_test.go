package catalog

import (
	"errors"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// wikiKinds registers the example wiki kind beside the three built-in kinds, as a composition root does.
func wikiKinds(t *testing.T) domain.KindSet {
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

// loadWithKinds loads the catalog that content describes, through fake file calls, with the registered kinds given.
func loadWithKinds(content string, kinds domain.KindSet) (domain.Catalog, error) {
	l := &Loader{path: "/work/project/sources.yml", files: contentFiles(content), kinds: kinds}
	return l.Load()
}

// TestCatalogLoadsASourceOfARegisteredKind checks the acceptance rule of a registered kind: a catalog that names it
// loads, and the source carries the kind with its display name.
func TestCatalogLoadsASourceOfARegisteredKind(t *testing.T) {
	cat, err := loadWithKinds(withField("kind", "example-wiki"), wikiKinds(t))
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	all := cat.All()
	if len(all) != 1 || all[0].Kind().String() != "example-wiki" || all[0].Kind().Display() != "Example Wiki" {
		t.Fatalf("catalog = %v, want one example-wiki source", all)
	}
}

// TestNewWithFilesLoadsWithTheRegisteredKinds checks the constructor that the composition root calls: a loader over an
// absolute path keeps the kind set it was given, so a catalog loaded through it accepts the registered kind.
func TestNewWithFilesLoadsWithTheRegisteredKinds(t *testing.T) {
	l, err := NewWithFiles("/work/project/sources.yml", contentFiles(withField("kind", "example-wiki")), wikiKinds(t))
	if err != nil {
		t.Fatalf("NewWithFiles error = %v, want nil", err)
	}
	cat, err := l.Load()
	if err != nil || len(cat.All()) != 1 {
		t.Fatalf("Load() = %v, %v; want one source of the registered kind", cat.All(), err)
	}
}

// TestCatalogLoadsRegisteredAndBuiltInKindsTogether checks that a catalog may mix a registered kind with the built-in ones.
func TestCatalogLoadsRegisteredAndBuiltInKindsTogether(t *testing.T) {
	content := oneSource + `  - id: wiki
    kind: example-wiki
    endpoint: https://wiki.example.test
    credential_env: EXAMPLE_WIKI_API_KEY
`
	cat, err := loadWithKinds(content, wikiKinds(t))
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if got := len(cat.All()); got != 2 {
		t.Fatalf("catalog holds %d sources, want 2", got)
	}
}

// TestCatalogRefusesAKindNoRegistrationNames checks the other half of the rule: a kind the set does not register fails,
// and the message lists the registered kinds in alphabetical order.
func TestCatalogRefusesAKindNoRegistrationNames(t *testing.T) {
	_, err := loadWithKinds(withField("kind", "notion"), wikiKinds(t))
	want := catalogPrefix + "source 1 kind: unknown kind (known: example-wiki, jira, kiwi, outline)"
	if err == nil || err.Error() != want {
		t.Fatalf("Load() error = %v, want %q", err, want)
	}
}

// TestBuiltInSetRefusesARegisteredKindThatWasNotRegistered checks that the kind of another composition root stays unknown
// to the built-in set: the built-in set lists only its three kinds.
func TestBuiltInSetRefusesARegisteredKindThatWasNotRegistered(t *testing.T) {
	_, err := loadWithKinds(withField("kind", "example-wiki"), domain.BuiltinKinds())
	want := catalogPrefix + "source 1 kind: unknown kind (known: jira, kiwi, outline)"
	if err == nil || err.Error() != want {
		t.Fatalf("Load() error = %v, want %q", err, want)
	}
}

// TestUnknownKindFailureKeepsItsStableCause checks that the failure still matches the sentinel and names the kind rule
// through the typed cause, so the stable code retriever.<area>.unknown_source_kind is unchanged.
func TestUnknownKindFailureKeepsItsStableCause(t *testing.T) {
	_, err := loadWithKinds(withField("kind", "notion"), wikiKinds(t))
	if !errors.Is(err, domain.ErrUnknownSourceKind) {
		t.Fatalf("error = %v, want it to match ErrUnknownSourceKind", err)
	}
	var catalogErr domain.CatalogError
	if !errors.As(err, &catalogErr) {
		t.Fatalf("error = %T, want domain.CatalogError", err)
	}
	var unknown domain.UnknownKindError
	if !errors.As(err, &unknown) || len(unknown.Known) != 4 {
		t.Fatalf("typed cause = %v, want the four registered kinds", unknown.Known)
	}
}

// TestUnknownKindNeverEchoesTheValue checks that a refused kind name is not printed: a catalog may hold a value pasted by
// mistake.
func TestUnknownKindNeverEchoesTheValue(t *testing.T) {
	_, err := loadWithKinds(withField("kind", sentinel), wikiKinds(t))
	if err == nil {
		t.Fatal("Load() error = nil, want an unknown kind")
	}
	if got := err.Error(); strings.Contains(got, sentinel) {
		t.Fatalf("Load() error echoes the refused value: %q", got)
	}
}
