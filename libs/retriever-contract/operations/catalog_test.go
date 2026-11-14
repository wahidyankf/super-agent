package operations

import (
	"errors"
	"slices"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// exampleWikiRow is a read row of a registered kind. Its command path starts with the kind name.
func exampleWikiRow() Row {
	return Row{
		ID:      domain.OperationID("example-wiki.pages"),
		CLI:     "example-wiki pages list",
		Summary: "list the pages of the example wiki",
		Effect:  domain.EffectRead,
		Fields:  []string{"id", "title"},
		ListKey: "pages",
		Noun:    "pages",
		Columns: []string{"id", "title"},
	}
}

// exampleWikiKinds registers the example wiki kind beside the three built-in kinds.
func exampleWikiKinds(t *testing.T) domain.KindSet {
	t.Helper()
	set, err := domain.NewKindSet(
		domain.KindDefinition{Name: "outline", Display: "Outline"},
		domain.KindDefinition{Name: "kiwi", Display: "Kiwi"},
		domain.KindDefinition{Name: "jira", Display: "Jira", UsesIdentity: true},
		domain.KindDefinition{Name: "example-wiki", Display: "Example Wiki"},
	)
	if err != nil {
		t.Fatalf("NewKindSet: %v", err)
	}
	return set
}

// idsOf returns the operation IDs of rows, in order.
func idsOf(rows []Row) []domain.OperationID {
	ids := make([]domain.OperationID, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids
}

// TestBuiltinRowsAreAcceptedByTheBuiltInKinds checks that the built-in rows form a valid catalog over the built-in
// kinds, so the built-in catalog never breaks the rules a registered catalog follows.
func TestBuiltinRowsAreAcceptedByTheBuiltInKinds(t *testing.T) {
	if _, err := NewCatalog(domain.BuiltinKinds(), builtinRows()); err != nil {
		t.Fatalf("NewCatalog(built-in kinds, built-in rows) error = %v, want nil", err)
	}
}

// TestBuiltinCatalogKeepsTheCatalogOrder checks that the built-in catalog lists its rows in the order the built-in rows
// are declared, which the help text and the tool list show.
func TestBuiltinCatalogKeepsTheCatalogOrder(t *testing.T) {
	got := idsOf(BuiltinCatalog().Rows())
	want := idsOf(builtinRows())
	if !slices.Equal(got, want) {
		t.Fatalf("BuiltinCatalog().Rows() order differs from builtinRows()")
	}
	if len(got) == 0 || got[0] != domain.OpCollectionsList {
		t.Fatalf("first built-in row = %v, want %v", got[:1], domain.OpCollectionsList)
	}
}

// TestNewCatalogRefusesARowOfAnUnregisteredKind checks that a row whose command path starts with a kind the set does not
// register is refused, so no row exists without a kind.
func TestNewCatalogRefusesARowOfAnUnregisteredKind(t *testing.T) {
	_, err := NewCatalog(domain.BuiltinKinds(), []Row{exampleWikiRow()})
	if !errors.Is(err, ErrUnregisteredRowKind) {
		t.Fatalf("NewCatalog error = %v, want ErrUnregisteredRowKind", err)
	}
}

// TestNewCatalogRefusesARowWithNoCommandPath checks that a row with no command path has no kind and is refused.
func TestNewCatalogRefusesARowWithNoCommandPath(t *testing.T) {
	row := exampleWikiRow()
	row.CLI = ""
	_, err := NewCatalog(exampleWikiKinds(t), []Row{row})
	if !errors.Is(err, ErrUnregisteredRowKind) {
		t.Fatalf("NewCatalog error = %v, want ErrUnregisteredRowKind", err)
	}
}

// TestNewCatalogRefusesARowWithNoID checks that every row carries an operation ID.
func TestNewCatalogRefusesARowWithNoID(t *testing.T) {
	row := exampleWikiRow()
	row.ID = ""
	_, err := NewCatalog(exampleWikiKinds(t), []Row{row})
	if !errors.Is(err, ErrInvalidRow) {
		t.Fatalf("NewCatalog error = %v, want ErrInvalidRow", err)
	}
}

// TestNewCatalogRefusesADuplicateID checks that two rows with one operation ID are refused, so Lookup has one answer.
func TestNewCatalogRefusesADuplicateID(t *testing.T) {
	_, err := NewCatalog(exampleWikiKinds(t), []Row{exampleWikiRow(), exampleWikiRow()})
	if !errors.Is(err, ErrDuplicateRow) {
		t.Fatalf("NewCatalog error = %v, want ErrDuplicateRow", err)
	}
}

// TestRegisteredRowIsFoundByLookupAndRowsOf checks that a row of a registered kind joins the catalog, and that its kind
// lists it and no other kind does.
func TestRegisteredRowIsFoundByLookupAndRowsOf(t *testing.T) {
	cat, err := NewCatalog(exampleWikiKinds(t), []Row{exampleWikiRow()})
	if err != nil {
		t.Fatalf("NewCatalog: %v", err)
	}
	row, ok := cat.Lookup(domain.OperationID("example-wiki.pages"))
	if !ok || row.CLI != "example-wiki pages list" {
		t.Fatalf("Lookup = %v, %v; want the registered row", row.CLI, ok)
	}
	kind, err := exampleWikiKinds(t).Kind("example-wiki")
	if err != nil {
		t.Fatalf("Kind(example-wiki): %v", err)
	}
	if got := cat.RowsOf(kind); len(got) != 1 || got[0].ID != "example-wiki.pages" {
		t.Fatalf("RowsOf(example-wiki) = %v, want the one registered row", idsOf(got))
	}
	outline, _ := exampleWikiKinds(t).Kind("outline")
	if got := cat.RowsOf(outline); len(got) != 0 {
		t.Fatalf("RowsOf(outline) = %v, want none", idsOf(got))
	}
}

// TestKindOfReadsTheFirstWordOfTheCommandPath checks how a row finds its kind: the first word of its command path,
// resolved through the set. A word that names no registered kind gives no kind.
func TestKindOfReadsTheFirstWordOfTheCommandPath(t *testing.T) {
	cat, err := NewCatalog(exampleWikiKinds(t), []Row{exampleWikiRow()})
	if err != nil {
		t.Fatalf("NewCatalog: %v", err)
	}
	kind, ok := cat.KindOf(exampleWikiRow())
	if !ok || kind.String() != "example-wiki" {
		t.Fatalf("KindOf(example-wiki row) = %v, %v; want example-wiki", kind, ok)
	}
	if _, ok := cat.KindOf(Row{CLI: "notion pages list"}); ok {
		t.Fatal("KindOf(notion row) found a kind, want none")
	}
	if _, ok := cat.KindOf(Row{}); ok {
		t.Fatal("KindOf(empty row) found a kind, want none")
	}
}

// TestRegisteredRowIsSupportedByItsKindOnly checks that Supports is true for the row's own kind and false for another.
func TestRegisteredRowIsSupportedByItsKindOnly(t *testing.T) {
	set := exampleWikiKinds(t)
	cat, err := NewCatalog(set, []Row{exampleWikiRow()})
	if err != nil {
		t.Fatalf("NewCatalog: %v", err)
	}
	wiki, _ := set.Kind("example-wiki")
	outline, _ := set.Kind("outline")
	if !cat.Supports(wiki, exampleWikiRow()) {
		t.Fatal("Supports(example-wiki, its row) = false, want true")
	}
	if cat.Supports(outline, exampleWikiRow()) {
		t.Fatal("Supports(outline, the example wiki row) = true, want false")
	}
}

// TestBuiltinKiwiSupportsOnlyItsClosedList checks that the closed Kiwi list still applies in the built-in catalog: a
// Kiwi row outside the list is not supported, while a listed row is.
func TestBuiltinKiwiSupportsOnlyItsClosedList(t *testing.T) {
	extra := Row{ID: domain.OperationID("kiwi.extra"), CLI: "kiwi extra", Effect: domain.EffectRead}
	cat, err := NewCatalog(domain.BuiltinKinds(), []Row{extra})
	if err != nil {
		t.Fatalf("NewCatalog: %v", err)
	}
	if cat.Supports(domain.KindKiwi, extra) {
		t.Fatal("Supports(kiwi, a Kiwi row outside the closed list) = true, want false")
	}
	listed, ok := BuiltinCatalog().Lookup(domain.OpKiwiProductsList)
	if !ok || !BuiltinCatalog().Supports(domain.KindKiwi, listed) {
		t.Fatal("the built-in Kiwi row is not supported by the Kiwi kind")
	}
}

// TestCatalogRowsReturnsACopy checks that a caller who changes the returned rows does not change the catalog.
func TestCatalogRowsReturnsACopy(t *testing.T) {
	cat, err := NewCatalog(exampleWikiKinds(t), []Row{exampleWikiRow()})
	if err != nil {
		t.Fatalf("NewCatalog: %v", err)
	}
	rows := cat.Rows()
	rows[0].CLI = "changed"
	if got := cat.Rows()[0].CLI; got != "example-wiki pages list" {
		t.Fatalf("catalog row after a change = %q, want the original", got)
	}
}

// TestNewCatalogOwnsItsRowsSlice checks that a change to the slice the caller passed does not change the catalog.
func TestNewCatalogOwnsItsRowsSlice(t *testing.T) {
	rows := []Row{exampleWikiRow()}
	cat, err := NewCatalog(exampleWikiKinds(t), rows)
	if err != nil {
		t.Fatalf("NewCatalog: %v", err)
	}
	rows[0].CLI = "changed"
	if got := cat.Rows()[0].CLI; got != "example-wiki pages list" {
		t.Fatalf("catalog row after the caller changed its slice = %q, want the original", got)
	}
}
