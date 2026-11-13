package sources

import (
	"context"
	"errors"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// oauthClientID is the ID of the OAuth client in the tests below, and oauthClientName is its name.
const (
	oauthClientID   = "b5000000-0000-4000-8000-000000000019"
	oauthClientName = "CI app"
)

// oauthClientReply is the reply of oauthClients.info for one client: the fields of the OAuth client row. The client secret is
// never a field of it, because the adapter's projection keeps no secret.
func oauthClientReply() domain.Record {
	return domain.NewRecord(
		domain.Field{Name: "id", Value: oauthClientID},
		domain.Field{Name: "name", Value: oauthClientName},
		domain.Field{Name: "description", Value: nil},
		domain.Field{Name: "developerName", Value: nil},
		domain.Field{Name: "developerUrl", Value: nil},
		domain.Field{Name: "clientId", Value: "client-1"},
		domain.Field{Name: "clientType", Value: "confidential"},
		domain.Field{Name: "redirectUris", Value: []any{"https://app.example.com/cb"}},
		domain.Field{Name: "published", Value: false},
		domain.Field{Name: "lastActiveAt", Value: nil},
		domain.Field{Name: "createdAt", Value: "2026-01-01T00:00:00Z"},
		domain.Field{Name: "updatedAt", Value: "2026-01-02T00:00:00Z"},
	)
}

// webhookReply is the reply of one webhook subscription: the fields of the webhook row, with the host of its URL only.
func webhookReply() domain.Record {
	return domain.NewRecord(
		domain.Field{Name: "id", Value: "b6000000-0000-4000-8000-000000000020"},
		domain.Field{Name: "name", Value: "CI events"},
		domain.Field{Name: "urlHost", Value: "new.example.test"},
		domain.Field{Name: "events", Value: []any{"documents.create"}},
		domain.Field{Name: "enabled", Value: true},
		domain.Field{Name: "createdById", Value: nil},
		domain.Field{Name: "createdAt", Value: "2026-01-01T00:00:00Z"},
		domain.Field{Name: "updatedAt", Value: "2026-01-02T00:00:00Z"},
	)
}

// TestAdministerCriticalOAuthClientCreateConfirmsTheNewName checks the canonical target of oauth-clients.create: the new client name, typed exactly. No client exists before the call, so the dry run reads nothing, and the name is the
// target; a --confirm in another case, or another name, is a mismatch, and only the exact name sends the request.
func TestAdministerCriticalOAuthClientCreateConfirmsTheNewName(t *testing.T) {
	reader := &fakeReader{reply: domain.NewRecord()}
	admin := &fakeAdmin{reply: oauthClientReply()}
	svc, _ := adminService(t, domain.AccessAdmin, reader, admin)
	raw := map[string]any{"name": oauthClientName, "redirectUris": "https://app.example.com/cb"}

	plan, err := Administer(context.Background(), svc, domain.OpOAuthClientsCreate, strPtr("outline"), raw, Confirmation{DryRun: true})
	if err != nil {
		t.Fatalf("dry run of oauth-clients.create: %v", err)
	}
	if plan.Str("target") != oauthClientName {
		t.Fatalf("the plan names target %q, want the new name", plan.Str("target"))
	}
	if len(reader.ops) != 0 {
		t.Fatalf("the dry run read %v, want no read: the client does not exist yet", reader.ops)
	}
	if _, err := Administer(context.Background(), svc, domain.OpOAuthClientsCreate, strPtr("outline"), raw, Confirmation{Yes: true}); !errors.Is(err, domain.ErrConfirmRequired) {
		t.Fatalf("a missing --confirm: %v", err)
	}
	for _, wrong := range []string{"ci app", "CI App", "CI app ", "other"} {
		if _, err := Administer(context.Background(), svc, domain.OpOAuthClientsCreate, strPtr("outline"), raw,
			Confirmation{Yes: true, Confirm: strPtr(wrong)}); !errors.Is(err, domain.ErrConfirmMismatch) {
			t.Fatalf("--confirm %q: %v, want a mismatch", wrong, err)
		}
	}
	if len(admin.ops) != 0 {
		t.Fatalf("a refused critical call reached the admin port: %v", admin.ops)
	}
	if _, err := Administer(context.Background(), svc, domain.OpOAuthClientsCreate, strPtr("outline"), raw,
		Confirmation{Yes: true, Confirm: strPtr(oauthClientName)}); err != nil {
		t.Fatalf("the exact --confirm: %v", err)
	}
	if len(admin.ops) != 1 || admin.ops[0] != domain.OpOAuthClientsCreate {
		t.Fatalf("the admin port was called %v, want oauth-clients.create once", admin.ops)
	}
}

// TestAdministerCriticalOAuthClientRotateConfirmsTheClientID checks the canonical target of oauth-clients.rotate-secret: the
// OAuth client ID that the target read returns. The dry run names the client and its name, a wrong value is a mismatch, and
// only the ID sends the request.
func TestAdministerCriticalOAuthClientRotateConfirmsTheClientID(t *testing.T) {
	reader := &fakeReader{reply: oauthClientReply()}
	admin := &fakeAdmin{reply: oauthClientReply()}
	svc, _ := adminService(t, domain.AccessAdmin, reader, admin)
	raw := map[string]any{"oauthClientId": oauthClientID}

	plan, err := Administer(context.Background(), svc, domain.OpOAuthClientsRotateSecret, strPtr("outline"), raw, Confirmation{DryRun: true})
	if err != nil {
		t.Fatalf("dry run of oauth-clients.rotate-secret: %v", err)
	}
	if plan.Str("target") != oauthClientID || plan.Str("targetName") != oauthClientName {
		t.Fatalf("the plan names target %q and name %q, want the ID and the name", plan.Str("target"), plan.Str("targetName"))
	}
	if _, err := Administer(context.Background(), svc, domain.OpOAuthClientsRotateSecret, strPtr("outline"), raw,
		Confirmation{Yes: true, Confirm: strPtr(oauthClientName)}); !errors.Is(err, domain.ErrConfirmMismatch) {
		t.Fatalf("the name given as --confirm: %v, want a mismatch", err)
	}
	if len(admin.ops) != 0 {
		t.Fatalf("a refused rotation reached the admin port: %v", admin.ops)
	}
	if _, err := Administer(context.Background(), svc, domain.OpOAuthClientsRotateSecret, strPtr("outline"), raw,
		Confirmation{Yes: true, Confirm: strPtr(oauthClientID)}); err != nil {
		t.Fatalf("the right --confirm: %v", err)
	}
	if len(admin.ops) != 1 || admin.ops[0] != domain.OpOAuthClientsRotateSecret {
		t.Fatalf("the admin port was called %v, want oauth-clients.rotate-secret once", admin.ops)
	}
}

// TestAdministerCriticalOAuthClientRotateStopsWhenTheClientDoesNotResolve checks that a rotation of a client the wiki does not
// hold stops before any change: the target does not resolve, so the call never reaches the admin port.
func TestAdministerCriticalOAuthClientRotateStopsWhenTheClientDoesNotResolve(t *testing.T) {
	reader := &fakeReader{err: domain.ErrNotFound}
	admin := &fakeAdmin{reply: oauthClientReply()}
	svc, _ := adminService(t, domain.AccessAdmin, reader, admin)
	_, err := Administer(context.Background(), svc, domain.OpOAuthClientsRotateSecret, strPtr("outline"),
		map[string]any{"oauthClientId": oauthClientID}, Confirmation{Yes: true, Confirm: strPtr(oauthClientID)})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("a missing client: %v, want ErrNotFound", err)
	}
	if len(admin.ops) != 0 {
		t.Fatalf("a rotation of a missing client reached the admin port: %v", admin.ops)
	}
}

