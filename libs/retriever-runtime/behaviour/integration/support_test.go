//go:build integration

package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/markdownfiles"
)

// This file holds the helpers that the integration bindings share. They work on real files, real folders, and real
// environment variables, and each one undoes its change when the test ends.

// writeFiles writes each named file under dir, creating the parent folders it needs.
func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("make the folder of %s: %v", name, err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
}

// wantError fails the test unless err is non-nil and its text is exactly message.
func wantError(t *testing.T, err error, message string) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want %q", message)
	}
	if err.Error() != message {
		t.Fatalf("error = %q, want %q", err.Error(), message)
	}
}

// unsetEnv makes the process variable name unset for the test. t.Setenv restores the old value when the test ends.
func unsetEnv(t *testing.T, name string) {
	t.Helper()
	t.Setenv(name, "")
	if err := os.Unsetenv(name); err != nil {
		t.Fatalf("unset %s: %v", name, err)
	}
}

// padToSize returns the YAML with a comment that fills it to exactly size bytes. The parser ignores a comment.
func padToSize(yaml string, size int) string {
	base := yaml + "# "
	return base + strings.Repeat("x", size-len(base))
}

// outlineDocsYAML is a valid catalog with one outline source named docs.
const outlineDocsYAML = `version: 1
sources:
  - id: docs
    kind: outline
    endpoint: https://docs.example.test
    credential_env: EXAMPLE_OUTLINE_API_KEY
`

// outlineSource returns an outline source with the ID given, served by the Outline API key.
func outlineSource(t *testing.T, id string) domain.Source {
	t.Helper()
	sid, err := domain.NewSourceID(id)
	if err != nil {
		t.Fatalf("NewSourceID: %v", err)
	}
	endpoint, err := domain.NewEndpoint("https://" + id + ".example.test")
	if err != nil {
		t.Fatalf("NewEndpoint: %v", err)
	}
	ref, err := domain.NewCredentialRef("EXAMPLE_OUTLINE_API_KEY")
	if err != nil {
		t.Fatalf("NewCredentialRef: %v", err)
	}
	set, err := domain.NewTagSet()
	if err != nil {
		t.Fatalf("NewTagSet: %v", err)
	}
	return domain.NewSource(sid, domain.KindOutline, set, endpoint, ref)
}

// docsGateway is the outline gateway of the integration bindings. It answers one fixed document, so the export has a
// body to write, and it serves no other call.
type docsGateway struct {
	markdown string
}

var _ ports.SourceGateway = docsGateway{}

// ListCollections returns no collections.
func (docsGateway) ListCollections(context.Context) ([]domain.Collection, error) { return nil, nil }

// CollectionTree returns no nodes.
func (docsGateway) CollectionTree(context.Context, domain.CollectionID) ([]domain.TreeNode, error) {
	return nil, nil
}

// Document returns the fixed document, with the ID asked for.
func (g docsGateway) Document(_ context.Context, id domain.DocumentID) (domain.SourceDocument, error) {
	return domain.SourceDocument{ID: id.String(), Title: "Notes", Markdown: g.markdown}, nil
}

// realCreateTemp is the temporary-file call of the export sink over the real disk.
func realCreateTemp(dir, pattern string) (markdownfiles.TempFile, error) {
	f, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return nil, err
	}
	return f, nil
}

// realSink returns the export sink over the real folder dir.
func realSink(dir string) *markdownfiles.Sink {
	return markdownfiles.New(dir, realCreateTemp, markdownfiles.Folder{
		MkdirAll: os.MkdirAll,
		Rename:   os.Rename,
		Remove:   os.Remove,
	})
}
