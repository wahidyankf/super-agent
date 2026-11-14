package domain

import (
	"errors"
	"strings"
	"testing"
)

// fixtureID, fixtureEndpoint, fixtureCredential, fixtureIdentity and fixtureTags build the value objects of a test source. Each
// helper fails the test on a bad fixture, so a test reads only the rule it checks.
func fixtureID(t *testing.T, raw string) SourceID {
	t.Helper()
	id, err := NewSourceID(raw)
	if err != nil {
		t.Fatalf("NewSourceID(%q): %v", raw, err)
	}
	return id
}

func fixtureEndpoint(t *testing.T, raw string) Endpoint {
	t.Helper()
	endpoint, err := NewEndpoint(raw)
	if err != nil {
		t.Fatalf("NewEndpoint(%q): %v", raw, err)
	}
	return endpoint
}

func fixtureCredential(t *testing.T, raw string) CredentialRef {
	t.Helper()
	ref, err := NewCredentialRef(raw)
	if err != nil {
		t.Fatalf("NewCredentialRef(%q): %v", raw, err)
	}
	return ref
}

func fixtureIdentity(t *testing.T, raw string) IdentityRef {
	t.Helper()
	ref, err := NewIdentityRef(raw)
	if err != nil {
		t.Fatalf("NewIdentityRef(%q): %v", raw, err)
	}
	return ref
}

func fixtureTags(t *testing.T, raw ...string) TagSet {
	t.Helper()
	tags, err := NewTagSet(raw...)
	if err != nil {
		t.Fatalf("NewTagSet(%q): %v", raw, err)
	}
	return tags
}

// TestIdentityErrorsNameTheKindAndMatchTheirSentinels checks the text and the sentinel match of both identity errors.
func TestIdentityErrorsNameTheKindAndMatchTheirSentinels(t *testing.T) {
	required := IdentityRequiredError{Kind: KindJira}
	if got, want := required.Error(), "required for a jira source"; got != want {
		t.Errorf("IdentityRequiredError.Error() = %q, want %q", got, want)
	}
	if !required.Is(ErrIdentityRequired) {
		t.Error("IdentityRequiredError does not match ErrIdentityRequired")
	}
	if required.Is(ErrIdentityNotUsed) {
		t.Error("IdentityRequiredError matches ErrIdentityNotUsed")
	}

	notUsed := IdentityNotUsedError{Kind: KindOutline}
	if got, want := notUsed.Error(), "not used by an outline source"; got != want {
		t.Errorf("IdentityNotUsedError.Error() = %q, want %q", got, want)
	}
	if !notUsed.Is(ErrIdentityNotUsed) {
		t.Error("IdentityNotUsedError does not match ErrIdentityNotUsed")
	}
	if notUsed.Is(ErrIdentityRequired) {
		t.Error("IdentityNotUsedError matches ErrIdentityRequired")
	}
}

// TestKindPhraseChoosesTheArticleBySound checks the article of a vowel kind and of a consonant kind.
func TestKindPhraseChoosesTheArticleBySound(t *testing.T) {
	if got, want := kindPhrase(KindOutline), "an outline source"; got != want {
		t.Errorf("kindPhrase(outline) = %q, want %q", got, want)
	}
	if got, want := kindPhrase(KindKiwi), "a kiwi source"; got != want {
		t.Errorf("kindPhrase(kiwi) = %q, want %q", got, want)
	}
}

