package engine_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine/replay"
)

// boolMapDoc returns a read whose notificationSettings is an open-key map with closed bool values: every event
// type the wiki returns is kept verbatim, sorted by event name. The mapping is reply-shape work shared by reads
// and writes, so the mapping tests run it as a read, where a refusal keeps its unexpected-response code and
// dotted path.
func boolMapDoc() string {
	return "version: 1\n" +
		"kind: example-wiki\n" +
		"summary: Reads a user with notification settings.\n" +
		"endpoint:\n" +
		"  base_path: /v1\n" +
		"  host_suffix:\n" +
		"    - .example.test\n" +
		"auth:\n" +
		"  type: bearer\n" +
		"  credential_env: EXAMPLE_WIKI_API_KEY\n" +
		"operations:\n" +
		"  - id: example-wiki.users.get\n" +
		"    summary: Read a user with notification settings.\n" +
		"    access: read\n" +
		"    method: GET\n" +
		"    path: /user\n" +
		"    response:\n" +
		"      fields:\n" +
		"        id:\n" +
		"          pointer: /data/id\n" +
		"          required: true\n" +
		"          format: uuid\n" +
		"        notificationSettings:\n" +
		"          pointer: /data/notificationSettings\n" +
		"          fallback_pointer: /data/settings\n" +
		"          bool_map: true\n" +
		"    examples:\n" +
		"      - request: {}\n" +
		"        response:\n" +
		"          status: 200\n" +
		"          body:\n" +
		"            data:\n" +
		"              id: ex-0001\n" +
		"        expect:\n" +
		"          id: ex-0001\n"
}

// boolMapRequiredDoc returns a read whose bool map is required: a missing map fails the read.
func boolMapRequiredDoc() string {
	return "version: 1\n" +
		"kind: example-wiki\n" +
		"summary: Reads a user with required notification settings.\n" +
		"endpoint:\n" +
		"  base_path: /v1\n" +
		"  host_suffix:\n" +
		"    - .example.test\n" +
		"auth:\n" +
		"  type: bearer\n" +
		"  credential_env: EXAMPLE_WIKI_API_KEY\n" +
		"operations:\n" +
		"  - id: example-wiki.users.get\n" +
		"    summary: Read a user with notification settings.\n" +
		"    access: read\n" +
		"    method: GET\n" +
		"    path: /user\n" +
		"    response:\n" +
		"      fields:\n" +
		"        id:\n" +
		"          pointer: /data/id\n" +
		"          required: true\n" +
		"          format: uuid\n" +
		"        notificationSettings:\n" +
		"          pointer: /data/notificationSettings\n" +
		"          required: true\n" +
		"          bool_map: true\n" +
		"    examples:\n" +
		"      - request: {}\n" +
		"        response:\n" +
		"          status: 200\n" +
		"          body:\n" +
		"            data:\n" +
		"              id: ex-0001\n" +
		"              notificationSettings: {}\n" +
		"        expect:\n" +
		"          id: ex-0001\n"
}

// boolMapWriteDoc returns a write whose notificationSettings is the same open-key map, so a wrong-shaped value
// exercises the write contract: the request left, the reply is unusable, and the outcome is unknown, never retried.
func boolMapWriteDoc() string {
	return "version: 1\n" +
		"kind: example-wiki\n" +
		"summary: Turn on notification settings.\n" +
		"endpoint:\n" +
		"  base_path: /v1\n" +
		"  host_suffix:\n" +
		"    - .example.test\n" +
		"auth:\n" +
		"  type: bearer\n" +
		"  credential_env: EXAMPLE_WIKI_API_KEY\n" +
		"operations:\n" +
		"  - id: example-wiki.users.notifications-subscribe\n" +
		"    summary: Turn on notification settings.\n" +
		"    access: write\n" +
		"    method: POST\n" +
		"    path: /notifications\n" +
		"    response:\n" +
		"      fields:\n" +
		"        id:\n" +
		"          pointer: /data/id\n" +
		"          required: true\n" +
		"          format: uuid\n" +
		"        notificationSettings:\n" +
		"          pointer: /data/notificationSettings\n" +
		"          bool_map: true\n" +
		"    examples:\n" +
		"      - request: {}\n" +
		"        response:\n" +
		"          status: 200\n" +
		"          body:\n" +
		"            data:\n" +
		"              id: ex-0001\n" +
		"        expect:\n" +
		"          id: ex-0001\n"
}

