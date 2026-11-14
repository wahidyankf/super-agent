package sources

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// readService returns a service over one outline source whose reader is reader.
func readService(t *testing.T, reader *fakeReader) (Service, *fakeResolver) {
	t.Helper()
	svc, res := outlineService(t, &fakeGateway{})
	res.reader = reader
	return svc, res
}

// documentPage is a reply the list operations of the catalog accept.
func documentPage() domain.Record {
	return domain.NewList([]domain.Record{domain.NewRecord(
		domain.Field{Name: "id", Value: "doc-1"},
		domain.Field{Name: "title", Value: "Getting Started"},
		domain.Field{Name: "url", Value: "/doc/getting-started"},
		domain.Field{Name: "collectionId", Value: "123e4567-e89b-12d3-a456-426614174000"},
	)})
}

// docObject is a reply the object operations of the catalog accept for documents.get.
func docObject() domain.Record {
	return domain.NewRecord(
		domain.Field{Name: "id", Value: "doc-1"},
		domain.Field{Name: "title", Value: "Getting Started"},
		domain.Field{Name: "url", Value: "/doc/getting-started"},
		domain.Field{Name: "collectionId", Value: "col-1"},
		domain.Field{Name: "revision", Value: 1},
		domain.Field{Name: "markdown", Value: "text"},
	)
}

func TestReadRefusesEveryBadArgumentBeforeAnySourceIsAsked(t *testing.T) {
	cases := []struct {
		name string
		op   domain.OperationID
		raw  map[string]any
		want error
	}{
		{"an unknown operation", domain.OperationID("nope.nope"), nil, domain.ErrUnsupportedOperation},
		{"a document ID with a dot", domain.OpDocumentsGet, map[string]any{"documentId": "a.b"}, domain.ErrInvalidDocumentID},
		{"a limit of 101", domain.OpDocumentsList, map[string]any{"limit": 101}, domain.ErrInvalidLimit},
		{"a negative offset", domain.OpDocumentsList, map[string]any{"offset": -1}, domain.ErrInvalidOffset},
		{"a blank query", domain.OpDocumentsSearch, map[string]any{"query": " "}, domain.ErrInvalidQuery},
		{"a list collection that is a urlId", domain.OpDocumentsList, map[string]any{"collectionId": "col-1"}, domain.ErrInvalidCollectionID},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reader := &fakeReader{reply: documentPage()}
			svc, res := readService(t, reader)
			if _, err := Read(context.Background(), svc, tc.op, nil, tc.raw); !errors.Is(err, tc.want) {
				t.Fatalf("error = %v, want %v", err, tc.want)
			}
			if len(res.asked) != 0 || len(reader.ops) != 0 {
				t.Fatalf("a bad call reached the source: asked %v, reader ran %v", res.asked, reader.ops)
			}
		})
	}
}

func TestReadBindsTheArgumentsAndCallsTheReaderWithThem(t *testing.T) {
	reader := &fakeReader{reply: documentPage()}
	svc, _ := readService(t, reader)
	raw := map[string]any{"collectionId": "123e4567-e89b-12d3-a456-426614174000", "limit": 10}
	reply, err := Read(context.Background(), svc, domain.OpDocumentsList, nil, raw)
	if err != nil {
		t.Fatalf("Read = %v", err)
	}
	if !reply.IsList() || len(reply.Items()) != 1 {
		t.Fatalf("reply = %+v, want one document", reply)
	}
	if !reflect.DeepEqual(reader.ops, []domain.OperationID{domain.OpDocumentsList}) {
		t.Fatalf("reader ops = %v", reader.ops)
	}
	got := reader.args[0]
	if got.String("collectionId") != "123e4567-e89b-12d3-a456-426614174000" || got.Int("limit") != 10 || got.Int("offset") != 0 {
		t.Fatalf("bound args = collectionId %q limit %d offset %d", got.String("collectionId"), got.Int("limit"), got.Int("offset"))
	}
}

func TestReadWithANamedSourceAsksThatSourceOnly(t *testing.T) {
	reader := &fakeReader{reply: docObject()}
	svc, res := readService(t, reader)
	id := "outline"
	if _, err := Read(context.Background(), svc, domain.OpDocumentsGet, &id, map[string]any{"documentId": "doc-1"}); err != nil {
		t.Fatalf("Read = %v", err)
	}
	if !reflect.DeepEqual(res.asked, []string{"outline"}) {
		t.Fatalf("asked sources = %v, want only outline", res.asked)
	}
}

func TestReadStopsAtAnUnknownOrInvalidSourceID(t *testing.T) {
	for _, bad := range []string{"Outline", "a--b", "missing"} {
		reader := &fakeReader{reply: docObject()}
		svc, res := readService(t, reader)
		src := bad
		_, err := Read(context.Background(), svc, domain.OpAuthWhoami, &src, nil)
		if err == nil {
			t.Fatalf("source %q: no error", bad)
		}
		if len(reader.ops) != 0 {
			t.Fatalf("source %q reached the reader", bad)
		}
		if bad == "Outline" && len(res.asked) != 0 {
			t.Fatalf("a bad source name reached For: %v", res.asked)
		}
	}
}

func TestReadReportsAMissingOrFailingReader(t *testing.T) {
	svc, _ := readService(t, nil)
	if _, err := Read(context.Background(), svc, domain.OpAuthWhoami, nil, nil); !errors.Is(err, domain.ErrUnsupportedSource) {
		t.Fatalf("a source with no reader: error = %v, want ErrUnsupportedSource", err)
	}

	failing := &fakeReader{err: domain.ErrOperationUnavailable}
	svc, _ = readService(t, failing)
	if _, err := Read(context.Background(), svc, domain.OpAuthWhoami, nil, nil); !errors.Is(err, domain.ErrOperationUnavailable) {
		t.Fatalf("a failing reader: error = %v, want ErrOperationUnavailable", err)
	}
}

// TestReadRefusesAReplyOfTheWrongShape covers finding A7: a reader whose reply part is missing or of the wrong shape
// yields ErrUnexpectedResponse, never a value a caller would dereference.
func TestReadRefusesAReplyOfTheWrongShape(t *testing.T) {
	cases := []struct {
		name  string
		op    domain.OperationID
		raw   map[string]any
		reply domain.Record
	}{
		{"an object for a list operation", domain.OpDocumentsList, nil, docObject()},
		{"a list for an object operation", domain.OpDocumentsGet, map[string]any{"documentId": "doc-1"}, documentPage()},
		{"an empty reply for an object operation", domain.OpCollectionsInfo, map[string]any{"collectionId": "col-1"}, domain.Record{}},
		{"an object missing a declared field", domain.OpDocumentsGet, map[string]any{"documentId": "doc-1"},
			domain.NewRecord(domain.Field{Name: "id", Value: "doc-1"})},
		// F5: the check reaches every item of a list, not only the list. An item that misses a declared field is the
		// same defect as an object that does.
		{"a list whose item misses a declared field", domain.OpDocumentsList, nil,
			domain.NewList([]domain.Record{domain.NewRecord(domain.Field{Name: "id", Value: "doc-1"})})},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, _ := readService(t, &fakeReader{reply: tc.reply})
			if _, err := Read(context.Background(), svc, tc.op, nil, tc.raw); !errors.Is(err, domain.ErrUnexpectedResponse) {
				t.Fatalf("error = %v, want ErrUnexpectedResponse", err)
			}
		})
	}
}
