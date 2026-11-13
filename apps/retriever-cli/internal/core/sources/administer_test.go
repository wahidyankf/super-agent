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

// fieldOf returns the value of one field of a Record, or nil when the Record holds no such field.
func fieldOf(rec domain.Record, name string) any {
	v, _ := rec.Get(name)
	return v
}

// ackReply is the reply of a destructive call that the wiki acknowledged.
func ackReply() domain.Record {
	return domain.NewRecord(domain.Field{Name: "ok", Value: true})
}

// targetDocument is the reply of documents.info for the document doc-1, which is the target of the tests below.
func targetDocument() domain.Record {
	return writtenDocument()
}

func TestAdministerRefusesEveryBadCallBeforeAnyWholeRequest(t *testing.T) {
	cases := []struct {
		name   string
		access domain.Access
		op     domain.OperationID
		source *string
		raw    map[string]any
		conf   Confirmation
		reader error
		want   error
	}{
		{"a read operation", domain.AccessAdmin, domain.OpDocumentsGet, strPtr("outline"), nil, Confirmation{Yes: true}, nil, domain.ErrWrongOperationClass},
		{"an unknown operation", domain.AccessAdmin, domain.OperationID("nope.nope"), strPtr("outline"), nil, Confirmation{Yes: true}, nil, domain.ErrUnsupportedOperation},
		{"no source named", domain.AccessAdmin, domain.OpDocumentsArchive, nil, map[string]any{"documentId": "doc-1"}, Confirmation{Yes: true}, nil, domain.ErrSourceRequired},
		{"a source below admin", domain.AccessWrite, domain.OpDocumentsArchive, strPtr("outline"), map[string]any{"documentId": "doc-1"}, Confirmation{Yes: true}, nil, domain.ErrAccessLevelTooLow},
		{"no --yes on a call that is not a dry run", domain.AccessAdmin, domain.OpDocumentsArchive, strPtr("outline"), map[string]any{"documentId": "doc-1"}, Confirmation{}, nil, domain.ErrYesRequired},
		{"a critical call without --confirm", domain.AccessAdmin, domain.OpDocumentsDelete, strPtr("outline"), map[string]any{"documentId": "doc-1", "permanent": true}, Confirmation{Yes: true}, nil, domain.ErrConfirmRequired},
		{"a critical call with the title, not the ID", domain.AccessAdmin, domain.OpDocumentsDelete, strPtr("outline"), map[string]any{"documentId": "doc-1", "permanent": true}, Confirmation{Yes: true, Confirm: strPtr("Getting Started")}, nil, domain.ErrConfirmMismatch},
		{"a critical call whose target does not resolve", domain.AccessAdmin, domain.OpDocumentsDelete, strPtr("outline"), map[string]any{"documentId": "doc-1", "permanent": true}, Confirmation{Yes: true, Confirm: strPtr("doc-1")}, domain.ErrDocumentNotFound, domain.ErrDocumentNotFound},
		{"empty-trash with a wrong source ID", domain.AccessAdmin, domain.OpDocumentsEmptyTrash, strPtr("outline"), nil, Confirmation{Yes: true, Confirm: strPtr("trash")}, nil, domain.ErrConfirmMismatch},
		{"a reason with a control character", domain.AccessAdmin, domain.OpDocumentsArchive, strPtr("outline"), map[string]any{"documentId": "doc-1", "reason": "a\x07b"}, Confirmation{Yes: true}, nil, domain.ErrInvalidReason},
		{"a source ID that the catalog does not hold", domain.AccessAdmin, domain.OpDocumentsArchive, strPtr("nope"), map[string]any{"documentId": "doc-1"}, Confirmation{Yes: true}, nil, domain.UnknownSourceError{ID: "nope"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reader := &fakeReader{reply: targetDocument(), err: tc.reader}
			admin := &fakeAdmin{reply: ackReply()}
			svc, _ := adminService(t, tc.access, reader, admin)
			_, err := Administer(context.Background(), svc, tc.op, tc.source, tc.raw, tc.conf)
			if !errors.Is(err, tc.want) {
				t.Fatalf("Administer error = %v, want %v", err, tc.want)
			}
			if len(admin.ops) != 0 {
				t.Fatalf("the admin port was called %v; a refused call must make no request", admin.ops)
			}
		})
	}
}

