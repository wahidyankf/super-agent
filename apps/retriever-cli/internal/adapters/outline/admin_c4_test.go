package outline

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
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

// TestSliceC4EveryRequestIsSentOnceWithItsExactBody checks, for each case of slice c4, the method path and the exact body of
// the change, and that the change is the only request (the rows of slice c4 read no target at the gateway; the critical
// check is the use case's, and it reads the target there).
func TestSliceC4EveryRequestIsSentOnceWithItsExactBody(t *testing.T) {
	for _, tc := range requestCasesC4 {
		t.Run(tc.name, func(t *testing.T) {
			gw, rec := recordingAdminCall(t, tc.op)
			if _, err := gw.Administer(context.Background(), tc.op, domain.NewArgs(tc.args)); err != nil {
				t.Fatalf("Administer = %v", err)
			}
			calls := rec.recorded()
			if want := 1 + len(adminPreReadsOf(tc.op)); len(calls) != want {
				t.Fatalf("requests = %d, want %d", len(calls), want)
			}
			change := calls[len(calls)-1]
			if change.Path != tc.path {
				t.Fatalf("path = %q, want %q", change.Path, tc.path)
			}
			if !reflect.DeepEqual(change.Body, bodyOf(t, tc.body)) {
				t.Fatalf("body = %v, want %s", change.Body, tc.body)
			}
		})
	}
}

// TestSliceC4EveryRowIsRefusedAtAWriteSourceBeforeAnyRequest checks layer 3 of ADR 007 section 3.2 for each row of the slice:
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

// TestSliceC4RepliesAreReadLikeTheirRows checks the field each row of the slice returns from its reply: an acknowledgement
// holds ok, and an object reply holds its own fields, with the secret and the link of the reply left out.
func TestSliceC4RepliesAreReadLikeTheirRows(t *testing.T) {
	cases := []struct {
		op     domain.OperationID
		field  string
		expect string
	}{
		{domain.OpOAuthAuthenticationsDelete, "ok", "true"},
		{domain.OpOAuthClientsCreate, "name", "CI app"},
		{domain.OpOAuthClientsDelete, "ok", "true"},
		{domain.OpOAuthClientsRotateSecret, "clientId", "client-1"},
		{domain.OpOAuthClientsUpdate, "name", "CI app renamed"},
		{domain.OpRevisionsDelete, "ok", "true"},
		{domain.OpSharesCreate, "documentId", "doc-1"},
		{domain.OpSharesRevoke, "ok", "true"},
		{domain.OpSharesUpdate, "title", "Plan for review"},
		{domain.OpTemplatesDelete, "ok", "true"},
		{domain.OpWebhookSubscriptionsCreate, "name", "CI events"},
		{domain.OpWebhookSubscriptionsDelete, "ok", "true"},
		{domain.OpWebhookSubscriptionsUpdate, "urlHost", "new.example.test"},
	}
	for _, tc := range cases {
		t.Run(string(tc.op), func(t *testing.T) {
			r := adminRig(t)
			method := adminEndpoints[tc.op].method
			r.stub.SetReply(method, http.StatusOK, string(loadSynthetic(t, method)))
			setPreReadReplies(t, r.stub, tc.op)
			out, err := r.gw.Administer(context.Background(), tc.op, domain.NewArgs(caseArgsC4(t, tc.op)))
			if err != nil {
				t.Fatalf("Administer = %v", err)
			}
			if v, _ := out.Get(tc.field); fmt.Sprint(v) != tc.expect {
				t.Fatalf("field %s = %v, want %q", tc.field, v, tc.expect)
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

// TestSliceC4SecretsAndLinksAreNeverProjected is the proof of the redaction of slice c4 at the adapter. Each reply of the
// slice carries a secret or a public link (the OAuth client secret of create, rotate, and update, the signing secret of a
// webhook, and the link of a share). The result of each call must hold none of them, in any field or nested object, and
// its fields must not name them. The sentinels are the ones of the synthetic replies.
func TestSliceC4SecretsAndLinksAreNeverProjected(t *testing.T) {
	sentinels := []string{"SYNTHETICCLIENTSECRET", "SYNTHETICROTATEDSECRET", "SYNTHETICWEBHOOKSECRET", "SYNTHETICSHARELINK"}
	ops := []domain.OperationID{
		domain.OpOAuthClientsCreate, domain.OpOAuthClientsRotateSecret, domain.OpOAuthClientsUpdate,
		domain.OpWebhookSubscriptionsCreate, domain.OpWebhookSubscriptionsUpdate, domain.OpSharesCreate, domain.OpSharesUpdate,
	}
	for _, op := range ops {
		t.Run(string(op), func(t *testing.T) {
			r := adminRig(t)
			method := adminEndpoints[op].method
			r.stub.SetReply(method, http.StatusOK, string(loadSynthetic(t, method)))
			out, err := r.gw.Administer(context.Background(), op, domain.NewArgs(caseArgsC4(t, op)))
			if err != nil {
				t.Fatalf("the call did not succeed, so the proof would pass vacuously: %v", err)
			}
			text := fmt.Sprintf("%#v", out)
			for _, s := range sentinels {
				if strings.Contains(text, s) {
					t.Fatalf("the result of %s holds %s", op, s)
				}
			}
			for _, name := range []string{"clientSecret", "secret", "url", "urlId"} {
				if _, has := out.Get(name); has {
					t.Fatalf("the result of %s has the field %s", op, name)
				}
			}
		})
	}
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
