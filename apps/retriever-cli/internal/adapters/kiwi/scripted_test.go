package kiwi

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// errScripted is the transport fault or sleep fault a scripted test makes a step return.
var errScripted = errors.New("scripted transport fault")

// scriptedAnswer is what the scripted server sends for one request. A zero status means 200. A non-nil rawBody is sent
// as it is; otherwise the body is a JSON-RPC reply that carries either result or, when rpcMessage is set, an error. A
// non-nil fault fails the exchange before any reply, and a non-nil bodyErr fails the body after the status.
type scriptedAnswer struct {
	status     int
	retryAfter string
	result     any
	rawBody    []byte
	rpcCode    int
	rpcMessage string
	bodyErr    error
	fault      error
}

// sReply is a 200 reply with the given JSON-RPC result.
func sReply(result any) scriptedAnswer {
	return scriptedAnswer{result: result}
}

// scriptedRT is a round tripper that answers each JSON-RPC request from a script and records the method it was asked for.
// It never touches the network.
type scriptedRT struct {
	answer  func(method string, params map[string]any) scriptedAnswer
	methods []string
}

// scBrokenBody is a reply body that fails on the first read.
type scBrokenBody struct {
	err error
}

func (b scBrokenBody) Read([]byte) (int, error) { return 0, b.err }
func (b scBrokenBody) Close() error             { return nil }

// RoundTrip answers one request from the script.
func (s *scriptedRT) RoundTrip(req *http.Request) (*http.Response, error) {
	raw, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	var sent struct {
		Method string         `json:"method"`
		Params map[string]any `json:"params"`
		ID     int64          `json:"id"`
	}
	if err := json.Unmarshal(raw, &sent); err != nil {
		return nil, err
	}
	s.methods = append(s.methods, sent.Method)
	ans := s.answer(sent.Method, sent.Params)
	if ans.fault != nil {
		return nil, ans.fault
	}
	status := ans.status
	if status == 0 {
		status = http.StatusOK
	}
	payload := ans.rawBody
	if payload == nil {
		env := map[string]any{"jsonrpc": "2.0", "id": sent.ID}
		if ans.rpcMessage != "" {
			env["error"] = map[string]any{"code": ans.rpcCode, "message": ans.rpcMessage}
		} else {
			env["result"] = ans.result
		}
		payload, err = json.Marshal(env)
		if err != nil {
			return nil, err
		}
	}
	var body io.ReadCloser = io.NopCloser(bytes.NewReader(payload))
	if ans.bodyErr != nil {
		body = scBrokenBody{err: ans.bodyErr}
	}
	header := http.Header{}
	if ans.retryAfter != "" {
		header.Set("Retry-After", ans.retryAfter)
	}
	return &http.Response{StatusCode: status, Header: header, Body: body, Request: req}, nil
}

// scriptFor returns an answer function that replies with the named answers, and an empty list for any other method.
func scriptFor(answers map[string]scriptedAnswer) func(string, map[string]any) scriptedAnswer {
	return func(method string, _ map[string]any) scriptedAnswer {
		if ans, ok := answers[method]; ok {
			return ans
		}
		return sReply([]any{})
	}
}

// newScriptedGateway builds a gateway at access over the given transport, with no pacing wait and a fake clock. Options
// given here replace the defaults.
func newScriptedGateway(t *testing.T, access domain.Access, rt http.RoundTripper, opts ...Option) *Gateway {
	t.Helper()
	clock := newFakeClock()
	all := append([]Option{WithMinDelay(0), WithClock(clock.Now), WithSleep(clock.Sleep)}, opts...)
	gw, err := NewWithBase(testSourceAt(t, access), testBase, &http.Client{Transport: rt}, &staticCreds{}, all...)
	if err != nil {
		t.Fatalf("NewWithBase: %v", err)
	}
	return gw
}

// timeoutError is a network error that reports a timeout, as a deadline on the connection does.
type timeoutError struct{}

func (timeoutError) Error() string   { return "i/o timeout" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }

// productRow, planRow, and caseRow are the replies of the filter methods that name one product, plan, or case.
func productRow(id int, name string) map[string]any {
	return map[string]any{"id": id, "name": name}
}

func planRow(id, product int, parent *int) map[string]any {
	row := map[string]any{"id": id, "name": "Plan", "product": product, "text": "Plan text"}
	if parent != nil {
		row["parent"] = *parent
	}
	return row
}

func caseRow(id int, summary string) map[string]any {
	return map[string]any{"id": id, "summary": summary, "text": "Case text", "category": 1, "priority": 2, "case_status": 1}
}

// intPtr returns a pointer to n, for the optional parent of a plan row.
func intPtr(n int) *int {
	return &n
}

// sentWrite reports whether the named method was sent.
func sentWrite(methods []string, method string) bool {
	for _, m := range methods {
		if m == method {
			return true
		}
	}
	return false
}

