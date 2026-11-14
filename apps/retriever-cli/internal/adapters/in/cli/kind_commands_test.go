package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/ports"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/sources"
)

// kiwiWiki is a Kiwi gateway for the CLI tests. It answers every read with a fixed record, records each read and write
// by operation ID, and never reaches a host. A test that expects no request checks the counts.
type kiwiWiki struct {
	reads  []domain.OperationID
	writes []domain.OperationID
}

func (k *kiwiWiki) ListCollections(context.Context) ([]domain.Collection, error) {
	return nil, domain.ErrUnsupportedOperation
}

func (k *kiwiWiki) CollectionTree(context.Context, domain.CollectionID) ([]domain.TreeNode, error) {
	return nil, domain.ErrUnsupportedOperation
}

func (k *kiwiWiki) Document(_ context.Context, id domain.DocumentID) (domain.SourceDocument, error) {
	return domain.SourceDocument{
		ID:           id.String(),
		Title:        "Login works",
		URL:          "https://kiwi.example.test/case/40/",
		Markdown:     "Steps to log in.\n",
		CollectionID: "product-3",
	}, nil
}

func (k *kiwiWiki) Read(_ context.Context, op domain.OperationID, _ domain.Args) (domain.Record, error) {
	k.reads = append(k.reads, op)
	switch op {
	case domain.OpKiwiProductsList:
		return domain.NewList([]domain.Record{
			domain.NewRecord(domain.Field{Name: "id", Value: "product-3"}, domain.Field{Name: "name", Value: "Web"}),
		}), nil
	case domain.OpKiwiDocumentsGet:
		return domain.NewRecord(
			domain.Field{Name: "id", Value: "case-40"},
			domain.Field{Name: "title", Value: "Login works"},
			domain.Field{Name: "url", Value: "https://kiwi.example.test/case/40/"},
			domain.Field{Name: "markdown", Value: "Steps to log in.\n"},
			domain.Field{Name: "collectionId", Value: "product-3"},
		), nil
	}
	return domain.Record{}, domain.ErrUnsupportedOperation
}

// Write answers each write with the fields its catalog row names, as the wiki does, so the reply passes the row check.
func (k *kiwiWiki) Write(_ context.Context, op domain.OperationID, _ domain.Args) (domain.Record, error) {
	k.writes = append(k.writes, op)
	if op == domain.OpKiwiPlansAddCase {
		return domain.NewRecord(domain.Field{Name: "planId", Value: "plan-12"}, domain.Field{Name: "caseId", Value: "case-40"}), nil
	}
	return domain.NewRecord(
		domain.Field{Name: "id", Value: "case-41"},
		domain.Field{Name: "title", Value: "New case"},
		domain.Field{Name: "url", Value: "https://kiwi.example.test/case/41/"},
	), nil
}

// kiwiService returns a service whose only source is a Kiwi source at write access, served by gw.
func kiwiService(gw ports.SourceGateway) sources.Service {
	id, err := domain.NewSourceID("kiwi")
	if err != nil {
		panic(err)
	}
	ep, err := domain.NewEndpoint("https://kiwi.example.test")
	if err != nil {
		panic(err)
	}
	ref, err := domain.NewCredentialRef("EXAMPLE_KIWI_API_KEY")
	if err != nil {
		panic(err)
	}
	tags, err := domain.NewTagSet()
	if err != nil {
		panic(err)
	}
	cat, err := domain.NewCatalog([]domain.Source{
		domain.NewSource(id, domain.KindKiwi, tags, domain.AccessWrite, ep, ref),
	})
	if err != nil {
		panic(err)
	}
	return sources.Service{Catalog: cat, Gateways: fixedResolver{gw: gw}}
}

// kiwiSetup gives the CLI a Kiwi service and an empty export sink.
func kiwiSetup(gw ports.SourceGateway) outlineSetup {
	return outlineSetup{service: kiwiService(gw), sink: newMemSink()}
}

// findChild walks the command tree along path, and returns nil when a word is missing.
func findChild(c *command, path ...string) *command {
	for _, word := range path {
		if c == nil {
			return nil
		}
		c = c.child(word)
	}
	return c
}

// TestKiwiGroupIsGeneratedFromTheKiwiRows checks that the kiwi group holds the six Kiwi commands and the export, each
// with its usage line, and that no Kiwi command sits under outline.
func TestKiwiGroupIsGeneratedFromTheKiwiRows(t *testing.T) {
	tree := commandTree()
	want := map[string]string{
		"products list":     "retriever kiwi products list",
		"products tree":     "retriever kiwi products tree <collectionId>",
		"documents get":     "retriever kiwi documents get <documentId>",
		"test-cases create": "retriever kiwi test-cases create <summary>",
		"test-cases update": "retriever kiwi test-cases update <documentId>",
		"plans add-case":    "retriever kiwi plans add-case <planId>",
		"export":            "retriever kiwi export <documentId> --out <dir>",
	}
	for path, usage := range want {
		c := findChild(tree, append([]string{"kiwi"}, strings.Fields(path)...)...)
		if c == nil || c.leaf == nil {
			t.Fatalf("retriever kiwi %s is missing from the command tree", path)
		}
		if !strings.HasPrefix(c.leaf.usage, usage) {
			t.Errorf("retriever kiwi %s usage = %q, want a prefix of %q", path, c.leaf.usage, usage)
		}
	}
	if findChild(tree, "outline", "products") != nil {
		t.Fatal("a Kiwi command sits under outline")
	}
}

