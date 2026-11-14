package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/connectorspec"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/operations"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/sources"
)

// wikiRowsReader is a reader of the spec kind of these tests. It returns one document for every read.
type wikiRowsReader struct{}

// Read returns one list item with an ID and a title.
func (wikiRowsReader) Read(context.Context, domain.OperationID, domain.Args) (domain.Record, error) {
	return domain.NewList([]domain.Record{domain.NewRecord(
		domain.Field{Name: "id", Value: "doc-1"},
		domain.Field{Name: "title", Value: "First document"},
	)}), nil
}

// specCLIRow is the CLI row of the example-wiki list operation, as the connector spec derives it.
func specCLIRow() operations.Row {
	return operations.Row{
		ID:      "example-wiki.docs.list",
		CLI:     "example-wiki docs list",
		Summary: "List documents.",
		Effect:  domain.EffectRead,
		ListKey: "items",
		Noun:    "document",
		Fields:  []string{"id", "title"},
		Columns: []string{"id", "title"},
	}
}

// specService returns a service whose composed operation catalog holds the spec row, and whose one source is of the spec
// kind. A reader serves the row.
func specService(t *testing.T, row operations.Row) sources.Service {
	t.Helper()
	kinds, err := domain.NewKindSet(domain.KindDefinition{Name: "example-wiki"})
	if err != nil {
		t.Fatalf("NewKindSet: %v", err)
	}
	kind, err := kinds.Kind("example-wiki")
	if err != nil {
		t.Fatalf("Kind: %v", err)
	}
	cat, err := operations.NewCatalog(kinds, []operations.Row{row})
	if err != nil {
		t.Fatalf("NewCatalog: %v", err)
	}
	id, err := domain.NewSourceID("wiki")
	if err != nil {
		t.Fatalf("NewSourceID: %v", err)
	}
	ep, err := domain.NewEndpoint("https://wiki.example.test")
	if err != nil {
		t.Fatalf("NewEndpoint: %v", err)
	}
	ref, err := domain.NewCredentialRef("EXAMPLE_WIKI_API_KEY")
	if err != nil {
		t.Fatalf("NewCredentialRef: %v", err)
	}
	tags, _ := domain.NewTagSet()
	catalog, err := domain.NewCatalog([]domain.Source{domain.NewSource(id, kind, tags, ep, ref)})
	if err != nil {
		t.Fatalf("NewCatalog sources: %v", err)
	}
	return sources.Service{
		Catalog:    catalog,
		Gateways:   fixedResolver{gw: specGateway{}},
		Operations: &cat,
	}
}

// specGateway serves the spec kind's reads through wikiRowsReader. The tree and document calls are not used by the spec
// commands of these tests, so they refuse.
type specGateway struct{}

// Read answers the read through wikiRowsReader.
func (specGateway) Read(ctx context.Context, op domain.OperationID, args domain.Args) (domain.Record, error) {
	return wikiRowsReader{}.Read(ctx, op, args)
}

// ListCollections refuses, because the spec commands of these tests do not list collections.
func (specGateway) ListCollections(context.Context) ([]domain.Collection, error) {
	return nil, domain.ErrUnsupportedOperation
}

// CollectionTree refuses, because the spec commands of these tests do not read trees.
func (specGateway) CollectionTree(context.Context, domain.CollectionID) ([]domain.TreeNode, error) {
	return nil, domain.ErrUnsupportedOperation
}

// Document refuses, because the spec commands of these tests do not read documents.
func (specGateway) Document(context.Context, domain.DocumentID) (domain.SourceDocument, error) {
	return domain.SourceDocument{}, domain.ErrUnsupportedOperation
}

var _ ports.SourceGateway = specGateway{}

// runSpecCommand runs argv through Run with the spec rows and service given, and returns the exit status and the output.
func runSpecCommand(argv []string, rows []operations.Row, svc sources.Service, setupErr error) (int, string, string) {
	var stdout, stderr bytes.Buffer
	deps := Deps{
		Stdout:   &stdout,
		Stderr:   &stderr,
		Getenv:   func(string) string { return "" },
		Service:  svc,
		Specs:    Specs{Catalog: svc.OperationCatalog(), Rows: rows},
		SetupErr: setupErr,
	}
	code := Run(argv, deps)
	return code, stdout.String(), stderr.String()
}

// TestASpecRowRunsAsACommandOfItsKindGroup checks that a loaded spec adds a command group named by its kind: the list row
// runs through the composed catalog and prints its document with exit 0.
func TestASpecRowRunsAsACommandOfItsKindGroup(t *testing.T) {
	row := specCLIRow()
	code, out, errOut := runSpecCommand([]string{"example-wiki", "docs", "list"}, []operations.Row{row}, specService(t, row), nil)
	if code != 0 || !strings.Contains(out, "doc-1") {
		t.Fatalf("code = %d, stdout = %q, stderr = %q; want 0 and doc-1", code, out, errOut)
	}
}

// TestASpecRowWithCLIOffHasNoCommand checks the CLI switch: a row the spec turns off for the command line stays in the
// catalog for the tools, and the command line has no command for it.
func TestASpecRowWithCLIOffHasNoCommand(t *testing.T) {
	row := specCLIRow()
	code, _, errOut := runSpecCommand([]string{"example-wiki", "docs", "list"}, nil, specService(t, row), nil)
	if code != 2 || !strings.Contains(errOut, "example-wiki") {
		t.Fatalf("code = %d, stderr = %q; want 2 and a refused command", code, errOut)
	}
}

// TestASpecKindGroupShowsItsSubcommands checks the help of the kind: the group lists its commands.
func TestASpecKindGroupShowsItsSubcommands(t *testing.T) {
	row := specCLIRow()
	code, out, _ := runSpecCommand([]string{"example-wiki", "--help"}, []operations.Row{row}, specService(t, row), nil)
	if code != 0 || !strings.Contains(out, "docs") {
		t.Fatalf("code = %d, stdout = %q; want 0 and the docs group", code, out)
	}
}

// TestAFailedSpecLoadExitsTwoWithItsCodeOnEveryCommandThatNeedsSources checks a refused spec: the sources command exits 2,
// and its JSON body carries the spec code and names the file and the rule, never a value.
func TestAFailedSpecLoadExitsTwoWithItsCodeOnEveryCommandThatNeedsSources(t *testing.T) {
	failure := &connectorspec.Error{File: "bad.yml", Path: "/kind", Rule: "collides with a built-in kind or a command word"}
	code, out, errOut := runSpecCommand([]string{"--output", "json", "sources", "list"}, nil, sources.Service{}, failure)
	body := out + errOut
	if code != 2 || !strings.Contains(body, "retriever.connector.spec_invalid") {
		t.Fatalf("code = %d, body = %q; want 2 and retriever.connector.spec_invalid", code, body)
	}
	if !strings.Contains(body, "bad.yml") {
		t.Fatalf("body = %q; want the file name", body)
	}
}
