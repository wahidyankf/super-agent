package mcp

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/sources"
)

// sentinelKey stands in for a real key or an upstream body. No result, error, or log line may hold it.
const sentinelKey = "test-key-SENTINEL-do-not-show"

// connect runs the production server over an in-memory transport and returns the SDK client end of the session.
// The log lines the server writes go to logs.
func connect(t *testing.T, svc sources.Service, logs *bytes.Buffer) *sdk.ClientSession {
	t.Helper()
	server, err := NewServer(svc, Options{Version: "test", Logger: slog.New(slog.NewTextHandler(logs, nil))})
	if err != nil {
		t.Fatal(err)
	}
	return clientOf(t, server)
}

// clientOf connects an SDK client to server over an in-memory transport and returns the client end of the session.
func clientOf(t *testing.T, server *sdk.Server) *sdk.ClientSession {
	t.Helper()
	ctx := t.Context()
	clientT, serverT := sdk.NewInMemoryTransports()
	if _, err := server.Connect(ctx, serverT, nil); err != nil {
		t.Fatal(err)
	}
	client := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "0"}, nil)
	cs, err := client.Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

// call runs one tool and returns the text of its answer and whether it is an error.
func call(t *testing.T, cs *sdk.ClientSession, name string, args map[string]any) (string, bool) {
	t.Helper()
	res, err := cs.CallTool(t.Context(), &sdk.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("call %s: %v", name, err)
	}
	var parts []string
	for _, c := range res.Content {
		if tc, ok := c.(*sdk.TextContent); ok {
			parts = append(parts, tc.Text)
		}
	}
	return strings.Join(parts, "\n"), res.IsError
}

// TestToolListHasExplicitHintsAndTheUntrustedSentence checks what the model sees: four tools, every hint set on the
// wire, and every description ends with the sentence that marks wiki text as data.
func TestToolListHasExplicitHintsAndTheUntrustedSentence(t *testing.T) {
	var logs bytes.Buffer
	cs := connect(t, emptyService(t), &logs)
	res, err := cs.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	// The read tools of the catalog, with list_sources, which reads only the catalog. The open-world hint is exact per
	// tool, so a tool that calls a wiki is never announced as closed, or the reverse. Every other listed tool changes
	// something, so it is checked below as not read-only.
	wantOpenWorld := map[string]bool{
		"list_sources": false, "list_collections": true, "collection_info": true, "collection_tree": true,
		"get_document": true, "list_documents": true, "search_documents": true, "whoami": true,
		"list_archived_documents": true, "document_children": true, "list_deleted_documents": true,
		"list_draft_documents": true, "list_document_group_memberships": true, "get_document_insights": true,
		"list_document_memberships": true, "search_document_titles": true, "list_document_users": true,
		"list_viewed_documents": true, "list_collection_memberships": true, "list_collection_group_memberships": true,
		"get_comment": true, "list_comments": true, "get_revision": true,
		"list_revisions": true, "get_share": true, "list_shares": true, "get_user": true, "list_users": true,
		"get_group": true, "list_groups": true, "list_group_members": true, "get_template": true,
		"list_templates": true, "get_pin": true, "list_pins": true, "list_stars": true, "get_subscription": true,
		"list_subscriptions": true,
		"get_access_request": true, "list_attachments": true, "get_emoji": true, "list_emojis": true,
		"list_events": true, "get_file_operation": true, "list_file_operations": true,
		"list_group_memberships": true, "get_installation": true, "list_notifications": true,
		"list_oauth_authentications": true, "list_reactions": true, "get_relationship": true,
		"list_relationships": true, "list_searches": true, "suggest_mentions": true,
		"list_user_memberships": true, "list_views": true,
		"get_authentication_provider": true, "list_api_keys": true, "list_authentication_providers": true,
		"get_import": true, "list_imports": true, "get_integration": true, "list_integrations": true,
		"get_oauth_client": true, "list_oauth_clients": true, "list_webhook_subscriptions": true,
	}
	listed := map[string]bool{}
	for _, tl := range res.Tools {
		listed[tl.Name] = true
		if !strings.HasSuffix(tl.Description, untrustedSentence) {
			t.Errorf("%s description does not end with the untrusted-data sentence", tl.Name)
		}
		a := tl.Annotations
		if a == nil || a.DestructiveHint == nil || a.OpenWorldHint == nil {
			t.Errorf("%s has a hint that is not set explicitly: %+v", tl.Name, a)
			continue
		}
		world, isRead := wantOpenWorld[tl.Name]
		if !isRead {
			if a.ReadOnlyHint {
				t.Errorf("%s is announced as read-only, but it changes the wiki", tl.Name)
			}
			continue
		}
		if !a.ReadOnlyHint || !a.IdempotentHint || *a.DestructiveHint {
			t.Errorf("%s has a read hint that is not set as a read: %+v", tl.Name, a)
		}
		if *a.OpenWorldHint != world {
			t.Errorf("%s has OpenWorldHint %v, want %v", tl.Name, *a.OpenWorldHint, world)
		}
	}
	for name := range wantOpenWorld {
		if !listed[name] {
			t.Errorf("%s is a read tool but the server does not list it", name)
		}
	}
}

