package sources

import "testing"

func TestSlugKeepsLowercaseASCIIAndJoinsOtherRunsWithOneHyphen(t *testing.T) {
	tests := []struct {
		name  string
		title string
		want  string
	}{
		{name: "two words are joined with a hyphen", title: "Getting Started", want: "getting-started"},
		{name: "runs of spaces and punctuation become one hyphen and the ends are trimmed", title: "  Hello,   World!! ", want: "hello-world"},
		{name: "digits and dots are kept or joined", title: "API v2.0 (Draft)", want: "api-v2-0-draft"},
		{name: "an already slugged title is unchanged", title: "already-slugged-1", want: "already-slugged-1"},
		{name: "leading and trailing separators are trimmed", title: "-leading and trailing-", want: "leading-and-trailing"},
		{name: "non-ASCII letters act as separators", title: "Café Menu", want: "caf-menu"},
		{name: "a title with no letters or digits gives an empty slug", title: "!!!", want: ""},
		{name: "an empty title gives an empty slug", title: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := slug(tt.title); got != tt.want {
				t.Fatalf("want %q, got %q", tt.want, got)
			}
		})
	}
}
