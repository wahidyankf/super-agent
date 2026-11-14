package kiwi

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/tests/support/kiwistub"
)

// mustDocumentID parses a plan or case ID for a test.
func mustDocumentID(t *testing.T, raw string) domain.DocumentID {
	t.Helper()
	id, err := domain.NewDocumentID(raw)
	if err != nil {
		t.Fatalf("NewDocumentID(%q): %v", raw, err)
	}
	return id
}

// mustCollectionID parses a product ID for a test.
func mustCollectionID(t *testing.T, raw string) domain.CollectionID {
	t.Helper()
	id, err := domain.NewCollectionID(raw)
	if err != nil {
		t.Fatalf("NewCollectionID(%q): %v", raw, err)
	}
	return id
}

// describeTree renders each node as "id depth=N parent=P", for an exact comparison of the flat tree.
func describeTree(nodes []domain.TreeNode) []string {
	out := make([]string, 0, len(nodes))
	for _, n := range nodes {
		parent := "-"
		if n.ParentID != nil {
			parent = *n.ParentID
		}
		out = append(out, fmt.Sprintf("%s depth=%d parent=%s", n.ID, n.Depth, parent))
	}
	return out
}

// TestListCollectionsReturnsEveryProductInOneRequest checks that the products come back in one Product.filter call.
func TestListCollectionsReturnsEveryProductInOneRequest(t *testing.T) {
	r := newRig(t, sampleDataset())
	got, err := r.gw.ListCollections(context.Background())
	if err != nil {
		t.Fatalf("ListCollections: %v", err)
	}
	want := []domain.Collection{{ID: "product-1", Name: "Example Portal"}, {ID: "product-2", Name: "Empty"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("collections = %+v, want %+v", got, want)
	}
	if r.stub.Calls(methodProductFilter) != 1 {
		t.Fatalf("Product.filter calls = %d, want 1", r.stub.Calls(methodProductFilter))
	}
}

// TestCollectionTreeFlattensPlansCasesAndChildren checks the depth-first order: a plan, its cases, then its child plans.
func TestCollectionTreeFlattensPlansCasesAndChildren(t *testing.T) {
	r := newRig(t, sampleDataset())
	nodes, err := r.gw.CollectionTree(context.Background(), mustCollectionID(t, "product-1"))
	if err != nil {
		t.Fatalf("CollectionTree: %v", err)
	}
	want := []string{
		"plan-1 depth=0 parent=-",
		"plan-2 depth=1 parent=plan-1",
		"case-12 depth=2 parent=plan-2",
		"case-13 depth=2 parent=plan-2",
	}
	if got := describeTree(nodes); !reflect.DeepEqual(got, want) {
		t.Fatalf("tree = %q, want %q", got, want)
	}
	if !strings.HasSuffix(nodes[0].URL, "/plan/1/") || !strings.HasSuffix(nodes[2].URL, "/case/12/") {
		t.Fatalf("URLs = %q and %q, want plan and case paths", nodes[0].URL, nodes[2].URL)
	}
}

// TestCollectionTreeOfAProductWithNoPlansIsEmpty checks an empty product: no error, no nodes.
func TestCollectionTreeOfAProductWithNoPlansIsEmpty(t *testing.T) {
	r := newRig(t, sampleDataset())
	nodes, err := r.gw.CollectionTree(context.Background(), mustCollectionID(t, "product-2"))
	if err != nil || len(nodes) != 0 {
		t.Fatalf("empty product = (%d nodes, %v), want no nodes and no error", len(nodes), err)
	}
}

// TestCollectionTreeOfAnUnknownProductIsNotFound checks that a product that does not exist is named as such.
func TestCollectionTreeOfAnUnknownProductIsNotFound(t *testing.T) {
	r := newRig(t, sampleDataset())
	_, err := r.gw.CollectionTree(context.Background(), mustCollectionID(t, "product-999"))
	if !errors.Is(err, domain.ErrCollectionNotFound) {
		t.Fatalf("error = %v, want ErrCollectionNotFound", err)
	}
}

// TestCollectionTreeStopsBeforeAnyCaseWhenTooManyPlans checks the 200-plan cap: the tree is refused before any case read.
func TestCollectionTreeStopsBeforeAnyCaseWhenTooManyPlans(t *testing.T) {
	d := sampleDataset()
	d.Plans = nil
	for i := 1; i <= planCap+1; i++ {
		d.Plans = append(d.Plans, kiwistub.Plan{ID: i, Name: "Plan", Product: 1})
	}
	r := newRig(t, d)
	_, err := r.gw.CollectionTree(context.Background(), mustCollectionID(t, "product-1"))
	if !errors.Is(err, domain.ErrPageCapReached) {
		t.Fatalf("error = %v, want ErrPageCapReached", err)
	}
	if r.stub.Calls(methodTestCaseFilter) != 0 {
		t.Fatalf("cases were read after the cap was hit")
	}
}

// TestPlansInAParentCycleAreStillListed checks that a loop of parent links hides no plan: each plan is listed once.
func TestPlansInAParentCycleAreStillListed(t *testing.T) {
	d := sampleDataset()
	d.Plans = []kiwistub.Plan{
		{ID: 1, Name: "Alpha", Product: 1, Parent: 2},
		{ID: 2, Name: "Beta", Product: 1, Parent: 1},
	}
	d.Cases = nil
	r := newRig(t, d)
	nodes, err := r.gw.CollectionTree(context.Background(), mustCollectionID(t, "product-1"))
	if err != nil {
		t.Fatalf("CollectionTree: %v", err)
	}
	want := []string{"plan-1 depth=0 parent=-", "plan-2 depth=1 parent=plan-1"}
	if got := describeTree(nodes); !reflect.DeepEqual(got, want) {
		t.Fatalf("tree = %q, want %q", got, want)
	}
}

// TestDocumentOfAPlanHoldsItsCases checks a plan's document: its ID, title, collection, and the cases it lists.
func TestDocumentOfAPlanHoldsItsCases(t *testing.T) {
	r := newRig(t, sampleDataset())
	doc, err := r.gw.Document(context.Background(), mustDocumentID(t, "plan-2"))
	if err != nil {
		t.Fatalf("Document: %v", err)
	}
	if doc.ID != "plan-2" || doc.Title != "Login" || doc.CollectionID != "product-1" || !strings.HasSuffix(doc.URL, "/plan/2/") {
		t.Fatalf("document = %+v, want plan-2 of product-1", doc)
	}
	for _, want := range []string{"case-12", "Login works", "case-13", "Logout works"} {
		if !strings.Contains(doc.Markdown, want) {
			t.Fatalf("Markdown lacks %q:\n%s", want, doc.Markdown)
		}
	}
}

// TestDocumentOfACaseNamesItsLookups checks that a case shows its status, priority, and category by name, and that its
// product is the product of its category.
func TestDocumentOfACaseNamesItsLookups(t *testing.T) {
	r := newRig(t, sampleDataset())
	doc, err := r.gw.Document(context.Background(), mustDocumentID(t, "case-12"))
	if err != nil {
		t.Fatalf("Document: %v", err)
	}
	if doc.ID != "case-12" || doc.Title != "Login works" || doc.CollectionID != "product-1" || !strings.HasSuffix(doc.URL, "/case/12/") {
		t.Fatalf("document = %+v, want case-12 of product-1", doc)
	}
	for _, want := range []string{"Open the login page.", "CONFIRMED", "P1", "Functional"} {
		if !strings.Contains(doc.Markdown, want) {
			t.Fatalf("Markdown lacks %q:\n%s", want, doc.Markdown)
		}
	}
}

// TestUnknownDocumentsAreNotFound checks that a missing plan and a missing case are both named as not found.
func TestUnknownDocumentsAreNotFound(t *testing.T) {
	r := newRig(t, sampleDataset())
	for _, raw := range []string{"plan-999", "case-999"} {
		_, err := r.gw.Document(context.Background(), mustDocumentID(t, raw))
		if !errors.Is(err, domain.ErrDocumentNotFound) {
			t.Fatalf("Document(%s) error = %v, want ErrDocumentNotFound", raw, err)
		}
	}
}

// TestLookupTablesAreReadOncePerGateway checks the cache: two case reads read each lookup table once.
func TestLookupTablesAreReadOncePerGateway(t *testing.T) {
	r := newRig(t, sampleDataset())
	for _, raw := range []string{"case-12", "case-13"} {
		if _, err := r.gw.Document(context.Background(), mustDocumentID(t, raw)); err != nil {
			t.Fatalf("Document(%s): %v", raw, err)
		}
	}
	for _, m := range []string{methodCategoryFilter, methodPriorityFilter, methodTestCaseStatusFilter} {
		if got := r.stub.Calls(m); got != 1 {
			t.Fatalf("%s calls = %d, want 1", m, got)
		}
	}
}

// TestReadProductsListRecord checks the record of kiwi.products.list: one item per product.
func TestReadProductsListRecord(t *testing.T) {
	r := newRig(t, sampleDataset())
	rec, err := r.gw.Read(context.Background(), domain.OpKiwiProductsList, domain.NewArgs(map[string]any{}))
	if err != nil {
		t.Fatalf("Read products list: %v", err)
	}
	items := rec.Items()
	if !rec.IsList() || len(items) != 2 || items[0].Str("id") != "product-1" || items[0].Str("name") != "Example Portal" {
		t.Fatalf("products list = %+v", items)
	}
}

// TestReadProductsTreeRecord checks the record of kiwi.products.tree: each item has its parent and its depth.
func TestReadProductsTreeRecord(t *testing.T) {
	r := newRig(t, sampleDataset())
	rec, err := r.gw.Read(context.Background(), domain.OpKiwiProductsTree, domain.NewArgs(map[string]any{"collectionId": "product-1"}))
	if err != nil {
		t.Fatalf("Read products tree: %v", err)
	}
	items := rec.Items()
	if len(items) != 4 || items[0].Str("id") != "plan-1" || items[0].Str("parentId") != "" {
		t.Fatalf("tree items = %+v", items)
	}
	if items[1].Str("id") != "plan-2" || items[1].Str("parentId") != "plan-1" {
		t.Fatalf("child plan = %+v, want plan-2 under plan-1", items[1])
	}
}

// TestReadDocumentsGetRecord checks the record of kiwi.documents.get for one case.
func TestReadDocumentsGetRecord(t *testing.T) {
	r := newRig(t, sampleDataset())
	rec, err := r.gw.Read(context.Background(), domain.OpKiwiDocumentsGet, domain.NewArgs(map[string]any{"documentId": "case-12"}))
	if err != nil {
		t.Fatalf("Read documents get: %v", err)
	}
	if rec.Str("id") != "case-12" || rec.Str("collectionId") != "product-1" || !strings.Contains(rec.Str("markdown"), "Open the login page.") {
		t.Fatalf("document record = %+v", rec.Fields())
	}
}

// TestReadRefusesAWriteOperation checks that the read path refuses a write operation before any request.
func TestReadRefusesAWriteOperation(t *testing.T) {
	r := newRig(t, sampleDataset())
	_, err := r.gw.Read(context.Background(), domain.OpKiwiPlansAddCase, domain.NewArgs(map[string]any{"planId": "plan-2", "caseId": "case-13"}))
	if !errors.Is(err, domain.ErrWrongOperationClass) {
		t.Fatalf("error = %v, want ErrWrongOperationClass", err)
	}
	if len(r.stub.Requests()) != 0 {
		t.Fatalf("a request was sent for a refused read")
	}
}

// TestRepliesWithAnIDOutsideTheIDRangeAreRefused checks that a plan or case ID in a reply is validated before it forms a
// plan-<n> or case-<n> ID: zero, a negative number, and a number of more than nine digits are an unexpected response, in
// the tree and in a plan document.
func TestRepliesWithAnIDOutsideTheIDRangeAreRefused(t *testing.T) {
	for _, bad := range []int{0, -5, 1000000000} {
		planBad := sampleDataset()
		planBad.Plans = append(planBad.Plans, kiwistub.Plan{ID: bad, Name: "Odd", Product: 1})
		caseBad := sampleDataset()
		caseBad.Cases = append(caseBad.Cases, kiwistub.Case{ID: bad, Summary: "Odd", Plans: []int{2}, Sortkey: 3, Status: 1, Priority: 2, Category: 1})
		for name, d := range map[string]kiwistub.Dataset{"plan": planBad, "case": caseBad} {
			r := newRig(t, d)
			_, err := r.gw.CollectionTree(context.Background(), mustCollectionID(t, "product-1"))
			if !errors.Is(err, domain.ErrUnexpectedResponse) {
				t.Fatalf("tree with a %s ID %d: error = %v, want ErrUnexpectedResponse", name, bad, err)
			}
		}
		r := newRig(t, caseBad)
		if _, err := r.gw.Document(context.Background(), mustDocumentID(t, "plan-2")); !errors.Is(err, domain.ErrUnexpectedResponse) {
			t.Fatalf("plan document with a case ID %d: error = %v, want ErrUnexpectedResponse", bad, err)
		}
	}
}
