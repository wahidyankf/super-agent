package outline

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// c2Reply is one reply of a slice c2 row, the arguments that call it, and what its projection must hold. An object reply
// checks one field of the Record. A list reply checks one field of its first item.
type c2Reply struct {
	op    domain.OperationID
	args  map[string]any
	reply string
	list  bool
	field string
	want  string
}

// c2Replies are the replies of the slice c2 rows, in the wire shapes of the router.
// The user objects carry an e-mail address and the team carries preferences, because the router sends them; the projection
// must drop them.
var c2Replies = []c2Reply{
	{domain.OpUsersActivate, map[string]any{"userId": uuidUser},
		`{"data":{"id":"` + uuidUser + `","name":"Ann","email":"ann@example.test","role":"member"},"policies":[]}`, false, "name", "Ann"},
	{domain.OpUsersSuspend, map[string]any{"userId": uuidUser},
		`{"data":{"id":"` + uuidUser + `","name":"Ann","isSuspended":true,"email":"ann@example.test"},"policies":[]}`, false, "isSuspended", "true"},
	{domain.OpUsersUpdateRole, map[string]any{"userId": uuidUser, "role": "viewer"},
		`{"data":{"id":"` + uuidUser + `","name":"Ann","role":"viewer"},"policies":[]}`, false, "role", "viewer"},
	{domain.OpUsersUpdate, map[string]any{"name": "Ann B"},
		`{"data":{"id":"` + uuidUser + `","name":"Ann B","avatarUrl":"https://avatars.example.test/a.png","preferences":{"x":1}}}`,
		false, "name", "Ann B"},
	{domain.OpUsersInvite, map[string]any{"email": "ann@example.com", "name": "Ann", "role": "member"},
		`{"data":{"sent":1,"unsent":0,"users":[{"id":"` + uuidUser + `","name":"Ann","email":"ann@example.com","role":"member"}]}}`,
		true, "id", uuidUser},
	{domain.OpGroupsAddUser, map[string]any{"groupId": uuidOther, "userId": uuidUser},
		`{"data":{"users":[],"groupMemberships":[{"groupId":"` + uuidOther + `","userId":"` + uuidUser + `","permission":"member","createdAt":"2026-01-04T00:00:00Z"}],"groups":[]}}`,
		true, "userId", uuidUser},
	{domain.OpGroupsCreate, map[string]any{"name": "Support"},
		`{"data":{"id":"` + uuidOther + `","name":"Support","externalId":"idp-9","memberCount":0},"policies":[]}`,
		false, "name", "Support"},
	{domain.OpGroupsUpdateUser, map[string]any{"groupId": uuidOther, "userId": uuidUser, "permission": "admin"},
		`{"data":{"users":[],"groupMemberships":[{"groupId":"` + uuidOther + `","userId":"` + uuidUser + `","permission":"admin"}],"groups":[]}}`,
		true, "permission", "admin"},
	{domain.OpGroupsRemoveUser, map[string]any{"groupId": uuidOther, "userId": uuidUser},
		`{"data":{"groups":[{"id":"` + uuidOther + `","name":"Engineering","externalId":"idp-42"}]}}`, true, "name", "Engineering"},
	{domain.OpGroupsUpdate, map[string]any{"groupId": uuidOther, "name": "Eng"},
		`{"data":{"id":"` + uuidOther + `","name":"Eng","externalId":"idp-42"},"policies":[]}`, false, "name", "Eng"},
	{domain.OpTeamsUpdate, map[string]any{"name": "Fixture Team"},
		`{"data":{"id":"` + uuidOther + `","name":"Fixture Team","sharing":false,"defaultUserRole":"member",` +
			`"preferences":{"separateEditMode":true},"guidanceMCP":"x","plan":"enterprise","entitlements":["a"],"avatarUrl":null},` +
			`"policies":[]}`, false, "name", "Fixture Team"},
}

// deniedC2Keys are keys that a projection of a slice c2 reply must never hold: the e-mail address, the avatar, the
// preferences, the plan, the entitlements, the MCP guidance, and any secret.
var deniedC2Keys = map[string]bool{
	"email": true, "avatarUrl": true, "preferences": true, "plan": true, "entitlements": true, "guidanceMCP": true,
	"secret": true, "clientSecret": true, "value": true, "token": true, "externalId": true,
}