// TestKiwiReadRefusesAnOperationOutsideTheReadTable checks that a read names one of the Kiwi read operations. An unknown
// operation, a write, a read of another kind, and a read with a bad argument are refused, and no request is sent.
func TestKiwiReadRefusesAnOperationOutsideTheReadTable(t *testing.T) {
	tests := []struct {
		name string
		op   domain.OperationID
		args map[string]any
		want error
	}{
		{name: "an unknown operation is unsupported", op: "kiwi.nope", args: map[string]any{}, want: domain.ErrUnsupportedOperation},
		{name: "a write operation is the wrong class to read", op: domain.OpKiwiTestCasesCreate, args: map[string]any{}, want: domain.ErrWrongOperationClass},
		{name: "a read of another kind's operation is unsupported", op: domain.OpCollectionsList, args: map[string]any{}, want: domain.ErrUnsupportedOperation},
		{name: "a read with a bad collection ID is refused before any request", op: domain.OpKiwiProductsTree, args: map[string]any{"collectionId": "bad value!"}},
		{name: "a product tree with an empty collection is refused before any request", op: domain.OpKiwiProductsTree, args: map[string]any{"collectionId": ""}, want: domain.ErrInvalidCollectionID},
		{name: "a document read with an empty ID is refused before any request", op: domain.OpKiwiDocumentsGet, args: map[string]any{"documentId": ""}, want: domain.ErrInvalidDocumentID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := &scriptedRT{answer: scriptFor(nil)}
			gw := newScriptedGateway(t, domain.AccessRead, rt)
			_, err := gw.Read(context.Background(), tt.op, domain.NewArgs(tt.args))
			if err == nil {
				t.Fatalf("Read returned no error")
			}
			if tt.want != nil && !errors.Is(err, tt.want) {
				t.Fatalf("Read error = %v, want %v", err, tt.want)
			}
			if len(rt.methods) != 0 {
				t.Fatalf("a refused read sent %v", rt.methods)
			}
		})
	}
}

// TestKiwiReadsReturnTheFaultOfTheReplyTheyNeed checks that a fault in any reply a read needs is returned, and that the
// read does not go on to the next reply. A failed exchange with a reply is unreachable for a read.
func TestKiwiReadsReturnTheFaultOfTheReplyTheyNeed(t *testing.T) {
	tests := []struct {
		name    string
		op      domain.OperationID
		args    map[string]any
		answers map[string]scriptedAnswer
		want    error
	}{
		{
			name:    "the product list fails when the product reply fails",
			op:      domain.OpKiwiProductsList,
			args:    map[string]any{},
			answers: map[string]scriptedAnswer{methodProductFilter: {fault: errScripted}},
			want:    domain.ErrUnreachable,
		},
		{
			name:    "the product tree fails when the product reply fails",
			op:      domain.OpKiwiProductsTree,
			args:    map[string]any{"collectionId": "product-1"},
			answers: map[string]scriptedAnswer{methodProductFilter: {fault: errScripted}},
			want:    domain.ErrUnreachable,
		},
		{
			name:    "a product reply that is not a list is unexpected",
			op:      domain.OpKiwiProductsList,
			args:    map[string]any{},
			answers: map[string]scriptedAnswer{methodProductFilter: sReply(map[string]any{"id": 1})},
			want:    domain.ErrUnexpectedResponse,
		},
		{
			name: "a product list row without an ID is unexpected",
			op:   domain.OpKiwiProductsList,
			args: map[string]any{},
			answers: map[string]scriptedAnswer{
				methodProductFilter: sReply([]any{map[string]any{"name": "Nameless"}}),
			},
			want: domain.ErrUnexpectedResponse,
		},
		{
			name: "a product tree fails when the plan reply fails",
			op:   domain.OpKiwiProductsTree,
			args: map[string]any{"collectionId": "product-1"},
			answers: map[string]scriptedAnswer{
				methodProductFilter:  sReply([]any{productRow(1, "Portal")}),
				methodTestPlanFilter: {fault: errScripted},
			},
			want: domain.ErrUnreachable,
		},
		{
			name: "a document read fails when the case reply fails",
			op:   domain.OpKiwiDocumentsGet,
			args: map[string]any{"documentId": "case-12"},
			answers: map[string]scriptedAnswer{
				methodTestCaseFilter: {fault: errScripted},
			},
			want: domain.ErrUnreachable,
		},
		{
			name: "a document read fails when the plan reply fails",
			op:   domain.OpKiwiDocumentsGet,
			args: map[string]any{"documentId": "plan-2"},
			answers: map[string]scriptedAnswer{
				methodTestPlanFilter: {fault: errScripted},
			},
			want: domain.ErrUnreachable,
		},
		{
			name: "a plan document with no product is unexpected",
			op:   domain.OpKiwiDocumentsGet,
			args: map[string]any{"documentId": "plan-2"},
			answers: map[string]scriptedAnswer{
				methodTestPlanFilter: sReply([]any{map[string]any{"id": 2, "name": "Login"}}),
			},
			want: domain.ErrUnexpectedResponse,
		},
		{
			name: "a case document fails when the lookup tables fail",
			op:   domain.OpKiwiDocumentsGet,
			args: map[string]any{"documentId": "case-12"},
			answers: map[string]scriptedAnswer{
				methodTestCaseFilter: sReply([]any{caseRow(12, "Login works")}),
				methodCategoryFilter: {fault: errScripted},
			},
			want: domain.ErrUnreachable,
		},
		{
			name: "a case document fails when the priority table fails",
			op:   domain.OpKiwiDocumentsGet,
			args: map[string]any{"documentId": "case-12"},
			answers: map[string]scriptedAnswer{
				methodTestCaseFilter: sReply([]any{caseRow(12, "Login works")}),
				methodPriorityFilter: {fault: errScripted},
			},
			want: domain.ErrUnreachable,
		},
		{
			name: "a case document fails when the status table fails",
			op:   domain.OpKiwiDocumentsGet,
			args: map[string]any{"documentId": "case-12"},
			answers: map[string]scriptedAnswer{
				methodTestCaseFilter:       sReply([]any{caseRow(12, "Login works")}),
				methodTestCaseStatusFilter: {fault: errScripted},
			},
			want: domain.ErrUnreachable,
		},
		{
			name: "a case document of a missing case is not found",
			op:   domain.OpKiwiDocumentsGet,
			args: map[string]any{"documentId": "case-12"},
			answers: map[string]scriptedAnswer{
				methodTestCaseFilter: sReply([]any{}),
			},
			want: domain.ErrDocumentNotFound,
		},
		{
			name: "a plan tree with a parent loop fails when a case reply in the loop fails",
			op:   domain.OpKiwiProductsTree,
			args: map[string]any{"collectionId": "product-1"},
			answers: map[string]scriptedAnswer{
				methodProductFilter:  sReply([]any{productRow(1, "Portal")}),
				methodTestPlanFilter: sReply([]any{planRow(1, 1, intPtr(2)), planRow(2, 1, intPtr(1))}),
				methodTestCaseFilter: {fault: errScripted},
			},
			want: domain.ErrUnreachable,
		},
		{
			name: "a plan tree fails when the sort-key reply has the wrong shape",
			op:   domain.OpKiwiProductsTree,
			args: map[string]any{"collectionId": "product-1"},
			answers: map[string]scriptedAnswer{
				methodProductFilter:    sReply([]any{productRow(1, "Portal")}),
				methodTestPlanFilter:   sReply([]any{planRow(2, 1, nil)}),
				methodTestCaseFilter:   sReply([]any{caseRow(12, "Login works")}),
				methodTestCaseSortkeys: sReply([]any{}),
			},
			want: domain.ErrUnexpectedResponse,
		},
		{
			name: "a plan tree fails when the sort-key reply fails",
			op:   domain.OpKiwiProductsTree,
			args: map[string]any{"collectionId": "product-1"},
			answers: map[string]scriptedAnswer{
				methodProductFilter:    sReply([]any{productRow(1, "Portal")}),
				methodTestPlanFilter:   sReply([]any{planRow(2, 1, nil)}),
				methodTestCaseFilter:   sReply([]any{caseRow(12, "Login works")}),
				methodTestCaseSortkeys: {fault: errScripted},
			},
			want: domain.ErrUnreachable,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gw := newScriptedGateway(t, domain.AccessRead, &scriptedRT{answer: scriptFor(tt.answers)})
			_, err := gw.Read(context.Background(), tt.op, domain.NewArgs(tt.args))
			if !errors.Is(err, tt.want) {
				t.Fatalf("Read error = %v, want %v", err, tt.want)
			}
		})
	}
}

