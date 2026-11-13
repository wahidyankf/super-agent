package graphql

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vektah/gqlparser/v2/gqlerror"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/ports"
)

// rawSecret stands in for upstream text that could carry the API key. It must never reach a response.
const rawSecret = "sentinel-key-7f3a9c"

// TestPublicMessageMapsEveryPublicError is the drift guard of the safe-error list for GraphQL. Each sentinel of
// domain.PublicErrors() maps to its own fixed text, bare and wrapped with raw detail, and the raw detail never shows.
func TestPublicMessageMapsEveryPublicError(t *testing.T) {
	for _, sentinel := range domain.PublicErrors() {
		t.Run(sentinel.Error(), func(t *testing.T) {
			got, ok := publicMessage(sentinel)
			if !ok || got != sentinel.Error() {
				t.Fatalf("publicMessage(%v) = (%q, %v), want (%q, true)", sentinel, got, ok, sentinel.Error())
			}
			wrapped := fmt.Errorf("%s: %w", rawSecret, sentinel)
			got, ok = publicMessage(wrapped)
			if !ok || got != sentinel.Error() {
				t.Fatalf("wrapped publicMessage = (%q, %v), want (%q, true)", got, ok, sentinel.Error())
			}
			if strings.Contains(got, rawSecret) {
				t.Fatalf("public message %q carries raw detail", got)
			}
		})
	}
}

// TestPublicMessageMapsSourceSelectionErrors checks the messages of the source-selection errors, which carry
// their own text: the unknown and ambiguous errors name the source, and each keeps its text when wrapped.
func TestPublicMessageMapsSourceSelectionErrors(t *testing.T) {
	ambiguous := domain.AmbiguousSourceError{Kind: domain.KindOutline, IDs: []string{"outline", "outline-engineering"}}
	cases := []struct {
		name string
		err  error
		want string
	}{
		{name: "an unknown source", err: domain.UnknownSourceError{ID: "rnd"}, want: `unknown source "rnd"`},
		{name: "an unknown source, wrapped with raw detail", err: fmt.Errorf("upstream %s: %w", rawSecret, domain.UnknownSourceError{ID: "rnd"}), want: `unknown source "rnd"`},
		{name: "an ambiguous source", err: ambiguous, want: "more than one outline source exists (outline, outline-engineering); pass the source argument"},
		{name: "an ambiguous source, wrapped with raw detail", err: fmt.Errorf("upstream %s: %w", rawSecret, ambiguous), want: "more than one outline source exists (outline, outline-engineering); pass the source argument"},
		{name: "no source of the kind", err: domain.ErrNoSourceOfKind, want: "no outline source is configured"},
		{name: "no source of the kind, wrapped with raw detail", err: fmt.Errorf("upstream %s: %w", rawSecret, domain.ErrNoSourceOfKind), want: "no outline source is configured"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := publicMessage(tc.err)
			if !ok || got != tc.want {
				t.Fatalf("publicMessage = (%q, %v), want (%q, true)", got, ok, tc.want)
			}
			if strings.Contains(got, rawSecret) {
				t.Fatalf("public message %q carries raw detail", got)
			}
		})
	}
}

// TestPublicMessageFragmentsOfTheFeatures pins the exact fragments that serve-graphql.feature and
// outline-secret-never-printed.feature assert on.
func TestPublicMessageFragmentsOfTheFeatures(t *testing.T) {
	cases := map[string]error{
		"unsupported source":               domain.ErrUnsupportedSource,
		"invalid document ID":              domain.ErrInvalidDocumentID,
		"Outline rejected the API key":     domain.ErrKeyRejected,
		"Outline denied access":            domain.ErrAccessDenied,
		"Outline returned an error":        domain.ErrUpstreamFailure,
		"Outline could not be reached":     domain.ErrUnreachable,
		"rate limited":                     domain.ErrSourceRateLimited,
		"unexpected response from Outline": domain.ErrUnexpectedResponse,
	}
	for fragment, sentinel := range cases {
		t.Run(fragment, func(t *testing.T) {
			got, ok := publicMessage(fmt.Errorf("%w: detail", sentinel))
			if !ok || !strings.Contains(got, fragment) {
				t.Fatalf("publicMessage = (%q, %v), want a message containing %q", got, ok, fragment)
			}
		})
	}
}

// TestPublicMessageNamesTheMissingVariableOnly checks that the missing-key error names the variable and no value.
func TestPublicMessageNamesTheMissingVariableOnly(t *testing.T) {
	got, ok := publicMessage(domain.MissingKeyError{Variable: "EXAMPLE_OUTLINE_API_KEY"})
	if !ok || got != "EXAMPLE_OUTLINE_API_KEY is not set" {
		t.Fatalf("publicMessage = (%q, %v), want the variable name and the missing text", got, ok)
	}
}

