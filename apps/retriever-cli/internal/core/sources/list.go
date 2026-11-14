package sources

import "github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"

// ListSources returns the summaries of the sources that match filter, sorted by ID. It does no I/O and calls no
// gateway. A summary never carries the endpoint or the credential name.
func ListSources(catalog domain.Catalog, filter domain.SourceFilter) []domain.SourceSummary {
	var out []domain.SourceSummary
	for _, src := range catalog.Find(filter) {
		out = append(out, src.Summary())
	}
	return out
}
