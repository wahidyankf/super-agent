package jirastub_test

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

	"github.com/wahidyankf/super-agent/libs/retriever-testkit/jirastub"
)

// hijackWriter is a reply writer that can take over its connection, as the writer of an HTTP/1 server can. Its Hijack
// returns conn, or err when err is set.
type hijackWriter struct {
	*httptest.ResponseRecorder
	conn net.Conn
	err  error
}

func (h *hijackWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h.err != nil {
		return nil, nil, h.err
	}
	return h.conn, nil, nil
}

// errBody is a request body that fails when it is read.
type errBody struct{}

func (errBody) Read([]byte) (int, error) { return 0, errors.New("the client reset the connection") }
func (errBody) Close() error             { return nil }

// request builds a request to the synthetic host, with an optional Authorization header.
func request(method, path, body, auth string) *http.Request {
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, "https://jira.example.test"+path, reader)
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	return req
}

func TestRecordsKeepTheTransportFactsAndNeverTheHeaderValues(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		auth           string
		cookie         string
		wantAuthOK     bool
		wantBearer     bool
		wantCookie     bool
		wantSecure     bool
		wantServerName string
	}{
		{
			name: "a signed https request is secure and names its TLS server", url: "https://jira.example.test/rest/api/3/myself",
			auth: basicHeader(testEmail, testToken), wantAuthOK: true, wantSecure: true, wantServerName: "jira.example.test",
		},
		{
			name: "a plain http request is not secure and has no TLS name", url: "http://jira.example.test/rest/api/3/myself",
			auth: basicHeader(testEmail, testToken), wantAuthOK: true,
		},
		{
			name: "a bearer token and a cookie are flagged, not kept", url: "https://jira.example.test/rest/api/3/myself",
			auth: "Bearer " + testToken, cookie: "session=cookie-secret", wantBearer: true, wantCookie: true, wantSecure: true,
			wantServerName: "jira.example.test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newStub(t)
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			if tt.auth != "" {
				req.Header.Set("Authorization", tt.auth)
			}
			if tt.cookie != "" {
				req.Header.Set("Cookie", tt.cookie)
			}
			s.ServeHTTP(httptest.NewRecorder(), req)

			got := s.Requests()
			if len(got) != 1 {
				t.Fatalf("recorded %d requests, want 1", len(got))
			}
			r := got[0]
			if r.AuthOK != tt.wantAuthOK || r.Bearer != tt.wantBearer || r.Cookie != tt.wantCookie {
				t.Fatalf("flags AuthOK=%v Bearer=%v Cookie=%v, want %v %v %v", r.AuthOK, r.Bearer, r.Cookie, tt.wantAuthOK, tt.wantBearer, tt.wantCookie)
			}
			if r.Secure != tt.wantSecure || r.ServerName != tt.wantServerName {
				t.Fatalf("secure=%v server name=%q, want %v %q", r.Secure, r.ServerName, tt.wantSecure, tt.wantServerName)
			}
			if r.Host != "jira.example.test" || !r.At.Equal(fixedTime) {
				t.Fatalf("host %q at %v, want jira.example.test at the injected clock", r.Host, r.At)
			}
			data, err := json.Marshal(got)
			if err != nil {
				t.Fatalf("marshal records: %v", err)
			}
			for _, secret := range []string{testToken, "cookie-secret"} {
				if strings.Contains(string(data), secret) {
					t.Fatalf("a record holds %q: %s", secret, data)
				}
			}
		})
	}
}

