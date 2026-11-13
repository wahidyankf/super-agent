package outlinestub

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
)

// fixtureFS holds the sanitized response fixtures. Each file is one response
// shape; the replay copies its fields and fills in the scripted data.
//
//go:embed testdata/*.json
var fixtureFS embed.FS

// fixtures are decoded once at start-up. A missing or broken fixture is a build
// defect in this package, so the program stops rather than replaying nothing.
var fixtures = loadFixtures()

// reply is one answer the stub sends. hang and drop are transport actions, not responses.
type reply struct {
	status int
	header http.Header
	body   []byte
	hang   bool
	drop   bool
}

// snapshot is the state one request is answered from. It is copied under the lock,
// so the replay runs without it.
type snapshot struct {
	collections   []Collection
	documents     []Document
	fullPages     bool
	fault         Fault
	unoffered     map[string]bool
	replies       map[string]reply
	ignoreVersion bool
	editorOnly    bool
	account       Account
}

// call is one request as the replay reads it: its verb, its Outline method, its body, and the API version it asked for.
type call struct {
	httpMethod string
	method     string
	in         requestBody
	bodyErr    error
	auth       string
	version    int
}

// respond answers one request from the snapshot. It returns a fault first, then the
// method, body, and HTTP checks, then the replay.
func (sn snapshot) respond(c call) reply {
	if sn.fault != (Fault{}) && (sn.fault.Method == "" || sn.fault.Method == c.method) {
		if rep, ok := faultReply(sn.fault, c.auth); ok {
			return rep
		}
	}
	if c.httpMethod != http.MethodPost {
		return errorReply(http.StatusMethodNotAllowed)
	}
	if c.bodyErr != nil {
		return errorReply(http.StatusBadRequest)
	}
	if sn.unoffered[c.method] {
		return endpointNotFound()
	}
	if rep, ok := sn.replies[c.method]; ok {
		return rep
	}
	switch c.method {
	case "auth.info":
		return sn.authInfo()
	case "collections.list":
		return sn.listCollections(c.in)
	case "collections.info":
		return sn.collectionInfo(c.in.ID)
	case "collections.documents":
		return sn.collectionTree(c.in.ID)
	case "documents.info":
		return sn.documentInfo(c.in.ID, c.version)
	case "documents.export":
		return sn.documentExport(c.in.ID)
	case "documents.list":
		return sn.listDocuments(c.in)
	case "documents.search":
		return sn.searchDocuments(c.in)
	}
	return endpointNotFound()
}

// faultReply returns the scripted answer, and false when the fault leaves the normal replay in place.
func faultReply(f Fault, auth string) (reply, bool) {
	if f.Hang {
		return reply{hang: true}, true
	}
	if f.Drop {
		return reply{drop: true}, true
	}
	if f.Status == 0 && f.Body == "" && !f.EchoAuthorization {
		return reply{}, false
	}
	status := f.Status
	if status == 0 {
		status = http.StatusOK
	}
	header := http.Header{}
	if f.RetryAfter != "" {
		header.Set("Retry-After", f.RetryAfter)
	}
	if f.Location != "" {
		header.Set("Location", f.Location)
	}
	switch {
	case f.EchoAuthorization:
		header.Set("Content-Type", "text/plain")
		return reply{status: status, header: header, body: []byte("upstream said: " + auth)}, true
	case f.Body != "":
		header.Set("Content-Type", "text/plain")
		return reply{status: status, header: header, body: []byte(f.Body)}, true
	case status >= http.StatusBadRequest:
		return jsonReply(status, errorBody(status), header), true
	}
	return reply{status: status, header: header}, true
}

// authInfo answers auth.info with the signed-in user and team. The user carries an e-mail address, which a client
// must never print.
func (sn snapshot) authInfo() reply {
	body := map[string]any{"data": map[string]any{
		"user": map[string]any{
			"id":    "fixture-user-0001",
			"name":  sn.account.User,
			"role":  "admin",
			"email": "fixture@example.com",
		},
		"team": map[string]any{"id": "fixture-team-0001", "name": sn.account.Team},
	}}
	return jsonReply(http.StatusOK, body, nil)
}