// TestAdministerCriticalWebhookCreateConfirmsTheHostOfItsURL checks the canonical target of webhook-subscriptions.create: the
// host of the destination URL, lower-cased. No subscription exists before the call, so the dry run reads nothing and shows the
// host alone, and a --confirm with the path, or another host, is a mismatch.
func TestAdministerCriticalWebhookCreateConfirmsTheHostOfItsURL(t *testing.T) {
	reader := &fakeReader{reply: domain.NewRecord()}
	admin := &fakeAdmin{reply: webhookReply()}
	svc, _ := adminService(t, domain.AccessAdmin, reader, admin)
	raw := map[string]any{"url": "https://hooks.example.test/SYNTHETICPATHTOKEN", "name": "CI events", "events": "documents.create"}

	plan, err := Administer(context.Background(), svc, domain.OpWebhookSubscriptionsCreate, strPtr("outline"), raw, Confirmation{DryRun: true})
	if err != nil {
		t.Fatalf("dry run of webhook-subscriptions.create: %v", err)
	}
	if plan.Str("target") != "hooks.example.test" {
		t.Fatalf("the plan names target %q, want the lower-cased host", plan.Str("target"))
	}
	if containsValue(plan, "SYNTHETICPATHTOKEN") {
		t.Fatalf("the dry run shows the path of the address: %v", plan.Fields())
	}
	if len(reader.ops) != 0 {
		t.Fatalf("the dry run read %v, want no read", reader.ops)
	}
	for _, wrong := range []string{"SYNTHETICPATHTOKEN", "https://hooks.example.test/SYNTHETICPATHTOKEN", "other.example.test"} {
		if _, err := Administer(context.Background(), svc, domain.OpWebhookSubscriptionsCreate, strPtr("outline"), raw,
			Confirmation{Yes: true, Confirm: strPtr(wrong)}); !errors.Is(err, domain.ErrConfirmMismatch) {
			t.Fatalf("--confirm %q: %v, want a mismatch", wrong, err)
		}
	}
	if _, err := Administer(context.Background(), svc, domain.OpWebhookSubscriptionsCreate, strPtr("outline"), raw, Confirmation{Yes: true}); !errors.Is(err, domain.ErrConfirmRequired) {
		t.Fatalf("a missing --confirm: %v", err)
	}
	if len(admin.ops) != 0 {
		t.Fatalf("a refused critical call reached the admin port: %v", admin.ops)
	}
	if _, err := Administer(context.Background(), svc, domain.OpWebhookSubscriptionsCreate, strPtr("outline"), raw,
		Confirmation{Yes: true, Confirm: strPtr("hooks.example.test")}); err != nil {
		t.Fatalf("the host as --confirm: %v", err)
	}
	if len(admin.ops) != 1 || admin.ops[0] != domain.OpWebhookSubscriptionsCreate {
		t.Fatalf("the admin port was called %v, want webhook-subscriptions.create once", admin.ops)
	}
}

