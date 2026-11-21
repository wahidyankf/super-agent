package unit_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-testkit/kiwistub"
)

// @scenario specs/libs/retriever-connector-kiwi/behaviours/reads-and-documents.feature: The product list returns every product in one request
func TestProductListReturnsEveryProductInOneRequest(t *testing.T) {
	r := newRig(t, sampleDataset())
	cols, err := r.gw.ListCollections(context.Background())
	must(t, err)
	if len(cols) != 2 || cols[0].Name != "Example Portal" || cols[1].Name != "Empty" {
		t.Errorf("the product list is %v, want both products", cols)
	}
	if n := len(r.stub.Requests()); n != 1 {
		t.Errorf("%d requests were sent, want 1", n)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/reads-and-documents.feature: The collection tree of a product lists each plan, then its cases, then its child plans
func TestCollectionTreeListsPlanThenCasesThenChildPlan(t *testing.T) {
	d := kiwistub.Dataset{
		Products: []kiwistub.Product{{ID: 1, Name: "Example Portal"}},
		Plans: []kiwistub.Plan{
			{ID: 1, Name: "Smoke", Product: 1},
			{ID: 2, Name: "Login", Product: 1, Parent: 1},
		},
		Cases: []kiwistub.Case{
			{ID: 12, Summary: "Login works", Plans: []int{1}, Sortkey: 1, Status: 1, Priority: 2, Category: 1},
			{ID: 13, Summary: "Logout works", Plans: []int{2}, Sortkey: 1, Status: 1, Priority: 2, Category: 1},
		},
		Categories: []kiwistub.Category{{ID: 1, Name: "Functional", Product: 1}},
		Priorities: []kiwistub.Priority{{ID: 2, Value: "P1"}},
		Statuses:   []kiwistub.Status{{ID: 1, Name: "CONFIRMED"}},
	}
	nodes, err := treeOf(newRig(t, d).gw, "product-1")
	must(t, err)
	want := []string{"plan-1", "case-12", "plan-2", "case-13"}
	got := nodeIDs(nodes)
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("the nodes are %v, want %v", got, want)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/reads-and-documents.feature: A product with no plans has an empty collection tree and no error
func TestProductWithNoPlansHasEmptyTree(t *testing.T) {
	nodes, err := treeOf(newRig(t, sampleDataset()).gw, "product-2")
	if err != nil || len(nodes) != 0 {
		t.Errorf("the tree is %v with error %v, want an empty tree and no error", nodes, err)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/reads-and-documents.feature: A product that does not exist is named as not found
func TestUnknownProductIsNamedAsNotFound(t *testing.T) {
	_, err := treeOf(newRig(t, sampleDataset()).gw, "product-9")
	if !errors.Is(err, domain.ErrCollectionNotFound) {
		t.Errorf("the read failed with %v, want the not-found error", err)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/reads-and-documents.feature: A collection tree with more than 200 plans is refused before any case is read
func TestCollectionTreeOverTwoHundredPlansIsRefused(t *testing.T) {
	d := kiwistub.Dataset{Products: []kiwistub.Product{{ID: 1, Name: "Big"}}}
	for i := 1; i <= 201; i++ {
		d.Plans = append(d.Plans, kiwistub.Plan{ID: i, Name: fmt.Sprintf("Plan %d", i), Product: 1})
	}
	r := newRig(t, d)
	_, err := treeOf(r.gw, "product-1")
	if !errors.Is(err, domain.ErrPageCapReached) {
		t.Errorf("the read failed with %v, want the page-cap error", err)
	}
	if n := r.stub.Calls("TestCase.filter") + r.stub.Calls("TestCase.sortkeys"); n != 0 {
		t.Errorf("%d case reads were sent", n)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/reads-and-documents.feature: Plans whose parent links form a loop are each listed once
func TestPlansWithParentLoopAreEachListedOnce(t *testing.T) {
	d := kiwistub.Dataset{
		Products: []kiwistub.Product{{ID: 1, Name: "Example Portal"}},
		Plans: []kiwistub.Plan{
			{ID: 1, Name: "A", Product: 1, Parent: 2},
			{ID: 2, Name: "B", Product: 1, Parent: 1},
		},
	}
	nodes, err := treeOf(newRig(t, d).gw, "product-1")
	must(t, err)
	got := nodeIDs(nodes)
	if fmt.Sprint(got) != fmt.Sprint([]string{"plan-1", "plan-2"}) {
		t.Errorf("the loop is listed as %v, want each plan once", got)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/reads-and-documents.feature: Two cases with the same sort key are listed in ID order whatever order the server sent them in
func TestCasesWithSameSortKeyAreListedInIDOrder(t *testing.T) {
	d := kiwistub.Dataset{
		Products: []kiwistub.Product{{ID: 1, Name: "Example Portal"}},
		Plans:    []kiwistub.Plan{{ID: 1, Name: "Smoke", Product: 1}},
		Cases: []kiwistub.Case{
			{ID: 13, Summary: "Second", Plans: []int{1}, Sortkey: 5, Status: 1, Priority: 2, Category: 1},
			{ID: 12, Summary: "First", Plans: []int{1}, Sortkey: 5, Status: 1, Priority: 2, Category: 1},
		},
		Categories: []kiwistub.Category{{ID: 1, Name: "Functional", Product: 1}},
		Priorities: []kiwistub.Priority{{ID: 2, Value: "P1"}},
		Statuses:   []kiwistub.Status{{ID: 1, Name: "CONFIRMED"}},
	}
	nodes, err := treeOf(newRig(t, d).gw, "product-1")
	must(t, err)
	got := nodeIDs(nodes)
	if fmt.Sprint(got) != fmt.Sprint([]string{"plan-1", "case-12", "case-13"}) {
		t.Errorf("the nodes are %v, want the cases in ID order", got)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/reads-and-documents.feature: The document of a plan holds its ID, title, collection, and the cases it lists
func TestPlanDocumentHoldsIDTitleCollectionAndCases(t *testing.T) {
	doc, err := docOf(newRig(t, sampleDataset()).gw, "plan-2")
	must(t, err)
	if doc.ID != "plan-2" || doc.Title != "Login" || doc.CollectionID != "product-1" {
		t.Errorf("the plan document holds ID %q, title %q, collection %q", doc.ID, doc.Title, doc.CollectionID)
	}
	mustContain(t, doc.Markdown, "- case-12: Login works", "- case-13: Logout works")
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/reads-and-documents.feature: The document of a case names its status, priority, and category by name
func TestCaseDocumentNamesStatusPriorityAndCategory(t *testing.T) {
	doc, err := docOf(newRig(t, sampleDataset()).gw, "case-12")
	must(t, err)
	mustContain(t, doc.Markdown, "- Status: CONFIRMED", "- Priority: P1", "- Category: Functional")
	if doc.CollectionID != "product-1" {
		t.Errorf("the product of the category is %q, want product-1", doc.CollectionID)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/reads-and-documents.feature: A document read for a missing <kind> is named as not found
func TestDocumentReadForMissingKindIsNotFound(t *testing.T) {
	for _, raw := range []string{"plan-99", "case-99"} {
		t.Run(raw, func(t *testing.T) {
			_, err := docOf(newRig(t, sampleDataset()).gw, raw)
			if !errors.Is(err, domain.ErrDocumentNotFound) {
				t.Errorf("the read of %s failed with %v, want the not-found error", raw, err)
			}
		})
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/reads-and-documents.feature: The lookup tables of the case documents are read once for the whole gateway
func TestLookupTablesAreReadOnceForTheWholeGateway(t *testing.T) {
	d := sampleDataset()
	d.Cases = append(d.Cases, kiwistub.Case{ID: 14, Summary: "Extra", Plans: []int{2}, Sortkey: 3, Status: 1, Priority: 2, Category: 1})
	r := newRig(t, d)
	_, err := docOf(r.gw, "case-12")
	must(t, err)
	_, err = docOf(r.gw, "case-13")
	must(t, err)
	for _, m := range []string{"Category.filter", "Priority.filter", "TestCaseStatus.filter"} {
		if n := r.stub.Calls(m); n != 1 {
			t.Errorf("%s was read %d times, want once", m, n)
		}
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/reads-and-documents.feature: A reply that names a plan or case ID outside the ID range fails the read as unexpected
func TestReplyIDOutsideRangeFailsAsUnexpected(t *testing.T) {
	for _, id := range []int{0, -1, 1000000000} {
		t.Run(fmt.Sprint(id), func(t *testing.T) {
			d := kiwistub.Dataset{
				Products: []kiwistub.Product{{ID: 1, Name: "Example Portal"}},
				Plans:    []kiwistub.Plan{{ID: id, Name: "Bad", Product: 1}},
			}
			_, err := treeOf(newRig(t, d).gw, "product-1")
			if !errors.Is(err, domain.ErrUnexpectedResponse) {
				t.Errorf("the read failed with %v, want the unexpected-response error", err)
			}
		})
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/reads-and-documents.feature: A read that names an operation outside the Kiwi read table is refused before any request
func TestReadOutsideKiwiReadTableIsRefusedBeforeAnyRequest(t *testing.T) {
	cases := []struct {
		name string
		op   domain.OperationID
		args map[string]any
	}{
		{"an unknown operation", "kiwi.unknown", map[string]any{}},
		{"a write", domain.OpKiwiTestCasesCreate, map[string]any{}},
		{"a read of another kind", domain.OpDocumentsGet, map[string]any{"documentId": "x"}},
		{"a read with a bad argument", domain.OpKiwiProductsTree, map[string]any{"collectionId": "product-0"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := newRig(t, sampleDataset())
			_, err := r.gw.Read(context.Background(), c.op, domain.NewArgs(c.args))
			if err == nil {
				t.Errorf("the read was not refused")
			}
			if n := len(r.stub.Requests()); n != 0 {
				t.Errorf("%d requests were sent for a refused read", n)
			}
		})
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/reads-and-documents.feature: A fault in any reply that a read needs is returned and the read goes no further
func TestFaultInLaterReplyIsReturnedAndReadGoesNoFurther(t *testing.T) {
	r := newRig(t, sampleDataset())
	r.stub.SetMethodFault("TestPlan.filter", kiwistub.Fault{Status: 500})
	_, err := treeOf(r.gw, "product-1")
	if !errors.Is(err, domain.ErrUpstreamFailure) {
		t.Errorf("the read returned %v, want the upstream fault", err)
	}
	if n := r.stub.Calls("TestCase.filter") + r.stub.Calls("TestCase.sortkeys"); n != 0 {
		t.Errorf("%d later replies were requested", n)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/reads-and-documents.feature: A reply that is not an object of case IDs to whole numbers fails the read as unexpected
func TestSortKeyReplyThatIsNotAnObjectOfWholeNumbersFailsAsUnexpected(t *testing.T) {
	shapes := []string{`[1,2]`, `5`, `{"abc":1}`, `{"12":1.5}`}
	for _, shape := range shapes {
		t.Run(shape, func(t *testing.T) {
			stub := kiwistub.New(sentinelKey, kiwistub.WithDataset(sampleDataset()))
			rt := shaped{next: stub.RoundTripper(), results: map[string]string{"TestCase.sortkeys": shape}}
			gw := hermeticGateway(t, rt, staticCreds{}, newFakeClock())
			_, err := treeOf(gw, "product-1")
			if !errors.Is(err, domain.ErrUnexpectedResponse) {
				t.Errorf("the sort-key reply %s failed with %v, want the unexpected-response error", shape, err)
			}
		})
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/reads-and-documents.feature: An unknown field in a row of the reply is ignored and a null where text is expected keeps the zero value
func TestUnknownFieldIgnoredAndNullTextKeepsZeroValue(t *testing.T) {
	stub := kiwistub.New(sentinelKey, kiwistub.WithDataset(sampleDataset()))
	rt := shaped{next: stub.RoundTripper(), results: map[string]string{
		"Product.filter": `[{"id":1,"name":null,"extra":"ignored"}]`,
	}}
	gw := hermeticGateway(t, rt, staticCreds{}, newFakeClock())
	cols, err := gw.ListCollections(context.Background())
	must(t, err)
	if len(cols) != 1 || cols[0].ID != "product-1" || cols[0].Name != "" {
		t.Errorf("the rows are %v, want one product with an empty name", cols)
	}
}
