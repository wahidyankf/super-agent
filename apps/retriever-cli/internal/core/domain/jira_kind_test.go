package domain

import (
	"errors"
	"testing"
)

// kindSource builds a source of any kind, with an identity name when identity is not empty. It is the test helper of the
// identity and ceiling rules of ADR 010 section 1.1.
func kindSource(t *testing.T, id string, kind SourceKind, host, cred, identity string, access Access) Source {
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
	src := NewSource(sid, kind, mustTagSet(t), access, ep, ref)
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
// "outline, kiwi, jira" (ADR 010 section 9).
func TestKindJiraIsAKnownKindAfterKiwi(t *testing.T) {
	got, err := NewSourceKind("jira")
	if err != nil || got != KindJira {
		t.Fatalf("NewSourceKind(jira) = %v, %v; want KindJira", got, err)
	}
	if knownKindNames() != "outline, kiwi, jira" {
		t.Fatalf("knownKindNames() = %q, want %q", knownKindNames(), "outline, kiwi, jira")
	}
	if KindJira.String() != "jira" {
		t.Fatalf("KindJira.String() = %q, want jira", KindJira.String())
	}
}

// TestKindTraitsOfEachKind pins the per-kind table of ADR 010 section 1.1: which kind needs an identity name, and the
// highest access the catalog accepts for it. A kind with no traits uses no identity and allows nothing.
func TestKindTraitsOfEachKind(t *testing.T) {
	cases := []struct {
		kind     SourceKind
		identity bool
		max      Access
	}{
		{KindOutline, false, AccessAdmin},
		{KindKiwi, false, AccessAdmin},
		{KindJira, true, AccessRead},
		// A kind with no traits (a test-only kind, or a zero value that zeroFault refuses) has no ceiling below admin.
		{SourceKind{}, false, AccessAdmin},
	}
	for _, tc := range cases {
		if got := tc.kind.UsesIdentity(); got != tc.identity {
			t.Errorf("%q.UsesIdentity() = %v, want %v", tc.kind, got, tc.identity)
		}
		if got := tc.kind.MaxAccess(); got != tc.max {
			t.Errorf("%q.MaxAccess() = %q, want %q", tc.kind, got, tc.max)
		}
	}
}

// TestSourceKindDisplayNamesJira checks the display name of the Jira kind, which the failure texts use (ADR 010 section 7).
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
	plain := kindSource(t, "jira", KindJira, "jira.example.test", "EXAMPLE_JIRA_API_TOKEN", "", AccessRead)
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

// TestNewCatalogIdentityAndCeilingRules checks each kind rule of ADR 010 section 1.1 for a source at position 2, with
// a valid first source. The text names the position, the field, and the rule, and never a value.
func TestNewCatalogIdentityAndCeilingRules(t *testing.T) {
	good := kindSource(t, "outline", KindOutline, "docs.example.com", "GOOD_API_KEY", "", AccessRead)
	cases := []struct {
		name  string
		src   Source
		field string
		want  error
		text  string
	}{
		{
			name:  "a jira source with no identity",
			src:   kindSource(t, "jira", KindJira, "jira.example.test", "JIRA_API_KEY", "", AccessRead),
			field: "identity_env", want: ErrIdentityRequired,
			text: "source 2 identity_env: required for a jira source",
		},
		{
			name:  "an outline source with an identity",
			src:   kindSource(t, "wiki", KindOutline, "wiki.example.com", "WIKI_API_KEY", "WIKI_EMAIL", AccessRead),
			field: "identity_env", want: ErrIdentityNotUsed,
			text: "source 2 identity_env: not used by an outline source",
		},
		{
			name:  "a kiwi source with an identity",
			src:   kindSource(t, "kiwi", KindKiwi, "kiwi.example.com", "KIWI_API_KEY", "KIWI_EMAIL", AccessRead),
			field: "identity_env", want: ErrIdentityNotUsed,
			text: "source 2 identity_env: not used by a kiwi source",
		},
		{
			name:  "a jira source with write access",
			src:   kindSource(t, "jira", KindJira, "jira.example.test", "JIRA_API_KEY", "JIRA_EMAIL", AccessWrite),
			field: "access", want: ErrAccessAboveKindCeiling,
			text: "source 2 access: a jira source allows read at most",
		},
		{
			name:  "a jira source with admin access",
			src:   kindSource(t, "jira", KindJira, "jira.example.test", "JIRA_API_KEY", "JIRA_EMAIL", AccessAdmin),
			field: "access", want: ErrAccessAboveKindCeiling,
			text: "source 2 access: a jira source allows read at most",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewCatalog([]Source{good, tc.src})
			var fault SourceFault
			if !errors.As(err, &fault) {
				t.Fatalf("NewCatalog error = %v, want a SourceFault", err)
			}
			if fault.Position != 2 || fault.Field != tc.field || !errors.Is(fault, tc.want) {
				t.Fatalf("fault = %+v, want position 2, field %q, and %v", fault, tc.field, tc.want)
			}
			if fault.Error() != tc.text {
				t.Fatalf("fault text = %q, want %q", fault.Error(), tc.text)
			}
		})
	}
}

