package kiwi

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// lastValues returns the "values" object of the last request with the given method, for the create and update checks.
func lastValues(t *testing.T, r rig, method string) map[string]any {
	t.Helper()
	var found map[string]any
	for _, req := range r.stub.Requests() {
		if req.Method != method {
			continue
		}
		v, ok := req.Params["values"].(map[string]any)
		if !ok {
			t.Fatalf("%s params = %+v, want a values object", method, req.Params)
		}
		found = v
	}
	if found == nil {
		t.Fatalf("no %s request was recorded", method)
	}
	return found
}

// lastParams returns the params of the last request with the given method.
func lastParams(t *testing.T, r rig, method string) map[string]any {
	t.Helper()
	var found map[string]any
	for _, req := range r.stub.Requests() {
		if req.Method == method {
			found = req.Params
		}
	}
	if found == nil {
		t.Fatalf("no %s request was recorded", method)
	}
	return found
}

// TestCreateCaseResolvesItsNamesAndSendsOneWrite checks create: the names become IDs, and one TestCase.create is sent.
func TestCreateCaseResolvesItsNamesAndSendsOneWrite(t *testing.T) {
	r := newRig(t, sampleDataset())
	rec, err := r.gw.Write(context.Background(), domain.OpKiwiTestCasesCreate, domain.NewArgs(map[string]any{
		"summary": "New case", "product": "product-1", "category": "Functional", "priority": "P1",
	}))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if r.stub.Calls(methodTestCaseCreate) != 1 {
		t.Fatalf("TestCase.create calls = %d, want 1", r.stub.Calls(methodTestCaseCreate))
	}
	values := lastValues(t, r, methodTestCaseCreate)
	if values["summary"] != "New case" || fmt.Sprint(values["category"]) != "1" || fmt.Sprint(values["priority"]) != "2" {
		t.Fatalf("create values = %+v, want the summary and the resolved IDs", values)
	}
	if _, has := values["case_status"]; has {
		t.Fatalf("create sent a status that was not named")
	}
	if !strings.HasPrefix(rec.Str("id"), "case-") || rec.Str("title") != "New case" {
		t.Fatalf("create record = %+v", rec.Fields())
	}
}

// TestCreateCaseSendsTheOptionalStatusAndText checks that a named status resolves to its ID, and the text is sent.
func TestCreateCaseSendsTheOptionalStatusAndText(t *testing.T) {
	r := newRig(t, sampleDataset())
	_, err := r.gw.Write(context.Background(), domain.OpKiwiTestCasesCreate, domain.NewArgs(map[string]any{
		"summary": "New case", "product": "product-1", "category": "Functional", "priority": "P1",
		"status": "CONFIRMED", "text": "Body",
	}))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	values := lastValues(t, r, methodTestCaseCreate)
	if fmt.Sprint(values["case_status"]) != "1" || values["text"] != "Body" {
		t.Fatalf("create values = %+v, want case_status 1 and the text", values)
	}
}

// TestCreateCaseWithAmbiguousOrUnknownNamesSendsNoWrite checks that a name matching zero or several rows is refused before
// the write.
func TestCreateCaseWithAmbiguousOrUnknownNamesSendsNoWrite(t *testing.T) {
	cases := map[string]map[string]any{
		"ambiguous category": {"summary": "New", "product": "product-1", "category": "Regression", "priority": "P1"},
		"unknown category":   {"summary": "New", "product": "product-1", "category": "Nope", "priority": "P1"},
		"unknown priority":   {"summary": "New", "product": "product-1", "category": "Functional", "priority": "P9"},
		"unknown status":     {"summary": "New", "product": "product-1", "category": "Functional", "priority": "P1", "status": "NOPE"},
	}
	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			r := newRig(t, sampleDataset())
			_, err := r.gw.Write(context.Background(), domain.OpKiwiTestCasesCreate, domain.NewArgs(args))
			if !errors.Is(err, domain.ErrUnresolvedName) {
				t.Fatalf("error = %v, want ErrUnresolvedName", err)
			}
			if r.stub.Calls(methodTestCaseCreate) != 0 {
				t.Fatalf("a create was sent for an unresolved name")
			}
		})
	}
}

// TestUpdateCaseChangesOnlyTheNamedFields checks update: the case is checked, and only the named field is sent.
func TestUpdateCaseChangesOnlyTheNamedFields(t *testing.T) {
	r := newRig(t, sampleDataset())
	rec, err := r.gw.Write(context.Background(), domain.OpKiwiTestCasesUpdate, domain.NewArgs(map[string]any{
		"documentId": "case-12", "summary": "Renamed",
	}))
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if r.stub.Calls(methodTestCaseUpdate) != 1 {
		t.Fatalf("TestCase.update calls = %d, want 1", r.stub.Calls(methodTestCaseUpdate))
	}
	if fmt.Sprint(lastParams(t, r, methodTestCaseUpdate)["case_id"]) != "12" {
		t.Fatalf("update named the wrong case: %+v", lastParams(t, r, methodTestCaseUpdate))
	}
	values := lastValues(t, r, methodTestCaseUpdate)
	if len(values) != 1 || values["summary"] != "Renamed" {
		t.Fatalf("update values = %+v, want the summary only", values)
	}
	if rec.Str("id") != "case-12" || rec.Str("title") != "Renamed" {
		t.Fatalf("update record = %+v", rec.Fields())
	}
}

