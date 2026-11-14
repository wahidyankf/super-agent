package engine_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine/replay"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// postReadOps are three reads over POST, appended to the wiki fixture of the engine tests. The find read sends one declared
// slot as a JSON value. The note read carries a text, for the size limit. The nest read holds an ID list at the eighth
// level of its body, so a list that the slot fills adds a ninth level. Each names example hosts only.
const postReadOps = `  - id: example-wiki.docs.find
    summary: Find documents in a collection.
    access: read
    method: POST
    read_over_post: true
    path: /docs/find
    params:
      - name: collection_id
        type: ID
        in: body
        required: false
    body:
      collectionId: "{{collection_id}}"
    response:
      items: /items
      fields:
        id:
          pointer: /id
    examples:
      - request:
          collection_id: col-1
        response:
          status: 200
          body:
            items:
              - id: doc-1
        expect:
          id: doc-1
  - id: example-wiki.docs.note
    summary: Search documents by a long text.
    access: read
    method: POST
    read_over_post: true
    path: /docs/note
    params:
      - name: text
        type: Text
        in: body
        required: false
    body:
      text: "{{text}}"
    response:
      items: /items
      fields:
        id:
          pointer: /id
    examples:
      - request:
          text: first
        response:
          status: 200
          body:
            items:
              - id: doc-1
        expect:
          id: doc-1
  - id: example-wiki.docs.nest
    summary: Search documents by a list of IDs at a deep level.
    access: read
    method: POST
    read_over_post: true
    path: /docs/nest
    params:
      - name: ids
        type: IDList
        in: body
        required: false
    body:
      a:
        b:
          c:
            d:
              e:
                f:
                  g:
                    h: "{{ids}}"
    response:
      items: /items
      fields:
        id:
          pointer: /id
    examples:
      - request: {}
        response:
          status: 200
          body:
            items:
              - id: doc-1
        expect:
          id: doc-1
`

// postWiki is the wiki fixture with the three POST reads appended. The base fixture keeps its operations, so no other test
// changes.
var postWiki = wikiYAML + postReadOps

// postPath is the target of the find read, which sends no query.
const postPath = "/v1/docs/find"

// opIndex returns the index of the operation of the id in a spec model, and fails the test when it is absent.
func opIndex(t *testing.T, s *spec.Spec, id string) int {
	t.Helper()
	for i, op := range s.Operations {
		if op.ID == id {
			return i
		}
	}
	t.Fatalf("the model has no operation %s", id)
	return -1
}

// jsonBody decodes the JSON body of a recorded request as a generic value.
func jsonBody(t *testing.T, r replay.Request) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(r.Body, &out); err != nil {
		t.Fatalf("the body of the request is not a JSON object: %v", err)
	}
	return out
}

// TestPostReadSendsItsDeclaredSlotAsJSON checks that a POST read sends its one declared slot as a JSON value, once, with the
// content type that the engine sets.
func TestPostReadSendsItsDeclaredSlotAsJSON(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, postWiki))
	rec, err := eng.Read(context.Background(), "example-wiki.docs.find", domain.NewArgs(map[string]any{"collection_id": "col-1"}))
	if err != nil {
		t.Fatalf("the POST read failed: %v", err)
	}
	if items := rec.Items(); len(items) != 1 || items[0].Str("id") != "doc-1" {
		t.Fatalf("the POST read mapped %d items; want doc-1", len(items))
	}
	requests := stub.Requests()
	if len(requests) != 1 {
		t.Fatalf("the POST read sent %d requests; want 1", len(requests))
	}
	got := requests[0]
	if got.Method != http.MethodPost || got.Path != postPath {
		t.Fatalf("request = %s %s; want POST %s", got.Method, got.Path, postPath)
	}
	if body := jsonBody(t, got); len(body) != 1 || body["collectionId"] != "col-1" {
		t.Fatalf("body = %v; want only collectionId", body)
	}
}

// TestPostReadSendsFixedJSONContentType checks that the request of a POST read carries the content type application/json,
// which the engine sets itself, and no other content type.
func TestPostReadSendsFixedJSONContentType(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, postWiki))
	if _, err := eng.Read(context.Background(), "example-wiki.docs.find", domain.NewArgs(map[string]any{"collection_id": "col-1"})); err != nil {
		t.Fatalf("the POST read failed: %v", err)
	}
	if got := stub.Requests()[0].Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q; want application/json", got)
	}
}

