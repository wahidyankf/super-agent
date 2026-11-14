package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/sources"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/sources/operations"
)

// The tests in this file reach the branches of the operation tools that the protocol tests do not: each one runs a tool
// through the client session, or calls one helper directly where the SDK validates the input before the handler runs.

const (
	gapDocumentID   = "d0c1d0c1-0000-4000-8000-000000000001"
	gapCollectionID = "1c1c1c1c-0000-4000-8000-000000000001"
)

// adminFake is a gateway that also runs the admin operations. It records each admin call and answers with the configured
// reply or error.
type adminFake struct {
	*fakeGateway
	adminReply domain.Record
	adminErr   error
	admins     []domain.OperationID
}

// Write records the call and answers with the configured reply or error.
func (g *adminFake) Write(_ context.Context, op domain.OperationID, _ domain.Args) (domain.Record, error) {
	g.admins = append(g.admins, op)
	return g.adminReply, g.adminErr
}

// panicGateway panics on every read, so a test can check that a panic in a catalog call reaches the model as the fixed
// text only.
type panicGateway struct{ *fakeGateway }

// Read panics with a value that must never reach the model.
func (panicGateway) Read(context.Context, domain.OperationID, domain.Args) (domain.Record, error) {
	panic("internal detail")
}

// writableOutline returns a service with one outline source, served by gw.
func writableOutline(t *testing.T, gw *fakeGateway) sources.Service {
	t.Helper()
	return serviceWith(t, gw, mustSource(t, "outline"))
}

// rowForTool returns the catalog row whose MCP tool has the given name.
func rowForTool(t *testing.T, name string) operations.Row {
	t.Helper()
	for _, row := range operationTools() {
		if row.Tool == name {
			return row
		}
	}
	t.Fatalf("no catalog row for tool %s", name)
	return operations.Row{}
}

// requestOf returns a call request whose arguments are the given JSON text.
func requestOf(raw string) *sdk.CallToolRequest {
	return &sdk.CallToolRequest{Params: &sdk.CallToolParamsRaw{Arguments: json.RawMessage(raw)}}
}

// TestAPanicInACatalogCallAnswersTheFixedText checks that a panic in a catalog read reaches the model as the unexpected
// error only, and that the panic value never does.
func TestAPanicInACatalogCallAnswersTheFixedText(t *testing.T) {
	svc := serviceWith(t, panicGateway{&fakeGateway{}}, mustSource(t, "outline"))
	var logs bytes.Buffer
	cs := connect(t, svc, &logs)
	text, isErr := call(t, cs, "list_collections", nil)
	if !isErr || text != "unexpected error" || strings.Contains(text, "internal detail") {
		t.Fatalf("a panic must answer the fixed text only, got %q (error %v)", text, isErr)
	}
}

// TestAnAmbiguousSourceNamesTheChoice checks that two outline sources with no source argument are refused, and that the
// refusal names the way out.
func TestAnAmbiguousSourceNamesTheChoice(t *testing.T) {
	svc := serviceWith(t, &fakeGateway{}, mustSource(t, "outline-a"), mustSource(t, "outline-b"))
	var logs bytes.Buffer
	text, isErr := call(t, connect(t, svc, &logs), "list_collections", nil)
	if !isErr || !strings.Contains(text, "more than one outline source exists") || !strings.Contains(text, "pass the source argument") {
		t.Fatalf("an ambiguous source must be refused with its way out, got %q (error %v)", text, isErr)
	}
}

// TestTheTypedGatewayErrorsKeepTheirPublicText checks that each typed error a gateway returns reaches the model with its
// own public text.
func TestTheTypedGatewayErrorsKeepTheirPublicText(t *testing.T) {
	unavailable := domain.OperationUnavailableError{Method: "collections.list"}
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"operation unavailable", unavailable, unavailable.Error()},
		{"no source of the kind", domain.ErrNoSourceOfKind, "no outline source is configured"},
	}
	for _, c := range cases {
		svc := serviceWith(t, &fakeGateway{err: c.err}, mustSource(t, "outline"))
		var logs bytes.Buffer
		text, isErr := call(t, connect(t, svc, &logs), "list_collections", nil)
		if !isErr || text != c.want {
			t.Errorf("%s: got %q (error %v), want %q", c.name, text, isErr, c.want)
		}
	}
}

