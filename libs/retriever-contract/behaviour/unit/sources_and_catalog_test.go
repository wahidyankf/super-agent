package unit_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// numberedOutlineSources returns n valid outline sources with distinct IDs and distinct credential names on one host.
func numberedOutlineSources(t *testing.T, n int) []domain.Source {
	t.Helper()
	sources := make([]domain.Source, 0, n)
	for i := 1; i <= n; i++ {
		sources = append(sources, outlineSource(t, fmt.Sprintf("src-%02d", i), outlineHost,
			fmt.Sprintf("OUTLINE_KEY_%02d_API_KEY", i)))
	}
	return sources
}

// @scenario specs/libs/retriever-contract/behaviours/sources-and-catalog.feature: A catalog built from no sources is refused
func TestCatalogRefusesNoSources(t *testing.T) {
	catalog, err := domain.NewCatalog([]domain.Source{})
	wantText(t, err, "the catalog needs at least one source")
	if n := len(catalog.All()); n != 0 {
		t.Errorf("returned a catalog of %d sources, want none", n)
	}
}

// @scenario specs/libs/retriever-contract/behaviours/sources-and-catalog.feature: A catalog of 64 sources is accepted
func TestCatalogAcceptsSixtyFourSources(t *testing.T) {
	catalog := catalogOf(t, numberedOutlineSources(t, 64)...)
	if n := len(catalog.All()); n != 64 {
		t.Errorf("catalog holds %d sources, want 64", n)
	}
}

// @scenario specs/libs/retriever-contract/behaviours/sources-and-catalog.feature: A catalog of 65 sources is refused
func TestCatalogRefusesSixtyFiveSources(t *testing.T) {
	catalog, err := domain.NewCatalog(numberedOutlineSources(t, 65))
	wantText(t, err, "the catalog holds at most 64 sources")
	if n := len(catalog.All()); n != 0 {
		t.Errorf("returned a catalog of %d sources, want none", n)
	}
}

// @scenario specs/libs/retriever-contract/behaviours/sources-and-catalog.feature: A source built with no ID is refused with its position and field
func TestCatalogRefusesSourceWithNoIDWithPositionAndField(t *testing.T) {
	src := domain.NewSource(domain.SourceID{}, domain.KindOutline, domain.TagSet{}, endpointOf(t, outlineHost),
		credentialOf(t, outlineKey))
	_, err := domain.NewCatalog([]domain.Source{src})
	wantText(t, err, "source 1 id: invalid source ID")
}

// @scenario specs/libs/retriever-contract/behaviours/sources-and-catalog.feature: Two sources with one ID are refused with the position of the second
func TestCatalogRefusesTwoSourcesWithOneIDWithPositionOfSecond(t *testing.T) {
	_, err := domain.NewCatalog([]domain.Source{
		outlineSource(t, "wiki", outlineHost, outlineKey),
		outlineSource(t, "wiki", outlineHost, outlineKey),
	})
	wantText(t, err, "source 2 id: duplicate")
}

// @scenario specs/libs/retriever-contract/behaviours/sources-and-catalog.feature: A jira source with no identity name is refused
func TestCatalogRefusesJiraSourceWithNoIdentityName(t *testing.T) {
	src := domain.NewSource(sourceIDOf(t, "jira"), domain.KindJira, tagsOf(t), endpointOf(t, "a.example.test"),
		credentialOf(t, "JIRA_API_KEY"))
	_, err := domain.NewCatalog([]domain.Source{src})
	wantText(t, err, "source 1 identity_env: required for a jira source")
}

// @scenario specs/libs/retriever-contract/behaviours/sources-and-catalog.feature: An outline source that names an identity is refused
func TestCatalogRefusesOutlineSourceThatNamesIdentity(t *testing.T) {
	src := outlineSource(t, "docs", outlineHost, outlineKey).WithIdentity(identityOf(t, "OUTLINE_EMAIL"))
	_, err := domain.NewCatalog([]domain.Source{src})
	wantText(t, err, "source 1 identity_env: not used by an outline source")
}

// @scenario specs/libs/retriever-contract/behaviours/sources-and-catalog.feature: Two sources of one host may share a credential name
func TestCatalogAcceptsTwoSourcesOfOneHostSharingCredentialName(t *testing.T) {
	catalog := catalogOf(t,
		outlineSource(t, "a", outlineHost, outlineKey),
		outlineSource(t, "b", outlineHost, outlineKey),
	)
	if n := len(catalog.All()); n != 2 {
		t.Errorf("catalog holds %d sources, want 2", n)
	}
}

// @scenario specs/libs/retriever-contract/behaviours/sources-and-catalog.feature: A credential name that serves two hosts is refused at the second source
func TestCatalogRefusesCredentialNameThatServesTwoHostsAtSecondSource(t *testing.T) {
	_, err := domain.NewCatalog([]domain.Source{
		outlineSource(t, "a", outlineHost, outlineKey),
		outlineSource(t, "b", "wiki.example.test", outlineKey),
	})
	wantText(t, err, "source 2 credential_env: one credential serves one host only")
}

