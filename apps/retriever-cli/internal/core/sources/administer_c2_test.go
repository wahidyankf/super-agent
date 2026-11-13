package sources

import (
	"context"
	"errors"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// keyUserID is the ID of the user of the API key in the tests below, as auth.whoami names it.
const keyUserID = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"

// otherUserID is the ID of a user that is not the key's, so a change to it is not refused.
const otherUserID = "44444444-4444-4444-8444-444444444444"

// whoamiRecord is the reply of auth.whoami for the user with the given ID and a team of the given name.
func whoamiRecord(userID, team string) domain.Record {
	return domain.NewRecord(
		domain.Field{Name: "user", Value: domain.NewRecord(
			domain.Field{Name: "id", Value: userID},
			domain.Field{Name: "name", Value: "Ann"},
		)},
		domain.Field{Name: "team", Value: domain.NewRecord(domain.Field{Name: "name", Value: team})},
	)
}

// userRecord is the reply of users.info for one user, which is the target of a user change.
func userRecord(id, name string) domain.Record {
	return domain.NewRecord(domain.Field{Name: "id", Value: id}, domain.Field{Name: "name", Value: name})
}

// providerRecord is the reply of authentication-providers.info for one sign-in provider.
func providerRecord(id, name string) domain.Record {
	return domain.NewRecord(domain.Field{Name: "id", Value: id}, domain.Field{Name: "name", Value: name})
}

// TestAdministerRefusesAChangeToTheKeysOwnUserBeforeAnyRequest checks rule 6: a delete, a suspension,
// or a role change of the user of the API key is refused, in a dry run as well as a real call, and the match ignores case.
func TestAdministerRefusesAChangeToTheKeysOwnUserBeforeAnyRequest(t *testing.T) {
	cases := []struct {
		name string
		op   domain.OperationID
		raw  map[string]any
		conf Confirmation
	}{
		{"suspend", domain.OpUsersSuspend, map[string]any{"userId": keyUserID}, Confirmation{Yes: true}},
		{"suspend with an upper-case ID", domain.OpUsersSuspend, map[string]any{"userId": "AAAAAAAA-AAAA-4AAA-8AAA-AAAAAAAAAAAA"}, Confirmation{Yes: true}},
		{"update-role", domain.OpUsersUpdateRole, map[string]any{"userId": keyUserID, "role": "viewer"}, Confirmation{Yes: true}},
		{"delete", domain.OpUsersDelete, map[string]any{"userId": keyUserID}, Confirmation{Yes: true, Confirm: strPtr(keyUserID)}},
		{"a dry run of suspend", domain.OpUsersSuspend, map[string]any{"userId": keyUserID}, Confirmation{DryRun: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reader := &fakeReader{reply: userRecord(keyUserID, "Ann"), byOp: map[domain.OperationID]domain.Record{
				domain.OpAuthWhoami: whoamiRecord(keyUserID, "Fixture Team"),
			}}
			admin := &fakeAdmin{reply: ackReply()}
			svc, _ := adminService(t, domain.AccessAdmin, reader, admin)
			_, err := Administer(context.Background(), svc, tc.op, strPtr("outline"), tc.raw, tc.conf)
			if !errors.Is(err, domain.ErrSelfTarget) {
				t.Fatalf("Administer error = %v, want ErrSelfTarget", err)
			}
			if len(admin.ops) != 0 {
				t.Fatalf("the admin port was called %v; a refused call must make no request", admin.ops)
			}
			if len(reader.ops) == 0 || reader.ops[0] != domain.OpAuthWhoami {
				t.Fatalf("the use case read %v first, want auth.whoami to learn the key's user", reader.ops)
			}
		})
	}
}

// TestAdministerRefusesAUserChangeWithoutYesBeforeAnyRead checks the order for the self-target rows: a
// change to a user that is not confirmed with --yes is refused as yes_required before the key's user is read, so a call that
// is not confirmed makes no read and no change at all, whoever the target is.
func TestAdministerRefusesAUserChangeWithoutYesBeforeAnyRead(t *testing.T) {
	cases := []struct {
		name string
		op   domain.OperationID
		raw  map[string]any
		conf Confirmation
	}{
		{"suspend of the key's own user", domain.OpUsersSuspend, map[string]any{"userId": keyUserID}, Confirmation{}},
		{"update-role of the key's own user", domain.OpUsersUpdateRole, map[string]any{"userId": keyUserID, "role": "viewer"}, Confirmation{}},
		{"delete of the key's own user", domain.OpUsersDelete, map[string]any{"userId": keyUserID}, Confirmation{Confirm: strPtr(keyUserID)}},
		{"suspend of another user", domain.OpUsersSuspend, map[string]any{"userId": otherUserID}, Confirmation{}},
		{"update-role of another user", domain.OpUsersUpdateRole, map[string]any{"userId": otherUserID, "role": "viewer"}, Confirmation{}},
		{"delete of another user", domain.OpUsersDelete, map[string]any{"userId": otherUserID}, Confirmation{Confirm: strPtr(otherUserID)}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reader := &fakeReader{reply: userRecord(otherUserID, "Ben"), byOp: map[domain.OperationID]domain.Record{
				domain.OpAuthWhoami: whoamiRecord(keyUserID, "Fixture Team"),
			}}
			admin := &fakeAdmin{reply: ackReply()}
			svc, _ := adminService(t, domain.AccessAdmin, reader, admin)
			_, err := Administer(context.Background(), svc, tc.op, strPtr("outline"), tc.raw, tc.conf)
			if !errors.Is(err, domain.ErrYesRequired) {
				t.Fatalf("Administer error = %v, want ErrYesRequired", err)
			}
			if len(reader.ops) != 0 {
				t.Fatalf("a call without --yes read %v; the refusal must come before any read", reader.ops)
			}
			if len(admin.ops) != 0 {
				t.Fatalf("a call without --yes reached the admin port: %v", admin.ops)
			}
		})
	}
}