// TestJiraGroupIsGeneratedFromTheJiraRows checks that the jira group holds the five Jira reads and the export.
func TestJiraGroupIsGeneratedFromTheJiraRows(t *testing.T) {
	tree := commandTree()
	for _, path := range []string{"whoami", "projects list", "issues search", "issues get", "comments list", "export"} {
		c := findChild(tree, append([]string{"jira"}, strings.Fields(path)...)...)
		if c == nil || c.leaf == nil {
			t.Fatalf("retriever jira %s is missing from the command tree", path)
		}
	}
}

// TestKiwiProductsListReadsTheKiwiSource checks that a Kiwi read runs the Kiwi operation against the Kiwi source, and
// prints the products.
func TestKiwiProductsListReadsTheKiwiSource(t *testing.T) {
	gw := &kiwiWiki{}
	code, stdout, stderr := runRetriever(t, []string{"kiwi", "products", "list"}, kiwiSetup(gw))
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, stderr)
	}
	if len(gw.reads) != 1 || gw.reads[0] != domain.OpKiwiProductsList {
		t.Fatalf("reads = %v, want one kiwi.products.list", gw.reads)
	}
	if !strings.Contains(stdout, "product-3") {
		t.Fatalf("stdout %q does not list the product", stdout)
	}
}

// TestKiwiDocumentsGetPrintsTheMarkdown checks that a Kiwi document prints its Markdown body.
func TestKiwiDocumentsGetPrintsTheMarkdown(t *testing.T) {
	gw := &kiwiWiki{}
	code, stdout, stderr := runRetriever(t, []string{"kiwi", "documents", "get", "case-40"}, kiwiSetup(gw))
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, stderr)
	}
	if stdout != "Steps to log in.\n" {
		t.Fatalf("stdout = %q, want the Markdown body", stdout)
	}
}

// TestKiwiTestCaseCreateWritesOnce checks that a Kiwi write runs once, with --source, and says what changed.
func TestKiwiTestCaseCreateWritesOnce(t *testing.T) {
	gw := &kiwiWiki{}
	argv := []string{"kiwi", "test-cases", "create", "New case", "--product", "product-3", "--category", "Smoke",
		"--priority", "P1", "--source", "kiwi"}
	code, stdout, stderr := runRetriever(t, argv, kiwiSetup(gw))
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, stderr)
	}
	if len(gw.writes) != 1 || gw.writes[0] != domain.OpKiwiTestCasesCreate {
		t.Fatalf("writes = %v, want one kiwi.test-cases.create", gw.writes)
	}
	if !strings.HasPrefix(stdout, "changed: kiwi test-cases create\n") {
		t.Fatalf("stdout %q does not name the change", stdout)
	}
}

// TestKiwiWriteWithoutSourceIsRefused checks that a Kiwi write names its source, and makes no request without it.
func TestKiwiWriteWithoutSourceIsRefused(t *testing.T) {
	gw := &kiwiWiki{}
	code, _, _ := runRetriever(t, []string{"kiwi", "plans", "add-case", "plan-12", "--case-id", "case-40"}, kiwiSetup(gw))
	if code == 0 {
		t.Fatal("a Kiwi write without --source exited 0")
	}
	if len(gw.writes) != 0 {
		t.Fatalf("a write without --source reached the gateway: %v", gw.writes)
	}
}

// TestKiwiTestCaseUpdateNeedsOneField checks that an update with no field is refused before any request.
func TestKiwiTestCaseUpdateNeedsOneField(t *testing.T) {
	gw := &kiwiWiki{}
	code, _, _ := runRetriever(t, []string{"kiwi", "test-cases", "update", "case-40", "--source", "kiwi"}, kiwiSetup(gw))
	if code == 0 {
		t.Fatal("an update with no field exited 0")
	}
	if len(gw.writes) != 0 {
		t.Fatalf("an update with no field reached the gateway: %v", gw.writes)
	}
}

// TestKiwiPlansAddCaseWrites checks the plan write, which takes the case as a flag.
func TestKiwiPlansAddCaseWrites(t *testing.T) {
	gw := &kiwiWiki{}
	code, stdout, stderr := runRetriever(t, []string{"kiwi", "plans", "add-case", "plan-12", "--case-id", "case-40",
		"--source", "kiwi"}, kiwiSetup(gw))
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, stderr)
	}
	if len(gw.writes) != 1 || gw.writes[0] != domain.OpKiwiPlansAddCase {
		t.Fatalf("writes = %v, want one kiwi.plans.add-case", gw.writes)
	}
	if !strings.HasPrefix(stdout, "changed: kiwi plans add-case\n") {
		t.Fatalf("stdout %q does not name the change", stdout)
	}
}