// TestAdministerRefusesATargetWithNoCanonicalValue checks that a target whose reply holds no value for the row's target
// field stops the call, because a dry run could not name it and a critical call could not be confirmed.
func TestAdministerRefusesATargetWithNoCanonicalValue(t *testing.T) {
	reader := &fakeReader{reply: domain.NewRecord()}
	admin := &fakeAdmin{reply: ackReply()}
	svc, _ := adminService(t, domain.AccessAdmin, reader, admin)
	_, err := Administer(context.Background(), svc, domain.OpDocumentsDelete, strPtr("outline"),
		map[string]any{"documentId": "doc-1"}, Confirmation{DryRun: true})
	if !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("Administer error = %v, want ErrUnexpectedResponse", err)
	}
	if len(admin.ops) != 0 {
		t.Fatalf("the admin port was called %v; a dry run must make no request", admin.ops)
	}
}

func TestAdministerDryRunNamesTheTargetAndSendsNothing(t *testing.T) {
	reader := &fakeReader{reply: targetDocument()}
	admin := &fakeAdmin{reply: ackReply()}
	svc, _ := adminService(t, domain.AccessAdmin, reader, admin)

	plan, err := Administer(context.Background(), svc, domain.OpDocumentsArchive, strPtr("outline"),
		map[string]any{"documentId": "doc-1"}, Confirmation{DryRun: true})
	if err != nil {
		t.Fatalf("dry run: %v", err)
	}
	if plan.Str("target") != "doc-1" || plan.Str("targetName") != "Getting Started" {
		t.Fatalf("the plan names target %q %q, want doc-1 and its title", plan.Str("target"), plan.Str("targetName"))
	}
	if plan.Str("operation") != "documents.archive" || fieldOf(plan, "changed") != false || fieldOf(plan, "dryRun") != true {
		t.Fatalf("the plan is not a dry run of documents.archive that changed nothing: %v", plan.Fields())
	}
	if len(reader.ops) != 1 || reader.ops[0] != domain.OpDocumentsGet {
		t.Fatalf("the dry run read %v, want one documents.get to name the target", reader.ops)
	}
	if len(admin.ops) != 0 {
		t.Fatalf("a dry run reached the admin port: %v", admin.ops)
	}
}

func TestAdministerDryRunNeedsNoConfirmForACriticalCall(t *testing.T) {
	reader := &fakeReader{reply: targetDocument()}
	admin := &fakeAdmin{reply: ackReply()}
	svc, _ := adminService(t, domain.AccessAdmin, reader, admin)

	plan, err := Administer(context.Background(), svc, domain.OpDocumentsDelete, strPtr("outline"),
		map[string]any{"documentId": "doc-1", "permanent": true}, Confirmation{DryRun: true})
	if err != nil {
		t.Fatalf("a dry run of a critical call needs no --confirm: %v", err)
	}
	if plan.Str("target") != "doc-1" {
		t.Fatalf("the plan names target %q, want doc-1, so a person can type it", plan.Str("target"))
	}
	if len(admin.ops) != 0 {
		t.Fatalf("a dry run reached the admin port: %v", admin.ops)
	}
}

func TestAdministerSendsANonCriticalCallOnceWithYes(t *testing.T) {
	reader := &fakeReader{reply: targetDocument()}
	admin := &fakeAdmin{reply: writtenDocument()}
	svc, _ := adminService(t, domain.AccessAdmin, reader, admin)

	rec, err := Administer(context.Background(), svc, domain.OpDocumentsArchive, strPtr("outline"),
		map[string]any{"documentId": "doc-1"}, Confirmation{Yes: true})
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
		t.Fatalf("a call that is not critical read the target: %v", reader.ops)
	}
}

