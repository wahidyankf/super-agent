package domain

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

// wikiDefinition is an example kind that a composition root registers. It needs no identity name.
var wikiDefinition = KindDefinition{Name: "example-wiki", Display: "Example Wiki"}

// registeredWithBuiltins returns a set that holds the three built-in kinds and the example wiki kind.
func registeredWithBuiltins(t *testing.T) KindSet {
	t.Helper()
	set, err := NewKindSet(
		KindDefinition{Name: "outline", Display: "Outline"},
		KindDefinition{Name: "kiwi", Display: "Kiwi"},
		KindDefinition{Name: "jira", Display: "Jira", UsesIdentity: true},
		wikiDefinition,
	)
	if err != nil {
		t.Fatalf("NewKindSet: %v", err)
	}
	return set
}

// TestBuiltinKindsHoldTheThreeKindsInAlphabeticalOrder checks the built-in set. Names are sorted, so a message that
// lists them is stable.
func TestBuiltinKindsHoldTheThreeKindsInAlphabeticalOrder(t *testing.T) {
	got := BuiltinKinds().Names()
	want := []string{"jira", "kiwi", "outline"}
	if !slices.Equal(got, want) {
		t.Fatalf("BuiltinKinds().Names() = %v, want %v", got, want)
	}
}

// TestBuiltinKindsResolveToTheBuiltInConstants checks that the set returns the same values as the exported constants,
// so the code that compares a kind with KindOutline still works.
func TestBuiltinKindsResolveToTheBuiltInConstants(t *testing.T) {
	set := BuiltinKinds()
	for name, want := range map[string]SourceKind{"outline": KindOutline, "kiwi": KindKiwi, "jira": KindJira} {
		got, err := set.Kind(name)
		if err != nil || got != want {
			t.Fatalf("Kind(%q) = %v, %v; want the built-in constant", name, got, err)
		}
	}
}

// TestOnlyJiraUsesIdentityInTheBuiltInSet checks the identity trait of each built-in kind, which the catalog rules read.
func TestOnlyJiraUsesIdentityInTheBuiltInSet(t *testing.T) {
	if !KindJira.UsesIdentity() {
		t.Fatal("KindJira.UsesIdentity() = false, want true")
	}
	if KindOutline.UsesIdentity() || KindKiwi.UsesIdentity() {
		t.Fatal("Outline and Kiwi must not use an identity name")
	}
}

// TestNewKindSetRegistersADefinitionWithItsDisplayName checks that a registered kind carries its name, display name,
// and identity trait, and that the set returns it.
func TestNewKindSetRegistersADefinitionWithItsDisplayName(t *testing.T) {
	set, err := NewKindSet(wikiDefinition)
	if err != nil {
		t.Fatalf("NewKindSet: %v", err)
	}
	kind, err := set.Kind("example-wiki")
	if err != nil {
		t.Fatalf("Kind(example-wiki) error = %v, want nil", err)
	}
	if kind.String() != "example-wiki" || kind.Display() != "Example Wiki" || kind.UsesIdentity() {
		t.Fatalf("kind = (%q, %q, %v), want (example-wiki, Example Wiki, false)", kind.String(), kind.Display(), kind.UsesIdentity())
	}
}

// TestNewKindSetDefaultsTheDisplayNameToTheName checks the fallback: a definition with no display name shows its name.
func TestNewKindSetDefaultsTheDisplayNameToTheName(t *testing.T) {
	set, err := NewKindSet(KindDefinition{Name: "example-tracker"})
	if err != nil {
		t.Fatalf("NewKindSet: %v", err)
	}
	kind, err := set.Kind("example-tracker")
	if err != nil || kind.Display() != "example-tracker" {
		t.Fatalf("Display() = %q, %v; want example-tracker", kind.Display(), err)
	}
}

// TestNewKindSetKeepsTheIdentityTrait checks that a definition that needs an identity name keeps that trait.
func TestNewKindSetKeepsTheIdentityTrait(t *testing.T) {
	set, err := NewKindSet(KindDefinition{Name: "example-mail", UsesIdentity: true})
	if err != nil {
		t.Fatalf("NewKindSet: %v", err)
	}
	kind, err := set.Kind("example-mail")
	if err != nil || !kind.UsesIdentity() {
		t.Fatalf("UsesIdentity() = %v, %v; want true", kind.UsesIdentity(), err)
	}
}