// TestUpdateCaseWithStatusAndPriorityResolvesBoth checks that the status and the priority resolve to IDs.
func TestUpdateCaseWithStatusAndPriorityResolvesBoth(t *testing.T) {
	r := newRig(t, sampleDataset())
	_, err := r.gw.Write(context.Background(), domain.OpKiwiTestCasesUpdate, domain.NewArgs(map[string]any{
		"documentId": "case-13", "status": "CONFIRMED", "priority": "P1",
	}))
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	values := lastValues(t, r, methodTestCaseUpdate)
	if fmt.Sprint(values["case_status"]) != "1" || fmt.Sprint(values["priority"]) != "2" {
		t.Fatalf("update values = %+v, want case_status 1 and priority 2", values)
	}
}

// TestUpdateOfAMissingCaseSendsNoUpdate checks that a case that does not exist is refused before the update.
func TestUpdateOfAMissingCaseSendsNoUpdate(t *testing.T) {
	r := newRig(t, sampleDataset())
	_, err := r.gw.Write(context.Background(), domain.OpKiwiTestCasesUpdate, domain.NewArgs(map[string]any{
		"documentId": "case-999", "summary": "Renamed",
	}))
	if !errors.Is(err, domain.ErrDocumentNotFound) {
		t.Fatalf("error = %v, want ErrDocumentNotFound", err)
	}
	if r.stub.Calls(methodTestCaseUpdate) != 0 {
		t.Fatalf("an update was sent for a missing case")
	}
}

// TestAddCaseLinksThePlanAndTheCase checks add-case: one TestPlan.add_case with the plan and the case IDs.
func TestAddCaseLinksThePlanAndTheCase(t *testing.T) {
	r := newRig(t, sampleDataset())
	rec, err := r.gw.Write(context.Background(), domain.OpKiwiPlansAddCase, domain.NewArgs(map[string]any{
		"planId": "plan-1", "caseId": "case-13",
	}))
	if err != nil {
		t.Fatalf("add-case: %v", err)
	}
	if r.stub.Calls(methodTestPlanAddCase) != 1 {
		t.Fatalf("TestPlan.add_case calls = %d, want 1", r.stub.Calls(methodTestPlanAddCase))
	}
	params := lastParams(t, r, methodTestPlanAddCase)
	if fmt.Sprint(params["plan_id"]) != "1" || fmt.Sprint(params["case_id"]) != "13" {
		t.Fatalf("add-case params = %+v", params)
	}
	if rec.Str("planId") != "plan-1" || rec.Str("caseId") != "case-13" {
		t.Fatalf("add-case record = %+v", rec.Fields())
	}
}

// TestAddCaseToAMissingPlanIsNotFound checks that a plan that does not exist is named as not found.
func TestAddCaseToAMissingPlanIsNotFound(t *testing.T) {
	r := newRig(t, sampleDataset())
	_, err := r.gw.Write(context.Background(), domain.OpKiwiPlansAddCase, domain.NewArgs(map[string]any{
		"planId": "plan-999", "caseId": "case-13",
	}))
	if !errors.Is(err, domain.ErrDocumentNotFound) {
		t.Fatalf("error = %v, want ErrDocumentNotFound", err)
	}
}

// TestWriteRefusesAReadOperation checks that the write path refuses a read operation before any request.
func TestWriteRefusesAReadOperation(t *testing.T) {
	r := newRig(t, sampleDataset())
	_, err := r.gw.Write(context.Background(), domain.OpKiwiProductsList, domain.NewArgs(map[string]any{}))
	if !errors.Is(err, domain.ErrWrongOperationClass) {
		t.Fatalf("error = %v, want ErrWrongOperationClass", err)
	}
	if len(r.stub.Requests()) != 0 {
		t.Fatalf("a request was sent for a refused write")
	}
}

// TestWriteRefusesAnUnknownOperation checks that an operation outside the catalog is unsupported.
func TestWriteRefusesAnUnknownOperation(t *testing.T) {
	r := newRig(t, sampleDataset())
	_, err := r.gw.Write(context.Background(), domain.OperationID("kiwi.nope"), domain.NewArgs(map[string]any{}))
	if !errors.Is(err, domain.ErrUnsupportedOperation) {
		t.Fatalf("error = %v, want ErrUnsupportedOperation", err)
	}
}

// TestCreateRejectsAProductInTheWrongShape checks that a bad product ID is refused before any lookup is sent.
func TestCreateRejectsAProductInTheWrongShape(t *testing.T) {
	r := newRig(t, sampleDataset())
	_, err := r.gw.Write(context.Background(), domain.OpKiwiTestCasesCreate, domain.NewArgs(map[string]any{
		"summary": "New", "product": "plan-1", "category": "Functional", "priority": "P1",
	}))
	if err == nil {
		t.Fatalf("a plan ID was accepted as a product")
	}
	if len(r.stub.Requests()) != 0 {
		t.Fatalf("a request was sent for a bad product ID")
	}
}
