package cli

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/tests/support/sourcecontract"
)

// collectionsOf returns a dataset with one collection per name, IDed col-1, col-2, and so on, and no documents.
func collectionsOf(names ...string) sourcecontract.Dataset {
	var cols []sourcecontract.Collection
	for i, name := range names {
		cols = append(cols, sourcecontract.Collection{ID: fmt.Sprintf("col-%d", i+1), Name: name})
	}
	return sourcecontract.Dataset{Collections: cols}
}

func TestOutlineCollectionsPrintsTheCountAndEveryCollection(t *testing.T) {
	setup, _ := wikiWith(collectionsOf("Engineering", "Sales", "People"))

	code, out, errOut := runRetriever(t, []string{"outline", "collections"}, setup)

	if code != 0 {
		t.Fatalf("want exit 0, got %d (stderr %q)", code, errOut)
	}
	for _, want := range []string{"3 collections", "col-1", "Engineering", "col-2", "Sales", "col-3", "People"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output must contain %q, got %q", want, out)
		}
	}
}

func TestOutlineCollectionsUsesTheSingularForOne(t *testing.T) {
	setup, _ := wikiWith(collectionsOf("Engineering"))

	code, out, _ := runRetriever(t, []string{"outline", "collections"}, setup)

	if code != 0 || !strings.Contains(out, "1 collection\n") || strings.Contains(out, "1 collections") {
		t.Fatalf("want the singular count line and exit 0, got %q (exit %d)", out, code)
	}
}

func TestOutlineCollectionsOfAnEmptyWikiPrintsZero(t *testing.T) {
	setup, _ := wikiWith(sourcecontract.Dataset{})

	code, out, _ := runRetriever(t, []string{"outline", "collections"}, setup)

	if code != 0 || !strings.Contains(out, "0 collections") {
		t.Fatalf("want the zero count and exit 0, got %q (exit %d)", out, code)
	}
}

func TestOutlineTreeIndentsEachLevelByItsDepth(t *testing.T) {
	setup, _ := wikiWith(treeDataset)

	code, out, errOut := runRetriever(t, []string{"outline", "tree", "col-1"}, setup)

	if code != 0 {
		t.Fatalf("want exit 0, got %d (stderr %q)", code, errOut)
	}
	root, child, grandchild := indentOf(out, "Getting Started"), indentOf(out, "Setup"), indentOf(out, "Install")
	if root < 0 || child <= root || grandchild <= child {
		t.Fatalf("each level must be indented deeper than its parent: root %d, child %d, grandchild %d\n%s",
			root, child, grandchild, out)
	}
}

func TestOutlineTreeShowsEachDocumentIDAndURL(t *testing.T) {
	setup, _ := wikiWith(treeDataset)

	_, out, _ := runRetriever(t, []string{"outline", "tree", "col-1"}, setup)

	for _, want := range []string{"doc-1", "/doc/getting-started", "doc-2", "/doc/setup"} {
		if !strings.Contains(out, want) {
			t.Fatalf("tree output must contain %q, got %q", want, out)
		}
	}
}

func TestOutlineTreeOfACollectionWithNoDocumentsSaysSo(t *testing.T) {
	setup, _ := wikiWith(treeDataset)

	code, out, errOut := runRetriever(t, []string{"outline", "tree", "col-empty"}, setup)

	if code != 0 || !strings.Contains(out, "No documents") {
		t.Fatalf("want No documents and exit 0, got %q (exit %d, stderr %q)", out, code, errOut)
	}
}

func TestOutlineTreeRejectsAnInvalidCollectionIDBeforeAnyRequest(t *testing.T) {
	setup, gw := wikiWith(treeDataset)

	code, _, errOut := runRetriever(t, []string{"outline", "tree", "col.1"}, setup)

	if code != 2 || !strings.Contains(errOut, "invalid collection ID") || gw.calls != 0 {
		t.Fatalf("want an invalid-ID error, exit 2, and no request; got %q (exit %d, calls %d)", errOut, code, gw.calls)
	}
}