// TestAdministerReadsTheKeysUserOnlyAfterYes checks that, with --yes, the self-check reads auth.whoami and stops a change to
// the key's own user before the target of a critical delete is read: the order is yes, self-check, target, admin.
func TestAdministerReadsTheKeysUserOnlyAfterYes(t *testing.T) {
	reader := &fakeReader{reply: userRecord(keyUserID, "Ann"), byOp: map[domain.OperationID]domain.Record{
		domain.OpAuthWhoami: whoamiRecord(keyUserID, "Fixture Team"),
	}}
	admin := &fakeAdmin{reply: ackReply()}
	svc, _ := adminService(t, domain.AccessAdmin, reader, admin)
	_, err := Administer(context.Background(), svc, domain.OpUsersDelete, strPtr("outline"),
		map[string]any{"userId": keyUserID}, Confirmation{Yes: true, Confirm: strPtr(keyUserID)})
	if !errors.Is(err, domain.ErrSelfTarget) {
		t.Fatalf("Administer error = %v, want ErrSelfTarget", err)
	}
	if len(reader.ops) != 1 || reader.ops[0] != domain.OpAuthWhoami {
		t.Fatalf("the reads were %v, want auth.whoami alone: the target is not read for a refused self change", reader.ops)
	}
	if len(admin.ops) != 0 {
		t.Fatalf("a refused self change reached the admin port: %v", admin.ops)
	}
}

// TestAdministerLetsAChangeToAnotherUserThroughTheSelfCheck checks that a change to a user who is not the key's is sent once,
// after the key's user was read with auth.whoami.
func TestAdministerLetsAChangeToAnotherUserThroughTheSelfCheck(t *testing.T) {
	reader := &fakeReader{reply: userRecord(otherUserID, "Ben"), byOp: map[domain.OperationID]domain.Record{
		domain.OpAuthWhoami: whoamiRecord(keyUserID, "Fixture Team"),
	}}
	admin := &fakeAdmin{reply: writtenUser()}
	svc, _ := adminService(t, domain.AccessAdmin, reader, admin)
	if _, err := Administer(context.Background(), svc, domain.OpUsersSuspend, strPtr("outline"),
		map[string]any{"userId": otherUserID}, Confirmation{Yes: true}); err != nil {
		t.Fatalf("suspend of another user: %v", err)
	}
	if len(admin.ops) != 1 || admin.ops[0] != domain.OpUsersSuspend {
		t.Fatalf("the admin port was called %v, want users.suspend once", admin.ops)
	}
}

