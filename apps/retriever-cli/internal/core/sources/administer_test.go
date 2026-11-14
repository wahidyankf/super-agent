package sources

import (
	"context"
	"errors"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/ports"
)

// adminService returns a service over one outline source at access. Its reader and admin are served through the
// resolver, so a test can see whether the use case asked for them.
func adminService(t *testing.T, access domain.Access, reader *fakeReader, admin *fakeAdmin) (Service, *fakeResolver) {
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
	res := &fakeResolver{gateways: map[string]ports.SourceGateway{"outline": &fakeGateway{}}, reader: reader, admin: admin}
	return Service{Catalog: cat, Gateways: res}, res
}

// TestAdministerRefusesAnUnknownOperationAndAnUnknownSourceBeforeAnyRequest checks the two lookups that come before the admin
// port: an operation the catalog does not know, and a source ID the catalog does not hold.
func TestAdministerRefusesAnUnknownOperationAndAnUnknownSourceBeforeAnyRequest(t *testing.T) {
	admin := &fakeAdmin{reply: ackReply()}
	svc, _ := adminService(t, domain.AccessAdmin, &fakeReader{}, admin)

	if _, err := Administer(context.Background(), svc, domain.OperationID("nope.nope"), strPtr("outline"),
		map[string]any{}); !errors.Is(err, domain.ErrUnsupportedOperation) {
		t.Fatalf("an unknown operation: %v, want ErrUnsupportedOperation", err)
	}
	var unknown domain.UnknownSourceError
	if _, err := Administer(context.Background(), svc, domain.OpDocumentsArchive, strPtr("missing"),
		map[string]any{"documentId": "doc-1"}); !errors.As(err, &unknown) {
		t.Fatalf("an unknown source: %v, want UnknownSourceError", err)
	}
	if len(admin.ops) != 0 {
		t.Fatalf("a refused call reached the admin port: %v", admin.ops)
	}
}

// ackReply is the reply of a destructive call that the wiki acknowledged.
func ackReply() domain.Record {
	return domain.NewRecord(domain.Field{Name: "ok", Value: true})
}

// TestAdministerSendsAChangeOnceWithoutAnyConfirmation checks that an admin call runs once with its bound arguments, and
// that the use case reads nothing first: there is no target to resolve and no flag to check.
func TestAdministerSendsAChangeOnceWithoutAnyConfirmation(t *testing.T) {
	reader := &fakeReader{reply: writtenDocument()}
	admin := &fakeAdmin{reply: writtenDocument()}
	svc, _ := adminService(t, domain.AccessAdmin, reader, admin)

	rec, err := Administer(context.Background(), svc, domain.OpDocumentsArchive, strPtr("outline"),
		map[string]any{"documentId": "doc-1"})
	if err != nil {
		t.Fatalf("Administer: %v", err)
	}
	if len(admin.ops) != 1 || admin.ops[0] != domain.OpDocumentsArchive {
		t.Fatalf("the admin port was called %v, want documents.archive once", admin.ops)
	}
	if admin.args[0].String("documentId") != "doc-1" {
		t.Fatalf("the admin port received documentId %q, want doc-1", admin.args[0].String("documentId"))
	}
	if rec.Str("id") != "doc-1" {
		t.Fatalf("the reply id is %q, want doc-1", rec.Str("id"))
	}
	if len(reader.ops) != 0 {
		t.Fatalf("a change read the source before it ran: %v", reader.ops)
	}
}

// TestAdministerRefusesAnAdminCallBelowTheLevelBeforeAnyRequest checks that the access level is still checked before the
// admin port is called. Removing the level is a later change, so this test goes with it.
func TestAdministerRefusesAnAdminCallBelowTheLevelBeforeAnyRequest(t *testing.T) {
	admin := &fakeAdmin{reply: ackReply()}
	svc, _ := adminService(t, domain.AccessWrite, &fakeReader{}, admin)

	_, err := Administer(context.Background(), svc, domain.OpDocumentsArchive, strPtr("outline"),
		map[string]any{"documentId": "doc-1"})
	if !errors.Is(err, domain.ErrAccessLevelTooLow) {
		t.Fatalf("a write source: %v, want ErrAccessLevelTooLow", err)
	}
	if len(admin.ops) != 0 {
		t.Fatalf("a refused call reached the admin port: %v", admin.ops)
	}
}

// TestAdministerRefusesEveryBadCallBeforeAnyRequest checks the refusals that come before the admin port: a read
// operation, a missing source, and a missing required argument.
func TestAdministerRefusesEveryBadCallBeforeAnyRequest(t *testing.T) {
	admin := &fakeAdmin{reply: ackReply()}
	svc, _ := adminService(t, domain.AccessAdmin, &fakeReader{}, admin)

	if _, err := Administer(context.Background(), svc, domain.OpDocumentsGet, strPtr("outline"),
		map[string]any{"documentId": "doc-1"}); !errors.Is(err, domain.ErrWrongOperationClass) {
		t.Fatalf("a read operation: %v, want ErrWrongOperationClass", err)
	}
	if _, err := Administer(context.Background(), svc, domain.OpDocumentsArchive, nil,
		map[string]any{"documentId": "doc-1"}); !errors.Is(err, domain.ErrSourceRequired) {
		t.Fatalf("no source: %v, want ErrSourceRequired", err)
	}
	if _, err := Administer(context.Background(), svc, domain.OpDocumentsArchive, strPtr("outline"),
		map[string]any{}); err == nil {
		t.Fatal("a call without its required documentId was accepted")
	}
	if len(admin.ops) != 0 {
		t.Fatalf("a refused call reached the admin port: %v", admin.ops)
	}
}

// TestAdministerRefusesAReplyOfTheWrongShape checks that an admin reply the use case cannot read is ErrWriteOutcomeUnknown:
// the request was sent, so the change may have run, and the caller never retries it.
func TestAdministerRefusesAReplyOfTheWrongShape(t *testing.T) {
	admin := &fakeAdmin{reply: domain.NewRecord()}
	svc, _ := adminService(t, domain.AccessAdmin, &fakeReader{}, admin)
	_, err := Administer(context.Background(), svc, domain.OpDocumentsDelete, strPtr("outline"),
		map[string]any{"documentId": "doc-1"})
	if !errors.Is(err, domain.ErrWriteOutcomeUnknown) {
		t.Fatalf("a reply without ok: %v, want ErrWriteOutcomeUnknown", err)
	}
	if len(admin.ops) != 1 {
		t.Fatalf("the admin port was called %v times, want once", len(admin.ops))
	}
}
