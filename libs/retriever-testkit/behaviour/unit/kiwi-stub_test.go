package unit_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"io"
	"net"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-testkit/kiwistub"
)

const (
	kiwiKey      = "EXAMPLE_KEY"
	kiwiEndpoint = "https://kiwi.example.test/json-rpc/"
)

// fakeListener is a listening socket that never accepts a connection. Start serves on it, so the stub opens no socket.
type fakeListener struct {
	closed chan struct{}
	once   sync.Once
}

func newFakeListener() *fakeListener {
	return &fakeListener{closed: make(chan struct{})}
}

// listen is the listen function of WithListener.
func (l *fakeListener) listen(_, _ string) (net.Listener, error) {
	return l, nil
}

func (l *fakeListener) Accept() (net.Conn, error) {
	<-l.closed
	return nil, net.ErrClosed
}

func (l *fakeListener) Close() error {
	l.once.Do(func() { close(l.closed) })
	return nil
}

func (l *fakeListener) Addr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 9443}
}

// newKiwiStub returns a stub with the synthetic key and the fixed clock.
func newKiwiStub(opts ...kiwistub.Option) *kiwistub.Stub {
	all := append([]kiwistub.Option{kiwistub.WithClock(fixedClock)}, opts...)
	return kiwistub.New(kiwiKey, all...)
}

// kiwiRequest posts one JSON-RPC call through the in-process transport of stub, under ctx.
func kiwiRequest(ctx context.Context, stub *kiwistub.Stub, auth, method string, params any) (int, map[string]any, error) {
	body, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
	if err != nil {
		return 0, nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, kiwiEndpoint, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	client := &http.Client{Transport: stub.RoundTripper()}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}
	var envelope map[string]any
	_ = json.Unmarshal(data, &envelope)
	return resp.StatusCode, envelope, nil
}

// kiwiCall posts one call with the synthetic key and fails the test on a transport error.
func kiwiCall(t *testing.T, stub *kiwistub.Stub, method string, params any) (int, map[string]any) {
	t.Helper()
	status, envelope, err := kiwiRequest(context.Background(), stub, "Bearer "+kiwiKey, method, params)
	if err != nil {
		t.Fatalf("call %s: %v", method, err)
	}
	return status, envelope
}

// resultRows returns the result of an envelope as a list of rows.
func resultRows(t *testing.T, envelope map[string]any) []any {
	t.Helper()
	rows, ok := envelope["result"].([]any)
	if !ok {
		t.Fatalf("the reply holds no result list: %v", envelope)
	}
	return rows
}

