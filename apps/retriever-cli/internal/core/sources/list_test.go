package sources

import (
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// listCatalog holds two outline sources, one with tags company and wiki, one with engineering and wiki.
func listCatalog(t *testing.T) domain.Catalog {
	t.Helper()
	company, _ := domain.NewSourceID("outline")
	eng, _ := domain.NewSourceID("outline-engineering")
	docs, _ := domain.NewEndpoint("https://docs.example.com")
	wiki, _ := domain.NewEndpoint("https://wiki.example.com")
	keyA, _ := domain.NewCredentialRef("COMPANY_API_KEY")
	keyB, _ := domain.NewCredentialRef("ENGINEERING_API_KEY")
	tagsA, _ := domain.NewTagSet("company", "wiki")
	tagsB, _ := domain.NewTagSet("engineering", "wiki")
	cat, err := domain.NewCatalog([]domain.Source{
		domain.NewSource(company, domain.KindOutline, tagsA, docs, keyA),
		domain.NewSource(eng, domain.KindOutline, tagsB, wiki, keyB),
	})
	if err != nil {
		t.Fatalf("NewCatalog: %v", err)
	}
	return cat
}

func TestListSourcesReturnsIDKindAndTags(t *testing.T) {
	got := ListSources(listCatalog(t), domain.SourceFilter{Tags: mustTagSet(t)})
	if len(got) != 2 || got[0].ID != "outline" || got[1].ID != "outline-engineering" {
		t.Fatalf("ListSources = %+v", got)
	}
	if strings.Join(got[1].Tags, ",") != "engineering,wiki" {
		t.Fatalf("tags = %v", got[1].Tags)
	}
}

func TestListSourcesAppliesKindAndAllOfTags(t *testing.T) {
	outline := domain.KindOutline
	cases := []struct {
		name   string
		filter domain.SourceFilter
		want   string
	}{
		{name: "no filter", filter: domain.SourceFilter{Tags: mustTagSet(t)}, want: "outline,outline-engineering"},
		{name: "the kind only", filter: domain.SourceFilter{Kind: &outline, Tags: mustTagSet(t)}, want: "outline,outline-engineering"},
		{name: "one tag", filter: domain.SourceFilter{Tags: mustTagSet(t, "wiki")}, want: "outline,outline-engineering"},
		{name: "two tags, all of them", filter: domain.SourceFilter{Tags: mustTagSet(t, "wiki", "engineering")}, want: "outline-engineering"},
		{name: "two tags, no source has both", filter: domain.SourceFilter{Tags: mustTagSet(t, "company", "engineering")}, want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var ids []string
			for _, s := range ListSources(listCatalog(t), tc.filter) {
				ids = append(ids, s.ID)
			}
			if got := strings.Join(ids, ","); got != tc.want {
				t.Fatalf("ids = %q, want %q", got, tc.want)
			}
		})
	}
}
