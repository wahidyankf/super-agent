// Package jirastub is a scripted stand-in for Jira Cloud. It answers the five REST paths that Retriever reads, from the
// synthetic dataset in fixtures.go, and it applies scripted faults. It records every request, so a test can assert the
// calls the client made.
//
// The stub checks the HTTP Basic header against one synthetic e-mail address and one synthetic token. It records only
// whether the header matched (AuthOK), never the header value, so a test can prove the secret never reaches a record.
//
// Nothing here calls the real Atlassian site. The stub is served over TLS on a loopback port (see NewTLSServer).
package jirastub

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Paths the stub answers. Each one is a constant of the Jira client's allow-list.
const (
	myselfPath        = "/rest/api/3/myself"
	projectSearchPath = "/rest/api/3/project/search"
	jqlSearchPath     = "/rest/api/3/search/jql"
	removedSearchPath = "/rest/api/3/search"
	issuePrefix       = "/rest/api/3/issue/"
)

// maxRequestBody caps what the stub reads from a request, so a broken test cannot exhaust memory.
const maxRequestBody = 1 << 20

// oversizeReply is larger than the 10 MiB reply cap of the client, so the client refuses it.
const oversizeReply = 10<<20 + 1024

// defaultPageToken is the token a RepeatToken fault carries when the fault names none.
const defaultPageToken = "p0"

// Fault is a scripted answer that replaces the normal replay for the requests whose path starts with Path (every request
// when Path is empty). A zero Fault changes nothing. Status above 0 sends that status with an error body. The other
// switches send a special reply: a redirect, a 10 MiB body, a non-JSON body, a body that lacks its members, a repeated or
// empty page token, or an ADF description nested past the depth limit. Hang never answers, and Drop closes the
// connection with no reply. EchoToken and EchoEmail put the synthetic secret in an error or non-JSON body, so a test can
// prove Retriever never shows it.
type Fault struct {
	Path               string
	Status             int
	RetryAfter         string
	Location           string
	Oversize           bool
	NonJSON            bool
	MissingMember      bool
	RepeatToken        bool
	EmptyPageWithToken bool
	DepthBomb          bool
	Hang               bool
	Drop               bool
	EchoToken          bool
	EchoEmail          bool
	PageToken          string
}

// Request is one request the stub received, in arrival order. Body is the JSON body of a POST, as sent. AuthOK is true when
// the Basic header matched the synthetic credentials. The header value itself is never recorded. At is the arrival time.
// Host is the Host header, and ServerName is the TLS server name the client asked for (empty without TLS). Secure is true
// for a request that arrived over TLS, or that an in-process caller sent to an https URL. Bearer and Cookie say only
// whether those headers were present. InFlight counts the requests in flight when this one arrived, this one included.
type Request struct {
	Method     string
	Path       string
	Query      string
	Body       map[string]any
	AuthOK     bool
	Status     int
	At         time.Time
	Host       string
	Secure     bool
	ServerName string
	Bearer     bool
	Cookie     bool
	InFlight   int
}

// Stub is a scripted Jira Cloud. Its methods are safe for concurrent use.
type Stub struct {
	mu        sync.Mutex
	email     string
	token     string
	basic     string // the expected Authorization header; kept private, never recorded
	projects  []Project
	issues    []Issue
	comments  map[string][]Comment
	account   Account
	fault     Fault
	hasFault  bool
	faultLeft int // replies the fault still applies to; -1 is every reply
	cookie    string
	requests  []Request
	inFlight  int
	recorder  func(Request)
	now       func() time.Time
	done      chan struct{}
	closeOnce sync.Once
}

// New returns a stub that accepts the synthetic email and token and nothing else.
func New(email, token string) *Stub {
	s := &Stub{
		projects:  append([]Project(nil), defaultProjects...),
		issues:    append([]Issue(nil), defaultIssues...),
		comments:  copyComments(defaultComments),
		account:   defaultAccount,
		faultLeft: -1,
		now:       time.Now,
		done:      make(chan struct{}),
	}
	s.SetCredentials(email, token)
	return s
}

// SetCredentials replaces the e-mail address and the token the stub accepts.
func (s *Stub) SetCredentials(email, token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.email = email
	s.token = token
	pair := []byte(email + ":" + token)
	s.basic = "Basic " + base64.StdEncoding.EncodeToString(pair)
}

// Credentials returns the e-mail address and the token the stub accepts. A test uses them to name the pair it expects.
func (s *Stub) Credentials() (email, token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.email, s.token
}