// writtenUser is the reply of users.suspend for one user: every field of the user projection, as the adapter sends it.
func writtenUser() domain.Record {
	return domain.NewRecord(
		domain.Field{Name: "id", Value: otherUserID},
		domain.Field{Name: "name", Value: "Ben"},
		domain.Field{Name: "role", Value: "member"},
		domain.Field{Name: "isSuspended", Value: true},
		domain.Field{Name: "lastActiveAt", Value: nil},
		domain.Field{Name: "createdAt", Value: "2026-01-04T00:00:00Z"},
		domain.Field{Name: "updatedAt", Value: "2026-01-04T00:00:00Z"},
		domain.Field{Name: "deletedAt", Value: nil},
	)
}

// TestAdministerRefusesAWhoamiReplyWithoutTheUserID checks that a reply that names no user ID stops the change, so the
// refusal of a change to the key's own user never depends on a guess.
func TestAdministerRefusesAWhoamiReplyWithoutTheUserID(t *testing.T) {
	reader := &fakeReader{reply: userRecord(otherUserID, "Ben"), byOp: map[domain.OperationID]domain.Record{
		domain.OpAuthWhoami: domain.NewRecord(domain.Field{Name: "team", Value: domain.NewRecord()}),
	}}
	admin := &fakeAdmin{reply: writtenUser()}
	svc, _ := adminService(t, domain.AccessAdmin, reader, admin)
	_, err := Administer(context.Background(), svc, domain.OpUsersSuspend, strPtr("outline"),
		map[string]any{"userId": otherUserID}, Confirmation{Yes: true})
	if !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("Administer error = %v, want ErrUnexpectedResponse", err)
	}
	if len(admin.ops) != 0 {
		t.Fatalf("the admin port was called %v; a call that cannot be checked must make no request", admin.ops)
	}
}

// TestAdministerPropagatesAFailedWhoamiRead checks that a failed read of the key's user stops the change with that error.
func TestAdministerPropagatesAFailedWhoamiRead(t *testing.T) {
	reader := &fakeReader{err: domain.ErrKeyRejected}
	admin := &fakeAdmin{reply: writtenUser()}
	svc, _ := adminService(t, domain.AccessAdmin, reader, admin)
	_, err := Administer(context.Background(), svc, domain.OpUsersSuspend, strPtr("outline"),
		map[string]any{"userId": otherUserID}, Confirmation{Yes: true})
	if !errors.Is(err, domain.ErrKeyRejected) {
		t.Fatalf("Administer error = %v, want ErrKeyRejected", err)
	}
	if len(admin.ops) != 0 {
		t.Fatalf("the admin port was called %v after a failed read", admin.ops)
	}
}

