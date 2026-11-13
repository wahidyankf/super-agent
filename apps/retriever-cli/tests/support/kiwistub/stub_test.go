package kiwistub

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

const testKey = "fixture-key-kiwistub"

// fixture is a small dataset: two products, a nested plan, and cases in one and two plans.
func fixture() Dataset {
	return Dataset{
		Products: []Product{{ID: 1, Name: "Example Portal"}, {ID: 2, Name: "Empty"}},
		Plans: []Plan{
			{ID: 1, Name: "Smoke", Product: 1, Text: "Run before each release."},
			{ID: 2, Name: "Login", Product: 1, Parent: 1},
		},
		Cases: []Case{
			{ID: 12, Summary: "Login works", Text: "Open the login page.", Plans: []int{2}, Sortkey: 1, Status: 1, Priority: 2, Category: 1},
			{ID: 13, Summary: "Logout works", Text: "Log out.", Plans: []int{2}, Sortkey: 2, Status: 1, Priority: 2, Category: 1},
		},
		Categories: []Category{{ID: 1, Name: "Functional", Product: 1}, {ID: 4, Name: "Regression", Product: 1}, {ID: 5, Name: "Regression", Product: 1}},
		Priorities: []Priority{{ID: 2, Value: "P1"}},
		Statuses:   []Status{{ID: 1, Name: "CONFIRMED"}},
	}
}

// call posts one JSON-RPC request through the in-process transport and returns the status, header, and body.
func call(t *testing.T, s *Stub, method string, params map[string]any) (int, http.Header, []byte) {
	t.Helper()
	body, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "method": method, "params": params, "id": 7})
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPost, "https://kiwi.example.test"+Endpoint, strings.NewReader(string(body)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+testKey)
	resp, err := s.RoundTripper().RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, resp.Header, raw
}

