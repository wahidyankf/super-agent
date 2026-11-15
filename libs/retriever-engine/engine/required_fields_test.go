package engine_test

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// requiredFieldsDoc is a read whose reply fields use the E3 keywords: an ID that is required and checked as an ID, a
// UUID-checked field, and a share-ID-checked field. The values are documentation fixtures, never real keys.
const requiredFieldsDoc = `version: 1
kind: example-wiki
summary: Reads entries with required and checked fields.
endpoint:
  base_path: /v1
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
operations:
  - id: example-wiki.entries.get
    summary: Read one entry.
    access: read
    method: GET
    path: /entries/{id}
    params:
      - name: id
        type: ID
        in: path
        required: true
    response:
      fields:
        id:
          pointer: /id
          required: true
          format: id
        uid:
          pointer: /uid
          format: uuid
        slug:
          pointer: /slug
          format: share-id
    examples:
      - request:
          id: doc-1
        response:
          status: 200
          body:
            id: doc-1
        expect:
          id: doc-1
`

// requiredListDoc is a list read whose item ID is required and checked as an ID.
const requiredListDoc = `version: 1
kind: example-wiki
summary: Lists entries with a required and checked ID.
endpoint:
  base_path: /v1
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
operations:
  - id: example-wiki.entries.list
    summary: List entries.
    access: read
    method: GET
    path: /entries
    response:
      items: /items
      fields:
        id:
          pointer: /id
          required: true
          format: id
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

// TestRequiredParamIsRefusedBeforeAnyRequest checks the request side of E3: a call with a missing argument is refused
// with missing_argument before any request, so the stub counts zero requests. The request side exists today; this test
// pins that no request leaves.
func TestRequiredParamIsRefusedBeforeAnyRequest(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, requiredFieldsDoc))
	if _, err := eng.Read(context.Background(), "example-wiki.entries.get", domain.NewArgs(nil)); !errors.Is(err, domain.ErrMissingArgument) {
		t.Fatalf("a missing argument = %v, want ErrMissingArgument", err)
	}
	if n := len(stub.Requests()); n != 0 {
		t.Fatalf("a request with a missing argument sent %d requests, want zero", n)
	}
}

// TestRequiredReplyFieldMissingIsUnexpected checks the reply side of E3: a missing, null, or empty required value is an
// unexpected_response, and no partial record is returned.
func TestRequiredReplyFieldMissingIsUnexpected(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"a missing field", `{"uid":"123e4567-e89b-12d3-a456-426614174000","slug":"my-share-slug"}`},
		{"a null field", `{"id":null,"uid":"123e4567-e89b-12d3-a456-426614174000","slug":"my-share-slug"}`},
		{"an empty field", `{"id":"","uid":"123e4567-e89b-12d3-a456-426614174000","slug":"my-share-slug"}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			eng, stub, _ := newStubEngine(t, loadSpecFrom(t, requiredFieldsDoc))
			stub.Script("GET", "/v1/entries/doc-1", page(c.body))
			rec, err := eng.Read(context.Background(), "example-wiki.entries.get", domain.NewArgs(map[string]any{"id": "doc-1"}))
			if !errors.Is(err, domain.ErrUnexpectedResponse) {
				t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
			}
			if len(rec.Fields()) != 0 {
				t.Fatalf("a refused read returned a partial record: %+v", rec)
			}
			if n := len(stub.Requests()); n != 1 {
				t.Fatalf("the refused read sent %d requests, want the one that carried the bad reply", n)
			}
		})
	}
}

// TestRequiredReplyFieldMissingFailsTheList checks that one item with a missing required value fails the whole list
// read: the read is an unexpected_response with no partial record.
func TestRequiredReplyFieldMissingFailsTheList(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, requiredListDoc))
	stub.Script("GET", "/v1/entries", page(`{"items":[{"id":"doc-1"},{"title":"no id here"}]}`))
	rec, err := eng.Read(context.Background(), "example-wiki.entries.list", domain.NewArgs(nil))
	if !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
	}
	if len(rec.Items()) != 0 {
		t.Fatalf("a refused list read returned %d partial records", len(rec.Items()))
	}
}

// TestReplyFormatMismatchIsUnexpected checks the format side of E3: a value that its closed format name refuses is an
// unexpected_response, while a value of the right shape maps. The check runs on the value, never as a pattern.
func TestReplyFormatMismatchIsUnexpected(t *testing.T) {
	cases := []struct {
		name  string
		body  string
		field string
	}{
		{"an ID with a dot", `{"id":"doc.1","uid":"123e4567-e89b-12d3-a456-426614174000","slug":"my-share-slug"}`, "id"},
		{"a text where a UUID goes", `{"id":"doc-1","uid":"not-a-uuid","slug":"my-share-slug"}`, "uid"},
		{"upper case where a share slug goes", `{"id":"doc-1","uid":"123e4567-e89b-12d3-a456-426614174000","slug":"My-Slug"}`, "slug"},
		{"a number where an ID goes", `{"id":7,"uid":"123e4567-e89b-12d3-a456-426614174000","slug":"my-share-slug"}`, "id"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			eng, stub, _ := newStubEngine(t, loadSpecFrom(t, requiredFieldsDoc))
			stub.Script("GET", "/v1/entries/doc-1", page(c.body))
			if _, err := eng.Read(context.Background(), "example-wiki.entries.get", domain.NewArgs(map[string]any{"id": "doc-1"})); !errors.Is(err, domain.ErrUnexpectedResponse) {
				t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
			}
		})
	}
}

// TestReplyFormatMatchMaps checks that a value of the right shape for each closed name maps: an ID, a UUID, and a
// share slug, including the UUID form of a share ID.
func TestReplyFormatMatchMaps(t *testing.T) {
	for _, slug := range []string{"my-share-slug", "123e4567-e89b-12d3-a456-426614174000"} {
		eng, stub, _ := newStubEngine(t, loadSpecFrom(t, requiredFieldsDoc))
		body := `{"id":"doc-1","uid":"123e4567-e89b-12d3-a456-426614174000","slug":` + strconv.Quote(slug) + `}`
		stub.Script("GET", "/v1/entries/doc-1", page(body))
		rec, err := eng.Read(context.Background(), "example-wiki.entries.get", domain.NewArgs(map[string]any{"id": "doc-1"}))
		if err != nil {
			t.Fatalf("a matching reply was refused: %v", err)
		}
		if rec.Str("id") != "doc-1" || rec.Str("slug") != slug {
			t.Fatalf("a matching reply did not map: %+v", rec)
		}
	}
}

// TestRefusalNamesPathNotValue checks R1 for the E3 refusal: the failure names the field path, and it never repeats
// the value of the reply.
func TestRefusalNamesPathNotValue(t *testing.T) {
	const planted = "not an id, and SECRET-PLANTED-VALUE must not echo"
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, requiredFieldsDoc))
	stub.Script("GET", "/v1/entries/doc-1", page(`{"id":"`+planted+`","uid":"123e4567-e89b-12d3-a456-426614174000","slug":"my-share-slug"}`))
	_, err := eng.Read(context.Background(), "example-wiki.entries.get", domain.NewArgs(map[string]any{"id": "doc-1"}))
	if !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
	}
	if !strings.Contains(err.Error(), "id") {
		t.Fatalf("the refusal does not name the field path: %q", err.Error())
	}
	if strings.Contains(err.Error(), planted) {
		t.Fatalf("the refusal echoes the value: %q", err.Error())
	}
}
