package domain

import (
	"errors"
	"testing"
)

// kindSource builds a source of any kind, with an identity name when identity is not empty. It is the test helper of the
// identity rules of the kind table.
func kindSource(t *testing.T, id string, kind SourceKind, host, cred, identity string) Source {
	t.Helper()
	sid, err := NewSourceID(id)
	if err != nil {
		t.Fatalf("NewSourceID(%q): %v", id, err)
	}
	ep, err := NewEndpoint("https://" + host)
	if err != nil {
		t.Fatalf("NewEndpoint(%q): %v", host, err)
	}
	ref, err := NewCredentialRef(cred)
	if err != nil {
		t.Fatalf("NewCredentialRef(%q): %v", cred, err)
	}
	src := NewSource(sid, kind, mustTagSet(t), ep, ref)
	if identity == "" {
		return src
	}
	iref, err := NewIdentityRef(identity)
	if err != nil {
		t.Fatalf("NewIdentityRef(%q): %v", identity, err)
	}
	return src.WithIdentity(iref)
}

// TestKindJiraIsAKnownKindAfterKiwi checks that jira is accepted and listed last, so the known-kinds text reads
// "outline, kiwi, jira".
func TestKindJiraIsAKnownKindAfterKiwi(t *testing.T) {
	got, err := NewSourceKind("jira")
	if err != nil || got != KindJira {
		t.Fatalf("NewSourceKind(jira) = %v, %v; want KindJira", got, err)
	}
	if ErrUnknownSourceKind.Error() != "unknown source kind (known: outline, kiwi, jira)" {
		t.Fatalf("ErrUnknownSourceKind = %q, want the built-in list in its order", ErrUnknownSourceKind.Error())
	}
	if KindJira.String() != "jira" {
		t.Fatalf("KindJira.String() = %q, want jira", KindJira.String())
	}
}

// TestSourceKindDisplayNamesJira checks the display name of the Jira kind, which the failure texts use.
func TestSourceKindDisplayNamesJira(t *testing.T) {
	if got := KindJira.Display(); got != "Jira" {
		t.Fatalf("KindJira.Display() = %q, want Jira", got)
	}
}

// TestSourceKindDisplayOfAKindWithoutADisplayNameIsItsName checks the fallback of Display: a kind with no display name shows
// its kind name, so a failure text is never empty.
func TestSourceKindDisplayOfAKindWithoutADisplayNameIsItsName(t *testing.T) {
	if got := otherKind.Display(); got != "other" {
		t.Fatalf("otherKind.Display() = %q, want other", got)
	}
}

// TestSourceIdentityIsZeroUntilSetAndWithIdentityCopies checks that NewSource leaves the identity empty, that
// WithIdentity sets it on a copy, and that the original source does not change.
func TestSourceIdentityIsZeroUntilSetAndWithIdentityCopies(t *testing.T) {
	plain := kindSource(t, "jira", KindJira, "jira.example.test", "EXAMPLE_JIRA_API_TOKEN", "")
	if plain.Identity() != (IdentityRef{}) {
		t.Fatalf("a new source has identity %q, want none", plain.Identity().Name())
	}
	iref, err := NewIdentityRef("EXAMPLE_JIRA_EMAIL")
	if err != nil {
		t.Fatalf("NewIdentityRef: %v", err)
	}
	with := plain.WithIdentity(iref)
	if with.Identity() != iref {
		t.Fatalf("WithIdentity identity = %q, want EXAMPLE_JIRA_EMAIL", with.Identity().Name())
	}
	if with.ID() != plain.ID() || with.Kind() != plain.Kind() || with.Credential() != plain.Credential() {
		t.Fatal("WithIdentity changed a field other than the identity")
	}
	if plain.Identity() != (IdentityRef{}) {
		t.Fatal("WithIdentity changed the original source")
	}
}

// TestNewCatalogAcceptsTheValidKindShapes checks the valid combinations: a jira source with an identity, and
// the outline and kiwi kinds with no identity.
func TestNewCatalogAcceptsTheValidKindShapes(t *testing.T) {
	list := []Source{
		kindSource(t, "outline", KindOutline, "docs.example.com", "OUTLINE_API_KEY", ""),
		kindSource(t, "kiwi", KindKiwi, "kiwi.example.com", "KIWI_API_KEY", ""),
		kindSource(t, "jira", KindJira, "jira.example.test", "JIRA_API_KEY", "JIRA_EMAIL"),
	}
	cat, err := NewCatalog(list)
	if err != nil {
		t.Fatalf("NewCatalog(valid kinds) error = %v", err)
	}
	if len(cat.All()) != 3 {
		t.Fatalf("catalog holds %d sources, want 3", len(cat.All()))
	}
}

// TestNewCatalogOneIdentityServesOneHostOnly checks the identity host rule: one identity name on two hosts is refused at
// the second source, and one identity name on one host is accepted.
func TestNewCatalogOneIdentityServesOneHostOnly(t *testing.T) {
	first := kindSource(t, "jira-a", KindJira, "one.example.com", "JIRA_A_API_KEY", "SHARED_EMAIL")

	other := kindSource(t, "jira-b", KindJira, "two.example.com", "JIRA_B_API_KEY", "SHARED_EMAIL")
	_, err := NewCatalog([]Source{first, other})
	var fault SourceFault
	if !errors.As(err, &fault) || fault.Position != 2 || fault.Field != "identity_env" || !errors.Is(err, ErrIdentityHostMismatch) {
		t.Fatalf("NewCatalog error = %v, want an identity_env host mismatch at position 2", err)
	}
	if fault.Error() != "source 2 identity_env: one identity serves one host only" {
		t.Fatalf("fault text = %q", fault.Error())
	}

	same := kindSource(t, "jira-c", KindJira, "one.example.com", "JIRA_C_API_KEY", "SHARED_EMAIL")
	if _, err := NewCatalog([]Source{first, same}); err != nil {
		t.Fatalf("NewCatalog with one identity on one host error = %v, want nil", err)
	}
}