func TestEachRequestCountsItselfInFlightWhileItWaits(t *testing.T) {
	s := newStub(t)
	arrived := make(chan jirastub.Request, 4)
	s.SetRecorder(func(r jirastub.Request) { arrived <- r })
	s.SetFault(jirastub.Fault{Path: "/rest/api/3/issue/", Hang: true})

	hung := make(chan struct{})
	go func() {
		defer close(hung)
		s.ServeHTTP(httptest.NewRecorder(), request(http.MethodGet, "/rest/api/3/issue/"+jirastub.IssueKey, "", basicHeader(testEmail, testToken)))
	}()
	first := <-arrived // the hanging request is recorded before it waits

	call(t, s, http.MethodGet, "/rest/api/3/myself", "", basicHeader(testEmail, testToken))
	second := <-arrived
	if first.InFlight != 1 || second.InFlight != 2 {
		t.Fatalf("in flight = %d then %d, want 1 then 2 (the hanging request still counts)", first.InFlight, second.InFlight)
	}

	s.Close()
	<-hung
}

func TestAHangFaultOverASocketReturnsWhenTheCallerGivesUpOrTheStubCloses(t *testing.T) {
	tests := []struct {
		name  string
		setup func(s *jirastub.Stub) *http.Request
	}{
		{
			name: "the caller has already given up",
			setup: func(*jirastub.Stub) *http.Request {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return request(http.MethodGet, "/rest/api/3/myself", "", basicHeader(testEmail, testToken)).WithContext(ctx)
			},
		},
		{
			name: "the stub is closed",
			setup: func(s *jirastub.Stub) *http.Request {
				s.Close()
				return request(http.MethodGet, "/rest/api/3/myself", "", basicHeader(testEmail, testToken))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newStub(t)
			s.SetFault(jirastub.Fault{Hang: true})
			req := tt.setup(s)
			rec := httptest.NewRecorder()
			s.ServeHTTP(rec, req)
			if rec.Body.Len() != 0 {
				t.Fatalf("a hanging request wrote a reply: %q", rec.Body.String())
			}
			if got := s.Requests(); len(got) != 1 || got[0].Path != "/rest/api/3/myself" {
				t.Fatalf("recorded %+v, want the one hanging request", got)
			}
			s.Close()
		})
	}
}

func TestADropFaultClosesAHijackingConnectionWithNoReply(t *testing.T) {
	tests := []struct {
		name       string
		hijackErr  error
		wantClosed bool
	}{
		{name: "a hijacking writer loses its connection", wantClosed: true},
		{name: "a writer that refuses the hijack keeps no connection", hijackErr: errors.New("hijack refused")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newStub(t)
			s.SetFault(jirastub.Fault{Drop: true})
			peer, conn := net.Pipe()
			defer peer.Close()
			w := &hijackWriter{ResponseRecorder: httptest.NewRecorder(), conn: conn, err: tt.hijackErr}
			s.ServeHTTP(w, request(http.MethodGet, "/rest/api/3/myself", "", basicHeader(testEmail, testToken)))

			if w.Body.Len() != 0 {
				t.Fatalf("a dropped request wrote %q", w.Body.String())
			}
			if tt.wantClosed {
				buf := make([]byte, 1)
				if n, err := peer.Read(buf); n != 0 || !errors.Is(err, io.EOF) {
					t.Fatalf("peer read = %d, %v; want the connection closed with no bytes", n, err)
				}
			}
			if got := s.Requests(); len(got) != 1 || got[0].Status != 0 {
				t.Fatalf("recorded %+v, want one dropped request with no status", got)
			}
		})
	}
}

func TestADropFaultOverAWriterThatCannotHijackRecordsTheRequestOnly(t *testing.T) {
	s := newStub(t)
	s.SetFault(jirastub.Fault{Drop: true})
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, request(http.MethodGet, "/rest/api/3/myself", "", basicHeader(testEmail, testToken)))
	if rec.Body.Len() != 0 || rec.Code != http.StatusOK {
		t.Fatalf("a drop with no hijack wrote %d %q", rec.Code, rec.Body.String())
	}
	if got := s.Requests(); len(got) != 1 || got[0].Status != 0 {
		t.Fatalf("recorded %+v, want one dropped request", got)
	}
}

