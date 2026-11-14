package cli

import (
	"context"
	"encoding/json"
	"io/fs"
	"strings"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/configpaths"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// fixedNow is the clock the tests give the CLI.
var fixedNow = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

// stubFS is a file system that holds no files, so no test reads the disk.
type stubFS struct{}

func (stubFS) ReadFile(string) ([]byte, error) { return nil, fs.ErrNotExist }

func (stubFS) Exists(string) bool { return false }

func TestVersionPrintsNameVersionAndBanner(t *testing.T) {
	code, out, errOut := runRetriever(t, []string{"--version"}, outlineSetup{})
	if code != 0 || out != "retriever 0.1.0\nALPHA: expect breaking changes\n" || errOut != "" {
		t.Fatalf("--version = %q (exit %d, stderr %q)", out, code, errOut)
	}
}

func TestInvalidOutputFormatIsAUsageError(t *testing.T) {
	code, out, errOut := runRetriever(t, []string{"--output", "markdown", "sources", "list"}, outlineSetup{})
	if code != 2 || out != "" || !strings.Contains(errOut, "--output must be text or json") {
		t.Fatalf("want a usage error with exit 2 and empty standard output, got %q (exit %d, stdout %q)", errOut, code, out)
	}
}

func TestInvalidColorIsAUsageError(t *testing.T) {
	code, out, errOut := runRetriever(t, []string{"--color", "sometimes", "sources", "list"}, outlineSetup{})
	if code != 2 || out != "" || !strings.Contains(errOut, "--color must be auto, always, or never") {
		t.Fatalf("want a usage error with exit 2, got %q (exit %d, stdout %q)", errOut, code, out)
	}
}

func TestBareInvocationExitsTwoWithNoStandardOutput(t *testing.T) {
	code, out, _ := runRetriever(t, nil, outlineSetup{})
	if code != 2 || out != "" {
		t.Fatalf("a bare invocation must exit 2 with empty standard output, got exit %d, stdout %q", code, out)
	}
}

func TestUsageFailureBodyNamesTheAreaInItsCode(t *testing.T) {
	code, out, errOut := runRetriever(t, []string{"sources", "list", "--bogus", "--output", "json"}, outlineSetup{})
	var body struct {
		ExitCode int `json:"exitCode"`
		Error    struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if code != 2 || out != "" || json.Unmarshal([]byte(errOut), &body) != nil || body.ExitCode != 2 ||
		!strings.HasPrefix(body.Error.Code, "retriever.sources.usage_") {
		t.Fatalf("want a usage body with a retriever.sources.usage_ code and exit 2, got %q (exit %d)", errOut, code)
	}
}

func TestFailureBodyHasTheFixedFieldOrder(t *testing.T) {
	setup, _ := wikiWith(treeDataset)
	code, out, errOut := runRetriever(t, []string{"outline", "tree", "col-missing", "--output", "json"}, setup)
	want := `{"schemaVersion":1,"command":"retriever outline tree","exitCode":2,` +
		`"error":{"code":"retriever.outline.collection_not_found","message":"collection not found","retryable":false}}` + "\n"
	if code != 2 || out != "" || errOut != want {
		t.Fatalf("failure body = %q (exit %d, stdout %q), want %q", errOut, code, out, want)
	}
}

func TestHelpSubcommandPrintsTheSameAsTheHelpFlag(t *testing.T) {
	code1, byFlag, _ := runRetriever(t, []string{"sources", "list", "--help"}, outlineSetup{})
	code2, byWord, _ := runRetriever(t, []string{"help", "sources", "list"}, outlineSetup{})
	if code1 != 0 || code2 != 0 || byFlag == "" || byFlag != byWord {
		t.Fatalf("help flag %q (exit %d) and help subcommand %q (exit %d) must match", byFlag, code1, byWord, code2)
	}
}

func TestHelpListsTheExitStatuses(t *testing.T) {
	code, out, _ := runRetriever(t, []string{"--help"}, outlineSetup{})
	if code != 0 || !strings.Contains(out, "124") || !strings.Contains(out, "125") {
		t.Fatalf("the root help must list the exit statuses, got %q (exit %d)", out, code)
	}
}

func TestPanicIsReportedAsAnInternalFailureWithoutItsText(t *testing.T) {
	setup := outlineSetup{serve: func(context.Context, int, func(string)) error { panic("internal detail") }}
	code, out, errOut := runRetriever(t, []string{"serve", "--output", "json"}, setup)
	if code != 2 || out != "" || !strings.Contains(errOut, `"code":"retriever.panic"`) || strings.Contains(errOut, "internal detail") {
		t.Fatalf("want retriever.panic with exit 2 and no panic text, got %q (exit %d, stdout %q)", errOut, code, out)
	}
}

func TestTimedOutRequestExitsOneTwentyFourAndIsRetryable(t *testing.T) {
	setup, _ := wikiWith(treeDataset)
	setup.service = oneOutlineService(failingGateway{err: domain.ErrTimedOut})
	code, out, errOut := runRetriever(t, []string{"outline", "collections", "--output", "json"}, setup)
	if code != 124 || out != "" || !strings.Contains(errOut, `"retryable":true`) {
		t.Fatalf("want exit 124 and a retryable body, got %q (exit %d, stdout %q)", errOut, code, out)
	}
}

func TestRefusedSetupExitsOneTwentyFive(t *testing.T) {
	code, _, errOut := runRetriever(t, []string{"outline", "collections"}, outlineSetup{setupErr: configpaths.ErrRefused})
	if code != 125 || !strings.Contains(errOut, "retriever: ") {
		t.Fatalf("want exit 125 with a text message, got %q (exit %d)", errOut, code)
	}
}

func TestSourcesListJSONPrintsEverySource(t *testing.T) {
	code, out, errOut := runRetriever(t, []string{"sources", "list", "--output", "json"}, outlineSetup{service: twoSourceService(t)})
	var doc struct {
		SchemaVersion int `json:"schemaVersion"`
		Sources       []struct {
			ID string `json:"id"`
		} `json:"sources"`
	}
	if code != 0 || json.Unmarshal([]byte(out), &doc) != nil || doc.SchemaVersion != 2 || len(doc.Sources) != 2 {
		t.Fatalf("want two sources in a schemaVersion 2 document, got %q (exit %d, stderr %q)", out, code, errOut)
	}
}

func TestSourcesListRejectsAnUnknownKind(t *testing.T) {
	code, out, errOut := runRetriever(t, []string{"sources", "list", "--kind", "confluence"}, outlineSetup{service: twoSourceService(t)})
	if code != 2 || out != "" || !strings.Contains(errOut, "unknown source kind") {
		t.Fatalf("want an unknown-kind error with exit 2, got %q (exit %d, stdout %q)", errOut, code, out)
	}
}

func TestParseGlobalsReadsThePathFlagsInAnyPosition(t *testing.T) {
	g, err := ParseGlobals([]string{"sources", "--catalog", "/work/retriever/sources.yml", "list", "--env-file", "/work/retriever/.env"})
	if err != nil || g.Catalog != "/work/retriever/sources.yml" || g.EnvFile != "/work/retriever/.env" {
		t.Fatalf("ParseGlobals = %+v, %v; want both paths", g, err)
	}
}

func TestDocumentGetPrintsTheMarkdown(t *testing.T) {
	setup, _ := wikiWith(treeDataset)
	code, out, errOut := runRetriever(t, []string{"document", "get", "doc-1"}, setup)
	if code != 0 || !strings.Contains(out, "Welcome to the wiki.") {
		t.Fatalf("want the Markdown body, got %q (exit %d, stderr %q)", out, code, errOut)
	}
}

func TestDocumentGetJSONHasTheFixedFields(t *testing.T) {
	setup, _ := wikiWith(treeDataset)
	code, out, errOut := runRetriever(t, []string{"document", "get", "doc-1", "--output", "json"}, setup)
	var doc map[string]any
	if code != 0 || json.Unmarshal([]byte(out), &doc) != nil {
		t.Fatalf("want a JSON document, got %q (exit %d, stderr %q)", out, code, errOut)
	}
	for _, key := range []string{"schemaVersion", "id", "title", "url", "markdown", "collectionId"} {
		if _, ok := doc[key]; !ok {
			t.Fatalf("the document has no %q field: %q", key, out)
		}
	}
}

func TestExportJSONPrintsTheLocation(t *testing.T) {
	setup, _ := wikiWith(treeDataset)
	code, out, errOut := runRetriever(t, []string{"outline", "export", "doc-1", "--out", "notes/outline", "--output", "json"}, setup)
	var doc struct {
		SchemaVersion int    `json:"schemaVersion"`
		Location      string `json:"location"`
	}
	if code != 0 || json.Unmarshal([]byte(out), &doc) != nil || doc.Location != "notes/outline/doc-1.md" {
		t.Fatalf("want the location in a JSON document, got %q (exit %d, stderr %q)", out, code, errOut)
	}
}