func TestOutlineTreeReportsAnUnknownCollection(t *testing.T) {
	setup, _ := wikiWith(treeDataset)

	code, _, errOut := runRetriever(t, []string{"outline", "tree", "col-missing"}, setup)

	if code != 2 || !strings.Contains(errOut, "collection not found") {
		t.Fatalf("want a collection-not-found error, got %q (exit %d)", errOut, code)
	}
}

func TestOutlineExportWritesTheFileAndPrintsItsLocation(t *testing.T) {
	setup, _ := wikiWith(treeDataset)

	code, out, errOut := runRetriever(t, []string{"outline", "export", "doc-1", "--out", "notes/outline"}, setup)

	if code != 0 || out != "Wrote notes/outline/doc-1.md\n" {
		t.Fatalf("want the written location and exit 0, got %q (exit %d, stderr %q)", out, code, errOut)
	}
	written := string(setup.sink.bodies["doc-1.md"])
	if !strings.Contains(written, "Welcome to the wiki.") {
		t.Fatalf("the file must hold the Markdown, got %q", written)
	}
}

func TestOutlineExportRequiresTheOutOption(t *testing.T) {
	setup, gw := wikiWith(treeDataset)

	code, _, errOut := runRetriever(t, []string{"outline", "export", "doc-1"}, setup)

	if code != 2 || !strings.Contains(errOut, "--out") || gw.calls != 0 {
		t.Fatalf("want a missing --out error before any request, got %q (exit %d, calls %d)", errOut, code, gw.calls)
	}
}

func TestOutlineExportRejectsAnInvalidDocumentIDBeforeAnyRequest(t *testing.T) {
	setup, gw := wikiWith(treeDataset)

	code, _, errOut := runRetriever(t, []string{"outline", "export", "doc.1", "--out", "notes/outline"}, setup)

	if code != 2 || !strings.Contains(errOut, "invalid document ID") || gw.calls != 0 || len(setup.sink.names) != 0 {
		t.Fatalf("want an invalid-ID error with no request and no write, got %q (exit %d)", errOut, code)
	}
}

func TestOutlineExportReportsAnUnknownDocument(t *testing.T) {
	setup, _ := wikiWith(treeDataset)

	code, _, errOut := runRetriever(t, []string{"outline", "export", "doc-missing", "--out", "notes/outline"}, setup)

	if code != 2 || !strings.Contains(errOut, "document not found") || len(setup.sink.names) != 0 {
		t.Fatalf("want a document-not-found error and no write, got %q (exit %d)", errOut, code)
	}
}

func TestOutlineExportWriteFailureNeverPrintsTheRawError(t *testing.T) {
	setup, _ := wikiWith(treeDataset)
	setup.sink.err = fmt.Errorf("%w: %w", domain.ErrExportWriteFailed,
		errors.New("open /private/store/getting-started.md: permission denied"))

	code, _, errOut := runRetriever(t, []string{"outline", "export", "doc-1", "--out", "notes/outline"}, setup)

	if code != 2 || !strings.Contains(errOut, "could not write the exported document") {
		t.Fatalf("want the could-not-write message, got %q (exit %d)", errOut, code)
	}
	if strings.Contains(errOut, "permission denied") || strings.Contains(errOut, "/private/store") {
		t.Fatalf("the raw write error must never be printed, got %q", errOut)
	}
}

