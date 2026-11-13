package outline

import (
	"context"
	"errors"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"net/http"
	"testing"
)

// oauthClientUUID, webhookUUID, and revisionUUID are the IDs of the slice c4 cases. They are canonical UUIDs, so the
// positional argument of each row takes them.
const (
	oauthClientUUID = "b5000000-0000-4000-8000-000000000019"
	webhookUUID     = "b6000000-0000-4000-8000-000000000020"
	revisionUUID    = "b7000000-0000-4000-8000-000000000021"
	shareUUID       = "88888888-8888-4888-8888-888888888888"
)

// requestCasesC4 are the request-shape cases of slice c4: the OAuth authorizations and clients, revisions, shares, templates,
// and webhook subscriptions rows. Each body is written by hand from the field reference and the wire
// facts. A redirect list and an event list are sent as JSON arrays, whether the caller gives them as one comma-separated text
// or as a list. The OAuth client's ID is the body's id, and the authorization's is oauthClientId.
var requestCasesC4 = []requestCase{
	{"oauth-authentications.delete", domain.OpOAuthAuthenticationsDelete,
		map[string]any{"oauthClientId": oauthClientUUID}, "/api/oauthAuthentications.delete",
		`{"oauthClientId":"` + oauthClientUUID + `"}`},
	{"oauth-clients.create with a type and a listing", domain.OpOAuthClientsCreate,
		map[string]any{"name": "CI app", "redirectUris": []string{"https://app.example.com/cb"}, "clientType": "public", "published": true},
		"/api/oauthClients.create",
		`{"name":"CI app","redirectUris":["https://app.example.com/cb"],"clientType":"public","published":true}`},
	{"oauth-clients.create with two redirects", domain.OpOAuthClientsCreate,
		map[string]any{"name": "CI app", "redirectUris": "https://a.example.test/cb,https://b.example.test/cb"},
		"/api/oauthClients.create",
		`{"name":"CI app","redirectUris":["https://a.example.test/cb","https://b.example.test/cb"]}`},
	{"oauth-clients.delete", domain.OpOAuthClientsDelete,
		map[string]any{"oauthClientId": oauthClientUUID}, "/api/oauthClients.delete", `{"id":"` + oauthClientUUID + `"}`},
	{"oauth-clients.rotate-secret", domain.OpOAuthClientsRotateSecret,
		map[string]any{"oauthClientId": oauthClientUUID}, "/api/oauthClients.rotate_secret", `{"id":"` + oauthClientUUID + `"}`},
	{"oauth-clients.update", domain.OpOAuthClientsUpdate,
		map[string]any{"oauthClientId": oauthClientUUID, "name": "Renamed", "published": false},
		"/api/oauthClients.update", `{"id":"` + oauthClientUUID + `","name":"Renamed","published":false}`},
	{"revisions.delete", domain.OpRevisionsDelete,
		map[string]any{"revisionId": revisionUUID}, "/api/revisions.delete", `{"id":"` + revisionUUID + `"}`},
	{"shares.create of a document", domain.OpSharesCreate,
		map[string]any{"documentId": "doc-1", "published": true, "allowIndexing": false},
		"/api/shares.create", `{"documentId":"doc-1","published":true,"allowIndexing":false}`},
	{"shares.create of a collection", domain.OpSharesCreate,
		map[string]any{"collectionId": oauthClientUUID, "includeChildDocuments": true},
		"/api/shares.create", `{"collectionId":"` + oauthClientUUID + `","includeChildDocuments":true}`},
	{"shares.revoke", domain.OpSharesRevoke,
		map[string]any{"shareId": shareUUID}, "/api/shares.revoke", `{"id":"` + shareUUID + `"}`},
	{"shares.update", domain.OpSharesUpdate,
		map[string]any{"shareId": shareUUID, "published": false, "title": "Plan for review"},
		"/api/shares.update", `{"id":"` + shareUUID + `","published":false,"title":"Plan for review"}`},
	{"templates.delete", domain.OpTemplatesDelete,
		map[string]any{"templateId": "tpl-1"}, "/api/templates.delete", `{"id":"tpl-1"}`},
	{"webhook-subscriptions.create", domain.OpWebhookSubscriptionsCreate,
		map[string]any{"url": "https://hooks.example.test/SYNTHETICPATHTOKEN", "name": "CI events", "events": "documents.create,*"},
		"/api/webhookSubscriptions.create",
		`{"name":"CI events","url":"https://hooks.example.test/SYNTHETICPATHTOKEN","events":["documents.create","*"]}`},
	{"webhook-subscriptions.delete", domain.OpWebhookSubscriptionsDelete,
		map[string]any{"webhookSubscriptionId": webhookUUID}, "/api/webhookSubscriptions.delete", `{"id":"` + webhookUUID + `"}`},
	{"webhook-subscriptions.update", domain.OpWebhookSubscriptionsUpdate,
		map[string]any{"webhookSubscriptionId": webhookUUID, "url": "https://new.example.test/x", "name": "CI events", "events": "documents.update"},
		"/api/webhookSubscriptions.update",
		`{"id":"` + webhookUUID + `","name":"CI events","url":"https://new.example.test/x","events":["documents.update"]}`},
}