// deniedKey returns the first denied key of a Record, its fields and its items included, or "" when it holds none.
func deniedKey(rec domain.Record) string {
	for _, f := range rec.Fields() {
		if deniedC2Keys[f.Name] {
			return f.Name
		}
		if nested, ok := f.Value.(domain.Record); ok {
			if k := deniedKey(nested); k != "" {
				return k
			}
		}
	}
	for _, item := range rec.Items() {
		if k := deniedKey(item); k != "" {
			return k
		}
	}
	return ""
}

// TestSliceC2RepliesAreReadLikeTheirRows checks that each reply of a slice c2 row is projected under the name the row
// declares, so a caller reads the object or the first item of the list the row names.
func TestSliceC2RepliesAreReadLikeTheirRows(t *testing.T) {
	for _, tc := range c2Replies {
		t.Run(string(tc.op), func(t *testing.T) {
			r := adminRig(t)
			r.stub.SetReply(string(adminEndpoints[tc.op].method), http.StatusOK, tc.reply)
			out, err := r.gw.Write(context.Background(), tc.op, domain.NewArgs(tc.args))
			if err != nil {
				t.Fatalf("Write = %v", err)
			}
			if tc.list {
				items := out.Items()
				if len(items) == 0 {
					t.Fatalf("the reply holds no item: %+v", out)
				}
				if got := items[0].Str(tc.field); got != tc.want {
					t.Fatalf("first item field %s = %q, want %q", tc.field, got, tc.want)
				}
				return
			}
			if v, _ := out.Get(tc.field); fmt.Sprint(v) != tc.want {
				t.Fatalf("field %s = %v, want %q", tc.field, v, tc.want)
			}
		})
	}
}

// TestSliceC2ProjectionsDropTheEmailAddressesAndThePrivateSettings checks the redaction of the users, groups, and teams
// replies: an e-mail address, an avatar, the preferences, the plan, the entitlements, the MCP guidance,
// and any secret never reach the record, however the router sent them.
func TestSliceC2ProjectionsDropTheEmailAddressesAndThePrivateSettings(t *testing.T) {
	for _, tc := range c2Replies {
		t.Run(string(tc.op), func(t *testing.T) {
			r := adminRig(t)
			r.stub.SetReply(string(adminEndpoints[tc.op].method), http.StatusOK, tc.reply)
			out, err := r.gw.Write(context.Background(), tc.op, domain.NewArgs(tc.args))
			if err != nil {
				t.Fatalf("Write = %v", err)
			}
			if k := deniedKey(out); k != "" {
				t.Fatalf("the projection of %s holds the key %q, which the redaction rules drop", tc.op, k)
			}
		})
	}
}

// TestSliceC2AckWithoutSuccessIsAnOutcomeUnknown checks that each acknowledgement of a slice c2 row is a success only when
// the reply says success true (the wire facts name {success:true} for these rows, and never {ok:true}). A reply without it
// is an outcome that is not known, and the change is not retried.
func TestSliceC2AckWithoutSuccessIsAnOutcomeUnknown(t *testing.T) {
	acks := []struct {
		op   domain.OperationID
		args map[string]any
	}{
		{domain.OpUsersDelete, map[string]any{"userId": uuidUser}},
		{domain.OpUsersRequestDelete, nil},
		{domain.OpUsersResendInvite, map[string]any{"userId": uuidUser}},
		{domain.OpGroupsDelete, map[string]any{"groupId": uuidOther}},
		{domain.OpGroupsDeleteAll, map[string]any{"authenticationProviderId": uuidOther}},
		{domain.OpTeamsDelete, map[string]any{"code": "ABC123"}},
		{domain.OpTeamsRequestDelete, nil},
	}
	replies := []string{`{"ok":true}`, `{"status":200}`, `{"success":false}`, `{}`}
	for _, ack := range acks {
		for _, reply := range replies {
			r := adminRig(t)
			r.stub.SetReply(string(adminEndpoints[ack.op].method), http.StatusOK, reply)
			_, err := r.gw.Write(context.Background(), ack.op, domain.NewArgs(ack.args))
			if !errors.Is(err, domain.ErrWriteOutcomeUnknown) {
				t.Fatalf("%s with %s: error = %v, want ErrWriteOutcomeUnknown", ack.op, reply, err)
			}
			if n := r.stub.Calls(string(adminEndpoints[ack.op].method)); n != 1 {
				t.Fatalf("%s: calls = %d, want 1 (an admin change is never retried)", ack.op, n)
			}
		}
	}
}
