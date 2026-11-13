package outline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/tests/support/outlinestub"
)

// requestCasesC are the request-shape cases of the stage c operations of this slice (ADR 007 section 3.6). Each body is
// the exact JSON the call sends, written by hand from the field reference. A body names only the keys the caller set,
// and the wire names are Outline's: id for the document or collection, and the group or user key.
var requestCasesC1 = []requestCase{
	{"documents.add-group", domain.OpDocumentsAddGroup,
		map[string]any{"documentId": "doc-1", "groupId": uuidOther, "permission": "read_write"},
		"/api/documents.add_group", `{"id":"doc-1","groupId":"` + uuidOther + `","permission":"read_write"}`},
	{"documents.add-user", domain.OpDocumentsAddUser,
		map[string]any{"documentId": "doc-1", "userId": uuidUser},
		"/api/documents.add_user", `{"id":"doc-1","userId":"` + uuidUser + `"}`},
	{"documents.archive", domain.OpDocumentsArchive,
		map[string]any{"documentId": "doc-1", "reason": "old"},
		"/api/documents.archive", `{"id":"doc-1","reason":"old"}`},
	{"documents.delete soft", domain.OpDocumentsDelete,
		map[string]any{"documentId": "doc-1"}, "/api/documents.delete", `{"id":"doc-1"}`},
	{"documents.delete permanent", domain.OpDocumentsDelete,
		map[string]any{"documentId": "doc-1", "permanent": true, "reason": "copy"},
		"/api/documents.delete", `{"id":"doc-1","permanent":true,"reason":"copy"}`},
	{"documents.empty-trash", domain.OpDocumentsEmptyTrash,
		nil, "/api/documents.empty_trash", `{}`},
	{"documents.remove-group", domain.OpDocumentsRemoveGroup,
		map[string]any{"documentId": "doc-1", "groupId": uuidOther}, "/api/documents.remove_group",
		`{"id":"doc-1","groupId":"` + uuidOther + `"}`},
	{"documents.remove-user", domain.OpDocumentsRemoveUser,
		map[string]any{"documentId": "doc-1", "userId": uuidUser}, "/api/documents.remove_user",
		`{"id":"doc-1","userId":"` + uuidUser + `"}`},
	{"collections.add-group", domain.OpCollectionsAddGroup,
		map[string]any{"collectionId": "col-1", "groupId": uuidOther}, "/api/collections.add_group",
		`{"id":"col-1","groupId":"` + uuidOther + `"}`},
	{"collections.add-user", domain.OpCollectionsAddUser,
		map[string]any{"collectionId": "col-1", "userId": uuidUser, "permission": "admin"}, "/api/collections.add_user",
		`{"id":"col-1","userId":"` + uuidUser + `","permission":"admin"}`},
	{"collections.archive", domain.OpCollectionsArchive,
		map[string]any{"collectionId": "col-1"}, "/api/collections.archive", `{"id":"col-1"}`},
	{"collections.delete", domain.OpCollectionsDelete,
		map[string]any{"collectionId": "col-1", "reason": "retired"}, "/api/collections.delete",
		`{"id":"col-1","reason":"retired"}`},
	{"collections.export-all", domain.OpCollectionsExportAll,
		map[string]any{"format": "json", "includeAttachments": false, "includePrivate": true},
		"/api/collections.export_all", `{"format":"json","includeAttachments":false,"includePrivate":true}`},
	{"collections.remove-group", domain.OpCollectionsRemoveGroup,
		map[string]any{"collectionId": "col-1", "groupId": uuidOther}, "/api/collections.remove_group",
		`{"id":"col-1","groupId":"` + uuidOther + `"}`},
	{"collections.remove-user", domain.OpCollectionsRemoveUser,
		map[string]any{"collectionId": "col-1", "userId": uuidUser}, "/api/collections.remove_user",
		`{"id":"col-1","userId":"` + uuidUser + `"}`},
}

