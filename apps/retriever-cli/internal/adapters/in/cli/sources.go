package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/sources"
)

// sourcesList prints the configured sources that match --kind and --tag. The tag filter is one comma-separated
// value, and every tag given must be on a source. Each row shows the id, kind, and tags, and never an
// endpoint or a credential name. JSON mode prints the same fields as one document.
func sourcesList(inv *invocation) error {
	rt := inv.rt
	if rt.setupErr != nil {
		return setupError{err: rt.setupErr}
	}
	filter, err := sourceFilter(inv)
	if err != nil {
		return err
	}
	rows := sources.ListSources(rt.service.Catalog, filter)
	if rt.format == domain.FormatJSON {
		return writeJSON(rt.out, sourcesJSON{SchemaVersion: sourcesSchemaVersion, Sources: sourceRowsJSON(rows)})
	}
	writeSourceRows(rt.out, rows)
	return nil
}

// sourcesSchemaVersion is the version of the sources list document. The access field left it in version 2, because a
// removed field raises the version of its document. The other documents keep schemaVersion.
const sourcesSchemaVersion = 2

// sourcesJSON is the document of sources list in JSON mode. Its field order is fixed.
type sourcesJSON struct {
	SchemaVersion int             `json:"schemaVersion"`
	Sources       []sourceRowJSON `json:"sources"`
}

// sourceRowJSON is one source of the sources list in JSON mode. Tags is never null, so an untagged source reads as [].
type sourceRowJSON struct {
	ID   string   `json:"id"`
	Kind string   `json:"kind"`
	Tags []string `json:"tags"`
}

// sourceRowsJSON returns the JSON rows of the sources, in the order the catalog lists them.
func sourceRowsJSON(rows []domain.SourceSummary) []sourceRowJSON {
	out := make([]sourceRowJSON, 0, len(rows))
	for _, r := range rows {
		tags := r.Tags
		if tags == nil {
			tags = []string{}
		}
		out = append(out, sourceRowJSON{ID: r.ID, Kind: r.Kind, Tags: tags})
	}
	return out
}

// sourceFilter reads --kind and --tag into a domain filter. The tag flag is one comma-separated value. A bad kind or
// tag is returned as it is, and the reporter maps it to its user text and its code.
func sourceFilter(inv *invocation) (domain.SourceFilter, error) {
	var kind *string
	if raw, ok := inv.values["--kind"]; ok {
		kind = &raw
	}
	var tags []string
	if text, ok := inv.values["--tag"]; ok {
		tags = strings.Split(text, ",")
	}
	return domain.NewSourceFilter(kind, tags...)
}

// writeSourceRows prints the count line, then one row per source. The first three columns are padded to their
// widest value and separated by two spaces. Tags come last, comma-joined, or "-" when there are none.
func writeSourceRows(w io.Writer, rows []domain.SourceSummary) {
	switch len(rows) {
	case 1:
		fmt.Fprintln(w, "1 source")
	default:
		fmt.Fprintf(w, "%d sources\n", len(rows))
	}
	idW, kindW := 0, 0
	for _, r := range rows {
		idW = max(idW, len(r.ID))
		kindW = max(kindW, len(r.Kind))
	}
	for _, r := range rows {
		tags := strings.Join(r.Tags, ",")
		if tags == "" {
			tags = "-"
		}
		fmt.Fprintf(w, "  %-*s  %-*s  %s\n", idW, r.ID, kindW, r.Kind, tags)
	}
}
