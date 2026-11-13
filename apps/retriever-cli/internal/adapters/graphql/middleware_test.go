package graphql

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// probe is the next handler behind a guard. It records that it ran and the body it received.
type probe struct {
	reached bool
	body    string
}

func (p *probe) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.reached = true
	data, _ := io.ReadAll(r.Body)
	p.body = string(data)
	w.WriteHeader(http.StatusOK)
}

// runGuard sends req through mw and returns the recorder and the probe behind it.
func runGuard(mw middleware, req *http.Request) (*httptest.ResponseRecorder, *probe) {
	next := &probe{}
	rec := httptest.NewRecorder()
	mw(next).ServeHTTP(rec, req)
	return rec, next
}

// assertPassed checks that the guard let the request through to the next handler.
func assertPassed(t *testing.T, rec *httptest.ResponseRecorder, next *probe) {
	t.Helper()
	if !next.reached {
		t.Fatalf("next handler was not reached; status %d, body %s", rec.Code, rec.Body.String())
	}
	assertNoCORS(t, rec)
}

// assertRefused checks a refusal: the status, the standard GraphQL error JSON with the message fragment,
// no call to the next handler, and no CORS headers.
func assertRefused(t *testing.T, rec *httptest.ResponseRecorder, next *probe, status int, fragment string) {
	t.Helper()
	if next.reached {
		t.Fatalf("next handler was reached for a refused request")
	}
	if rec.Code != status {
		t.Fatalf("status = %d, want %d; body %s", rec.Code, status, rec.Body.String())
	}
	var body struct {
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("refusal body is not GraphQL error JSON: %v; body %s", err, rec.Body.String())
	}
	if len(body.Errors) != 1 || !strings.Contains(body.Errors[0].Message, fragment) {
		t.Fatalf("errors = %+v, want one message containing %q", body.Errors, fragment)
	}
	assertNoCORS(t, rec)
}

// assertNoCORS fails when any response header is a CORS header.
func assertNoCORS(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	for name := range rec.Header() {
		if strings.HasPrefix(strings.ToLower(name), "access-control-") {
			t.Fatalf("response carries CORS header %q", name)
		}
	}
}

func TestRequireLoopbackHost(t *testing.T) {
	mw := requireLoopbackHost(8080)
	cases := []struct {
		host   string
		passed bool
	}{
		{"127.0.0.1:8080", true},
		{"localhost:8080", true},
		{"evil.example.com:8080", false},
		{"127.0.0.1.evil.example.com:8080", false},
		{"localhost", false},
		{"127.0.0.1", false},
		{"127.0.0.1:9090", false},
		{"localhost:9090", false},
		{"[::1]:8080", false},
		{"LOCALHOST:8080", false},
		{"", false},
	}
	for _, tc := range cases {
		t.Run("host "+tc.host, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/graphql", nil)
			req.Host = tc.host
			rec, next := runGuard(mw, req)
			if tc.passed {
				assertPassed(t, rec, next)
				return
			}
			assertRefused(t, rec, next, http.StatusForbidden, "host is not allowed")
		})
	}
}

func TestRejectBrowserOrigin(t *testing.T) {
	mw := rejectBrowserOrigin()
	cases := []struct {
		name   string
		origin string
		set    bool
		passed bool
	}{
		{name: "no origin header", passed: true},
		{name: "a browser origin", origin: "https://evil.example.com", set: true},
		{name: "the null origin", origin: "null", set: true},
		{name: "an empty origin header", origin: "", set: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/graphql", nil)
			if tc.set {
				req.Header.Set("Origin", tc.origin)
			}
			rec, next := runGuard(mw, req)
			if tc.passed {
				assertPassed(t, rec, next)
				return
			}
			assertRefused(t, rec, next, http.StatusForbidden, "browser origins are not allowed")
		})
	}
}

func TestRequirePOST(t *testing.T) {
	mw := requirePOST()
	t.Run("POST passes", func(t *testing.T) {
		rec, next := runGuard(mw, httptest.NewRequest(http.MethodPost, "/graphql", nil))
		assertPassed(t, rec, next)
	})
	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodOptions} {
		t.Run(method+" is refused", func(t *testing.T) {
			rec, next := runGuard(mw, httptest.NewRequest(method, "/graphql", nil))
			assertRefused(t, rec, next, http.StatusMethodNotAllowed, "only POST is allowed")
			if got := rec.Header().Get("Allow"); got != http.MethodPost {
				t.Fatalf("Allow = %q, want POST", got)
			}
		})
	}
}

