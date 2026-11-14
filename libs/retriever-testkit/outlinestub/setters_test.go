package outlinestub_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-testkit/outlinestub"
)

// versionedCall sends one request with an X-API-Version header to the stub's ServeHTTP and returns the recorder.
func versionedCall(s *outlinestub.Stub, method, body, version string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, outlineURL(method), strings.NewReader(body))
	req.Header.Set("X-API-Version", version)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	return rec
}

// replyObject decodes a JSON reply and returns its top-level object.
func replyObject(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err != nil {
		t.Fatalf("reply is not a JSON object: %v: %s", err, body)
	}
	return obj
}

func TestSetUnofferedRefusesTheNamedMethodsAndReplacesTheList(t *testing.T) {
	s := newStub(outlinestub.WithCollections(outlinestub.NumberedCollections(1)...))

	s.SetUnoffered("collections.list")
	rec := serveRecorded(t, s, http.MethodPost, outlineURL("collections.list"), `{"offset":0,"limit":25}`)
	if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), "Endpoint not found") {
		t.Fatalf("an unoffered method = %d %q, want 404 Endpoint not found", rec.Code, rec.Body.String())
	}

	s.SetUnoffered("auth.info")
	if rec := serveRecorded(t, s, http.MethodPost, outlineURL("collections.list"), `{"offset":0,"limit":25}`); rec.Code != http.StatusOK {
		t.Fatalf("collections.list after the list changed = %d, want 200 again", rec.Code)
	}
	if rec := serveRecorded(t, s, http.MethodPost, outlineURL("auth.info"), `{}`); rec.Code != http.StatusNotFound {
		t.Fatalf("auth.info = %d, want 404 once it is unoffered", rec.Code)
	}
}

