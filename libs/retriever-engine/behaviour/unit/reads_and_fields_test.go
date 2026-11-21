package unit_test

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine/replay"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// The reads of this file map a reply to the fields that a spec declares. Each spec is built from one of the heads below
// and a block of fields, so each test shows the one field rule that it binds.

// readsListDoc is the list read of the example wiki: a GET of the documents with a constant header, a cursor, and one
// canned example.
const readsListDoc = `  - id: example-wiki.docs.list
    summary: List documents.
    access: read
    method: GET
    path: /docs
    params:
      - name: cursor
        type: ID
        in: query
        required: false
      - name: limit
        type: Limit
        in: query
        required: false
        default: 20
    headers:
      X-Example-Client: retriever
    pagination:
      style: cursor
      max_pages: 5
      cursor_param: cursor
      next_pointer: /next_cursor
    response:
      items: /items
      fields:
        id:
          pointer: /id
        title:
          pointer: /title
    examples:
      - request:
          limit: 20
        response:
          status: 200
          body:
            items:
              - id: doc-1
                title: First document
            next_cursor: null
        expect:
          id: doc-1
          title: First document
`

// readsObjectHead opens the single-document read, which takes a document id in its path.
const readsObjectHead = `  - id: example-wiki.docs.one
    summary: Read one document.
    access: read
    method: GET
    path: /docs/{id}
    params:
      - name: id
        type: ID
        in: path
        required: true
    response:
      fields:
        title:
          pointer: /title
`

// readsObjectExamples is the canned example of the single-document read.
const readsObjectExamples = `    examples:
      - request:
          id: doc-1
        response:
          status: 200
          body:
            title: First document
            body: Hello
        expect:
          title: First document
`

// readsObjectDoc returns the single-document read with the given extra field block, after the title field.
func readsObjectDoc(fields string) string {
	return readsObjectHead + fields + readsObjectExamples
}

// readsFirstDoc is the first-item read: it maps item zero of the list at /data, and nothing of the rest.
const readsFirstDoc = `  - id: example-wiki.docs.first
    summary: Read the first document of a list.
    access: read
    method: GET
    path: /docs/first
    response:
      first: /data
      fields:
        id:
          pointer: /id
        title:
          pointer: /title
    examples:
      - request: {}
        response:
          status: 200
          body:
            data:
              - id: doc-1
                title: First document
        expect:
          id: doc-1
          title: First document
`

// readsIssueDoc is a single-document read whose path takes an issue key.
const readsIssueDoc = `  - id: example-wiki.issues.get
    summary: Read one issue.
    access: read
    method: GET
    path: /issues/{key}
    params:
      - name: key
        type: IssueKey
        in: path
        required: true
    response:
      fields:
        title:
          pointer: /title
    examples:
      - request:
          key: PROJ-1
        response:
          status: 200
          body:
            title: First issue
        expect:
          title: First issue
`

// readsValueDoc is a read whose parameters are the catalog kinds of an icon, a list of URLs, and a list of events.
const readsValueDoc = `  - id: example-wiki.pages.find
    summary: Find pages by their icon, redirect addresses, and events.
    access: read
    method: GET
    path: /pages
    params:
      - name: icon
        type: Icon
        in: query
        required: false
      - name: urls
        type: URLList
        in: query
        required: false
      - name: events
        type: EventList
        in: query
        required: false
    response:
      fields:
        id:
          pointer: /id
    examples:
      - request:
          icon: star
          urls: [https://docs.example.test/callback]
          events: [documents.create]
        response:
          status: 200
          body:
            id: page-1
        expect:
          id: page-1
`

// readsStubEngine builds the engine of a spec over its replay stub, and returns the stub to inspect the requests.
func readsStubEngine(t *testing.T, s *spec.Spec) (*engine.Engine, *replay.Stub) {
	t.Helper()
	stub := rebindStub(t, s)
	return rebindEngine(t, s, stub, &rebindClock{}), stub
}

// readsFlagsEmpty reports whether the flag map of a record is present and holds no key.
func readsFlagsEmpty(rec domain.Record) bool {
	v, ok := rec.Get("flags")
	if !ok {
		return false
	}
	flags, isRec := v.(domain.Record)
	return isRec && len(flags.Fields()) == 0
}

