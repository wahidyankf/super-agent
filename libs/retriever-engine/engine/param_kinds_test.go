package engine_test

import (
	"context"
	_ "embed"
	"errors"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// paramKindsYAML is the spec fixture of the parameter-kind tests: a write with an Icon parameter, one with a URLList
// parameter, and one with an EventList parameter. It names example hosts only.
//
//go:embed testdata/param_kinds.yml
var paramKindsYAML string

// TestAnInvalidIconValueIsRefusedBeforeAnyRequest checks that a blank icon fails with the catalog refusal, and that
// the refusal happens before the request leaves: the engine binds every spec argument with the catalog rules, so the
// Icon check that the Go gateway runs also runs here, with no new power.
func TestAnInvalidIconValueIsRefusedBeforeAnyRequest(t *testing.T) {
	eng, stub, _ := writeEngine(t, paramKindsYAML)
	args := map[string]any{"title": "Weekly", "icon": "   "}
	_, err := eng.Write(context.Background(), "example-wiki.templates.create", domain.NewArgs(args))
	if !errors.Is(err, domain.ErrInvalidIcon) {
		t.Fatalf("err = %v, want ErrInvalidIcon", err)
	}
	if n := len(stub.Requests()); n != 0 {
		t.Fatalf("a refused call sent %d requests", n)
	}
}

// TestAnInvalidURLListValueIsRefusedBeforeAnyRequest checks that a redirect address that is not HTTPS fails with the
// catalog refusal, and that the refusal happens before the request leaves.
func TestAnInvalidURLListValueIsRefusedBeforeAnyRequest(t *testing.T) {
	eng, stub, _ := writeEngine(t, paramKindsYAML)
	args := map[string]any{"name": "App", "redirectUris": []any{"http://app.example.test/callback"}}
	_, err := eng.Write(context.Background(), "example-wiki.clients.create", domain.NewArgs(args))
	if !errors.Is(err, domain.ErrInvalidURL) {
		t.Fatalf("err = %v, want ErrInvalidURL", err)
	}
	if n := len(stub.Requests()); n != 0 {
		t.Fatalf("a refused call sent %d requests", n)
	}
}

// TestAnInvalidEventListValueIsRefusedBeforeAnyRequest checks that an event name that is neither * nor dotted fails
// with the catalog refusal, and that the refusal happens before the request leaves.
func TestAnInvalidEventListValueIsRefusedBeforeAnyRequest(t *testing.T) {
	eng, stub, _ := writeEngine(t, paramKindsYAML)
	args := map[string]any{"name": "Hook", "events": []any{"bogus"}}
	_, err := eng.Write(context.Background(), "example-wiki.hooks.create", domain.NewArgs(args))
	if !errors.Is(err, domain.ErrInvalidName) {
		t.Fatalf("err = %v, want ErrInvalidName", err)
	}
	if n := len(stub.Requests()); n != 0 {
		t.Fatalf("a refused call sent %d requests", n)
	}
}

// TestValidIconURLListAndEventListValuesAreSent checks that values the catalog accepts reach the replay stub: the stub
// keys each canned reply by the request the engine renders, so a run proves the replay support of the three kinds.
func TestValidIconURLListAndEventListValuesAreSent(t *testing.T) {
	eng, stub, _ := writeEngine(t, paramKindsYAML)
	valid := []struct {
		op   string
		args map[string]any
		want string
	}{
		{"example-wiki.templates.create", map[string]any{"title": "Weekly", "icon": "star"}, "doc-2"},
		{"example-wiki.clients.create", map[string]any{"name": "App", "redirectUris": []any{"https://app.example.test/callback"}}, "doc-3"},
		{"example-wiki.hooks.create", map[string]any{"name": "Hook", "events": []any{"*"}}, "doc-4"},
	}
	for _, c := range valid {
		rec, err := eng.Write(context.Background(), domain.OperationID(c.op), domain.NewArgs(c.args))
		if err != nil {
			t.Fatalf("Write %s: %v", c.op, err)
		}
		if got := rec.Str("id"); got != c.want {
			t.Fatalf("Write %s id = %q, want %q", c.op, got, c.want)
		}
	}
	if n := len(stub.Requests()); n != len(valid) {
		t.Fatalf("requests = %d, want %d", n, len(valid))
	}
}