// requestCasesC2 are the request-shape cases of slice c2: the users, groups, and teams rows (ADR 007 section 3.6). Each body is
// written by hand from the field reference and the wire facts. users.invite sends one invitation in its list, and the
// e-mail address is sent in the body only; it is never in a reply that Retriever returns.
var requestCasesC2 = []requestCase{
	{"users.activate", domain.OpUsersActivate,
		map[string]any{"userId": uuidUser}, "/api/users.activate", `{"id":"` + uuidUser + `"}`},
	{"users.delete", domain.OpUsersDelete,
		map[string]any{"userId": uuidUser}, "/api/users.delete", `{"id":"` + uuidUser + `"}`},
	{"users.invite", domain.OpUsersInvite,
		map[string]any{"email": "ann@example.com", "name": "Ann", "role": "member"}, "/api/users.invite",
		`{"invites":[{"email":"ann@example.com","name":"Ann","role":"member"}]}`},
	{"users.invite with suppressEmail", domain.OpUsersInvite,
		map[string]any{"email": "ann@example.com", "name": "Ann", "role": "viewer", "suppressEmail": true},
		"/api/users.invite", `{"invites":[{"email":"ann@example.com","name":"Ann","role":"viewer"}],"suppressEmail":true}`},
	{"users.request-delete", domain.OpUsersRequestDelete,
		nil, "/api/users.requestDelete", `{}`},
	{"users.resend-invite", domain.OpUsersResendInvite,
		map[string]any{"userId": uuidUser}, "/api/users.resendInvite", `{"id":"` + uuidUser + `"}`},
	{"users.suspend", domain.OpUsersSuspend,
		map[string]any{"userId": uuidUser}, "/api/users.suspend", `{"id":"` + uuidUser + `"}`},
	{"users.update", domain.OpUsersUpdate,
		map[string]any{"userId": uuidUser, "name": "Ann B", "language": "en_US", "timezone": "Asia/Jakarta"},
		"/api/users.update", `{"id":"` + uuidUser + `","name":"Ann B","language":"en_US","timezone":"Asia/Jakarta"}`},
	{"users.update of the own profile", domain.OpUsersUpdate,
		map[string]any{"name": "Ann B"}, "/api/users.update", `{"name":"Ann B"}`},
	{"users.update-role", domain.OpUsersUpdateRole,
		map[string]any{"userId": uuidUser, "role": "viewer"}, "/api/users.update_role",
		`{"id":"` + uuidUser + `","role":"viewer"}`},
	{"groups.add-user", domain.OpGroupsAddUser,
		map[string]any{"groupId": uuidOther, "userId": uuidUser, "permission": "admin"}, "/api/groups.add_user",
		`{"id":"` + uuidOther + `","userId":"` + uuidUser + `","permission":"admin"}`},
	{"groups.create", domain.OpGroupsCreate,
		map[string]any{"name": "Support", "description": "Help", "color": "#3a7bd5", "icon": "book", "disableMentions": true},
		"/api/groups.create", `{"name":"Support","description":"Help","color":"#3a7bd5","icon":"book","disableMentions":true}`},
	{"groups.delete", domain.OpGroupsDelete,
		map[string]any{"groupId": uuidOther}, "/api/groups.delete", `{"id":"` + uuidOther + `"}`},
	{"groups.delete-all", domain.OpGroupsDeleteAll,
		map[string]any{"authenticationProviderId": uuidOther}, "/api/groups.deleteAll",
		`{"authenticationProviderId":"` + uuidOther + `"}`},
	{"groups.remove-user", domain.OpGroupsRemoveUser,
		map[string]any{"groupId": uuidOther, "userId": uuidUser}, "/api/groups.remove_user",
		`{"id":"` + uuidOther + `","userId":"` + uuidUser + `"}`},
	{"groups.update", domain.OpGroupsUpdate,
		map[string]any{"groupId": uuidOther, "name": "Engineering"}, "/api/groups.update",
		`{"id":"` + uuidOther + `","name":"Engineering"}`},
	{"groups.update-user", domain.OpGroupsUpdateUser,
		map[string]any{"groupId": uuidOther, "userId": uuidUser, "permission": "member"}, "/api/groups.update_user",
		`{"id":"` + uuidOther + `","userId":"` + uuidUser + `","permission":"member"}`},
	{"teams.delete", domain.OpTeamsDelete,
		map[string]any{"code": "ABC123"}, "/api/teams.delete", `{"code":"ABC123"}`},
	{"teams.request-delete", domain.OpTeamsRequestDelete,
		nil, "/api/teams.requestDelete", `{}`},
	{"teams.update", domain.OpTeamsUpdate,
		map[string]any{"name": "Fixture Team", "sharing": true, "defaultUserRole": "viewer"}, "/api/teams.update",
		`{"name":"Fixture Team","sharing":true,"defaultUserRole":"viewer"}`},
}

// requestCasesC is every stage c request case that the adapter sends: the cases of slices c1, c2, and c3 (requestCasesC3 is
// in admin_c3_test.go).
var requestCasesC = slices.Concat(requestCasesC1, requestCasesC2, requestCasesC3, requestCasesC4)

// adminRig returns a rig whose gateway is at admin access, so a test can script the stub's reply to an admin call.
func adminRig(t *testing.T) rig {
	t.Helper()
	clock := newFakeClock()
	stub := outlinestub.New(outlinestub.WithClock(clock.Now))
	client := &http.Client{Transport: stub.RoundTripper()}
	gw, err := NewWithBase(testSourceAt(t, testHost, domain.AccessAdmin), testBase, client, &staticCreds{},
		WithMinDelay(0), WithClock(clock.Now), WithSleep(clock.Sleep))
	if err != nil {
		t.Fatalf("NewWithBase: %v", err)
	}
	return rig{stub: stub, clock: clock, gw: gw}
}