// TestKiwiIDsThatAreNotCanonicalAreRefusedBeforeAnyRequest checks the kiwi number rules that sit under the domain ID
// rules: a product or a plan named with a leading zero is refused, so no request is sent for it.
func TestKiwiIDsThatAreNotCanonicalAreRefusedBeforeAnyRequest(t *testing.T) {
	rt := &scriptedRT{answer: scriptFor(nil)}
	gw := newScriptedGateway(t, domain.AccessRead, rt)
	if _, err := gw.CollectionTree(context.Background(), mustCollectionID(t, "product-0")); !errors.Is(err, domain.ErrInvalidCollectionID) {
		t.Fatalf("CollectionTree(product-0) error = %v, want ErrInvalidCollectionID", err)
	}
	if _, err := gw.Document(context.Background(), mustDocumentID(t, "plan-0")); !errors.Is(err, domain.ErrInvalidDocumentID) {
		t.Fatalf("Document(plan-0) error = %v, want ErrInvalidDocumentID", err)
	}
	if len(rt.methods) != 0 {
		t.Fatalf("a non-canonical ID sent %v", rt.methods)
	}
}

// TestKiwiUpdateRefusesAPlanNamedAsACase checks that an update names a case, so a plan ID is refused before any request.
func TestKiwiUpdateRefusesAPlanNamedAsACase(t *testing.T) {
	rt := &scriptedRT{answer: scriptFor(nil)}
	gw := newScriptedGateway(t, domain.AccessWrite, rt)
	_, err := gw.Write(context.Background(), domain.OpKiwiTestCasesUpdate, domain.NewArgs(map[string]any{"documentId": "plan-2", "summary": "x"}))
	if !errors.Is(err, domain.ErrInvalidDocumentID) {
		t.Fatalf("update of a plan error = %v, want ErrInvalidDocumentID", err)
	}
	if len(rt.methods) != 0 {
		t.Fatalf("an update of a plan sent %v", rt.methods)
	}
}

// TestKiwiWriteOfAnotherKindsOperationIsUnsupported checks that a valid write of another kind's operation is not sent to
// Kiwi: the writer has no rule for it and refuses it.
func TestKiwiWriteOfAnotherKindsOperationIsUnsupported(t *testing.T) {
	rt := &scriptedRT{answer: scriptFor(nil)}
	gw := newScriptedGateway(t, domain.AccessWrite, rt)
	_, err := gw.Write(context.Background(), domain.OpDocumentsAddGroup, domain.NewArgs(map[string]any{
		"documentId": "doc-1", "groupId": "123e4567-e89b-12d3-a456-426614174000",
	}))
	if !errors.Is(err, domain.ErrUnsupportedOperation) {
		t.Fatalf("Write error = %v, want ErrUnsupportedOperation", err)
	}
	if len(rt.methods) != 0 {
		t.Fatalf("another kind's write sent %v", rt.methods)
	}
}