// listCollections answers collections.list from the dataset, or from full pages when the stub is set so.
func (sn snapshot) listCollections(in requestBody) reply {
	if in.Limit <= 0 || in.Offset < 0 {
		return errorReply(http.StatusBadRequest)
	}
	template := fixtureObject("collections.list", "data", 0)
	items := []any{}
	if sn.fullPages {
		for i := 0; i < in.Limit; i++ {
			id := fmt.Sprintf("full-%d", in.Offset+i)
			items = append(items, renderCollection(template, Collection{ID: id, Name: "Full " + id}))
		}
	} else {
		start, end := window(len(sn.collections), in.Offset, in.Limit)
		for _, c := range sn.collections[start:end] {
			items = append(items, renderCollection(template, c))
		}
	}
	body := map[string]any{
		"data": items,
		"pagination": map[string]any{
			"offset": in.Offset,
			"limit":  in.Limit,
		},
	}
	return jsonReply(http.StatusOK, body, nil)
}

// collectionInfo answers collections.info with one collection object, or 404 for an unknown collection.
func (sn snapshot) collectionInfo(id string) reply {
	if !sn.hasCollection(id) {
		return errorReply(http.StatusNotFound)
	}
	for _, c := range sn.collections {
		if c.ID == id {
			obj := map[string]any{
				"id":          c.ID,
				"name":        c.Name,
				"url":         "/collection/" + c.ID,
				"description": "Fake description.",
			}
			return jsonReply(http.StatusOK, map[string]any{"data": obj}, nil)
		}
	}
	return errorReply(http.StatusNotFound)
}

// collectionTree answers collections.documents with the nested navigation tree of one collection.
func (sn snapshot) collectionTree(collectionID string) reply {
	if !sn.hasCollection(collectionID) {
		return errorReply(http.StatusNotFound)
	}
	template := fixtureObject("collections.documents", "data", 0)
	children := map[string][]Document{}
	var roots []Document
	for _, d := range sn.documents {
		if d.CollectionID != collectionID {
			continue
		}
		if d.ParentID == "" {
			roots = append(roots, d)
			continue
		}
		children[d.ParentID] = append(children[d.ParentID], d)
	}
	var build func(docs []Document) []any
	build = func(docs []Document) []any {
		out := make([]any, 0, len(docs))
		for _, d := range docs {
			node := cloneObject(template)
			node["id"] = d.ID
			node["title"] = d.Title
			node["url"] = d.URL
			node["children"] = build(children[d.ID])
			out = append(out, node)
		}
		return out
	}
	return jsonReply(http.StatusOK, map[string]any{"data": build(roots)}, nil)
}

// documentInfo answers documents.info in the shape the X-API-Version header asks for (ADR 007, section 6.1):
// below 2 the document is the data itself with its Markdown text; 2 wraps it in data.document with the text; 3 or
// higher wraps it and sends editor data with no text. A server that ignores the header answers the unwrapped shape.
// editorOnly forces the version-3 shape.
func (sn snapshot) documentInfo(id string, version int) reply {
	d, ok := sn.findDocument(id)
	if !ok {
		return errorReply(http.StatusNotFound)
	}
	obj := documentObject(d)
	switch {
	case sn.editorOnly, !sn.ignoreVersion && version >= 3:
		delete(obj, "text")
		obj["data"] = map[string]any{"type": "doc", "content": []any{}}
		return jsonReply(http.StatusOK, map[string]any{"data": map[string]any{"document": obj}}, nil)
	case !sn.ignoreVersion && version == 2:
		return jsonReply(http.StatusOK, map[string]any{"data": map[string]any{"document": obj}}, nil)
	}
	return jsonReply(http.StatusOK, map[string]any{"data": obj}, nil)
}

