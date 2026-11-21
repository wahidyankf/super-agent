//go:build integration

package integration_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-testkit/kiwistub"
)

// treeDataset is the Kiwi server of the tree bindings: one product, a parent plan with a child plan, and a case in each plan.
func treeDataset() kiwistub.Dataset {
	return kiwistub.Dataset{
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
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/reads-and-documents.feature: The product list returns every product in one request
func TestProductListReturnsEveryProductInOneRequest(t *testing.T) {
	srv := serveKiwi(t, newStub(sampleDataset()), nil)
	gw := gatewayAt(t, srv, staticCreds{})
	cols, err := gw.ListCollections(context.Background())
	must(t, err)
	if len(cols) != 2 || cols[0].Name != "Example Portal" || cols[1].Name != "Empty" {
		t.Errorf("the product list is %v, want both products", cols)
	}
	if n := len(srv.Stub.Requests()); n != 1 {
		t.Errorf("%d requests were sent, want 1", n)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/reads-and-documents.feature: The collection tree of a product lists each plan, then its cases, then its child plans
func TestCollectionTreeListsPlanThenCasesThenChildPlan(t *testing.T) {
	srv := serveKiwi(t, newStub(treeDataset()), nil)
	gw := gatewayAt(t, srv, staticCreds{})
	nodes, err := treeOf(gw, "product-1")
	must(t, err)
	want := []string{"plan-1", "case-12", "plan-2", "case-13"}
	if got := nodeIDs(nodes); fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("the nodes are %v, want %v", got, want)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/reads-and-documents.feature: A product with no plans has an empty collection tree and no error
func TestProductWithNoPlansHasEmptyTree(t *testing.T) {
	srv := serveKiwi(t, newStub(sampleDataset()), nil)
	gw := gatewayAt(t, srv, staticCreds{})
	nodes, err := treeOf(gw, "product-2")
	if err != nil || len(nodes) != 0 {
		t.Errorf("the tree is %v with error %v, want an empty tree and no error", nodes, err)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/reads-and-documents.feature: A product that does not exist is named as not found
func TestUnknownProductIsNamedAsNotFound(t *testing.T) {
	srv := serveKiwi(t, newStub(sampleDataset()), nil)
	gw := gatewayAt(t, srv, staticCreds{})
	_, err := treeOf(gw, "product-9")
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
	srv := serveKiwi(t, newStub(d), nil)
	gw := gatewayAt(t, srv, staticCreds{})
	_, err := treeOf(gw, "product-1")
	if !errors.Is(err, domain.ErrPageCapReached) {
		t.Errorf("the read failed with %v, want the page-cap error", err)
	}
	if n := srv.Stub.Calls("TestCase.filter") + srv.Stub.Calls("TestCase.sortkeys"); n != 0 {
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
	srv := serveKiwi(t, newStub(d), nil)
	gw := gatewayAt(t, srv, staticCreds{})
	nodes, err := treeOf(gw, "product-1")
	must(t, err)
	if got := nodeIDs(nodes); fmt.Sprint(got) != fmt.Sprint([]string{"plan-1", "plan-2"}) {
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
	srv := serveKiwi(t, newStub(d), nil)
	gw := gatewayAt(t, srv, staticCreds{})
	nodes, err := treeOf(gw, "product-1")
	must(t, err)
	if got := nodeIDs(nodes); fmt.Sprint(got) != fmt.Sprint([]string{"plan-1", "case-12", "case-13"}) {
		t.Errorf("the nodes are %v, want the cases in ID order", got)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/reads-and-documents.feature: The document of a plan holds its ID, title, collection, and the cases it lists
func TestPlanDocumentHoldsIDTitleCollectionAndCases(t *testing.T) {
	srv := serveKiwi(t, newStub(sampleDataset()), nil)
	gw := gatewayAt(t, srv, staticCreds{})
	doc, err := docOf(gw, "plan-2")
	must(t, err)
	if doc.ID != "plan-2" || doc.Title != "Login" || doc.CollectionID != "product-1" {
		t.Errorf("the plan document holds ID %q, title %q, collection %q", doc.ID, doc.Title, doc.CollectionID)
	}
	for _, want := range []string{"- case-12: Login works", "- case-13: Logout works"} {
		if !containsText(doc.Markdown, want) {
			t.Errorf("the plan document does not list %q", want)
		}
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/reads-and-documents.feature: The document of a case names its status, priority, and category by name
func TestCaseDocumentNamesStatusPriorityAndCategory(t *testing.T) {
	srv := serveKiwi(t, newStub(sampleDataset()), nil)
	gw := gatewayAt(t, srv, staticCreds{})
	doc, err := docOf(gw, "case-12")
	must(t, err)
	for _, want := range []string{"- Status: CONFIRMED", "- Priority: P1", "- Category: Functional"} {
		if !containsText(doc.Markdown, want) {
			t.Errorf("the case document does not hold %q", want)
		}
	}
	if doc.CollectionID != "product-1" {
		t.Errorf("the product of the category is %q, want product-1", doc.CollectionID)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/reads-and-documents.feature: A document read for a missing <kind> is named as not found
func TestDocumentReadForMissingKindIsNotFound(t *testing.T) {
	for _, raw := range []string{"plan-99", "case-99"} {
		t.Run(raw, func(t *testing.T) {
			srv := serveKiwi(t, newStub(sampleDataset()), nil)
			gw := gatewayAt(t, srv, staticCreds{})
			_, err := docOf(gw, raw)
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
	srv := serveKiwi(t, newStub(d), nil)
	gw := gatewayAt(t, srv, staticCreds{})
	_, err := docOf(gw, "case-12")
	must(t, err)
	_, err = docOf(gw, "case-13")
	must(t, err)
	for _, m := range []string{"Category.filter", "Priority.filter", "TestCaseStatus.filter"} {
		if n := srv.Stub.Calls(m); n != 1 {
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
			srv := serveKiwi(t, newStub(d), nil)
			gw := gatewayAt(t, srv, staticCreds{})
			_, err := treeOf(gw, "product-1")
			if !errors.Is(err, domain.ErrUnexpectedResponse) {
				t.Errorf("the read failed with %v, want the unexpected-response error", err)
			}
		})
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/reads-and-documents.feature: A fault in any reply that a read needs is returned and the read goes no further
func TestFaultInLaterReplyIsReturnedAndReadGoesNoFurther(t *testing.T) {
	srv := serveKiwi(t, newStub(sampleDataset()), nil)
	srv.Stub.SetMethodFault("TestPlan.filter", kiwistub.Fault{Status: 500})
	gw := gatewayAt(t, srv, staticCreds{})
	_, err := treeOf(gw, "product-1")
	if !errors.Is(err, domain.ErrUpstreamFailure) {
		t.Errorf("the read returned %v, want the upstream fault", err)
	}
	if n := srv.Stub.Calls("TestCase.filter") + srv.Stub.Calls("TestCase.sortkeys"); n != 0 {
		t.Errorf("%d later replies were requested", n)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/reads-and-documents.feature: A reply that is not an object of case IDs to whole numbers fails the read as unexpected
func TestSortKeyReplyThatIsNotAnObjectOfWholeNumbersFailsAsUnexpected(t *testing.T) {
	shapes := []string{`[1,2]`, `5`, `{"abc":1}`, `{"12":1.5}`}
	for _, shape := range shapes {
		t.Run(shape, func(t *testing.T) {
			mux := &rpcMux{results: map[string]string{"TestCase.sortkeys": shape}}
			srv := serveKiwi(t, newStub(sampleDataset()), mux.wrap)
			gw := gatewayAt(t, srv, staticCreds{})
			_, err := treeOf(gw, "product-1")
			if !errors.Is(err, domain.ErrUnexpectedResponse) {
				t.Errorf("the sort-key reply %s failed with %v, want the unexpected-response error", shape, err)
			}
		})
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/reads-and-documents.feature: An unknown field in a row of the reply is ignored and a null where text is expected keeps the zero value
func TestUnknownFieldIgnoredAndNullTextKeepsZeroValue(t *testing.T) {
	mux := &rpcMux{results: map[string]string{"Product.filter": `[{"id":1,"name":null,"extra":"ignored"}]`}}
	srv := serveKiwi(t, newStub(sampleDataset()), mux.wrap)
	gw := gatewayAt(t, srv, staticCreds{})
	cols, err := gw.ListCollections(context.Background())
	must(t, err)
	if len(cols) != 1 || cols[0].ID != "product-1" || cols[0].Name != "" {
		t.Errorf("the rows are %v, want one product with an empty name", cols)
	}
}

// containsText reports whether text holds want.
func containsText(text, want string) bool {
	return strings.Contains(text, want)
}