// SetClock replaces the clock that stamps each request's arrival. A unit test passes its fake clock, so pacing shows in
// the stamps without any real wait.
func (s *Stub) SetClock(now func() time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.now = now
}

// SetFault makes the stub answer with f until ClearFault is called. A fault with a Path applies only to that path prefix.
func (s *Stub) SetFault(f Fault) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fault = f
	s.hasFault = true
	s.faultLeft = -1
}

// SetFaultFirst makes the stub answer the next n matching requests with f, then the normal replay resumes.
func (s *Stub) SetFaultFirst(n int, f Fault) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fault = f
	s.hasFault = true
	s.faultLeft = n
}

// ClearFault restores the normal replay.
func (s *Stub) ClearFault() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fault = Fault{}
	s.hasFault = false
	s.faultLeft = -1
}

// SetCookie makes every reply carry Set-Cookie for the named cookie, with a synthetic value. An empty name stops it.
func (s *Stub) SetCookie(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cookie = ""
	if name != "" {
		s.cookie = name + "=synthetic-session; Path=/"
	}
}

// SetProjects replaces the projects the stub holds.
func (s *Stub) SetProjects(projects []Project) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.projects = append([]Project(nil), projects...)
}

// SetIssues replaces the issues the stub holds.
func (s *Stub) SetIssues(issues []Issue) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.issues = append([]Issue(nil), issues...)
}

// SetComments replaces the comments of every issue. The map is copied.
func (s *Stub) SetComments(comments map[string][]Comment) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.comments = copyComments(comments)
}

// SetAccount replaces the account that GET /rest/api/3/myself answers with.
func (s *Stub) SetAccount(account Account) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.account = account
}

// Projects returns a copy of the projects the stub holds.
func (s *Stub) Projects() []Project {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Project(nil), s.projects...)
}

// Issues returns a copy of the issues the stub holds.
func (s *Stub) Issues() []Issue {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Issue(nil), s.issues...)
}

// Comments returns a copy of the comments of every issue.
func (s *Stub) Comments() map[string][]Comment {
	s.mu.Lock()
	defer s.mu.Unlock()
	return copyComments(s.comments)
}

// SetRecorder calls fn with each request as it is recorded. A process that keeps the record outside memory uses it.
func (s *Stub) SetRecorder(fn func(Request)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.recorder = fn
}

// Requests returns a copy of every request received, in arrival order.
func (s *Stub) Requests() []Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Request(nil), s.requests...)
}

// Close releases every request that is still hanging, so that a server can stop. It is safe to call more than once.
func (s *Stub) Close() {
	s.closeOnce.Do(func() { close(s.done) })
}

// replyControl lets the in-process transport learn what the stub did with a request, since it has no connection to close
// or to leave open. A request carries it in its context only when it comes from RoundTripper.
type replyControl struct {
	hung    bool
	dropped bool
}

// controlKey is the context key of the replyControl of an in-process request.
type controlKey struct{}

// withControl returns a context that carries a new replyControl, and the control.
func withControl(ctx context.Context) (context.Context, *replyControl) {
	ctl := &replyControl{}
	return context.WithValue(ctx, controlKey{}, ctl), ctl
}

// controlFrom returns the replyControl of an in-process request, or nil for a request that came over a socket.
func controlFrom(ctx context.Context) *replyControl {
	ctl, _ := ctx.Value(controlKey{}).(*replyControl)
	return ctl
}

// ServeHTTP answers one request: the fault, then the credential check, then the route. Each request is recorded once.
func (s *Stub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	req, body := s.arrive(r)

	s.mu.Lock()
	fault, faulted := s.faultFor(r.URL.Path)
	email, token := s.email, s.token
	cookie := s.cookie
	s.mu.Unlock()

	if cookie != "" {
		w.Header().Set("Set-Cookie", cookie)
	}
	switch {
	case faulted && fault.Hang:
		// The stub received the request, so it is recorded before the wait. The wait ends when the caller gives up or the stub closes.
		s.record(req)
		if ctl := controlFrom(r.Context()); ctl != nil {
			ctl.hung = true
		} else {
			select {
			case <-r.Context().Done():
			case <-s.done:
			}
		}
		s.depart()
		return
	case faulted && fault.Drop:
		if ctl := controlFrom(r.Context()); ctl != nil {
			ctl.dropped = true
		} else {
			hijackAndClose(w)
		}
	case faulted:
		req.Status = writeFault(w, fault, r.URL.Path, email, token)
	case !req.AuthOK:
		req.Status = writeError(w, http.StatusUnauthorized, "Client must be authenticated to access this resource.")
	default:
		req.Status = s.route(w, r, body)
	}
	s.depart()
	s.record(req)
}

