//go:build integration

package integration_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-testkit/kiwistub"
)

// requestOf returns the params the server received in the first call of a method, and fails the binding when none came.
func requestOf(t *testing.T, stub *kiwistub.Stub, method string) map[string]any {
	t.Helper()
	for _, q := range stub.Requests() {
		if q.Method == method {
			return q.Params
		}
	}
	t.Fatalf("no %s request reached the server", method)
	return nil
}

// valuesOf returns the values object the server received in the first call of a method.
func valuesOf(t *testing.T, stub *kiwistub.Stub, method string) map[string]any {
	t.Helper()
	values, _ := requestOf(t, stub, method)["values"].(map[string]any)
	return values
}

// createArgs returns the arguments of a create of the case "New case" in product-1, with the given category name.
func createArgs(category string) domain.Args {
	return domain.NewArgs(map[string]any{
		"summary": "New case", "product": "product-1", "category": category, "priority": "P1",
	})
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/writes.feature: Creating a case resolves its names to IDs and sends one create
func TestCreatingCaseResolvesNamesToIDsAndSendsOneCreate(t *testing.T) {
	srv := serveKiwi(t, newStub(sampleDataset()), nil)
	gw := gatewayAt(t, srv, staticCreds{})
	rec, err := gw.Write(context.Background(), domain.OpKiwiTestCasesCreate, createArgs("Functional"))
	must(t, err)
	if rec.Str("id") != "case-14" {
		t.Errorf("the created case is %q, want case-14", rec.Str("id"))
	}
	values := valuesOf(t, srv.Stub, "TestCase.create")
	if values["category"] != float64(1) || values["priority"] != float64(2) {
		t.Errorf("the create carries category %v and priority %v, want the IDs 1 and 2", values["category"], values["priority"])
	}
	if n := srv.Stub.Calls("TestCase.create"); n != 1 {
		t.Errorf("%d create requests were sent, want 1", n)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/writes.feature: Creating a case sends the optional status and text when the caller gives them
func TestCreatingCaseSendsOptionalStatusAndText(t *testing.T) {
	srv := serveKiwi(t, newStub(sampleDataset()), nil)
	gw := gatewayAt(t, srv, staticCreds{})
	_, err := gw.Write(context.Background(), domain.OpKiwiTestCasesCreate, domain.NewArgs(map[string]any{
		"summary": "New case", "product": "product-1", "category": "Functional", "priority": "P1", "status": "CONFIRMED", "text": "Body text",
	}))
	must(t, err)
	values := valuesOf(t, srv.Stub, "TestCase.create")
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
			srv := serveKiwi(t, newStub(sampleDataset()), nil)
			gw := gatewayAt(t, srv, staticCreds{})
			_, err := gw.Write(context.Background(), domain.OpKiwiTestCasesCreate, createArgs(row.category))
			if !errors.Is(err, domain.ErrUnresolvedName) {
				t.Errorf("the create failed with %v, want the unresolved-name error", err)
			}
			if n := srv.Stub.Calls("TestCase.create"); n != 0 {
				t.Errorf("%d create requests were sent", n)
			}
		})
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/writes.feature: Updating a case sends only the field the caller names
func TestUpdatingCaseSendsOnlyTheNamedField(t *testing.T) {
	srv := serveKiwi(t, newStub(sampleDataset()), nil)
	gw := gatewayAt(t, srv, staticCreds{})
	_, err := gw.Write(context.Background(), domain.OpKiwiTestCasesUpdate, domain.NewArgs(map[string]any{
		"documentId": "case-12", "summary": "Renamed",
	}))
	must(t, err)
	reqs := srv.Stub.Requests()
	if len(reqs) < 2 || reqs[0].Method != "TestCase.filter" {
		t.Fatalf("the case was not checked first: %d requests", len(reqs))
	}
	values := valuesOf(t, srv.Stub, "TestCase.update")
	if len(values) != 1 || values["summary"] != "Renamed" {
		t.Errorf("the update carries %v, want only the summary", values)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/writes.feature: Updating a case with a status and a priority resolves both names to IDs
func TestUpdatingCaseWithStatusAndPriorityResolvesBothNames(t *testing.T) {
	srv := serveKiwi(t, newStub(sampleDataset()), nil)
	gw := gatewayAt(t, srv, staticCreds{})
	_, err := gw.Write(context.Background(), domain.OpKiwiTestCasesUpdate, domain.NewArgs(map[string]any{
		"documentId": "case-12", "status": "DRAFT", "priority": "P1",
	}))
	must(t, err)
	values := valuesOf(t, srv.Stub, "TestCase.update")
	if values["case_status"] != float64(2) || values["priority"] != float64(2) {
		t.Errorf("the update carries status %v and priority %v, want the IDs 2 and 2", values["case_status"], values["priority"])
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/writes.feature: Updating a case that does not exist sends no update
func TestUpdatingMissingCaseSendsNoUpdate(t *testing.T) {
	srv := serveKiwi(t, newStub(sampleDataset()), nil)
	gw := gatewayAt(t, srv, staticCreds{})
	_, err := gw.Write(context.Background(), domain.OpKiwiTestCasesUpdate, domain.NewArgs(map[string]any{
		"documentId": "case-99", "summary": "Renamed",
	}))
	if !errors.Is(err, domain.ErrDocumentNotFound) {
		t.Errorf("the update failed with %v, want the not-found error", err)
	}
	if n := srv.Stub.Calls("TestCase.update"); n != 0 {
		t.Errorf("%d update requests were sent", n)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/writes.feature: Adding a case to a plan sends one add that links the plan and the case
func TestAddingCaseToPlanSendsOneAddThatLinksBoth(t *testing.T) {
	srv := serveKiwi(t, newStub(sampleDataset()), nil)
	gw := gatewayAt(t, srv, staticCreds{})
	rec, err := gw.Write(context.Background(), domain.OpKiwiPlansAddCase, domain.NewArgs(map[string]any{
		"planId": "plan-2", "caseId": "case-12",
	}))
	must(t, err)
	if rec.Str("planId") != "plan-2" || rec.Str("caseId") != "case-12" {
		t.Errorf("the add holds plan %q and case %q", rec.Str("planId"), rec.Str("caseId"))
	}
	if n := srv.Stub.Calls("TestPlan.add_case"); n != 1 {
		t.Errorf("%d add requests were sent, want 1", n)
	}
	params := requestOf(t, srv.Stub, "TestPlan.add_case")
	if params["plan_id"] != float64(2) || params["case_id"] != float64(12) {
		t.Errorf("the add carries plan %v and case %v", params["plan_id"], params["case_id"])
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/writes.feature: Adding a case to a plan that does not exist is named as not found
func TestAddingCaseToMissingPlanIsNamedAsNotFound(t *testing.T) {
	srv := serveKiwi(t, newStub(sampleDataset()), nil)
	gw := gatewayAt(t, srv, staticCreds{})
	_, err := gw.Write(context.Background(), domain.OpKiwiPlansAddCase, domain.NewArgs(map[string]any{
		"planId": "plan-99", "caseId": "case-12",
	}))
	if !errors.Is(err, domain.ErrDocumentNotFound) {
		t.Errorf("the add failed with %v, want the not-found error", err)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/writes.feature: A create whose name lookup fails stops before the write is sent
func TestCreateWhoseNameLookupFailsStopsBeforeWrite(t *testing.T) {
	srv := serveKiwi(t, newStub(sampleDataset()), nil)
	srv.Stub.SetMethodFault("Category.filter", kiwistub.Fault{Status: http.StatusInternalServerError})
	gw := gatewayAt(t, srv, staticCreds{})
	_, err := gw.Write(context.Background(), domain.OpKiwiTestCasesCreate, createArgs("Functional"))
	if !errors.Is(err, domain.ErrUpstreamFailure) {
		t.Errorf("the create returned %v, want the lookup fault", err)
	}
	if n := srv.Stub.Calls("TestCase.create"); n != 0 {
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
			srv := serveKiwi(t, newStub(sampleDataset()), nil)
			srv.Stub.SetMethodFault(row.method, kiwistub.Fault{Status: http.StatusInternalServerError})
			gw := gatewayAt(t, srv, staticCreds{})
			_, err := gw.Write(context.Background(), domain.OpKiwiTestCasesUpdate, domain.NewArgs(row.values))
			if !errors.Is(err, domain.ErrUpstreamFailure) {
				t.Errorf("the update returned %v, want the fault of the step", err)
			}
			if n := srv.Stub.Calls("TestCase.update"); n != 0 {
				t.Errorf("%d update requests were sent", n)
			}
		})
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/writes.feature: A create whose reply does not name the new case is an unknown outcome and is not sent again
func TestCreateWhoseReplyDoesNotNameCaseIsUnknownOutcome(t *testing.T) {
	mux := &rpcMux{results: map[string]string{"TestCase.create": "{}"}}
	srv := serveKiwi(t, newStub(sampleDataset()), mux.wrap)
	gw := gatewayAt(t, srv, staticCreds{})
	_, err := gw.Write(context.Background(), domain.OpKiwiTestCasesCreate, createArgs("Functional"))
	if !errors.Is(err, domain.ErrWriteOutcomeUnknown) {
		t.Errorf("the create returned %v, want the unknown-outcome error", err)
	}
	if n := mux.count("TestCase.create"); n != 1 {
		t.Errorf("%d create requests were sent, want 1", n)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/writes.feature: An add-case whose exchange fails is an unknown outcome and is not sent again
func TestAddCaseWhoseExchangeFailsIsUnknownOutcome(t *testing.T) {
	srv := serveKiwi(t, newStub(sampleDataset()), nil)
	srv.Stub.SetMethodFault("TestPlan.add_case", kiwistub.Fault{Drop: true})
	gw := gatewayAt(t, srv, staticCreds{})
	err := addCase(gw)
	if !errors.Is(err, domain.ErrWriteOutcomeUnknown) {
		t.Errorf("the add returned %v, want the unknown-outcome error", err)
	}
	if n := srv.Stub.Calls("TestPlan.add_case"); n != 1 {
		t.Errorf("%d add requests were sent, want 1", n)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/writes.feature: A write whose body breaks after Kiwi answered its status is an unknown outcome
func TestWriteWhoseBodyBreaksAfterStatusIsUnknownOutcome(t *testing.T) {
	srv := serveKiwi(t, newStub(sampleDataset()), breakBody)
	gw := gatewayAt(t, srv, staticCreds{})
	err := addCase(gw)
	if !errors.Is(err, domain.ErrWriteOutcomeUnknown) {
		t.Errorf("the write returned %v, want the unknown-outcome error", err)
	}
	if n := srv.Log.count(); n != 1 {
		t.Errorf("%d requests were sent, want 1", n)
	}
}