// TestNewKindSetNamesAreAlphabetical checks that the names come back sorted, whatever the registration order.
func TestNewKindSetNamesAreAlphabetical(t *testing.T) {
	set, err := NewKindSet(KindDefinition{Name: "zeta"}, KindDefinition{Name: "example-wiki"}, KindDefinition{Name: "alpha"})
	if err != nil {
		t.Fatalf("NewKindSet: %v", err)
	}
	if got, want := set.Names(), []string{"alpha", "example-wiki", "zeta"}; !slices.Equal(got, want) {
		t.Fatalf("Names() = %v, want %v", got, want)
	}
}

// TestNewKindSetRefusesNoDefinition checks that a set needs at least one kind.
func TestNewKindSetRefusesNoDefinition(t *testing.T) {
	if _, err := NewKindSet(); !errors.Is(err, ErrNoKinds) {
		t.Fatalf("NewKindSet() error = %v, want ErrNoKinds", err)
	}
}

// TestNewKindSetRefusesAnInvalidName checks the slug grammar of a kind name. A name is a command word and the area of an
// error code, so it follows the same rule as a source ID, up to 32 characters.
func TestNewKindSetRefusesAnInvalidName(t *testing.T) {
	bad := []string{"", "Example", "1wiki", "example_wiki", "example-", "-example", "example--wiki", "example wiki", "exämple", strings.Repeat("a", 33)}
	for _, name := range bad {
		if _, err := NewKindSet(KindDefinition{Name: name}); !errors.Is(err, ErrInvalidKindName) {
			t.Fatalf("NewKindSet(%q) error = %v, want ErrInvalidKindName", name, err)
		}
	}
}

// TestNewKindSetAcceptsANameOfTheLongestLength checks the boundary of the length rule: 32 characters is accepted.
func TestNewKindSetAcceptsANameOfTheLongestLength(t *testing.T) {
	if _, err := NewKindSet(KindDefinition{Name: strings.Repeat("a", 32)}); err != nil {
		t.Fatalf("NewKindSet(32 characters) error = %v, want nil", err)
	}
}

// TestNewKindSetRefusesADuplicateName checks that two definitions of one name are refused, so a kind has one meaning.
func TestNewKindSetRefusesADuplicateName(t *testing.T) {
	_, err := NewKindSet(wikiDefinition, KindDefinition{Name: "example-wiki"})
	if !errors.Is(err, ErrDuplicateKind) {
		t.Fatalf("NewKindSet(duplicate) error = %v, want ErrDuplicateKind", err)
	}
}

// TestUnknownKindListsTheRegisteredKindsAlphabetically checks the message of an unregistered kind: it names every
// registered kind in alphabetical order, matches ErrUnknownSourceKind, and never repeats the text it was given.
func TestUnknownKindListsTheRegisteredKindsAlphabetically(t *testing.T) {
	_, err := registeredWithBuiltins(t).Kind("notion")
	if err == nil {
		t.Fatal("Kind(notion) error = nil, want an unknown kind")
	}
	if !errors.Is(err, ErrUnknownSourceKind) {
		t.Fatalf("error = %v, want it to match ErrUnknownSourceKind", err)
	}
	var unknown UnknownKindError
	if !errors.As(err, &unknown) || !slices.Equal(unknown.Known, []string{"example-wiki", "jira", "kiwi", "outline"}) {
		t.Fatalf("Known = %v, want the four registered kinds alphabetically", unknown.Known)
	}
	if got, want := err.Error(), "unknown source kind (known: example-wiki, jira, kiwi, outline)"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
	if strings.Contains(err.Error(), "notion") {
		t.Fatalf("Error() = %q repeats the refused kind", err.Error())
	}
}

// TestNewSourceKindKeepsTheFlagTextOfTheBuiltInKinds checks the flag path. The flags and GraphQL accept the built-in
// kinds only, and their unknown-kind text keeps the order it always had.
func TestNewSourceKindKeepsTheFlagTextOfTheBuiltInKinds(t *testing.T) {
	_, err := NewSourceKind("bogus")
	if err != ErrUnknownSourceKind {
		t.Fatalf("NewSourceKind(bogus) error = %v, want ErrUnknownSourceKind", err)
	}
	if got, want := ErrUnknownSourceKind.Error(), "unknown source kind (known: outline, kiwi, jira)"; got != want {
		t.Fatalf("ErrUnknownSourceKind = %q, want %q", got, want)
	}
	if _, err := NewSourceKind("example-wiki"); err == nil {
		t.Fatal("NewSourceKind(example-wiki) accepted a kind that only a registration names")
	}
}