// TestKiwiTreeOrdersCasesWithTheSameSortKeyByID checks that two cases with one sort key are listed in ID order, whatever
// order the server sent them in.
func TestKiwiTreeOrdersCasesWithTheSameSortKeyByID(t *testing.T) {
	gw := newScriptedGateway(t, domain.AccessRead, &scriptedRT{answer: scriptFor(map[string]scriptedAnswer{
		methodProductFilter:    sReply([]any{productRow(1, "Portal")}),
		methodTestPlanFilter:   sReply([]any{planRow(2, 1, nil)}),
		methodTestCaseFilter:   sReply([]any{caseRow(13, "Logout works"), caseRow(12, "Login works")}),
		methodTestCaseSortkeys: sReply(map[string]any{"12": 1, "13": 1}),
	})})
	nodes, err := gw.CollectionTree(context.Background(), mustCollectionID(t, "product-1"))
	if err != nil {
		t.Fatalf("CollectionTree: %v", err)
	}
	var order []string
	for _, n := range nodes {
		order = append(order, n.ID)
	}
	if strings.Join(order, ",") != "plan-2,case-12,case-13" {
		t.Fatalf("tree order = %v, want plan-2, case-12, case-13", order)
	}
}

// TestKiwiWritesRefuseBadArgumentsAndReadOperations checks that a write names one of the Kiwi write operations, and that
// bad arguments are refused before any request.
func TestKiwiWritesRefuseBadArgumentsAndReadOperations(t *testing.T) {
	tests := []struct {
		name string
		op   domain.OperationID
		args map[string]any
		want error
	}{
		{name: "a create with no arguments is refused", op: domain.OpKiwiTestCasesCreate, args: map[string]any{}},
		{name: "an update with no case is refused", op: domain.OpKiwiTestCasesUpdate, args: map[string]any{}},
		{name: "an add-case with no plan is refused", op: domain.OpKiwiPlansAddCase, args: map[string]any{"caseId": "case-13"}},
		{name: "an add-case with a bad plan is refused", op: domain.OpKiwiPlansAddCase, args: map[string]any{"planId": "plan-x", "caseId": "case-13"}},
		{name: "an add-case with a bad case is refused", op: domain.OpKiwiPlansAddCase, args: map[string]any{"planId": "plan-1", "caseId": "case-x"}},
		{name: "a read operation is the wrong class to write", op: domain.OpKiwiProductsList, args: map[string]any{}, want: domain.ErrWrongOperationClass},
		{name: "an unknown operation is unsupported", op: "kiwi.nope", args: map[string]any{}, want: domain.ErrUnsupportedOperation},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := &scriptedRT{answer: scriptFor(nil)}
			gw := newScriptedGateway(t, domain.AccessWrite, rt)
			_, err := gw.Write(context.Background(), tt.op, domain.NewArgs(tt.args))
			if err == nil {
				t.Fatalf("Write returned no error")
			}
			if tt.want != nil && !errors.Is(err, tt.want) {
				t.Fatalf("Write error = %v, want %v", err, tt.want)
			}
			if len(rt.methods) != 0 {
				t.Fatalf("a refused write sent %v", rt.methods)
			}
		})
	}
}

// createAnswers are the replies a case create needs: each name resolves to one row, and the new case is sent back. The
// extra replies replace the defaults for the same method.
func createAnswers(extra map[string]scriptedAnswer) map[string]scriptedAnswer {
	answers := map[string]scriptedAnswer{
		methodCategoryFilter:       sReply([]any{map[string]any{"id": 1, "name": "Functional", "product": 1}}),
		methodPriorityFilter:       sReply([]any{map[string]any{"id": 2, "value": "P1"}}),
		methodTestCaseStatusFilter: sReply([]any{map[string]any{"id": 1, "name": "CONFIRMED"}}),
		methodTestCaseCreate:       sReply(map[string]any{"id": 14, "summary": "New case"}),
	}
	for method, ans := range extra {
		answers[method] = ans
	}
	return answers
}

// createArgs is a valid create of one case in product 1, with the optional status and text.
func createArgs() map[string]any {
	return map[string]any{
		"summary": "New case", "product": "product-1", "category": "Functional", "priority": "P1",
		"status": "CONFIRMED", "text": "Steps",
	}
}

