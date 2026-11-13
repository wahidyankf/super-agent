package sources

import (
	"context"
	"errors"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// TestAdministerSelfCheckWithoutAReaderStopsTheChange checks that a change to a user is not sent when the source has no read
// port, so the key's own user cannot be checked (ADR 007 section 3.5, rule 6): the source's refusal is returned, and nothing
// is changed.
func TestAdministerSelfCheckWithoutAReaderStopsTheChange(t *testing.T) {
	admin := &fakeAdmin{reply: ackReply()}
	svc, _ := adminService(t, domain.AccessAdmin, nil, admin)
	_, err := Administer(context.Background(), svc, domain.OpUsersSuspend, strPtr("outline"),
		map[string]any{"userId": otherUserID}, Confirmation{Yes: true})
	if !errors.Is(err, domain.ErrUnsupportedSource) {
		t.Fatalf("Administer error = %v, want ErrUnsupportedSource", err)
	}
	if len(admin.ops) != 0 {
		t.Fatalf("the admin port was called %v; a change that cannot be checked must make no request", admin.ops)
	}
}

// TestAdministerDryRunOfATeamWithoutAReaderIsRefused checks that a dry run cannot name the team of teams.delete when the
// source has no read port, so the dry run stops with the source's refusal.
func TestAdministerDryRunOfATeamWithoutAReaderIsRefused(t *testing.T) {
	admin := &fakeAdmin{reply: ackReply()}
	svc, _ := adminService(t, domain.AccessAdmin, nil, admin)
	_, err := Administer(context.Background(), svc, domain.OpTeamsDelete, strPtr("outline"),
		map[string]any{"code": "ABC123"}, Confirmation{DryRun: true})
	if !errors.Is(err, domain.ErrUnsupportedSource) {
		t.Fatalf("Administer error = %v, want ErrUnsupportedSource", err)
	}
	if len(admin.ops) != 0 {
		t.Fatalf("a dry run reached the admin port: %v", admin.ops)
	}
}

// TestAdministerCriticalTeamWithoutANameIsNotDeleted checks that a team whose name auth.whoami does not give cannot be
// confirmed, so the critical call stops with ErrUnexpectedResponse and sends no delete.
func TestAdministerCriticalTeamWithoutANameIsNotDeleted(t *testing.T) {
	reader := &fakeReader{reply: domain.NewRecord(), byOp: map[domain.OperationID]domain.Record{
		domain.OpAuthWhoami: whoamiRecord(keyUserID, ""),
	}}
	admin := &fakeAdmin{reply: ackReply()}
	svc, _ := adminService(t, domain.AccessAdmin, reader, admin)
	_, err := Administer(context.Background(), svc, domain.OpTeamsDelete, strPtr("outline"),
		map[string]any{"code": "ABC123"}, Confirmation{Yes: true, Confirm: strPtr("")})
	if !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("Administer error = %v, want ErrUnexpectedResponse", err)
	}
	if len(admin.ops) != 0 {
		t.Fatalf("the admin port was called %v for a team with no name", admin.ops)
	}
}

// TestAdministerNeedsTheAdminPortForAChange checks that a change is refused when the source has no admin port, so no call
// is made through a missing port.
func TestAdministerNeedsTheAdminPortForAChange(t *testing.T) {
	svc, _ := adminService(t, domain.AccessAdmin, &fakeReader{reply: domain.NewRecord()}, nil)
	_, err := Administer(context.Background(), svc, domain.OpUsersActivate, strPtr("outline"),
		map[string]any{"userId": otherUserID}, Confirmation{Yes: true})
	if !errors.Is(err, domain.ErrUnsupportedSource) {
		t.Fatalf("Administer error = %v, want ErrUnsupportedSource", err)
	}
}
