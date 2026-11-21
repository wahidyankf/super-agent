package unit_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-testkit/kiwistub"
)

// valuesSent returns the values object of the first request of a method that the stub received.
func valuesSent(t *testing.T, stub *kiwistub.Stub, method string) map[string]any {
	t.Helper()
	for _, q := range stub.Requests() {
		if q.Method == method {
			values, _ := q.Params["values"].(map[string]any)
			return values
		}
	}
	t.Fatalf("no %s request was sent", method)
	return nil
}

// paramsSent returns the params of the first request of a method that the stub received.
func paramsSent(t *testing.T, stub *kiwistub.Stub, method string) map[string]any {
	t.Helper()
	for _, q := range stub.Requests() {
		if q.Method == method {
			return q.Params
		}
	}
	t.Fatalf("no %s request was sent", method)
	return nil
}

// countMethod returns how many recorded calls named the method.
func countMethod(calls []call, method string) int {
	n := 0
	for _, c := range calls {
		if c.method == method {
			n++
		}
	}
	return n
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/writes.feature: Creating a case resolves its names to IDs and sends one create
func TestCreatingCaseResolvesNamesToIDsAndSendsOneCreate(t *testing.T) {
	r := newRig(t, sampleDataset())
	rec, err := r.gw.Write(context.Background(), domain.OpKiwiTestCasesCreate, domain.NewArgs(map[string]any{
		"summary": "New case", "product": "product-1", "category": "Functional", "priority": "P1",
	}))
	must(t, err)
	if rec.Str("id") != "case-14" {
		t.Errorf("the created case is %q, want case-14", rec.Str("id"))
	}
	values := valuesSent(t, r.stub, "TestCase.create")
	if values["category"] != float64(1) || values["priority"] != float64(2) {
		t.Errorf("the create carries category %v and priority %v, want the IDs 1 and 2", values["category"], values["priority"])
	}
	if n := r.stub.Calls("TestCase.create"); n != 1 {
		t.Errorf("%d create requests were sent, want 1", n)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/writes.feature: Creating a case sends the optional status and text when the caller gives them
func TestCreatingCaseSendsOptionalStatusAndText(t *testing.T) {
	r := newRig(t, sampleDataset())
	_, err := r.gw.Write(context.Background(), domain.OpKiwiTestCasesCreate, domain.NewArgs(map[string]any{
		"summary": "New case", "product": "product-1", "category": "Functional", "priority": "P1", "status": "CONFIRMED", "text": "Body text",
	}))
	must(t, err)
	values := valuesSent(t, r.stub, "TestCase.create")
	if values["case_status"] != float64(1) || values["text"] != "Body text" {
		t.Errorf("the create carries status %v and text %v", values["case_status"], values["text"])
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/writes.feature: A create whose name matches <matches> sends no write
func TestCreateWhoseNameMatchesNoRowOrSeveralSendsNoWrite(t *testing.T) {
	rows := []struct{ matches, category string }{
		{"no row", "Missing"},
		{"several rows", "Regression"},
	}
	for _, row := range rows {
		t.Run(row.matches, func(t *testing.T) {
			r := newRig(t, sampleDataset())
			_, err := r.gw.Write(context.Background(), domain.OpKiwiTestCasesCreate, domain.NewArgs(map[string]any{
				"summary": "New case", "product": "product-1", "category": row.category, "priority": "P1",
			}))
			if !errors.Is(err, domain.ErrUnresolvedName) {
				t.Errorf("the create failed with %v, want the unresolved-name error", err)
			}
			if n := r.stub.Calls("TestCase.create"); n != 0 {
				t.Errorf("%d create requests were sent", n)
			}
		})
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/writes.feature: Updating a case sends only the field the caller names
func TestUpdatingCaseSendsOnlyTheNamedField(t *testing.T) {
	r := newRig(t, sampleDataset())
	_, err := r.gw.Write(context.Background(), domain.OpKiwiTestCasesUpdate, domain.NewArgs(map[string]any{
		"documentId": "case-12", "summary": "Renamed",
	}))
	must(t, err)
	reqs := r.stub.Requests()
	if len(reqs) < 2 || reqs[0].Method != "TestCase.filter" {
		t.Fatalf("the case was not checked first: %d requests", len(reqs))
	}
	values := valuesSent(t, r.stub, "TestCase.update")
	if len(values) != 1 || values["summary"] != "Renamed" {
		t.Errorf("the update carries %v, want only the summary", values)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/writes.feature: Updating a case with a status and a priority resolves both names to IDs
func TestUpdatingCaseWithStatusAndPriorityResolvesBothNames(t *testing.T) {
	r := newRig(t, sampleDataset())
	_, err := r.gw.Write(context.Background(), domain.OpKiwiTestCasesUpdate, domain.NewArgs(map[string]any{
		"documentId": "case-12", "status": "DRAFT", "priority": "P1",
	}))
	must(t, err)
	values := valuesSent(t, r.stub, "TestCase.update")
	if values["case_status"] != float64(2) || values["priority"] != float64(2) {
		t.Errorf("the update carries status %v and priority %v, want the IDs 2 and 2", values["case_status"], values["priority"])
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/writes.feature: Updating a case that does not exist sends no update
func TestUpdatingMissingCaseSendsNoUpdate(t *testing.T) {
	r := newRig(t, sampleDataset())
	_, err := r.gw.Write(context.Background(), domain.OpKiwiTestCasesUpdate, domain.NewArgs(map[string]any{
		"documentId": "case-99", "summary": "Renamed",
	}))
	if !errors.Is(err, domain.ErrDocumentNotFound) {
		t.Errorf("the update failed with %v, want the not-found error", err)
	}
	if n := r.stub.Calls("TestCase.update"); n != 0 {
		t.Errorf("%d update requests were sent", n)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/writes.feature: Adding a case to a plan sends one add that links the plan and the case
func TestAddingCaseToPlanSendsOneAddThatLinksBoth(t *testing.T) {
	r := newRig(t, sampleDataset())
	rec, err := r.gw.Write(context.Background(), domain.OpKiwiPlansAddCase, domain.NewArgs(map[string]any{
		"planId": "plan-2", "caseId": "case-12",
	}))
	must(t, err)
	if rec.Str("planId") != "plan-2" || rec.Str("caseId") != "case-12" {
		t.Errorf("the add holds plan %q and case %q", rec.Str("planId"), rec.Str("caseId"))
	}
	if n := r.stub.Calls("TestPlan.add_case"); n != 1 {
		t.Errorf("%d add requests were sent, want 1", n)
	}
	params := paramsSent(t, r.stub, "TestPlan.add_case")
	if params["plan_id"] != float64(2) || params["case_id"] != float64(12) {
		t.Errorf("the add carries plan %v and case %v", params["plan_id"], params["case_id"])
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/writes.feature: Adding a case to a plan that does not exist is named as not found
func TestAddingCaseToMissingPlanIsNamedAsNotFound(t *testing.T) {
	r := newRig(t, sampleDataset())
	_, err := r.gw.Write(context.Background(), domain.OpKiwiPlansAddCase, domain.NewArgs(map[string]any{
		"planId": "plan-99", "caseId": "case-12",
	}))
	if !errors.Is(err, domain.ErrDocumentNotFound) {
		t.Errorf("the add failed with %v, want the not-found error", err)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/writes.feature: A write refuses a read operation and sends no request
func TestWriteRefusesReadOperationAndSendsNoRequest(t *testing.T) {
	r := newRig(t, sampleDataset())
	_, err := r.gw.Write(context.Background(), domain.OpKiwiProductsList, domain.NewArgs(map[string]any{}))
	if !errors.Is(err, domain.ErrWrongOperationClass) {
		t.Errorf("the write failed with %v, want the wrong-class error", err)
	}
	if n := len(r.stub.Requests()); n != 0 {
		t.Errorf("%d requests were sent", n)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/writes.feature: A write of an operation outside the catalog is unsupported
func TestWriteOfOperationOutsideCatalogIsUnsupported(t *testing.T) {
	r := newRig(t, sampleDataset())
	_, err := r.gw.Write(context.Background(), "kiwi.unknown.write", domain.NewArgs(map[string]any{}))
	if !errors.Is(err, domain.ErrUnsupportedOperation) {
		t.Errorf("the write failed with %v, want the unsupported error", err)
	}
	if n := len(r.stub.Requests()); n != 0 {
		t.Errorf("%d requests were sent", n)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/writes.feature: An update that names a plan ID is refused before any request
func TestUpdateNamingPlanIDIsRefusedBeforeAnyRequest(t *testing.T) {
	r := newRig(t, sampleDataset())
	_, err := r.gw.Write(context.Background(), domain.OpKiwiTestCasesUpdate, domain.NewArgs(map[string]any{
		"documentId": "plan-2", "summary": "Renamed",
	}))
	if !errors.Is(err, domain.ErrInvalidDocumentID) {
		t.Errorf("the update failed with %v, want the invalid-document error", err)
	}
	if n := len(r.stub.Requests()); n != 0 {
		t.Errorf("%d requests were sent", n)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/writes.feature: A write of another kind's operation is not sent to Kiwi
func TestWriteOfAnotherKindsOperationIsNotSentToKiwi(t *testing.T) {
	r := newRig(t, sampleDataset())
	_, err := r.gw.Write(context.Background(), domain.OpDocumentsUpdate, domain.NewArgs(map[string]any{
		"documentId": "x", "title": "Renamed",
	}))
	if err == nil {
		t.Fatal("another kind's write was accepted")
	}
	if n := len(r.stub.Requests()); n != 0 {
		t.Errorf("%d requests were sent to Kiwi", n)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/writes.feature: A create whose name lookup fails stops before the write is sent
func TestCreateWhoseNameLookupFailsStopsBeforeWrite(t *testing.T) {
	r := newRig(t, sampleDataset())
	r.stub.SetMethodFault("Category.filter", kiwistub.Fault{Status: http.StatusInternalServerError})
	_, err := r.gw.Write(context.Background(), domain.OpKiwiTestCasesCreate, domain.NewArgs(map[string]any{
		"summary": "New case", "product": "product-1", "category": "Functional", "priority": "P1",
	}))
	if !errors.Is(err, domain.ErrUpstreamFailure) {
		t.Errorf("the create returned %v, want the lookup fault", err)
	}
	if n := r.stub.Calls("TestCase.create"); n != 0 {
		t.Errorf("%d create requests were sent", n)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/writes.feature: An update whose <step> fails stops before the write is sent
func TestUpdateWhoseStepFailsStopsBeforeWrite(t *testing.T) {
	rows := []struct {
		step, method string
		values       map[string]any
	}{
		{"case lookup", "TestCase.filter", map[string]any{"documentId": "case-12", "summary": "Renamed"}},
		{"status or priority lookup", "TestCaseStatus.filter", map[string]any{"documentId": "case-12", "status": "DRAFT"}},
	}
	for _, row := range rows {
		t.Run(row.step, func(t *testing.T) {
			r := newRig(t, sampleDataset())
			r.stub.SetMethodFault(row.method, kiwistub.Fault{Status: http.StatusInternalServerError})
			_, err := r.gw.Write(context.Background(), domain.OpKiwiTestCasesUpdate, domain.NewArgs(row.values))
			if !errors.Is(err, domain.ErrUpstreamFailure) {
				t.Errorf("the update returned %v, want the fault of the step", err)
			}
			if n := r.stub.Calls("TestCase.update"); n != 0 {
				t.Errorf("%d update requests were sent", n)
			}
		})
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/writes.feature: A create whose reply does not name the new case is an unknown outcome and is not sent again
func TestCreateWhoseReplyDoesNotNameCaseIsUnknownOutcome(t *testing.T) {
	stub := kiwistub.New(sentinelKey, kiwistub.WithDataset(sampleDataset()))
	srv := &scripted{serve: func(req *http.Request, c call) (*http.Response, error) {
		if c.method == "TestCase.create" {
			return rpcReply(req, c, "{}", nil), nil
		}
		return stub.RoundTripper().RoundTrip(req)
	}}
	gw := hermeticGateway(t, srv, staticCreds{}, newFakeClock())
	_, err := gw.Write(context.Background(), domain.OpKiwiTestCasesCreate, domain.NewArgs(map[string]any{
		"summary": "New case", "product": "product-1", "category": "Functional", "priority": "P1",
	}))
	if !errors.Is(err, domain.ErrWriteOutcomeUnknown) {
		t.Errorf("the create returned %v, want the unknown-outcome error", err)
	}
	if n := countMethod(srv.requests(), "TestCase.create"); n != 1 {
		t.Errorf("%d create requests were sent, want 1", n)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/writes.feature: An add-case whose exchange fails is an unknown outcome and is not sent again
func TestAddCaseWhoseExchangeFailsIsUnknownOutcome(t *testing.T) {
	srv := &scripted{serve: func(*http.Request, call) (*http.Response, error) {
		return nil, fmt.Errorf("connection reset by peer")
	}}
	gw := hermeticGateway(t, srv, staticCreds{}, newFakeClock())
	_, err := gw.Write(context.Background(), domain.OpKiwiPlansAddCase, domain.NewArgs(map[string]any{
		"planId": "plan-2", "caseId": "case-12",
	}))
	if !errors.Is(err, domain.ErrWriteOutcomeUnknown) {
		t.Errorf("the add returned %v, want the unknown-outcome error", err)
	}
	if n := countMethod(srv.requests(), "TestPlan.add_case"); n != 1 {
		t.Errorf("%d add requests were sent, want 1", n)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/writes.feature: A write whose body breaks after Kiwi answered its status is an unknown outcome
func TestWriteWhoseBodyBreaksAfterStatusIsUnknownOutcome(t *testing.T) {
	srv := &scripted{serve: func(req *http.Request, _ call) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: &brokenBody{}, Request: req}, nil
	}}
	gw := hermeticGateway(t, srv, staticCreds{}, newFakeClock())
	_, err := gw.Write(context.Background(), domain.OpKiwiPlansAddCase, domain.NewArgs(map[string]any{
		"planId": "plan-2", "caseId": "case-12",
	}))
	if !errors.Is(err, domain.ErrWriteOutcomeUnknown) {
		t.Errorf("the write returned %v, want the unknown-outcome error", err)
	}
	if n := len(srv.requests()); n != 1 {
		t.Errorf("%d requests were sent, want 1", n)
	}
}
