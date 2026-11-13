package outlinestub_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/tests/support/outlinestub"
)

// post sends one Outline-style request through the in-process transport and returns the status and body.
func post(t *testing.T, s *outlinestub.Stub, method, body string) (int, []byte, *http.Response) {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost,
		"https://outline-stub.invalid/api/"+method, strings.NewReader(body))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Authorization", "Bearer stub-test-key")
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.RoundTripper().RoundTrip(req)
	if err != nil {
		return 0, nil, nil
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return resp.StatusCode, raw, resp
}

func decodeObject(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		t.Fatalf("body is not JSON: %v: %s", err, raw)
	}
	return obj
}

func TestCollectionsPageReplaysTheFixtureShapeAndPagination(t *testing.T) {
	s := outlinestub.New(outlinestub.WithCollections(outlinestub.NumberedCollections(3)...))

	status, raw, _ := post(t, s, "collections.list", `{"offset":0,"limit":25}`)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	obj := decodeObject(t, raw)
	data, ok := obj["data"].([]any)
	if !ok || len(data) != 3 {
		t.Fatalf("data = %v, want 3 collections", obj["data"])
	}
	first := data[0].(map[string]any)
	if first["id"] != "col-001" || first["name"] != "Collection 001" {
		t.Fatalf("first collection = %v, want col-001", first)
	}
	if _, ok := first["sort"]; !ok {
		t.Fatalf("collection lacks the fixture field sort: %v", first)
	}
	page := obj["pagination"].(map[string]any)
	if page["offset"] != float64(0) || page["limit"] != float64(25) {
		t.Fatalf("pagination = %v, want offset 0 and limit 25", page)
	}
}

func TestFullPagesAnswerAFullPageWhateverTheOffset(t *testing.T) {
	s := outlinestub.New(outlinestub.WithFullPages())

	_, raw, _ := post(t, s, "collections.list", `{"offset":500,"limit":25}`)
	data := decodeObject(t, raw)["data"].([]any)
	if len(data) != 25 {
		t.Fatalf("full page holds %d items, want 25", len(data))
	}
}

func TestTreeNestsDocumentsByParentInSliceOrder(t *testing.T) {
	s := outlinestub.New(
		outlinestub.WithCollections(outlinestub.Collection{ID: "col-1", Name: "One"}),
		outlinestub.WithDocuments(
			outlinestub.Document{ID: "doc-1", CollectionID: "col-1", Title: "Root", URL: "/doc/root"},
			outlinestub.Document{ID: "doc-2", CollectionID: "col-1", ParentID: "doc-1", Title: "Child", URL: "/doc/child"},
			outlinestub.Document{ID: "doc-9", CollectionID: "col-other", Title: "Elsewhere", URL: "/doc/else"},
		),
	)

	status, raw, _ := post(t, s, "collections.documents", `{"id":"col-1"}`)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	roots := decodeObject(t, raw)["data"].([]any)
	if len(roots) != 1 {
		t.Fatalf("roots = %d, want 1 (documents of other collections are excluded)", len(roots))
	}
	root := roots[0].(map[string]any)
	children := root["children"].([]any)
	if len(children) != 1 || children[0].(map[string]any)["id"] != "doc-2" {
		t.Fatalf("children of doc-1 = %v, want [doc-2]", children)
	}
}

func TestUnknownCollectionAndDocumentAreNotFound(t *testing.T) {
	s := outlinestub.New()
	for _, call := range []struct{ method, body string }{
		{"collections.documents", `{"id":"col-missing"}`},
		{"documents.info", `{"id":"doc-missing"}`},
		{"documents.export", `{"id":"doc-missing"}`},
	} {
		status, raw, _ := post(t, s, call.method, call.body)
		if status != http.StatusNotFound {
			t.Fatalf("%s status = %d, want 404", call.method, status)
		}
		if obj := decodeObject(t, raw); obj["error"] != "not_found" {
			t.Fatalf("%s error body = %v, want the sanitized not_found envelope", call.method, obj)
		}
	}
}

