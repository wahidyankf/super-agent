package graphql

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	usecases "github.com/wahidyankf/super-agent/libs/retriever-runtime/sources"
)

const (
	testPort    = 8080
	allowedHost = "127.0.0.1:8080"
	exportQuery = `mutation { exportDocument(source: "outline", id: "doc-1", targetPath: "vault/sources/outline/getting-started.md") { path } }`
	exportPath  = "vault/sources/outline/getting-started.md"
)

// harness is a served handler over a counting gateway.
type harness struct {
	t  *testing.T
	gw *countingGateway
	h  http.Handler
}

// newHarness builds the handler on a fake listener at the test port, so no socket is bound.
func newHarness(t *testing.T, gw *countingGateway, files map[string][]byte) *harness {
	t.Helper()
	h, err := NewHandler(
		Deps{Service: oneOutlineSources(gw), Logger: discardLogger()},
		WithListener(tcpListener("127.0.0.1", testPort)),
	)
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	if h == nil {
		t.Fatal("NewHandler returned no handler")
	}
	return &harness{t: t, gw: gw, h: h}
}

// request describes one HTTP request to the handler.
type request struct {
	method      string
	target      string
	host        string
	contentType string
	origin      string
	body        string
}

// graphqlBody encodes query as the JSON request body.
func graphqlBody(query string) string {
	data, err := json.Marshal(map[string]string{"query": query})
	if err != nil {
		panic(err) // a string map always marshals; a failure here is a bug in the test itself
	}
	return string(data)
}

// graphqlPOST is a well-formed POST of query to the allowed host.
func graphqlPOST(query string) request {
	return request{method: http.MethodPost, host: allowedHost, contentType: "application/json", body: graphqlBody(query)}
}

// send runs one request through the handler and returns the recorder.
func (h *harness) send(req request) *httptest.ResponseRecorder {
	h.t.Helper()
	return sendTo(h.t, h.h, req)
}

// sendTo runs one request through handler and returns the recorder.
func sendTo(t *testing.T, handler http.Handler, req request) *httptest.ResponseRecorder {
	t.Helper()
	target := req.target
	if target == "" {
		target = "/graphql"
	}
	httpReq := httptest.NewRequest(req.method, target, strings.NewReader(req.body))
	httpReq.Host = req.host
	if req.contentType != "" {
		httpReq.Header.Set("Content-Type", req.contentType)
	}
	if req.origin != "" {
		httpReq.Header.Set("Origin", req.origin)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httpReq)
	return rec
}