// TestReadsFields_ListReadSendsOneRequestAndMapsFields binds the list read to one request with the declared method, path,
// and header, and to the declared fields of its items.
// @scenario specs/libs/retriever-engine/behaviours/reads-and-fields.feature: A list read sends one declared request and maps each declared field
func TestReadsFields_ListReadSendsOneRequestAndMapsFields(t *testing.T) {
	s := rebindSpec(t, readsListDoc)
	eng, stub := readsStubEngine(t, s)
	rec, err := pagingRead(eng, "example-wiki.docs.list", map[string]any{"limit": 20})
	if err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	reqs := stub.Requests()
	if len(reqs) != 1 {
		t.Fatalf("%d requests, want one", len(reqs))
	}
	if reqs[0].Method != http.MethodGet || reqs[0].Path != "/v1/docs" || reqs[0].Header.Get("X-Example-Client") != "retriever" {
		t.Fatalf("the request is %s %s with client %q, want the declared method, path, and header",
			reqs[0].Method, reqs[0].Path, reqs[0].Header.Get("X-Example-Client"))
	}
	items := rec.Items()
	if len(items) != 1 || items[0].Str("id") != "doc-1" || items[0].Str("title") != "First document" {
		t.Fatalf("the items do not hold each declared field: %d items", len(items))
	}
}

// TestReadsFields_ObjectReadMapsEachDeclaredField binds the single-document read to each declared field of its canned
// reply.
// @scenario specs/libs/retriever-engine/behaviours/reads-and-fields.feature: An object read maps each declared field of its canned reply
func TestReadsFields_ObjectReadMapsEachDeclaredField(t *testing.T) {
	s := rebindSpec(t, readsObjectDoc("        body:\n          pointer: /body\n"))
	eng, _ := readsStubEngine(t, s)
	rec, err := pagingRead(eng, "example-wiki.docs.one", map[string]any{"id": "doc-1"})
	if err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	if rec.Str("title") != "First document" || rec.Str("body") != "Hello" {
		t.Fatalf("the record holds title %q and body %q, want both declared fields", rec.Str("title"), rec.Str("body"))
	}
}

// TestReadsFields_ObjectReadRefusesList binds a single-document read whose reply is a list to an unexpected response with
// no record.
// @scenario specs/libs/retriever-engine/behaviours/reads-and-fields.feature: An object read refuses a reply that is a list
func TestReadsFields_ObjectReadRefusesList(t *testing.T) {
	s := rebindSpec(t, readsObjectDoc(""))
	tr := rebindReplies(rebindJSON(http.StatusOK, `[{"title":"First document"}]`))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	rec, err := pagingRead(eng, "example-wiki.docs.one", map[string]any{"id": "doc-1"})
	if !errors.Is(err, domain.ErrUnexpectedResponse) || len(rec.Fields()) != 0 {
		t.Fatalf("the error is %v with %d fields, want an unexpected response and no record", err, len(rec.Fields()))
	}
}

// TestReadsFields_ScalarFieldHoldingObjectFailsWholeRead binds a declared scalar field that holds an object to a failure of
// the whole read, with no record.
// @scenario specs/libs/retriever-engine/behaviours/reads-and-fields.feature: A declared scalar field that holds an object fails the whole read
func TestReadsFields_ScalarFieldHoldingObjectFailsWholeRead(t *testing.T) {
	s := rebindSpec(t, readsObjectDoc(""))
	tr := rebindReplies(rebindJSON(http.StatusOK, `{"title":{"text":"First document"}}`))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	rec, err := pagingRead(eng, "example-wiki.docs.one", map[string]any{"id": "doc-1"})
	if err == nil || len(rec.Fields()) != 0 {
		t.Fatalf("the read returns %v with %d fields, want a failure and no record", err, len(rec.Fields()))
	}
}

// TestReadsFields_FirstItemMapsOnlyItemZero binds a first-item read to the fields of its first item only.
// @scenario specs/libs/retriever-engine/behaviours/reads-and-fields.feature: A first-item read maps only item zero
func TestReadsFields_FirstItemMapsOnlyItemZero(t *testing.T) {
	s := rebindSpec(t, readsFirstDoc)
	tr := rebindReplies(rebindJSON(http.StatusOK, `{"data":[{"id":"doc-1","title":"First document"},{"id":"doc-2","title":"Second"}]}`))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	rec, err := pagingRead(eng, "example-wiki.docs.first", nil)
	if err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	if rec.IsList() || rec.Str("id") != "doc-1" || rec.Str("title") != "First document" {
		t.Fatalf("the record holds id %q, want the fields of item zero only", rec.Str("id"))
	}
}