// TestAWriteWhoseGatewayFailsIsAFailure checks that a write the wiki refuses reaches the model as the unexpected error,
// and that the raw upstream text does not.
func TestAWriteWhoseGatewayFailsIsAFailure(t *testing.T) {
	gw := &fakeGateway{err: errors.New("upstream detail")}
	var logs bytes.Buffer
	text, isErr := call(t, connect(t, writableOutline(t, gw), &logs), "create_document", map[string]any{
		"source": "outline", "title": "Synthetic", "text": "Synthetic body", "collection_id": gapCollectionID,
	})
	if !isErr || text != "unexpected error" || strings.Contains(text, "upstream detail") {
		t.Fatalf("a failed write must answer the fixed text, got %q (error %v)", text, isErr)
	}
}

// TestAWriteToAnUnknownSourceIsRefusedBeforeAnyRequest checks that a write naming a source the catalog does not hold is
// refused, and that no request reaches the wiki.
func TestAWriteToAnUnknownSourceIsRefusedBeforeAnyRequest(t *testing.T) {
	gw := &fakeGateway{}
	var logs bytes.Buffer
	text, isErr := call(t, connect(t, writableOutline(t, gw), &logs), "create_document", map[string]any{
		"title": "Synthetic", "text": "Synthetic body", "collection_id": gapCollectionID, "source": "missing",
	})
	if !isErr || text != `unknown source "missing"` || gw.calls != 0 {
		t.Fatalf("an unknown source must be refused with no request, got %q (error %v, calls %d)", text, isErr, gw.calls)
	}
}

// TestAWriteAnswersWithTheRecordItWrote checks that a write answers with the record the wiki returned.
func TestAWriteAnswersWithTheRecordItWrote(t *testing.T) {
	gw := &fakeGateway{writeReply: docReply()}
	var logs bytes.Buffer
	text, isErr := call(t, connect(t, writableOutline(t, gw), &logs), "create_document", map[string]any{
		"source": "outline", "title": "Synthetic", "text": "Synthetic body", "collection_id": gapCollectionID,
	})
	if isErr || !strings.Contains(text, `"title":"Synthetic"`) || len(gw.writes) != 1 {
		t.Fatalf("a write must answer with its record and make one request, got %q (error %v, writes %d)", text, isErr, len(gw.writes))
	}
}

// TestAWriteWithAnInvalidIdentifierIsRefused checks that a write whose identifier is malformed is refused before any
// request.
func TestAWriteWithAnInvalidIdentifierIsRefused(t *testing.T) {
	gw := &fakeGateway{}
	var logs bytes.Buffer
	text, isErr := call(t, connect(t, writableOutline(t, gw), &logs), "create_document", map[string]any{
		"title": "Synthetic", "text": "Synthetic body", "collection_id": "not-an-identifier",
	})
	if !isErr || text == "" || gw.calls != 0 {
		t.Fatalf("a malformed identifier must be refused with no request, got %q (error %v, calls %d)", text, isErr, gw.calls)
	}
}

// TestAnUpdateThatDoesNotReplaceTextNeedsNoRevision checks that an update with no text is sent without a revision.
func TestAnUpdateThatDoesNotReplaceTextNeedsNoRevision(t *testing.T) {
	gw := &fakeGateway{writeReply: docReply()}
	var logs bytes.Buffer
	_, isErr := call(t, connect(t, writableOutline(t, gw), &logs), "update_document", map[string]any{
		"document_id": gapDocumentID, "source": "outline", "title": "Renamed",
	})
	if isErr || len(gw.writes) != 1 {
		t.Fatalf("an update without text must be sent, got error %v, writes %d", isErr, len(gw.writes))
	}
}

// TestAnAppendNeedsNoRevision checks that an append to a document is sent without a revision.
func TestAnAppendNeedsNoRevision(t *testing.T) {
	gw := &fakeGateway{writeReply: docReply()}
	var logs bytes.Buffer
	_, isErr := call(t, connect(t, writableOutline(t, gw), &logs), "update_document", map[string]any{
		"document_id": gapDocumentID, "source": "outline", "text": "Synthetic addition", "edit_mode": "append",
	})
	if isErr || len(gw.writes) != 1 {
		t.Fatalf("an append must be sent without a revision, got error %v, writes %d", isErr, len(gw.writes))
	}
}

// adminFakeService returns a service with one outline source, served by gw.
func adminFakeService(t *testing.T, gw *adminFake) sources.Service {
	t.Helper()
	gw.fakeGateway.doc = domain.SourceDocument{ID: gapDocumentID, Title: "Synthetic", URL: "https://docs.example.test/doc", CollectionID: gapCollectionID}
	return serviceWith(t, gw, mustSource(t, "outline"))
}