// TestAdministerCriticalWebhookUpdateConfirmsTheNewHost checks the canonical target of webhook-subscriptions.update: the host of
// the new destination URL. The wiki has no read of one subscription, so nothing is read first; the host the subscription held
// before is not the value to type.
func TestAdministerCriticalWebhookUpdateConfirmsTheNewHost(t *testing.T) {
	reader := &fakeReader{reply: domain.NewRecord()}
	admin := &fakeAdmin{reply: webhookReply()}
	svc, _ := adminService(t, domain.AccessAdmin, reader, admin)
	raw := map[string]any{
		"webhookSubscriptionId": "b6000000-0000-4000-8000-000000000020",
		"url":                   "https://new.example.test/SYNTHETICPATHTOKEN",
		"name":                  "CI events",
		"events":                "documents.create",
	}

	plan, err := Administer(context.Background(), svc, domain.OpWebhookSubscriptionsUpdate, strPtr("outline"), raw, Confirmation{DryRun: true})
	if err != nil {
		t.Fatalf("dry run of webhook-subscriptions.update: %v", err)
	}
	if plan.Str("target") != "new.example.test" {
		t.Fatalf("the plan names target %q, want the host of the new URL", plan.Str("target"))
	}
	if len(reader.ops) != 0 {
		t.Fatalf("the dry run read %v, want no read: there is no read of one subscription", reader.ops)
	}
	if _, err := Administer(context.Background(), svc, domain.OpWebhookSubscriptionsUpdate, strPtr("outline"), raw,
		Confirmation{Yes: true, Confirm: strPtr("hooks.example.test")}); !errors.Is(err, domain.ErrConfirmMismatch) {
		t.Fatalf("the host held before: %v, want a mismatch", err)
	}
	if len(admin.ops) != 0 {
		t.Fatalf("a refused critical call reached the admin port: %v", admin.ops)
	}
	if _, err := Administer(context.Background(), svc, domain.OpWebhookSubscriptionsUpdate, strPtr("outline"), raw,
		Confirmation{Yes: true, Confirm: strPtr("new.example.test")}); err != nil {
		t.Fatalf("the host of the new URL: %v", err)
	}
	if len(admin.ops) != 1 || admin.ops[0] != domain.OpWebhookSubscriptionsUpdate {
		t.Fatalf("the admin port was called %v, want webhook-subscriptions.update once", admin.ops)
	}
}

