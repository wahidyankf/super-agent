package operations

import (
	"fmt"
	"slices"
	"strings"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// Resource returns the resource word of the row: the word after outline in its command path, such as documents for
// "outline documents get". It is the word a user names in the --tools flag of retriever mcp", tool count and context cost).
func (r Row) Resource() string {
	path := r.Path()
	if len(path) < 2 {
		return ""
	}
	return path[1]
}

// Resources returns the distinct resource words of the Outline rows, sorted. Each word is read from a row, never written by
// hand, so a new group of Outline rows adds its word here. A resource with no tool is still listed. The Kiwi rows are left out, so a Kiwi word never joins the --tools flag of retriever mcp.
func Resources() []string {
	seen := map[string]bool{}
	var out []string
	for _, row := range rows {
		if k, ok := row.Kind(); !ok || k != domain.KindOutline {
			continue
		}
		if word := row.Resource(); word != "" && !seen[word] {
			seen[word] = true
			out = append(out, word)
		}
	}
	slices.Sort(out)
	return out
}

// ParseResources reads the value of the --tools flag: a comma-separated list of resource words, each one a word of
// Resources. An empty value, an empty entry, and a word the catalog does not hold are errors, and each error lists the
// valid words. A repeated word is kept once, in the order it first appears. Matching is exact: case and spaces count.
func ParseResources(raw string) ([]string, error) {
	valid := Resources()
	want := "want a comma-separated list of: " + strings.Join(valid, ", ")
	if raw == "" {
		return nil, fmt.Errorf("the value is empty (%s)", want)
	}
	var out []string
	for _, word := range strings.Split(raw, ",") {
		if word == "" {
			return nil, fmt.Errorf("an entry is empty (%s)", want)
		}
		if !slices.Contains(valid, word) {
			return nil, fmt.Errorf("unknown resource %q (%s)", word, want)
		}
		if !slices.Contains(out, word) {
			out = append(out, word)
		}
	}
	return out, nil
}