// TestPostReadBodyOnlyFromDeclaredSlots checks that a value which looks like a slot, or like JSON, stays a literal value of
// the one declared slot, and that an argument the spec does not declare never reaches the body.
func TestPostReadBodyOnlyFromDeclaredSlots(t *testing.T) {
	cases := []struct {
		name  string
		value string
	}{
		{"a value shaped like another slot", "{{ids}}"},
		{"a value shaped like a template function", "{{ upper(collection_id) }}"},
		{"a value with a quote that would close a JSON string", `col-1","injected":"yes`},
	}
	// The note read takes a Text slot, which accepts any printable value, so each value reaches the body as it was given.
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			eng, stub, _ := newStubEngine(t, loadSpecFrom(t, postWiki))
			args := domain.NewArgs(map[string]any{"text": tc.value})
			if _, err := eng.Read(context.Background(), "example-wiki.docs.note", args); err != nil {
				t.Fatalf("the POST read failed: %v", err)
			}
			body := jsonBody(t, stub.Requests()[0])
			if len(body) != 1 || body["text"] != tc.value {
				t.Fatalf("body = %v; want the value as one literal text", body)
			}
		})
	}
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, postWiki))
	_, err := eng.Read(context.Background(), "example-wiki.docs.find",
		domain.NewArgs(map[string]any{"collection_id": "col-1", "other": "x"}))
	if !errors.Is(err, domain.ErrUnknownArgument) {
		t.Fatalf("err = %v; want ErrUnknownArgument for an undeclared argument", err)
	}
	if n := len(stub.Requests()); n != 0 {
		t.Fatalf("an undeclared argument caused %d requests; want none", n)
	}
}

// TestPostReadBodyIsCappedAtSixtyFourKiB checks the size limit of the body of a POST read at its exact edge. A body of
// 65536 bytes is sent, and one byte more fails before any request.
func TestPostReadBodyIsCappedAtSixtyFourKiB(t *testing.T) {
	const overhead = len(`{"text":""}`)
	atLimit := strings.Repeat("a", spec.MaxPostBodyBytes-overhead)
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, postWiki))
	if _, err := eng.Read(context.Background(), "example-wiki.docs.note", domain.NewArgs(map[string]any{"text": atLimit})); err != nil {
		t.Fatalf("a body of exactly 64 KiB was refused: %v", err)
	}
	if n := len(stub.Requests()); n != 1 {
		t.Fatalf("requests = %d; want 1", n)
	}
	over := strings.Repeat("a", spec.MaxPostBodyBytes-overhead+1)
	eng, stub, _ = newStubEngine(t, loadSpecFrom(t, postWiki))
	_, err := eng.Read(context.Background(), "example-wiki.docs.note", domain.NewArgs(map[string]any{"text": over}))
	if !errors.Is(err, engine.ErrBodyLimit) {
		t.Fatalf("err = %v; want ErrBodyLimit for one byte over 64 KiB", err)
	}
	if n := len(stub.Requests()); n != 0 {
		t.Fatalf("an over-limit body caused %d requests; want none", n)
	}
}

// TestPostReadBodyIsCappedAtEightLevels checks the depth limit of the rendered body. The template nests eight levels, which
// passes, but the list that its slot fills adds a ninth, so the read fails before any request.
func TestPostReadBodyIsCappedAtEightLevels(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, postWiki))
	_, err := eng.Read(context.Background(), "example-wiki.docs.nest", domain.NewArgs(map[string]any{"ids": []string{"col-1"}}))
	if !errors.Is(err, engine.ErrBodyLimit) {
		t.Fatalf("err = %v; want ErrBodyLimit for a ninth level", err)
	}
	if n := len(stub.Requests()); n != 0 {
		t.Fatalf("a too-deep body caused %d requests; want none", n)
	}
}

// TestNewRechecksTheAccessMatrix checks that the engine constructor refuses a model that breaks the matrix or the rules of
// a POST read, even when the model was built by hand and never validated. Each case is refused before any request.
func TestNewRechecksTheAccessMatrix(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*spec.Spec, int)
	}{
		{"a POST read without the marker", func(s *spec.Spec, i int) { s.Operations[i].ReadOverPost = false }},
		{"a GET with the marker", func(s *spec.Spec, i int) { s.Operations[i].Method = http.MethodGet }},
		{"a write with the marker", func(s *spec.Spec, i int) { s.Operations[i].Access = "write" }},
		{"an admin with a GET", func(s *spec.Spec, i int) {
			s.Operations[i].Access = "admin"
			s.Operations[i].ReadOverPost = false
			s.Operations[i].Method = http.MethodGet
		}},
		{"a POST read with no body", func(s *spec.Spec, i int) { s.Operations[i].Body = nil }},
		{"a POST read that sets Content-Type", func(s *spec.Spec, i int) {
			s.Operations[i].Headers = map[string]string{"Content-Type": "text/plain"}
		}},
		{"a POST read with a resolve", func(s *spec.Spec, i int) {
			s.Operations[i].Params[0].Resolve = &spec.Resolve{Operation: "example-wiki.docs.list", Field: "id"}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			model := loadSpecFrom(t, postWiki)
			tc.mutate(model, opIndex(t, model, "example-wiki.docs.find"))
			stub, err := replay.New(loadSpecFrom(t, postWiki))
			if err != nil {
				t.Fatalf("build the replay stub: %v", err)
			}
			_, err = engine.New(model, testSource(t), stubPorts(stub, &fakeClock{}))
			if err == nil {
				t.Fatal("the engine accepted a model that breaks the matrix")
			}
			if !errors.Is(err, spec.ErrInvalid) {
				t.Fatalf("err = %v; want the spec invalid error", err)
			}
			if n := len(stub.Requests()); n != 0 {
				t.Fatalf("a refused model caused %d requests; want none", n)
			}
		})
	}
}