// TestReadsFields_FirstItemOfEmptyListIsNotFound binds a first-item read of an empty list to the not-found code, with no
// record.
// @scenario specs/libs/retriever-engine/behaviours/reads-and-fields.feature: A first-item read of an empty list fails with not found and no partial record
func TestReadsFields_FirstItemOfEmptyListIsNotFound(t *testing.T) {
	s := rebindSpec(t, readsFirstDoc)
	tr := rebindReplies(rebindJSON(http.StatusOK, `{"data":[]}`))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	rec, err := pagingRead(eng, "example-wiki.docs.first", nil)
	if !errors.Is(err, domain.ErrNotFound) || len(rec.Fields()) != 0 {
		t.Fatalf("the error is %v with %d fields, want not found and no record", err, len(rec.Fields()))
	}
}

// TestReadsFields_FirstPointerNotListIsUnexpected binds a first pointer that holds an object or a text to an unexpected
// response.
// @scenario specs/libs/retriever-engine/behaviours/reads-and-fields.feature: A first-item read whose first pointer holds a value that is not a list fails as unexpected
func TestReadsFields_FirstPointerNotListIsUnexpected(t *testing.T) {
	rows := []struct{ value, body string }{
		{value: "an object", body: `{"data":{"id":"doc-1"}}`},
		{value: "a text", body: `{"data":"doc-1"}`},
	}
	for _, row := range rows {
		t.Run(row.value, func(t *testing.T) {
			s := rebindSpec(t, readsFirstDoc)
			tr := rebindReplies(rebindJSON(http.StatusOK, row.body))
			eng := rebindEngine(t, s, tr, &rebindClock{})
			if _, err := pagingRead(eng, "example-wiki.docs.first", nil); !errors.Is(err, domain.ErrUnexpectedResponse) {
				t.Fatalf("the error is %v, want an unexpected response", err)
			}
		})
	}
}

// TestReadsFields_RequiredFieldStateFailsRead binds a required reply field that is missing, null, or empty to an unexpected
// response with no record.
// @scenario specs/libs/retriever-engine/behaviours/reads-and-fields.feature: A required reply field that is <state> fails the read with no partial record
func TestReadsFields_RequiredFieldStateFailsRead(t *testing.T) {
	rows := []struct{ state, body string }{
		{state: "missing", body: `{"body":"Hello"}`},
		{state: "null", body: `{"title":null,"body":"Hello"}`},
		{state: "empty", body: `{"title":"","body":"Hello"}`},
	}
	for _, row := range rows {
		t.Run(row.state, func(t *testing.T) {
			doc := strings.Replace(readsObjectDoc(""), "        title:\n          pointer: /title\n", "        title:\n          pointer: /title\n          required: true\n", 1)
			s := rebindSpec(t, doc)
			tr := rebindReplies(rebindJSON(http.StatusOK, row.body))
			eng := rebindEngine(t, s, tr, &rebindClock{})
			rec, err := pagingRead(eng, "example-wiki.docs.one", map[string]any{"id": "doc-1"})
			if !errors.Is(err, domain.ErrUnexpectedResponse) || len(rec.Fields()) != 0 {
				t.Fatalf("the error is %v with %d fields, want an unexpected response and no record", err, len(rec.Fields()))
			}
		})
	}
}

// TestReadsFields_ListItemMissingRequiredValueFailsWholeList binds a list whose second item lacks a required value to a
// failure of the whole list, with no partial list.
// @scenario specs/libs/retriever-engine/behaviours/reads-and-fields.feature: One list item with a missing required value fails the whole list read
func TestReadsFields_ListItemMissingRequiredValueFailsWholeList(t *testing.T) {
	doc := strings.Replace(readsListDoc, "        title:\n          pointer: /title\n", "        title:\n          pointer: /title\n          required: true\n", 1)
	s := rebindSpec(t, doc)
	tr := rebindReplies(rebindJSON(http.StatusOK, `{"items":[{"id":"doc-1","title":"First"},{"id":"doc-2"}],"next_cursor":null}`))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	rec, err := pagingRead(eng, "example-wiki.docs.list", nil)
	if !errors.Is(err, domain.ErrUnexpectedResponse) || len(rec.Items()) != 0 {
		t.Fatalf("the error is %v with %d items, want an unexpected response and no partial list", err, len(rec.Items()))
	}
}