// TestAdministerSliceC4NeedsYesAndAdmin checks the two gates every row of slice c4 shares: a change without --yes is refused
// before the admin port, and a source below admin is refused before any change, with the level error.
func TestAdministerSliceC4NeedsYesAndAdmin(t *testing.T) {
	cases := []struct {
		op  domain.OperationID
		raw map[string]any
	}{
		{domain.OpOAuthAuthenticationsDelete, map[string]any{"oauthClientId": oauthClientID}},
		{domain.OpOAuthClientsDelete, map[string]any{"oauthClientId": oauthClientID}},
		{domain.OpOAuthClientsUpdate, map[string]any{"oauthClientId": oauthClientID, "name": "x"}},
		{domain.OpRevisionsDelete, map[string]any{"revisionId": "55555555-5555-4555-8555-555555555555"}},
		{domain.OpSharesCreate, map[string]any{"documentId": "doc-1"}},
		{domain.OpSharesRevoke, map[string]any{"shareId": "88888888-8888-4888-8888-888888888888"}},
		{domain.OpSharesUpdate, map[string]any{"shareId": "88888888-8888-4888-8888-888888888888", "published": true}},
		{domain.OpTemplatesDelete, map[string]any{"templateId": "tpl-1"}},
		{domain.OpWebhookSubscriptionsDelete, map[string]any{"webhookSubscriptionId": "b6000000-0000-4000-8000-000000000020"}},
	}
	for _, tc := range cases {
		reader := &fakeReader{reply: domain.NewRecord()}
		admin := &fakeAdmin{reply: domain.NewRecord()}
		svc, _ := adminService(t, domain.AccessAdmin, reader, admin)
		if _, err := Administer(context.Background(), svc, tc.op, strPtr("outline"), tc.raw, Confirmation{}); !errors.Is(err, domain.ErrYesRequired) {
			t.Errorf("%s without --yes: %v, want ErrYesRequired", tc.op, err)
		}
		if len(admin.ops) != 0 {
			t.Errorf("%s without --yes reached the admin port: %v", tc.op, admin.ops)
		}
		lowAdmin := &fakeAdmin{reply: domain.NewRecord()}
		low, _ := adminService(t, domain.AccessWrite, &fakeReader{reply: domain.NewRecord()}, lowAdmin)
		if _, err := Administer(context.Background(), low, tc.op, strPtr("outline"), tc.raw, Confirmation{Yes: true}); err == nil {
			t.Errorf("%s on a write source succeeded", tc.op)
		}
		if len(lowAdmin.ops) != 0 {
			t.Errorf("%s on a write source reached the admin port: %v", tc.op, lowAdmin.ops)
		}
	}
}
