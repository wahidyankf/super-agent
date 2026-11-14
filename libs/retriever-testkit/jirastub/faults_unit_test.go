package jirastub_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-testkit/jirastub"
)

func TestFaultRepliesCarryTheirShapes(t *testing.T) {
	const myself = "/rest/api/3/myself"
	const search = "/rest/api/3/search/jql"
	const jql = `{"jql":"project = EXAMPLE"}`
	tests := []struct {
		name       string
		fault      jirastub.Fault
		method     string
		path       string
		body       string
		wantStatus int
		wantHeader map[string]string
		wantBody   []string
		notBody    []string
		wantEmpty  bool
		notJSON    bool
	}{
		{
			name: "no status and no location is a 500 error naming the path", fault: jirastub.Fault{},
			method: http.MethodGet, path: myself, wantStatus: http.StatusInternalServerError,
			wantBody: []string{"errorMessages", "Scripted fault for " + myself + "."},
		},
		{
			name: "a location alone is a 302 redirect", fault: jirastub.Fault{Location: "https://login.example.test/"},
			method: http.MethodGet, path: myself, wantStatus: http.StatusFound,
			wantHeader: map[string]string{"Location": "https://login.example.test/"}, wantEmpty: true,
		},
		{
			name: "a 3xx status writes its location and no body", fault: jirastub.Fault{Status: 301, Location: "https://login.example.test/"},
			method: http.MethodGet, path: myself, wantStatus: http.StatusMovedPermanently,
			wantHeader: map[string]string{"Location": "https://login.example.test/"}, wantEmpty: true,
		},
		{
			name: "a 429 carries its Retry-After", fault: jirastub.Fault{Status: http.StatusTooManyRequests, RetryAfter: "7"},
			method: http.MethodGet, path: myself, wantStatus: http.StatusTooManyRequests,
			wantHeader: map[string]string{"Retry-After": "7"}, wantBody: []string{"errorMessages"},
		},
		{
			name:   "an error status echoes the token and the e-mail only when asked",
			fault:  jirastub.Fault{Status: http.StatusServiceUnavailable, EchoToken: true, EchoEmail: true},
			method: http.MethodGet, path: myself, wantStatus: http.StatusServiceUnavailable,
			wantBody: []string{testToken, testEmail},
		},
		{
			name:   "an error status without the echo switches shows neither secret",
			fault:  jirastub.Fault{Status: http.StatusServiceUnavailable},
			method: http.MethodGet, path: myself, wantStatus: http.StatusServiceUnavailable,
			wantBody: []string{"errorMessages"}, notBody: []string{testToken, testEmail},
		},
		{
			name: "a non-JSON reply echoes the e-mail when asked", fault: jirastub.Fault{NonJSON: true, EchoEmail: true},
			method: http.MethodGet, path: myself, wantStatus: http.StatusOK, notJSON: true,
			wantBody: []string{"Log in to continue", testEmail},
		},
		{
			name: "a repeated token carries the token the fault names", fault: jirastub.Fault{RepeatToken: true, PageToken: "custom"},
			method: http.MethodPost, path: search, body: jql, wantStatus: http.StatusOK,
			wantBody: []string{`"nextPageToken":"custom"`},
		},
		{
			name: "an empty page carries a token past the end", fault: jirastub.Fault{EmptyPageWithToken: true},
			method: http.MethodPost, path: search, body: jql, wantStatus: http.StatusOK,
			wantBody: []string{`"issues":[]`, `"nextPageToken":"p500"`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newStub(t)
			s.SetFault(tt.fault)
			rec := httptest.NewRecorder()
			s.ServeHTTP(rec, request(tt.method, tt.path, tt.body, basicHeader(testEmail, testToken)))
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			for key, want := range tt.wantHeader {
				if got := rec.Header().Get(key); got != want {
					t.Fatalf("header %s = %q, want %q", key, got, want)
				}
			}
			if tt.wantEmpty && rec.Body.Len() != 0 {
				t.Fatalf("a redirect wrote a body: %q", rec.Body.String())
			}
			for _, want := range tt.wantBody {
				if !strings.Contains(rec.Body.String(), want) {
					t.Fatalf("reply %q lacks %q", rec.Body.String(), want)
				}
			}
			for _, secret := range tt.notBody {
				if strings.Contains(rec.Body.String(), secret) {
					t.Fatalf("reply shows %q without its echo switch: %q", secret, rec.Body.String())
				}
			}
			if valid := json.Valid(rec.Body.Bytes()); !tt.wantEmpty && valid == tt.notJSON {
				t.Fatalf("reply JSON validity = %v, want a body that is JSON = %v: %q", valid, !tt.notJSON, rec.Body.String())
			}
		})
	}
}
