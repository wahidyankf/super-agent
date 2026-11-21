package unit_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-connector-kiwi/kiwi"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-testkit/kiwistub"
)

// @scenario specs/libs/retriever-connector-kiwi/behaviours/methods-and-errors.feature: The method table holds exactly the ten allowed methods and no removal or authentication method
func TestMethodTableHoldsTenMethodsAndNoRemovalOrAuth(t *testing.T) {
	list := kiwi.TestSeamAllowedMethods()
	if len(list) != 10 {
		t.Fatalf("the table holds %d methods, want 10: %v", len(list), list)
	}
	for _, m := range list {
		for _, bad := range []string{"remove", "delete", "auth", "login", "token"} {
			if strings.Contains(strings.ToLower(m), bad) {
				t.Errorf("the table holds %q, which names %q", m, bad)
			}
		}
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/methods-and-errors.feature: Exactly three allowed methods change Kiwi, and only those three carry the write failure rule
func TestExactlyThreeAllowedMethodsChangeKiwi(t *testing.T) {
	allowed := setOf(kiwi.TestSeamAllowedMethods())
	writes := kiwi.TestSeamWriteMethods()
	if len(writes) != 3 {
		t.Fatalf("the write list holds %d methods, want 3: %v", len(writes), writes)
	}
	for _, m := range []string{"TestCase.create", "TestCase.update", "TestPlan.add_case"} {
		if !setOf(writes)[m] {
			t.Errorf("the write list does not hold %s", m)
		}
		if !allowed[m] {
			t.Errorf("the write method %s is not in the allowed list", m)
		}
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/methods-and-errors.feature: A method outside the allowed list never reaches the network
func TestMethodOutsideAllowedListNeverReachesTheNetwork(t *testing.T) {
	r := newRig(t, sampleDataset())
	_, err := r.gw.TestSeamCall(context.Background(), "TestCase.remove", map[string]any{"query": map[string]any{}})
	if err == nil {
		t.Fatal("a method outside the allowed list was sent")
	}
	if n := len(r.stub.Requests()); n != 0 {
		t.Errorf("%d requests reached the server", n)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/methods-and-errors.feature: An argument that Kiwi refuses is refused by the adapter at any depth before the request leaves
func TestForbiddenArgumentKeyIsRefusedAtAnyDepth(t *testing.T) {
	params := map[string]any{"query": map[string]any{"deep": []any{map[string]any{"token": "x"}}}}
	if err := kiwi.TestSeamGuardArguments(params); !errors.Is(err, domain.ErrBadRequest) {
		t.Errorf("the guard did not refuse the nested key: %v", err)
	}
	r := newRig(t, sampleDataset())
	_, err := r.gw.TestSeamCall(context.Background(), "TestCase.filter", params)
	if !errors.Is(err, domain.ErrBadRequest) {
		t.Errorf("the call was not refused by the guard: %v", err)
	}
	if n := len(r.stub.Requests()); n != 0 {
		t.Errorf("%d requests were sent for a refused argument", n)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/methods-and-errors.feature: A value that JSON cannot hold is refused before any request
func TestValueThatJSONCannotHoldIsRefusedBeforeAnyRequest(t *testing.T) {
	r := newRig(t, sampleDataset())
	params := map[string]any{"query": map[string]any{"when": func() {}}}
	if _, err := r.gw.TestSeamCall(context.Background(), "Product.filter", params); err == nil {
		t.Fatal("a value JSON cannot hold was encoded and sent")
	}
	if n := len(r.stub.Requests()); n != 0 {
		t.Errorf("%d requests were sent for an unencodable value", n)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/methods-and-errors.feature: Every request that the adapter sends passes the argument guard
func TestEveryRequestPassesTheGuardAndEveryAllowedMethodRuns(t *testing.T) {
	r := newRig(t, sampleDataset())
	ctx := context.Background()
	_, err := r.gw.ListCollections(ctx)
	must(t, err)
	_, err = treeOf(r.gw, "product-1")
	must(t, err)
	_, err = docOf(r.gw, "plan-2")
	must(t, err)
	_, err = docOf(r.gw, "case-12")
	must(t, err)
	_, err = r.gw.Write(ctx, domain.OpKiwiTestCasesCreate, domain.NewArgs(map[string]any{
		"summary": "New case", "product": "product-1", "category": "Functional", "priority": "P1", "status": "CONFIRMED", "text": "Body",
	}))
	must(t, err)
	_, err = r.gw.Write(ctx, domain.OpKiwiTestCasesUpdate, domain.NewArgs(map[string]any{"documentId": "case-12", "summary": "Renamed"}))
	must(t, err)
	_, err = r.gw.Write(ctx, domain.OpKiwiPlansAddCase, domain.NewArgs(map[string]any{"planId": "plan-1", "caseId": "case-13"}))
	must(t, err)

	seen := map[string]bool{}
	for _, req := range r.stub.Requests() {
		if err := kiwi.TestSeamGuardArguments(req.Params); err != nil {
			t.Errorf("the %s request was refused by the guard: %v", req.Method, err)
		}
		seen[req.Method] = true
	}
	for _, m := range kiwi.TestSeamAllowedMethods() {
		if !seen[m] {
			t.Errorf("the allowed method %s was never sent", m)
		}
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/methods-and-errors.feature: An HTTP status of <status> on a <call> call fails with <error>
func TestHTTPStatusFailsEachCallWithItsError(t *testing.T) {
	rows := []struct {
		status int
		call   string
		want   error
	}{
		{401, "read", domain.ErrKeyRejected},
		{403, "read", domain.ErrAccessDenied},
		{404, "read", domain.ErrUnexpectedResponse},
		{500, "read", domain.ErrUpstreamFailure},
		{500, "write", domain.ErrWriteOutcomeUnknown},
		{503, "write", domain.ErrWriteOutcomeUnknown},
		{401, "write", domain.ErrKeyRejected},
	}
	for _, row := range rows {
		t.Run(fmt.Sprintf("%d %s", row.status, row.call), func(t *testing.T) {
			r := newRig(t, sampleDataset())
			r.stub.SetFault(kiwistub.Fault{Status: row.status})
			if err := callOf(r.gw, row.call); !errors.Is(err, row.want) {
				t.Errorf("the call failed with %v, want %v", err, row.want)
			}
		})
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/methods-and-errors.feature: A JSON-RPC error code and its message map to one error each
func TestJSONRPCErrorCodeAndMessageMapToOneErrorEach(t *testing.T) {
	rows := []struct {
		code    int
		message string
		want    error
	}{
		{-32600, "Invalid Request", domain.ErrBadRequest},
		{-32601, "Method not found", domain.ErrBadRequest},
		{-32602, "Invalid params", domain.ErrBadRequest},
		{-32603, "authentication credentials were not provided", domain.ErrKeyRejected},
		{-32603, "You do not have permission to read products", domain.ErrAccessDenied},
		{-32603, "Product matching query does not exist.", domain.ErrCollectionNotFound},
		{-32603, "This field is required: name.", domain.ErrBadRequest},
		{-32603, "the server had a fault", domain.ErrUpstreamFailure},
	}
	for _, row := range rows {
		t.Run(row.message, func(t *testing.T) {
			r := newRig(t, sampleDataset())
			r.stub.SetMethodError("Product.filter", row.code, row.message)
			if _, err := r.gw.ListCollections(context.Background()); !errors.Is(err, row.want) {
				t.Errorf("the call failed with %v, want %v", err, row.want)
			}
		})
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/methods-and-errors.feature: No error text carries the key, even when the reply echoes it
func TestNoErrorTextCarriesTheKey(t *testing.T) {
	r := newRig(t, sampleDataset())
	r.stub.SetMethodError("Product.filter", -32603, "authentication failed for "+sentinelKey)
	_, err := r.gw.ListCollections(context.Background())
	if err == nil || strings.Contains(err.Error(), sentinelKey) {
		t.Errorf("the JSON-RPC error text is %v, want no key", err)
	}
	r2 := newRig(t, sampleDataset())
	r2.stub.SetFault(kiwistub.Fault{Status: 500, EchoKey: true})
	_, err = r2.gw.ListCollections(context.Background())
	if err == nil || strings.Contains(err.Error(), sentinelKey) {
		t.Errorf("the HTTP error text is %v, want no key", err)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/methods-and-errors.feature: A missing key is named by its variable and nothing is sent
func TestMissingKeyIsNamedByItsVariableAndNothingIsSent(t *testing.T) {
	clock := newFakeClock()
	stub := kiwistub.New(sentinelKey, kiwistub.WithDataset(sampleDataset()), kiwistub.WithClock(clock.Now))
	gw := hermeticGateway(t, stub.RoundTripper(), missingCreds{}, clock)
	_, err := gw.ListCollections(context.Background())
	var missing domain.MissingKeyError
	if !errors.As(err, &missing) || missing.Variable != keyVariable {
		t.Errorf("the error %v does not name the key variable %s", err, keyVariable)
	}
	if n := len(stub.Requests()); n != 0 {
		t.Errorf("%d requests were sent without a key", n)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/methods-and-errors.feature: Every failure the adapter returns names Kiwi as its kind
func TestEveryFailureNamesKiwiAsItsKind(t *testing.T) {
	r := newRig(t, sampleDataset())
	r.stub.SetFault(kiwistub.Fault{Status: 500})
	ctx := context.Background()
	errs := []error{}
	_, err := r.gw.ListCollections(ctx)
	errs = append(errs, err)
	_, err = treeOf(r.gw, "product-1")
	errs = append(errs, err)
	_, err = docOf(r.gw, "plan-2")
	errs = append(errs, err)
	_, err = r.gw.Read(ctx, "kiwi.unknown", domain.NewArgs(nil))
	errs = append(errs, err)
	_, err = r.gw.Write(ctx, domain.OpKiwiPlansAddCase, domain.NewArgs(map[string]any{"planId": "plan-2", "caseId": "case-12"}))
	errs = append(errs, err)
	for i, e := range errs {
		if e == nil || !isKiwiKind(e) {
			t.Errorf("failure %d (%v) does not name Kiwi as its kind", i, e)
		}
	}
}