// TestKiwiCreateFaultsAreReturnedBeforeTheWrite checks each name lookup of a create: a lookup fault stops the create before
// the write is sent. A write whose exchange fails, or whose reply does not name the new case, has an unknown outcome.
func TestKiwiCreateFaultsAreReturnedBeforeTheWrite(t *testing.T) {
	tests := []struct {
		name    string
		answers map[string]scriptedAnswer
		want    error
		write   bool
	}{
		{name: "a category lookup fault stops the create", answers: map[string]scriptedAnswer{methodCategoryFilter: {fault: errScripted}}, want: domain.ErrUnreachable},
		{name: "a priority lookup fault stops the create", answers: map[string]scriptedAnswer{methodPriorityFilter: {fault: errScripted}}, want: domain.ErrUnreachable},
		{name: "a status lookup fault stops the create", answers: map[string]scriptedAnswer{methodTestCaseStatusFilter: {fault: errScripted}}, want: domain.ErrUnreachable},
		{name: "a status that matches no row stops the create", answers: map[string]scriptedAnswer{methodTestCaseStatusFilter: sReply([]any{})}, want: domain.ErrUnresolvedName},
		{name: "a priority with no ID is unexpected", answers: map[string]scriptedAnswer{methodPriorityFilter: sReply([]any{map[string]any{"value": "P1"}})}, want: domain.ErrUnexpectedResponse},
		{name: "a write whose exchange fails has an unknown outcome", answers: map[string]scriptedAnswer{methodTestCaseCreate: {fault: errScripted}}, want: domain.ErrWriteOutcomeUnknown, write: true},
		{name: "a reply without the new case ID is an unknown outcome", answers: map[string]scriptedAnswer{methodTestCaseCreate: sReply(map[string]any{"summary": "x"})}, want: domain.ErrWriteOutcomeUnknown, write: true},
		{name: "a reply that is not an object is an unknown outcome", answers: map[string]scriptedAnswer{methodTestCaseCreate: sReply([]any{})}, want: domain.ErrWriteOutcomeUnknown, write: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := &scriptedRT{answer: scriptFor(createAnswers(tt.answers))}
			gw := newScriptedGateway(t, domain.AccessWrite, rt)
			_, err := gw.Write(context.Background(), domain.OpKiwiTestCasesCreate, domain.NewArgs(createArgs()))
			if !errors.Is(err, tt.want) {
				t.Fatalf("Write error = %v, want %v", err, tt.want)
			}
			if got := sentWrite(rt.methods, methodTestCaseCreate); got != tt.write {
				t.Fatalf("the create sent its write = %v, want %v (methods %v)", got, tt.write, rt.methods)
			}
		})
	}
}

// TestKiwiUpdateFaultsAreReturnedBeforeTheWrite checks each step of an update: the case lookup, the status and priority
// lookups, the write, and the reply check. A fault before the write means no write is sent.
func TestKiwiUpdateFaultsAreReturnedBeforeTheWrite(t *testing.T) {
	caseFound := sReply([]any{caseRow(12, "Login works")})
	tests := []struct {
		name    string
		args    map[string]any
		answers map[string]scriptedAnswer
		want    error
		write   bool
	}{
		{name: "an update of a missing case is not found", args: map[string]any{"documentId": "case-12", "summary": "Renamed"}, answers: map[string]scriptedAnswer{methodTestCaseFilter: sReply([]any{})}, want: domain.ErrDocumentNotFound},
		{name: "a case lookup fault stops the update", args: map[string]any{"documentId": "case-12", "summary": "Renamed"}, answers: map[string]scriptedAnswer{methodTestCaseFilter: {fault: errScripted}}, want: domain.ErrUnreachable},
		{name: "a text update sends the text", args: map[string]any{"documentId": "case-12", "text": "New steps"}, answers: map[string]scriptedAnswer{methodTestCaseFilter: caseFound, methodTestCaseUpdate: sReply(map[string]any{"id": 12})}, write: true},
		{name: "a status lookup fault stops the update", args: map[string]any{"documentId": "case-12", "status": "CONFIRMED"}, answers: map[string]scriptedAnswer{methodTestCaseFilter: caseFound, methodTestCaseStatusFilter: {fault: errScripted}}, want: domain.ErrUnreachable},
		{name: "a priority lookup fault stops the update", args: map[string]any{"documentId": "case-12", "priority": "P1"}, answers: map[string]scriptedAnswer{methodTestCaseFilter: caseFound, methodPriorityFilter: {fault: errScripted}}, want: domain.ErrUnreachable},
		{name: "a write whose exchange fails has an unknown outcome", args: map[string]any{"documentId": "case-12", "summary": "Renamed"}, answers: map[string]scriptedAnswer{methodTestCaseFilter: caseFound, methodTestCaseUpdate: {fault: errScripted}}, want: domain.ErrWriteOutcomeUnknown, write: true},
		{name: "a reply that names another case is an unknown outcome", args: map[string]any{"documentId": "case-12", "summary": "Renamed"}, answers: map[string]scriptedAnswer{methodTestCaseFilter: caseFound, methodTestCaseUpdate: sReply(map[string]any{"id": 13})}, want: domain.ErrWriteOutcomeUnknown, write: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := &scriptedRT{answer: scriptFor(tt.answers)}
			gw := newScriptedGateway(t, domain.AccessWrite, rt)
			_, err := gw.Write(context.Background(), domain.OpKiwiTestCasesUpdate, domain.NewArgs(tt.args))
			if tt.want == nil && err != nil {
				t.Fatalf("Write error = %v, want none", err)
			}
			if tt.want != nil && !errors.Is(err, tt.want) {
				t.Fatalf("Write error = %v, want %v", err, tt.want)
			}
			if got := sentWrite(rt.methods, methodTestCaseUpdate); got != tt.write {
				t.Fatalf("the update sent its write = %v, want %v (methods %v)", got, tt.write, rt.methods)
			}
		})
	}
}

