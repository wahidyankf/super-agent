package unit_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// @scenario specs/libs/retriever-contract/behaviours/identifiers.feature: A collection or document ID of letters, digits, _ and - is accepted
func TestIdentifiersAcceptCollectionAndDocumentIDsOfTheGrammar(t *testing.T) {
	cases := []struct{ kind, id string }{
		{"collection", "col-tree"},
		{"collection", "col_2"},
		{"collection", "A1-b_2"},
		{"document", "doc-1"},
		{"document", "Doc_9"},
	}
	for _, tc := range cases {
		t.Run(tc.kind+" "+tc.id, func(t *testing.T) {
			var got string
			if tc.kind == "collection" {
				id, err := domain.NewCollectionID(tc.id)
				noError(t, err)
				got = id.String()
			} else {
				id, err := domain.NewDocumentID(tc.id)
				noError(t, err)
				got = id.String()
			}
			if got != tc.id {
				t.Errorf("ID text = %q, want %q", got, tc.id)
			}
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/identifiers.feature: A collection or document ID with another character is refused
func TestIdentifiersRefuseCollectionAndDocumentIDsWithAnotherCharacter(t *testing.T) {
	cases := []struct{ kind, id string }{
		{"collection", "col tree"},
		{"collection", "col/tree"},
		{"collection", "col.tree"},
		{"document", "doc 1"},
		{"document", "doc#1"},
		{"document", ""},
	}
	for _, tc := range cases {
		t.Run(tc.kind+" "+tc.id, func(t *testing.T) {
			if tc.kind == "collection" {
				id, err := domain.NewCollectionID(tc.id)
				wantSentinel(t, err, domain.ErrInvalidCollectionID)
				if id != (domain.CollectionID{}) {
					t.Errorf("returned collection ID %q, want none", id.String())
				}
			} else {
				id, err := domain.NewDocumentID(tc.id)
				wantSentinel(t, err, domain.ErrInvalidDocumentID)
				if id != (domain.DocumentID{}) {
					t.Errorf("returned document ID %q, want none", id.String())
				}
			}
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/identifiers.feature: An ID of exactly 255 characters passes the ID check
func TestIdentifiersPassIDOf255Characters(t *testing.T) {
	noError(t, domain.ValidateID(strings.Repeat("a", 255)))
}

// @scenario specs/libs/retriever-contract/behaviours/identifiers.feature: An ID of 256 characters fails the ID check
func TestIdentifiersFailIDOf256Characters(t *testing.T) {
	wantSentinel(t, domain.ValidateID(strings.Repeat("a", 256)), domain.ErrInvalidID)
}

// @scenario specs/libs/retriever-contract/behaviours/identifiers.feature: A source ID that is a lowercase slug is accepted
func TestIdentifiersAcceptLowercaseSlugSourceIDs(t *testing.T) {
	for _, raw := range []string{"outline", "kiwi", "jira", "a1-b2", "team-2-eu"} {
		t.Run(raw, func(t *testing.T) {
			id, err := domain.NewSourceID(raw)
			noError(t, err)
			if got := id.String(); got != raw {
				t.Errorf("source ID text = %q, want %q", got, raw)
			}
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/identifiers.feature: A source ID that is not a lowercase slug is refused
func TestIdentifiersRefuseNonSlugSourceIDs(t *testing.T) {
	for _, raw := range []string{"Outline", "a--b", "1outline", "outline-", "out line", "a/b", ""} {
		t.Run(raw, func(t *testing.T) {
			id, err := domain.NewSourceID(raw)
			wantSentinel(t, err, domain.ErrInvalidSourceID)
			if id != (domain.SourceID{}) {
				t.Errorf("returned source ID %q, want none", id.String())
			}
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/identifiers.feature: A source ID of exactly 32 characters is accepted
func TestIdentifiersAcceptSourceIDOf32Characters(t *testing.T) {
	raw := strings.Repeat("a", 32)
	id, err := domain.NewSourceID(raw)
	noError(t, err)
	if got := id.String(); got != raw {
		t.Errorf("source ID text = %q, want the 32 letter a characters", got)
	}
}

// @scenario specs/libs/retriever-contract/behaviours/identifiers.feature: A source ID of 33 characters is refused
func TestIdentifiersRefuseSourceIDOf33Characters(t *testing.T) {
	id, err := domain.NewSourceID(strings.Repeat("a", 33))
	wantSentinel(t, err, domain.ErrInvalidSourceID)
	if id != (domain.SourceID{}) {
		t.Errorf("returned source ID %q, want none", id.String())
	}
}

// @scenario specs/libs/retriever-contract/behaviours/identifiers.feature: A tag that is a lowercase slug is accepted
func TestIdentifiersAcceptLowercaseSlugTags(t *testing.T) {
	for _, raw := range []string{"wiki", "2026", "team-a", "tests-2"} {
		t.Run(raw, func(t *testing.T) {
			_, err := domain.NewTag(raw)
			noError(t, err)
			// A tag has no text accessor of its own, so its text is read through the set that holds it.
			if got := tagsOf(t, raw).Values(); !sameStrings(got, []string{raw}) {
				t.Errorf("tag text = %v, want %q", got, raw)
			}
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/identifiers.feature: A tag with an uppercase letter, double hyphen, or symbol is refused
func TestIdentifiersRefuseTagsWithUppercaseDoubleHyphenOrSymbol(t *testing.T) {
	for _, raw := range []string{"Wiki", "team--a", "team_a", "-team", "team a", ""} {
		t.Run(raw, func(t *testing.T) {
			tag, err := domain.NewTag(raw)
			wantSentinel(t, err, domain.ErrInvalidTag)
			if tag != (domain.Tag{}) {
				t.Errorf("returned a tag, want none")
			}
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/identifiers.feature: A tag of 32 characters is accepted
func TestIdentifiersAcceptTagOf32Characters(t *testing.T) {
	raw := strings.Repeat("a", 32)
	_, err := domain.NewTag(raw)
	noError(t, err)
	if got := tagsOf(t, raw).Values(); !sameStrings(got, []string{raw}) {
		t.Errorf("tag text = %v, want the 32 letter a characters", got)
	}
}

// @scenario specs/libs/retriever-contract/behaviours/identifiers.feature: A tag of 33 characters is refused
func TestIdentifiersRefuseTagOf33Characters(t *testing.T) {
	tag, err := domain.NewTag(strings.Repeat("a", 33))
	wantSentinel(t, err, domain.ErrInvalidTag)
	if tag != (domain.Tag{}) {
		t.Errorf("returned a tag, want none")
	}
}

// @scenario specs/libs/retriever-contract/behaviours/identifiers.feature: A tag set drops a repeated tag and keeps the rest in sorted order
func TestIdentifiersTagSetDropsRepeatsAndSorts(t *testing.T) {
	set, err := domain.NewTagSet("wiki", "tests", "wiki")
	noError(t, err)
	if got, want := set.Values(), []string{"tests", "wiki"}; !sameStrings(got, want) {
		t.Errorf("tag set values = %v, want %v", got, want)
	}
}

// @scenario specs/libs/retriever-contract/behaviours/identifiers.feature: A tag set of sixteen distinct tags is accepted
func TestIdentifiersAcceptTagSetOfSixteenTags(t *testing.T) {
	var raw []string
	for i := 1; i <= 16; i++ {
		raw = append(raw, fmt.Sprintf("t%02d", i))
	}
	set, err := domain.NewTagSet(raw...)
	noError(t, err)
	if got := len(set.Values()); got != 16 {
		t.Errorf("tag set holds %d tags, want 16", got)
	}
}

// @scenario specs/libs/retriever-contract/behaviours/identifiers.feature: A tag set of seventeen distinct tags is refused
func TestIdentifiersRefuseTagSetOfSeventeenTags(t *testing.T) {
	var raw []string
	for i := 1; i <= 17; i++ {
		raw = append(raw, fmt.Sprintf("t%02d", i))
	}
	set, err := domain.NewTagSet(raw...)
	wantSentinel(t, err, domain.ErrTooManyTags)
	if got := len(set.Values()); got != 0 {
		t.Errorf("returned a tag set of %d tags, want none", got)
	}
}

// @scenario specs/libs/retriever-contract/behaviours/identifiers.feature: A tag set contains another only when it holds every tag of that set
func TestIdentifiersTagSetContainsOnlyWhenItHoldsEveryTag(t *testing.T) {
	set := tagsOf(t, "wiki", "tests")
	cases := []struct {
		wanted []string
		want   bool
	}{
		{[]string{"wiki"}, true},
		{[]string{"wiki", "tests"}, true},
		{[]string{"wiki", "kiwi"}, false},
		{[]string{"jira"}, false},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprint(tc.wanted), func(t *testing.T) {
			if got := set.ContainsAll(tagsOf(t, tc.wanted...)); got != tc.want {
				t.Errorf("contains %v = %v, want %v", tc.wanted, got, tc.want)
			}
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/identifiers.feature: An empty tag set is contained by every tag set
func TestIdentifiersEmptyTagSetIsContainedByEverySet(t *testing.T) {
	set := tagsOf(t, "wiki", "tests")
	if got := set.ContainsAll(tagsOf(t)); !got {
		t.Errorf("contains an empty tag set = %v, want true", got)
	}
}

// @scenario specs/libs/retriever-contract/behaviours/identifiers.feature: A canonical UUID with hyphens is accepted in either letter case
func TestIdentifiersAcceptCanonicalUUIDInEitherLetterCase(t *testing.T) {
	for _, raw := range []string{
		"550e8400-e29b-41d4-a716-446655440000",
		"550E8400-E29B-41D4-A716-446655440000",
		"550e8400-E29B-41d4-a716-446655440000",
	} {
		t.Run(raw, func(t *testing.T) {
			noError(t, domain.ValidateUUID(raw))
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/identifiers.feature: Text that is not a canonical UUID is refused as an invalid UUID
func TestIdentifiersRefuseTextThatIsNotCanonicalUUID(t *testing.T) {
	for _, raw := range []string{
		"550e8400e29b41d4a716446655440000",
		"550e8400-e29b-41d4-a716-44665544000",
		"urlslug",
		"",
	} {
		t.Run(raw, func(t *testing.T) {
			wantSentinel(t, domain.ValidateUUID(raw), domain.ErrInvalidUUID)
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/identifiers.feature: A share ID that is a UUID or a lowercase slug is accepted
func TestIdentifiersAcceptShareIDThatIsUUIDOrLowercaseSlug(t *testing.T) {
	for _, raw := range []string{"550e8400-e29b-41d4-a716-446655440000", "public-wiki-2", "a"} {
		t.Run(raw, func(t *testing.T) {
			noError(t, domain.ValidateShareID(raw))
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/identifiers.feature: A share ID that is neither a UUID nor a lowercase slug is refused
func TestIdentifiersRefuseShareIDThatIsNeitherUUIDNorSlug(t *testing.T) {
	for _, raw := range []string{"Share_1", "share id", "SHARE", ""} {
		t.Run(raw, func(t *testing.T) {
			wantSentinel(t, domain.ValidateShareID(raw), domain.ErrInvalidShareID)
		})
	}
}