// arrive builds the record of one request as it arrives, and counts it as in flight. It returns the JSON body too.
func (s *Stub) arrive(r *http.Request) (Request, map[string]any) {
	body := readJSONBody(r)
	auth := r.Header.Get("Authorization")
	authOK := s.authOK(auth)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.inFlight++
	return Request{
		Method:     r.Method,
		Path:       r.URL.Path,
		Query:      r.URL.RawQuery,
		Body:       body,
		AuthOK:     authOK,
		At:         s.now(),
		Host:       r.Host,
		Secure:     r.TLS != nil || r.URL.Scheme == "https",
		ServerName: serverName(r),
		Bearer:     strings.HasPrefix(auth, "Bearer "),
		Cookie:     r.Header.Get("Cookie") != "",
		InFlight:   s.inFlight,
	}, body
}

// depart counts one request as no longer in flight.
func (s *Stub) depart() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inFlight--
}

// serverName returns the TLS server name of a request that arrived over TLS, and "" for any other request.
func serverName(r *http.Request) string {
	if r.TLS == nil {
		return ""
	}
	return r.TLS.ServerName
}

// hijackAndClose closes the connection with no reply, for a Drop fault over a socket.
func hijackAndClose(w http.ResponseWriter) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		return
	}
	conn, _, err := hijacker.Hijack()
	if err != nil {
		return
	}
	_ = conn.Close()
}

// authOK reports whether the Authorization header is the expected Basic credential. The comparison is constant-time.
func (s *Stub) authOK(header string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return subtle.ConstantTimeCompare([]byte(header), []byte(s.basic)) == 1
}

// faultFor returns the scripted fault for path, when one applies, and counts the reply it answers. It must be called with
// s.mu held.
func (s *Stub) faultFor(path string) (Fault, bool) {
	if !s.hasFault || s.faultLeft == 0 {
		return Fault{}, false
	}
	if s.fault.Path != "" && !strings.HasPrefix(path, s.fault.Path) {
		return Fault{}, false
	}
	if s.faultLeft > 0 {
		s.faultLeft--
	}
	return s.fault, true
}

// transportFault reports whether the scripted fault for path makes an in-process request hang or drop. It counts the reply
// when it does, as ServeHTTP would.
func (s *Stub) transportFault(path string) (hang, drop bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fault, faulted := s.faultFor(path)
	if !faulted {
		return false, false
	}
	return fault.Hang, fault.Drop
}

// record appends one request to the recorder, and gives it to the recorder hook when one is set.
func (s *Stub) record(req Request) {
	s.mu.Lock()
	s.requests = append(s.requests, req)
	hook := s.recorder
	s.mu.Unlock()
	if hook != nil {
		hook(req)
	}
}

// route answers one authenticated request by method and path. It returns the status it wrote.
func (s *Stub) route(w http.ResponseWriter, r *http.Request, body map[string]any) int {
	switch {
	case r.URL.Path == removedSearchPath:
		return writeError(w, http.StatusGone, "This Jira server does not offer the endpoint.")
	case r.URL.Path == myselfPath && r.Method == http.MethodGet:
		return s.answerMyself(w)
	case r.URL.Path == projectSearchPath && r.Method == http.MethodGet:
		return s.answerProjects(w, r)
	case r.URL.Path == jqlSearchPath && r.Method == http.MethodPost:
		return s.answerSearch(w, body)
	case strings.HasPrefix(r.URL.Path, issuePrefix) && r.Method == http.MethodGet:
		rest := strings.TrimPrefix(r.URL.Path, issuePrefix)
		if key, ok := strings.CutSuffix(rest, "/comment"); ok {
			return s.answerComments(w, r, key)
		}
		return s.answerIssue(w, rest)
	}
	return writeError(w, http.StatusNotFound, "Not found.")
}

// answerMyself writes the signed-in account. It carries the e-mail address, so a test can prove Retriever never echoes it.
func (s *Stub) answerMyself(w http.ResponseWriter) int {
	s.mu.Lock()
	account := s.account
	email := s.email
	s.mu.Unlock()
	return writeJSON(w, http.StatusOK, map[string]any{
		"accountId":    account.AccountID,
		"accountType":  account.AccountType,
		"active":       account.Active,
		"displayName":  account.DisplayName,
		"emailAddress": email,
	})
}