// TestReadsFields_ClosedFormatOfAnotherShapeFailsRead binds a value of another shape for a field with the id format to an
// unexpected response.
// @scenario specs/libs/retriever-engine/behaviours/reads-and-fields.feature: A reply value that its closed format refuses fails the read as unexpected
func TestReadsFields_ClosedFormatOfAnotherShapeFailsRead(t *testing.T) {
	s := rebindSpec(t, readsObjectDoc("        id:\n          pointer: /id\n          format: id\n"))
	tr := rebindReplies(rebindJSON(http.StatusOK, `{"id":7,"title":"First document"}`))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	if _, err := pagingRead(eng, "example-wiki.docs.one", map[string]any{"id": "doc-1"}); !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("the error is %v, want an unexpected response", err)
	}
}

// TestReadsFields_ClosedFormatShapeMapsToField binds a reply value of the right shape for each closed format name to its
// field.
// @scenario specs/libs/retriever-engine/behaviours/reads-and-fields.feature: A reply value of the right shape for a closed format name maps to its field
func TestReadsFields_ClosedFormatShapeMapsToField(t *testing.T) {
	rows := []struct{ format, value string }{
		{format: "id", value: "doc-1"},
		{format: "uuid", value: "123e4567-e89b-12d3-a456-426614174000"},
		{format: "share-id", value: "share-a1-b2"},
	}
	for _, row := range rows {
		t.Run(row.format, func(t *testing.T) {
			s := rebindSpec(t, readsObjectDoc(fmt.Sprintf("        value:\n          pointer: /value\n          format: %s\n", row.format)))
			tr := rebindReplies(rebindJSON(http.StatusOK, fmt.Sprintf(`{"title":"First document","value":%q}`, row.value)))
			eng := rebindEngine(t, s, tr, &rebindClock{})
			rec, err := pagingRead(eng, "example-wiki.docs.one", map[string]any{"id": "doc-1"})
			if err != nil {
				t.Fatalf("the read fails: %v", err)
			}
			if rec.Str("value") != row.value {
				t.Fatalf("the field holds %q, want %q", rec.Str("value"), row.value)
			}
		})
	}
}

// TestReadsFields_RefusalNamesFieldPathNotValue binds a refused reply value to an error that names the field path and never
// the value.
// @scenario specs/libs/retriever-engine/behaviours/reads-and-fields.feature: A refusal of a reply value names the field path and never the value
func TestReadsFields_RefusalNamesFieldPathNotValue(t *testing.T) {
	s := rebindSpec(t, readsObjectDoc("        doc_id:\n          pointer: /id\n          format: id\n"))
	tr := rebindReplies(rebindJSON(http.StatusOK, `{"id":"not valid!","title":"First document"}`))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	_, err := pagingRead(eng, "example-wiki.docs.one", map[string]any{"id": "doc-1"})
	if err == nil || !strings.Contains(err.Error(), "doc_id") || strings.Contains(err.Error(), "not valid!") {
		t.Fatalf("the error is %v, want the field path and not the value", err)
	}
}

// TestReadsFields_StringsFieldKeepsReplyOrder binds a strings field to the list of strings in the reply order.
// @scenario specs/libs/retriever-engine/behaviours/reads-and-fields.feature: A strings field maps a reply list of strings in reply order
func TestReadsFields_StringsFieldKeepsReplyOrder(t *testing.T) {
	s := rebindSpec(t, readsObjectDoc("        labels:\n          pointer: /labels\n          format: strings\n"))
	tr := rebindReplies(rebindJSON(http.StatusOK, `{"title":"First document","labels":["b","a"]}`))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	rec, err := pagingRead(eng, "example-wiki.docs.one", map[string]any{"id": "doc-1"})
	if err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	if got, _ := rec.Get("labels"); fmt.Sprint(got) != "[b a]" {
		t.Fatalf("the strings field holds %v, want the reply order b then a", got)
	}
}

