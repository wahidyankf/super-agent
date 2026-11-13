package graphql

import (
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/ports"
	usecases "github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/sources"
)

// byKindResolver serves one gateway per kind, so a test shows which kind a query reached. A kind with no gateway is
// ErrUnsupportedSource, as the registry answers for a kind it has no builder for.
type byKindResolver struct {
	gws map[domain.SourceKind]ports.SourceGateway
}

func (r byKindResolver) For(src domain.Source) (ports.SourceGateway, error) {
	gw, ok := r.gws[src.Kind()]
	if !ok {
		return nil, domain.ErrUnsupportedSource
	}
	return gw, nil
}

func (r byKindResolver) Reader(src domain.Source) (ports.SourceReader, error) {
	gw, err := r.For(src)
	if err != nil {
		return nil, err
	}
	reader, ok := gw.(ports.SourceReader)
	if !ok {
		return nil, domain.ErrUnsupportedSource
	}
	return reader, nil
}

func (r byKindResolver) Writer(src domain.Source) (ports.SourceWriter, error) {
	gw, err := r.For(src)
	if err != nil {
		return nil, err
	}
	writer, ok := gw.(ports.SourceWriter)
	if !ok {
		return nil, domain.ErrUnsupportedSource
	}
	return writer, nil
}

func (r byKindResolver) Admin(src domain.Source) (ports.SourceAdmin, error) {
	gw, err := r.For(src)
	if err != nil {
		return nil, err
	}
	admin, ok := gw.(ports.SourceAdmin)
	if !ok {
		return nil, domain.ErrUnsupportedSource
	}
	return admin, nil
}

// mustKindSource builds one source of the given kind. The constant inputs are valid, so a constructor failure panics.
func mustKindSource(kind domain.SourceKind, id, host, credential string, tags []string, access domain.Access) domain.Source {
	sid, err := domain.NewSourceID(id)
	if err != nil {
		panic(err)
	}
	ep, err := domain.NewEndpoint("https://" + host)
	if err != nil {
		panic(err)
	}
	ref, err := domain.NewCredentialRef(credential)
	if err != nil {
		panic(err)
	}
	set, err := domain.NewTagSet(tags...)
	if err != nil {
		panic(err)
	}
	return domain.NewSource(sid, kind, set, access, ep, ref)
}

// outlineAndKiwiServices returns a catalog with one Outline source and one Kiwi source, each served by its own gateway.
func outlineAndKiwiServices(outlineGW, kiwiGW ports.SourceGateway) usecases.Service {
	cat, err := domain.NewCatalog([]domain.Source{
		mustSource("outline", "docs.example.test", "EXAMPLE_OUTLINE_API_KEY", []string{"company"}, domain.AccessRead),
		mustKindSource(domain.KindKiwi, "kiwi", "kiwi.example.test", "EXAMPLE_KIWI_API_KEY",
			[]string{"testing"}, domain.AccessWrite),
	})
	if err != nil {
		panic(err)
	}
	return usecases.Service{Catalog: cat, Gateways: byKindResolver{gws: map[domain.SourceKind]ports.SourceGateway{
		domain.KindOutline: outlineGW,
		domain.KindKiwi:    kiwiGW,
	}}}
}

// kiwiCases returns a Kiwi gateway with one collection, one tree node, and one document.
func kiwiCases() *countingGateway {
	return &countingGateway{
		collections: []domain.Collection{{ID: "product-3", Name: "Web"}},
		tree:        []domain.TreeNode{{ID: "plan-12", Title: "Smoke", URL: "https://kiwi.example.test/plan/12/"}},
		doc: domain.SourceDocument{
			ID:           "case-40",
			Title:        "Login works",
			URL:          "https://kiwi.example.test/case/40/",
			Markdown:     "Steps to log in.\n",
			CollectionID: "product-3",
		},
	}
}

// TestCollectionsWithKindKiwiReadsOnlyTheKiwiSource checks that a kind names the source kind a query reaches, and that
// the Outline gateway is never called for it.
func TestCollectionsWithKindKiwiReadsOnlyTheKiwiSource(t *testing.T) {
	outlineGW := &countingGateway{collections: []domain.Collection{{ID: "col-1", Name: "Engineering"}}}
	kiwiGW := kiwiCases()
	r := sendServed(t, outlineAndKiwiServices(outlineGW, kiwiGW), `{ collections(kind: "kiwi") { id name } }`)
	rows := listRows(t, r, "collections")
	if len(rows) != 1 || rows[0]["id"] != "product-3" || rows[0]["name"] != "Web" {
		t.Fatalf("collections = %v, want the Kiwi product", rows)
	}
	if outlineGW.calls.Load() != 0 || kiwiGW.calls.Load() != 1 {
		t.Fatalf("calls: outline %d, kiwi %d; want 0 and 1", outlineGW.calls.Load(), kiwiGW.calls.Load())
	}
}

// TestCollectionsWithoutKindStaysOnOutline checks the default: an omitted kind is outline, as every call before the
// argument existed.
func TestCollectionsWithoutKindStaysOnOutline(t *testing.T) {
	outlineGW := &countingGateway{collections: []domain.Collection{{ID: "col-1", Name: "Engineering"}}}
	kiwiGW := kiwiCases()
	r := sendServed(t, outlineAndKiwiServices(outlineGW, kiwiGW), `{ collections { id } }`)
	rows := listRows(t, r, "collections")
	if len(rows) != 1 || rows[0]["id"] != "col-1" {
		t.Fatalf("collections = %v, want the Outline collection", rows)
	}
	if outlineGW.calls.Load() != 1 || kiwiGW.calls.Load() != 0 {
		t.Fatalf("calls: outline %d, kiwi %d; want 1 and 0", outlineGW.calls.Load(), kiwiGW.calls.Load())
	}
}

