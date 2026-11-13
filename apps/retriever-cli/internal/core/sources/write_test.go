package sources

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/ports"
)

// writeService returns a service over one outline source at access, whose writer is writer. The writer is served
// through the resolver, so a test can see whether the use case asked for it.
func writeService(t *testing.T, writer *fakeWriter, access domain.Access) (Service, *fakeResolver) {
	t.Helper()
	id, err := domain.NewSourceID("outline")
	if err != nil {
		t.Fatalf("NewSourceID: %v", err)
	}
	ep, err := domain.NewEndpoint("https://wiki0.example.com")
	if err != nil {
		t.Fatalf("NewEndpoint: %v", err)
	}
	ref, err := domain.NewCredentialRef("KEY0_API_KEY")
	if err != nil {
		t.Fatalf("NewCredentialRef: %v", err)
	}
	tags, _ := domain.NewTagSet()
	cat, err := domain.NewCatalog([]domain.Source{domain.NewSource(id, domain.KindOutline, tags, access, ep, ref)})
	if err != nil {
		t.Fatalf("NewCatalog: %v", err)
	}
	res := &fakeResolver{gateways: map[string]ports.SourceGateway{"outline": &fakeGateway{}}, writer: writer}
	return Service{Catalog: cat, Gateways: res}, res
}

// writtenDocument is a reply the documents.update row accepts: a document with its revision.
func writtenDocument() domain.Record {
	return domain.NewRecord(
		domain.Field{Name: "id", Value: "doc-1"},
		domain.Field{Name: "title", Value: "Getting Started"},
		domain.Field{Name: "url", Value: "/doc/getting-started"},
		domain.Field{Name: "collectionId", Value: "col-1"},
		domain.Field{Name: "revision", Value: 3},
	)
}

func TestWriteRefusesEveryBadCallBeforeAnySourceIsAskedOrAnyRequestIsMade(t *testing.T) {
	cases := []struct {
		name   string
		op     domain.OperationID
		source *string
		raw    map[string]any
		want   error
	}{
		{"a read operation", domain.OpDocumentsList, strPtr("outline"), nil, domain.ErrWrongOperationClass},
		{"an unknown operation", domain.OperationID("nope.nope"), strPtr("outline"), nil, domain.ErrUnsupportedOperation},
		{"no source named", domain.OpDocumentsUpdate, nil, map[string]any{"documentId": "doc-1"}, domain.ErrSourceRequired},
		{"a source ID that is not a slug", domain.OpDocumentsUpdate, strPtr("Outline"), map[string]any{"documentId": "doc-1"}, domain.ErrInvalidSourceID},
		{"a document ID with a dot", domain.OpDocumentsUpdate, strPtr("outline"), map[string]any{"documentId": "a.b"}, domain.ErrInvalidDocumentID},
		{"no document named", domain.OpDocumentsUpdate, strPtr("outline"), map[string]any{}, domain.ErrMissingArgument},
		{"an argument the row does not declare", domain.OpDocumentsUpdate, strPtr("outline"), map[string]any{"documentId": "doc-1", "bogus": 1}, domain.ErrUnknownArgument},
		{"an edit mode outside the list", domain.OpDocumentsUpdate, strPtr("outline"), map[string]any{"documentId": "doc-1", "editMode": "merge"}, domain.ErrInvalidEnum},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			writer := &fakeWriter{reply: writtenDocument()}
			svc, res := writeService(t, writer, domain.AccessWrite)
			if _, err := Write(context.Background(), svc, tc.op, tc.source, tc.raw, false); !errors.Is(err, tc.want) {
				t.Fatalf("error = %v, want %v", err, tc.want)
			}
			if len(res.asked) != 0 || len(writer.ops) != 0 {
				t.Fatalf("a bad call reached the source: asked %v, writer ran %v", res.asked, writer.ops)
			}
		})
	}
}