// TestKiwiAddCaseWriteFaultHasAnUnknownOutcome checks that an add-case whose exchange fails is an unknown outcome, so the
// caller never retries it.
func TestKiwiAddCaseWriteFaultHasAnUnknownOutcome(t *testing.T) {
	gw := newScriptedGateway(t, domain.AccessWrite, &scriptedRT{answer: scriptFor(map[string]scriptedAnswer{
		methodTestPlanAddCase: {fault: errScripted},
	})})
	_, err := gw.Write(context.Background(), domain.OpKiwiPlansAddCase, domain.NewArgs(map[string]any{"planId": "plan-1", "caseId": "case-13"}))
	if !errors.Is(err, domain.ErrWriteOutcomeUnknown) {
		t.Fatalf("add-case error = %v, want ErrWriteOutcomeUnknown", err)
	}
}

// TestKiwiRetryAfterValuesAreReadOrRefused checks Retry-After on a 429 reply. A usable value up to 60 seconds is waited;
// a missing or unusable one is not retried; a longer one is refused as too long. Each gives up after the one request.
func TestKiwiRetryAfterValuesAreReadOrRefused(t *testing.T) {
	tests := []struct {
		name       string
		retryAfter string
		want       error
	}{
		{name: "a missing Retry-After is rate limited without a retry", retryAfter: "", want: domain.ErrSourceRateLimited},
		{name: "a text Retry-After is rate limited without a retry", retryAfter: "soon", want: domain.ErrSourceRateLimited},
		{name: "a negative Retry-After is rate limited without a retry", retryAfter: "-1", want: domain.ErrSourceRateLimited},
		{name: "a Retry-After over 60 seconds is refused", retryAfter: "61", want: domain.ErrRetryAfterTooLong},
		{name: "a Retry-After too large to hold is refused", retryAfter: "99999999999999999999999", want: domain.ErrRetryAfterTooLong},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := &scriptedRT{answer: scriptFor(map[string]scriptedAnswer{
				methodProductFilter: {status: http.StatusTooManyRequests, retryAfter: tt.retryAfter},
			})}
			gw := newScriptedGateway(t, domain.AccessRead, rt)
			_, err := gw.Read(context.Background(), domain.OpKiwiProductsList, domain.NewArgs(map[string]any{}))
			if !errors.Is(err, tt.want) {
				t.Fatalf("Read error = %v, want %v", err, tt.want)
			}
			if len(rt.methods) != 1 {
				t.Fatalf("requests = %d, want 1 with no retry", len(rt.methods))
			}
		})
	}
}

// TestKiwiRetryWaitFaultIsReturned checks that a fault while a Retry-After wait is pending stops the call with that fault,
// and that no retry is sent.
func TestKiwiRetryWaitFaultIsReturned(t *testing.T) {
	rt := &scriptedRT{answer: scriptFor(map[string]scriptedAnswer{
		methodProductFilter: {status: http.StatusTooManyRequests, retryAfter: "1"},
	})}
	gw := newScriptedGateway(t, domain.AccessRead, rt, WithSleep(func(context.Context, time.Duration) error { return errScripted }))
	_, err := gw.Read(context.Background(), domain.OpKiwiProductsList, domain.NewArgs(map[string]any{}))
	if !errors.Is(err, errScripted) {
		t.Fatalf("Read error = %v, want the wait fault", err)
	}
	if len(rt.methods) != 1 {
		t.Fatalf("requests = %d, want 1 with no retry after the wait fault", len(rt.methods))
	}
}

// TestKiwiPacingFaultStopsTheNextRequest checks that a fault while the pacing wait is pending stops the request, so the
// second request is never sent.
func TestKiwiPacingFaultStopsTheNextRequest(t *testing.T) {
	rt := &scriptedRT{answer: scriptFor(map[string]scriptedAnswer{methodProductFilter: sReply([]any{})})}
	gw := newScriptedGateway(t, domain.AccessRead, rt,
		WithMinDelay(time.Second),
		WithSleep(func(context.Context, time.Duration) error { return errScripted }))
	if _, err := gw.Read(context.Background(), domain.OpKiwiProductsList, domain.NewArgs(map[string]any{})); err != nil {
		t.Fatalf("first Read: %v", err)
	}
	_, err := gw.Read(context.Background(), domain.OpKiwiProductsList, domain.NewArgs(map[string]any{}))
	if !errors.Is(err, errScripted) {
		t.Fatalf("second Read error = %v, want the pacing fault", err)
	}
	if len(rt.methods) != 1 {
		t.Fatalf("requests = %d, want 1 when pacing stops the second", len(rt.methods))
	}
}

// TestKiwiPacingNeverWaitsWhenTheGapHasPassed checks the pacing rule when the gap has already passed: the second request
// goes out with no wait at all.
func TestKiwiPacingNeverWaitsWhenTheGapHasPassed(t *testing.T) {
	now := epoch
	clock := func() time.Time {
		now = now.Add(time.Hour)
		return now
	}
	slept := 0
	rt := &scriptedRT{answer: scriptFor(map[string]scriptedAnswer{methodProductFilter: sReply([]any{})})}
	gw := newScriptedGateway(t, domain.AccessRead, rt,
		WithMinDelay(time.Second),
		WithClock(clock),
		WithSleep(func(context.Context, time.Duration) error { slept++; return nil }))
	for i := 0; i < 2; i++ {
		if _, err := gw.Read(context.Background(), domain.OpKiwiProductsList, domain.NewArgs(map[string]any{})); err != nil {
			t.Fatalf("Read %d: %v", i, err)
		}
	}
	if slept != 0 {
		t.Fatalf("pacing slept %d times, want none when the gap has passed", slept)
	}
}