// TestADestructiveCallAnswersWithItsReply checks that an admin call answers with the record the wiki returned, and makes one
// call.
func TestADestructiveCallAnswersWithItsReply(t *testing.T) {
	gw := &adminFake{fakeGateway: &fakeGateway{}, adminReply: domain.NewRecord(domain.Field{Name: "ok", Value: true})}
	var logs bytes.Buffer
	text, isErr := call(t, connect(t, adminFakeService(t, gw), &logs), "delete_document", map[string]any{"document_id": gapDocumentID, "source": "outline"})
	if isErr || !strings.Contains(text, `"ok":true`) || len(gw.admins) != 1 {
		t.Fatalf("an admin call must answer with its reply and make one call, got %q (error %v, admin calls %d)", text, isErr, len(gw.admins))
	}
}

// TestADestructiveCallWhoseWikiFailsIsAFailure checks that an admin call the wiki refuses reaches the model as the fixed text.
func TestADestructiveCallWhoseWikiFailsIsAFailure(t *testing.T) {
	gw := &adminFake{fakeGateway: &fakeGateway{}, adminErr: errors.New("upstream detail")}
	var logs bytes.Buffer
	text, isErr := call(t, connect(t, adminFakeService(t, gw), &logs), "delete_document", map[string]any{"document_id": gapDocumentID, "source": "outline"})
	if !isErr || text != "unexpected error" || strings.Contains(text, "upstream detail") {
		t.Fatalf("a failed admin call must answer the fixed text, got %q (error %v)", text, isErr)
	}
}

// TestAGetDocumentPastTheEndIsRefused checks that an offset beyond the body is refused with its public text.
func TestAGetDocumentPastTheEndIsRefused(t *testing.T) {
	gw := &fakeGateway{doc: domain.SourceDocument{ID: gapDocumentID, Title: "Synthetic", Markdown: "Synthetic body"}}
	var logs bytes.Buffer
	text, isErr := call(t, connect(t, serviceWith(t, gw, mustSource(t, "outline")), &logs), "get_document",
		map[string]any{"document_id": gapDocumentID, "source": "outline", "offset": 9999})
	if !isErr || text != errOffsetPastEnd.Error() {
		t.Fatalf("an offset past the end must be refused, got %q (error %v)", text, isErr)
	}
}

// TestALongBodyIsPagedWithANextOffset checks that a body over the page cap is cut, and names the offset to read on from.
func TestALongBodyIsPagedWithANextOffset(t *testing.T) {
	gw := &fakeGateway{doc: domain.SourceDocument{ID: gapDocumentID, Title: "Synthetic", Markdown: strings.Repeat("x", 40000)}}
	var logs bytes.Buffer
	text, isErr := call(t, connect(t, serviceWith(t, gw, mustSource(t, "outline")), &logs), "get_document",
		map[string]any{"document_id": gapDocumentID, "source": "outline"})
	if isErr || !strings.Contains(text, `"truncated":true`) || !strings.Contains(text, `"next_offset"`) {
		t.Fatalf("a long body must be cut with a next offset, got error %v, text prefix %.80q", isErr, text)
	}
}

// TestTheBodyOffsetIsCheckedBeforeAnyRequest checks each offset that the body read refuses, and the ones it accepts.
func TestTheBodyOffsetIsCheckedBeforeAnyRequest(t *testing.T) {
	body := operations.Row{Body: "markdown"}
	if _, err := bodyOffset(body, map[string]any{bodyParam: 1.5}); !errors.Is(err, errBadOffset) {
		t.Errorf("a fractional offset = %v, want errBadOffset", err)
	}
	if _, err := bodyOffset(body, map[string]any{bodyParam: -1.0}); !errors.Is(err, errNegativeOffset) {
		t.Errorf("a negative offset = %v, want errNegativeOffset", err)
	}
	if got, err := bodyOffset(body, map[string]any{bodyParam: 2.0}); err != nil || got != 2 {
		t.Errorf("a whole offset = %d, %v, want 2", got, err)
	}
	if got, err := bodyOffset(body, map[string]any{}); err != nil || got != 0 {
		t.Errorf("an omitted offset = %d, %v, want 0", got, err)
	}
}

// TestDecodeArgsReadsAnEmptyOrBrokenRequest checks the three shapes of a call's arguments: none, an object, and text that
// is not JSON.
func TestDecodeArgsReadsAnEmptyOrBrokenRequest(t *testing.T) {
	if args, err := decodeArgs(nil); err != nil || len(args) != 0 {
		t.Errorf("a nil request = %v, %v, want no arguments", args, err)
	}
	if args, err := decodeArgs(&sdk.CallToolRequest{}); err != nil || len(args) != 0 {
		t.Errorf("a request with no params = %v, %v, want no arguments", args, err)
	}
	if _, err := decodeArgs(requestOf("{")); err == nil {
		t.Error("arguments that are not JSON must be an error")
	}
}