func TestDocumentInfoAndExportReplayTheDocument(t *testing.T) {
	s := outlinestub.New(outlinestub.WithDocuments(outlinestub.Document{
		ID: "doc-1", CollectionID: "col-1", Title: "Getting Started",
		URL: "/doc/getting-started", Markdown: "# Getting Started\n",
	}))

	_, raw, _ := post(t, s, "documents.info", `{"id":"doc-1"}`)
	info := decodeObject(t, raw)["data"].(map[string]any)
	if info["id"] != "doc-1" || info["title"] != "Getting Started" || info["collectionId"] != "col-1" {
		t.Fatalf("documents.info data = %v", info)
	}
	if info["parentDocumentId"] != nil {
		t.Fatalf("parentDocumentId = %v, want null for a root", info["parentDocumentId"])
	}
	_, raw, _ = post(t, s, "documents.export", `{"id":"doc-1"}`)
	if got := decodeObject(t, raw)["data"]; got != "# Getting Started\n" {
		t.Fatalf("documents.export data = %v, want the Markdown", got)
	}
}

func TestFaultFirstNAppliesOnlyToTheFirstRequests(t *testing.T) {
	s := outlinestub.New(outlinestub.WithCollections(outlinestub.NumberedCollections(1)...))
	s.SetFaultFirst(1, outlinestub.Fault{Status: 429, RetryAfter: "1"})

	first, _, resp := post(t, s, "collections.list", `{"offset":0,"limit":25}`)
	if first != http.StatusTooManyRequests || resp.Header.Get("Retry-After") != "1" {
		t.Fatalf("first request = %d with Retry-After %q, want 429 with 1", first, resp.Header.Get("Retry-After"))
	}
	second, _, _ := post(t, s, "collections.list", `{"offset":0,"limit":25}`)
	if second != http.StatusOK {
		t.Fatalf("second request = %d, want 200 once the fault is spent", second)
	}
}

func TestFaultForOneMethodLeavesTheOthersAlone(t *testing.T) {
	s := outlinestub.New(
		outlinestub.WithCollections(outlinestub.NumberedCollections(1)...),
		outlinestub.WithDocuments(outlinestub.Document{ID: "doc-1", CollectionID: "col-001", Title: "T", URL: "/t", Markdown: "m"}),
	)
	s.SetFault(outlinestub.Fault{Method: "documents.export", Status: 500})

	if status, _, _ := post(t, s, "collections.list", `{"offset":0,"limit":25}`); status != http.StatusOK {
		t.Fatalf("collections.list = %d, want 200", status)
	}
	if status, _, _ := post(t, s, "documents.export", `{"id":"doc-1"}`); status != http.StatusInternalServerError {
		t.Fatalf("documents.export = %d, want 500", status)
	}
}

func TestEchoAuthorizationRepeatsTheHeaderInANonJSONBody(t *testing.T) {
	s := outlinestub.New()
	s.SetFault(outlinestub.Fault{Status: 200, EchoAuthorization: true})

	_, raw, _ := post(t, s, "collections.list", `{"offset":0,"limit":25}`)
	if !strings.Contains(string(raw), "Bearer stub-test-key") {
		t.Fatalf("echo body = %q, want the Authorization header", raw)
	}
	var v any
	if json.Unmarshal(raw, &v) == nil {
		t.Fatalf("echo body parsed as JSON, want non-JSON text")
	}
}