// TestWriteRefusesAReadSourceBeforeTheWriterIsAsked checks the access rule: a source below the
// level a write needs is refused, and a dry run is refused the same way.
func TestWriteRefusesAReadSourceBeforeTheWriterIsAsked(t *testing.T) {
	for _, dryRun := range []bool{false, true} {
		writer := &fakeWriter{reply: writtenDocument()}
		svc, res := writeService(t, writer, domain.AccessRead)
		_, err := Write(context.Background(), svc, domain.OpDocumentsUpdate, strPtr("outline"), map[string]any{"documentId": "doc-1"}, dryRun)
		if !errors.Is(err, domain.ErrAccessLevelTooLow) {
			t.Fatalf("dryRun %v: error = %v, want ErrAccessLevelTooLow", dryRun, err)
		}
		var lvl domain.AccessLevelError
		if !errors.As(err, &lvl) || lvl.Have != domain.AccessRead || lvl.Need != domain.AccessWrite {
			t.Fatalf("dryRun %v: the error does not name the two levels: %v", dryRun, err)
		}
		if len(res.asked) != 0 || len(writer.ops) != 0 {
			t.Fatalf("dryRun %v: a refused write reached the source: asked %v, writer ran %v", dryRun, res.asked, writer.ops)
		}
	}
}

// TestWriteDryRunMakesNoCallAndReportsThePlan checks the dry run: it asks no gateway, and its
// reply names the operation, the source, the arguments, and that nothing changed. Text is counted, never echoed.
func TestWriteDryRunMakesNoCallAndReportsThePlan(t *testing.T) {
	writer := &fakeWriter{reply: writtenDocument()}
	svc, res := writeService(t, writer, domain.AccessWrite)
	raw := map[string]any{"documentId": "doc-1", "text": "hello world", "editMode": "append", "lastRevision": 3}
	plan, err := Write(context.Background(), svc, domain.OpDocumentsUpdate, strPtr("outline"), raw, true)
	if err != nil {
		t.Fatalf("dry run = %v", err)
	}
	if len(res.asked) != 0 || len(writer.ops) != 0 {
		t.Fatalf("a dry run reached the source: asked %v, writer ran %v", res.asked, writer.ops)
	}
	if got := plan.Str("operation"); got != "documents.update" {
		t.Fatalf("operation = %q", got)
	}
	if got := plan.Str("source"); got != "outline" {
		t.Fatalf("source = %q", got)
	}
	if v, _ := plan.Get("dryRun"); v != true {
		t.Fatalf("dryRun = %v, want true", v)
	}
	if v, _ := plan.Get("changed"); v != false {
		t.Fatalf("changed = %v, want false", v)
	}
	args := plan.Rec("arguments")
	if got := args.Str("documentId"); got != "doc-1" {
		t.Fatalf("arguments.documentId = %q", got)
	}
	if got := args.Str("text"); got != "<11 bytes of text>" {
		t.Fatalf("arguments.text = %q, want the byte count only", got)
	}
	if got := args.Str("editMode"); got != "append" {
		t.Fatalf("arguments.editMode = %q", got)
	}
	if v, _ := args.Get("lastRevision"); fmt.Sprint(v) != "3" {
		t.Fatalf("arguments.lastRevision = %v, want 3", v)
	}
}

// TestWriteCallsTheWriterWithTheBoundArgsAndReturnsItsReply checks the call: the bound arguments reach the writer,
// and its reply comes back when it has the row's shape.
func TestWriteCallsTheWriterWithTheBoundArgsAndReturnsItsReply(t *testing.T) {
	writer := &fakeWriter{reply: writtenDocument()}
	svc, res := writeService(t, writer, domain.AccessWrite)
	reply, err := Write(context.Background(), svc, domain.OpDocumentsUpdate, strPtr("outline"), map[string]any{"documentId": "doc-1", "title": "New"}, false)
	if err != nil {
		t.Fatalf("Write = %v", err)
	}
	if got := reply.Str("id"); got != "doc-1" {
		t.Fatalf("reply id = %q", got)
	}
	if !reflect.DeepEqual(res.asked, []string{"outline"}) {
		t.Fatalf("asked sources = %v, want only outline", res.asked)
	}
	if !reflect.DeepEqual(writer.ops, []domain.OperationID{domain.OpDocumentsUpdate}) {
		t.Fatalf("writer ops = %v", writer.ops)
	}
	if got := writer.args[0].String("documentId"); got != "doc-1" {
		t.Fatalf("bound documentId = %q", got)
	}
	if got := writer.args[0].String("title"); got != "New" {
		t.Fatalf("bound title = %q", got)
	}
}

