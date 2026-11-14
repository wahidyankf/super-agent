package outlinestub_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-testkit/outlinestub"
)

// replayStub holds two collections and three documents: doc-2 is a child of doc-1, and doc-3 is in the second
// collection. Each method test reads this one dataset.
func replayStub(opts ...outlinestub.Option) *outlinestub.Stub {
	base := []outlinestub.Option{
		outlinestub.WithCollections(
			outlinestub.Collection{ID: "c1", Name: "Engineering"},
			outlinestub.Collection{ID: "c2", Name: "Finance"},
		),
		outlinestub.WithDocuments(
			outlinestub.Document{ID: "doc-1", CollectionID: "c1", Title: "Setup Guide", URL: "/doc/setup",
				Markdown: "Install the tool and run it."},
			outlinestub.Document{ID: "doc-2", CollectionID: "c1", ParentID: "doc-1", Title: "Install Steps",
				URL: "/doc/install", Markdown: "Run the installer."},
			outlinestub.Document{ID: "doc-3", CollectionID: "c2", Title: "Budget", URL: "/doc/budget",
				Markdown: "Quarterly numbers mention installer costs."},
		),
	}
	return newStub(append(base, opts...)...)
}

// listOf returns the data array of a reply, failing the test when the reply has none.
func listOf(t *testing.T, body []byte) []any {
	t.Helper()
	obj := replyObject(t, body)
	list, ok := obj["data"].([]any)
	if !ok {
		t.Fatalf("reply has no data array: %s", body)
	}
	return list
}

// idsOf returns the id of each object of a data array, in order.
func idsOf(t *testing.T, list []any) []string {
	t.Helper()
	ids := make([]string, 0, len(list))
	for _, item := range list {
		obj, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("list item %v is not an object", item)
		}
		id, _ := obj["id"].(string)
		ids = append(ids, id)
	}
	return ids
}

func TestCollectionsListPagesTheCollectionsAndNamesThePagination(t *testing.T) {
	cases := []struct {
		name       string
		body       string
		wantStatus int
		wantIDs    []string
	}{
		{name: "the first page holds every collection", body: `{"offset":0,"limit":25}`,
			wantStatus: http.StatusOK, wantIDs: []string{"c1", "c2"}},
		{name: "an offset past the end holds none", body: `{"offset":5,"limit":25}`,
			wantStatus: http.StatusOK, wantIDs: []string{}},
		{name: "a zero limit is refused", body: `{"offset":0,"limit":0}`, wantStatus: http.StatusBadRequest},
		{name: "a negative offset is refused", body: `{"offset":-1,"limit":25}`, wantStatus: http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := replayStub()

			rec := serveRecorded(t, s, http.MethodPost, outlineURL("collections.list"), tc.body)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d: %s", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantStatus != http.StatusOK {
				return
			}
			list := listOf(t, rec.Body.Bytes())
			if got := idsOf(t, list); strings.Join(got, ",") != strings.Join(tc.wantIDs, ",") {
				t.Fatalf("ids = %v, want %v", got, tc.wantIDs)
			}
			pagination := replyObject(t, rec.Body.Bytes())["pagination"].(map[string]any)
			if pagination["limit"] != float64(25) {
				t.Fatalf("pagination = %v, want the requested limit of 25 echoed", pagination)
			}
		})
	}
}