func TestEveryAdminSendsItsExactRequest(t *testing.T) {
	for _, tc := range requestCasesC {
		t.Run(tc.name, func(t *testing.T) {
			gw, rec := recordingAdminCall(t, tc.op)
			_, _ = gw.Administer(context.Background(), tc.op, domain.NewArgs(tc.args))
			calls := rec.recorded()
			// A change that reads the object first (adminPreReads) sends that read before the change, so the change is the last call.
			if want := 1 + len(adminPreReadsOf(tc.op)); len(calls) != want {
				t.Fatalf("requests = %d, want %d", len(calls), want)
			}
			change := calls[len(calls)-1]
			if change.Path != tc.path {
				t.Fatalf("path = %q, want %q", change.Path, tc.path)
			}
			var want any
			if err := json.Unmarshal([]byte(tc.body), &want); err != nil {
				t.Fatalf("bad expected body: %v", err)
			}
			if !reflect.DeepEqual(change.Body, want) {
				t.Fatalf("body = %v, want %v", change.Body, want)
			}
		})
	}
}

// TestTheAdminTableHoldsExactlyTheStageCRowsOfThisSlice checks the admin endpoint table against the list of this slice.
// A row of another stage is not in the table, so an unbuilt operation cannot reach the network.
func TestTheAdminTableHoldsExactlyTheStageCRowsOfThisSlice(t *testing.T) {
	var got, want []string
	for op := range adminEndpoints {
		got = append(got, string(op))
	}
	for _, tc := range requestCasesC {
		want = append(want, string(tc.op))
	}
	slices.Sort(got)
	slices.Sort(want)
	want = slices.Compact(want)
	if !slices.Equal(got, want) {
		t.Fatalf("admin endpoint table = %v, want %v", got, want)
	}
}

// TestAdminRefusesAWriteSourceBeforeAnyRequest checks layer 3 of ADR 007 section 3.2: an admin call on a source below admin
// never reaches the network.
func TestAdminRefusesAWriteSourceBeforeAnyRequest(t *testing.T) {
	gw, rec := recordingGatewayAt(t, domain.AccessWrite)
	_, err := gw.Administer(context.Background(), domain.OpDocumentsArchive, domain.NewArgs(map[string]any{"documentId": "doc-1"}))
	if err == nil {
		t.Fatal("an admin call on a write source succeeded")
	}
	if n := len(rec.recorded()); n != 0 {
		t.Fatalf("requests = %d, want 0", n)
	}
}

// TestAdminRepliesAreReadLikeTheirRows checks the reply of each row: an acknowledgement needs success true, a membership
// list is read under its key, a document or collection is projected, and an export answers its file operation.
func TestAdminRepliesAreReadLikeTheirRows(t *testing.T) {
	cases := []struct {
		op     domain.OperationID
		reply  string
		field  string
		expect string
	}{
		{domain.OpDocumentsAddGroup, `{"data":{"groupMemberships":[{"id":"a7000000-0000-4000-8000-000000000001","groupId":"` + uuidOther + `","documentId":"doc-1","permission":"read_write"}]}}`, "", "a7000000-0000-4000-8000-000000000001"},
		{domain.OpDocumentsAddUser, `{"data":{"memberships":[{"id":"a7000000-0000-4000-8000-000000000002","userId":"` + uuidUser + `","documentId":"doc-1"}],"users":[]}}`, "", "a7000000-0000-4000-8000-000000000002"},
		{domain.OpDocumentsArchive, `{"data":{"id":"doc-1","title":"Plan","url":"/doc/plan","collectionId":"col-1"}}`, "title", "Plan"},
		{domain.OpDocumentsDelete, `{"success":true}`, "ok", "true"},
		{domain.OpDocumentsEmptyTrash, `{"success":true}`, "ok", "true"},
		{domain.OpCollectionsArchive, `{"data":{"id":"col-1","name":"Wiki"}}`, "name", "Wiki"},
		{domain.OpCollectionsExportAll, `{"success":true,"data":{"fileOperation":{"id":"a6000000-0000-4000-8000-000000000006","state":"creating"}}}`, "id", "a6000000-0000-4000-8000-000000000006"},
	}
	for _, tc := range cases {
		r := adminRig(t)
		r.stub.SetReply(string(adminEndpoints[tc.op].method), http.StatusOK, tc.reply)
		out, err := r.gw.Administer(context.Background(), tc.op, domain.NewArgs(adminArgsFor(tc.op)))
		if err != nil {
			t.Fatalf("%s: Administer = %v", tc.op, err)
		}
		if v, _ := out.Get(tc.field); tc.field != "" && fmt.Sprint(v) != tc.expect {
			t.Fatalf("%s: field %s = %v, want %q", tc.op, tc.field, v, tc.expect)
		}
		if tc.field == "" {
			items := out.Items()
			if len(items) == 0 || items[0].Str("id") != tc.expect {
				t.Fatalf("%s: the first item of the list is not %q: %+v", tc.op, tc.expect, out)
			}
		}
	}
}