// TestCollectionTreeWithKindKiwiReadsTheKiwiTree checks the tree query with the kind argument.
func TestCollectionTreeWithKindKiwiReadsTheKiwiTree(t *testing.T) {
	outlineGW := &countingGateway{}
	kiwiGW := kiwiCases()
	r := sendServed(t, outlineAndKiwiServices(outlineGW, kiwiGW),
		`{ collectionTree(id: "product-3", kind: "kiwi") { id depth } }`)
	if len(r.messages()) != 0 {
		t.Fatalf("collectionTree errors %v", r.messages())
	}
	if outlineGW.calls.Load() != 0 || kiwiGW.calls.Load() != 1 {
		t.Fatalf("calls: outline %d, kiwi %d; want 0 and 1", outlineGW.calls.Load(), kiwiGW.calls.Load())
	}
}

// TestDocumentWithKindKiwiReturnsTheKiwiDocument checks the document query with the kind argument.
func TestDocumentWithKindKiwiReturnsTheKiwiDocument(t *testing.T) {
	outlineGW := &countingGateway{}
	kiwiGW := kiwiCases()
	r := sendServed(t, outlineAndKiwiServices(outlineGW, kiwiGW),
		`{ document(id: "case-40", kind: "kiwi") { id markdown } }`)
	if len(r.messages()) != 0 {
		t.Fatalf("document errors %v", r.messages())
	}
	doc, ok := r.Data["document"].(map[string]any)
	if !ok || doc["id"] != "case-40" || doc["markdown"] != "Steps to log in.\n" {
		t.Fatalf("document = %v, want the Kiwi case", r.Data["document"])
	}
	if outlineGW.calls.Load() != 0 {
		t.Fatalf("the Outline gateway was called %d times", outlineGW.calls.Load())
	}
}

// TestUnknownKindIsRefusedBeforeAnyRequest checks the safe text for a kind the code does not know, and that no gateway
// is called for it.
func TestUnknownKindIsRefusedBeforeAnyRequest(t *testing.T) {
	outlineGW := &countingGateway{}
	kiwiGW := kiwiCases()
	r := sendServed(t, outlineAndKiwiServices(outlineGW, kiwiGW), `{ collections(kind: "notion") { id } }`)
	assertErrorContains(t, r, "unknown source kind (known: outline, kiwi, jira)")
	if outlineGW.calls.Load() != 0 || kiwiGW.calls.Load() != 0 {
		t.Fatalf("an unknown kind reached a gateway")
	}
}

// TestKiwiSourceNamedWithoutKindIsWrongKind checks that a Kiwi source named by a query with no kind is refused, and not
// guessed: the kind is outline, so the source is of another kind.
func TestKiwiSourceNamedWithoutKindIsWrongKind(t *testing.T) {
	outlineGW := &countingGateway{}
	kiwiGW := kiwiCases()
	r := sendServed(t, outlineAndKiwiServices(outlineGW, kiwiGW), `{ collections(source: "kiwi") { id } }`)
	assertErrorContains(t, r, "the source is of another kind")
	if kiwiGW.calls.Load() != 0 {
		t.Fatalf("a Kiwi source named without a kind reached the gateway")
	}
}

// TestJiraKindWithNoJiraSourceSaysSo checks that jira is an accepted kind, and that a catalog with no jira source says
// so with the jira name.
func TestJiraKindWithNoJiraSourceSaysSo(t *testing.T) {
	outlineGW := &countingGateway{}
	kiwiGW := kiwiCases()
	r := sendServed(t, outlineAndKiwiServices(outlineGW, kiwiGW), `{ collections(kind: "jira") { id } }`)
	assertErrorContains(t, r, "no jira source is configured")
}

// TestServedKiwiSourceIsCappedToRead checks that the served service, capped to read, reports a Kiwi source at read.
func TestServedKiwiSourceIsCappedToRead(t *testing.T) {
	svc := outlineAndKiwiServices(&countingGateway{}, kiwiCases()).Capped(domain.AccessRead)
	rows := sourceRows(t, sendServed(t, svc, `{ sources(kind: "kiwi") { id access } }`))
	if len(rows) != 1 || rows[0]["access"] != "read" {
		t.Fatalf("sources = %v, want the Kiwi source at read", rows)
	}
}

// TestServeHasNoKiwiMutation checks that the served API has no write for a Kiwi test case: the schema has no such
// mutation, so the request is a validation error.
func TestServeHasNoKiwiMutation(t *testing.T) {
	r := sendServed(t, outlineAndKiwiServices(&countingGateway{}, kiwiCases()),
		`mutation { createTestCase(summary: "New case") { id } }`)
	assertErrorContains(t, r, `does not support operation type "mutation"`)
}

// listRows returns the rows of one list field of a reply that has no errors.
func listRows(t *testing.T, r reply, field string) []map[string]any {
	t.Helper()
	if len(r.Errors) != 0 {
		t.Fatalf("reply has errors %v, want none", r.messages())
	}
	raw, ok := r.Data[field].([]any)
	if !ok {
		t.Fatalf("data.%s is %T, want a list", field, r.Data[field])
	}
	rows := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		row, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("a row is %T, want an object", item)
		}
		rows = append(rows, row)
	}
	return rows
}