// TestReadsFields_StringsFieldRefusesNonStrings binds a strings field whose reply holds a number, or is not a list, to an
// unexpected response with no record.
// @scenario specs/libs/retriever-engine/behaviours/reads-and-fields.feature: A strings field whose reply is not a list of strings fails the read
func TestReadsFields_StringsFieldRefusesNonStrings(t *testing.T) {
	rows := []struct{ reply, body string }{
		{reply: "a list with a number in it", body: `{"title":"First document","labels":["a",1]}`},
		{reply: "a value that is not a list", body: `{"title":"First document","labels":"a"}`},
	}
	for _, row := range rows {
		t.Run(row.reply, func(t *testing.T) {
			s := rebindSpec(t, readsObjectDoc("        labels:\n          pointer: /labels\n          format: strings\n"))
			tr := rebindReplies(rebindJSON(http.StatusOK, row.body))
			eng := rebindEngine(t, s, tr, &rebindClock{})
			rec, err := pagingRead(eng, "example-wiki.docs.one", map[string]any{"id": "doc-1"})
			if !errors.Is(err, domain.ErrUnexpectedResponse) || len(rec.Fields()) != 0 {
				t.Fatalf("the error is %v with %d fields, want an unexpected response and no record", err, len(rec.Fields()))
			}
		})
	}
}

// TestReadsFields_MissingOptionalStringsIsNull binds a missing optional strings field to null, with every declared field
// still in the record.
// @scenario specs/libs/retriever-engine/behaviours/reads-and-fields.feature: A missing optional strings field maps to null and the record keeps every declared field
func TestReadsFields_MissingOptionalStringsIsNull(t *testing.T) {
	s := rebindSpec(t, readsObjectDoc("        labels:\n          pointer: /labels\n          format: strings\n"))
	tr := rebindReplies(rebindJSON(http.StatusOK, `{"title":"First document"}`))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	rec, err := pagingRead(eng, "example-wiki.docs.one", map[string]any{"id": "doc-1"})
	if err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	if v, ok := rec.Get("labels"); !ok || v != nil {
		t.Fatalf("the strings field is %v, want null", v)
	}
	if rec.Str("title") != "First document" || len(rec.Fields()) != 2 {
		t.Fatalf("the record holds %d fields, want every declared field", len(rec.Fields()))
	}
}

// TestReadsFields_NestedRecordWithholdsUndeclaredField binds a nested record to its declared sub-field, with an undeclared
// e-mail address that reaches no output and no error.
// @scenario specs/libs/retriever-engine/behaviours/reads-and-fields.feature: A nested record maps its declared sub-fields and withholds an undeclared one
func TestReadsFields_NestedRecordWithholdsUndeclaredField(t *testing.T) {
	s := rebindSpec(t, readsObjectDoc("        author:\n          object:\n            name:\n              pointer: /author/name\n"))
	tr := rebindReplies(rebindJSON(http.StatusOK, `{"title":"First document","author":{"name":"Ann","email":"ann@example.test"}}`))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	rec, err := pagingRead(eng, "example-wiki.docs.one", map[string]any{"id": "doc-1"})
	if err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	nested := rec.Rec("author")
	if len(nested.Fields()) != 1 || nested.Str("name") != "Ann" {
		t.Fatalf("the nested record holds %d fields, want the declared name only", len(nested.Fields()))
	}
	if strings.Contains(fmt.Sprintf("%v", rec.Fields()), "ann@example.test") {
		t.Fatalf("the e-mail address reaches the output")
	}
}

// TestReadsFields_NestedSubFieldReachingObjectNamesPath binds a nested sub-field whose pointer reaches a whole object to a
// failure that names its dotted path, with no record.
// @scenario specs/libs/retriever-engine/behaviours/reads-and-fields.feature: A nested sub-field whose pointer reaches a whole object fails the read and names its dotted path
func TestReadsFields_NestedSubFieldReachingObjectNamesPath(t *testing.T) {
	s := rebindSpec(t, readsObjectDoc("        author:\n          object:\n            info:\n              pointer: /author\n"))
	tr := rebindReplies(rebindJSON(http.StatusOK, `{"title":"First document","author":{"name":"Ann"}}`))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	rec, err := pagingRead(eng, "example-wiki.docs.one", map[string]any{"id": "doc-1"})
	if !errors.Is(err, domain.ErrUnexpectedResponse) || !strings.Contains(err.Error(), "author.info") || len(rec.Fields()) != 0 {
		t.Fatalf("the error is %v, want an unexpected response naming author.info and no record", err)
	}
}