// TestPublicMessageHidesUnknownErrors checks that an error outside the typed set becomes the generic
// message, so its text never leaves the server.
func TestPublicMessageHidesUnknownErrors(t *testing.T) {
	got, ok := publicMessage(fmt.Errorf("disk on fire %s", rawSecret))
	if ok {
		t.Fatalf("publicMessage reported a typed message %q for an unknown error", got)
	}
	if got != internalMessage {
		t.Fatalf("publicMessage = %q, want %q", got, internalMessage)
	}
}

// TestErrorPresenter checks the presenter used for every resolver error: a GraphQL error from gqlgen
// keeps its message, and a core error is mapped to its typed message.
func TestErrorPresenter(t *testing.T) {
	present := newErrorPresenter(discardLogger())
	t.Run("a gqlgen validation error keeps its message", func(t *testing.T) {
		got := present(context.Background(), &gqlerror.Error{Message: `Cannot query field "children" on type "TreeNode".`})
		if got.Message != `Cannot query field "children" on type "TreeNode".` {
			t.Fatalf("message = %q", got.Message)
		}
	})
	t.Run("a core error is mapped, not copied", func(t *testing.T) {
		got := present(context.Background(), fmt.Errorf("body %s: %w", rawSecret, domain.ErrKeyRejected))
		if got.Message != domain.ErrKeyRejected.Error() {
			t.Fatalf("message = %q, want %q", got.Message, domain.ErrKeyRejected.Error())
		}
	})
	t.Run("an unknown error becomes the generic message", func(t *testing.T) {
		got := present(context.Background(), errors.New("raw "+rawSecret))
		if got.Message != internalMessage || strings.Contains(got.Message, rawSecret) {
			t.Fatalf("message = %q, want %q", got.Message, internalMessage)
		}
	})
}

// panicGateway is a source gateway whose every call panics with the value it holds.
type panicGateway struct{ value any }

func (g panicGateway) ListCollections(context.Context) ([]domain.Collection, error) {
	panic(g.value)
}

func (g panicGateway) CollectionTree(context.Context, domain.CollectionID) ([]domain.TreeNode, error) {
	panic(g.value)
}

func (g panicGateway) Document(context.Context, domain.DocumentID) (domain.SourceDocument, error) {
	panic(g.value)
}

// servedWithLog serves gateway through the full handler, logging to logs, and sends one query to it.
func servedWithLog(t *testing.T, gateway ports.SourceGateway, logs *bytes.Buffer, query string) *httptest.ResponseRecorder {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(logs, nil))
	h, err := NewHandler(
		Deps{Service: oneOutlineSources(gateway), Logger: logger},
		WithListener(tcpListener("127.0.0.1", testPort)),
	)
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	return sendTo(t, h, graphqlPOST(query))
}

// TestResolverPanicLeaksNothing checks that a panic in a resolver reaches the caller as the generic message, and
// that the log names the panic value's type while the value and its text stay out of both the response and the log.
func TestResolverPanicLeaksNothing(t *testing.T) {
	cases := map[string]any{
		"a string carrying the key": rawSecret,
		"an error carrying the key": errors.New("body " + rawSecret),
	}
	for name, value := range cases {
		t.Run(name, func(t *testing.T) {
			var logs bytes.Buffer
			rec := servedWithLog(t, panicGateway{value: value}, &logs, `{ collections { id } }`)
			assertErrorContains(t, decode(t, rec), internalMessage)
			if strings.Contains(rec.Body.String(), rawSecret) {
				t.Fatalf("response body exposes the panic value: %s", rec.Body.String())
			}
			out := logs.String()
			if !strings.Contains(out, "graphql resolver panicked") {
				t.Fatalf("log %q does not record the panic", out)
			}
			wantType := fmt.Sprintf("panic_type=%T", value)
			if !strings.Contains(out, wantType) {
				t.Fatalf("log %q does not carry %s", out, wantType)
			}
			if strings.Contains(out, rawSecret) {
				t.Fatalf("log %q holds the panic value", out)
			}
		})
	}
}

// TestUnknownResolverErrorIsLoggedByTypeOnly checks that an unknown resolver error reaches the caller as the generic
// message, and that the log names its type while its raw text stays out of both the response and the log.
func TestUnknownResolverErrorIsLoggedByTypeOnly(t *testing.T) {
	var logs bytes.Buffer
	gw := &countingGateway{err: errors.New("internal detail " + rawSecret)}
	rec := servedWithLog(t, gw, &logs, `{ collections { id } }`)
	assertErrorContains(t, decode(t, rec), internalMessage)
	if strings.Contains(rec.Body.String(), rawSecret) {
		t.Fatalf("response body exposes the raw error: %s", rec.Body.String())
	}
	out := logs.String()
	if !strings.Contains(out, "graphql resolver failed") || !strings.Contains(out, "error_type=*errors.errorString") {
		t.Fatalf("log %q does not record the failure by type", out)
	}
	if strings.Contains(out, rawSecret) {
		t.Fatalf("log %q holds the raw error text", out)
	}
}
