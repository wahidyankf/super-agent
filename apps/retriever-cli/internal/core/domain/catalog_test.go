package domain

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// otherKind is a second kind that exists only inside this package, to test the wrong-kind rule.
var otherKind = SourceKind{value: "other"}

func testSource(t *testing.T, id, kind string, host, cred string) Source {
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
	k := KindOutline
	if kind == otherKind.value {
		k = otherKind
	}
	return NewSource(sid, k, mustTagSet(t), AccessRead, ep, ref)
}

func tagged(t *testing.T, src Source, access Access, tags ...string) Source {
	t.Helper()
	return NewSource(src.ID(), src.Kind(), mustTagSet(t, tags...), access, src.Endpoint(), src.Credential())
}

func manySources(t *testing.T, n int) []Source {
	t.Helper()
	out := make([]Source, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, testSource(t, fmt.Sprintf("src-%d", i), "outline", fmt.Sprintf("h%d.example.com", i), fmt.Sprintf("K%d_API_KEY", i)))
	}
	return out
}

func TestNewCatalogSizeLimits(t *testing.T) {
	if _, err := NewCatalog(nil); !errors.Is(err, ErrEmptyCatalog) {
		t.Fatalf("empty catalog error = %v, want ErrEmptyCatalog", err)
	}
	if _, err := NewCatalog(manySources(t, 65)); !errors.Is(err, ErrTooManySources) {
		t.Fatalf("65 sources error = %v, want ErrTooManySources", err)
	}
	if _, err := NewCatalog(manySources(t, 64)); err != nil {
		t.Fatalf("64 sources error = %v, want nil", err)
	}
}

func TestNewCatalogRejectsDuplicateIDsWithPosition(t *testing.T) {
	a := testSource(t, "outline", "outline", "docs.example.com", "A_API_KEY")
	b := testSource(t, "outline", "outline", "wiki.example.com", "B_API_KEY")
	_, err := NewCatalog([]Source{a, b})
	if !errors.Is(err, ErrDuplicateSource) {
		t.Fatalf("duplicate error = %v, want ErrDuplicateSource", err)
	}
	var fault SourceFault
	if !errors.As(err, &fault) || fault.Position != 2 || fault.Field != "id" {
		t.Fatalf("fault = %+v, want position 2 field id", fault)
	}
	if want := "source 2 id: " + ErrDuplicateSource.Error(); fault.Error() != want {
		t.Fatalf("fault.Error() = %q, want %q", fault.Error(), want)
	}
}

func TestNewCatalogRejectsOneCredentialOnTwoHosts(t *testing.T) {
	a := testSource(t, "outline", "outline", "docs.example.com", "SHARED_API_KEY")
	b := testSource(t, "outline-engineering", "outline", "wiki.example.com", "SHARED_API_KEY")
	_, err := NewCatalog([]Source{a, b})
	var fault SourceFault
	if !errors.Is(err, ErrCredentialHostMismatch) || !errors.As(err, &fault) || fault.Position != 2 || fault.Field != "credential_env" {
		t.Fatalf("host mismatch error = %v", err)
	}
}

func TestNewCatalogAllowsOneCredentialOnOneHost(t *testing.T) {
	a := testSource(t, "outline", "outline", "docs.example.com", "SHARED_API_KEY")
	b := testSource(t, "outline-two", "outline", "docs.example.com", "SHARED_API_KEY")
	if _, err := NewCatalog([]Source{a, b}); err != nil {
		t.Fatalf("same credential on the same host error = %v, want nil", err)
	}
}

func TestCatalogAllIsSortedAndByIDFinds(t *testing.T) {
	a := testSource(t, "outline-engineering", "outline", "wiki.example.com", "E_API_KEY")
	b := testSource(t, "outline", "outline", "docs.example.com", "O_API_KEY")
	cat, err := NewCatalog([]Source{a, b})
	if err != nil {
		t.Fatalf("NewCatalog error = %v", err)
	}
	all := cat.All()
	if len(all) != 2 || all[0].ID().String() != "outline" || all[1].ID().String() != "outline-engineering" {
		t.Fatalf("All() is not sorted by ID: %v", all)
	}
	id, _ := NewSourceID("outline-engineering")
	if got, ok := cat.ByID(id); !ok || got.ID() != id {
		t.Fatalf("ByID(outline-engineering) = %v, %v", got, ok)
	}
	missing, _ := NewSourceID("rnd")
	if _, ok := cat.ByID(missing); ok {
		t.Fatalf("ByID(rnd) found a source")
	}
}

