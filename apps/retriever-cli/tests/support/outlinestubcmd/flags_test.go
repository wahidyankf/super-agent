package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/tests/support/outlinestub"
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

func TestRunRefusesToStartWithoutACAFile(t *testing.T) {
	err := run(context.Background(), []string{}, &bytes.Buffer{})

	if !errors.Is(err, errNoCAFile) {
		t.Fatalf("run without -ca-file returned %v, want errNoCAFile", err)
	}
}

func TestCollectionFlagReadsIDAndNameInListingOrder(t *testing.T) {
	var flags collectionFlag

	for _, value := range []string{"c1=Engineering", "c2=R&D = Drafts"} {
		if err := flags.Set(value); err != nil {
			t.Fatalf("Set(%q): %v", value, err)
		}
	}

	want := []outlinestub.Collection{{ID: "c1", Name: "Engineering"}, {ID: "c2", Name: "R&D = Drafts"}}
	if len(flags) != len(want) || flags[0] != want[0] || flags[1] != want[1] {
		t.Fatalf("collections = %v, want %v", flags, want)
	}
}

func TestCollectionFlagRefusesAValueWithNoIDOrName(t *testing.T) {
	cases := []struct {
		name  string
		value string
		want  string
	}{
		{name: "no separator", value: "no-equals-sign", want: `-collection wants id=name, got "no-equals-sign"`},
		{name: "empty id", value: "=Engineering", want: `-collection wants id=name, got "=Engineering"`},
		{name: "empty name", value: "c1=", want: `-collection wants id=name, got "c1="`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var flags collectionFlag
			err := flags.Set(tc.value)
			if err == nil || err.Error() != tc.want {
				t.Fatalf("Set(%q) returned %v, want %q", tc.value, err, tc.want)
			}
			if len(flags) != 0 {
				t.Fatalf("Set(%q) stored %v after refusing it", tc.value, flags)
			}
		})
	}
}