// @scenario specs/libs/retriever-contract/behaviours/sources-and-catalog.feature: An identity name that serves two hosts is refused at the second source
func TestCatalogRefusesIdentityNameThatServesTwoHostsAtSecondSource(t *testing.T) {
	_, err := domain.NewCatalog([]domain.Source{
		jiraSource(t, "jira-a", "a.example.test", "JIRA_A_API_KEY", "JIRA_EMAIL"),
		jiraSource(t, "jira-b", "b.example.test", "JIRA_B_API_KEY", "JIRA_EMAIL"),
	})
	wantText(t, err, "source 2 identity_env: one identity serves one host only")
}

// @scenario specs/libs/retriever-contract/behaviours/sources-and-catalog.feature: The catalog lists its sources sorted by ID
func TestCatalogListsSourcesSortedByID(t *testing.T) {
	catalog := catalogOf(t,
		outlineSource(t, "wiki", outlineHost, outlineKey),
		outlineSource(t, "docs", outlineHost, outlineKey),
		outlineSource(t, "team", outlineHost, outlineKey),
	)
	if got, want := idsOf(catalog.All()), []string{"docs", "team", "wiki"}; !sameStrings(got, want) {
		t.Errorf("listed order = %v, want %v", got, want)
	}
}

// @scenario specs/libs/retriever-contract/behaviours/sources-and-catalog.feature: A source is found by its ID only when the catalog holds it
func TestCatalogFindsSourceByIDOnlyWhenHeld(t *testing.T) {
	catalog := catalogOf(t,
		outlineSource(t, "docs", outlineHost, outlineKey),
		outlineSource(t, "wiki", outlineHost, outlineKey),
	)
	cases := []struct {
		id    string
		found bool
	}{
		{"docs", true},
		{"team", false},
	}
	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			_, ok := catalog.ByID(sourceIDOf(t, tc.id))
			if ok != tc.found {
				t.Errorf("found %q = %v, want %v", tc.id, ok, tc.found)
			}
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/sources-and-catalog.feature: A filter selects the sources of its kind that carry all of its tags
func TestCatalogFilterSelectsSourcesOfKindCarryingAllTags(t *testing.T) {
	catalog := catalogOf(t,
		outlineSource(t, "a", outlineHost, outlineKey, "wiki", "tests"),
		outlineSource(t, "b", outlineHost, outlineKey, "wiki"),
		kiwiSource(t, "c", kiwiHost, kiwiKey, "wiki", "tests"),
	)
	outline, kiwi, jira := "outline", "kiwi", "jira"
	cases := []struct {
		name string
		kind *string
		tags []string
		want []string
	}{
		{"no kind and no tags", nil, nil, []string{"a", "b", "c"}},
		{"kind outline", &outline, nil, []string{"a", "b"}},
		{"tag wiki", nil, []string{"wiki"}, []string{"a", "b", "c"}},
		{"tags wiki and tests", nil, []string{"wiki", "tests"}, []string{"a", "c"}},
		{"kind kiwi and tags wiki and tests", &kiwi, []string{"wiki", "tests"}, []string{"c"}},
		{"kind jira", &jira, nil, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			filter, err := domain.NewSourceFilter(tc.kind, tc.tags...)
			noError(t, err)
			if got := idsOf(catalog.Find(filter)); !sameStrings(got, tc.want) {
				t.Errorf("found %v, want %v", got, tc.want)
			}
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/sources-and-catalog.feature: A filter with a kind that is not known is refused
func TestCatalogFilterRefusesKindThatIsNotKnown(t *testing.T) {
	github := "github"
	_, err := domain.NewSourceFilter(&github)
	wantText(t, err, "unknown source kind (known: outline, kiwi, jira)")
}

// @scenario specs/libs/retriever-contract/behaviours/sources-and-catalog.feature: A filter with an invalid tag is refused
func TestCatalogFilterRefusesInvalidTag(t *testing.T) {
	// No kind is given: an empty kind text would be refused as an unknown kind before the tag is read.
	_, err := domain.NewSourceFilter(nil, "Wiki")
	wantText(t, err, "invalid tag")
}

// @scenario specs/libs/retriever-contract/behaviours/sources-and-catalog.feature: A source named by ID is selected when it exists and has the kind asked for
func TestCatalogSelectsNamedSourceWhenItExistsWithKind(t *testing.T) {
	catalog := catalogOf(t,
		outlineSource(t, "docs", outlineHost, outlineKey),
		kiwiSource(t, "tests", kiwiHost, kiwiKey),
	)
	id := sourceIDOf(t, "docs")
	src, err := catalog.Select(domain.KindOutline, &id)
	noError(t, err)
	if got := src.ID().String(); got != "docs" {
		t.Errorf("selected source %q, want %q", got, "docs")
	}
}

// @scenario specs/libs/retriever-contract/behaviours/sources-and-catalog.feature: A named source that the catalog does not hold is refused
func TestCatalogRefusesNamedSourceThatCatalogDoesNotHold(t *testing.T) {
	catalog := catalogOf(t, outlineSource(t, "docs", outlineHost, outlineKey))
	id := sourceIDOf(t, "team")
	src, err := catalog.Select(domain.KindOutline, &id)
	if !errors.Is(err, domain.ErrUnknownSource) {
		t.Fatalf("error = %v, want the unknown source error", err)
	}
	wantText(t, err, `unknown source "team"`)
	if id := src.ID().String(); id != "" {
		t.Errorf("returned source %q, want none", id)
	}
}

// @scenario specs/libs/retriever-contract/behaviours/sources-and-catalog.feature: A named source of another kind is refused
func TestCatalogRefusesNamedSourceOfAnotherKind(t *testing.T) {
	catalog := catalogOf(t,
		outlineSource(t, "docs", outlineHost, outlineKey),
		kiwiSource(t, "tests", kiwiHost, kiwiKey),
	)
	id := sourceIDOf(t, "tests")
	_, err := catalog.Select(domain.KindOutline, &id)
	wantText(t, err, "the source is of another kind")
}

// @scenario specs/libs/retriever-contract/behaviours/sources-and-catalog.feature: With no ID, the only source of the kind is selected
func TestCatalogSelectsOnlySourceOfKindWithoutID(t *testing.T) {
	catalog := catalogOf(t,
		outlineSource(t, "docs", outlineHost, outlineKey),
		kiwiSource(t, "tests", kiwiHost, kiwiKey),
	)
	src, err := catalog.Select(domain.KindKiwi, nil)
	noError(t, err)
	if got := src.ID().String(); got != "tests" {
		t.Errorf("selected source %q, want %q", got, "tests")
	}
}

// @scenario specs/libs/retriever-contract/behaviours/sources-and-catalog.feature: With no ID, a kind with no source is refused
func TestCatalogRefusesKindWithNoSourceWithoutID(t *testing.T) {
	catalog := catalogOf(t, outlineSource(t, "docs", outlineHost, outlineKey))
	_, err := catalog.Select(domain.KindKiwi, nil)
	wantText(t, err, "no kiwi source is configured")
}

// @scenario specs/libs/retriever-contract/behaviours/sources-and-catalog.feature: With no ID, a kind with two sources is refused and both IDs are listed
func TestCatalogRefusesKindWithTwoSourcesWithoutIDAndListsBoth(t *testing.T) {
	catalog := catalogOf(t,
		outlineSource(t, "wiki", outlineHost, outlineKey),
		outlineSource(t, "docs", outlineHost, outlineKey),
	)
	_, err := catalog.Select(domain.KindOutline, nil)
	wantText(t, err, "more than one outline source exists: docs, wiki")
}

// @scenario specs/libs/retriever-contract/behaviours/sources-and-catalog.feature: A source summary shows the ID, kind, and tags, and no endpoint
func TestSourceSummaryShowsIDKindAndTagsOnly(t *testing.T) {
	src := outlineSource(t, "docs", outlineHost, outlineKey, "wiki", "tests")
	summary := src.Summary()
	if summary.ID != "docs" {
		t.Errorf("summary ID = %q, want %q", summary.ID, "docs")
	}
	if summary.Kind != "outline" {
		t.Errorf("summary kind = %q, want %q", summary.Kind, "outline")
	}
	if want := []string{"tests", "wiki"}; !sameStrings(summary.Tags, want) {
		t.Errorf("summary tags = %v, want %v", summary.Tags, want)
	}
}

// @scenario specs/libs/retriever-contract/behaviours/sources-and-catalog.feature: A source given a plugin reports the pinned plugin and no other source does
func TestSourceGivenPluginReportsPinnedPlugin(t *testing.T) {
	ref, err := domain.NewPluginRef("/opt/retriever/plugins/example", validPluginPin)
	noError(t, err)
	src := outlineSource(t, "docs", outlineHost, outlineKey).WithPlugin(ref)
	if !src.HasPlugin() {
		t.Fatalf("HasPlugin = false, want true")
	}
	if got := src.Plugin().Path(); got != "/opt/retriever/plugins/example" {
		t.Errorf("plugin path = %q, want %q", got, "/opt/retriever/plugins/example")
	}
}

// @scenario specs/libs/retriever-contract/behaviours/sources-and-catalog.feature: A source with no plugin reports none
func TestSourceWithNoPluginReportsNone(t *testing.T) {
	src := outlineSource(t, "docs", outlineHost, outlineKey)
	if src.HasPlugin() {
		t.Errorf("HasPlugin = true, want false")
	}
	if strings.TrimSpace(src.Plugin().Path()) != "" {
		t.Errorf("plugin path = %q, want none", src.Plugin().Path())
	}
}