// TestNewCatalogAcceptsTheValidKindShapes checks the valid combinations: a jira source with read and an identity, and
// the outline and kiwi kinds at admin with no identity.
func TestNewCatalogAcceptsTheValidKindShapes(t *testing.T) {
	list := []Source{
		kindSource(t, "outline", KindOutline, "docs.example.com", "OUTLINE_API_KEY", "", AccessAdmin),
		kindSource(t, "kiwi", KindKiwi, "kiwi.example.com", "KIWI_API_KEY", "", AccessAdmin),
		kindSource(t, "jira", KindJira, "jira.example.test", "JIRA_API_KEY", "JIRA_EMAIL", AccessRead),
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
	first := kindSource(t, "jira-a", KindJira, "one.example.com", "JIRA_A_API_KEY", "SHARED_EMAIL", AccessRead)

	other := kindSource(t, "jira-b", KindJira, "two.example.com", "JIRA_B_API_KEY", "SHARED_EMAIL", AccessRead)
	_, err := NewCatalog([]Source{first, other})
	var fault SourceFault
	if !errors.As(err, &fault) || fault.Position != 2 || fault.Field != "identity_env" || !errors.Is(err, ErrIdentityHostMismatch) {
		t.Fatalf("NewCatalog error = %v, want an identity_env host mismatch at position 2", err)
	}
	if fault.Error() != "source 2 identity_env: one identity serves one host only" {
		t.Fatalf("fault text = %q", fault.Error())
	}

	same := kindSource(t, "jira-c", KindJira, "one.example.com", "JIRA_C_API_KEY", "SHARED_EMAIL", AccessRead)
	if _, err := NewCatalog([]Source{first, same}); err != nil {
		t.Fatalf("NewCatalog with one identity on one host error = %v, want nil", err)
	}
}

// TestKindRuleErrorsKeepTheirSentinels checks that each typed kind rule matches its sentinel with errors.Is.
func TestKindRuleErrorsKeepTheirSentinels(t *testing.T) {
	if !errors.Is(IdentityRequiredError{Kind: KindJira}, ErrIdentityRequired) {
		t.Error("IdentityRequiredError does not match ErrIdentityRequired")
	}
	if !errors.Is(IdentityNotUsedError{Kind: KindOutline}, ErrIdentityNotUsed) {
		t.Error("IdentityNotUsedError does not match ErrIdentityNotUsed")
	}
	if !errors.Is(KindCeilingError{Kind: KindJira, Max: AccessRead}, ErrAccessAboveKindCeiling) {
		t.Error("KindCeilingError does not match ErrAccessAboveKindCeiling")
	}
	if errors.Is(IdentityRequiredError{Kind: KindJira}, ErrIdentityNotUsed) {
		t.Error("IdentityRequiredError matches the wrong sentinel")
	}
}

// TestAccessAboveKindCeilingIsPublic checks that the ceiling sentinel is in the public list, so the presenters show its
// fixed text.
func TestAccessAboveKindCeilingIsPublic(t *testing.T) {
	for _, e := range PublicErrors() {
		if e == ErrAccessAboveKindCeiling {
			return
		}
	}
	t.Fatal("PublicErrors does not list ErrAccessAboveKindCeiling")
}