// TestKiwiExportWritesTheDocumentUnderTheOutFolder checks the Kiwi export: one file, named with the case ID, written
// under the --out folder, and the path printed.
func TestKiwiExportWritesTheDocumentUnderTheOutFolder(t *testing.T) {
	gw := &kiwiWiki{}
	setup := kiwiSetup(gw)
	code, stdout, stderr := runRetriever(t, []string{"kiwi", "export", "case-40", "--out", "notes/kiwi"}, setup)
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, stderr)
	}
	if len(setup.sink.names) != 1 || !strings.HasPrefix(setup.sink.names[0], "case-40-") {
		t.Fatalf("exported %v, want one file named with case-40", setup.sink.names)
	}
	if !strings.HasPrefix(stdout, "Wrote notes/kiwi/case-40-") {
		t.Fatalf("stdout %q does not print the written location", stdout)
	}
}

// TestKiwiExportFailureCarriesTheKiwiCode checks that a failed Kiwi export reports a JSON code with the kiwi prefix,
// on the export path and not only on the reads.
func TestKiwiExportFailureCarriesTheKiwiCode(t *testing.T) {
	setup := kiwiSetup(&kiwiWiki{})
	setup.sink.err = fmt.Errorf("%w: %w", domain.ErrExportWriteFailed,
		errors.New("open /private/store/case-40.md: permission denied"))
	code, stdout, stderr := runRetriever(t, []string{"kiwi", "export", "case-40", "--out", "notes/kiwi", "--output", "json"}, setup)
	if code == 0 {
		t.Fatal("a failed export exited 0")
	}
	if stdout != "" {
		t.Fatalf("a failed JSON export wrote to stdout: %q", stdout)
	}
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(stderr), &body); err != nil {
		t.Fatalf("stderr is not a JSON error body: %v (%q)", err, stderr)
	}
	if !strings.HasPrefix(body.Error.Code, "retriever.kiwi.") {
		t.Fatalf("error code = %q, want the retriever.kiwi. prefix", body.Error.Code)
	}
}

// TestKiwiCommandWithNoKiwiSourceNamesTheKiwiCode checks that a kiwi command with no kiwi source says so, and that its
// JSON code carries the kiwi prefix, not the Outline one.
func TestKiwiCommandWithNoKiwiSourceNamesTheKiwiCode(t *testing.T) {
	setup := outlineSetup{service: oneOutlineService(&kiwiWiki{}), sink: newMemSink()}
	code, _, stderr := runRetriever(t, []string{"kiwi", "products", "list", "--output", "json"}, setup)
	if code == 0 {
		t.Fatal("a kiwi command with no kiwi source exited 0")
	}
	if !strings.Contains(stderr, `"code":"retriever.kiwi.`) {
		t.Fatalf("stderr %q does not carry the kiwi error code", stderr)
	}
	if strings.Contains(stderr, "outline") {
		t.Fatalf("stderr %q names outline for a kiwi command", stderr)
	}
}

// TestSourcesListFiltersByTheKiwiKind checks that sources list --kind kiwi names the Kiwi source and not the Outline one.
func TestSourcesListFiltersByTheKiwiKind(t *testing.T) {
	setup := outlineSetup{service: outlineAndKiwiCatalog(&kiwiWiki{}), sink: newMemSink()}
	code, stdout, stderr := runRetriever(t, []string{"sources", "list", "--kind", "kiwi"}, setup)
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, stderr)
	}
	if !strings.HasPrefix(stdout, "1 source\n") || !strings.Contains(stdout, "kiwi") {
		t.Fatalf("stdout = %q, want one kiwi source", stdout)
	}
	if strings.Contains(stdout, "outline") {
		t.Fatalf("stdout %q names the outline source for --kind kiwi", stdout)
	}
}

// outlineAndKiwiCatalog returns a service with the Outline source and the Kiwi source, both served by gw.
func outlineAndKiwiCatalog(gw ports.SourceGateway) sources.Service {
	outlineID, err := domain.NewSourceID("outline")
	if err != nil {
		panic(err)
	}
	outlineEP, err := domain.NewEndpoint("https://docs.example.test")
	if err != nil {
		panic(err)
	}
	outlineRef, err := domain.NewCredentialRef("EXAMPLE_OUTLINE_API_KEY")
	if err != nil {
		panic(err)
	}
	tags, err := domain.NewTagSet()
	if err != nil {
		panic(err)
	}
	cat, err := domain.NewCatalog([]domain.Source{
		domain.NewSource(outlineID, domain.KindOutline, tags, domain.AccessRead, outlineEP, outlineRef),
		kiwiService(gw).Catalog.All()[0],
	})
	if err != nil {
		panic(err)
	}
	return sources.Service{Catalog: cat, Gateways: fixedResolver{gw: gw}}
}
