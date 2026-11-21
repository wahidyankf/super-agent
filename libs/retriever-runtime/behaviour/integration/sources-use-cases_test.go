//go:build integration

package integration_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/registry"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/sources"
)

// exportService returns a service over the one outline source docs, whose gateway answers the document with markdown.
func exportService(t *testing.T, markdown string) sources.Service {
	t.Helper()
	cat, err := domain.NewCatalog([]domain.Source{outlineSource(t, "docs")})
	if err != nil {
		t.Fatalf("NewCatalog: %v", err)
	}
	return sources.Service{
		Catalog: cat,
		Gateways: registry.New(map[domain.SourceKind]registry.Builder{
			domain.KindOutline: func(domain.Source) (ports.SourceGateway, error) {
				return docsGateway{markdown: markdown}, nil
			},
		}),
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/sources-use-cases.feature: An export writes the Markdown under a file name made from the ID
func TestAnExportWritesTheMarkdownUnderAFileNameMadeFromTheID(t *testing.T) {
	cases := []string{"doc-1", "doc-2"}
	for _, id := range cases {
		t.Run(id, func(t *testing.T) {
			dir := t.TempDir()
			location, err := sources.ExportDocument(context.Background(), exportService(t, "# Notes"), realSink(dir),
				domain.KindOutline, nil, id)
			if err != nil {
				t.Fatalf("ExportDocument error = %v", err)
			}
			if want := filepath.Join(dir, id+".md"); location != want {
				t.Fatalf("location = %q, want %q", location, want)
			}
			data, err := os.ReadFile(filepath.Join(dir, id+".md"))
			if err != nil || string(data) != "# Notes" {
				t.Fatalf("file %s.md holds %q, error %v; want the body", id, data, err)
			}
		})
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/sources-use-cases.feature: An export whose file write fails returns the write error
func TestAnExportWhoseFileWriteFailsReturnsTheWriteError(t *testing.T) {
	// A regular file where the export folder should be makes the sink's folder step fail on the real disk.
	blocker := filepath.Join(t.TempDir(), "blocker")
	writeFiles(t, filepath.Dir(blocker), map[string]string{"blocker": "not a folder"})
	_, err := sources.ExportDocument(context.Background(), exportService(t, "# Notes"),
		realSink(filepath.Join(blocker, "exports")), domain.KindOutline, nil, "doc-1")
	if !errors.Is(err, domain.ErrExportWriteFailed) {
		t.Fatalf("error = %v, want the export write error", err)
	}
	if !strings.HasPrefix(err.Error(), "could not write the exported document") {
		t.Fatalf("error text = %q, want the export write text", err.Error())
	}
}