// adminArgsFor returns valid arguments for one admin operation, for a test that does not check the request.
func adminArgsFor(op domain.OperationID) map[string]any {
	for _, tc := range requestCasesC {
		if tc.op == op {
			return tc.args
		}
	}
	return nil
}

// TestAnAdminAckWithoutSuccessIsAnOutcomeUnknown checks that an acknowledgement the wiki did not confirm is never a success
// (finding m1): the reply must say success true, or the call reports that the change may or may not have happened. The
// {ok:true} shape of the old reading is refused too, because the wiki never sends it for an acknowledgement.
func TestAnAdminAckWithoutSuccessIsAnOutcomeUnknown(t *testing.T) {
	cases := []struct {
		op    domain.OperationID
		reply string
	}{
		{domain.OpDocumentsDelete, `{"status":200}`},
		{domain.OpDocumentsDelete, `{"ok":true,"status":200}`},
		{domain.OpDocumentsDelete, `{"success":false}`},
		{domain.OpDocumentsEmptyTrash, `{"ok":true}`},
		{domain.OpDocumentsRemoveGroup, `{"ok":true}`},
		{domain.OpDocumentsRemoveUser, `{"status":200}`},
		{domain.OpCollectionsDelete, `{"ok":true}`},
		{domain.OpCollectionsRemoveGroup, `{"status":200}`},
		{domain.OpCollectionsRemoveUser, `{"ok":true}`},
		// An export also says success true: a reply that holds its file operation but no success is an unknown outcome.
		{domain.OpCollectionsExportAll, `{"data":{"fileOperation":{"id":"a6000000-0000-4000-8000-000000000006","state":"creating"}}}`},
		{domain.OpCollectionsExportAll, `{"ok":true,"data":{"fileOperation":{"id":"a6000000-0000-4000-8000-000000000006","state":"creating"}}}`},
		// A success without the file operation it names is not the router's export reply either.
		{domain.OpCollectionsExportAll, `{"success":true}`},
	}
	for _, tc := range cases {
		r := adminRig(t)
		r.stub.SetReply(string(adminEndpoints[tc.op].method), http.StatusOK, tc.reply)
		_, err := r.gw.Administer(context.Background(), tc.op, domain.NewArgs(adminArgsFor(tc.op)))
		if !errors.Is(err, domain.ErrWriteOutcomeUnknown) {
			t.Fatalf("%s with %s: error = %v, want ErrWriteOutcomeUnknown", tc.op, tc.reply, err)
		}
		if n := r.stub.Calls(string(adminEndpoints[tc.op].method)); n != 1 {
			t.Fatalf("%s: calls = %d, want 1 (an admin change is never retried)", tc.op, n)
		}
	}
}

// TestNoAdminReachesANonLoopbackHost is the proof that no test admin call leaves the in-process stub for a real host (ADR
// 007 sections 3.4 and 3.6, and the strict behaviour gate). Every admin row of this slice is sent through the client at
// admin access; the stub refuses any host that is not a loopback address or a .invalid name, and counts it. Each call must
// succeed and reach the stub once, and the count of refused hosts must stay at zero.
func TestNoAdminReachesANonLoopbackHost(t *testing.T) {
	for _, tc := range requestCasesC {
		t.Run(tc.name, func(t *testing.T) {
			r := adminRig(t)
			method := adminEndpoints[tc.op].method
			// The router-faithful synthetic reply of the method, so the call projects and succeeds.
			r.stub.SetReply(method, http.StatusOK, string(loadSynthetic(t, method)))
			setPreReadReplies(t, r.stub, tc.op)
			// The call must succeed and reach the stub, or the count of refused hosts would prove nothing (finding m6).
			if _, err := r.gw.Administer(context.Background(), tc.op, domain.NewArgs(tc.args)); err != nil {
				t.Fatalf("the admin call did not succeed, so the proof would pass vacuously: %v", err)
			}
			if n := r.stub.Calls(method); n != 1 {
				t.Fatalf("%s calls = %d, want 1", method, n)
			}
			if n := r.stub.Refused(); n != 0 {
				t.Fatalf("an admin call reached a host that is not loopback or .invalid: %d refused", n)
			}
		})
	}
}