// answerProjects writes one page of projects, filtered by the query text when one is given.
func (s *Stub) answerProjects(w http.ResponseWriter, r *http.Request) int {
	query := strings.ToLower(r.URL.Query().Get("query"))
	var matched []Project
	for _, p := range s.Projects() {
		if query == "" || strings.Contains(strings.ToLower(p.Key), query) || strings.Contains(strings.ToLower(p.Name), query) {
			matched = append(matched, p)
		}
	}
	start, size := page(r, 50)
	end := min(start+size, len(matched))
	var values []map[string]any
	for i := min(start, len(matched)); i < end; i++ {
		p := matched[i]
		values = append(values, map[string]any{
			"id": p.ID, "key": p.Key, "name": p.Name, "projectTypeKey": p.Type, "style": p.Style,
		})
	}
	if values == nil {
		values = []map[string]any{}
	}
	return writeJSON(w, http.StatusOK, map[string]any{
		"startAt": start, "maxResults": size, "total": len(matched), "isLast": end >= len(matched), "values": values,
	})
}

// answerSearch writes one page of issues for a JQL body. The stub refuses an empty JQL, a JQL with no "=" clause (it is
// unbounded), and a JQL that names the word "invalid", with 400. Otherwise it filters the issues by a project clause.
func (s *Stub) answerSearch(w http.ResponseWriter, body map[string]any) int {
	jql, _ := body["jql"].(string)
	if strings.TrimSpace(jql) == "" || !strings.Contains(jql, "=") || strings.Contains(jql, "invalid") {
		return writeError(w, http.StatusBadRequest, "Invalid JQL.")
	}
	project := projectClause(jql)
	var matched []Issue
	for _, is := range s.Issues() {
		if project == "" || strings.HasPrefix(is.Key, project+"-") {
			matched = append(matched, is)
		}
	}
	size := intField(body, "maxResults", 50)
	size = min(max(size, 1), 100)
	start := offsetToken(stringField(body, "nextPageToken"))
	end := min(start+size, len(matched))
	issues := []map[string]any{}
	for i := min(start, len(matched)); i < end; i++ {
		issues = append(issues, issueSummaryJSON(matched[i]))
	}
	reply := map[string]any{"issues": issues, "isLast": end >= len(matched)}
	if end < len(matched) {
		reply["nextPageToken"] = "p" + strconv.Itoa(end) // a token that names the next offset
	}
	return writeJSON(w, http.StatusOK, reply)
}

// answerIssue writes one issue with the fields Retriever reads. An unknown key is 404. A description that is not JSON is
// written as it is, so the reply itself is not JSON.
func (s *Stub) answerIssue(w http.ResponseWriter, key string) int {
	is, ok := s.issue(key)
	if !ok {
		return writeError(w, http.StatusNotFound, "Issue does not exist or you do not have permission to see it.")
	}
	if len(is.Description) > 0 && !json.Valid(is.Description) {
		return writeRaw(w, http.StatusOK, []byte(`{"id":"`+is.ID+`","key":"`+is.Key+`","fields":{"description":`+string(is.Description)+`}}`))
	}
	return writeJSON(w, http.StatusOK, map[string]any{"id": is.ID, "key": is.Key, "fields": issueFieldsJSON(is)})
}

// answerComments writes one page of the comments of an issue. An unknown issue is 404; an issue with none is an empty page.
func (s *Stub) answerComments(w http.ResponseWriter, r *http.Request, key string) int {
	if _, ok := s.issue(key); !ok {
		return writeError(w, http.StatusNotFound, "Issue does not exist or you do not have permission to see it.")
	}
	all := s.Comments()[key]
	start, size := page(r, 50)
	end := min(start+size, len(all))
	values := []map[string]any{}
	for i := min(start, len(all)); i < end; i++ {
		c := all[i]
		values = append(values, map[string]any{
			"id":      c.ID,
			"author":  map[string]any{"displayName": c.Author, "accountId": "acc-" + c.ID},
			"created": c.Created,
			"updated": c.Updated,
			"body":    c.Body,
		})
	}
	return writeJSON(w, http.StatusOK, map[string]any{
		"startAt": start, "maxResults": size, "total": len(all), "comments": values,
	})
}

