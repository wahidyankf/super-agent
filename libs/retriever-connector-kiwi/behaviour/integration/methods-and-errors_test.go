//go:build integration

package integration_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-connector-kiwi/kiwi"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-testkit/kiwistub"
)

// @scenario specs/libs/retriever-connector-kiwi/behaviours/methods-and-errors.feature: A method outside the allowed list never reaches the network
func TestMethodOutsideAllowedListNeverReachesTheNetwork(t *testing.T) {
	srv := serveKiwi(t, newStub(sampleDataset()), nil)
	gw := gatewayAt(t, srv, staticCreds{})
	_, err := gw.TestSeamCall(context.Background(), "TestCase.remove", map[string]any{"query": map[string]any{}})
	if err == nil {
		t.Fatal("a method outside the allowed list was sent")
	}
	if n := srv.Log.count(); n != 0 {
		t.Errorf("%d requests reached the server", n)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/methods-and-errors.feature: An argument that Kiwi refuses is refused by the adapter at any depth before the request leaves
func TestForbiddenArgumentKeyIsRefusedAtAnyDepthBeforeRequestLeaves(t *testing.T) {
	srv := serveKiwi(t, newStub(sampleDataset()), nil)
	gw := gatewayAt(t, srv, staticCreds{})
	params := map[string]any{"query": map[string]any{"deep": []any{map[string]any{"token": "x"}}}}
	_, err := gw.TestSeamCall(context.Background(), "TestCase.filter", params)
	if !errors.Is(err, domain.ErrBadRequest) {
		t.Errorf("the call was not refused by the guard: %v", err)
	}
	if n := srv.Log.count(); n != 0 {
		t.Errorf("%d requests were sent for a refused argument", n)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/methods-and-errors.feature: A value that JSON cannot hold is refused before any request
func TestValueThatJSONCannotHoldIsRefusedBeforeAnyRequest(t *testing.T) {
	srv := serveKiwi(t, newStub(sampleDataset()), nil)
	gw := gatewayAt(t, srv, staticCreds{})
	params := map[string]any{"query": map[string]any{"when": func() {}}}
	if _, err := gw.TestSeamCall(context.Background(), "Product.filter", params); err == nil {
		t.Fatal("a value JSON cannot hold was encoded and sent")
	}
	if n := srv.Log.count(); n != 0 {
		t.Errorf("%d requests were sent for an unencodable value", n)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/methods-and-errors.feature: Every request that the adapter sends passes the argument guard
func TestEveryRequestPassesTheGuardAndEveryAllowedMethodRuns(t *testing.T) {
	srv := serveKiwi(t, newStub(sampleDataset()), nil)
	gw := gatewayAt(t, srv, staticCreds{})
	ctx := context.Background()
	_, err := gw.ListCollections(ctx)
	must(t, err)
	_, err = treeOf(gw, "product-1")
	must(t, err)
	_, err = docOf(gw, "plan-2")
	must(t, err)
	_, err = docOf(gw, "case-12")
	must(t, err)
	_, err = gw.Write(ctx, domain.OpKiwiTestCasesCreate, domain.NewArgs(map[string]any{
		"summary": "New case", "product": "product-1", "category": "Functional", "priority": "P1", "status": "CONFIRMED", "text": "Body",
	}))
	must(t, err)
	_, err = gw.Write(ctx, domain.OpKiwiTestCasesUpdate, domain.NewArgs(map[string]any{"documentId": "case-12", "summary": "Renamed"}))
	must(t, err)
	_, err = gw.Write(ctx, domain.OpKiwiPlansAddCase, domain.NewArgs(map[string]any{"planId": "plan-1", "caseId": "case-13"}))
	must(t, err)

	seen := map[string]bool{}
	for _, req := range srv.Stub.Requests() {
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
			srv := serveKiwi(t, newStub(sampleDataset()), nil)
			srv.Stub.SetFault(kiwistub.Fault{Status: row.status})
			gw := gatewayAt(t, srv, staticCreds{})
			if err := callOf(gw, row.call); !errors.Is(err, row.want) {
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
			srv := serveKiwi(t, newStub(sampleDataset()), nil)
			srv.Stub.SetMethodError("Product.filter", row.code, row.message)
			gw := gatewayAt(t, srv, staticCreds{})
			if _, err := gw.ListCollections(context.Background()); !errors.Is(err, row.want) {
				t.Errorf("the call failed with %v, want %v", err, row.want)
			}
		})
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/methods-and-errors.feature: No error text carries the key, even when the reply echoes it
func TestNoErrorTextCarriesTheKey(t *testing.T) {
	srv := serveKiwi(t, newStub(sampleDataset()), nil)
	srv.Stub.SetMethodError("Product.filter", -32603, "authentication failed for "+sentinelKey)
	gw := gatewayAt(t, srv, staticCreds{})
	_, err := gw.ListCollections(context.Background())
	if err == nil || strings.Contains(err.Error(), sentinelKey) {
		t.Errorf("the JSON-RPC error text is %v, want no key", err)
	}

	echo := serveKiwi(t, newStub(sampleDataset()), nil)
	echo.Stub.SetFault(kiwistub.Fault{Status: http.StatusInternalServerError, EchoKey: true})
	echoGW := gatewayAt(t, echo, staticCreds{})
	_, err = echoGW.ListCollections(context.Background())
	if err == nil || strings.Contains(err.Error(), sentinelKey) {
		t.Errorf("the HTTP error text is %v, want no key", err)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/methods-and-errors.feature: A missing key is named by its variable and nothing is sent
func TestMissingKeyIsNamedByItsVariableAndNothingIsSent(t *testing.T) {
	srv := serveKiwi(t, newStub(sampleDataset()), nil)
	gw := gatewayAt(t, srv, missingCreds{})
	_, err := gw.ListCollections(context.Background())
	var missing domain.MissingKeyError
	if !errors.As(err, &missing) || missing.Variable != keyVariable {
		t.Errorf("the error %v does not name the key variable %s", err, keyVariable)
	}
	if n := srv.Log.count(); n != 0 {
		t.Errorf("%d requests were sent without a key", n)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/methods-and-errors.feature: Every failure the adapter returns names Kiwi as its kind
func TestEveryFailureNamesKiwiAsItsKind(t *testing.T) {
	srv := serveKiwi(t, newStub(sampleDataset()), nil)
	srv.Stub.SetFault(kiwistub.Fault{Status: http.StatusInternalServerError})
	gw := gatewayAt(t, srv, staticCreds{})
	ctx := context.Background()
	var errs []error
	_, err := gw.ListCollections(ctx)
	errs = append(errs, err)
	_, err = treeOf(gw, "product-1")
	errs = append(errs, err)
	_, err = docOf(gw, "plan-2")
	errs = append(errs, err)
	_, err = gw.Read(ctx, "kiwi.unknown", domain.NewArgs(nil))
	errs = append(errs, err)
	errs = append(errs, addCase(gw))
	for i, e := range errs {
		if e == nil || !isKiwiKind(e) {
			t.Errorf("failure %d (%v) does not name Kiwi as its kind", i, e)
		}
	}
}