// documentObject returns one document in the shape of the document presenter, with its Markdown text.
func documentObject(d Document) map[string]any {
	obj := map[string]any{
		"id":           d.ID,
		"urlId":        d.ID,
		"url":          d.URL,
		"title":        d.Title,
		"text":         d.Markdown,
		"collectionId": d.CollectionID,
		// The stub keeps no revision history, so every document is at revision 1. A write names it as last_revision.
		"revision": 1,
	}
	if d.ParentID == "" {
		obj["parentDocumentId"] = nil
	} else {
		obj["parentDocumentId"] = d.ParentID
	}
	return obj
}

// documentExport answers documents.export with the Markdown string of one document.
func (sn snapshot) documentExport(id string) reply {
	d, ok := sn.findDocument(id)
	if !ok {
		return errorReply(http.StatusNotFound)
	}
	return jsonReply(http.StatusOK, map[string]any{"data": d.Markdown}, nil)
}

// listDocuments answers documents.list with one page of the documents, optionally of one collection. The reply names
// the next page path, as the router does, and the client must not follow it.
func (sn snapshot) listDocuments(in requestBody) reply {
	if in.Limit < 1 || in.Limit > 100 || in.Offset < 0 {
		return errorReply(http.StatusBadRequest)
	}
	docs := sn.documentsOf(in.CollectionID)
	start, end := window(len(docs), in.Offset, in.Limit)
	items := make([]any, 0, end-start)
	for _, d := range docs[start:end] {
		items = append(items, documentObject(d))
	}
	return jsonReply(http.StatusOK, pagedBody(items, in), nil)
}

// searchDocuments answers documents.search with one page of the documents whose title or text holds the query, in
// the router's shape: each result is a document with a context and a ranking.
func (sn snapshot) searchDocuments(in requestBody) reply {
	if in.Limit < 1 || in.Limit > 100 || in.Offset < 0 || in.Query == "" {
		return errorReply(http.StatusBadRequest)
	}
	query := strings.ToLower(in.Query)
	var hits []Document
	for _, d := range sn.documentsOf(in.CollectionID) {
		if strings.Contains(strings.ToLower(d.Title), query) || strings.Contains(strings.ToLower(d.Markdown), query) {
			hits = append(hits, d)
		}
	}
	start, end := window(len(hits), in.Offset, in.Limit)
	items := make([]any, 0, end-start)
	for _, d := range hits[start:end] {
		items = append(items, map[string]any{
			"context":  "…" + d.Title + "…",
			"ranking":  0.5,
			"document": documentObject(d),
		})
	}
	return jsonReply(http.StatusOK, pagedBody(items, in), nil)
}

// documentsOf returns the documents of one collection, or every document when collectionID is empty.
func (sn snapshot) documentsOf(collectionID string) []Document {
	if collectionID == "" {
		return sn.documents
	}
	var out []Document
	for _, d := range sn.documents {
		if d.CollectionID == collectionID {
			out = append(out, d)
		}
	}
	return out
}

// pagedBody wraps one page of items with the pagination block of the router. The next path is always set here, so a
// client that follows it would be caught by the request count.
func pagedBody(items []any, in requestBody) map[string]any {
	return map[string]any{
		"data": items,
		"pagination": map[string]any{
			"offset":   in.Offset,
			"limit":    in.Limit,
			"nextPath": fmt.Sprintf("/api/documents.list?offset=%d&limit=%d", in.Offset+in.Limit, in.Limit),
		},
	}
}

// endpointNotFound answers the router's catch-all: 404 with the message "Endpoint not found".
func endpointNotFound() reply {
	body := errorBody(http.StatusNotFound)
	body["message"] = "Endpoint not found"
	return jsonReply(http.StatusNotFound, body, nil)
}

func (sn snapshot) hasCollection(id string) bool {
	for _, c := range sn.collections {
		if c.ID == id {
			return true
		}
	}
	return false
}

func (sn snapshot) findDocument(id string) (Document, bool) {
	for _, d := range sn.documents {
		if d.ID == id {
			return d, true
		}
	}
	return Document{}, false
}