// TestKindFaultRefusesAnIdentityOnAKindThatDoesNotUseOne checks the fault for an outline source with an identity name,
// and that a source without one passes.
func TestKindFaultRefusesAnIdentityOnAKindThatDoesNotUseOne(t *testing.T) {
	base := NewSource(fixtureID(t, "docs"), KindOutline, fixtureTags(t), fixtureEndpoint(t, "https://docs.example.test"),
		fixtureCredential(t, "EXAMPLE_API_KEY"))

	if fault := kindFault(1, base); fault != nil {
		t.Fatalf("kindFault on a source without identity = %v, want nil", fault)
	}

	withIdentity := base.WithIdentity(fixtureIdentity(t, "EXAMPLE_EMAIL"))
	fault := kindFault(3, withIdentity)
	var sourceFault SourceFault
	if !errors.As(fault, &sourceFault) {
		t.Fatalf("kindFault = %v, want a SourceFault", fault)
	}
	if sourceFault.Position != 3 || sourceFault.Field != "identity_env" {
		t.Errorf("fault position and field = %d %q, want 3 %q", sourceFault.Position, sourceFault.Field, "identity_env")
	}
	if !errors.Is(fault, ErrIdentityNotUsed) {
		t.Errorf("fault = %v, want it to match ErrIdentityNotUsed", fault)
	}
}

// TestNewCatalogRefusesEachSourceFault checks each invariant that NewCatalog holds, by a fixture that breaks only it.
func TestNewCatalogRefusesEachSourceFault(t *testing.T) {
	outline := func(id, credential, endpoint string) Source {
		return NewSource(fixtureID(t, id), KindOutline, fixtureTags(t), fixtureEndpoint(t, endpoint), fixtureCredential(t, credential))
	}
	jira := func(id, endpoint, identity string) Source {
		credential := strings.ToUpper(id) + "_API_KEY"
		src := NewSource(fixtureID(t, id), KindJira, fixtureTags(t), fixtureEndpoint(t, endpoint), fixtureCredential(t, credential))
		if identity != "" {
			src = src.WithIdentity(fixtureIdentity(t, identity))
		}
		return src
	}

	tests := []struct {
		name    string
		sources []Source
		want    error
	}{
		{
			name:    "a jira source without an identity",
			sources: []Source{jira("tracker", "https://tracker.example.test", "")},
			want:    ErrIdentityRequired,
		},
		{
			name:    "one identity name on two hosts",
			sources: []Source{jira("one", "https://one.example.test", "EXAMPLE_EMAIL"), jira("two", "https://two.example.test", "EXAMPLE_EMAIL")},
			want:    ErrIdentityHostMismatch,
		},
		{
			name:    "the same source ID twice",
			sources: []Source{outline("docs", "DOCS_API_KEY", "https://docs.example.test"), outline("docs", "OTHER_API_KEY", "https://other.example.test")},
			want:    ErrDuplicateSource,
		},
		{
			name:    "one credential name on two hosts",
			sources: []Source{outline("docs", "SHARED_API_KEY", "https://docs.example.test"), outline("wiki", "SHARED_API_KEY", "https://wiki.example.test")},
			want:    ErrCredentialHostMismatch,
		},
		{
			name:    "the zero source",
			sources: []Source{{}},
			want:    ErrInvalidSourceID,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewCatalog(tt.sources)
			if !errors.Is(err, tt.want) {
				t.Fatalf("NewCatalog error = %v, want it to match %v", err, tt.want)
			}
		})
	}
}

// TestSourceAccessorsAndSummaryPrintOnlyTheSafeFields checks the tags and the printable summary of a source, which never
// hold the endpoint or the credential name.
func TestSourceAccessorsAndSummaryPrintOnlyTheSafeFields(t *testing.T) {
	src := NewSource(fixtureID(t, "docs"), KindOutline, fixtureTags(t, "alpha", "beta"),
		fixtureEndpoint(t, "https://docs.example.test"), fixtureCredential(t, "EXAMPLE_API_KEY"))

	if got := src.Tags().Values(); len(got) != 2 || got[0] != "alpha" || got[1] != "beta" {
		t.Errorf("Tags().Values() = %v, want [alpha beta]", got)
	}

	summary := src.Summary()
	if summary.ID != "docs" || summary.Kind != "outline" {
		t.Errorf("Summary() ID and Kind = %q %q, want docs outline", summary.ID, summary.Kind)
	}
	if len(summary.Tags) != 2 || summary.Tags[0] != "alpha" || summary.Tags[1] != "beta" {
		t.Errorf("Summary().Tags = %v, want [alpha beta]", summary.Tags)
	}
}