func TestTheRecorderHookSeesEachRecordAsItIsKept(t *testing.T) {
	s := newStub(t)
	var seen []jirastub.Request
	s.SetRecorder(func(r jirastub.Request) { seen = append(seen, r) })
	call(t, s, http.MethodGet, "/rest/api/3/myself", "", basicHeader(testEmail, testToken))
	call(t, s, http.MethodGet, "/rest/api/3/myself", "", "")
	kept := s.Requests()
	if len(seen) != 2 || len(kept) != 2 || seen[1].Status != http.StatusUnauthorized || kept[0].Status != http.StatusOK {
		t.Fatalf("hook saw %+v, record holds %+v", seen, kept)
	}
	s.SetRecorder(nil)
	call(t, s, http.MethodGet, "/rest/api/3/myself", "", basicHeader(testEmail, testToken))
	if len(seen) != 2 || len(s.Requests()) != 3 {
		t.Fatal("a cleared recorder hook still received a record")
	}
}

func TestSetFaultFirstAnswersOnlyTheNextMatchingReplies(t *testing.T) {
	s := newStub(t)
	auth := basicHeader(testEmail, testToken)
	s.SetFaultFirst(2, jirastub.Fault{Status: http.StatusInternalServerError})
	var got []int
	for i := 0; i < 3; i++ {
		status, _ := call(t, s, http.MethodGet, "/rest/api/3/myself", "", auth)
		got = append(got, status)
	}
	if got[0] != 500 || got[1] != 500 || got[2] != 200 {
		t.Fatalf("statuses = %v, want [500 500 200]", got)
	}
}

func TestCookieIsSentOnEveryReplyUntilItIsCleared(t *testing.T) {
	s := newStub(t)
	s.SetCookie("JSESSIONID")
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, request(http.MethodGet, "/rest/api/3/myself", "", ""))
	if rec.Code != http.StatusUnauthorized || rec.Header().Get("Set-Cookie") != "JSESSIONID=synthetic-session; Path=/" {
		t.Fatalf("refused reply = %d, Set-Cookie %q", rec.Code, rec.Header().Get("Set-Cookie"))
	}
	s.SetCookie("")
	rec = httptest.NewRecorder()
	s.ServeHTTP(rec, request(http.MethodGet, "/rest/api/3/myself", "", basicHeader(testEmail, testToken)))
	if rec.Header().Get("Set-Cookie") != "" {
		t.Fatalf("a cleared cookie is still sent: %q", rec.Header().Get("Set-Cookie"))
	}
}

func TestJQLSearchBoundsThePageSize(t *testing.T) {
	tests := []struct {
		name       string
		size       string
		wantRows   int
		wantCursor string
	}{
		{name: "a zero size is one row", size: "0", wantRows: 1, wantCursor: "p1"},
		{name: "a size above the cap is capped at 100", size: "500", wantRows: 100, wantCursor: "p100"},
		{name: "a size that is not a number takes the default of 50", size: `"many"`, wantRows: 50, wantCursor: "p50"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newStub(t)
			s.SetIssues(jirastub.NumberedIssues(jirastub.ProjectKey, 120))
			_, body := call(t, s, http.MethodPost, "/rest/api/3/search/jql", `{"jql":"project = EXAMPLE","maxResults":`+tt.size+`}`, basicHeader(testEmail, testToken))
			page := decode(t, body)
			if rows := len(page["issues"].([]any)); rows != tt.wantRows || page["nextPageToken"] != tt.wantCursor {
				t.Fatalf("rows = %d, cursor = %v; want %d and %s", rows, page["nextPageToken"], tt.wantRows, tt.wantCursor)
			}
		})
	}
}