// TestReadsFields_MissingRequiredSubFieldNamesPath binds a missing required nested sub-field to a failure that names its
// dotted path.
// @scenario specs/libs/retriever-engine/behaviours/reads-and-fields.feature: A missing required nested sub-field fails the read and names its dotted path
func TestReadsFields_MissingRequiredSubFieldNamesPath(t *testing.T) {
	s := rebindSpec(t, readsObjectDoc("        author:\n          object:\n            name:\n              pointer: /author/name\n              required: true\n"))
	tr := rebindReplies(rebindJSON(http.StatusOK, `{"title":"First document","author":{"email":"ann@example.test"}}`))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	_, err := pagingRead(eng, "example-wiki.docs.one", map[string]any{"id": "doc-1"})
	if !errors.Is(err, domain.ErrUnexpectedResponse) || !strings.Contains(err.Error(), "author.name") {
		t.Fatalf("the error is %v, want an unexpected response naming author.name", err)
	}
}

// TestReadsFields_OpenFlagMapKeepsKeysSorted binds an open flag map to its three keys in sorted order with their flag values.
// @scenario specs/libs/retriever-engine/behaviours/reads-and-fields.feature: An open flag map keeps every key in sorted order with its flag value
func TestReadsFields_OpenFlagMapKeepsKeysSorted(t *testing.T) {
	s := rebindSpec(t, readsObjectDoc("        flags:\n          pointer: /flags\n          bool_map: true\n"))
	tr := rebindReplies(rebindJSON(http.StatusOK, `{"title":"First document","flags":{"zeta":true,"alpha":false,"mid":true}}`))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	rec, err := pagingRead(eng, "example-wiki.docs.one", map[string]any{"id": "doc-1"})
	if err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	var got []string
	for _, f := range rec.Rec("flags").Fields() {
		got = append(got, fmt.Sprintf("%s=%v", f.Name, f.Value))
	}
	if want := "alpha=false,mid=true,zeta=true"; strings.Join(got, ",") != want {
		t.Fatalf("the flags are %v, want %s", got, want)
	}
}

// TestReadsFields_EmptyFlagMapIsEmptyRecord binds an empty flag object to an empty record and a successful read.
// @scenario specs/libs/retriever-engine/behaviours/reads-and-fields.feature: An open flag map with an empty reply holds an empty record and no failure
func TestReadsFields_EmptyFlagMapIsEmptyRecord(t *testing.T) {
	s := rebindSpec(t, readsObjectDoc("        flags:\n          pointer: /flags\n          bool_map: true\n"))
	tr := rebindReplies(rebindJSON(http.StatusOK, `{"title":"First document","flags":{}}`))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	rec, err := pagingRead(eng, "example-wiki.docs.one", map[string]any{"id": "doc-1"})
	if err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	if !readsFlagsEmpty(rec) {
		t.Fatalf("the flag map is not an empty record")
	}
}

// TestReadsFields_FlagValueNotFlagFailsNamingKeyPath binds a flag map holding a value that is not a flag to a failure that
// names the dotted path of the key, with the value kept out of the record.
// @scenario specs/libs/retriever-engine/behaviours/reads-and-fields.feature: An open flag map that holds a value which is not a flag fails the read naming the dotted path
func TestReadsFields_FlagValueNotFlagFailsNamingKeyPath(t *testing.T) {
	rows := []struct{ value, body string }{
		{value: "a number", body: `{"title":"First document","flags":{"alpha":5}}`},
		{value: "a text", body: `{"title":"First document","flags":{"alpha":"yes"}}`},
	}
	for _, row := range rows {
		t.Run(row.value, func(t *testing.T) {
			s := rebindSpec(t, readsObjectDoc("        flags:\n          pointer: /flags\n          bool_map: true\n"))
			tr := rebindReplies(rebindJSON(http.StatusOK, row.body))
			eng := rebindEngine(t, s, tr, &rebindClock{})
			rec, err := pagingRead(eng, "example-wiki.docs.one", map[string]any{"id": "doc-1"})
			if err == nil || !strings.Contains(err.Error(), "flags.alpha") || len(rec.Fields()) != 0 {
				t.Fatalf("the error is %v with %d fields, want the path flags.alpha and no record", err, len(rec.Fields()))
			}
		})
	}
}