func TestSetReplyAnswersOneMethodWithTheGivenReplyUntilItIsReplaced(t *testing.T) {
	s := newStub()

	s.SetReply("collections.info", http.StatusAccepted, `{"data":{"id":"fixed"}}`)
	rec := serveRecorded(t, s, http.MethodPost, outlineURL("collections.info"), `{"id":"anything"}`)
	if rec.Code != http.StatusAccepted || rec.Body.String() != `{"data":{"id":"fixed"}}` {
		t.Fatalf("the reply = %d %q, want the fixed 202 reply", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}

	s.SetReply("collections.info", http.StatusInternalServerError, `{"ok":false}`)
	if rec := serveRecorded(t, s, http.MethodPost, outlineURL("collections.info"), `{"id":"anything"}`); rec.Code != http.StatusInternalServerError {
		t.Fatalf("the replaced reply = %d, want 500", rec.Code)
	}

	if rec := serveRecorded(t, s, http.MethodPost, outlineURL("auth.info"), `{}`); rec.Code != http.StatusOK {
		t.Fatalf("auth.info = %d, want the normal replay, which SetReply left alone", rec.Code)
	}
}

func TestSetIgnoreAPIVersionAnswersTheUnwrappedDocumentWhateverTheHeaderAsks(t *testing.T) {
	docs := []outlinestub.Document{{ID: "doc-1", CollectionID: "c1", Title: "Setup", URL: "/doc/setup", Markdown: "Run it."}}
	s := newStub(outlinestub.WithDocuments(docs...))

	wrapped := replyObject(t, versionedCall(s, "documents.info", `{"id":"doc-1"}`, "2").Body.Bytes())
	if _, ok := wrapped["data"].(map[string]any)["document"]; !ok {
		t.Fatalf("version 2 without ignoring it = %v, want data.document", wrapped)
	}

	s.SetIgnoreAPIVersion(true)
	for _, version := range []string{"2", "3", ""} {
		obj := replyObject(t, versionedCall(s, "documents.info", `{"id":"doc-1"}`, version).Body.Bytes())
		data, _ := obj["data"].(map[string]any)
		if data["title"] != "Setup" || data["text"] != "Run it." {
			t.Fatalf("version %q with the version ignored = %v, want the unwrapped document", version, obj)
		}
	}
}

func TestSetEditorOnlyDocumentsAnswersEditorDataWithNoText(t *testing.T) {
	docs := []outlinestub.Document{{ID: "doc-1", CollectionID: "c1", Title: "Setup", URL: "/doc/setup", Markdown: "Run it."}}
	s := newStub(outlinestub.WithDocuments(docs...))

	s.SetEditorOnlyDocuments(true)
	obj := replyObject(t, versionedCall(s, "documents.info", `{"id":"doc-1"}`, "").Body.Bytes())
	document, ok := obj["data"].(map[string]any)["document"].(map[string]any)
	if !ok {
		t.Fatalf("editor-only reply = %v, want data.document", obj)
	}
	if _, has := document["text"]; has {
		t.Fatal("an editor-only document carries Markdown text")
	}
	if editor, _ := document["data"].(map[string]any); editor["type"] != "doc" {
		t.Fatalf("editor-only document data = %v, want an editor doc", document["data"])
	}

	s.SetEditorOnlyDocuments(false)
	obj = replyObject(t, versionedCall(s, "documents.info", `{"id":"doc-1"}`, "").Body.Bytes())
	if obj["data"].(map[string]any)["text"] != "Run it." {
		t.Fatalf("with editor-only off = %v, want the Markdown text back", obj)
	}
}

func TestSetAccountChangesTheUserAndTheTeamAndKeepsAnEmptyField(t *testing.T) {
	s := newStub()
	authInfo := func() (string, string) {
		obj := replyObject(t, serveRecorded(t, s, http.MethodPost, outlineURL("auth.info"), `{}`).Body.Bytes())
		data := obj["data"].(map[string]any)
		return data["user"].(map[string]any)["name"].(string), data["team"].(map[string]any)["name"].(string)
	}

	if user, team := authInfo(); user != "Fixture Author" || team != "Fixture Team" {
		t.Fatalf("the default account = %q %q, want the fixture account", user, team)
	}

	s.SetAccount("Example Author", "Example Team")
	if user, team := authInfo(); user != "Example Author" || team != "Example Team" {
		t.Fatalf("the account after SetAccount = %q %q", user, team)
	}

	s.SetAccount("", "Second Team")
	if user, team := authInfo(); user != "Fixture Author" || team != "Second Team" {
		t.Fatalf("an empty user kept %q and team %q, want the default user and the new team", user, team)
	}
}

func TestSetFaultFirstZeroLeavesEveryRequestAnswered(t *testing.T) {
	s := newStub()
	s.SetFaultFirst(0, outlinestub.Fault{Status: http.StatusInternalServerError})

	if rec := serveRecorded(t, s, http.MethodPost, outlineURL("auth.info"), `{}`); rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 when the fault covers no request", rec.Code)
	}
}

func TestSetFaultFirstCountsOnlyTheRequestsItCovers(t *testing.T) {
	s := newStub()
	s.SetFaultFirst(1, outlinestub.Fault{Status: http.StatusServiceUnavailable, Body: "maintenance"})

	first := serveRecorded(t, s, http.MethodPost, outlineURL("auth.info"), `{}`)
	second := serveRecorded(t, s, http.MethodPost, outlineURL("auth.info"), `{}`)

	if first.Code != http.StatusServiceUnavailable || first.Body.String() != "maintenance" {
		t.Fatalf("the first request = %d %q, want the fault", first.Code, first.Body.String())
	}
	if second.Code != http.StatusOK {
		t.Fatalf("the second request = %d, want the normal replay after one faulted request", second.Code)
	}
}