func TestJQLSearchReadsTheOffsetTokenAndDefaultsABadOne(t *testing.T) {
	tests := []struct {
		name    string
		token   string
		wantKey string
	}{
		{name: "a token names the offset", token: `"p2"`, wantKey: "EXAMPLE-3"},
		{name: "a token that is not a string starts at zero", token: `7`, wantKey: "EXAMPLE-1"},
		{name: "a token without the p prefix starts at zero", token: `"x2"`, wantKey: "EXAMPLE-1"},
		{name: "a negative offset starts at zero", token: `"p-4"`, wantKey: "EXAMPLE-1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newStub(t)
			s.SetIssues(jirastub.NumberedIssues(jirastub.ProjectKey, 3))
			_, body := call(t, s, http.MethodPost, "/rest/api/3/search/jql", `{"jql":"project = EXAMPLE","nextPageToken":`+tt.token+`}`, basicHeader(testEmail, testToken))
			page := decode(t, body)
			rows := page["issues"].([]any)
			if len(rows) == 0 || rows[0].(map[string]any)["key"] != tt.wantKey {
				t.Fatalf("first row = %v, want %s", rows, tt.wantKey)
			}
		})
	}
}

func TestJQLSearchSelectsTheProjectOfABareOrQuotedClause(t *testing.T) {
	tests := []struct {
		name    string
		jql     string
		wantKey string
		wantAll bool
	}{
		{name: "a bare key selects that project", jql: `project = OPS`, wantKey: jirastub.OtherProject},
		{name: "a quoted key selects that project", jql: `project="OPS"`, wantKey: jirastub.OtherProject},
		{name: "no project clause leaves every project", jql: `labels = login`, wantAll: true},
		{name: "a project word with no equals sign after it selects every project", jql: `project OPS AND id = 1`, wantAll: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newStub(t)
			body := `{"jql":` + mustJSON(t, tt.jql) + `}`
			_, reply := call(t, s, http.MethodPost, "/rest/api/3/search/jql", body, basicHeader(testEmail, testToken))
			page := decode(t, reply)
			rows := page["issues"].([]any)
			projects := map[string]bool{}
			for _, r := range rows {
				projects[strings.SplitN(r.(map[string]any)["key"].(string), "-", 2)[0]] = true
			}
			if tt.wantAll {
				if !projects[jirastub.ProjectKey] || !projects[jirastub.OtherProject] {
					t.Fatalf("rows span %v, want both projects", projects)
				}
				return
			}
			if len(projects) != 1 || !projects[tt.wantKey] {
				t.Fatalf("rows span %v, want only %s", projects, tt.wantKey)
			}
		})
	}
}

func TestJQLSearchReadsOnlyAJSONObjectBodyWithinItsCap(t *testing.T) {
	oversize := `{"jql":"project = EXAMPLE","pad":"` + strings.Repeat("a", 2<<20) + `"}`
	tests := []struct {
		name     string
		body     func(req *http.Request)
		wantBody bool
		wantCode int
	}{
		{name: "a JSON object is read", body: func(*http.Request) {}, wantBody: true, wantCode: http.StatusOK},
		{name: "a request with no body is refused", body: func(req *http.Request) { req.Body = nil }, wantCode: http.StatusBadRequest},
		{name: "an empty body is refused", body: func(req *http.Request) { req.Body = http.NoBody }, wantCode: http.StatusBadRequest},
		{name: "a JSON array is not an object", body: func(req *http.Request) {
			req.Body = io.NopCloser(strings.NewReader(`[{"jql":"project = EXAMPLE"}]`))
		}, wantCode: http.StatusBadRequest},
		{name: "invalid JSON is refused", body: func(req *http.Request) {
			req.Body = io.NopCloser(strings.NewReader(`{"jql":`))
		}, wantCode: http.StatusBadRequest},
		{name: "a body that fails while it is read is refused", body: func(req *http.Request) {
			req.Body = errBody{}
		}, wantCode: http.StatusBadRequest},
		{name: "a body past the cap is cut and refused", body: func(req *http.Request) {
			req.Body = io.NopCloser(strings.NewReader(oversize))
		}, wantCode: http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newStub(t)
			req := request(http.MethodPost, "/rest/api/3/search/jql", `{"jql":"project = EXAMPLE"}`, basicHeader(testEmail, testToken))
			tt.body(req)
			rec := httptest.NewRecorder()
			s.ServeHTTP(rec, req)
			if rec.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantCode)
			}
			got := s.Requests()[0].Body
			if tt.wantBody != (got != nil) {
				t.Fatalf("recorded body = %v, want a body: %v", got, tt.wantBody)
			}
		})
	}
}