// TestReadsFields_AbsentOptionalFlagMapIsEmptyRecord binds an absent optional flag map to an empty record.
// @scenario specs/libs/retriever-engine/behaviours/reads-and-fields.feature: An absent optional flag map maps to an empty record
func TestReadsFields_AbsentOptionalFlagMapIsEmptyRecord(t *testing.T) {
	s := rebindSpec(t, readsObjectDoc("        flags:\n          pointer: /flags\n          bool_map: true\n"))
	tr := rebindReplies(rebindJSON(http.StatusOK, `{"title":"First document"}`))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	rec, err := pagingRead(eng, "example-wiki.docs.one", map[string]any{"id": "doc-1"})
	if err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	if !readsFlagsEmpty(rec) {
		t.Fatalf("the absent flag map is not an empty record")
	}
}

// TestReadsFields_AbsentRequiredFlagMapNamesField binds an absent required flag map to a failure that names the field.
// @scenario specs/libs/retriever-engine/behaviours/reads-and-fields.feature: An absent required flag map fails the read naming the field
func TestReadsFields_AbsentRequiredFlagMapNamesField(t *testing.T) {
	s := rebindSpec(t, readsObjectDoc("        flags:\n          pointer: /flags\n          bool_map: true\n          required: true\n"))
	tr := rebindReplies(rebindJSON(http.StatusOK, `{"title":"First document"}`))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	_, err := pagingRead(eng, "example-wiki.docs.one", map[string]any{"id": "doc-1"})
	if !errors.Is(err, domain.ErrUnexpectedResponse) || !strings.Contains(err.Error(), "flags") {
		t.Fatalf("the error is %v, want an unexpected response naming flags", err)
	}
}

// TestReadsFields_FallbackPointerMapsSecondShape binds a flag map with a fallback pointer to the flags of the fallback
// shape, when the reply holds only that shape.
// @scenario specs/libs/retriever-engine/behaviours/reads-and-fields.feature: A flag map whose fallback pointer holds the second reply shape maps that shape
func TestReadsFields_FallbackPointerMapsSecondShape(t *testing.T) {
	s := rebindSpec(t, readsObjectDoc("        flags:\n          pointer: /flags_v2\n          fallback_pointer: /flags\n          bool_map: true\n"))
	tr := rebindReplies(rebindJSON(http.StatusOK, `{"title":"First document","flags":{"alpha":true}}`))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	rec, err := pagingRead(eng, "example-wiki.docs.one", map[string]any{"id": "doc-1"})
	if err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	if v, _ := rec.Rec("flags").Get("alpha"); v != true {
		t.Fatalf("the flag map holds alpha %v, want the fallback flags", v)
	}
}

// TestReadsFields_UndeclaredArgumentRefusedBeforeRequest binds a read with an argument the spec does not declare to a
// refusal before any request.
// @scenario specs/libs/retriever-engine/behaviours/reads-and-fields.feature: An undeclared argument is refused before any request
func TestReadsFields_UndeclaredArgumentRefusedBeforeRequest(t *testing.T) {
	s := rebindSpec(t, readsObjectDoc(""))
	tr := rebindReplies(rebindJSON(http.StatusOK, `{"title":"First document"}`))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	_, err := pagingRead(eng, "example-wiki.docs.one", map[string]any{"id": "doc-1", "extra": "x"})
	if !errors.Is(err, domain.ErrUnknownArgument) || len(tr.seen()) != 0 {
		t.Fatalf("the error is %v with %d requests, want a refused argument and no request", err, len(tr.seen()))
	}
}

// TestReadsFields_BadPathArgumentRefusedBeforeRequest binds a call that leaves out a required path argument, or gives one of
// the wrong kind, to a refusal before any request.
// @scenario specs/libs/retriever-engine/behaviours/reads-and-fields.feature: A call that leaves out a required path argument or gives one of the wrong kind is refused before any request
func TestReadsFields_BadPathArgumentRefusedBeforeRequest(t *testing.T) {
	rows := []struct {
		problem string
		args    map[string]any
	}{
		{problem: "a missing required path argument", args: map[string]any{}},
		{problem: "a path argument of the wrong kind", args: map[string]any{"id": 42}},
	}
	for _, row := range rows {
		t.Run(row.problem, func(t *testing.T) {
			s := rebindSpec(t, readsObjectDoc(""))
			tr := rebindReplies(rebindJSON(http.StatusOK, `{"title":"First document"}`))
			eng := rebindEngine(t, s, tr, &rebindClock{})
			if _, err := pagingRead(eng, "example-wiki.docs.one", row.args); err == nil || len(tr.seen()) != 0 {
				t.Fatalf("the call is served with %d requests, want a refusal and no request", len(tr.seen()))
			}
		})
	}
}

