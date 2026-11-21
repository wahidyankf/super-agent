package unit_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// @scenario specs/libs/retriever-contract/behaviours/source-kinds.feature: A built-in kind name is accepted with its display name and identity rule
func TestSourceKindsAcceptBuiltInKindWithDisplayAndIdentityRule(t *testing.T) {
	cases := []struct{ kind, display, identity string }{
		{"outline", "Outline", "not used"},
		{"kiwi", "Kiwi", "not used"},
		{"jira", "Jira", "required"},
	}
	for _, tc := range cases {
		t.Run(tc.kind, func(t *testing.T) {
			kind, err := domain.NewSourceKind(tc.kind)
			noError(t, err)
			if got := kind.String(); got != tc.kind {
				t.Errorf("kind name = %q, want %q", got, tc.kind)
			}
			if got := kind.Display(); got != tc.display {
				t.Errorf("display name = %q, want %q", got, tc.display)
			}
			identity := "not used"
			if kind.UsesIdentity() {
				identity = "required"
			}
			if identity != tc.identity {
				t.Errorf("identity rule = %q, want %q", identity, tc.identity)
			}
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/source-kinds.feature: A name that is not a built-in kind is refused and the kinds are listed
func TestSourceKindsRefuseNameThatIsNotBuiltInAndListKinds(t *testing.T) {
	for _, raw := range []string{"github", "Kiwi", "kiwi-2", ""} {
		t.Run(raw, func(t *testing.T) {
			kind, err := domain.NewSourceKind(raw)
			wantText(t, err, "unknown source kind (known: outline, kiwi, jira)")
			if kind != (domain.SourceKind{}) {
				t.Errorf("returned kind %q, want none", kind.String())
			}
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/source-kinds.feature: A registered kind takes its name as its display name when it has none
func TestSourceKindsRegisteredKindTakesNameAsDisplayWhenNone(t *testing.T) {
	set, err := domain.NewKindSet(domain.KindDefinition{Name: "acme-wiki"})
	noError(t, err)
	kind, err := set.Kind("acme-wiki")
	noError(t, err)
	if got := kind.String(); got != "acme-wiki" {
		t.Errorf("kind name = %q, want %q", got, "acme-wiki")
	}
	if got := kind.Display(); got != "acme-wiki" {
		t.Errorf("display name = %q, want %q", got, "acme-wiki")
	}
}

// @scenario specs/libs/retriever-contract/behaviours/source-kinds.feature: A registered kind keeps the display name it was given
func TestSourceKindsRegisteredKindKeepsDisplayNameItWasGiven(t *testing.T) {
	set, err := domain.NewKindSet(domain.KindDefinition{Name: "acme-wiki", Display: "Acme Wiki"})
	noError(t, err)
	kind, err := set.Kind("acme-wiki")
	noError(t, err)
	if got := kind.Display(); got != "Acme Wiki" {
		t.Errorf("display name = %q, want %q", got, "Acme Wiki")
	}
}

// @scenario specs/libs/retriever-contract/behaviours/source-kinds.feature: A kind set refuses a kind name that breaks the slug rule or is used twice
func TestSourceKindsRefuseKindNameThatBreaksSlugRuleOrIsUsedTwice(t *testing.T) {
	const slugMessage = "invalid kind name: use a lowercase slug of at most 32 characters"
	cases := []struct {
		names   []string
		message string
	}{
		{[]string{"Acme"}, slugMessage},
		{[]string{"1wiki"}, slugMessage},
		{[]string{"acme_wiki"}, slugMessage},
		{[]string{"acme-wiki", "acme-wiki"}, "duplicate kind name"},
	}
	for _, tc := range cases {
		t.Run(strings.Join(tc.names, ", "), func(t *testing.T) {
			defs := make([]domain.KindDefinition, 0, len(tc.names))
			for _, name := range tc.names {
				defs = append(defs, domain.KindDefinition{Name: name})
			}
			set, err := domain.NewKindSet(defs...)
			wantText(t, err, tc.message)
			if len(set.Names()) != 0 {
				t.Errorf("returned a kind set, want none")
			}
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/source-kinds.feature: A kind set refuses an empty list of definitions
func TestSourceKindsRefuseEmptyListOfDefinitions(t *testing.T) {
	set, err := domain.NewKindSet()
	wantText(t, err, "a kind set needs at least one kind")
	if len(set.Names()) != 0 {
		t.Errorf("returned a kind set, want none")
	}
}

// @scenario specs/libs/retriever-contract/behaviours/source-kinds.feature: A kind name of 33 characters is refused
func TestSourceKindsRefuseKindNameOf33Characters(t *testing.T) {
	_, err := domain.NewKindSet(domain.KindDefinition{Name: strings.Repeat("a", 33)})
	wantText(t, err, "invalid kind name: use a lowercase slug of at most 32 characters")
}

// @scenario specs/libs/retriever-contract/behaviours/source-kinds.feature: A kind set finds a registered kind by name
func TestSourceKindsSetFindsRegisteredKindByName(t *testing.T) {
	set, err := domain.NewKindSet(domain.KindDefinition{Name: "kiwi"}, domain.KindDefinition{Name: "acme-wiki"})
	noError(t, err)
	kind, err := set.Kind("kiwi")
	noError(t, err)
	if got := kind.String(); got != "kiwi" {
		t.Errorf("kind name = %q, want %q", got, "kiwi")
	}
}

// @scenario specs/libs/retriever-contract/behaviours/source-kinds.feature: A kind set names the registered kinds when it does not know a name
func TestSourceKindsSetNamesRegisteredKindsWhenItDoesNotKnowName(t *testing.T) {
	set, err := domain.NewKindSet(domain.KindDefinition{Name: "kiwi"}, domain.KindDefinition{Name: "acme-wiki"})
	noError(t, err)
	kind, err := set.Kind("jira")
	wantText(t, err, "unknown source kind (known: acme-wiki, kiwi)")
	if !errors.Is(err, domain.ErrUnknownSourceKind) {
		t.Errorf("error does not match the unknown source kind sentinel")
	}
	if kind != (domain.SourceKind{}) {
		t.Errorf("returned kind %q, want none", kind.String())
	}
}

// @scenario specs/libs/retriever-contract/behaviours/source-kinds.feature: A failure of a kind never shows the word Outline for another kind
func TestSourceKindsFailureNeverShowsOutlineForAnotherKind(t *testing.T) {
	cases := []struct{ kind, failure, shown string }{
		{"kiwi", "Outline rejected the API key", "Kiwi rejected the API key"},
		{"jira", "Outline denied access", "Jira denied access"},
		{"outline", "Outline denied access", "Outline denied access"},
	}
	for _, tc := range cases {
		t.Run(tc.kind, func(t *testing.T) {
			kind, err := domain.BuiltinKinds().Kind(tc.kind)
			noError(t, err)
			failure := domain.KindError{Kind: kind, Err: errors.New(tc.failure)}
			if got := failure.Error(); got != tc.shown {
				t.Errorf("failure text = %q, want %q", got, tc.shown)
			}
		})
	}
}
