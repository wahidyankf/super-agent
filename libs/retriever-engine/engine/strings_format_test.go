package engine_test

import (
	"context"
	"errors"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// stringsYAML is a read whose reply holds a list of strings, mapped with the
// strings format, and a nested record with a strings sub-field.
const stringsYAML = `version: 1
kind: example-wiki
summary: Reads with string-list reply fields of an example wiki.
endpoint:
  base_path: /v1
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
operations:
  - id: example-wiki.hooks.list
    summary: List the webhook subscriptions of the wiki.
    access: read
    method: GET
    path: /hooks
    response:
      items: /items
      fields:
        id:
          pointer: /id
          required: true
          format: id
        events:
          pointer: /events
          format: strings
        settings:
          object:
            channel:
              pointer: /settings/channel
            aliases:
              pointer: /settings/aliases
              format: strings
    examples:
      - request: {}
        response:
          status: 200
          body:
            items:
              - id: h1
                events: [documents.update]
        expect:
          id: h1
`

// TestAStringsFieldMapsAListOfStrings checks the strings format: each element
// of the reply list maps to the output field, in order.
func TestAStringsFieldMapsAListOfStrings(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, stringsYAML))
	stub.Script("GET", "/v1/hooks", page(`{"items":[{"id":"h1","events":["documents.update","documents.create"],"settings":{"channel":"#gen","aliases":["a","b"]}}]}`))
	rec, err := eng.Read(context.Background(), "example-wiki.hooks.list", domain.NewArgs(nil))
	if err != nil {
		t.Fatalf("Read() error = %v, want nil", err)
	}
	items := rec.Items()
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}
	got, _ := items[0].Get("events")
	list, ok := got.([]any)
	if !ok || len(list) != 2 || list[0] != "documents.update" || list[1] != "documents.create" {
		t.Fatalf("events = %#v, want the two reply strings in order", got)
	}
	settings, ok := items[0].Get("settings")
	if !ok {
		t.Fatal("settings is missing")
	}
	settingsRec, ok := settings.(domain.Record)
	if !ok {
		t.Fatalf("settings = %#v, want a record", settings)
	}
	if got, _ := settingsRec.Get("channel"); got != "#gen" {
		t.Fatalf("settings.channel = %#v, want #gen", got)
	}
	aliases, _ := settingsRec.Get("aliases")
	list, isList := aliases.([]any)
	if !isList || len(list) != 2 || list[0] != "a" || list[1] != "b" {
		t.Fatalf("settings.aliases = %#v, want [a b]", aliases)
	}
}

// TestAStringsFieldWithANonStringElementFailsTheRead checks that a reply list
// holding a non-string under a strings field fails the read, so no partial
// record is returned.
func TestAStringsFieldWithANonStringElementFailsTheRead(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, stringsYAML))
	stub.Script("GET", "/v1/hooks", page(`{"items":[{"id":"h1","events":["documents.update",7]}]}`))
	if _, err := eng.Read(context.Background(), "example-wiki.hooks.list", domain.NewArgs(nil)); !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
	}
}

// TestAStringsFieldHoldingNoListFailsTheRead checks that a reply value that is
// not a list under a strings field fails the read.
func TestAStringsFieldHoldingNoListFailsTheRead(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, stringsYAML))
	stub.Script("GET", "/v1/hooks", page(`{"items":[{"id":"h1","events":{"name":"documents.update"}}]}`))
	if _, err := eng.Read(context.Background(), "example-wiki.hooks.list", domain.NewArgs(nil)); !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
	}
}

// TestAStringsFieldMissingMapsToNull checks that a missing strings field maps
// to null like any other missing optional field, so the record still holds
// every declared field.
func TestAStringsFieldMissingMapsToNull(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, stringsYAML))
	stub.Script("GET", "/v1/hooks", page(`{"items":[{"id":"h1"}]}`))
	rec, err := eng.Read(context.Background(), "example-wiki.hooks.list", domain.NewArgs(nil))
	if err != nil {
		t.Fatalf("Read() error = %v, want nil", err)
	}
	v, ok := rec.Items()[0].Get("events")
	if !ok || v != nil {
		t.Fatalf("events = (%#v, %v), want (nil, true)", v, ok)
	}
}