func TestOutlineFailureMessagesAreTheFixedTexts(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"missing key", domain.MissingKeyError{Variable: "EXAMPLE_OUTLINE_API_KEY"}, "EXAMPLE_OUTLINE_API_KEY is not set"},
		{"wrapped missing key", fmt.Errorf("outline credentials: %w", domain.MissingKeyError{Variable: "EXAMPLE_OUTLINE_API_KEY"}), "EXAMPLE_OUTLINE_API_KEY is not set"},
		{"key rejected", domain.ErrKeyRejected, "Outline rejected the API key"},
		{"access denied", domain.ErrAccessDenied, "Outline denied access"},
		{"rate limited", domain.ErrSourceRateLimited, "rate limited"},
		{"retry after too long", domain.ErrRetryAfterTooLong, "longer than 60 seconds"},
		{"redirected", domain.ErrRedirected, "redirect"},
		{"timed out", domain.ErrTimedOut, "timed out"},
		{"https required", domain.ErrHTTPSRequired, "HTTPS is required"},
		{"unreachable", domain.ErrUnreachable, "Outline could not be reached"},
		{"upstream failure", domain.ErrUpstreamFailure, "Outline returned an error"},
		{"unexpected response", domain.ErrUnexpectedResponse, "unexpected response from Outline"},
		{"page cap", domain.ErrPageCapReached, "page cap of 20"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setup, _ := wikiWith(treeDataset)
			setup.service = oneOutlineService(failingGateway{err: tc.err})

			code, _, errOut := runRetriever(t, []string{"outline", "collections"}, setup)

			wantCode := 2
			if tc.err == domain.ErrTimedOut || tc.err == domain.ErrSourceRateLimited {
				wantCode = 124
			}
			if code != wantCode || !strings.Contains(errOut, tc.want) {
				t.Fatalf("want %q with exit %d, got %q (exit %d)", tc.want, wantCode, errOut, code)
			}
		})
	}
}

func TestOutlineUnexpectedErrorPrintsAGenericMessage(t *testing.T) {
	setup, _ := wikiWith(treeDataset)
	setup.service = oneOutlineService(failingGateway{err: errors.New("upstream echoed sentinel-key-7f3a9c")})

	code, out, errOut := runRetriever(t, []string{"outline", "collections"}, setup)

	if code != 2 || !strings.Contains(errOut, "unexpected error") {
		t.Fatalf("want the generic message with exit 2, got %q (exit %d)", errOut, code)
	}
	if strings.Contains(out+errOut, "sentinel-key-7f3a9c") {
		t.Fatal("an unexpected error must never print its raw text")
	}
}

func TestOutlineCommandsReportTheSetupErrorWhenWiringIsMissing(t *testing.T) {
	setup := outlineSetup{setupErr: errors.New("no repository holds the working directory")}
	argvs := [][]string{
		{"outline", "collections"},
		{"outline", "tree", "col-1"},
		{"outline", "export", "doc-1", "--out", "notes/outline"},
	}
	for _, argv := range argvs {
		code, _, errOut := runRetriever(t, argv, setup)
		if code != 2 || !strings.Contains(errOut, "no repository holds the working directory") {
			t.Fatalf("%v: want the setup error with exit 2, got %q (exit %d)", argv, errOut, code)
		}
	}
}

func TestOutlineUsageErrorsExitTwoInEveryGroup(t *testing.T) {
	code, _, errOut := runRetriever(t, []string{"outline", "documents", "get", "doc-1", "--bogus"}, outlineSetup{})
	if code != 2 || !strings.Contains(errOut, "--bogus") {
		t.Fatalf("want a usage error with exit 2, got %q (exit %d)", errOut, code)
	}
	code, _, errOut = runRetriever(t, []string{"outline", "tree", "col-1", "--bogus"}, outlineSetup{})
	if code != 2 || !strings.Contains(errOut, "--bogus") {
		t.Fatalf("want a usage error with exit 2 for outline tree, got %q (exit %d)", errOut, code)
	}
	code, _, errOut = runRetriever(t, []string{"sources", "list", "--bogus"}, outlineSetup{})
	if code != 2 || !strings.Contains(errOut, "--bogus") {
		t.Fatalf("want a usage error with exit 2 outside outline, got %q (exit %d)", errOut, code)
	}
}

func TestOutlineHelpListsItsCommands(t *testing.T) {
	code, out, _ := runRetriever(t, []string{"outline", "--help"}, outlineSetup{})

	if code != 0 || !strings.Contains(out, "collections") || !strings.Contains(out, "tree") || !strings.Contains(out, "export") {
		t.Fatalf("outline help must list its commands, got %q (exit %d)", out, code)
	}
}

func TestTopLevelHelpListsOutlineAndServe(t *testing.T) {
	code, out, _ := runRetriever(t, []string{"--help"}, outlineSetup{})

	if code != 0 || !strings.Contains(out, "outline") || !strings.Contains(out, "serve") {
		t.Fatalf("top-level help must list outline and serve, got %q (exit %d)", out, code)
	}
}