func TestFaultReplyTable(t *testing.T) {
	cases := []struct {
		name       string
		fault      outlinestub.Fault
		method     string
		wantStatus int
		wantBody   string
		wantHeader map[string]string
		wantType   string
	}{
		{
			name:       "a fault for another method leaves this one alone",
			fault:      outlinestub.Fault{Method: "documents.info", Status: http.StatusInternalServerError},
			method:     "auth.info",
			wantStatus: http.StatusOK,
		},
		{
			name:       "a fault with only a header leaves the normal replay in place",
			fault:      outlinestub.Fault{RetryAfter: "3"},
			method:     "auth.info",
			wantStatus: http.StatusOK,
		},
		{
			name:       "a plain body is sent as text with the given status",
			fault:      outlinestub.Fault{Body: "plain text fault"},
			method:     "auth.info",
			wantStatus: http.StatusOK,
			wantBody:   "plain text fault",
			wantType:   "text/plain",
		},
		{
			name:       "a plain body keeps its own status",
			fault:      outlinestub.Fault{Status: http.StatusServiceUnavailable, Body: "maintenance"},
			method:     "auth.info",
			wantStatus: http.StatusServiceUnavailable,
			wantBody:   "maintenance",
			wantType:   "text/plain",
		},
		{
			name:       "an authorization echo repeats the header in text",
			fault:      outlinestub.Fault{Status: http.StatusUnauthorized, EchoAuthorization: true},
			method:     "auth.info",
			wantStatus: http.StatusUnauthorized,
			wantBody:   "upstream said: " + testAuthorization,
			wantType:   "text/plain",
		},
		{
			name:       "a redirect carries its location and no body",
			fault:      outlinestub.Fault{Status: http.StatusFound, Location: "/moved"},
			method:     "auth.info",
			wantStatus: http.StatusFound,
			wantHeader: map[string]string{"Location": "/moved"},
		},
		{
			name:       "an error status gets the sanitized error body and the retry header",
			fault:      outlinestub.Fault{Status: http.StatusTooManyRequests, RetryAfter: "9"},
			method:     "auth.info",
			wantStatus: http.StatusTooManyRequests,
			wantHeader: map[string]string{"Retry-After": "9"},
			wantType:   "application/json",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newStub()
			s.SetFault(tc.fault)

			rec := serveRecorded(t, s, http.MethodPost, outlineURL(tc.method), `{}`)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if tc.wantBody != "" && rec.Body.String() != tc.wantBody {
				t.Fatalf("body = %q, want %q", rec.Body.String(), tc.wantBody)
			}
			if tc.wantBody == "" && tc.wantStatus == http.StatusFound && rec.Body.Len() != 0 {
				t.Fatalf("a redirect wrote %q, want no body", rec.Body.String())
			}
			for name, want := range tc.wantHeader {
				if got := rec.Header().Get(name); got != want {
					t.Fatalf("%s = %q, want %q", name, got, want)
				}
			}
			if tc.wantType != "" && rec.Header().Get("Content-Type") != tc.wantType {
				t.Fatalf("Content-Type = %q, want %q", rec.Header().Get("Content-Type"), tc.wantType)
			}
		})
	}
}

func TestAnInProcessStubHasNoListenerNoURLAndNoCertificate(t *testing.T) {
	s := newStub()

	if u := s.URL(); u != "" {
		t.Fatalf("URL = %q, want none for an in-process stub", u)
	}
	if c := s.Client(); c != nil {
		t.Fatal("an in-process stub has a client")
	}
	if _, err := s.CAPEM(); err == nil {
		t.Fatal("CAPEM succeeded for a stub that was never started")
	}
	if _, err := s.WriteCAFile("no-such-dir"); err == nil {
		t.Fatal("WriteCAFile succeeded for a stub that was never started")
	}
}

func TestCloseIsSafeToRepeatOnAnInProcessStub(t *testing.T) {
	s := newStub()
	s.Close()
	s.Close()

	if rec := serveRecorded(t, s, http.MethodPost, outlineURL("auth.info"), `{}`); rec.Code != http.StatusOK {
		t.Fatalf("a closed in-process stub answered %d, want its replies to keep working", rec.Code)
	}
}
