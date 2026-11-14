package outlinestub_test

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-testkit/outlinestub"
)

const testAuthorization = "Bearer stub-test-key"

// serveRecorded sends one request to the stub's ServeHTTP, as the TLS listener does, and returns the recorder.
func serveRecorded(t *testing.T, s *outlinestub.Stub, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Authorization", testAuthorization)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	return rec
}

// outlineURL is the URL of one Outline method on the stub's endpoint host.
func outlineURL(method string) string {
	return "https://outline-stub.invalid/api/" + method
}

// hijackRecorder is a response writer that can be hijacked. Its connection is an in-memory pipe end.
type hijackRecorder struct {
	*httptest.ResponseRecorder
	conn     net.Conn
	err      error
	hijacked bool
}

func (h *hijackRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h.hijacked = true
	if h.err != nil {
		return nil, nil, h.err
	}
	return h.conn, bufio.NewReadWriter(bufio.NewReader(h.conn), bufio.NewWriter(h.conn)), nil
}

// failingBody is a request body that cannot be read.
type failingBody struct{}

var errBodyUnreadable = errors.New("the body cannot be read")

func (failingBody) Read([]byte) (int, error) { return 0, errBodyUnreadable }
func (failingBody) Close() error             { return nil }

func TestServeHTTPAnswersAMethodFromTheTLSPath(t *testing.T) {
	s := newStub()

	rec := serveRecorded(t, s, http.MethodPost, outlineURL("auth.info"), `{}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	reqs := s.Requests()
	if len(reqs) != 1 {
		t.Fatalf("recorded %d requests, want 1", len(reqs))
	}
	got := reqs[0]
	if got.Method != "auth.info" || got.HTTPMethod != http.MethodPost || !got.Secure || got.Status != http.StatusOK {
		t.Fatalf("recorded %+v, want auth.info over TLS answered 200", got)
	}
	if got.Authorization != testAuthorization || got.Host != "outline-stub.invalid" {
		t.Fatalf("recorded authorization %q and host %q", got.Authorization, got.Host)
	}
	if !got.At.Equal(testClock) {
		t.Fatalf("recorded at %v, want the test clock %v", got.At, testClock)
	}
}

func TestServeHTTPRefusesEveryPathOutsideTheAPIPrefix(t *testing.T) {
	s := newStub()

	rec := serveRecorded(t, s, http.MethodPost, "https://outline-stub.invalid/other/auth.info", `{}`)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Endpoint not found") {
		t.Fatalf("body %q lacks the router's catch-all message", rec.Body.String())
	}
	if reqs := s.Requests(); len(reqs) != 1 || reqs[0].Method != "" {
		t.Fatalf("recorded %+v, want one request with no method name", reqs)
	}
}

func TestServeHTTPRecordsTheBodyAndRefusesItWhenItIsBad(t *testing.T) {
	cases := []struct {
		name       string
		method     string
		body       string
		wantStatus int
		wantBody   map[string]any
	}{
		{
			name:       "a body that is not JSON",
			method:     "documents.info",
			body:       `{"id":`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "a body whose field has the wrong type",
			method:     "collections.list",
			body:       `{"offset":"first","limit":25}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   map[string]any{"offset": "first", "limit": float64(25)},
		},
		{
			name:       "a JSON value that is not an object",
			method:     "collections.list",
			body:       `[1,2]`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "a blank body on a method that needs a limit",
			method:     "collections.list",
			body:       "  \n ",
			wantStatus: http.StatusBadRequest,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newStub()

			rec := serveRecorded(t, s, http.MethodPost, outlineURL(tc.method), tc.body)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			reqs := s.Requests()
			if len(reqs) != 1 || reqs[0].Status != tc.wantStatus {
				t.Fatalf("recorded %+v, want one request answered %d", reqs, tc.wantStatus)
			}
			if tc.wantBody != nil {
				for k, want := range tc.wantBody {
					if got := reqs[0].Body[k]; got != want {
						t.Fatalf("recorded body[%q] = %v, want %v", k, got, want)
					}
				}
			}
		})
	}
}

func TestServeHTTPAnswersABodyThatCannotBeReadAsBadRequest(t *testing.T) {
	s := newStub()
	req := httptest.NewRequest(http.MethodPost, outlineURL("auth.info"), http.NoBody)
	req.Body = failingBody{}
	rec := httptest.NewRecorder()

	s.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for a body that cannot be read", rec.Code)
	}
	if reqs := s.Requests(); len(reqs) != 1 || reqs[0].Body != nil {
		t.Fatalf("recorded %+v, want one request with no body", reqs)
	}
}

func TestServeHTTPTreatsAMissingBodyAsAnEmptyRequest(t *testing.T) {
	s := newStub()
	req := httptest.NewRequest(http.MethodPost, outlineURL("auth.info"), nil)
	req.Body = nil
	rec := httptest.NewRecorder()

	s.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 for a request with no body", rec.Code)
	}
}

