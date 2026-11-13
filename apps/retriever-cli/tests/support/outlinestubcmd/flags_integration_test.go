//go:build integration

package main

import (
	"encoding/json"
	"errors"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/tests/support/outlinestub"
	"os"
	"path/filepath"
	"testing"
)

func TestReadDocumentsReadsTheDocumentsFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "documents.json")
	content := `[{"ID":"doc-2","CollectionID":"c1","ParentID":"doc-1","Title":"Setup","URL":"/doc/doc-2","Markdown":"Run the installer."}]`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write documents file: %v", err)
	}

	docs, err := readDocuments(path)
	if err != nil {
		t.Fatalf("readDocuments: %v", err)
	}

	want := outlinestub.Document{ID: "doc-2", CollectionID: "c1", ParentID: "doc-1", Title: "Setup", URL: "/doc/doc-2", Markdown: "Run the installer."}
	if len(docs) != 1 || docs[0] != want {
		t.Fatalf("documents = %v, want [%v]", docs, want)
	}
}

func TestReadDocumentsRefusesAFileThatIsNotJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "documents.json")
	if err := os.WriteFile(path, []byte("not json"), 0o600); err != nil {
		t.Fatalf("write documents file: %v", err)
	}

	_, err := readDocuments(path)
	if err == nil {
		t.Fatal("readDocuments returned no error for a file that is not JSON")
	}
	var syntaxErr *json.SyntaxError
	if !errors.As(err, &syntaxErr) {
		t.Fatalf("readDocuments returned %v, want a *json.SyntaxError from the parse", err)
	}
}