// boolMapEngine builds an engine for the bool-map fixture over a replay stub.
func boolMapEngine(t *testing.T) (*engine.Engine, *replay.Stub) {
	t.Helper()
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, boolMapDoc()))
	return eng, stub
}

// TestBoolMapKeepsEveryKeySorted checks the all-keys-kept rule: the engine keeps every event type the wiki
// returns, verbatim, in sorted order, like the Go projection of the Outline notification writes.
func TestBoolMapKeepsEveryKeySorted(t *testing.T) {
	eng, stub := boolMapEngine(t)
	stub.Script(http.MethodGet, "/v1/user", replay.Reply{Status: http.StatusOK,
		Body: []byte(`{"data":{"id":"123e4567-e89b-12d3-a456-426614174000",` +
			`"notificationSettings":{"documents.update":true,"comments.create":false,"emails.features":true}}}`)})
	rec, err := eng.Read(context.Background(), "example-wiki.users.get", domain.NewArgs(nil))
	if err != nil {
		t.Fatalf("Read = %v, want nil", err)
	}
	settings := rec.Rec("notificationSettings")
	var names []string
	for _, f := range settings.Fields() {
		names = append(names, f.Name)
	}
	want := []string{"comments.create", "documents.update", "emails.features"}
	if len(names) != len(want) {
		t.Fatalf("settings keys = %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("settings keys = %v, want %v", names, want)
		}
	}
	if v, _ := settings.Get("documents.update"); v != true {
		t.Fatalf("documents.update = %v, want true", v)
	}
	if v, _ := settings.Get("comments.create"); v != false {
		t.Fatalf("comments.create = %v, want false", v)
	}
}

// TestBoolMapEmptyMapOk checks the empty map: a wiki with no settings answers an empty record, never a failure.
func TestBoolMapEmptyMapOk(t *testing.T) {
	eng, stub := boolMapEngine(t)
	stub.Script(http.MethodGet, "/v1/user", replay.Reply{Status: http.StatusOK,
		Body: []byte(`{"data":{"id":"123e4567-e89b-12d3-a456-426614174000","notificationSettings":{}}}`)})
	rec, err := eng.Read(context.Background(), "example-wiki.users.get", domain.NewArgs(nil))
	if err != nil {
		t.Fatalf("Read = %v, want nil", err)
	}
	if got := len(rec.Rec("notificationSettings").Fields()); got != 0 {
		t.Fatalf("settings fields = %d, want 0", got)
	}
}

// TestBoolMapNonBoolFailsNamingPath checks the strict rule: a value that is not a flag fails the read as an
// unexpected response naming the dotted path, and the value is never kept. The Go projection drops such a value;
// the engine refuses it instead, which the owner accepted.
func TestBoolMapNonBoolFailsNamingPath(t *testing.T) {
	eng, stub := boolMapEngine(t)
	stub.Script(http.MethodGet, "/v1/user", replay.Reply{Status: http.StatusOK,
		Body: []byte(`{"data":{"id":"123e4567-e89b-12d3-a456-426614174000",` +
			`"notificationSettings":{"documents.update":"yes"}}}`)})
	_, err := eng.Read(context.Background(), "example-wiki.users.get", domain.NewArgs(nil))
	if !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
	}
	if got := err.Error(); !strings.Contains(got, "notificationSettings.documents.update") {
		t.Fatalf("err = %q, want it to name notificationSettings.documents.update", got)
	}
}

