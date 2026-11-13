package mcp

import "testing"

// TestTagsOrEmptyShowsAnUntaggedSourceAsAnEmptyList checks that a source with no tags reads as an empty list, and that a
// source with tags keeps them.
func TestTagsOrEmptyShowsAnUntaggedSourceAsAnEmptyList(t *testing.T) {
	if got := tagsOrEmpty(nil); got == nil || len(got) != 0 {
		t.Errorf("no tags = %#v, want an empty list", got)
	}
	if got := tagsOrEmpty([]string{"wiki"}); len(got) != 1 || got[0] != "wiki" {
		t.Errorf("tags = %#v, want them kept", got)
	}
}