// issue returns the issue with key, if the stub holds it.
func (s *Stub) issue(key string) (Issue, bool) {
	for _, is := range s.Issues() {
		if is.Key == key {
			return is, true
		}
	}
	return Issue{}, false
}

// issueSummaryJSON returns one issue as a search row, with the fixed fields of the client's search list.
func issueSummaryJSON(is Issue) map[string]any {
	return map[string]any{
		"id": is.ID, "key": is.Key,
		"fields": map[string]any{
			"summary":   is.Summary,
			"status":    map[string]any{"name": is.Status},
			"issuetype": map[string]any{"name": is.Type},
			"priority":  map[string]any{"name": is.Priority},
			"assignee":  nameOrNull(is.Assignee),
			"updated":   is.Updated,
			"project":   map[string]any{"key": projectOf(is.Key)},
			"parent":    keyOrNull(is.Parent),
		},
	}
}

// issueFieldsJSON returns the fields of one issue, as the issue endpoint sends them.
func issueFieldsJSON(is Issue) map[string]any {
	description := any(nil)
	if len(is.Description) > 0 && string(is.Description) != "null" {
		description = json.RawMessage(is.Description)
	}
	return map[string]any{
		"summary":     is.Summary,
		"description": description,
		"issuetype":   map[string]any{"name": is.Type},
		"status":      map[string]any{"name": is.Status},
		"priority":    map[string]any{"name": is.Priority},
		"resolution":  nameOrNull(is.Resolution),
		"assignee":    nameOrNull(is.Assignee),
		"reporter":    nameOrNull(is.Reporter),
		"labels":      is.Labels,
		"created":     is.Created,
		"updated":     is.Updated,
		"duedate":     nullable(is.Due),
		"project":     map[string]any{"key": projectOf(is.Key), "name": "Example Org"},
		"parent":      keyOrNull(is.Parent),
	}
}

// nameOrNull returns {"displayName": name}, or null when name is empty.
func nameOrNull(name string) any {
	if name == "" {
		return nil
	}
	return map[string]any{"displayName": name, "name": name}
}

// keyOrNull returns {"key": key}, or null when key is empty.
func keyOrNull(key string) any {
	if key == "" {
		return nil
	}
	return map[string]any{"key": key}
}

// nullable returns s, or null when s is empty.
func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// projectOf returns the project part of an issue key.
func projectOf(key string) string {
	project, _, _ := strings.Cut(key, "-")
	return project
}

// projectClause returns the project key of a JQL that has project = KEY or project = "KEY", or "".
func projectClause(jql string) string {
	lower := strings.ToLower(jql)
	i := strings.Index(lower, "project")
	if i < 0 {
		return ""
	}
	rest := strings.TrimLeft(jql[i+len("project"):], " \t\n\r")
	rest, ok := strings.CutPrefix(rest, "=")
	if !ok {
		return ""
	}
	rest = strings.TrimLeft(rest, " \t\n\r\"")
	end := 0
	for end < len(rest) && (rest[end] >= 'A' && rest[end] <= 'Z' || rest[end] >= '0' && rest[end] <= '9' || rest[end] == '_') {
		end++
	}
	return rest[:end]
}

// page reads startAt and maxResults from the query string, with a default size, and bounds the size to 1 to 100.
func page(r *http.Request, defaultSize int) (start, size int) {
	q := r.URL.Query()
	start = max(atoi(q.Get("startAt")), 0)
	size = atoi(q.Get("maxResults"))
	if size <= 0 {
		size = defaultSize
	}
	return start, min(size, 100)
}

// offsetToken encodes an offset as a page token, and reads one back. A token that does not parse is offset 0.
func offsetToken(s string) int {
	n, ok := strings.CutPrefix(s, "p")
	if !ok {
		return 0
	}
	return atoi(n)
}

// atoi parses a non-negative integer, or returns 0.
func atoi(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// intField reads an integer member of a JSON body, or def.
func intField(body map[string]any, key string, def int) int {
	f, ok := body[key].(float64)
	if !ok {
		return def
	}
	return int(f)
}

// stringField reads a string member of a JSON body, or "".
func stringField(body map[string]any, key string) string {
	s, _ := body[key].(string)
	return s
}

// readJSONBody reads a JSON object from a request body, up to maxRequestBody. A body that is not a JSON object is nil.
func readJSONBody(r *http.Request) map[string]any {
	if r.Body == nil {
		return nil
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBody))
	if err != nil || len(data) == 0 {
		return nil
	}
	var body map[string]any
	if json.Unmarshal(data, &body) != nil {
		return nil
	}
	return body
}