// TestBrokenArgumentsAreAUsageFailure checks that a call whose arguments are not a JSON object is refused as bad
// arguments, with no request.
func TestBrokenArgumentsAreAUsageFailure(t *testing.T) {
	gw := &fakeGateway{}
	_, _, class, public := runOperation(context.Background(), writableOutline(t, gw), rowForTool(t, "create_document"), requestOf("{"))
	if class != "bad arguments" || public == nil || gw.calls != 0 {
		t.Fatalf("broken arguments must be refused, got class %q (public %v)", class, public)
	}
}

// TestAURLListHasItsBounds checks the schema of a list of URLs: at least one and at most the cap.
func TestAURLListHasItsBounds(t *testing.T) {
	schema := propertySchema(operations.Param{Kind: operations.KindURLList, Help: "links"})
	if schema["type"] != "array" || schema["minItems"] != 1 || schema["maxItems"] != domain.MaxURLList {
		t.Fatalf("URL list schema = %v", schema)
	}
}

// TestSnakeRecordConvertsNestedRecords checks that a nested record keeps its fields, with snake_case names.
func TestSnakeRecordConvertsNestedRecords(t *testing.T) {
	rec := domain.NewRecord(domain.Field{Name: "authorName", Value: domain.NewRecord(domain.Field{Name: "firstName", Value: "Synthetic"})})
	nested, ok := snakeRecord(rec)["author_name"].(map[string]any)
	if !ok || nested["first_name"] != "Synthetic" {
		t.Fatalf("a nested record must keep its fields in snake_case, got %v", snakeRecord(rec))
	}
}

// TestAnAnswerThatCannotBeEncodedIsAnError checks that a result which JSON cannot encode becomes the unexpected error.
func TestAnAnswerThatCannotBeEncodedIsAnError(t *testing.T) {
	res := toolResult(map[string]any{"bad": make(chan int)})
	if !res.IsError {
		t.Fatal("an answer that cannot be encoded must be a tool error")
	}
}

// TestAListThatCannotBeEncodedIsAnError checks that a list whose rows cannot be encoded is refused, not sent.
func TestAListThatCannotBeEncodedIsAnError(t *testing.T) {
	rows := domain.NewList([]domain.Record{domain.NewRecord(domain.Field{Name: "bad", Value: make(chan int)})})
	if _, err := renderList(operations.Row{ListKey: "items"}, "outline", rows); err == nil {
		t.Fatal("a list that cannot be encoded must be an error")
	}
}

// TestListSourcesRefusesAnUnknownKindAndShowsAnUntaggedSource checks the two branches of the catalog tool: a kind that
// does not exist is refused, and a source with no tags is listed with an empty tag list.
func TestListSourcesRefusesAnUnknownKindAndShowsAnUntaggedSource(t *testing.T) {
	svc := serviceWith(t, &fakeGateway{}, mustSource(t, "outline"))
	var logs bytes.Buffer
	cs := connect(t, svc, &logs)
	if _, isErr := call(t, cs, "list_sources", map[string]any{"kind": "nope"}); !isErr {
		t.Error("an unknown kind must be refused")
	}
	text, isErr := call(t, cs, "list_sources", nil)
	if isErr || !strings.Contains(text, `"tags":[]`) {
		t.Errorf("an untagged source must list an empty tag list, got %q (error %v)", text, isErr)
	}
}

// TestAServerWithADuplicateToolIsNotBuilt checks that a table naming one tool twice stops the server, both when it is
// built and when a session is served, and that the tool listing refuses it too.
func TestAServerWithADuplicateToolIsNotBuilt(t *testing.T) {
	row := tool{name: "list_sources", effect: domain.EffectRead, bind: bind(listSources)}
	dup := []tool{row, row}
	svc := serviceWith(t, &fakeGateway{}, mustSource(t, "outline"))
	if _, err := newServer(svc, Options{Logger: slog.New(slog.DiscardHandler)}, dup); err == nil {
		t.Error("a server over a duplicated table must not be built")
	}
	if err := serveWith(context.Background(), svc, Options{}, strings.NewReader(""), io.Discard, dup); err == nil {
		t.Error("a session over a duplicated table must not start")
	}
	if _, err := registeredTools(svc, nil, dup); err == nil {
		t.Error("the tool listing over a duplicated table must be refused")
	}
}

// docReply returns a document record in the shape a write answers with: the fields the write tool's row projects.
func docReply() domain.Record {
	return domain.NewRecord(
		domain.Field{Name: "id", Value: gapDocumentID}, domain.Field{Name: "title", Value: "Synthetic"},
		domain.Field{Name: "url", Value: "https://docs.example.test/doc"}, domain.Field{Name: "collectionId", Value: gapCollectionID},
		domain.Field{Name: "revision", Value: 2})
}
