package unit_test

import (
	"context"
	"errors"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-testkit/kiwistub"
)

// @scenario specs/libs/retriever-connector-kiwi/behaviours/identifiers.feature: A product ID of the canonical form is accepted
func TestCanonicalProductIDIsAccepted(t *testing.T) {
	for _, raw := range []string{"product-1", "product-123456789"} {
		t.Run(raw, func(t *testing.T) {
			d := sampleDataset()
			d.Products = append(d.Products, kiwistub.Product{ID: 123456789, Name: "Large"})
			r := newRig(t, d)
			_, err := treeOf(r.gw, raw)
			if errors.Is(err, domain.ErrInvalidCollectionID) {
				t.Fatalf("the canonical product ID %q was refused: %v", raw, err)
			}
			if r.stub.Calls("Product.filter") != 1 {
				t.Errorf("the product ID was not sent to the server")
			}
		})
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/identifiers.feature: A product ID that is not canonical is refused before any request
func TestNonCanonicalProductIDIsRefusedBeforeAnyRequest(t *testing.T) {
	for _, raw := range []string{"product-0", "product-01", "product-1234567890", "product--1", "PRODUCT-1", "product.1", "product-+1"} {
		t.Run(raw, func(t *testing.T) {
			r := newRig(t, sampleDataset())
			_, err := treeOf(r.gw, raw)
			if !errors.Is(err, domain.ErrInvalidCollectionID) {
				t.Errorf("the product ID %q was not refused as invalid: %v", raw, err)
			}
			if n := len(r.stub.Requests()); n != 0 {
				t.Errorf("%d requests were sent for an invalid product ID", n)
			}
		})
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/identifiers.feature: A document ID of the form <id> is accepted as a plan or a case
func TestDocumentIDOfPlanOrCaseFormIsAccepted(t *testing.T) {
	d := sampleDataset()
	d.Cases = append(d.Cases, kiwistub.Case{ID: 3, Summary: "Third", Sortkey: 3, Status: 1, Priority: 2, Category: 1})
	for _, raw := range []string{"plan-2", "case-3"} {
		t.Run(raw, func(t *testing.T) {
			doc, err := docOf(newRig(t, d).gw, raw)
			must(t, err)
			if doc.ID != raw {
				t.Errorf("the document ID is %q, want %q", doc.ID, raw)
			}
		})
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/identifiers.feature: A document ID in the wrong position is refused as an invalid document ID
func TestDocumentIDInWrongPositionIsRefused(t *testing.T) {
	rows := []struct{ id, position string }{{"case-3", "plan"}, {"plan-2", "case"}}
	for _, row := range rows {
		t.Run(row.id, func(t *testing.T) {
			planID, caseID := "plan-2", "case-12"
			if row.position == "plan" {
				planID = row.id
			} else {
				caseID = row.id
			}
			r := newRig(t, sampleDataset())
			_, err := r.gw.Write(context.Background(), domain.OpKiwiPlansAddCase,
				domain.NewArgs(map[string]any{"planId": planID, "caseId": caseID}))
			if !errors.Is(err, domain.ErrInvalidDocumentID) {
				t.Errorf("the ID %s in the %s position was not refused as an invalid document ID: %v", row.id, row.position, err)
			}
			if n := r.stub.Calls("TestPlan.add_case"); n != 0 {
				t.Errorf("%d add requests were sent for a misplaced ID", n)
			}
		})
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/identifiers.feature: The formatters give back the canonical text that the parsers accept
func TestFormattersGiveBackTheCanonicalTextTheParsersAccept(t *testing.T) {
	r := newRig(t, sampleDataset())
	cols, err := r.gw.ListCollections(context.Background())
	must(t, err)
	if len(cols) == 0 || cols[0].ID != "product-1" {
		t.Fatalf("the product ID is %v, want product-1", cols)
	}
	nodes, err := treeOf(r.gw, cols[0].ID)
	must(t, err)
	have := setOf(nodeIDs(nodes))
	for _, want := range []string{"plan-1", "plan-2", "case-12", "case-13"} {
		if !have[want] {
			t.Errorf("the tree does not name %s: %v", want, nodeIDs(nodes))
		}
	}
	for _, raw := range []string{"plan-2", "case-12"} {
		doc, err := docOf(r.gw, raw)
		must(t, err)
		if doc.ID != raw {
			t.Errorf("parsing %s returned %q", raw, doc.ID)
		}
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/identifiers.feature: A create that names a product ID in the wrong shape sends no lookup
// @scenario specs/libs/retriever-connector-kiwi/behaviours/writes.feature: A write whose product ID has the wrong shape is refused before any lookup is sent
func TestCreateWithWrongProductShapeSendsNoLookup(t *testing.T) {
	r := newRig(t, sampleDataset())
	_, err := r.gw.Write(context.Background(), domain.OpKiwiTestCasesCreate, domain.NewArgs(map[string]any{
		"summary": "New case", "product": "product-01", "category": "Functional", "priority": "P1",
	}))
	if !errors.Is(err, domain.ErrInvalidCollectionID) {
		t.Errorf("the create was not refused for its product ID: %v", err)
	}
	if n := len(r.stub.Requests()); n != 0 {
		t.Errorf("%d requests were sent, so a lookup went out", n)
	}
}