// window returns the slice bounds of one page, clamped to the held items.
func window(total, offset, limit int) (int, int) {
	start := min(offset, total)
	end := min(offset+limit, total)
	return start, end
}

// renderCollection copies the fixture collection and sets the scripted fields.
func renderCollection(template map[string]any, c Collection) map[string]any {
	obj := cloneObject(template)
	obj["id"] = c.ID
	obj["name"] = c.Name
	obj["url"] = "/collection/" + c.ID
	return obj
}

// errorReply answers with the sanitized error body for status.
func errorReply(status int) reply {
	return jsonReply(status, errorBody(status), nil)
}

// errorBody returns the sanitized error envelope for status. Statuses without their
// own fixture reuse the 500 fixture with the status replaced.
func errorBody(status int) map[string]any {
	name := fmt.Sprintf("error-%d", status)
	if _, ok := fixtures[name]; !ok {
		name = "error-500"
	}
	body := cloneObject(fixtureTop(name))
	body["status"] = status
	return body
}

// jsonReply encodes body as the JSON response of status.
func jsonReply(status int, body any, header http.Header) reply {
	raw, err := json.Marshal(body)
	if err != nil {
		// The bodies are built from fixtures and plain values, so encoding cannot fail.
		panic(fmt.Sprintf("outlinestub: encode reply: %v", err))
	}
	if header == nil {
		header = http.Header{}
	}
	header.Set("Content-Type", "application/json")
	return reply{status: status, header: header, body: raw}
}

// fixtureTop returns the decoded top-level object of a named fixture.
func fixtureTop(name string) map[string]any {
	obj, ok := fixtures[name]
	if !ok {
		panic(fmt.Sprintf("outlinestub: unknown fixture %q", name))
	}
	return obj
}

// fixtureObject returns the template object of a fixture. An index of -1 reads a
// key holding one object; an index of 0 or more reads the element at that position
// of a key holding an array.
func fixtureObject(name, key string, index int) map[string]any {
	top := fixtureTop(name)
	value, ok := top[key]
	if !ok {
		panic(fmt.Sprintf("outlinestub: fixture %q lacks %q", name, key))
	}
	if index < 0 {
		obj, ok := value.(map[string]any)
		if !ok {
			panic(fmt.Sprintf("outlinestub: fixture %q key %q is not an object", name, key))
		}
		return obj
	}
	list, ok := value.([]any)
	if !ok || index >= len(list) {
		panic(fmt.Sprintf("outlinestub: fixture %q key %q lacks element %d", name, key, index))
	}
	obj, ok := list[index].(map[string]any)
	if !ok {
		panic(fmt.Sprintf("outlinestub: fixture %q element is not an object", name))
	}
	return obj
}

// cloneObject copies the top level of an object, so one replay never changes the fixture.
func cloneObject(template map[string]any) map[string]any {
	out := make(map[string]any, len(template))
	for k, v := range template {
		out[k] = v
	}
	return out
}

// loadFixtures decodes every embedded fixture, keyed by file name without ".json".
func loadFixtures() map[string]map[string]any {
	entries, err := fs.ReadDir(fixtureFS, "testdata")
	if err != nil {
		panic(fmt.Sprintf("outlinestub: read fixtures: %v", err))
	}
	out := make(map[string]map[string]any, len(entries))
	for _, entry := range entries {
		raw, err := fixtureFS.ReadFile("testdata/" + entry.Name())
		if err != nil {
			panic(fmt.Sprintf("outlinestub: read fixture %s: %v", entry.Name(), err))
		}
		var obj map[string]any
		dec := json.NewDecoder(bytes.NewReader(raw))
		if err := dec.Decode(&obj); err != nil {
			panic(fmt.Sprintf("outlinestub: decode fixture %s: %v", entry.Name(), err))
		}
		out[strings.TrimSuffix(entry.Name(), ".json")] = obj
	}
	return out
}
