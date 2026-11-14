package cli

import (
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/sources"
)

// twoSourceService returns a service over the two-source catalog: outline, read, company and wiki tags, and
// outline-engineering, write, engineering and wiki tags. Its gateways are never reached by these tests.
func twoSourceService(t *testing.T) sources.Service {
	t.Helper()
	mk := func(id, host, key string, tags ...string) domain.Source {
		sid, err := domain.NewSourceID(id)
		if err != nil {
			t.Fatalf("NewSourceID(%q): %v", id, err)
		}
		ep, err := domain.NewEndpoint("https://" + host)
		if err != nil {
			t.Fatalf("NewEndpoint(%q): %v", host, err)
		}
		ref, err := domain.NewCredentialRef(key)
		if err != nil {
			t.Fatalf("NewCredentialRef(%q): %v", key, err)
		}
		set, err := domain.NewTagSet(tags...)
		if err != nil {
			t.Fatalf("NewTagSet(%v): %v", tags, err)
		}
		return domain.NewSource(sid, domain.KindOutline, set, ep, ref)
	}
	cat, err := domain.NewCatalog([]domain.Source{
		mk("outline", "docs.example.test", "EXAMPLE_OUTLINE_API_KEY", "company", "wiki"),
		mk("outline-engineering", "wiki.engineering.example.com", "ENGINEERING_OUTLINE_API_KEY", "engineering", "wiki"),
	})
	if err != nil {
		t.Fatalf("NewCatalog: %v", err)
	}
	return sources.Service{Catalog: cat, Gateways: fixedResolver{}}
}

// TestSourcesListHelpNamesNoAccessLevel checks that the list help does not mention access levels, which were removed:
// the API token of a source is the only limit on what it can do.
func TestSourcesListHelpNamesNoAccessLevel(t *testing.T) {
	code, out, _ := runRetriever(t, []string{"sources", "list", "--help"}, outlineSetup{})

	if code != 0 || strings.Contains(strings.ToLower(out), "access") {
		t.Fatalf("sources list help must not mention access, got %q (exit %d)", out, code)
	}
}

func TestSourcesListPrintsEverySourceWithItsKindAndTags(t *testing.T) {
	setup := outlineSetup{service: twoSourceService(t), sink: newMemSink()}

	code, out, errOut := runRetriever(t, []string{"sources", "list"}, setup)

	if code != 0 || errOut != "" || !strings.HasPrefix(out, "2 sources\n") {
		t.Fatalf("sources list = %q (stderr %q, exit %d), want a count of two", out, errOut, code)
	}
	if !strings.Contains(out, "outline-engineering  outline  engineering,wiki") {
		t.Fatalf("the engineering row is missing from %q", out)
	}
	if strings.Contains(out, "https://") || strings.Contains(out, "EXAMPLE") || strings.Contains(out, "ENGINEERING_") {
		t.Fatalf("sources list printed an endpoint or a credential name: %q", out)
	}
}

func TestSourcesListFilterByKindAndTagsReachesTheOutput(t *testing.T) {
	setup := outlineSetup{service: twoSourceService(t), sink: newMemSink()}

	code, out, _ := runRetriever(t, []string{"sources", "list", "--kind", "outline", "--tag", "wiki,engineering"}, setup)

	if code != 0 || !strings.HasPrefix(out, "1 source\n") || !strings.Contains(out, "outline-engineering") {
		t.Fatalf("filtered list = %q (exit %d), want only outline-engineering", out, code)
	}
}

func TestSourcesListRefusesAnUnknownKindAndAnInvalidTag(t *testing.T) {
	setup := outlineSetup{service: twoSourceService(t), sink: newMemSink()}
	cases := []struct {
		name string
		argv []string
		want string
	}{
		{name: "an unknown kind", argv: []string{"sources", "list", "--kind", "notion"}, want: "unknown source kind (known: outline, kiwi, jira)"},
		{name: "an invalid tag", argv: []string{"sources", "list", "--tag", "Wiki"}, want: "invalid tag"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, out, errOut := runRetriever(t, tc.argv, setup)
			if code != 2 || !strings.Contains(errOut, tc.want) || out != "" {
				t.Fatalf("want exit 2 with %q on stderr and no rows, got stdout %q stderr %q (exit %d)", tc.want, out, errOut, code)
			}
		})
	}
}

func TestSourceFilterParsesKindAndTheCommaSeparatedTags(t *testing.T) {
	inv := &invocation{values: map[string]string{"--kind": "outline", "--tag": "wiki,engineering"}}

	filter, err := sourceFilter(inv)

	if err != nil {
		t.Fatalf("sourceFilter error = %v", err)
	}
	if filter.Kind == nil || filter.Kind.String() != "outline" {
		t.Fatalf("kind = %v, want outline", filter.Kind)
	}
	if got := strings.Join(filter.Tags.Values(), ","); got != "engineering,wiki" {
		t.Fatalf("tags = %q, want engineering,wiki", got)
	}
}

func TestSourceFilterWithNoFlagsMatchesEverything(t *testing.T) {
	filter, err := sourceFilter(&invocation{values: map[string]string{}})

	if err != nil || filter.Kind != nil || len(filter.Tags.Values()) != 0 {
		t.Fatalf("sourceFilter = %+v, %v; want no kind and no tags", filter, err)
	}
}

func TestSourceFilterReturnsTheUserTextForABadFlag(t *testing.T) {
	cases := []struct {
		name   string
		values map[string]string
		want   string
	}{
		{name: "an unknown kind", values: map[string]string{"--kind": "notion"}, want: "unknown source kind (known: outline, kiwi, jira)"},
		{name: "an invalid tag", values: map[string]string{"--tag": "wiki,Wiki"}, want: "invalid tag"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := sourceFilter(&invocation{values: tc.values})
			if err == nil || err.Error() != tc.want {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestWriteSourceRowsPadsColumnsAndShowsADashForNoTags(t *testing.T) {
	var b strings.Builder
	writeSourceRows(&b, []domain.SourceSummary{
		{ID: "outline", Kind: "outline"},
		{ID: "outline-engineering", Kind: "outline", Tags: []string{"engineering", "wiki"}},
	})

	want := "2 sources\n" +
		"  outline              outline  -\n" +
		"  outline-engineering  outline  engineering,wiki\n"
	if b.String() != want {
		t.Fatalf("rows = %q, want %q", b.String(), want)
	}
}

func TestWriteSourceRowsUsesTheSingularForOneSource(t *testing.T) {
	var b strings.Builder
	writeSourceRows(&b, []domain.SourceSummary{{ID: "outline", Kind: "outline"}})

	if !strings.HasPrefix(b.String(), "1 source\n") {
		t.Fatalf("rows = %q, want the singular count", b.String())
	}
}

func TestWriteSourceRowsOfNoSourcesPrintsZero(t *testing.T) {
	var b strings.Builder
	writeSourceRows(&b, nil)

	if b.String() != "0 sources\n" {
		t.Fatalf("rows = %q, want the zero count and no row", b.String())
	}
}

func TestSourceArgIsNilWhenTheFlagIsOmittedAndTheValueOtherwise(t *testing.T) {
	if got := sourceArg(&invocation{values: map[string]string{}}); got != nil {
		t.Fatalf("sourceArg with no --source = %q, want nil so the default rule applies", *got)
	}
	got := sourceArg(&invocation{values: map[string]string{"--source": "outline-engineering"}})
	if got == nil || *got != "outline-engineering" {
		t.Fatalf("sourceArg = %v, want outline-engineering", got)
	}
}