// reply is the decoded GraphQL response body.
type reply struct {
	Data   map[string]any `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// decode parses the response body and checks that it carries no CORS header.
func decode(t *testing.T, rec *httptest.ResponseRecorder) reply {
	t.Helper()
	assertNoCORS(t, rec)
	var r reply
	if err := json.Unmarshal(rec.Body.Bytes(), &r); err != nil {
		t.Fatalf("response is not GraphQL JSON: %v; body %s", err, rec.Body.String())
	}
	return r
}

// messages returns the error messages of a reply.
func (r reply) messages() []string {
	out := make([]string, 0, len(r.Errors))
	for _, e := range r.Errors {
		out = append(out, e.Message)
	}
	return out
}

// assertErrorContains fails unless one error message of the reply contains fragment.
func assertErrorContains(t *testing.T, r reply, fragment string) {
	t.Helper()
	for _, msg := range r.messages() {
		if strings.Contains(msg, fragment) {
			return
		}
	}
	t.Fatalf("errors %v do not contain %q", r.messages(), fragment)
}

// assertNoUpstreamOrWrite fails when a gateway was called.
func (h *harness) assertNoUpstreamOrWrite() {
	h.t.Helper()
	if n := h.gw.calls.Load(); n != 0 {
		h.t.Fatalf("gateway was called %d times, want 0", n)
	}
}

func TestCollectionsQueryReturnsCollections(t *testing.T) {
	gw := &countingGateway{collections: []domain.Collection{{ID: "col-1", Name: "Engineering"}, {ID: "col-2", Name: "Operations"}}}
	h := newHarness(t, gw, nil)
	rec := h.send(graphqlPOST(`{ collections { id name } }`))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", rec.Code, rec.Body.String())
	}
	r := decode(t, rec)
	if len(r.Errors) != 0 {
		t.Fatalf("errors = %v, want none", r.messages())
	}
	list, ok := r.Data["collections"].([]any)
	if !ok || len(list) != 2 {
		t.Fatalf("data.collections = %v, want 2 items", r.Data["collections"])
	}
	if n := gw.calls.Load(); n != 1 {
		t.Fatalf("gateway calls = %d, want 1", n)
	}
}

func TestCollectionTreeIsFlatWithParentIDsAndDepths(t *testing.T) {
	parent := "doc-1"
	gw := &countingGateway{tree: []domain.TreeNode{
		{ID: "doc-1", Title: "Getting Started", URL: "/doc/getting-started", ParentID: nil, Depth: 0},
		{ID: "doc-2", Title: "Setup", URL: "/doc/setup", ParentID: &parent, Depth: 1},
	}}
	h := newHarness(t, gw, nil)
	rec := h.send(graphqlPOST(`{ collectionTree(id: "col-1") { id title url parentId depth } }`))
	r := decode(t, rec)
	if len(r.Errors) != 0 {
		t.Fatalf("errors = %v, want none", r.messages())
	}
	tree, ok := r.Data["collectionTree"].([]any)
	if !ok || len(tree) != 2 {
		t.Fatalf("data.collectionTree = %v, want 2 items", r.Data["collectionTree"])
	}
	first := tree[0].(map[string]any)
	second := tree[1].(map[string]any)
	if first["title"] != "Getting Started" || first["depth"] != float64(0) || first["parentId"] != nil {
		t.Errorf("root node = %v, want title Getting Started, depth 0, parentId null", first)
	}
	if second["title"] != "Setup" || second["depth"] != float64(1) || second["parentId"] != "doc-1" {
		t.Errorf("child node = %v, want title Setup, depth 1, parentId doc-1", second)
	}
}

func TestDocumentQueryReturnsMarkdown(t *testing.T) {
	gw := &countingGateway{doc: domain.SourceDocument{ID: "doc-1", Title: "Getting Started", URL: "/doc/getting-started", Markdown: "Welcome to the wiki.", CollectionID: "col-1"}}
	h := newHarness(t, gw, nil)
	r := decode(t, h.send(graphqlPOST(`{ document(id: "doc-1") { id title url markdown collectionId } }`)))
	if len(r.Errors) != 0 {
		t.Fatalf("errors = %v, want none", r.messages())
	}
	doc, ok := r.Data["document"].(map[string]any)
	if !ok || doc["title"] != "Getting Started" || doc["markdown"] != "Welcome to the wiki." || doc["collectionId"] != "col-1" {
		t.Fatalf("data.document = %v", r.Data["document"])
	}
}

func TestCollectionTreeHasNoChildrenField(t *testing.T) {
	gw := &countingGateway{}
	h := newHarness(t, gw, nil)
	r := decode(t, h.send(graphqlPOST(`{ collectionTree(id: "col-1") { id children { id } } }`)))
	assertErrorContains(t, r, "children")
	h.assertNoUpstreamOrWrite()
}

func TestUnsupportedSourceIsRefusedBeforeUpstream(t *testing.T) {
	gw := &countingGateway{}
	h := newHarness(t, gw, nil)
	r := decode(t, h.send(graphqlPOST(`{ collections(source: "slack") { id } }`)))
	assertErrorContains(t, r, "unknown source")
	h.assertNoUpstreamOrWrite()
}

func TestInvalidDocumentIDIsRefusedBeforeUpstream(t *testing.T) {
	gw := &countingGateway{}
	h := newHarness(t, gw, nil)
	r := decode(t, h.send(graphqlPOST(`{ document(id: "doc.1") { id } }`)))
	assertErrorContains(t, r, "invalid document ID")
	h.assertNoUpstreamOrWrite()
}

// TestUpstreamErrorTextIsNeverExposed checks that a typed upstream error reaches the caller as its fixed text only.
func TestUpstreamErrorTextIsNeverExposed(t *testing.T) {
	gw := &countingGateway{err: fmt.Errorf("status 401 body echoed %s: %w", rawSecret, domain.ErrKeyRejected)}
	h := newHarness(t, gw, nil)
	rec := h.send(graphqlPOST(`{ collections { id } }`))
	r := decode(t, rec)
	assertErrorContains(t, r, "Outline rejected the API key")
	if strings.Contains(rec.Body.String(), rawSecret) {
		t.Fatalf("response body exposes raw upstream text: %s", rec.Body.String())
	}
}

func TestUnknownErrorIsGeneric(t *testing.T) {
	gw := &countingGateway{err: errors.New("internal detail " + rawSecret)}
	h := newHarness(t, gw, nil)
	rec := h.send(graphqlPOST(`{ collections { id } }`))
	assertErrorContains(t, decode(t, rec), internalMessage)
	if strings.Contains(rec.Body.String(), rawSecret) {
		t.Fatalf("response body exposes an unknown error: %s", rec.Body.String())
	}
}

func TestIntrospectionIsRefusedBeforeUpstream(t *testing.T) {
	gw := &countingGateway{}
	h := newHarness(t, gw, nil)
	rec := h.send(graphqlPOST(`{ __schema { queryType { name } } }`))
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
	assertErrorContains(t, decode(t, rec), "introspection is disabled")
	h.assertNoUpstreamOrWrite()
}

func TestTypenameIsAllowed(t *testing.T) {
	h := newHarness(t, &countingGateway{}, nil)
	r := decode(t, h.send(graphqlPOST(`{ __typename }`)))
	if len(r.Errors) != 0 {
		t.Fatalf("errors = %v, want none", r.messages())
	}
}

func TestTwoOutlineFieldsWithinComplexityLimit(t *testing.T) {
	gw := &countingGateway{}
	h := newHarness(t, gw, nil)
	r := decode(t, h.send(graphqlPOST(`{ collections { id } collectionTree(id: "col-1") { id } }`)))
	if len(r.Errors) != 0 {
		t.Fatalf("errors = %v, want none", r.messages())
	}
	if n := gw.calls.Load(); n != 2 {
		t.Fatalf("gateway calls = %d, want 2", n)
	}
}

func TestThreeOutlineFieldsRefusedBeforeUpstream(t *testing.T) {
	gw := &countingGateway{}
	h := newHarness(t, gw, nil)
	r := decode(t, h.send(graphqlPOST(`{ collections { id } collectionTree(id: "col-1") { id } document(id: "doc-1") { id } }`)))
	assertErrorContains(t, r, "complexity limit")
	h.assertNoUpstreamOrWrite()
}

func TestQueryOver150FieldsIsRefused(t *testing.T) {
	gw := &countingGateway{}
	h := newHarness(t, gw, nil)
	query := "{" + strings.Repeat(" __typename", 150) + " }"
	assertErrorContains(t, decode(t, h.send(graphqlPOST(query))), "complexity limit")
	h.assertNoUpstreamOrWrite()
}

func TestQueryNestedPastDepthLimitIsRefused(t *testing.T) {
	gw := &countingGateway{}
	h := newHarness(t, gw, nil)
	query := `{ a { b { c { d { e { f { g { h { i { j { k { l } } } } } } } } } } } }`
	assertErrorContains(t, decode(t, h.send(graphqlPOST(query))), "depth limit")
	h.assertNoUpstreamOrWrite()
}

func TestOversizedBodyIsRefused(t *testing.T) {
	gw := &countingGateway{}
	h := newHarness(t, gw, nil)
	body := `{"query":"` + strings.Repeat("a", maxBodyBytes) + `"}`
	rec := h.send(request{method: http.MethodPost, host: allowedHost, contentType: "application/json", body: body})
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", rec.Code)
	}
	h.assertNoUpstreamOrWrite()
}

func withOrigin(r request, origin string) request {
	r.origin = origin
	return r
}

func withHost(r request, host string) request {
	r.host = host
	return r
}

func withBody(r request, query string) request {
	r.body = graphqlBody(query)
	return r
}

func withContentType(r request, contentType string) request {
	r.contentType = contentType
	return r
}

func TestNewHandlerRequiresAListener(t *testing.T) {
	cases := []struct {
		name string
		deps Deps
		opts []Option
	}{
		{name: "no listener", deps: Deps{}},
		{
			name: "a listener without a TCP address",
			deps: Deps{},
			opts: []Option{WithListener(&fakeListener{addr: &net.UnixAddr{Name: "sock", Net: "unix"}})},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NewHandler(tc.deps, tc.opts...); err == nil {
				t.Fatal("NewHandler error = nil, want an error")
			}
		})
	}
}

// TestRefusalsAreLoggedWithoutRequestContent checks that a refusal is logged with its status and no query text.
func TestRefusalsAreLoggedWithoutRequestContent(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	h, err := NewHandler(
		Deps{Service: usecases.Service{}, Logger: logger},
		WithListener(tcpListener("127.0.0.1", testPort)),
	)
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	const marker = "secret-query-marker"
	req := withOrigin(graphqlPOST(`{ collections { id } } # `+marker), "https://evil.example.com")
	httpReq := httptest.NewRequest(req.method, "/graphql", strings.NewReader(req.body))
	httpReq.Host = req.host
	httpReq.Header.Set("Content-Type", req.contentType)
	httpReq.Header.Set("Origin", req.origin)
	h.ServeHTTP(httptest.NewRecorder(), httpReq)
	out := logs.String()
	if !strings.Contains(out, "status=403") {
		t.Fatalf("log %q does not record the refusal status", out)
	}
	if strings.Contains(out, marker) {
		t.Fatalf("log %q holds request content", out)
	}
}