func TestRequestsAreRecordedWithMethodHostAuthorizationAndTime(t *testing.T) {
	clock := time.Date(2026, 11, 3, 9, 0, 0, 0, time.UTC)
	s := outlinestub.New(
		outlinestub.WithClock(func() time.Time { return clock }),
		outlinestub.WithCollections(outlinestub.NumberedCollections(1)...),
	)

	post(t, s, "collections.list", `{"offset":0,"limit":25}`)
	reqs := s.Requests()
	if len(reqs) != 1 {
		t.Fatalf("recorded %d requests, want 1", len(reqs))
	}
	got := reqs[0]
	if got.Method != "collections.list" || got.Host != "outline-stub.invalid" ||
		got.Authorization != "Bearer stub-test-key" || got.Limit != 25 || got.Status != http.StatusOK ||
		!got.At.Equal(clock) {
		t.Fatalf("recorded request = %+v", got)
	}
	if s.Calls("collections.list") != 1 || s.PeakInFlight() != 1 {
		t.Fatalf("calls or peak in flight wrong: calls=%d peak=%d", s.Calls("collections.list"), s.PeakInFlight())
	}
}

func TestOnlyPOSTIsAccepted(t *testing.T) {
	s := outlinestub.New()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet,
		"https://outline-stub.invalid/api/collections.list", bytes.NewReader(nil))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp, err := s.RoundTripper().RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("GET status = %d, want 405", resp.StatusCode)
	}
}

func TestHangAndDropAreTransportFailuresInProcess(t *testing.T) {
	s := outlinestub.New()
	s.SetFault(outlinestub.Fault{Hang: true})
	if _, _, resp := post(t, s, "collections.list", `{"offset":0,"limit":25}`); resp != nil {
		t.Fatalf("a hang returned a response, want a transport error")
	}
	s.SetFault(outlinestub.Fault{Drop: true})
	if _, _, resp := post(t, s, "collections.list", `{"offset":0,"limit":25}`); resp != nil {
		t.Fatalf("a dropped connection returned a response, want a transport error")
	}
}

func TestRequestsRecordTheHTTPMethodAndTheSecureFlag(t *testing.T) {
	s := outlinestub.New(outlinestub.WithCollections(outlinestub.NumberedCollections(1)...))

	post(t, s, "collections.list", `{"offset":0,"limit":25}`)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet,
		"http://outline-stub.invalid/api/collections.list", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if resp, err := s.RoundTripper().RoundTrip(req); err == nil {
		resp.Body.Close()
	}

	reqs := s.Requests()
	if len(reqs) != 2 {
		t.Fatalf("recorded %d requests, want 2", len(reqs))
	}
	if got := reqs[0]; got.HTTPMethod != http.MethodPost || !got.Secure {
		t.Fatalf("https POST recorded as %q secure=%v, want POST with secure=true", got.HTTPMethod, got.Secure)
	}
	if got := reqs[1]; got.HTTPMethod != http.MethodGet || got.Secure {
		t.Fatalf("http GET recorded as %q secure=%v, want GET with secure=false", got.HTTPMethod, got.Secure)
	}
}

// TestDropFailsEveryOutlineMethodInProcess checks the no-network fault: a dropped connection
// fails every Outline method, and each attempt is still recorded.
func TestDropFailsEveryOutlineMethodInProcess(t *testing.T) {
	s := outlinestub.New(outlinestub.WithCollections(outlinestub.NumberedCollections(1)...))
	s.SetFault(outlinestub.Fault{Drop: true})
	calls := []struct{ method, body string }{
		{"collections.list", `{"offset":0,"limit":25}`},
		{"collections.documents", `{"id":"col-001"}`},
		{"documents.info", `{"id":"doc-1"}`},
		{"documents.export", `{"id":"doc-1"}`},
	}
	for _, call := range calls {
		if _, _, resp := post(t, s, call.method, call.body); resp != nil {
			t.Fatalf("%s got a response, want a dropped connection", call.method)
		}
	}
	if n := len(s.Requests()); n != len(calls) {
		t.Fatalf("recorded %d requests, want %d", n, len(calls))
	}
}