func TestServeHTTPMarksAPlainHTTPRequestAsNotSecure(t *testing.T) {
	s := newStub()
	req := httptest.NewRequest(http.MethodPost, "http://outline-stub.invalid/api/auth.info", strings.NewReader("{}"))
	req.TLS = nil
	rec := httptest.NewRecorder()

	s.ServeHTTP(rec, req)

	if reqs := s.Requests(); len(reqs) != 1 || reqs[0].Secure {
		t.Fatalf("recorded %+v, want one request that is not secure", reqs)
	}
}

func TestServeHTTPRecordsAVersionHeaderAndTheClientVersionAsSent(t *testing.T) {
	s := newStub()
	req := httptest.NewRequest(http.MethodPost, outlineURL("documents.info"), strings.NewReader(`{"id":"doc-1"}`))
	req.Header.Set("X-API-Version", "3")
	req.Header.Set("X-Client-Version", "retriever/0.0.0-test")
	rec := httptest.NewRecorder()

	s.ServeHTTP(rec, req)

	reqs := s.Requests()
	if len(reqs) != 1 {
		t.Fatalf("recorded %d requests, want 1", len(reqs))
	}
	if reqs[0].APIVersion != "3" || reqs[0].ClientVersion != "retriever/0.0.0-test" || reqs[0].ID != "doc-1" {
		t.Fatalf("recorded %+v, want the version headers and the id as sent", reqs[0])
	}
}

func TestServeHTTPHangsUntilItsRequestEnds(t *testing.T) {
	s := newStub()
	s.SetFault(outlinestub.Fault{Hang: true})
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodPost, outlineURL("auth.info"), strings.NewReader("{}")).WithContext(ctx)
	rec := httptest.NewRecorder()
	done := make(chan struct{})

	go func() {
		s.ServeHTTP(rec, req)
		close(done)
	}()
	cancel()

	<-done
	if rec.Body.Len() != 0 {
		t.Fatalf("a hung request wrote %q, want nothing", rec.Body.String())
	}
}

func TestServeHTTPHangsUntilTheStubCloses(t *testing.T) {
	s := newStub()
	s.SetFault(outlinestub.Fault{Hang: true})
	req := httptest.NewRequest(http.MethodPost, outlineURL("auth.info"), strings.NewReader("{}"))
	rec := httptest.NewRecorder()
	done := make(chan struct{})

	go func() {
		s.ServeHTTP(rec, req)
		close(done)
	}()
	s.Close()

	<-done
}

func TestServeHTTPDropsTheConnectionOfAHijackableWriter(t *testing.T) {
	s := newStub()
	s.SetFault(outlinestub.Fault{Drop: true})
	client, server := net.Pipe()
	defer client.Close()
	w := &hijackRecorder{ResponseRecorder: httptest.NewRecorder(), conn: server}

	s.ServeHTTP(w, httptest.NewRequest(http.MethodPost, outlineURL("auth.info"), strings.NewReader("{}")))

	if !w.hijacked {
		t.Fatal("a dropped request was not hijacked")
	}
	if w.Body.Len() != 0 {
		t.Fatalf("a dropped request wrote %q before the drop", w.Body.String())
	}
	if _, err := client.Read(make([]byte, 1)); !errors.Is(err, io.EOF) {
		t.Fatalf("the client read %v after the drop, want the connection closed", err)
	}
}

func TestServeHTTPDropsNothingWhenTheWriterCannotBeHijacked(t *testing.T) {
	s := newStub()
	s.SetFault(outlinestub.Fault{Drop: true})
	rec := httptest.NewRecorder()

	s.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, outlineURL("auth.info"), strings.NewReader("{}")))

	if rec.Body.Len() != 0 || rec.Flushed {
		t.Fatalf("a dropped request on a plain writer wrote %q", rec.Body.String())
	}
}

func TestServeHTTPWritesNothingWhenTheHijackFails(t *testing.T) {
	s := newStub()
	s.SetFault(outlinestub.Fault{Drop: true})
	w := &hijackRecorder{ResponseRecorder: httptest.NewRecorder(), err: errors.New("hijack refused")}

	s.ServeHTTP(w, httptest.NewRequest(http.MethodPost, outlineURL("auth.info"), strings.NewReader("{}")))

	if !w.hijacked {
		t.Fatal("the drop did not try to hijack the connection")
	}
	if w.Body.Len() != 0 {
		t.Fatalf("a drop whose hijack failed wrote %q", w.Body.String())
	}
}

func TestServeHTTPWritesTheFaultHeadersAndBody(t *testing.T) {
	s := newStub()
	s.SetFault(outlinestub.Fault{Status: http.StatusTooManyRequests, RetryAfter: "7"})

	rec := serveRecorded(t, s, http.MethodPost, outlineURL("collections.list"), `{"offset":0,"limit":25}`)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", rec.Code)
	}
	if got := rec.Header().Get("Retry-After"); got != "7" {
		t.Fatalf("Retry-After = %q, want 7", got)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("the 429 body is not JSON: %v", err)
	}
	if body["status"] != float64(http.StatusTooManyRequests) {
		t.Fatalf("the 429 body = %v, want its status", body)
	}
}