// TestZeroSourceKindIsNoKind checks the zero value: no name, no display name, and no identity trait.
func TestZeroSourceKindIsNoKind(t *testing.T) {
	var kind SourceKind
	if kind.String() != "" || kind.Display() != "" || kind.UsesIdentity() {
		t.Fatalf("zero kind = (%q, %q, %v), want no kind", kind.String(), kind.Display(), kind.UsesIdentity())
	}
}

// TestZeroKindSetRegistersNothing checks the zero value of a set: every name is unknown, and it lists no kind.
func TestZeroKindSetRegistersNothing(t *testing.T) {
	var set KindSet
	if _, err := set.Kind("outline"); !errors.Is(err, ErrUnknownSourceKind) {
		t.Fatalf("zero set Kind(outline) error = %v, want ErrUnknownSourceKind", err)
	}
	if names := set.Names(); len(names) != 0 {
		t.Fatalf("zero set Names() = %v, want none", names)
	}
}

// TestKindSetNamesReturnsACopy checks that a caller who changes the returned names does not change the set.
func TestKindSetNamesReturnsACopy(t *testing.T) {
	set := BuiltinKinds()
	names := set.Names()
	names[0] = "changed"
	if got := set.Names()[0]; got != "jira" {
		t.Fatalf("Names()[0] after a change = %q, want jira", got)
	}
}

// TestRegisteredKindWithoutIdentityRefusesAnIdentityName checks that a registered kind follows the identity rule of
// its trait: a kind that uses no identity name refuses one.
func TestRegisteredKindWithoutIdentityRefusesAnIdentityName(t *testing.T) {
	kind := mustRegisteredKind(t, wikiDefinition)
	src := kindSource(t, "example-wiki", kind, "wiki.example.test", "EXAMPLE_WIKI_API_KEY", "EXAMPLE_WIKI_EMAIL")
	_, err := NewCatalog([]Source{src})
	if !errors.Is(err, ErrIdentityNotUsed) {
		t.Fatalf("NewCatalog error = %v, want ErrIdentityNotUsed", err)
	}
}

// TestRegisteredKindWithIdentityNeedsOne checks that a registered kind with the identity trait needs an identity name.
func TestRegisteredKindWithIdentityNeedsOne(t *testing.T) {
	kind := mustRegisteredKind(t, KindDefinition{Name: "example-mail", UsesIdentity: true})
	src := kindSource(t, "example-mail", kind, "mail.example.test", "EXAMPLE_MAIL_API_KEY", "")
	_, err := NewCatalog([]Source{src})
	if !errors.Is(err, ErrIdentityRequired) {
		t.Fatalf("NewCatalog error = %v, want ErrIdentityRequired", err)
	}
}

// TestCatalogAcceptsARegisteredKind checks that a source of a registered kind joins a catalog with its kind intact.
func TestCatalogAcceptsARegisteredKind(t *testing.T) {
	kind := mustRegisteredKind(t, wikiDefinition)
	src := kindSource(t, "example-wiki", kind, "wiki.example.test", "EXAMPLE_WIKI_API_KEY", "")
	cat, err := NewCatalog([]Source{src})
	if err != nil {
		t.Fatalf("NewCatalog error = %v, want nil", err)
	}
	got, ok := cat.ByID(src.ID())
	if !ok || got.Kind() != kind {
		t.Fatalf("ByID kind = %v, %v; want the registered kind", got.Kind(), ok)
	}
}

// mustRegisteredKind registers one definition and returns the kind it names.
func mustRegisteredKind(t *testing.T, def KindDefinition) SourceKind {
	t.Helper()
	set, err := NewKindSet(def)
	if err != nil {
		t.Fatalf("NewKindSet(%q): %v", def.Name, err)
	}
	kind, err := set.Kind(def.Name)
	if err != nil {
		t.Fatalf("Kind(%q): %v", def.Name, err)
	}
	return kind
}