func TestCommentsPageFromTheirStartAndFallBackOnBadQueryValues(t *testing.T) {
	s := newStub(t)
	auth := basicHeader(testEmail, testToken)
	path := "/rest/api/3/issue/" + jirastub.IssueKey + "/comment"

	_, body := call(t, s, http.MethodGet, path+"?startAt=5&maxResults=1", "", auth)
	past := decode(t, body)
	if got := past["comments"].([]any); len(got) != 0 || past["total"] != float64(2) {
		t.Fatalf("a start past the end = %v, want no comments of 2", past)
	}

	_, body = call(t, s, http.MethodGet, path+"?startAt=-3&maxResults=abc", "", auth)
	bad := decode(t, body)
	if got := bad["comments"].([]any); len(got) != 2 || bad["startAt"] != float64(0) || bad["maxResults"] != float64(50) {
		t.Fatalf("bad query values = %v, want start 0, size 50, both comments", bad)
	}
}

func TestACommentWhoseBodyIsNotJSONFailsTheReplyWith500(t *testing.T) {
	s := newStub(t)
	s.SetComments(map[string][]jirastub.Comment{
		jirastub.IssueKey: {{ID: "30001", Author: "Fixture Author", Body: json.RawMessage(`not json`)}},
	})
	status, body := call(t, s, http.MethodGet, "/rest/api/3/issue/"+jirastub.IssueKey+"/comment", "", basicHeader(testEmail, testToken))
	if status != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", status)
	}
	if _, ok := decode(t, body)["errorMessages"]; !ok {
		t.Fatalf("the failed encode carries no error body: %s", body)
	}
}

func TestAnIssueSendsEachFieldItHolds(t *testing.T) {
	s := newStub(t)
	auth := basicHeader(testEmail, testToken)
	s.SetIssues([]jirastub.Issue{
		{ID: "40001", Key: "EXAMPLE-40", Summary: "Fields", Type: "Task", Status: "Done", Priority: "Low", Resolution: "Done",
			Assignee: "Fixture Reviewer", Reporter: "Fixture Author", Parent: jirastub.ParentKey, Due: "2026-12-24",
			Created: "2026-11-01T09:00:00.000+0700", Updated: "2026-11-02T09:00:00.000+0700", Labels: []string{"ops"},
			Description: json.RawMessage(`{"type":"doc","version":1,"content":[]}`)},
		{ID: "40002", Key: "EXAMPLE-41", Summary: "Bare", Type: "Task", Status: "To Do", Priority: "Low", Labels: []string{}},
	})

	_, body := call(t, s, http.MethodGet, "/rest/api/3/issue/EXAMPLE-40", "", auth)
	fields := decode(t, body)["fields"].(map[string]any)
	if fields["resolution"].(map[string]any)["displayName"] != "Done" || fields["assignee"].(map[string]any)["displayName"] != "Fixture Reviewer" {
		t.Fatalf("resolution or assignee = %v", fields)
	}
	if fields["duedate"] != "2026-12-24" || fields["parent"].(map[string]any)["key"] != jirastub.ParentKey {
		t.Fatalf("duedate or parent = %v", fields)
	}
	if fields["description"] == nil || fields["project"].(map[string]any)["key"] != jirastub.ProjectKey {
		t.Fatalf("description or project = %v", fields)
	}

	_, body = call(t, s, http.MethodGet, "/rest/api/3/issue/EXAMPLE-41", "", auth)
	bare := decode(t, body)["fields"].(map[string]any)
	if bare["description"] != nil || bare["resolution"] != nil || bare["duedate"] != nil || bare["parent"] != nil {
		t.Fatalf("an issue with no description, resolution, due date, or parent must send nulls: %v", bare)
	}
}