// @scenario specs/libs/retriever-testkit/behaviours/kiwi-stub.feature: The stub listens on the loopback and certifies the catalog hosts
func TestStubListensOnLoopbackAndCertifiesCatalogHosts(t *testing.T) {
	stub := newKiwiStub(
		kiwistub.WithListener(newFakeListener().listen),
		kiwistub.WithRandom(&fixedRandom{}),
	)
	if err := stub.Start(); err != nil {
		t.Fatalf("start the stub: %v", err)
	}
	defer stub.Close()
	if !strings.HasPrefix(stub.URL(), "https://127.0.0.1:") {
		t.Fatalf("the stub URL %q is not a loopback TLS address", stub.URL())
	}
	block, _ := pem.Decode(stub.CAPEM())
	if block == nil {
		t.Fatal("the CA file holds no PEM certificate")
	}
	leaf, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("parse the certificate: %v", err)
	}
	for _, host := range append([]string{"127.0.0.1"}, kiwistub.CatalogHosts...) {
		if err := leaf.VerifyHostname(host); err != nil {
			t.Fatalf("the certificate does not cover %s: %v", host, err)
		}
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/kiwi-stub.feature: A stub that was never started has no URL and no certificate
func TestStubNeverStartedHasNoURLAndNoCertificate(t *testing.T) {
	stub := newKiwiStub()
	if stub.URL() != "" {
		t.Fatalf("the URL of an unstarted stub is %q", stub.URL())
	}
	if stub.CAPEM() != nil {
		t.Fatal("an unstarted stub holds a certificate")
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/kiwi-stub.feature: A filter method answers the lookup table of its kind from the dataset
func TestFilterMethodAnswersLookupTableOfItsKind(t *testing.T) {
	cases := []struct {
		method string
		table  string
		want   int
	}{
		{"Product.filter", "products", 2},
		{"TestCase.filter", "cases", 1},
	}
	for _, c := range cases {
		t.Run(c.table, func(t *testing.T) {
			stub := newKiwiStub(kiwistub.WithDataset(kiwistub.Dataset{
				Products: []kiwistub.Product{{ID: 1, Name: "Example"}, {ID: 2, Name: "Second"}},
				Cases:    []kiwistub.Case{{ID: 1, Summary: "Login"}},
			}))
			status, envelope := kiwiCall(t, stub, c.method, map[string]any{"query": map[string]any{}})
			if status != http.StatusOK {
				t.Fatalf("status = %d, want 200", status)
			}
			if rows := resultRows(t, envelope); len(rows) != c.want {
				t.Fatalf("the %s answer holds %d rows, want %d", c.table, len(rows), c.want)
			}
		})
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/kiwi-stub.feature: A wrong Bearer key is refused with HTTP 401 and its digest is recorded
func TestWrongBearerKeyRefusedWith401AndDigestRecorded(t *testing.T) {
	stub := newKiwiStub()
	const header = "Bearer EXAMPLE_OTHER"
	status, _, err := kiwiRequest(context.Background(), stub, header, "Product.filter", map[string]any{"query": map[string]any{}})
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if status != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", status)
	}
	records := stub.Requests()
	if len(records) != 1 {
		t.Fatalf("the record holds %d requests, want 1", len(records))
	}
	sum := sha256.Sum256([]byte(header))
	if records[0].AuthDigest != hex.EncodeToString(sum[:]) {
		t.Fatalf("the recorded digest is %q", records[0].AuthDigest)
	}
	if records[0].AuthOK || strings.Contains(records[0].AuthDigest, "EXAMPLE_OTHER") {
		t.Fatal("the record keeps the header value or accepts the wrong key")
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/kiwi-stub.feature: A missing or non-Bearer header is refused when any key is accepted
func TestMissingOrNonBearerHeaderRefusedWhenAnyKeyAccepted(t *testing.T) {
	cases := []struct {
		header string
		auth   string
	}{
		{"", ""},
		{"Basic EXAMPLE_KEY", "Basic EXAMPLE_KEY"},
	}
	for _, c := range cases {
		t.Run(c.header, func(t *testing.T) {
			stub := newKiwiStub(kiwistub.WithAnyBearer())
			status, _, err := kiwiRequest(context.Background(), stub, c.auth, "Product.filter", map[string]any{"query": map[string]any{}})
			if err != nil {
				t.Fatalf("call: %v", err)
			}
			if status != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", status)
			}
		})
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/kiwi-stub.feature: A non-empty Bearer key of any value is accepted in any-key mode
func TestNonEmptyBearerKeyOfAnyValueAcceptedInAnyKeyMode(t *testing.T) {
	stub := newKiwiStub(kiwistub.WithAnyBearer())
	status, envelope, err := kiwiRequest(context.Background(), stub, "Bearer EXAMPLE_ANY", "Product.filter", map[string]any{"query": map[string]any{}})
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	resultRows(t, envelope)
}

// @scenario specs/libs/retriever-testkit/behaviours/kiwi-stub.feature: A call that names an argument the stub never accepts is refused
func TestCallNamingUnacceptedArgumentIsRefused(t *testing.T) {
	stub := newKiwiStub()
	_, envelope := kiwiCall(t, stub, "Product.filter", map[string]any{"query": map[string]any{"password": "EXAMPLE"}})
	if _, ok := envelope["error"]; !ok {
		t.Fatalf("the call with a refused argument was answered: %v", envelope)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/kiwi-stub.feature: A create, an update, and an add-case call write to the dataset
func TestCreateUpdateAndAddCaseWriteToDataset(t *testing.T) {
	stub := newKiwiStub(kiwistub.WithDataset(kiwistub.Dataset{
		Products:   []kiwistub.Product{{ID: 1, Name: "Example"}},
		Plans:      []kiwistub.Plan{{ID: 5, Name: "Plan", Product: 1}},
		Categories: []kiwistub.Category{{ID: 1, Name: "Functional", Product: 1}},
		Priorities: []kiwistub.Priority{{ID: 1, Value: "P1"}},
		Statuses:   []kiwistub.Status{{ID: 1, Name: "CONFIRMED"}},
	}))
	_, created := kiwiCall(t, stub, "TestCase.create", map[string]any{"values": map[string]any{
		"summary": "New case", "category": 1, "priority": 1, "case_status": 1,
	}})
	result, ok := created["result"].(map[string]any)
	if !ok {
		t.Fatalf("the create holds no result: %v", created)
	}
	caseID := result["id"]
	kiwiCall(t, stub, "TestCase.update", map[string]any{"case_id": caseID, "values": map[string]any{"summary": "Renamed"}})
	kiwiCall(t, stub, "TestPlan.add_case", map[string]any{"plan_id": 5, "case_id": caseID})
	_, members := kiwiCall(t, stub, "TestCase.filter", map[string]any{"query": map[string]any{"plan": 5}})
	rows := resultRows(t, members)
	if len(rows) != 1 {
		t.Fatalf("the plan holds %d cases, want the one added", len(rows))
	}
	row, _ := rows[0].(map[string]any)
	if row["summary"] != "Renamed" || row["id"] != caseID {
		t.Fatalf("the plan's case is %v, want the renamed case %v", row, caseID)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/kiwi-stub.feature: An update of a case with no case ID is refused
func TestUpdateOfCaseWithNoCaseIDIsRefused(t *testing.T) {
	stub := newKiwiStub()
	_, envelope := kiwiCall(t, stub, "TestCase.update", map[string]any{"values": map[string]any{"summary": "x"}})
	if _, ok := envelope["error"]; !ok {
		t.Fatalf("an update with no case ID was answered: %v", envelope)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/kiwi-stub.feature: A sort-key call with no plan is refused
func TestSortKeyCallWithNoPlanIsRefused(t *testing.T) {
	stub := newKiwiStub()
	_, envelope := kiwiCall(t, stub, "TestCase.sortkeys", map[string]any{"query": map[string]any{}})
	if _, ok := envelope["error"]; !ok {
		t.Fatalf("a sort-key call with no plan was answered: %v", envelope)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/kiwi-stub.feature: A fault set for every call answers with its failure until it is cleared
func TestFaultForEveryCallAnswersUntilCleared(t *testing.T) {
	stub := newKiwiStub()
	stub.SetFault(kiwistub.Fault{Status: http.StatusServiceUnavailable})
	params := map[string]any{"query": map[string]any{}}
	first, _ := kiwiCall(t, stub, "Product.filter", params)
	second, _ := kiwiCall(t, stub, "Product.filter", params)
	stub.ClearFault()
	third, _ := kiwiCall(t, stub, "Product.filter", params)
	if first != http.StatusServiceUnavailable || second != http.StatusServiceUnavailable || third != http.StatusOK {
		t.Fatalf("statuses %d, %d, %d; want 503, 503, 200", first, second, third)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/kiwi-stub.feature: A fault set for the next call answers that call only
func TestFaultForNextCallAnswersThatCallOnly(t *testing.T) {
	stub := newKiwiStub()
	stub.SetFaultFirst(kiwistub.Fault{Status: http.StatusInternalServerError})
	params := map[string]any{"query": map[string]any{}}
	first, _ := kiwiCall(t, stub, "Product.filter", params)
	second, _ := kiwiCall(t, stub, "Product.filter", params)
	if first != http.StatusInternalServerError || second != http.StatusOK {
		t.Fatalf("statuses %d and %d, want 500 then 200", first, second)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/kiwi-stub.feature: A method fault takes precedence over a persistent fault
func TestMethodFaultTakesPrecedenceOverPersistentFault(t *testing.T) {
	stub := newKiwiStub()
	stub.SetFault(kiwistub.Fault{Status: http.StatusInternalServerError})
	stub.SetMethodFault("Product.filter", kiwistub.Fault{Status: http.StatusTooManyRequests, RetryAfter: "1"})
	status, _ := kiwiCall(t, stub, "Product.filter", map[string]any{"query": map[string]any{}})
	if status != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", status)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/kiwi-stub.feature: A method error answers inside HTTP 200 with a JSON-RPC error object
func TestMethodErrorAnswersInsideHTTP200(t *testing.T) {
	stub := newKiwiStub()
	stub.SetMethodError("Product.filter", 1234, "EXAMPLE error")
	status, envelope := kiwiCall(t, stub, "Product.filter", map[string]any{"query": map[string]any{}})
	errObj, _ := envelope["error"].(map[string]any)
	if status != http.StatusOK || errObj["code"] != float64(1234) {
		t.Fatalf("status %d and error %v; want 200 with code 1234", status, envelope["error"])
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/kiwi-stub.feature: A text with HTML characters is escaped unless the stub is set to raw text
func TestTextWithHTMLCharactersIsEscaped(t *testing.T) {
	stub := newKiwiStub(kiwistub.WithDataset(kiwistub.Dataset{
		Cases: []kiwistub.Case{{ID: 1, Summary: "Login", Text: "<b>"}},
	}))
	_, envelope := kiwiCall(t, stub, "TestCase.filter", map[string]any{"query": map[string]any{"id": 1}})
	rows := resultRows(t, envelope)
	if len(rows) != 1 {
		t.Fatalf("the read returned %d cases", len(rows))
	}
	row, _ := rows[0].(map[string]any)
	if row["text"] != "&lt;b&gt;" {
		t.Fatalf("the text is %v, want &lt;b&gt;", row["text"])
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/kiwi-stub.feature: The request record holds each request and the peak in flight
func TestRequestRecordHoldsEachRequestAndPeakInFlight(t *testing.T) {
	stub := newKiwiStub()
	stub.SetFault(kiwistub.Fault{Never: true})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, _ = kiwiRequest(ctx, stub, "Bearer "+kiwiKey, "Product.filter", map[string]any{"query": map[string]any{}})
		}()
	}
	// Both requests are held in the stub until the context ends, so their overlap is seen without any wait on the clock.
	for i := 0; stub.PeakInFlight() < 2; i++ {
		if i > 1<<20 {
			t.Fatal("the two requests never overlapped")
		}
		runtime.Gosched()
	}
	cancel()
	wg.Wait()
	if n := len(stub.Requests()); n != 2 {
		t.Fatalf("the record holds %d requests, want 2", n)
	}
	if peak := stub.PeakInFlight(); peak != 2 {
		t.Fatalf("the peak in flight is %d, want 2", peak)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/kiwi-stub.feature: A request to a host with its own dataset is served from that dataset
func TestRequestToHostWithOwnDatasetIsServedFromThatDataset(t *testing.T) {
	stub := newKiwiStub()
	stub.SetHostDataset("kiwi.example.test", kiwistub.Dataset{
		Products: []kiwistub.Product{{ID: 9, Name: "Own product"}},
	})
	_, envelope := kiwiCall(t, stub, "Product.filter", map[string]any{"query": map[string]any{}})
	rows := resultRows(t, envelope)
	if len(rows) != 1 {
		t.Fatalf("the host dataset holds %d products, want 1", len(rows))
	}
	row, _ := rows[0].(map[string]any)
	if row["name"] != "Own product" {
		t.Fatalf("the product is %v, want the host's own", row["name"])
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/kiwi-stub.feature: A client made from the stub trusts the stub only
func TestClientMadeFromStubTrustsStubCertificateOnly(t *testing.T) {
	stub := newKiwiStub(
		kiwistub.WithListener(newFakeListener().listen),
		kiwistub.WithRandom(&fixedRandom{}),
	)
	if err := stub.Start(); err != nil {
		t.Fatalf("start the stub: %v", err)
	}
	defer stub.Close()
	client, err := stub.Client()
	if err != nil {
		t.Fatalf("make the client: %v", err)
	}
	transport, ok := client.Transport.(*http.Transport)
	if !ok || transport.TLSClientConfig == nil {
		t.Fatal("the client has no TLS transport")
	}
	want := x509.NewCertPool()
	if !want.AppendCertsFromPEM(stub.CAPEM()) {
		t.Fatal("the CA file holds no certificate")
	}
	if !transport.TLSClientConfig.RootCAs.Equal(want) {
		t.Fatal("the client trusts a certificate other than the stub's")
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/kiwi-stub.feature: The in-process round trip answers a request without a socket
func TestInProcessRoundTripAnswersRequestWithoutSocket(t *testing.T) {
	stub := newKiwiStub()
	status, envelope := kiwiCall(t, stub, "Product.filter", map[string]any{"query": map[string]any{}})
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	resultRows(t, envelope)
}