func TestAdministerCriticalTierNeedsTheCanonicalTargetValue(t *testing.T) {
	reader := &fakeReader{reply: targetDocument()}
	admin := &fakeAdmin{reply: writtenDocument()}
	svc, _ := adminService(t, domain.AccessAdmin, reader, admin)

	// A missing --confirm and a mismatch are both refused, and nothing is sent.
	if _, err := Administer(context.Background(), svc, domain.OpDocumentsDelete, strPtr("outline"),
		map[string]any{"documentId": "doc-1", "permanent": true}, Confirmation{Yes: true}); !errors.Is(err, domain.ErrConfirmRequired) {
		t.Fatalf("a missing --confirm: %v", err)
	}
	if _, err := Administer(context.Background(), svc, domain.OpDocumentsDelete, strPtr("outline"),
		map[string]any{"documentId": "doc-1", "permanent": true}, Confirmation{Yes: true, Confirm: strPtr("doc-9")}); !errors.Is(err, domain.ErrConfirmMismatch) {
		t.Fatalf("a wrong --confirm: %v", err)
	}
	if len(admin.ops) != 0 {
		t.Fatalf("a refused critical call reached the admin port: %v", admin.ops)
	}

	// The right value is the ID that Retriever read, and it sends the call once with permanent true.
	admin.reply = ackReply()
	if _, err := Administer(context.Background(), svc, domain.OpDocumentsDelete, strPtr("outline"),
		map[string]any{"documentId": "doc-1", "permanent": true}, Confirmation{Yes: true, Confirm: strPtr("doc-1")}); err != nil {
		t.Fatalf("the right --confirm: %v", err)
	}
	if len(admin.ops) != 1 || admin.args[0].Values()["permanent"] != true {
		t.Fatalf("the admin port was called %v with %v, want one permanent delete", admin.ops, admin.args)
	}
}

func TestAdministerEmptyTrashConfirmsWithTheSourceID(t *testing.T) {
	reader := &fakeReader{reply: domain.NewList([]domain.Record{targetDocument()})}
	admin := &fakeAdmin{reply: ackReply()}
	svc, _ := adminService(t, domain.AccessAdmin, reader, admin)

	plan, err := Administer(context.Background(), svc, domain.OpDocumentsEmptyTrash, strPtr("outline"), nil, Confirmation{DryRun: true})
	if err != nil {
		t.Fatalf("dry run of empty-trash: %v", err)
	}
	if plan.Str("target") != "outline" || fieldOf(plan, "targetCount") != 1 {
		t.Fatalf("the plan names target %q with count %d, want the source ID and the trash count", plan.Str("target"), fieldOf(plan, "targetCount"))
	}
	if _, err := Administer(context.Background(), svc, domain.OpDocumentsEmptyTrash, strPtr("outline"), nil,
		Confirmation{Yes: true, Confirm: strPtr("outline")}); err != nil {
		t.Fatalf("empty-trash with the source ID: %v", err)
	}
	if len(admin.ops) != 1 || admin.ops[0] != domain.OpDocumentsEmptyTrash {
		t.Fatalf("the admin port was called %v, want documents.empty-trash once", admin.ops)
	}
}

// TestAdministerRefusesAReplyOfTheWrongShape checks that an admin reply the use case cannot read is ErrWriteOutcomeUnknown:
// the request was sent, so the change may have run, and the caller never retries it (m9).
func TestAdministerRefusesAReplyOfTheWrongShape(t *testing.T) {
	reader := &fakeReader{reply: targetDocument()}
	admin := &fakeAdmin{reply: domain.NewRecord()}
	svc, _ := adminService(t, domain.AccessAdmin, reader, admin)
	_, err := Administer(context.Background(), svc, domain.OpDocumentsDelete, strPtr("outline"),
		map[string]any{"documentId": "doc-1"}, Confirmation{Yes: true, Confirm: strPtr("doc-1")})
	if !errors.Is(err, domain.ErrWriteOutcomeUnknown) {
		t.Fatalf("a reply without ok: %v, want ErrWriteOutcomeUnknown", err)
	}
	if len(admin.ops) != 1 {
		t.Fatalf("the admin port was called %v times, want once", len(admin.ops))
	}
}