// TestKiwiTransportFaultsMapToTheirSentinels checks the mapping of a failed exchange for a read: a cancelled caller, a
// timeout, and any other fault each get their own error.
func TestKiwiTransportFaultsMapToTheirSentinels(t *testing.T) {
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	tests := []struct {
		name  string
		ctx   context.Context
		fault error
		want  error
	}{
		{name: "a cancelled caller gets its own cancellation", ctx: cancelled, fault: errScripted, want: context.Canceled},
		{name: "a timeout is timed out", ctx: context.Background(), fault: timeoutError{}, want: domain.ErrTimedOut},
		{name: "a deadline is timed out", ctx: context.Background(), fault: context.DeadlineExceeded, want: domain.ErrTimedOut},
		{name: "any other fault is unreachable", ctx: context.Background(), fault: errScripted, want: domain.ErrUnreachable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gw := newScriptedGateway(t, domain.AccessRead, &scriptedRT{answer: scriptFor(map[string]scriptedAnswer{
				methodProductFilter: {fault: tt.fault},
			})})
			_, err := gw.Read(tt.ctx, domain.OpKiwiProductsList, domain.NewArgs(map[string]any{}))
			if !errors.Is(err, tt.want) {
				t.Fatalf("Read error = %v, want %v", err, tt.want)
			}
		})
	}
}

// TestKiwiBodyFaultAfterAStatusIsUnreachableForARead checks a reply body that fails after its status: a read that cannot
// finish reading is unreachable.
func TestKiwiBodyFaultAfterAStatusIsUnreachableForARead(t *testing.T) {
	gw := newScriptedGateway(t, domain.AccessRead, &scriptedRT{answer: scriptFor(map[string]scriptedAnswer{
		methodProductFilter: {bodyErr: errScripted},
	})})
	if _, err := gw.Read(context.Background(), domain.OpKiwiProductsList, domain.NewArgs(map[string]any{})); !errors.Is(err, domain.ErrUnreachable) {
		t.Fatalf("Read error = %v, want ErrUnreachable", err)
	}
}

// TestKiwiBodyFaultAfterAStatusIsAnUnknownOutcomeForAWrite checks the same body fault on a write: Kiwi answered its status,
// so the write may already have run, and the outcome is unknown.
func TestKiwiBodyFaultAfterAStatusIsAnUnknownOutcomeForAWrite(t *testing.T) {
	gw := newScriptedGateway(t, domain.AccessWrite, &scriptedRT{answer: scriptFor(map[string]scriptedAnswer{
		methodTestPlanAddCase: {bodyErr: errScripted},
	})})
	_, err := gw.Write(context.Background(), domain.OpKiwiPlansAddCase, domain.NewArgs(map[string]any{"planId": "plan-1", "caseId": "case-13"}))
	if !errors.Is(err, domain.ErrWriteOutcomeUnknown) {
		t.Fatalf("Write error = %v, want ErrWriteOutcomeUnknown", err)
	}
}

// TestKiwiTLSHandshakeFaultIsUnreachableEvenForAWrite checks that a connection that fails its TLS handshake sent no
// request, so a write over it has a known outcome. The connection is an in-process pipe whose far end is closed.
func TestKiwiTLSHandshakeFaultIsUnreachableEvenForAWrite(t *testing.T) {
	pipeDial := func(context.Context, string, string) (net.Conn, error) {
		client, server := net.Pipe()
		if err := server.Close(); err != nil {
			return nil, err
		}
		return client, nil
	}
	gw := newScriptedGateway(t, domain.AccessWrite, &http.Transport{DialContext: pipeDial})
	_, err := gw.Write(context.Background(), domain.OpKiwiPlansAddCase, domain.NewArgs(map[string]any{"planId": "plan-1", "caseId": "case-13"}))
	if !errors.Is(err, domain.ErrUnreachable) {
		t.Fatalf("Write error = %v, want ErrUnreachable for a failed handshake", err)
	}
}

// TestKiwiClientRefusesAForbiddenArgumentAndAnUnencodableOne checks the two guards that run before a request leaves: a
// forbidden argument name at any depth, and a value that JSON cannot hold. Neither sends a request.
func TestKiwiClientRefusesAForbiddenArgumentAndAnUnencodableOne(t *testing.T) {
	tests := []struct {
		name   string
		params map[string]any
	}{
		{name: "a forbidden key is refused", params: map[string]any{"token": "x"}},
		{name: "a value JSON cannot encode is refused", params: map[string]any{"query": math.NaN()}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := &scriptedRT{answer: scriptFor(nil)}
			gw := newScriptedGateway(t, domain.AccessRead, rt)
			if _, err := gw.call(context.Background(), methodProductFilter, tt.params); err == nil {
				t.Fatalf("call returned no error")
			}
			if len(rt.methods) != 0 {
				t.Fatalf("a refused call sent %v", rt.methods)
			}
		})
	}
}

// TestKiwiRequestWithAnInvalidURLIsInvalidBaseURL checks that a base URL the request builder rejects is reported as an
// invalid base, and no request is sent.
func TestKiwiRequestWithAnInvalidURLIsInvalidBaseURL(t *testing.T) {
	rt := &scriptedRT{answer: scriptFor(nil)}
	gw := newScriptedGateway(t, domain.AccessRead, rt)
	gw.base = "https://bad\x7fhost"
	if _, err := gw.Read(context.Background(), domain.OpKiwiProductsList, domain.NewArgs(map[string]any{})); !errors.Is(err, errInvalidBaseURL) {
		t.Fatalf("Read error = %v, want errInvalidBaseURL", err)
	}
	if len(rt.methods) != 0 {
		t.Fatalf("an invalid URL sent %v", rt.methods)
	}
}