func TestCollectionsInfoAnswersOneCollectionOrNotFound(t *testing.T) {
	s := replayStub()

	rec := serveRecorded(t, s, http.MethodPost, outlineURL("collections.info"), `{"id":"c1"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("collections.info c1 = %d, want 200", rec.Code)
	}
	data := dataObject(t, rec.Body.Bytes())
	if data["id"] != "c1" || data["name"] != "Engineering" || data["url"] != "/collection/c1" {
		t.Fatalf("collection = %v, want c1 Engineering at /collection/c1", data)
	}

	if rec := serveRecorded(t, s, http.MethodPost, outlineURL("collections.info"), `{"id":"c9"}`); rec.Code != http.StatusNotFound {
		t.Fatalf("collections.info c9 = %d, want 404", rec.Code)
	}
}

func TestCollectionsDocumentsNestsEachChildUnderItsParent(t *testing.T) {
	s := replayStub()

	rec := serveRecorded(t, s, http.MethodPost, outlineURL("collections.documents"), `{"id":"c1"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("collections.documents c1 = %d, want 200", rec.Code)
	}
	roots := listOf(t, rec.Body.Bytes())
	if got := idsOf(t, roots); strings.Join(got, ",") != "doc-1" {
		t.Fatalf("roots = %v, want [doc-1] only, because doc-2 is its child", got)
	}
	root := roots[0].(map[string]any)
	children := root["children"].([]any)
	if got := idsOf(t, children); strings.Join(got, ",") != "doc-2" {
		t.Fatalf("children of doc-1 = %v, want [doc-2]", got)
	}
	if children[0].(map[string]any)["title"] != "Install Steps" {
		t.Fatalf("child title = %v, want Install Steps", children[0])
	}

	if rec := serveRecorded(t, s, http.MethodPost, outlineURL("collections.documents"), `{"id":"c9"}`); rec.Code != http.StatusNotFound {
		t.Fatalf("collections.documents c9 = %d, want 404", rec.Code)
	}
}

func TestDocumentInfoAnswersTheShapeTheVersionHeaderAsks(t *testing.T) {
	cases := []struct {
		name    string
		version string
		check   func(t *testing.T, obj map[string]any)
	}{
		{
			name:    "no version header gets the unwrapped document with its text",
			version: "",
			check: func(t *testing.T, obj map[string]any) {
				data := obj["data"].(map[string]any)
				if data["title"] != "Setup Guide" || data["text"] != "Install the tool and run it." {
					t.Fatalf("unwrapped document = %v", data)
				}
				if data["parentDocumentId"] != nil || data["revision"] != float64(1) {
					t.Fatalf("root document fields = %v, want no parent and revision 1", data)
				}
			},
		},
		{
			name:    "version 1 gets the unwrapped document",
			version: "1",
			check: func(t *testing.T, obj map[string]any) {
				if obj["data"].(map[string]any)["title"] != "Setup Guide" {
					t.Fatalf("version 1 = %v, want the unwrapped document", obj)
				}
			},
		},
		{
			name:    "version 2 wraps the document with its text",
			version: "2",
			check: func(t *testing.T, obj map[string]any) {
				document := obj["data"].(map[string]any)["document"].(map[string]any)
				if document["text"] != "Install the tool and run it." {
					t.Fatalf("version 2 document = %v, want its text", document)
				}
			},
		},
		{
			name:    "version 3 wraps editor data and drops the text",
			version: "3",
			check: func(t *testing.T, obj map[string]any) {
				document := obj["data"].(map[string]any)["document"].(map[string]any)
				if _, has := document["text"]; has {
					t.Fatalf("version 3 document still has text: %v", document)
				}
				if editor := document["data"].(map[string]any); editor["type"] != "doc" {
					t.Fatalf("version 3 editor data = %v, want an editor doc", editor)
				}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := replayStub()

			rec := versionedCall(s, "documents.info", `{"id":"doc-1"}`, tc.version)

			if rec.Code != http.StatusOK {
				t.Fatalf("documents.info = %d, want 200", rec.Code)
			}
			tc.check(t, replyObject(t, rec.Body.Bytes()))
		})
	}

	s := replayStub()
	if rec := serveRecorded(t, s, http.MethodPost, outlineURL("documents.info"), `{"id":"doc-9"}`); rec.Code != http.StatusNotFound {
		t.Fatalf("documents.info doc-9 = %d, want 404", rec.Code)
	}
}

func TestDocumentExportAnswersTheMarkdownOfOneDocument(t *testing.T) {
	s := replayStub()

	rec := serveRecorded(t, s, http.MethodPost, outlineURL("documents.export"), `{"id":"doc-3"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("documents.export doc-3 = %d, want 200", rec.Code)
	}
	if got := replyObject(t, rec.Body.Bytes())["data"]; got != "Quarterly numbers mention installer costs." {
		t.Fatalf("export data = %v, want the Markdown text", got)
	}

	if rec := serveRecorded(t, s, http.MethodPost, outlineURL("documents.export"), `{"id":"doc-9"}`); rec.Code != http.StatusNotFound {
		t.Fatalf("documents.export doc-9 = %d, want 404", rec.Code)
	}
}

func TestDocumentsListPagesDocumentsAndNamesTheNextPage(t *testing.T) {
	cases := []struct {
		name       string
		body       string
		wantStatus int
		wantIDs    []string
		wantNext   string
	}{
		{name: "every document, with the next page named", body: `{"limit":10,"offset":0}`,
			wantStatus: http.StatusOK, wantIDs: []string{"doc-1", "doc-2", "doc-3"},
			wantNext: "/api/documents.list?offset=10&limit=10"},
		{name: "one collection only", body: `{"limit":10,"offset":0,"collectionId":"c1"}`,
			wantStatus: http.StatusOK, wantIDs: []string{"doc-1", "doc-2"},
			wantNext: "/api/documents.list?offset=10&limit=10"},
		{name: "a page from the middle", body: `{"limit":1,"offset":1}`,
			wantStatus: http.StatusOK, wantIDs: []string{"doc-2"},
			wantNext: "/api/documents.list?offset=2&limit=1"},
		{name: "an offset past the end", body: `{"limit":5,"offset":40}`,
			wantStatus: http.StatusOK, wantIDs: []string{}, wantNext: "/api/documents.list?offset=45&limit=5"},
		{name: "a zero limit is refused", body: `{"limit":0,"offset":0}`, wantStatus: http.StatusBadRequest},
		{name: "a limit over one hundred is refused", body: `{"limit":101,"offset":0}`, wantStatus: http.StatusBadRequest},
		{name: "a negative offset is refused", body: `{"limit":5,"offset":-1}`, wantStatus: http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := replayStub()

			rec := serveRecorded(t, s, http.MethodPost, outlineURL("documents.list"), tc.body)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if tc.wantStatus != http.StatusOK {
				return
			}
			if got := idsOf(t, listOf(t, rec.Body.Bytes())); strings.Join(got, ",") != strings.Join(tc.wantIDs, ",") {
				t.Fatalf("ids = %v, want %v", got, tc.wantIDs)
			}
			pagination := replyObject(t, rec.Body.Bytes())["pagination"].(map[string]any)
			if pagination["nextPath"] != tc.wantNext {
				t.Fatalf("nextPath = %v, want %q", pagination["nextPath"], tc.wantNext)
			}
		})
	}
}

func TestDocumentsSearchFindsTitlesAndBodiesCaseInsensitively(t *testing.T) {
	cases := []struct {
		name       string
		body       string
		wantStatus int
		wantIDs    []string
	}{
		{name: "a word in a body and a title", body: `{"query":"installer","limit":10,"offset":0}`,
			wantStatus: http.StatusOK, wantIDs: []string{"doc-2", "doc-3"}},
		{name: "the query ignores case and matches inside a word", body: `{"query":"INSTALL","limit":10,"offset":0}`,
			wantStatus: http.StatusOK, wantIDs: []string{"doc-1", "doc-2", "doc-3"}},
		{name: "one collection only", body: `{"query":"installer","limit":10,"offset":0,"collectionId":"c2"}`,
			wantStatus: http.StatusOK, wantIDs: []string{"doc-3"}},
		{name: "a page of the hits", body: `{"query":"installer","limit":1,"offset":1}`,
			wantStatus: http.StatusOK, wantIDs: []string{"doc-3"}},
		{name: "an empty query is refused", body: `{"query":"","limit":10,"offset":0}`, wantStatus: http.StatusBadRequest},
		{name: "a zero limit is refused", body: `{"query":"x","limit":0,"offset":0}`, wantStatus: http.StatusBadRequest},
		{name: "a negative offset is refused", body: `{"query":"x","limit":5,"offset":-1}`, wantStatus: http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := replayStub()

			rec := serveRecorded(t, s, http.MethodPost, outlineURL("documents.search"), tc.body)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if tc.wantStatus != http.StatusOK {
				return
			}
			list := listOf(t, rec.Body.Bytes())
			var ids []string
			for _, item := range list {
				result := item.(map[string]any)
				if _, ok := result["context"].(string); !ok || result["ranking"] != 0.5 {
					t.Fatalf("result %v lacks its context or ranking", result)
				}
				ids = append(ids, result["document"].(map[string]any)["id"].(string))
			}
			if strings.Join(ids, ",") != strings.Join(tc.wantIDs, ",") {
				t.Fatalf("ids = %v, want %v", ids, tc.wantIDs)
			}
		})
	}
}

func TestDocumentsSearchTitlesIgnoresWordsThatAppearOnlyInBodies(t *testing.T) {
	cases := []struct {
		name       string
		body       string
		wantStatus int
		wantIDs    []string
	}{
		{name: "a title word", body: `{"query":"install","limit":10,"offset":0}`,
			wantStatus: http.StatusOK, wantIDs: []string{"doc-2"}},
		{name: "a word only in a body finds nothing", body: `{"query":"tool","limit":10,"offset":0}`,
			wantStatus: http.StatusOK, wantIDs: []string{}},
		{name: "one collection only", body: `{"query":"setup","limit":10,"offset":0,"collectionId":"c1"}`,
			wantStatus: http.StatusOK, wantIDs: []string{"doc-1"}},
		{name: "an empty query is refused", body: `{"query":"","limit":10,"offset":0}`, wantStatus: http.StatusBadRequest},
		{name: "a limit over one hundred is refused", body: `{"query":"x","limit":101,"offset":0}`, wantStatus: http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := replayStub()

			rec := serveRecorded(t, s, http.MethodPost, outlineURL("documents.search_titles"), tc.body)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if tc.wantStatus != http.StatusOK {
				return
			}
			if got := idsOf(t, listOf(t, rec.Body.Bytes())); strings.Join(got, ",") != strings.Join(tc.wantIDs, ",") {
				t.Fatalf("ids = %v, want %v", got, tc.wantIDs)
			}
		})
	}
}

func TestAnUnknownMethodIsTheRouterCatchAll(t *testing.T) {
	s := replayStub()

	rec := serveRecorded(t, s, http.MethodPost, outlineURL("users.unknownThing"), `{}`)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if obj := replyObject(t, rec.Body.Bytes()); obj["message"] != "Endpoint not found" || obj["status"] != float64(404) {
		t.Fatalf("catch-all body = %v, want status 404 and Endpoint not found", obj)
	}
}

func TestOnlyPostReachesTheReplayAndAnyOtherVerbIsRefused(t *testing.T) {
	s := replayStub()

	rec := serveRecorded(t, s, http.MethodGet, outlineURL("collections.list"), "")

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET = %d, want 405", rec.Code)
	}
	if obj := replyObject(t, rec.Body.Bytes()); obj["status"] != float64(405) {
		t.Fatalf("405 body = %v, want the sanitized error envelope", obj)
	}
}

func TestEndpointNotFoundForAnUnofferedMethodCarriesTheRouterMessage(t *testing.T) {
	s := replayStub()
	s.SetUnoffered("documents.export")

	rec := serveRecorded(t, s, http.MethodPost, outlineURL("documents.export"), `{"id":"doc-1"}`)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if obj := replyObject(t, rec.Body.Bytes()); obj["message"] != "Endpoint not found" {
		t.Fatalf("body = %v, want the router's Endpoint not found", obj)
	}
}

func TestAnErrorStatusWithNoFixtureGetsTheGenericEnvelope(t *testing.T) {
	s := replayStub()
	s.SetFault(outlinestub.Fault{Status: http.StatusTeapot})

	rec := serveRecorded(t, s, http.MethodPost, outlineURL("auth.info"), `{}`)

	if rec.Code != http.StatusTeapot {
		t.Fatalf("status = %d, want 418", rec.Code)
	}
	if obj := replyObject(t, rec.Body.Bytes()); obj["status"] != float64(http.StatusTeapot) || obj["ok"] != false {
		t.Fatalf("418 body = %v, want the generic error envelope carrying 418", obj)
	}
}