// result decodes the result of a 200 reply, and fails the test when the reply holds an error.
func result(t *testing.T, raw []byte) any {
	t.Helper()
	var env struct {
		ID     int64 `json:"id"`
		Result any   `json:"result"`
		Error  any   `json:"error"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("reply is not JSON: %v: %s", err, raw)
	}
	if env.Error != nil {
		t.Fatalf("reply holds an error: %v", env.Error)
	}
	if env.ID != 7 {
		t.Fatalf("reply id = %d, want 7", env.ID)
	}
	return env.Result
}

// errorOf decodes the JSON-RPC error of a 200 reply.
func errorOf(t *testing.T, raw []byte) (int, string) {
	t.Helper()
	var env struct {
		Error struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("reply is not JSON: %v", err)
	}
	return env.Error.Code, env.Error.Message
}

// TestStubAnswersTheFilterMethodsOverTheDataset checks the envelope and the reads of the fixture.
func TestStubAnswersTheFilterMethodsOverTheDataset(t *testing.T) {
	s := New(testKey, WithDataset(fixture()))
	status, _, raw := call(t, s, "Product.filter", map[string]any{"query": map[string]any{}})
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	products := result(t, raw).([]any)
	if len(products) != 2 {
		t.Fatalf("Product.filter returned %d rows, want 2", len(products))
	}

	_, _, raw = call(t, s, "TestPlan.filter", map[string]any{"query": map[string]any{"id": 1}})
	plans := result(t, raw).([]any)
	if len(plans) != 1 || plans[0].(map[string]any)["parent"] != nil {
		t.Fatalf("TestPlan.filter(id 1) = %v, want one root plan with a null parent", plans)
	}

	_, _, raw = call(t, s, "TestCase.filter", map[string]any{"query": map[string]any{"plan": 2}})
	if got := len(result(t, raw).([]any)); got != 2 {
		t.Fatalf("TestCase.filter(plan 2) returned %d rows, want 2", got)
	}
	_, _, raw = call(t, s, "TestCase.sortkeys", map[string]any{"query": map[string]any{"plan": 2}})
	// The real TestCase.sortkeys reply is one object that maps each case ID, as a string key, to its sort key number.
	keys, isObject := result(t, raw).(map[string]any)
	if !isObject || len(keys) != 2 || keys["12"] != float64(1) || keys["13"] != float64(2) {
		t.Fatalf("TestCase.sortkeys(plan 2) = %v, want an object {\"12\": 1, \"13\": 2}", result(t, raw))
	}
	_, _, raw = call(t, s, "TestCase.sortkeys", map[string]any{"query": map[string]any{"plan": 1}})
	if keys, isObject := result(t, raw).(map[string]any); !isObject || len(keys) != 0 {
		t.Fatalf("TestCase.sortkeys(plan without cases) = %v, want an empty object", result(t, raw))
	}
	_, _, raw = call(t, s, "Category.filter", map[string]any{"query": map[string]any{"product": 1, "name": "Regression"}})
	if got := len(result(t, raw).([]any)); got != 2 {
		t.Fatalf("Category.filter(name Regression) returned %d rows, want 2 (the ambiguous case)", got)
	}
}

// TestStubRefusesAWrongKeyAndRecordsIt checks the Bearer check: a wrong key is 401, and the record keeps only the result of
// the check, never the header.
func TestStubRefusesAWrongKeyAndRecordsIt(t *testing.T) {
	s := New(testKey, WithDataset(fixture()))
	body := `{"jsonrpc":"2.0","method":"Product.filter","params":{"query":{}},"id":7}`
	req, _ := http.NewRequest(http.MethodPost, "https://kiwi.example.test"+Endpoint, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer wrong")
	resp, err := s.RoundTripper().RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
	reqs := s.Requests()
	if len(reqs) != 1 || reqs[0].AuthOK || reqs[0].Method != "Product.filter" || reqs[0].Host != "" && reqs[0].Host != "kiwi.example.test" {
		t.Fatalf("recorded = %+v, want one unauthorized Product.filter", reqs)
	}
}

// TestStubRefusesAForbiddenArgumentName checks that a key such as password is refused as an invalid parameter, and that
// nothing changes.
func TestStubRefusesAForbiddenArgumentName(t *testing.T) {
	s := New(testKey, WithDataset(fixture()))
	_, _, raw := call(t, s, "Product.filter", map[string]any{"query": map[string]any{"Password": "x"}})
	code, _ := errorOf(t, raw)
	if code != -32602 {
		t.Fatalf("code = %d, want -32602", code)
	}
}

// TestStubWritesCreateUpdateAndAddCase checks the three writes and their effect on the dataset.
func TestStubWritesCreateUpdateAndAddCase(t *testing.T) {
	s := New(testKey, WithDataset(fixture()))
	_, _, raw := call(t, s, "TestCase.create", map[string]any{"values": map[string]any{"summary": "Relogin", "category": 1, "priority": 2, "case_status": 1, "text": "Steps"}})
	created := result(t, raw).(map[string]any)
	if created["id"] != float64(14) {
		t.Fatalf("created id = %v, want 14 (one more than the largest)", created["id"])
	}
	_, _, raw = call(t, s, "TestCase.update", map[string]any{"case_id": 14, "values": map[string]any{"summary": "Relogin again"}})
	if got := result(t, raw).(map[string]any)["summary"]; got != "Relogin again" {
		t.Fatalf("updated summary = %v", got)
	}
	_, _, raw = call(t, s, "TestPlan.add_case", map[string]any{"plan_id": 1, "case_id": 14})
	result(t, raw)
	if s.Calls("TestPlan.add_case") != 1 {
		t.Fatalf("add_case calls = %d, want 1", s.Calls("TestPlan.add_case"))
	}
	_, _, raw = call(t, s, "TestPlan.add_case", map[string]any{"plan_id": 99, "case_id": 14})
	if _, msg := errorOf(t, raw); msg != "Plan does not exist" {
		t.Fatalf("add_case to a missing plan = %q", msg)
	}
	_, _, raw = call(t, s, "TestCase.create", map[string]any{"values": map[string]any{"text": "no summary"}})
	if _, msg := errorOf(t, raw); !strings.Contains(msg, "required") {
		t.Fatalf("create without a summary = %q, want a required-field message", msg)
	}
	_, _, raw = call(t, s, "TestCase.update", map[string]any{"case_id": 999, "values": map[string]any{"summary": "x"}})
	if _, msg := errorOf(t, raw); !strings.Contains(msg, "does not exist") {
		t.Fatalf("update of a missing case = %q", msg)
	}
}

// TestStubAnswersUnknownMethodsAndBadEnvelopes checks the method-not-found and invalid-request replies.
func TestStubAnswersUnknownMethodsAndBadEnvelopes(t *testing.T) {
	s := New(testKey, WithDataset(fixture()))
	_, _, raw := call(t, s, "Auth.login", map[string]any{})
	if code, _ := errorOf(t, raw); code != -32601 {
		t.Fatalf("Auth.login code = %d, want -32601", code)
	}
	req, _ := http.NewRequest(http.MethodPost, "https://kiwi.example.test"+Endpoint, strings.NewReader("not json"))
	req.Header.Set("Authorization", "Bearer "+testKey)
	resp, err := s.RoundTripper().RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ = io.ReadAll(resp.Body)
	if code, _ := errorOf(t, raw); code != -32600 {
		t.Fatalf("invalid JSON code = %d, want -32600", code)
	}
}

// TestStubEscapesTextUnlessRaw checks that Kiwi's replies escape their text, and that WithRawText turns it off.
func TestStubEscapesTextUnlessRaw(t *testing.T) {
	d := fixture()
	d.Cases[0].Summary = "Login & logout <b>"
	escaped := New(testKey, WithDataset(d))
	summaryOf := func(s *Stub) string {
		_, _, raw := call(t, s, "TestCase.filter", map[string]any{"query": map[string]any{"id": 12}})
		return result(t, raw).([]any)[0].(map[string]any)["summary"].(string)
	}
	if got := summaryOf(escaped); got != "Login &amp; logout &lt;b&gt;" {
		t.Fatalf("escaped summary = %q, want the HTML-escaped text", got)
	}
	if got := summaryOf(New(testKey, WithDataset(d), WithRawText())); got != "Login & logout <b>" {
		t.Fatalf("raw summary = %q, want the text as stored", got)
	}
}

// TestStubFaultsAnswerWhatEachFaultNames checks each fault of the stub, one at a time.
func TestStubFaultsAnswerWhatEachFaultNames(t *testing.T) {
	cases := []struct {
		name   string
		fault  Fault
		status int
		check  func(t *testing.T, body []byte, header http.Header)
	}{
		{"status 401", Fault{Status: 401}, 401, nil},
		{"status 500 echoes the key", Fault{Status: 500, EchoKey: true}, 500, func(t *testing.T, body []byte, _ http.Header) {
			if !strings.Contains(string(body), testKey) {
				t.Errorf("echo fault body does not echo the key")
			}
		}},
		{"429 with Retry-After", Fault{Status: 429, RetryAfter: "1"}, 429, func(t *testing.T, _ []byte, header http.Header) {
			if header.Get("Retry-After") != "1" {
				t.Errorf("Retry-After = %q, want 1", header.Get("Retry-After"))
			}
		}},
		{"redirect", Fault{Redirect: true}, 302, func(t *testing.T, _ []byte, header http.Header) {
			if header.Get("Location") == "" {
				t.Errorf("redirect has no Location")
			}
		}},
		{"oversize", Fault{Oversize: true}, 200, func(t *testing.T, body []byte, _ http.Header) {
			if len(body) <= 10<<20 {
				t.Errorf("oversize body is %d bytes, want more than 10 MiB", len(body))
			}
		}},
		{"not JSON", Fault{NotJSON: true, EchoKey: true}, 200, func(t *testing.T, body []byte, _ http.Header) {
			if json.Valid(body) || !strings.Contains(string(body), testKey) {
				t.Errorf("not-JSON body = %q, want invalid JSON that echoes the key", body)
			}
		}},
		{"wrong id", Fault{WrongID: true}, 200, func(t *testing.T, body []byte, _ http.Header) {
			var env struct {
				ID int64 `json:"id"`
			}
			_ = json.Unmarshal(body, &env)
			if env.ID == 7 {
				t.Errorf("wrong-id reply repeats the request id")
			}
		}},
		{"neither result nor error", Fault{NoResultError: true}, 200, func(t *testing.T, body []byte, _ http.Header) {
			if strings.Contains(string(body), `"result"`) || strings.Contains(string(body), `"error"`) {
				t.Errorf("reply has a result or an error: %s", body)
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := New(testKey, WithDataset(fixture()))
			s.SetFault(tc.fault)
			status, header, body := call(t, s, "Product.filter", map[string]any{"query": map[string]any{}})
			if status != tc.status {
				t.Fatalf("status = %d, want %d", status, tc.status)
			}
			if tc.check != nil {
				tc.check(t, body, header)
			}
		})
	}
}

// TestStubNeverAnswersUntilTheClientGivesUp checks the hold fault: the request ends only with its context.
func TestStubNeverAnswersUntilTheClientGivesUp(t *testing.T) {
	s := New(testKey, WithDataset(fixture()))
	s.SetFault(Fault{Never: true})
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://kiwi.example.test"+Endpoint, strings.NewReader("{}"))
	req.Header.Set("Authorization", "Bearer "+testKey)
	if _, err := s.RoundTripper().RoundTrip(req); err == nil {
		t.Fatalf("a request that is never answered returned a reply")
	}
	if s.Calls("") != 1 {
		t.Fatalf("the held request was not recorded")
	}
}

// TestStubFaultFirstAnswersOnlyTheNextRequest checks the one-shot fault that a retry test uses.
func TestStubFaultFirstAnswersOnlyTheNextRequest(t *testing.T) {
	s := New(testKey, WithDataset(fixture()))
	s.SetFaultFirst(Fault{Status: 429, RetryAfter: "1"})
	status, _, _ := call(t, s, "Product.filter", map[string]any{"query": map[string]any{}})
	if status != 429 {
		t.Fatalf("first status = %d, want 429", status)
	}
	status, _, _ = call(t, s, "Product.filter", map[string]any{"query": map[string]any{}})
	if status != 200 {
		t.Fatalf("second status = %d, want 200", status)
	}
}

// TestStubMethodFaultFirstAnswersOnlyTheNextCallOfThatMethod checks the one-shot fault of one method: a write that reads
// first must get its fault on the write, not on the lookup before it.
func TestStubMethodFaultFirstAnswersOnlyTheNextCallOfThatMethod(t *testing.T) {
	s := New(testKey, WithDataset(fixture()))
	s.SetMethodFaultFirst("TestCase.update", Fault{Status: 429, RetryAfter: "1"})
	status, _, _ := call(t, s, "Product.filter", map[string]any{"query": map[string]any{}})
	if status != 200 {
		t.Fatalf("a read before the faulted method got status %d, want 200", status)
	}
	status, _, _ = call(t, s, "TestCase.update", map[string]any{"values": map[string]any{}})
	if status != 429 {
		t.Fatalf("first call of the faulted method got status %d, want 429", status)
	}
	status, _, _ = call(t, s, "TestCase.update", map[string]any{"values": map[string]any{}})
	if status != 200 {
		t.Fatalf("second call of the faulted method got status %d, want 200", status)
	}
}

// TestStubMethodErrorAnswersOneMethod checks that a method error answers that method and no other.
func TestStubMethodErrorAnswersOneMethod(t *testing.T) {
	s := New(testKey, WithDataset(fixture()))
	s.SetMethodError("TestCase.create", -32603, "This field is required: summary.")
	_, _, raw := call(t, s, "TestCase.create", map[string]any{"values": map[string]any{"summary": "x"}})
	if code, msg := errorOf(t, raw); code != -32603 || msg != "This field is required: summary." {
		t.Fatalf("error = %d %q", code, msg)
	}
	_, _, raw = call(t, s, "Product.filter", map[string]any{"query": map[string]any{}})
	result(t, raw)
}

// TestStubRecordsEachRequestAndCountsInFlight checks the recorder: method, params, verb, path, host, and the peak count.
func TestStubRecordsEachRequestAndCountsInFlight(t *testing.T) {
	s := New(testKey, WithDataset(fixture()))
	call(t, s, "Product.filter", map[string]any{"query": map[string]any{"id": 1}})
	reqs := s.Requests()
	if len(reqs) != 1 {
		t.Fatalf("recorded %d requests, want 1", len(reqs))
	}
	got := reqs[0]
	if got.Verb != http.MethodPost || got.Path != Endpoint || got.Method != "Product.filter" || !got.AuthOK {
		t.Fatalf("recorded = %+v", got)
	}
	if q := got.Params["query"].(map[string]any); q["id"] != float64(1) {
		t.Fatalf("recorded query = %v", q)
	}
	if s.PeakInFlight() != 1 {
		t.Fatalf("peak = %d, want 1", s.PeakInFlight())
	}
}

// TestStubRejectsAnotherVerbAndPath checks that only POST /json-rpc/ is answered.
func TestStubRejectsAnotherVerbAndPath(t *testing.T) {
	s := New(testKey)
	if r := s.answer(http.MethodGet, Endpoint, "h", "", nil); r.status != http.StatusMethodNotAllowed {
		t.Fatalf("GET status = %d, want 405", r.status)
	}
	if r := s.answer(http.MethodPost, "/xml-rpc/", "h", "", nil); r.status != http.StatusNotFound {
		t.Fatalf("other path status = %d, want 404", r.status)
	}
}

// TestStubWithoutStartHasNoURLOrCertificate checks the zero state of a stub that never started.
func TestStubWithoutStartHasNoURLOrCertificate(t *testing.T) {
	s := New(testKey)
	if s.URL() != "" || s.CAPEM() != nil {
		t.Fatalf("an unstarted stub has a URL or a certificate")
	}
	if _, err := s.Client(); err == nil {
		t.Fatalf("Client of an unstarted stub returned no error")
	}
	s.Close()
}