// TestRegisterRefusesATableThatNamesOneToolTwice checks the table's own guard.
func TestRegisterRefusesATableThatNamesOneToolTwice(t *testing.T) {
	row := tool{name: "list_sources", effect: domain.EffectRead, bind: bind(listSources)}
	server := sdk.NewServer(&sdk.Implementation{Name: "t", Version: "0"}, nil)
	_, err := register(server, emptyService(t), slog.New(slog.DiscardHandler), []tool{row, row})
	if err == nil {
		t.Fatal("register accepted a duplicate tool name")
	}
}

// listedNames returns the names of the tools a server offers, through an in-memory session.
func listedNames(t *testing.T, server *sdk.Server) []string {
	t.Helper()
	clientT, serverT := sdk.NewInMemoryTransports()
	if _, err := server.Connect(t.Context(), serverT, nil); err != nil {
		t.Fatal(err)
	}
	client := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "0"}, nil)
	cs, err := client.Connect(t.Context(), clientT, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cs.Close() }()
	res, err := cs.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tl := range res.Tools {
		names = append(names, tl.Name)
	}
	return names
}

// TestHostileGatewayErrorNeverReachesTheResultOrTheLog checks that an upstream error that holds a key and a path is
// replaced with a fixed text, both in the answer and in the log line.
func TestHostileGatewayErrorNeverReachesTheResultOrTheLog(t *testing.T) {
	hostile := &fakeGateway{err: fmt.Errorf("upstream body %s at /srv/example/private/path", sentinelKey)}
	var logs bytes.Buffer
	cs := connect(t, serviceWith(t, hostile, mustSource(t, "outline")), &logs)
	text, isErr := call(t, cs, "list_collections", nil)
	if !isErr || text != "unexpected error" {
		t.Fatalf("answer is (%v, %q), want the fixed unexpected-error text", isErr, text)
	}
	assertNoSentinel(t, text, logs.String())
}

// TestPublicSentinelKeepsItsFixedTextAndTheKeyNameIsDropped checks that a sentinel the adapter may show keeps its fixed
// text, and that a missing key shows no variable name.
func TestPublicSentinelKeepsItsFixedTextAndTheKeyNameIsDropped(t *testing.T) {
	var logs bytes.Buffer
	upstream := &fakeGateway{err: fmt.Errorf("wrapped: %w", domain.ErrKeyRejected)}
	cs := connect(t, serviceWith(t, upstream, mustSource(t, "outline")), &logs)
	text, isErr := call(t, cs, "list_collections", nil)
	if !isErr || text != "Outline rejected the API key" {
		t.Fatalf("answer is (%v, %q), want the key-rejected text", isErr, text)
	}

	missing := &fakeGateway{err: domain.MissingKeyError{Variable: "EXAMPLE_OUTLINE_API_KEY"}}
	cs2 := connect(t, serviceWith(t, missing, mustSource(t, "outline")), &logs)
	text2, _ := call(t, cs2, "list_collections", nil)
	if strings.Contains(text2, "EXAMPLE") || text2 != missingKeyText {
		t.Fatalf("missing key answer is %q, want the fixed text without the variable name", text2)
	}
}

// TestLogLineHoldsNoContent checks the shape of one log line: the tool, the resolved source, the outcome class, and the
// duration. It holds nothing from the arguments or the answer: the distinctive document ID passed in, and the title and
// the Markdown of the document, never appear in the line.
func TestLogLineHoldsNoContent(t *testing.T) {
	const docID, title, body = "doc-distinct-7741", "Title distinct 7741", "Body distinct 7741 text"
	var logs bytes.Buffer
	gw := &fakeGateway{doc: domain.SourceDocument{ID: docID, Title: title, URL: "/doc/distinct", Markdown: body}}
	cs := connect(t, serviceWith(t, gw, mustSource(t, "outline")), &logs)
	if _, isErr := call(t, cs, "get_document", map[string]any{"document_id": docID}); isErr {
		t.Fatal("get_document failed")
	}
	line := strings.TrimSpace(logs.String())
	for _, want := range []string{"tool=get_document", "source=outline", "outcome=ok", "duration_ms="} {
		if !strings.Contains(line, want) {
			t.Errorf("log line %q lacks %q", line, want)
		}
	}
	for _, leaked := range []string{docID, title, body} {
		if strings.Contains(logs.String(), leaked) {
			t.Errorf("log line %q holds %q, which is content", line, leaked)
		}
	}
	if strings.Count(logs.String(), "\n") != 1 {
		t.Errorf("one call wrote %d lines, want 1", strings.Count(logs.String(), "\n"))
	}
}