func TestSetDatasetReplacesTheDataAtRuntime(t *testing.T) {
	s := outlinestub.New(outlinestub.WithCollections(outlinestub.NumberedCollections(3)...))
	if n := listedCount(t, s, 0); n != 3 {
		t.Fatalf("before SetDataset the stub lists %d collections, want 3", n)
	}

	s.SetDataset(outlinestub.NumberedCollections(1), []outlinestub.Document{
		{ID: "doc-9", CollectionID: "col-001", Title: "Nine", URL: "/doc/nine", Markdown: "# Nine\n"},
	})
	if n := listedCount(t, s, 0); n != 1 {
		t.Fatalf("after SetDataset the stub lists %d collections, want 1", n)
	}
	_, raw, _ := post(t, s, "documents.info", `{"id":"doc-9"}`)
	if data := dataObject(t, raw); data["title"] != "Nine" {
		t.Fatalf("documents.info after SetDataset = %v, want the new document", data)
	}
}

func TestSetDatasetKeepsItsOwnCopyOfTheSlices(t *testing.T) {
	s := outlinestub.New()
	docs := []outlinestub.Document{{ID: "doc-1", CollectionID: "col-001", Title: "Original", URL: "/doc/1", Markdown: "m"}}
	s.SetDataset(nil, docs)
	docs[0].Title = "Changed"

	_, raw, _ := post(t, s, "documents.info", `{"id":"doc-1"}`)
	if data := dataObject(t, raw); data["title"] != "Original" {
		t.Fatalf("title = %v after the caller changed its slice, want Original", data["title"])
	}
}

// TestSetDatasetIsSafeWhileRequestsRun changes the data while other goroutines send requests.
// Each reply holds one or two collections, and a read after the last change sees the last dataset.
func TestSetDatasetIsSafeWhileRequestsRun(t *testing.T) {
	s := outlinestub.New(outlinestub.WithCollections(outlinestub.NumberedCollections(1)...))
	var wg sync.WaitGroup
	problems := make(chan error, 4*50)
	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				n, err := listPage(s, 0)
				if err != nil {
					problems <- err
					continue
				}
				if n != 1 && n != 2 {
					problems <- fmt.Errorf("a reply listed %d collections, want 1 or 2", n)
				}
			}
		}()
	}
	for i := 0; i < 50; i++ {
		s.SetDataset(outlinestub.NumberedCollections(1+i%2), nil)
	}
	wg.Wait()
	close(problems)
	for err := range problems {
		t.Error(err)
	}
	if n := listedCount(t, s, 0); n != 2 {
		t.Fatalf("the final listing has %d collections, want 2 from the last dataset", n)
	}
}

func TestSetFullPagesTurnsTheFullPageAnswerOnAndOff(t *testing.T) {
	s := outlinestub.New(outlinestub.WithCollections(outlinestub.NumberedCollections(3)...))

	s.SetFullPages(true)
	if n := listedCount(t, s, 500); n != 25 {
		t.Fatalf("with full pages on, offset 500 lists %d, want 25", n)
	}
	s.SetFullPages(false)
	if n := listedCount(t, s, 500); n != 0 {
		t.Fatalf("with full pages off, offset 500 lists %d, want 0 past the data", n)
	}
}

// listPage sends collections.list for one page and returns how many collections the reply holds.
// It never fails a test, so goroutines may call it.
func listPage(s *outlinestub.Stub, offset int) (int, error) {
	body := fmt.Sprintf(`{"offset":%d,"limit":25}`, offset)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost,
		"https://outline-stub.invalid/api/collections.list", strings.NewReader(body))
	if err != nil {
		return 0, err
	}
	resp, err := s.RoundTripper().RoundTrip(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	var obj struct {
		Data []json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&obj); err != nil {
		return 0, err
	}
	return len(obj.Data), nil
}

// listedCount is listPage for the test goroutine, which may fail the test.
func listedCount(t *testing.T, s *outlinestub.Stub, offset int) int {
	t.Helper()
	n, err := listPage(s, offset)
	if err != nil {
		t.Fatalf("collections.list: %v", err)
	}
	return n
}

// dataObject returns the data object of a reply, and fails the test when it has none.
func dataObject(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	data, ok := decodeObject(t, raw)["data"].(map[string]any)
	if !ok {
		t.Fatalf("reply has no data object: %s", raw)
	}
	return data
}