// TestSliceC4EveryRowIsRefusedAtAWriteSourceBeforeAnyRequest checks layer 3 for each row of the slice:
// a source below admin never sends the change.
func TestSliceC4EveryRowIsRefusedAtAWriteSourceBeforeAnyRequest(t *testing.T) {
	for _, tc := range requestCasesC4 {
		t.Run(tc.name, func(t *testing.T) {
			gw, rec := recordingGatewayAt(t, domain.AccessWrite)
			if _, err := gw.Administer(context.Background(), tc.op, domain.NewArgs(tc.args)); err == nil {
				t.Fatal("an admin call on a write source succeeded")
			}
			if n := len(rec.recorded()); n != 0 {
				t.Fatalf("requests = %d, want 0", n)
			}
		})
	}
}

// caseArgsC4 returns the arguments of the case of an operation of this slice.
func caseArgsC4(t *testing.T, op domain.OperationID) map[string]any {
	t.Helper()
	for _, tc := range requestCasesC4 {
		if tc.op == op {
			return tc.args
		}
	}
	t.Fatalf("no request case for %s", op)
	return nil
}

// TestSliceC4AckWithoutSuccessIsAnOutcomeUnknown checks that an acknowledgement of the slice that the wiki did not confirm
// is an unknown outcome, and that the change is never retried (finding m1).
func TestSliceC4AckWithoutSuccessIsAnOutcomeUnknown(t *testing.T) {
	cases := []struct {
		op    domain.OperationID
		reply string
	}{
		{domain.OpOAuthAuthenticationsDelete, `{"ok":true}`},
		{domain.OpOAuthClientsDelete, `{"status":200}`},
		{domain.OpRevisionsDelete, `{"success":false}`},
		{domain.OpSharesRevoke, `{"ok":true}`},
		{domain.OpTemplatesDelete, `{"status":200}`},
		{domain.OpWebhookSubscriptionsDelete, `{"ok":true}`},
	}
	for _, tc := range cases {
		r := adminRig(t)
		method := adminEndpoints[tc.op].method
		r.stub.SetReply(method, http.StatusOK, tc.reply)
		_, err := r.gw.Administer(context.Background(), tc.op, domain.NewArgs(caseArgsC4(t, tc.op)))
		if !errors.Is(err, domain.ErrWriteOutcomeUnknown) {
			t.Fatalf("%s with %s: error = %v, want ErrWriteOutcomeUnknown", tc.op, tc.reply, err)
		}
		if n := r.stub.Calls(method); n != 1 {
			t.Fatalf("%s: calls = %d, want 1", tc.op, n)
		}
	}
}