// TestReadsFields_NonCanonicalIssueKeyRefused binds an issue key that is not in the canonical form to a refusal before any
// request, and the error does not show the key.
// @scenario specs/libs/retriever-engine/behaviours/reads-and-fields.feature: An issue key that is not in the canonical form is refused before any request
func TestReadsFields_NonCanonicalIssueKeyRefused(t *testing.T) {
	rows := []struct{ name, key string }{
		{name: "a lower-case key", key: "proj-1"},
		{name: "a key with a separator", key: "PROJ/1"},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			s := rebindSpec(t, readsIssueDoc)
			tr := rebindReplies(rebindJSON(http.StatusOK, `{"title":"First issue"}`))
			eng := rebindEngine(t, s, tr, &rebindClock{})
			_, err := pagingRead(eng, "example-wiki.issues.get", map[string]any{"key": row.key})
			if !errors.Is(err, domain.ErrInvalidDocumentID) || strings.Contains(err.Error(), row.key) || len(tr.seen()) != 0 {
				t.Fatalf("the error is %v, want invalid document ID, no key in it, and no request", err)
			}
		})
	}
}

// TestReadsFields_CanonicalIssueKeyAnsweredByReplay binds a canonical issue key that fills the path to the replay answer of
// the read.
// @scenario specs/libs/retriever-engine/behaviours/reads-and-fields.feature: A canonical issue key fills the path and the replay answers the read
func TestReadsFields_CanonicalIssueKeyAnsweredByReplay(t *testing.T) {
	s := rebindSpec(t, readsIssueDoc)
	eng, stub := readsStubEngine(t, s)
	rec, err := pagingRead(eng, "example-wiki.issues.get", map[string]any{"key": "PROJ-1"})
	if err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	if rec.Str("title") != "First issue" || len(stub.Requests()) != 1 {
		t.Fatalf("the replay did not answer the rendered request")
	}
}

// TestReadsFields_InvalidParameterValueRefused binds a parameter value that the catalog refuses to a refusal before any
// request.
// @scenario specs/libs/retriever-engine/behaviours/reads-and-fields.feature: A parameter value that the catalog refuses is refused before any request
func TestReadsFields_InvalidParameterValueRefused(t *testing.T) {
	rows := []struct {
		kind, name string
		value      any
	}{
		{kind: "icon", name: "icon", value: "   "},
		{kind: "URL list", name: "urls", value: []string{"http://docs.example.test/callback"}},
		{kind: "event list", name: "events", value: []string{"bogus"}},
	}
	for _, row := range rows {
		t.Run(row.kind, func(t *testing.T) {
			s := rebindSpec(t, readsValueDoc)
			tr := rebindReplies(rebindJSON(http.StatusOK, `{"id":"page-1"}`))
			eng := rebindEngine(t, s, tr, &rebindClock{})
			if _, err := pagingRead(eng, "example-wiki.pages.find", map[string]any{row.name: row.value}); err == nil || len(tr.seen()) != 0 {
				t.Fatalf("the call is served with %d requests, want a refusal and no request", len(tr.seen()))
			}
		})
	}
}

// TestReadsFields_ValidValuesReachReplay binds icon, URL list, and event list values that the catalog accepts to the replay
// answer of the read.
// @scenario specs/libs/retriever-engine/behaviours/reads-and-fields.feature: Icon, URL list, and event list values the catalog accepts reach the replay stub
func TestReadsFields_ValidValuesReachReplay(t *testing.T) {
	s := rebindSpec(t, readsValueDoc)
	eng, stub := readsStubEngine(t, s)
	args := map[string]any{
		"icon":   "star",
		"urls":   []string{"https://docs.example.test/callback"},
		"events": []string{"documents.create"},
	}
	rec, err := pagingRead(eng, "example-wiki.pages.find", args)
	if err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	if rec.Str("id") != "page-1" || len(stub.Requests()) != 1 {
		t.Fatalf("the replay did not answer the rendered request")
	}
}
