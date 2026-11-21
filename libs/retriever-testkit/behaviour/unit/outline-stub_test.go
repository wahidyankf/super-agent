package unit_test

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-testkit/outlinestub"
)

const outlineBase = "https://outline.invalid/api/"

// outlineCall makes one call of an Outline method through the in-process transport of stub. A nil body sends no body, and
// version is the X-API-Version header when it is set.
func outlineCall(t *testing.T, stub *outlinestub.Stub, verb, method, version string, body any) (*http.Response, []byte) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("encode the body: %v", err)
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(context.Background(), verb, outlineBase+method, reader)
	if err != nil {
		t.Fatalf("build the request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if version != "" {
		req.Header.Set("X-API-Version", version)
	}
	client := &http.Client{Transport: stub.RoundTripper()}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("call %s %s: %v", verb, method, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read the reply: %v", err)
	}
	return resp, data
}

// newOutlineStub returns an in-process stub with the fixed clock and the given data.
func newOutlineStub(opts ...outlinestub.Option) *outlinestub.Stub {
	all := append([]outlinestub.Option{outlinestub.WithClock(fixedClock)}, opts...)
	return outlinestub.New(all...)
}

// decodeOutline decodes a reply into a generic object, failing the test when it is not JSON.
func decodeOutline(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var obj map[string]any
	if err := json.Unmarshal(data, &obj); err != nil {
		t.Fatalf("the reply is not JSON: %v: %s", err, data)
	}
	return obj
}