func TestAnIssueWhoseDescriptionIsNotJSONIsSentAsItIs(t *testing.T) {
	s := newStub(t)
	s.SetIssues([]jirastub.Issue{
		{ID: "40003", Key: "EXAMPLE-42", Summary: "Raw", Type: "Task", Status: "To Do", Description: json.RawMessage(`plain text`)},
	})
	status, body := call(t, s, http.MethodGet, "/rest/api/3/issue/EXAMPLE-42", "", basicHeader(testEmail, testToken))
	if status != http.StatusOK || json.Valid(body) || !strings.Contains(string(body), "plain text") {
		t.Fatalf("status %d, body %q; want the raw description sent as it is", status, body)
	}
}

func TestSettersAndGettersCopyTheirLists(t *testing.T) {
	s := newStub(t)

	projects := jirastub.NumberedProjects(1)
	s.SetProjects(projects)
	projects[0].Key = "CHANGED"
	got := s.Projects()
	if got[0].Key != "PRJ1" {
		t.Fatalf("a changed input leaked into the stub: %q", got[0].Key)
	}
	got[0].Key = "MUTATED"
	if s.Projects()[0].Key != "PRJ1" {
		t.Fatal("a changed output leaked back into the stub")
	}

	issues := jirastub.NumberedIssues(jirastub.ProjectKey, 1)
	s.SetIssues(issues)
	issues[0].Summary = "CHANGED"
	if s.Issues()[0].Summary != "Numbered issue 1" {
		t.Fatal("a changed issue input leaked into the stub")
	}

	comments := map[string][]jirastub.Comment{jirastub.IssueKey: {{ID: "50001", Author: "Fixture Author"}}}
	s.SetComments(comments)
	comments[jirastub.IssueKey][0].Author = "CHANGED"
	comments[jirastub.OtherKey] = nil
	if c := s.Comments(); c[jirastub.IssueKey][0].Author != "Fixture Author" || len(c) != 1 {
		t.Fatalf("a changed comment input leaked into the stub: %v", c)
	}
	s.Comments()[jirastub.IssueKey][0].Author = "MUTATED"
	if s.Comments()[jirastub.IssueKey][0].Author != "Fixture Author" {
		t.Fatal("a changed comment output leaked back into the stub")
	}
}

func TestAccountAndCommentsAreAnsweredFromTheirSetters(t *testing.T) {
	s := newStub(t)
	auth := basicHeader(testEmail, testToken)
	s.SetAccount(jirastub.Account{AccountID: "acc-other", DisplayName: "Other Reviewer", AccountType: "atlassian", Active: false})
	s.SetComments(map[string][]jirastub.Comment{
		jirastub.UnassignedKey: {{ID: "50002", Author: "Fixture Reviewer", Body: json.RawMessage(`null`)}},
	})

	_, body := call(t, s, http.MethodGet, "/rest/api/3/myself", "", auth)
	me := decode(t, body)
	if me["displayName"] != "Other Reviewer" || me["active"] != false || me["emailAddress"] != testEmail {
		t.Fatalf("myself = %v, want the set account with the stub's e-mail", me)
	}
	_, body = call(t, s, http.MethodGet, "/rest/api/3/issue/"+jirastub.UnassignedKey+"/comment", "", auth)
	if list := decode(t, body)["comments"].([]any); len(list) != 1 {
		t.Fatalf("comments of the set issue = %v, want one", list)
	}
}

// mustJSON returns s as a JSON string literal.
func mustJSON(t *testing.T, s string) string {
	t.Helper()
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal %q: %v", s, err)
	}
	return string(data)
}