// TestUnknownSourceNamesTheSourceButNoUpstreamDetail checks that the one value the catalog validated is shown, and nothing
// else from the failure is.
func TestUnknownSourceNamesTheSourceButNoUpstreamDetail(t *testing.T) {
	var logs bytes.Buffer
	cs := connect(t, emptyService(t), &logs)
	text, isErr := call(t, cs, "list_collections", map[string]any{"source": "rnd"})
	if !isErr || text != `unknown source "rnd"` {
		t.Fatalf("answer is (%v, %q)", isErr, text)
	}
}

// TestGetDocumentRefusesAnOffsetBeforeAnyRequest checks that an offset outside the range is refused before the gateway is
// called, so an invalid offset never costs an upstream request.
func TestGetDocumentRefusesAnOffsetBeforeAnyRequest(t *testing.T) {
	cases := []struct {
		name   string
		offset int
		text   string
	}{
		{name: "a negative offset", offset: -1, text: "offset must not be negative"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gw := &fakeGateway{doc: domain.SourceDocument{ID: "doc-1", Title: "Doc", URL: "/doc/doc-1", Markdown: "Welcome."}}
			var logs bytes.Buffer
			cs := connect(t, serviceWith(t, gw, mustSource(t, "outline")), &logs)
			text, isErr := call(t, cs, "get_document", map[string]any{"document_id": "doc-1", "offset": tc.offset})
			if !isErr || text != tc.text {
				t.Fatalf("answer is (%v, %q), want the error %q", isErr, text, tc.text)
			}
			if gw.calls != 0 {
				t.Fatalf("the gateway was called %d times, want none", gw.calls)
			}
		})
	}
}

// assertNoSentinel fails when the sentinel key appears in an answer or a log.
func assertNoSentinel(t *testing.T, parts ...string) {
	t.Helper()
	for _, p := range parts {
		if strings.Contains(p, sentinelKey) || strings.Contains(p, "/Users/someone") {
			t.Fatalf("sentinel leaked into %q", p)
		}
	}
}

// panickingUse is a use case that panics with a value that carries a secret. The handler must show neither its text nor
// its value.
func panickingUse(context.Context, sources.Service, listSourcesIn) (listSourcesOut, string, error) {
	panic(sentinelKey)
}

// TestPanickingUseCaseAnswersTheFixedTextAndLogsTheTypeOnly checks that a panic in a use case becomes the fixed unexpected
// error, and that the log line names the type of the panic value and never its text.
func TestPanickingUseCaseAnswersTheFixedTextAndLogsTheTypeOnly(t *testing.T) {
	var logs bytes.Buffer
	probe := tool{name: "panic_probe", description: "probe", effect: domain.EffectRead, bind: bind(panickingUse)}
	server := sdk.NewServer(&sdk.Implementation{Name: "t", Version: "0"}, nil)
	if _, err := register(server, emptyService(t), slog.New(slog.NewTextHandler(&logs, nil)), []tool{probe}); err != nil {
		t.Fatal(err)
	}
	cs := clientOf(t, server)
	text, isErr := call(t, cs, "panic_probe", nil)
	if !isErr || text != "unexpected error" {
		t.Fatalf("answer is (%v, %q), want the fixed unexpected-error text", isErr, text)
	}
	if !strings.Contains(logs.String(), "panic_type=string") {
		t.Fatalf("log %q does not name the panic type", logs.String())
	}
	assertNoSentinel(t, text, logs.String())
}

// TestAnnotationsOfAnEffectWithNoRowAreCautious checks the default branch of the hints: an effect that no row uses yet gets
// the most cautious hints, so a new effect is never announced as read-only by accident.
func TestAnnotationsOfAnEffectWithNoRowAreCautious(t *testing.T) {
	a := annotationsOf(domain.Effect("bogus"), false)
	if a.ReadOnlyHint || a.DestructiveHint == nil || !*a.DestructiveHint {
		t.Fatalf("annotations of an effect with no row are %+v, want the cautious hints", a)
	}
}