// TestWriteAcceptsAListReplyForAListRow checks that a list row (documents.move answers the documents that moved) takes
// a list reply, and that each item holds the declared fields.
func TestWriteAcceptsAListReplyForAListRow(t *testing.T) {
	writer := &fakeWriter{reply: domain.NewList([]domain.Record{writtenDocument()})}
	svc, _ := writeService(t, writer, domain.AccessWrite)
	reply, err := Write(context.Background(), svc, domain.OpDocumentsMove, strPtr("outline"), map[string]any{"documentId": "doc-1"}, false)
	if err != nil {
		t.Fatalf("Write = %v", err)
	}
	if !reply.IsList() || len(reply.Items()) != 1 {
		t.Fatalf("reply = %+v, want one document", reply)
	}
}

// TestWriteRefusesAReplyOfTheWrongShape covers a writer whose reply part is missing or of the wrong shape: the caller
// gets ErrWriteOutcomeUnknown, because the request was sent and the change may have run, and never a value it would
// dereference.
func TestWriteRefusesAReplyOfTheWrongShape(t *testing.T) {
	cases := []struct {
		name  string
		op    domain.OperationID
		reply domain.Record
	}{
		{"an object missing a declared field", domain.OpDocumentsUpdate,
			domain.NewRecord(domain.Field{Name: "id", Value: "doc-1"})},
		{"a list for an object row", domain.OpDocumentsUpdate,
			domain.NewList([]domain.Record{writtenDocument()})},
		{"an object for a list row", domain.OpDocumentsMove, writtenDocument()},
		{"a list item missing a declared field", domain.OpDocumentsMove,
			domain.NewList([]domain.Record{domain.NewRecord(domain.Field{Name: "id", Value: "doc-1"})})},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, _ := writeService(t, &fakeWriter{reply: tc.reply}, domain.AccessWrite)
			if _, err := Write(context.Background(), svc, tc.op, strPtr("outline"), map[string]any{"documentId": "doc-1"}, false); !errors.Is(err, domain.ErrWriteOutcomeUnknown) {
				t.Fatalf("error = %v, want ErrWriteOutcomeUnknown", err)
			}
		})
	}
}

// TestWriteReturnsTheWritersErrors checks that an error of the writer reaches the caller unchanged: a timed out write
// stays ErrWriteOutcomeUnknown, and a conflict stays ErrDocumentConflict.
func TestWriteReturnsTheWritersErrors(t *testing.T) {
	for _, want := range []error{domain.ErrWriteOutcomeUnknown, domain.ErrDocumentConflict} {
		svc, _ := writeService(t, &fakeWriter{err: want}, domain.AccessWrite)
		if _, err := Write(context.Background(), svc, domain.OpDocumentsUpdate, strPtr("outline"), map[string]any{"documentId": "doc-1"}, false); !errors.Is(err, want) {
			t.Fatalf("error = %v, want %v", err, want)
		}
	}
}

// TestWriteReportsAMissingWriter checks that a source with no writer is unsupported, and that the call stops there.
func TestWriteReportsAMissingWriter(t *testing.T) {
	svc, _ := writeService(t, nil, domain.AccessWrite)
	if _, err := Write(context.Background(), svc, domain.OpDocumentsUpdate, strPtr("outline"), map[string]any{"documentId": "doc-1"}, false); !errors.Is(err, domain.ErrUnsupportedSource) {
		t.Fatalf("error = %v, want ErrUnsupportedSource", err)
	}
}
