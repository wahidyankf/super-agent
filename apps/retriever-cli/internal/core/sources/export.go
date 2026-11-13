package sources

import (
	"context"
	"fmt"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/ports"
)

// ExportDocument fetches one document of the selected source of the kind and hands its Markdown body to sink. The sink
// receives a file name made from the document's title, such as "getting-started.md", and the sink decides where the file
// goes. It returns the location the sink reports. A refused ID or source, or a failed fetch, makes no write.
func ExportDocument(
	ctx context.Context,
	svc Service,
	sink ports.ExportSink,
	kind domain.SourceKind,
	source *string,
	documentID string,
) (string, error) {
	src, err := selectSource(svc, kind, source)
	if err != nil {
		return "", err
	}
	id, err := domain.NewDocumentID(documentID)
	if err != nil {
		return "", err
	}
	gw, err := svc.Gateways.For(src)
	if err != nil {
		return "", err
	}
	doc, err := gw.Document(ctx, id)
	if err != nil {
		return "", err
	}
	location, err := sink.Write(ctx, fileStem(kind, doc.Title, id)+".md", []byte(doc.Markdown))
	if err != nil {
		return "", fmt.Errorf("%w: %w", domain.ErrExportWriteFailed, err)
	}
	return location, nil
}

// fileStem names an export. Outline takes the title slug, or the ID when the title has no slug. Kiwi and Jira put the ID
// first and then the slug, such as case-12-login-works or issue-EXAMPLE-12-login-fails, because many documents share a title;
// a title with no slug is the ID alone.
func fileStem(kind domain.SourceKind, title string, id domain.DocumentID) string {
	s := slug(title)
	if kind != domain.KindKiwi && kind != domain.KindJira {
		if s != "" {
			return s
		}
		return id.String()
	}
	if s == "" {
		return id.String()
	}
	return id.String() + "-" + s
}
