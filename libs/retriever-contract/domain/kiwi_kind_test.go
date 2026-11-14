package domain

import (
	"errors"
	"strings"
	"testing"
)

// oneSourceCatalog returns a catalog of one source with the given kind and ID.
func oneSourceCatalog(t *testing.T, kind SourceKind, id string) Catalog {
	t.Helper()
	sid, err := NewSourceID(id)
	if err != nil {
		t.Fatalf("NewSourceID(%q): %v", id, err)
	}
	ep, err := NewEndpoint("https://" + id + ".example.com")
	if err != nil {
		t.Fatalf("NewEndpoint: %v", err)
	}
	ref, err := NewCredentialRef("ONE_API_KEY")
	if err != nil {
		t.Fatalf("NewCredentialRef: %v", err)
	}
	cat, err := NewCatalog([]Source{NewSource(sid, kind, mustTagSet(t), ep, ref)})
	if err != nil {
		t.Fatalf("NewCatalog: %v", err)
	}
	return cat
}

// TestKindKiwiIsAKnownKind checks that the Kiwi kind is accepted by name and is not the Outline kind.
func TestKindKiwiIsAKnownKind(t *testing.T) {
	kind, err := NewSourceKind("kiwi")
	if err != nil || kind != KindKiwi || kind.String() != "kiwi" {
		t.Fatalf("NewSourceKind(kiwi) = %v, %v", kind, err)
	}
	if KindKiwi == KindOutline {
		t.Fatal("KindKiwi equals KindOutline")
	}
}

// TestUnknownKindTextNamesEveryKnownKind pins the text of the unknown-kind error and checks that it names every kind of
// the known list, so the text cannot drift from the list again.
func TestUnknownKindTextNamesEveryKnownKind(t *testing.T) {
	const want = "unknown source kind (known: outline, kiwi, jira)"
	if got := ErrUnknownSourceKind.Error(); got != want {
		t.Fatalf("ErrUnknownSourceKind = %q, want %q", got, want)
	}
	for _, kind := range knownKinds {
		if !strings.Contains(ErrUnknownSourceKind.Error(), kind.String()) {
			t.Fatalf("the unknown-kind text does not name the known kind %q", kind)
		}
	}
}

// TestNoSourceOfKindErrorNamesTheKind checks the typed error: its text names the kind, and it matches the sentinel.
func TestNoSourceOfKindErrorNamesTheKind(t *testing.T) {
	cases := []struct {
		kind SourceKind
		want string
	}{
		{KindOutline, "no outline source is configured"},
		{KindKiwi, "no kiwi source is configured"},
	}
	for _, tc := range cases {
		err := NoSourceOfKindError{Kind: tc.kind}
		if err.Error() != tc.want {
			t.Fatalf("NoSourceOfKindError(%s).Error() = %q, want %q", tc.kind, err.Error(), tc.want)
		}
		if !errors.Is(err, ErrNoSourceOfKind) {
			t.Fatalf("NoSourceOfKindError(%s) does not match ErrNoSourceOfKind", tc.kind)
		}
	}
	if errors.Is(NoSourceOfKindError{Kind: KindKiwi}, ErrUnknownSource) {
		t.Fatal("NoSourceOfKindError matches an unrelated sentinel")
	}
}

// TestSelectReturnsTheKindNamedNoSourceError checks that Catalog.Select names the kind it looked for when no source of
// that kind exists, while the other kind is configured.
func TestSelectReturnsTheKindNamedNoSourceError(t *testing.T) {
	cat := oneSourceCatalog(t, KindOutline, "outline")
	_, err := cat.Select(KindKiwi, nil)
	var none NoSourceOfKindError
	if !errors.As(err, &none) || none.Kind != KindKiwi {
		t.Fatalf("Select(kiwi) error = %v, want NoSourceOfKindError for kiwi", err)
	}
	if !errors.Is(err, ErrNoSourceOfKind) {
		t.Fatalf("Select(kiwi) error = %v, want a match for ErrNoSourceOfKind", err)
	}
}

// TestUnresolvedNameIsAPublicError checks the text and the public listing of the name-lookup error.
func TestUnresolvedNameIsAPublicError(t *testing.T) {
	const want = "a name matches no entry, or more than one, in the source"
	if ErrUnresolvedName.Error() != want {
		t.Fatalf("ErrUnresolvedName = %q, want %q", ErrUnresolvedName.Error(), want)
	}
	for _, e := range PublicErrors() {
		if e == ErrUnresolvedName {
			return
		}
	}
	t.Fatal("ErrUnresolvedName is not in PublicErrors")
}

// TestKiwiOperationIDsAreTheSixNames checks the six Kiwi operation IDs, which name the Kiwi rows of the catalog.
func TestKiwiOperationIDsAreTheSixNames(t *testing.T) {
	cases := map[OperationID]string{
		OpKiwiProductsList:    "kiwi.products.list",
		OpKiwiProductsTree:    "kiwi.products.tree",
		OpKiwiDocumentsGet:    "kiwi.documents.get",
		OpKiwiTestCasesCreate: "kiwi.test-cases.create",
		OpKiwiTestCasesUpdate: "kiwi.test-cases.update",
		OpKiwiPlansAddCase:    "kiwi.plans.add-case",
	}
	for id, want := range cases {
		if string(id) != want {
			t.Fatalf("operation ID = %q, want %q", id, want)
		}
	}
}