// TestAdministerCriticalUsersDeleteConfirmsTheUserID checks the canonical target of users.delete: the
// user ID that users.info reads, with no --confirm the call is refused, and a wrong value is refused too.
func TestAdministerCriticalUsersDeleteConfirmsTheUserID(t *testing.T) {
	reader := &fakeReader{reply: userRecord(otherUserID, "Ben"), byOp: map[domain.OperationID]domain.Record{
		domain.OpAuthWhoami: whoamiRecord(keyUserID, "Fixture Team"),
	}}
	admin := &fakeAdmin{reply: ackReply()}
	svc, _ := adminService(t, domain.AccessAdmin, reader, admin)
	raw := map[string]any{"userId": otherUserID}

	if _, err := Administer(context.Background(), svc, domain.OpUsersDelete, strPtr("outline"), raw, Confirmation{Yes: true}); !errors.Is(err, domain.ErrConfirmRequired) {
		t.Fatalf("a missing --confirm: %v", err)
	}
	if _, err := Administer(context.Background(), svc, domain.OpUsersDelete, strPtr("outline"), raw,
		Confirmation{Yes: true, Confirm: strPtr("Ben")}); !errors.Is(err, domain.ErrConfirmMismatch) {
		t.Fatalf("the name, not the ID: %v", err)
	}
	if len(admin.ops) != 0 {
		t.Fatalf("a refused critical call reached the admin port: %v", admin.ops)
	}
	if _, err := Administer(context.Background(), svc, domain.OpUsersDelete, strPtr("outline"), raw,
		Confirmation{Yes: true, Confirm: strPtr(otherUserID)}); err != nil {
		t.Fatalf("the right --confirm: %v", err)
	}
	if len(admin.ops) != 1 || admin.ops[0] != domain.OpUsersDelete {
		t.Fatalf("the admin port was called %v, want users.delete once", admin.ops)
	}
}

// TestAdministerCriticalTeamsDeleteConfirmsTheTeamName checks the canonical target of teams.delete: the team name that
// auth.whoami reads under team.name. The dry run names it, and the code is shown only as its byte count.
func TestAdministerCriticalTeamsDeleteConfirmsTheTeamName(t *testing.T) {
	reader := &fakeReader{reply: domain.NewRecord(), byOp: map[domain.OperationID]domain.Record{
		domain.OpAuthWhoami: whoamiRecord(keyUserID, "Fixture Team"),
	}}
	admin := &fakeAdmin{reply: ackReply()}
	svc, _ := adminService(t, domain.AccessAdmin, reader, admin)
	raw := map[string]any{"code": "CODE-SECRET-1"}

	plan, err := Administer(context.Background(), svc, domain.OpTeamsDelete, strPtr("outline"), raw, Confirmation{DryRun: true})
	if err != nil {
		t.Fatalf("dry run of teams.delete: %v", err)
	}
	if plan.Str("target") != "Fixture Team" {
		t.Fatalf("the plan names target %q, want the team name", plan.Str("target"))
	}
	if containsValue(plan, "CODE-SECRET-1") {
		t.Fatalf("the dry run shows the delete code: %v", plan.Fields())
	}
	if _, err := Administer(context.Background(), svc, domain.OpTeamsDelete, strPtr("outline"), raw,
		Confirmation{Yes: true, Confirm: strPtr("fixture team")}); !errors.Is(err, domain.ErrConfirmMismatch) {
		t.Fatalf("a team name in another case: %v", err)
	}
	if _, err := Administer(context.Background(), svc, domain.OpTeamsDelete, strPtr("outline"), raw,
		Confirmation{Yes: true, Confirm: strPtr("Fixture Team")}); err != nil {
		t.Fatalf("the right --confirm: %v", err)
	}
	if len(admin.ops) != 1 || admin.ops[0] != domain.OpTeamsDelete {
		t.Fatalf("the admin port was called %v, want teams.delete once", admin.ops)
	}
	if got := admin.args[0].String("code"); got != "CODE-SECRET-1" {
		t.Fatalf("the code sent = %q, want the code the caller gave", got)
	}
}

