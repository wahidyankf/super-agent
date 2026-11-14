package operations

import (
	"errors"
	"slices"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// TestRowsOfReturnsTheRowsOfOneKind checks the kind filter of the catalog: the Kiwi kind holds exactly the six Kiwi rows, the
// Outline kind holds every other row, and a kind with no rows holds none.
func TestRowsOfReturnsTheRowsOfOneKind(t *testing.T) {
	kiwi := RowsOf(domain.KindKiwi)
	if len(kiwi) != len(kiwiWants) {
		t.Fatalf("RowsOf(kiwi) holds %d rows, want %d", len(kiwi), len(kiwiWants))
	}
	for i, want := range kiwiWants {
		if kiwi[i].ID != want.id {
			t.Errorf("RowsOf(kiwi)[%d] = %s, want %s", i, kiwi[i].ID, want.id)
		}
	}
	outline := RowsOf(domain.KindOutline)
	jira := RowsOf(domain.KindJira)
	if len(outline)+len(kiwi)+len(jira) != len(Rows()) {
		t.Fatalf("outline %d, kiwi %d, and jira %d rows do not add up to the %d catalog rows", len(outline), len(kiwi), len(jira), len(Rows()))
	}
	for _, row := range outline {
		if k, ok := row.Kind(); !ok || k != domain.KindOutline {
			t.Fatalf("RowsOf(outline) holds %s", row.ID)
		}
	}
	if got := RowsOf(domain.SourceKind{}); len(got) != 0 {
		t.Fatalf("RowsOf(zero kind) holds %d rows, want none", len(got))
	}
}

// kiwiWant is the expected shape of one Kiwi row.
type kiwiWant struct {
	id     domain.OperationID
	cli    string
	effect domain.Effect
	fields []string
}

// kiwiWants are the six Kiwi rows, in the order of the spec table.
var kiwiWants = []kiwiWant{
	{domain.OpKiwiProductsList, "kiwi products list", domain.EffectRead, []string{"id", "name"}},
	{domain.OpKiwiProductsTree, "kiwi products tree", domain.EffectRead, []string{"id", "title", "url", "parentId", "depth"}},
	{domain.OpKiwiDocumentsGet, "kiwi documents get", domain.EffectRead, []string{"id", "title", "url", "markdown", "collectionId"}},
	{domain.OpKiwiTestCasesCreate, "kiwi test-cases create", domain.EffectAdditive, []string{"id", "title", "url"}},
	{domain.OpKiwiTestCasesUpdate, "kiwi test-cases update", domain.EffectUpdate, []string{"id", "title", "url"}},
	{domain.OpKiwiPlansAddCase, "kiwi plans add-case", domain.EffectAdditive, []string{"planId", "caseId"}},
}

// TestKiwiRowParameters checks the positional parameter and the required fields of the Kiwi rows.
func TestKiwiRowParameters(t *testing.T) {
	cases := []struct {
		id       domain.OperationID
		required string
		kind     Kind
	}{
		{domain.OpKiwiProductsTree, "collectionId", KindCollectionID},
		{domain.OpKiwiDocumentsGet, "documentId", KindDocumentID},
		{domain.OpKiwiTestCasesUpdate, "documentId", KindDocumentID},
		{domain.OpKiwiPlansAddCase, "planId", KindDocumentID},
	}
	for _, tc := range cases {
		row, _ := Lookup(tc.id)
		if got := row.Positional(); got != tc.required {
			t.Errorf("%s positional = %q, want %q", tc.id, got, tc.required)
		}
		p, ok := row.Param(tc.required)
		if !ok || p.Kind != tc.kind || !p.Required {
			t.Errorf("%s param %s = %+v, want a required %s", tc.id, tc.required, p, tc.kind)
		}
	}
	create, _ := Lookup(domain.OpKiwiTestCasesCreate)
	for _, name := range []string{"summary", "product", "category", "priority"} {
		if p, ok := create.Param(name); !ok || !p.Required {
			t.Errorf("create param %s = %+v, want required", name, p)
		}
	}
	for _, name := range []string{"status", "text"} {
		if p, ok := create.Param(name); !ok || p.Required {
			t.Errorf("create param %s = %+v, want optional", name, p)
		}
	}
	if p, _ := create.Param("product"); p.Kind != KindCollectionID {
		t.Errorf("create product kind = %s, want CollectionID", p.Kind)
	}
}

// TestEveryRowNamesAKnownKind checks that the first word of every row's CLI path is a known kind, so Row.Kind is never
// unset for a catalog row.
func TestEveryRowNamesAKnownKind(t *testing.T) {
	for _, row := range Rows() {
		if _, ok := row.Kind(); !ok {
			t.Errorf("%s has CLI %q, which names no known kind", row.ID, row.CLI)
		}
	}
}

// TestRowKindReadsTheFirstWordOfTheCLIPath checks Row.Kind on synthetic rows, including the ones it refuses.
func TestRowKindReadsTheFirstWordOfTheCLIPath(t *testing.T) {
	cases := []struct {
		cli  string
		want domain.SourceKind
		ok   bool
	}{
		{"outline documents get", domain.KindOutline, true},
		{"kiwi products list", domain.KindKiwi, true},
		{"notion pages list", domain.SourceKind{}, false},
		{"", domain.SourceKind{}, false},
	}
	for _, tc := range cases {
		got, ok := Row{CLI: tc.cli}.Kind()
		if ok != tc.ok || got != tc.want {
			t.Errorf("Row{CLI: %q}.Kind() = %v, %v, want %v, %v", tc.cli, got, ok, tc.want, tc.ok)
		}
	}
}

// TestSupportsIsAnExplicitStatementPerKind checks that Outline supports every Outline row and no Kiwi row, and that
// Kiwi supports exactly the six Kiwi rows.
func TestSupportsIsAnExplicitStatementPerKind(t *testing.T) {
	outline, kiwi := 0, 0
	for _, row := range Rows() {
		k, _ := row.Kind()
		if k == domain.KindOutline {
			outline++
			if !Supports(domain.KindOutline, row) || Supports(domain.KindKiwi, row) {
				t.Errorf("%s: outline row supported by outline=%v kiwi=%v", row.ID, Supports(domain.KindOutline, row), Supports(domain.KindKiwi, row))
			}
		}
		if k == domain.KindKiwi {
			kiwi++
			if !Supports(domain.KindKiwi, row) || Supports(domain.KindOutline, row) {
				t.Errorf("%s: kiwi row supported by kiwi=%v outline=%v", row.ID, Supports(domain.KindKiwi, row), Supports(domain.KindOutline, row))
			}
		}
	}
	if kiwi != len(kiwiWants) || outline == 0 {
		t.Fatalf("rows by kind: outline %d, kiwi %d", outline, kiwi)
	}
	if Supports(domain.SourceKind{}, Row{ID: domain.OpKiwiProductsList}) {
		t.Fatal("the zero kind supports a row")
	}
}

// TestKiwiOperationListIsClosed checks that the closed list of Kiwi operations holds the six IDs and nothing else.
func TestKiwiOperationListIsClosed(t *testing.T) {
	if len(kiwiOperations) != len(kiwiWants) {
		t.Fatalf("kiwiOperations holds %d IDs, want %d", len(kiwiOperations), len(kiwiWants))
	}
	for _, want := range kiwiWants {
		if !slices.Contains(kiwiOperations, want.id) {
			t.Errorf("kiwiOperations misses %s", want.id)
		}
	}
}

// TestAnyOfRequiresOneOfTheListedFields checks the new binder rule. The Kiwi update must change something, so a call with
// none of its four fields is refused with ErrMissingArgument, and a call with one of them binds.
func TestAnyOfRequiresOneOfTheListedFields(t *testing.T) {
	update, _ := Lookup(domain.OpKiwiTestCasesUpdate)
	if !slices.Equal(update.AnyOf, []string{"summary", "text", "status", "priority"}) {
		t.Fatalf("AnyOf = %v, want summary, text, status, priority", update.AnyOf)
	}
	if _, err := Bind(update, map[string]any{"documentId": "case-1"}); !errors.Is(err, domain.ErrMissingArgument) {
		t.Fatalf("update with no field error = %v, want ErrMissingArgument", err)
	}
	if _, err := Bind(update, map[string]any{"documentId": "case-1", "status": "Confirmed"}); err != nil {
		t.Fatalf("update with one field error = %v, want nil", err)
	}
	if err := Validate(update, domain.NewArgs(map[string]any{"documentId": "case-1", "priority": "P1"})); err != nil {
		t.Fatalf("Validate with one field error = %v, want nil", err)
	}
}

// TestAnyOfIsEmptyOnRowsThatDoNotNeedIt checks that a row without AnyOf keeps its old binding.
func TestAnyOfIsEmptyOnRowsThatDoNotNeedIt(t *testing.T) {
	get, _ := Lookup(domain.OpKiwiDocumentsGet)
	if len(get.AnyOf) != 0 {
		t.Fatalf("documents.get AnyOf = %v, want none", get.AnyOf)
	}
	if _, err := Bind(get, map[string]any{"documentId": "plan-1"}); err != nil {
		t.Fatalf("Bind(documents.get) error = %v", err)
	}
}

// TestResourcesListOutlineRowsOnly checks that the --tools words come from Outline rows alone, so no Kiwi word joins
// them.
func TestResourcesListOutlineRowsOnly(t *testing.T) {
	for _, word := range []string{"products", "test-cases", "plans"} {
		if slices.Contains(Resources(), word) {
			t.Fatalf("Resources() holds the Kiwi word %q", word)
		}
	}
	if !slices.Contains(Resources(), "collections") {
		t.Fatalf("Resources() = %v, want the Outline word collections", Resources())
	}
}