// TestKiwiSleepHonoursItsContext checks sleepContext: a cancelled context ends the wait at once, and a zero wait returns
// at once.
func TestKiwiSleepHonoursItsContext(t *testing.T) {
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := sleepContext(cancelled, time.Hour); !errors.Is(err, context.Canceled) {
		t.Fatalf("sleepContext with a cancelled context = %v, want context.Canceled", err)
	}
	if err := sleepContext(context.Background(), 0); err != nil {
		t.Fatalf("sleepContext of zero = %v, want nil", err)
	}
}

// TestKiwiRetryAfterParsesWholeSeconds checks parseRetryAfter: a whole number of seconds is read, and a value too large to
// hold is reported one over the limit.
func TestKiwiRetryAfterParsesWholeSeconds(t *testing.T) {
	tests := []struct {
		name   string
		value  string
		want   int
		wantOK bool
	}{
		{name: "whole seconds are read", value: "12", want: 12, wantOK: true},
		{name: "a value too large to hold is one over the limit", value: "99999999999999999999999", want: maxRetryAfter + 1, wantOK: true},
		{name: "a negative value is unusable", value: "-3", want: 0, wantOK: false},
		{name: "text is unusable", value: "later", want: 0, wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseRetryAfter(tt.value)
			if got != tt.want || ok != tt.wantOK {
				t.Fatalf("parseRetryAfter(%q) = %d, %v; want %d, %v", tt.value, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

// otherTransport is a round tripper that is not an HTTP transport, so it has no proxy or TLS settings of its own.
type otherTransport struct{}

func (*otherTransport) RoundTrip(*http.Request) (*http.Response, error) { return nil, errScripted }

// TestKiwiTLSNameIsTheEndpointHostOnAnyTransport checks hardenTransport: the copy has no proxy and the endpoint host as
// its TLS name, whether or not the caller's transport had TLS settings. A missing transport gets the default copy, and a
// transport that is not an HTTP transport is returned as it is.
func TestKiwiTLSNameIsTheEndpointHostOnAnyTransport(t *testing.T) {
	host := "kiwi.example.test"
	bare := hardenTransport(&http.Transport{}, host).(*http.Transport)
	if bare.Proxy != nil || bare.TLSClientConfig == nil || bare.TLSClientConfig.ServerName != host {
		t.Fatalf("a transport with no TLS settings = proxy set %v, TLS %+v; want no proxy and the host as the TLS name", bare.Proxy != nil, bare.TLSClientConfig)
	}
	withTLS := hardenTransport(&http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}}, host).(*http.Transport)
	if withTLS.TLSClientConfig.ServerName != host || withTLS.TLSClientConfig.MinVersion != tls.VersionTLS12 {
		t.Fatalf("a transport with TLS settings lost them or the host: %+v", withTLS.TLSClientConfig)
	}
	def := hardenTransport(nil, host).(*http.Transport)
	if def.Proxy != nil {
		t.Fatalf("the default copy has a proxy")
	}
	other := &otherTransport{}
	if got := hardenTransport(other, host); got != http.RoundTripper(other) {
		t.Fatalf("a round tripper that is not a transport was changed")
	}
}

// roundTripFunc adapts a function to http.RoundTripper.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

// TestKiwiBaseURLRules checks CheckBaseURL: an HTTPS URL with a host and no user information is accepted; a URL that does
// not parse, has no host, or holds user information is an invalid base; any other scheme is refused as not HTTPS.
func TestKiwiBaseURLRules(t *testing.T) {
	tests := []struct {
		name string
		base string
		want error
	}{
		{name: "an HTTPS base with a host is accepted", base: "https://kiwi.example.test"},
		{name: "a base that does not parse is invalid", base: "https://[::1", want: errInvalidBaseURL},
		{name: "a base with no host is invalid", base: "https://", want: errInvalidBaseURL},
		{name: "a base with user information is invalid", base: "https://user@kiwi.example.test", want: errInvalidBaseURL},
		{name: "a plain HTTP base is refused", base: "http://kiwi.example.test", want: domain.ErrHTTPSRequired},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckBaseURL(tt.base)
			if tt.want == nil && err != nil {
				t.Fatalf("CheckBaseURL = %v, want accepted", err)
			}
			if tt.want != nil && !errors.Is(err, tt.want) {
				t.Fatalf("CheckBaseURL = %v, want %v", err, tt.want)
			}
		})
	}
}

// TestKiwiProductionTransportSurvivesAReplacedDefault checks productionTransport: when the process default transport is
// not an HTTP transport, the production transport is built from its own settings, still with no proxy.
func TestKiwiProductionTransportSurvivesAReplacedDefault(t *testing.T) {
	old := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, errScripted })
	t.Cleanup(func() { http.DefaultTransport = old })
	tr := productionTransport()
	if tr.Proxy != nil || tr.DialContext == nil || tr.TLSHandshakeTimeout == 0 {
		t.Fatalf("productionTransport with a replaced default = proxy set %v, dial set %v, TLS timeout %v", tr.Proxy != nil, tr.DialContext != nil, tr.TLSHandshakeTimeout)
	}
}
