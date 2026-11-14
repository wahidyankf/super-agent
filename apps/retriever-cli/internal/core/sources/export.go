package sources

import (
	"context"
	"fmt"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/ports"
)

// ExportDocument fetches one document of the selected source of the kind and hands its Markdown body to sink. The sink
// receives a file name made from the document ID, such as "doc-1.md". No title, and no other file, decides the name, so
// a second export of the same document replaces the first, and two documents never share a file. The sink decides where
// the file goes. It returns the location the sink reports. A refused ID or source, or a failed fetch, makes no write.
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
	location, err := sink.Write(ctx, id.String()+".md", []byte(doc.Markdown))
	if err != nil {
		return "", fmt.Errorf("%w: %w", domain.ErrExportWriteFailed, err)
	}
	return location, nil
}