// TestBoolMapNonBoolWriteIsAnUnknownOutcomeAndIsNotSentAgain checks the write contract for a wrong-shaped value:
// the request left and the reply is unusable, so the outcome is unknown with the stable code of the Go connectors,
// and the write is sent exactly once, never retried.
func TestBoolMapNonBoolWriteIsAnUnknownOutcomeAndIsNotSentAgain(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, boolMapWriteDoc()))
	stub.Script(http.MethodPost, "/v1/notifications", replay.Reply{Status: http.StatusOK,
		Body: []byte(`{"data":{"id":"123e4567-e89b-12d3-a456-426614174000",` +
			`"notificationSettings":{"documents.update":"yes"}}}`)})
	_, err := eng.Write(context.Background(), "example-wiki.users.notifications-subscribe", domain.NewArgs(nil))
	if !errors.Is(err, domain.ErrWriteOutcomeUnknown) {
		t.Fatalf("err = %v, want ErrWriteOutcomeUnknown", err)
	}
	if got := engine.Code("example-wiki", err); got != "retriever.example-wiki.write_outcome_unknown" {
		t.Fatalf("code = %q, want retriever.example-wiki.write_outcome_unknown", got)
	}
	if n := len(stub.Requests()); n != 1 {
		t.Fatalf("the write caused %d requests, want 1: a write with an unknown outcome is never sent again", n)
	}
}

// TestBoolMapMissingKeyAnswersAnEmptyRecord checks the missing map: a reply without the settings object answers an
// empty record, because the field is optional.
func TestBoolMapMissingKeyAnswersAnEmptyRecord(t *testing.T) {
	eng, stub := boolMapEngine(t)
	stub.Script(http.MethodGet, "/v1/user", replay.Reply{Status: http.StatusOK,
		Body: []byte(`{"data":{"id":"123e4567-e89b-12d3-a456-426614174000"}}`)})
	rec, err := eng.Read(context.Background(), "example-wiki.users.get", domain.NewArgs(nil))
	if err != nil {
		t.Fatalf("Read = %v, want nil", err)
	}
	if got := len(rec.Rec("notificationSettings").Fields()); got != 0 {
		t.Fatalf("settings fields = %d, want 0", got)
	}
}

// TestBoolMapMissingRequiredKeyFails checks the required map: a reply without the settings object fails the read as
// an unexpected response naming the field.
func TestBoolMapMissingRequiredKeyFails(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, boolMapRequiredDoc()))
	stub.Script(http.MethodGet, "/v1/user", replay.Reply{Status: http.StatusOK,
		Body: []byte(`{"data":{"id":"123e4567-e89b-12d3-a456-426614174000"}}`)})
	_, err := eng.Read(context.Background(), "example-wiki.users.get", domain.NewArgs(nil))
	if !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
	}
	if got := err.Error(); !strings.Contains(got, "notificationSettings") {
		t.Fatalf("err = %q, want it to name notificationSettings", got)
	}
}

// TestBoolMapWrongShapeFailsNamingTheField checks the shape rule: a value that is not an object fails the read as
// an unexpected response naming the field, never the value.
func TestBoolMapWrongShapeFailsNamingTheField(t *testing.T) {
	eng, stub := boolMapEngine(t)
	stub.Script(http.MethodGet, "/v1/user", replay.Reply{Status: http.StatusOK,
		Body: []byte(`{"data":{"id":"123e4567-e89b-12d3-a456-426614174000","notificationSettings":["x"]}}`)})
	_, err := eng.Read(context.Background(), "example-wiki.users.get", domain.NewArgs(nil))
	if !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
	}
	if got := err.Error(); !strings.Contains(got, "notificationSettings") {
		t.Fatalf("err = %q, want it to name notificationSettings", got)
	}
}

// TestBoolMapFallbackPointerReadsTheSecondShape checks the dual shape: a reply that lacks the pointer but holds the
// fallback object maps the fallback, so one field maps two reply shapes.
func TestBoolMapFallbackPointerReadsTheSecondShape(t *testing.T) {
	eng, stub := boolMapEngine(t)
	stub.Script(http.MethodGet, "/v1/user", replay.Reply{Status: http.StatusOK,
		Body: []byte(`{"data":{"id":"123e4567-e89b-12d3-a456-426614174000",` +
			`"settings":{"documents.update":true}}}`)})
	rec, err := eng.Read(context.Background(), "example-wiki.users.get", domain.NewArgs(nil))
	if err != nil {
		t.Fatalf("Read = %v, want nil", err)
	}
	if v, _ := rec.Rec("notificationSettings").Get("documents.update"); v != true {
		t.Fatalf("documents.update = %v, want true", v)
	}
}