func TestCatalogFindIsAllOfTagsAndKind(t *testing.T) {
	company := tagged(t, testSource(t, "outline", "outline", "docs.example.com", "O_API_KEY"), AccessRead, "company", "wiki")
	eng := tagged(t, testSource(t, "outline-engineering", "outline", "wiki.example.com", "E_API_KEY"), AccessWrite, "engineering", "wiki")
	other := tagged(t, testSource(t, "notes", "other", "notes.example.com", "N_API_KEY"), AccessRead, "wiki")
	cat, err := NewCatalog([]Source{company, eng, other})
	if err != nil {
		t.Fatalf("NewCatalog error = %v", err)
	}
	outline := KindOutline
	cases := []struct {
		name   string
		filter SourceFilter
		want   string
	}{
		{name: "an empty filter returns every source", filter: SourceFilter{Tags: mustTagSet(t)}, want: "notes,outline,outline-engineering"},
		{name: "one tag matches every source that has it", filter: SourceFilter{Tags: mustTagSet(t, "wiki")}, want: "notes,outline,outline-engineering"},
		{name: "all tags must match", filter: SourceFilter{Tags: mustTagSet(t, "wiki", "engineering")}, want: "outline-engineering"},
		{name: "a kind narrows the list", filter: SourceFilter{Kind: &outline, Tags: mustTagSet(t, "wiki")}, want: "outline,outline-engineering"},
		{name: "no source has every tag", filter: SourceFilter{Tags: mustTagSet(t, "company", "engineering")}, want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var ids []string
			for _, s := range cat.Find(tc.filter) {
				ids = append(ids, s.ID().String())
			}
			if got := strings.Join(ids, ","); got != tc.want {
				t.Fatalf("Find() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestCatalogSelectDefaultRule(t *testing.T) {
	one := testSource(t, "outline", "outline", "docs.example.com", "O_API_KEY")
	solo, err := NewCatalog([]Source{one})
	if err != nil {
		t.Fatalf("NewCatalog error = %v", err)
	}
	got, err := solo.Select(KindOutline, nil)
	if err != nil || got.ID().String() != "outline" {
		t.Fatalf("Select(outline, nil) with one source = %v, %v", got.ID(), err)
	}

	two := testSource(t, "outline-engineering", "outline", "wiki.example.com", "E_API_KEY")
	pair, err := NewCatalog([]Source{one, two})
	if err != nil {
		t.Fatalf("NewCatalog error = %v", err)
	}
	var amb AmbiguousSourceError
	if _, err := pair.Select(KindOutline, nil); !errors.As(err, &amb) || len(amb.IDs) != 2 || amb.IDs[0] != "outline" {
		t.Fatalf("Select(outline, nil) with two sources = %v, want AmbiguousSourceError", err)
	}
	if want := "more than one outline source exists: outline, outline-engineering"; amb.Error() != want {
		t.Fatalf("AmbiguousSourceError text = %q, want %q", amb.Error(), want)
	}
	id, _ := NewSourceID("outline-engineering")
	if got, err := pair.Select(KindOutline, &id); err != nil || got.ID() != id {
		t.Fatalf("Select with an ID = %v, %v", got.ID(), err)
	}
}

func TestCatalogSelectByIDErrors(t *testing.T) {
	one := testSource(t, "outline", "outline", "docs.example.com", "O_API_KEY")
	foreign := testSource(t, "notes", "other", "notes.example.com", "N_API_KEY")
	cat, err := NewCatalog([]Source{one, foreign})
	if err != nil {
		t.Fatalf("NewCatalog error = %v", err)
	}
	rnd, _ := NewSourceID("rnd")
	var unknown UnknownSourceError
	if _, err := cat.Select(KindOutline, &rnd); !errors.Is(err, ErrUnknownSource) || !errors.As(err, &unknown) || unknown.ID != "rnd" {
		t.Fatalf("unknown ID error = %v", err)
	}
	if want := `unknown source "rnd"`; unknown.Error() != want {
		t.Fatalf("UnknownSourceError text = %q, want %q", unknown.Error(), want)
	}
	notes, _ := NewSourceID("notes")
	if _, err := cat.Select(KindOutline, &notes); !errors.Is(err, ErrWrongSourceKind) {
		t.Fatalf("wrong kind error = %v, want ErrWrongSourceKind", err)
	}
}

func TestCatalogSelectWithNoSourceOfKind(t *testing.T) {
	foreign := testSource(t, "notes", "other", "notes.example.com", "N_API_KEY")
	cat, err := NewCatalog([]Source{foreign})
	if err != nil {
		t.Fatalf("NewCatalog error = %v", err)
	}
	if _, err := cat.Select(KindOutline, nil); !errors.Is(err, ErrNoSourceOfKind) {
		t.Fatalf("no source of kind error = %v, want ErrNoSourceOfKind", err)
	}
	var zero Catalog
	if _, err := zero.Select(KindOutline, nil); !errors.Is(err, ErrNoSourceOfKind) {
		t.Fatalf("zero catalog Select error = %v, want ErrNoSourceOfKind", err)
	}
	if len(zero.All()) != 0 || len(zero.Find(SourceFilter{})) != 0 {
		t.Fatalf("zero catalog holds sources")
	}
}

// TestNewSourceFilterParsesTheTextArguments checks the text filter that the CLI flags and the GraphQL arguments share:
// an absent kind matches every kind, a given kind must be known, and each tag is validated.
func TestNewSourceFilterParsesTheTextArguments(t *testing.T) {
	outline := "outline"
	notion := "notion"

	none, err := NewSourceFilter(nil)
	if err != nil || none.Kind != nil || len(none.Tags.Values()) != 0 {
		t.Fatalf("no arguments: filter = %+v, err = %v; want no kind and no tags", none, err)
	}
	kinded, err := NewSourceFilter(&outline, "wiki", "company")
	if err != nil || kinded.Kind == nil || *kinded.Kind != KindOutline || len(kinded.Tags.Values()) != 2 {
		t.Fatalf("outline with two tags: filter = %+v, err = %v", kinded, err)
	}
	if _, err := NewSourceFilter(&notion); !errors.Is(err, ErrUnknownSourceKind) {
		t.Fatalf("unknown kind error = %v, want ErrUnknownSourceKind", err)
	}
	if _, err := NewSourceFilter(nil, "Wiki"); !errors.Is(err, ErrInvalidTag) {
		t.Fatalf("invalid tag error = %v, want ErrInvalidTag", err)
	}
}
