package domain

import (
	"regexp"
	"slices"
	"strings"
)

// tagPattern is the shape of one tag: lowercase slugs with no hierarchy and no key:value form.
var tagPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// maxTagLength is the longest tag in bytes.
const maxTagLength = 32

// maxTags is the most distinct tags one TagSet holds.
const maxTags = 16

// Tag is a free label on a source. A tag is metadata for filtering and never grants or denies anything.
type Tag struct {
	value string
}

// NewTag validates raw. Uppercase letters are rejected, not lowered.
func NewTag(raw string) (Tag, error) {
	if len(raw) < 1 || len(raw) > maxTagLength || !tagPattern.MatchString(raw) {
		return Tag{}, ErrInvalidTag
	}
	return Tag{value: raw}, nil
}

// TagSet is an immutable, sorted set of distinct tags with at most 16 members. The zero value is the empty set.
type TagSet struct {
	tags []Tag
}

// NewTagSet validates each raw tag, drops duplicates, and sorts the rest. A duplicate is not an error.
// More than 16 distinct tags returns ErrTooManyTags.
func NewTagSet(raw ...string) (TagSet, error) {
	seen := map[string]bool{}
	var tags []Tag
	for _, text := range raw {
		tag, err := NewTag(text)
		if err != nil {
			return TagSet{}, err
		}
		if seen[tag.value] {
			continue
		}
		seen[tag.value] = true
		tags = append(tags, tag)
	}
	if len(tags) > maxTags {
		return TagSet{}, ErrTooManyTags
	}
	slices.SortFunc(tags, func(a, b Tag) int { return strings.Compare(a.value, b.value) })
	return TagSet{tags: tags}, nil
}

// Values returns the tags as sorted strings. The caller may change the returned slice.
func (s TagSet) Values() []string {
	out := make([]string, 0, len(s.tags))
	for _, tag := range s.tags {
		out = append(out, tag.value)
	}
	return out
}

// ContainsAll reports whether every tag of other is in s. An empty other is always contained, so it is the
// "no tag filter" case.
func (s TagSet) ContainsAll(other TagSet) bool {
	for _, want := range other.tags {
		if !slices.ContainsFunc(s.tags, func(have Tag) bool { return have.value == want.value }) {
			return false
		}
	}
	return true
}