func TestRequireJSON(t *testing.T) {
	mw := requireJSON()
	cases := []struct {
		name        string
		contentType string
		set         bool
		passed      bool
	}{
		{name: "application/json", contentType: "application/json", set: true, passed: true},
		{name: "application/json with charset", contentType: "application/json; charset=utf-8", set: true, passed: true},
		{name: "text/plain", contentType: "text/plain", set: true},
		{name: "form urlencoded", contentType: "application/x-www-form-urlencoded", set: true},
		{name: "graphql media type", contentType: "application/graphql", set: true},
		{name: "missing content type"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/graphql", nil)
			if tc.set {
				req.Header.Set("Content-Type", tc.contentType)
			}
			rec, next := runGuard(mw, req)
			if tc.passed {
				assertPassed(t, rec, next)
				return
			}
			assertRefused(t, rec, next, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
		})
	}
}

func TestLimitBody(t *testing.T) {
	const limit = 16
	mw := limitBody(limit)
	t.Run("a body of exactly the limit passes unchanged", func(t *testing.T) {
		body := strings.Repeat("a", limit)
		rec, next := runGuard(mw, httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(body)))
		assertPassed(t, rec, next)
		if next.body != body {
			t.Fatalf("next body = %q, want the original body", next.body)
		}
	})
	t.Run("a body one byte over the limit is refused", func(t *testing.T) {
		rec, next := runGuard(mw, httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(strings.Repeat("a", limit+1))))
		assertRefused(t, rec, next, http.StatusRequestEntityTooLarge, "request body exceeds")
	})
}

func TestLimitQuery(t *testing.T) {
	mw := limitQuery()
	bodyOf := func(query string) string {
		data, err := json.Marshal(map[string]string{"query": query})
		if err != nil {
			t.Fatalf("marshal query: %v", err)
		}
		return string(data)
	}
	t.Run("a query within the limits passes with its body intact", func(t *testing.T) {
		body := bodyOf(`{ collections { id } }`)
		req := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewReader([]byte(body)))
		rec, next := runGuard(mw, req)
		assertPassed(t, rec, next)
		if next.body != body {
			t.Fatalf("next body = %q, want %q", next.body, body)
		}
	})
	t.Run("a query nested past the depth limit is refused", func(t *testing.T) {
		rec, next := runGuard(mw, httptest.NewRequest(http.MethodPost, "/graphql",
			strings.NewReader(bodyOf(`{ a { b { c { d { e { f { g { h { i { j { k { l } } } } } } } } } } } }`))))
		assertRefused(t, rec, next, http.StatusUnprocessableEntity, "depth limit")
	})
	t.Run("a query over the complexity limit is refused", func(t *testing.T) {
		rec, next := runGuard(mw, httptest.NewRequest(http.MethodPost, "/graphql",
			strings.NewReader(bodyOf(`{ collections { id } collectionTree(id: "c") { id } document(id: "d") { id } }`))))
		assertRefused(t, rec, next, http.StatusUnprocessableEntity, "complexity limit")
	})
	t.Run("introspection is refused", func(t *testing.T) {
		rec, next := runGuard(mw, httptest.NewRequest(http.MethodPost, "/graphql",
			strings.NewReader(bodyOf(`{ __schema { types { name } } }`))))
		assertRefused(t, rec, next, http.StatusUnprocessableEntity, "introspection is disabled")
	})
	t.Run("a body that is not JSON is refused", func(t *testing.T) {
		rec, next := runGuard(mw, httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader("{not json")))
		assertRefused(t, rec, next, http.StatusBadRequest, "invalid GraphQL request body")
	})
	t.Run("a JSON array (batch) is refused", func(t *testing.T) {
		rec, next := runGuard(mw, httptest.NewRequest(http.MethodPost, "/graphql",
			strings.NewReader(`[{"query":"{ __typename }"}]`)))
		assertRefused(t, rec, next, http.StatusBadRequest, "invalid GraphQL request body")
	})
	t.Run("a syntax error passes so the executor reports it", func(t *testing.T) {
		rec, next := runGuard(mw, httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(bodyOf(`{ collections {`))))
		assertPassed(t, rec, next)
	})
}