// TestAdministerCriticalGroupsDeleteAllConfirmsTheProviderID checks the canonical target of groups.deleteAll: the sign-in
// provider ID that authentication-providers.info reads, not a group.
func TestAdministerCriticalGroupsDeleteAllConfirmsTheProviderID(t *testing.T) {
	const providerID = "55555555-5555-4555-8555-555555555555"
	reader := &fakeReader{reply: domain.NewRecord(), byOp: map[domain.OperationID]domain.Record{
		domain.OpAuthenticationProvidersGet: providerRecord(providerID, "Google"),
	}}
	admin := &fakeAdmin{reply: ackReply()}
	svc, _ := adminService(t, domain.AccessAdmin, reader, admin)
	raw := map[string]any{"authenticationProviderId": providerID}

	plan, err := Administer(context.Background(), svc, domain.OpGroupsDeleteAll, strPtr("outline"), raw, Confirmation{DryRun: true})
	if err != nil {
		t.Fatalf("dry run of groups.deleteAll: %v", err)
	}
	if plan.Str("target") != providerID || plan.Str("targetName") != "Google" {
		t.Fatalf("the plan names %q %q, want the provider ID and its name", plan.Str("target"), plan.Str("targetName"))
	}
	if _, err := Administer(context.Background(), svc, domain.OpGroupsDeleteAll, strPtr("outline"), raw,
		Confirmation{Yes: true, Confirm: strPtr("Google")}); !errors.Is(err, domain.ErrConfirmMismatch) {
		t.Fatalf("the provider name, not its ID: %v", err)
	}
	if _, err := Administer(context.Background(), svc, domain.OpGroupsDeleteAll, strPtr("outline"), raw,
		Confirmation{Yes: true, Confirm: strPtr(providerID)}); err != nil {
		t.Fatalf("the right --confirm: %v", err)
	}
	if len(admin.ops) != 1 || admin.ops[0] != domain.OpGroupsDeleteAll {
		t.Fatalf("the admin port was called %v, want groups.deleteAll once", admin.ops)
	}
}

// TestAdministerTeamsDeleteNeedsItsCodeBeforeAnyRequest checks that the delete code is a required argument, so a call without
// it stops in the binder, before the wiki is read or changed.
func TestAdministerTeamsDeleteNeedsItsCodeBeforeAnyRequest(t *testing.T) {
	reader := &fakeReader{reply: domain.NewRecord(), byOp: map[domain.OperationID]domain.Record{
		domain.OpAuthWhoami: whoamiRecord(keyUserID, "Fixture Team"),
	}}
	admin := &fakeAdmin{reply: ackReply()}
	svc, _ := adminService(t, domain.AccessAdmin, reader, admin)
	_, err := Administer(context.Background(), svc, domain.OpTeamsDelete, strPtr("outline"), nil,
		Confirmation{Yes: true, Confirm: strPtr("Fixture Team")})
	if !errors.Is(err, domain.ErrMissingArgument) {
		t.Fatalf("Administer error = %v, want ErrMissingArgument", err)
	}
	if len(reader.ops) != 0 || len(admin.ops) != 0 {
		t.Fatalf("a call without its code made requests: read %v, admin %v", reader.ops, admin.ops)
	}
}

// TestAdministerDryRunOfAnInvitationNeverShowsTheAddress checks that the plan of users.invite names the e-mail address only
// as a mark: the address appears nowhere in the plan.
func TestAdministerDryRunOfAnInvitationNeverShowsTheAddress(t *testing.T) {
	reader := &fakeReader{reply: domain.NewRecord()}
	admin := &fakeAdmin{reply: domain.NewRecord()}
	svc, _ := adminService(t, domain.AccessAdmin, reader, admin)
	plan, err := Administer(context.Background(), svc, domain.OpUsersInvite, strPtr("outline"),
		map[string]any{"email": "ann@example.com", "name": "Ann", "role": "member"}, Confirmation{DryRun: true})
	if err != nil {
		t.Fatalf("dry run of users.invite: %v", err)
	}
	if containsValue(plan, "ann@example.com") {
		t.Fatalf("the dry run shows the e-mail address: %v", plan.Fields())
	}
	if got := plan.Rec("arguments").Str("email"); got != "<an e-mail address, not shown>" {
		t.Fatalf("the plan shows the e-mail argument as %q", got)
	}
	if len(admin.ops) != 0 {
		t.Fatalf("a dry run reached the admin port: %v", admin.ops)
	}
}

// containsValue reports whether any text of a Record, at any depth, holds the text s.
func containsValue(rec domain.Record, s string) bool {
	for _, f := range rec.Fields() {
		switch v := f.Value.(type) {
		case string:
			if v == s {
				return true
			}
		case domain.Record:
			if containsValue(v, s) {
				return true
			}
		}
	}
	for _, item := range rec.Items() {
		if containsValue(item, s) {
			return true
		}
	}
	return false
}
