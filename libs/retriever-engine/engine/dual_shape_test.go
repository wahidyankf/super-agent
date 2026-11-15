package engine_test

import (
	"context"
	_ "embed"
	"errors"
	"net/http"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine/replay"
)

// dualShapeYAML is a read whose fields map two reply shapes: the primary pointer reads the wrapped shape, and the
// fallback pointer reads the unwrapped shape. A 404 carries the mapped reason, unless its body message is exactly
// the declared unavailable message. It names example hosts only.
const dualShapeYAML = `version: 1
kind: example-wiki
summary: Reads one document in either reply shape.
endpoint:
  base_path: /v1
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
operations:
  - id: example-wiki.docs.get
    summary: Read one document.
    access: read
    method: POST
    read_over_post: true
    path: /docs/get
    params:
      - name: id
        type: ID
        in: body
        required: true
    body:
      id: "{{id}}"
    response:
      fields:
        id:
          pointer: /data/document/id
          fallback_pointer: /data/id
          required: true
        title:
          pointer: /data/document/title
          fallback_pointer: /data/title
      errors:
        "404": not_found
      unavailable_message: Endpoint not found
    examples:
      - request:
          id: doc-1
        response:
          status: 200
          body:
            data:
              id: doc-1
              title: Setup
        expect:
          id: doc-1
`

// TestFallbackPointerMapsTheWrappedShape replays the wrapped reply against a dual-shape read: the primary pointers
// map it, and the fallbacks stay unused.
func TestFallbackPointerMapsTheWrappedShape(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, dualShapeYAML))
	stub.Script("POST", "/v1/docs/get", page(`{"data":{"document":{"id":"doc-1","title":"Setup"}}}`))
	rec, err := eng.Read(context.Background(), "example-wiki.docs.get", domain.NewArgs(map[string]any{"id": "doc-1"}))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got := rec.Str("id"); got != "doc-1" {
		t.Fatalf("id = %q, want doc-1", got)
	}
	if got := rec.Str("title"); got != "Setup" {
		t.Fatalf("title = %q, want Setup", got)
	}
}

// TestFallbackPointerMapsTheUnwrappedShape replays the unwrapped reply against a dual-shape read: the primary
// pointers find nothing, and each fallback pointer maps its field.
func TestFallbackPointerMapsTheUnwrappedShape(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, dualShapeYAML))
	stub.Script("POST", "/v1/docs/get", page(`{"data":{"id":"doc-1","title":"Setup"}}`))
	rec, err := eng.Read(context.Background(), "example-wiki.docs.get", domain.NewArgs(map[string]any{"id": "doc-1"}))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got := rec.Str("id"); got != "doc-1" {
		t.Fatalf("id = %q, want doc-1", got)
	}
	if got := rec.Str("title"); got != "Setup" {
		t.Fatalf("title = %q, want Setup", got)
	}
}

// TestFallbackPointerPrefersAnExplicitNull checks that a primary pointer that finds an explicit null wins: the
// fallback serves only the shape that lacks the pointer, never a second guess inside one shape. The required id is
// null, so the read fails, even though the fallback shape holds it.
func TestFallbackPointerPrefersAnExplicitNull(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, dualShapeYAML))
	stub.Script("POST", "/v1/docs/get", page(`{"data":{"document":{"id":null},"id":"doc-1"}}`))
	if _, err := eng.Read(context.Background(), "example-wiki.docs.get", domain.NewArgs(map[string]any{"id": "doc-1"})); !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
	}
}

// unavailableReply returns a 404 reply whose body carries the given message, as the router's catch-all does.
func unavailableReply(message string) []byte {
	return []byte(`{"message":` + `"` + message + `"}`)
}

// TestUnavailableMessageDistinguishesAMissingMethod replays the router's catch-all 404 against a read with the rule:
// the exact declared message fails as operation unavailable, instead of the mapped not found.
func TestUnavailableMessageDistinguishesAMissingMethod(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, dualShapeYAML))
	stub.Script("POST", "/v1/docs/get", replay.Reply{Status: http.StatusNotFound, Body: unavailableReply("Endpoint not found")})
	_, err := eng.Read(context.Background(), "example-wiki.docs.get", domain.NewArgs(map[string]any{"id": "doc-1"}))
	var unavailable domain.OperationUnavailableError
	if !errors.As(err, &unavailable) {
		t.Fatalf("err = %v, want OperationUnavailableError", err)
	}
}

// TestUnavailableMessageKeepsAMissingObjectNotFound replays another 404 against a read with the rule: a body without
// the declared message falls through to the mapped reason, so a missing object keeps its own failure.
func TestUnavailableMessageKeepsAMissingObjectNotFound(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, dualShapeYAML))
	stub.Script("POST", "/v1/docs/get", replay.Reply{Status: http.StatusNotFound, Body: unavailableReply("No document with this ID")})
	_, err := eng.Read(context.Background(), "example-wiki.docs.get", domain.NewArgs(map[string]any{"id": "doc-1"}))
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

// TestUnavailableMessageLeavesOtherStatusesAlone replays a 401 that carries the declared message: the rule is
// 404-only, so the reply keeps the closed error of its own status.
func TestUnavailableMessageLeavesOtherStatusesAlone(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, dualShapeYAML))
	stub.Script("POST", "/v1/docs/get", replay.Reply{Status: http.StatusUnauthorized, Body: unavailableReply("Endpoint not found")})
	_, err := eng.Read(context.Background(), "example-wiki.docs.get", domain.NewArgs(map[string]any{"id": "doc-1"}))
	if !errors.Is(err, domain.ErrKeyRejected) {
		t.Fatalf("err = %v, want ErrKeyRejected", err)
	}
}
