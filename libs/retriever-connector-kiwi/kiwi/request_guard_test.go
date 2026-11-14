package kiwi

import (
	"context"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// TestEveryRequestPassesTheArgumentGuardAndEveryMethodIsExercised drives every request path of the adapter through one
// gateway. Each recorded request must pass the argument guard, and all ten allowed methods must be used.
func TestEveryRequestPassesTheArgumentGuardAndEveryMethodIsExercised(t *testing.T) {
	r := newRig(t, sampleDataset())
	ctx := context.Background()

	if _, err := r.gw.ListCollections(ctx); err != nil {
		t.Fatalf("ListCollections: %v", err)
	}
	if _, err := r.gw.CollectionTree(ctx, mustCollectionID(t, "product-1")); err != nil {
		t.Fatalf("CollectionTree: %v", err)
	}
	if _, err := r.gw.Document(ctx, mustDocumentID(t, "plan-2")); err != nil {
		t.Fatalf("Document plan: %v", err)
	}
	if _, err := r.gw.Document(ctx, mustDocumentID(t, "case-12")); err != nil {
		t.Fatalf("Document case: %v", err)
	}
	reads := []struct {
		op   domain.OperationID
		args map[string]any
	}{
		{domain.OpKiwiProductsList, map[string]any{}},
		{domain.OpKiwiProductsTree, map[string]any{"collectionId": "product-1"}},
		{domain.OpKiwiDocumentsGet, map[string]any{"documentId": "case-12"}},
	}
	for _, rd := range reads {
		if _, err := r.gw.Read(ctx, rd.op, domain.NewArgs(rd.args)); err != nil {
			t.Fatalf("Read %s: %v", rd.op, err)
		}
	}
	if _, err := r.gw.Write(ctx, domain.OpKiwiTestCasesCreate, domain.NewArgs(map[string]any{
		"summary": "New case", "product": "product-1", "category": "Functional", "priority": "P1",
		"status": "CONFIRMED", "text": "Body",
	})); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := r.gw.Write(ctx, domain.OpKiwiTestCasesUpdate, domain.NewArgs(map[string]any{
		"documentId": "case-12", "summary": "Renamed", "status": "CONFIRMED", "priority": "P1",
	})); err != nil {
		t.Fatalf("update: %v", err)
	}
	if _, err := r.gw.Write(ctx, domain.OpKiwiPlansAddCase, domain.NewArgs(map[string]any{
		"planId": "plan-1", "caseId": "case-13",
	})); err != nil {
		t.Fatalf("add-case: %v", err)
	}

	for _, req := range r.stub.Requests() {
		if err := guardArguments(req.Params); err != nil {
			t.Fatalf("request %s failed the argument guard: %v", req.Method, err)
		}
	}
	allMethods := []string{
		methodProductFilter, methodTestPlanFilter, methodTestCaseFilter, methodTestCaseSortkeys,
		methodCategoryFilter, methodPriorityFilter, methodTestCaseStatusFilter,
		methodTestCaseCreate, methodTestCaseUpdate, methodTestPlanAddCase,
	}
	for _, m := range allMethods {
		if r.stub.Calls(m) == 0 {
			t.Fatalf("method %s was never exercised", m)
		}
		if !allowedMethods[m] {
			t.Fatalf("method %s is not in the allowlist", m)
		}
	}
}