// @scenario specs/libs/retriever-testkit/behaviours/outline-stub.feature: The collections list answers one page and names the next page
func TestCollectionsListAnswersOnePageAndEchoesPagination(t *testing.T) {
	stub := newOutlineStub(outlinestub.WithCollections(outlinestub.NumberedCollections(3)...))
	resp, data := outlineCall(t, stub, http.MethodPost, "collections.list", "", map[string]any{"offset": 0, "limit": 2})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	obj := decodeOutline(t, data)
	items, _ := obj["data"].([]any)
	pagination, _ := obj["pagination"].(map[string]any)
	if len(items) != 2 || pagination["offset"] != float64(0) || pagination["limit"] != float64(2) {
		t.Fatalf("the page holds %d items and pagination %v", len(items), pagination)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/outline-stub.feature: The collections info method answers one collection or not found
func TestCollectionsInfoAnswersOneCollectionOrNotFound(t *testing.T) {
	cases := []struct {
		id     string
		status int
	}{
		{"c1", http.StatusOK},
		{"c9", http.StatusNotFound},
	}
	for _, c := range cases {
		t.Run(c.id, func(t *testing.T) {
			stub := newOutlineStub(outlinestub.WithCollections(outlinestub.Collection{ID: "c1", Name: "Eng"}))
			resp, _ := outlineCall(t, stub, http.MethodPost, "collections.info", "", map[string]any{"id": c.id})
			if resp.StatusCode != c.status {
				t.Fatalf("status = %d, want %d", resp.StatusCode, c.status)
			}
		})
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/outline-stub.feature: The collections documents method nests each child under its parent
func TestCollectionsDocumentsNestsChildUnderParent(t *testing.T) {
	stub := newOutlineStub(
		outlinestub.WithCollections(outlinestub.Collection{ID: "c1", Name: "Eng"}),
		outlinestub.WithDocuments(
			outlinestub.Document{ID: "doc-1", CollectionID: "c1", Title: "Parent", URL: "/doc/p", Markdown: "# P"},
			outlinestub.Document{ID: "doc-2", CollectionID: "c1", ParentID: "doc-1", Title: "Child", URL: "/doc/c", Markdown: "# C"},
		),
	)
	_, data := outlineCall(t, stub, http.MethodPost, "collections.documents", "", map[string]any{"id": "c1"})
	roots, _ := decodeOutline(t, data)["data"].([]any)
	if len(roots) != 1 {
		t.Fatalf("the tree holds %d roots, want the one parent", len(roots))
	}
	root, _ := roots[0].(map[string]any)
	children, _ := root["children"].([]any)
	if len(children) != 1 {
		t.Fatalf("the parent holds %d children, want the child", len(children))
	}
	child, _ := children[0].(map[string]any)
	if child["id"] != "doc-2" {
		t.Fatalf("the child is %v, want doc-2", child["id"])
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/outline-stub.feature: The documents info method answers the shape the header asks for
func TestDocumentsInfoAnswersShapeTheHeaderAsks(t *testing.T) {
	stub := newOutlineStub(outlinestub.WithDocuments(outlinestub.Document{ID: "doc-1", CollectionID: "c1", Title: "Notes", Markdown: "# Notes"}))
	body := map[string]any{"id": "doc-1"}

	_, plain := outlineCall(t, stub, http.MethodPost, "documents.info", "", body)
	if decodeOutline(t, plain)["data"].(map[string]any)["text"] != "# Notes" {
		t.Fatalf("the unversioned reply does not carry the text: %s", plain)
	}
	_, v2 := outlineCall(t, stub, http.MethodPost, "documents.info", "2", body)
	if decodeOutline(t, v2)["data"].(map[string]any)["document"].(map[string]any)["text"] != "# Notes" {
		t.Fatalf("the version 2 reply does not wrap the document with its text: %s", v2)
	}
	_, v3 := outlineCall(t, stub, http.MethodPost, "documents.info", "3", body)
	document := decodeOutline(t, v3)["data"].(map[string]any)["document"].(map[string]any)
	if _, ok := document["text"]; ok {
		t.Fatalf("the version 3 reply carries the text: %s", v3)
	}
	if document["data"].(map[string]any)["type"] != "doc" {
		t.Fatalf("the version 3 reply holds no editor data: %s", v3)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/outline-stub.feature: The documents export method answers one document's Markdown or 404
func TestDocumentsExportAnswersOneDocumentsMarkdownOr404(t *testing.T) {
	cases := []struct {
		id     string
		status int
	}{
		{"doc-1", http.StatusOK},
		{"doc-9", http.StatusNotFound},
	}
	for _, c := range cases {
		t.Run(c.id, func(t *testing.T) {
			stub := newOutlineStub(outlinestub.WithDocuments(outlinestub.Document{ID: "doc-1", CollectionID: "c1", Markdown: "# Notes"}))
			resp, data := outlineCall(t, stub, http.MethodPost, "documents.export", "", map[string]any{"id": c.id})
			if resp.StatusCode != c.status {
				t.Fatalf("status = %d, want %d", resp.StatusCode, c.status)
			}
			if c.status == http.StatusOK && decodeOutline(t, data)["data"] != "# Notes" {
				t.Fatalf("the export is %s, want the Markdown", data)
			}
		})
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/outline-stub.feature: The documents list method answers one page and names the next
func TestDocumentsListAnswersOnePageAndNamesNext(t *testing.T) {
	stub := newOutlineStub(outlinestub.WithDocuments(
		outlinestub.Document{ID: "doc-1", CollectionID: "c1", Title: "One", Markdown: "1"},
		outlinestub.Document{ID: "doc-2", CollectionID: "c1", Title: "Two", Markdown: "2"},
		outlinestub.Document{ID: "doc-3", CollectionID: "c1", Title: "Three", Markdown: "3"},
	))
	_, data := outlineCall(t, stub, http.MethodPost, "documents.list", "", map[string]any{"offset": 0, "limit": 2})
	obj := decodeOutline(t, data)
	items, _ := obj["data"].([]any)
	pagination, _ := obj["pagination"].(map[string]any)
	if len(items) != 2 {
		t.Fatalf("the page holds %d documents, want 2", len(items))
	}
	if next, _ := pagination["nextPath"].(string); next == "" {
		t.Fatal("the page does not name the next page")
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/outline-stub.feature: The documents search method finds a term in a title or body
func TestDocumentsSearchFindsTermInTitleOrBody(t *testing.T) {
	cases := []string{"install", "INSTALL"}
	for _, term := range cases {
		t.Run(term, func(t *testing.T) {
			stub := newOutlineStub(outlinestub.WithDocuments(
				outlinestub.Document{ID: "doc-1", CollectionID: "c1", Title: "Install Guide", Markdown: "Install"},
				outlinestub.Document{ID: "doc-2", CollectionID: "c1", Title: "Notes", Markdown: "Install"},
			))
			_, data := outlineCall(t, stub, http.MethodPost, "documents.search", "", map[string]any{"query": term, "offset": 0, "limit": 25})
			items, _ := decodeOutline(t, data)["data"].([]any)
			if len(items) != 2 {
				t.Fatalf("the search holds %d documents, want the 2 that hold the term", len(items))
			}
		})
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/outline-stub.feature: The documents search titles method ignores words found only in bodies
func TestDocumentsSearchTitlesIgnoresWordsFoundOnlyInBodies(t *testing.T) {
	stub := newOutlineStub(outlinestub.WithDocuments(outlinestub.Document{ID: "doc-1", CollectionID: "c1", Title: "Notes", Markdown: "Install"}))
	_, data := outlineCall(t, stub, http.MethodPost, "documents.search_titles", "", map[string]any{"query": "install", "offset": 0, "limit": 25})
	items, _ := decodeOutline(t, data)["data"].([]any)
	if len(items) != 0 {
		t.Fatalf("the title search answered %d documents, want none", len(items))
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/outline-stub.feature: Only a POST reaches a replay and any other verb is refused
func TestOnlyPOSTReachesReplayAndOtherVerbsAreRefused(t *testing.T) {
	cases := []string{http.MethodGet, http.MethodPut}
	for _, verb := range cases {
		t.Run(verb, func(t *testing.T) {
			stub := newOutlineStub(outlinestub.WithCollections(outlinestub.NumberedCollections(1)...))
			resp, _ := outlineCall(t, stub, verb, "collections.list", "", nil)
			if resp.StatusCode != http.StatusMethodNotAllowed {
				t.Fatalf("status = %d, want 405", resp.StatusCode)
			}
		})
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/outline-stub.feature: A method the stub does not offer gets the endpoint not found message
func TestMethodStubDoesNotOfferGetsEndpointNotFound(t *testing.T) {
	stub := newOutlineStub()
	resp, data := outlineCall(t, stub, http.MethodPost, "users.list", "", map[string]any{})
	if resp.StatusCode != http.StatusNotFound || decodeOutline(t, data)["message"] != "Endpoint not found" {
		t.Fatalf("status %d and body %s; want 404 Endpoint not found", resp.StatusCode, data)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/outline-stub.feature: A set reply answers with its status and body
func TestSetReplyAnswersWithItsStatusAndBody(t *testing.T) {
	stub := newOutlineStub()
	stub.SetReply("collections.info", http.StatusInternalServerError, `{"ok":false}`)
	resp, data := outlineCall(t, stub, http.MethodPost, "collections.info", "", map[string]any{"id": "anything"})
	if resp.StatusCode != http.StatusInternalServerError || string(data) != `{"ok":false}` {
		t.Fatalf("status %d and body %s; want 500 with the set body", resp.StatusCode, data)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/outline-stub.feature: A fault for one method answers only calls of that method
func TestFaultForOneMethodAnswersOnlyCallsOfThatMethod(t *testing.T) {
	stub := newOutlineStub(outlinestub.WithCollections(outlinestub.NumberedCollections(1)...))
	stub.SetFault(outlinestub.Fault{Method: "documents.info", Status: http.StatusInternalServerError})
	faulted, _ := outlineCall(t, stub, http.MethodPost, "documents.info", "", map[string]any{"id": "doc-1"})
	other, _ := outlineCall(t, stub, http.MethodPost, "collections.list", "", map[string]any{"offset": 0, "limit": 1})
	if faulted.StatusCode != http.StatusInternalServerError {
		t.Fatalf("the named method answered %d, want 500", faulted.StatusCode)
	}
	if other.StatusCode != http.StatusOK {
		t.Fatalf("another method answered %d, want 200", other.StatusCode)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/outline-stub.feature: The reply key check refuses a key that the router does not send
func TestReplyKeyCheckRefusesKeyRouterDoesNotSend(t *testing.T) {
	stub := newOutlineStub(outlinestub.WithCollections(outlinestub.NumberedCollections(1)...))
	_, data := outlineCall(t, stub, http.MethodPost, "collections.list", "", map[string]any{"offset": 0, "limit": 1})
	obj := decodeOutline(t, data)
	obj["fabricatedKey"] = true
	reply, err := json.Marshal(obj)
	if err != nil {
		t.Fatalf("encode the reply: %v", err)
	}
	if err := outlinestub.CheckReplyKeys("collections.list", reply); err == nil {
		t.Fatal("the check accepted a key the router does not send")
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/outline-stub.feature: The reply key check accepts the keys that the presenters send
func TestReplyKeyCheckAcceptsKeysThatPresentersSend(t *testing.T) {
	stub := newOutlineStub(outlinestub.WithCollections(outlinestub.NumberedCollections(1)...))
	_, data := outlineCall(t, stub, http.MethodPost, "collections.list", "", map[string]any{"offset": 0, "limit": 1})
	if err := outlinestub.CheckReplyKeys("collections.list", data); err != nil {
		t.Fatalf("the check refused a reply with the presenters' keys: %v", err)
	}
}

// caWrite is one call of the recording CA writer.
type caWrite struct {
	name string
	data []byte
	perm os.FileMode
}

// @scenario specs/libs/retriever-testkit/behaviours/outline-stub.feature: The CA file of the stub is written with owner-only access
func TestCAFileOfStubIsWrittenWithOwnerOnlyAccess(t *testing.T) {
	var written []caWrite
	record := func(name string, data []byte, perm os.FileMode) error {
		written = append(written, caWrite{name: name, data: append([]byte(nil), data...), perm: perm})
		return nil
	}
	stub := outlinestub.Start(t,
		outlinestub.WithListener(newFakeListener().listen),
		outlinestub.WithCertificateRandom(&fixedRandom{}),
		outlinestub.WithCertificateClock(fixedClock),
		outlinestub.WithCAWriter(record),
	)
	dir := filepath.Join("unit", "ca")
	path, err := stub.WriteCAFile(dir)
	if err != nil {
		t.Fatalf("write the CA file: %v", err)
	}
	if len(written) != 1 || written[0].name != path {
		t.Fatalf("the writer was called %d times with %v", len(written), written)
	}
	if written[0].perm != 0o600 {
		t.Fatalf("the file mode is %o, want owner only 600", written[0].perm)
	}
	block, _ := pem.Decode(written[0].data)
	if block == nil || block.Type != "CERTIFICATE" {
		t.Fatal("the file does not hold a PEM certificate")
	}
}