// writeJSON writes v as a JSON reply with status and returns status.
func writeJSON(w http.ResponseWriter, status int, v any) int {
	data, err := json.Marshal(v)
	if err != nil {
		return writeError(w, http.StatusInternalServerError, "The stub could not encode the reply.")
	}
	return writeRaw(w, status, data)
}

// writeRaw writes data as a JSON reply with status and returns status.
func writeRaw(w http.ResponseWriter, status int, data []byte) int {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(data)
	return status
}

// writeError writes a Jira-style error body with status and returns status.
func writeError(w http.ResponseWriter, status int, message string) int {
	return writeJSON(w, status, map[string]any{"errorMessages": []string{message}, "errors": map[string]any{}})
}

// writeFault writes the scripted reply and returns its status. The echo switches add the synthetic e-mail address or token
// to the body, so a test can check that Retriever shows neither.
func writeFault(w http.ResponseWriter, f Fault, path, email, token string) int {
	switch {
	case f.Oversize:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.Copy(w, io.MultiReader(strings.NewReader(`{"x":"`), repeatReader('a', oversizeReply), strings.NewReader(`"}`)))
		return http.StatusOK
	case f.NonJSON:
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "<html><body>Log in to continue"+echoText(f, email, token)+"</body></html>")
		return http.StatusOK
	case f.MissingMember:
		return writeJSON(w, http.StatusOK, map[string]any{"unexpected": true})
	case f.RepeatToken:
		token := f.PageToken
		if token == "" {
			token = defaultPageToken
		}
		// One issue keeps the page non-empty, so a client reads the repeated token on its second request, not on the first.
		return writeJSON(w, http.StatusOK, map[string]any{
			"issues": []any{issueSummaryJSON(defaultIssues[0])}, "isLast": false, "nextPageToken": token,
		})
	case f.EmptyPageWithToken:
		return writeJSON(w, http.StatusOK, map[string]any{"issues": []any{}, "isLast": false, "nextPageToken": "p500"})
	case f.DepthBomb:
		return writeJSON(w, http.StatusOK, map[string]any{"id": "10012", "key": IssueKey, "fields": map[string]any{
			"summary": "Depth bomb", "description": json.RawMessage(nestedADF(40)), "issuetype": map[string]any{"name": "Bug"},
			"status": map[string]any{"name": "To Do"}, "priority": map[string]any{"name": "Medium"}, "labels": []string{},
			"created": "2026-11-02T09:15:00.000+0700", "updated": "2026-11-02T09:15:00.000+0700", "project": map[string]any{"key": ProjectKey},
		}})
	}
	if f.Location != "" {
		w.Header().Set("Location", f.Location)
	}
	if f.RetryAfter != "" {
		w.Header().Set("Retry-After", f.RetryAfter)
	}
	status := f.Status
	if status == 0 {
		status = http.StatusFound
		if f.Location == "" {
			status = http.StatusInternalServerError
		}
	}
	if status >= 300 && status < 400 {
		w.WriteHeader(status)
		return status
	}
	return writeError(w, status, fmt.Sprintf("Scripted fault for %s.%s", path, echoText(f, email, token)))
}

// echoText returns the synthetic secrets that a fault's echo switches ask to repeat, each with a leading space.
func echoText(f Fault, email, token string) string {
	var out strings.Builder
	if f.EchoToken {
		out.WriteString(" " + token)
	}
	if f.EchoEmail {
		out.WriteString(" " + email)
	}
	return out.String()
}

// nestedADF returns a document whose paragraph sits depth panels deep.
func nestedADF(depth int) string {
	inner := `{"type":"paragraph","content":[{"type":"text","text":"too deep"}]}`
	for i := 0; i < depth; i++ {
		inner = `{"type":"panel","content":[` + inner + `]}`
	}
	return `{"type":"doc","version":1,"content":[` + inner + `]}`
}

// repeatReader yields n copies of b.
func repeatReader(b byte, n int) io.Reader {
	return bytes.NewReader(bytes.Repeat([]byte{b}, n))
}

// copyComments returns a deep enough copy of a comment map: its slices are new, so the caller may change them.
func copyComments(src map[string][]Comment) map[string][]Comment {
	out := make(map[string][]Comment, len(src))
	for key, list := range src {
		out[key] = append([]Comment(nil), list...)
	}
	return out
}
