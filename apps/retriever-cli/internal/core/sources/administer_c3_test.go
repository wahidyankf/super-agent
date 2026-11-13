package sources

import (
	"context"
	"errors"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// integrationID is the ID of the integration in the tests below.
const integrationID = "b4000000-0000-4000-8000-000000000018"

// integrationReply is the reply of integrations.info for one integration: every field of the integration row, with the host
// of its address only (the projection of the adapter keeps no more of the address).
func integrationReply() domain.Record {
	return domain.NewRecord(
		domain.Field{Name: "id", Value: integrationID},
		domain.Field{Name: "type", Value: "analytics"},
		domain.Field{Name: "service", Value: "matomo"},
		domain.Field{Name: "events", Value: []any{}},
		domain.Field{Name: "collectionId", Value: nil},
		domain.Field{Name: "userId", Value: nil},
		domain.Field{Name: "createdAt", Value: "2026-01-01T00:00:00Z"},
		domain.Field{Name: "updatedAt", Value: "2026-01-02T00:00:00Z"},
		domain.Field{Name: "settings", Value: domain.NewRecord(domain.Field{Name: "urlHost", Value: "old.example.test"})},
	)
}

// TestAdministerCriticalIntegrationsCreateConfirmsTheHostOfItsURL checks the canonical target of integrations.create (ADR 007
// section 3.5): the host of the URL setting, lower-cased. No object exists before the call, so the dry run reads nothing and
// shows the host alone; a --confirm in another case, or another host, is a mismatch, and only the host sends the request.
func TestAdministerCriticalIntegrationsCreateConfirmsTheHostOfItsURL(t *testing.T) {
	reader := &fakeReader{reply: domain.NewRecord()}
	admin := &fakeAdmin{reply: integrationReply()}
	svc, _ := adminService(t, domain.AccessAdmin, reader, admin)
	raw := map[string]any{"url": "https://matomo.example.com/SYNTHETICPATHTOKEN", "type": "analytics", "service": "matomo"}

	plan, err := Administer(context.Background(), svc, domain.OpIntegrationsCreate, strPtr("outline"), raw, Confirmation{DryRun: true})
	if err != nil {
		t.Fatalf("dry run of integrations.create: %v", err)
	}
	if plan.Str("target") != "matomo.example.com" {
		t.Fatalf("the plan names target %q, want the lower-cased host", plan.Str("target"))
	}
	if containsValue(plan, "SYNTHETICPATHTOKEN") {
		t.Fatalf("the dry run shows the path of the address: %v", plan.Fields())
	}
	if len(reader.ops) != 0 {
		t.Fatalf("the dry run read %v, want no read: the integration does not exist yet", reader.ops)
	}
	if _, err := Administer(context.Background(), svc, domain.OpIntegrationsCreate, strPtr("outline"), raw, Confirmation{Yes: true}); !errors.Is(err, domain.ErrConfirmRequired) {
		t.Fatalf("a missing --confirm: %v", err)
	}
	if _, err := Administer(context.Background(), svc, domain.OpIntegrationsCreate, strPtr("outline"), raw,
		Confirmation{Yes: true, Confirm: strPtr("Matomo.Example.com")}); !errors.Is(err, domain.ErrConfirmMismatch) {
		t.Fatalf("a host in another case: %v", err)
	}
	if _, err := Administer(context.Background(), svc, domain.OpIntegrationsCreate, strPtr("outline"), raw,
		Confirmation{Yes: true, Confirm: strPtr("other.example.com")}); !errors.Is(err, domain.ErrConfirmMismatch) {
		t.Fatalf("another host: %v", err)
	}
	if len(admin.ops) != 0 {
		t.Fatalf("a refused critical call reached the admin port: %v", admin.ops)
	}
	if _, err := Administer(context.Background(), svc, domain.OpIntegrationsCreate, strPtr("outline"), raw,
		Confirmation{Yes: true, Confirm: strPtr("matomo.example.com")}); err != nil {
		t.Fatalf("the right --confirm: %v", err)
	}
	if len(admin.ops) != 1 || admin.ops[0] != domain.OpIntegrationsCreate {
		t.Fatalf("the admin port was called %v, want integrations.create once", admin.ops)
	}
	if got := admin.args[0].String("url"); got != "https://matomo.example.com/SYNTHETICPATHTOKEN" {
		t.Fatalf("the url sent = %q, want the address the caller gave", got)
	}
}

// TestAdministerCriticalIntegrationsUpdateConfirmsTheNewHost checks the canonical target of integrations.update (ADR 007
// section 3.5, the stricter reading): the host of the new URL setting, which is the destination the call opens. The target
// read confirms that the integration exists; the host the wiki held before is not the value to type, so typing it is a
// mismatch.
func TestAdministerCriticalIntegrationsUpdateConfirmsTheNewHost(t *testing.T) {
	reader := &fakeReader{reply: domain.NewRecord(), byOp: map[domain.OperationID]domain.Record{
		domain.OpIntegrationsGet: integrationReply(),
	}}
	admin := &fakeAdmin{reply: integrationReply()}
	svc, _ := adminService(t, domain.AccessAdmin, reader, admin)
	raw := map[string]any{"integrationId": integrationID, "url": "https://new.example.test/SYNTHETICPATHTOKEN"}

	plan, err := Administer(context.Background(), svc, domain.OpIntegrationsUpdate, strPtr("outline"), raw, Confirmation{DryRun: true})
	if err != nil {
		t.Fatalf("dry run of integrations.update: %v", err)
	}
	if plan.Str("target") != "new.example.test" {
		t.Fatalf("the plan names target %q, want the new host", plan.Str("target"))
	}
	if containsValue(plan, "SYNTHETICPATHTOKEN") {
		t.Fatalf("the dry run shows the path of the address: %v", plan.Fields())
	}
	if len(reader.ops) != 1 || reader.ops[0] != domain.OpIntegrationsGet {
		t.Fatalf("the dry run read %v, want integrations.get once", reader.ops)
	}
	if _, err := Administer(context.Background(), svc, domain.OpIntegrationsUpdate, strPtr("outline"), raw,
		Confirmation{Yes: true, Confirm: strPtr("old.example.test")}); !errors.Is(err, domain.ErrConfirmMismatch) {
		t.Fatalf("the host the wiki held before: %v", err)
	}
	if len(admin.ops) != 0 {
		t.Fatalf("a refused critical call reached the admin port: %v", admin.ops)
	}
	if _, err := Administer(context.Background(), svc, domain.OpIntegrationsUpdate, strPtr("outline"), raw,
		Confirmation{Yes: true, Confirm: strPtr("new.example.test")}); err != nil {
		t.Fatalf("the right --confirm: %v", err)
	}
	if len(admin.ops) != 1 || admin.ops[0] != domain.OpIntegrationsUpdate {
		t.Fatalf("the admin port was called %v, want integrations.update once", admin.ops)
	}
}

// TestAdministerIntegrationsUpdateOfAMissingIntegrationStopsBeforeAnyChange checks that a target that does not resolve stops
// the critical call before the admin port, even with the right --confirm.
func TestAdministerIntegrationsUpdateOfAMissingIntegrationStopsBeforeAnyChange(t *testing.T) {
	reader := &fakeReader{err: domain.ErrNotFound}
	admin := &fakeAdmin{reply: integrationReply()}
	svc, _ := adminService(t, domain.AccessAdmin, reader, admin)
	raw := map[string]any{"integrationId": integrationID, "url": "https://new.example.test/x"}

	if _, err := Administer(context.Background(), svc, domain.OpIntegrationsUpdate, strPtr("outline"), raw,
		Confirmation{Yes: true, Confirm: strPtr("new.example.test")}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("a missing integration: %v", err)
	}
	if len(admin.ops) != 0 {
		t.Fatalf("a missing integration reached the admin port: %v", admin.ops)
	}
}

// TestAdministerCriticalAuthProvidersDeleteConfirmsTheProviderID checks the canonical target of authenticationProviders.delete:
// the sign-in provider ID that authentication-providers.info reads, and not its name.
func TestAdministerCriticalAuthProvidersDeleteConfirmsTheProviderID(t *testing.T) {
	const providerID = "b2000000-0000-4000-8000-000000000016"
	reader := &fakeReader{reply: domain.NewRecord(), byOp: map[domain.OperationID]domain.Record{
		domain.OpAuthenticationProvidersGet: providerRecord(providerID, "Google"),
	}}
	admin := &fakeAdmin{reply: ackReply()}
	svc, _ := adminService(t, domain.AccessAdmin, reader, admin)
	raw := map[string]any{"authenticationProviderId": providerID}

	if _, err := Administer(context.Background(), svc, domain.OpAuthenticationProvidersDelete, strPtr("outline"), raw,
		Confirmation{Yes: true, Confirm: strPtr("Google")}); !errors.Is(err, domain.ErrConfirmMismatch) {
		t.Fatalf("the name, not the ID: %v", err)
	}
	if len(admin.ops) != 0 {
		t.Fatalf("a refused critical call reached the admin port: %v", admin.ops)
	}
	if _, err := Administer(context.Background(), svc, domain.OpAuthenticationProvidersDelete, strPtr("outline"), raw,
		Confirmation{Yes: true, Confirm: strPtr(providerID)}); err != nil {
		t.Fatalf("the right --confirm: %v", err)
	}
	if len(admin.ops) != 1 || admin.ops[0] != domain.OpAuthenticationProvidersDelete {
		t.Fatalf("the admin port was called %v, want authenticationProviders.delete once", admin.ops)
	}
}

// TestAdministerDryRunOfAttachmentsDeleteShowsThePlanAndReadsNothing checks the row with no target read: its dry run has no
// target field and sends no read, and it needs no --confirm, because it is not critical.
func TestAdministerDryRunOfAttachmentsDeleteShowsThePlanAndReadsNothing(t *testing.T) {
	reader := &fakeReader{reply: domain.NewRecord()}
	admin := &fakeAdmin{reply: ackReply()}
	svc, _ := adminService(t, domain.AccessAdmin, reader, admin)

	plan, err := Administer(context.Background(), svc, domain.OpAttachmentsDelete, strPtr("outline"),
		map[string]any{"attachmentId": "a2000000-0000-4000-8000-000000000002"}, Confirmation{DryRun: true})
	if err != nil {
		t.Fatalf("dry run of attachments.delete: %v", err)
	}
	if _, ok := plan.Get("target"); ok {
		t.Fatalf("the dry run of a row with no target names one: %v", plan.Fields())
	}
	if len(reader.ops) != 0 || len(admin.ops) != 0 {
		t.Fatalf("the dry run read %v and changed %v, want nothing", reader.ops, admin.ops)
	}
}

// TestAdministerDryRunOfAnIntegrationNeverShowsItsAddress checks that a dry run of an integration change shows the host of
// the address and never the address: a chat or webhook address carries a secret in its path.
func TestAdministerDryRunOfAnIntegrationNeverShowsItsAddress(t *testing.T) {
	reader := &fakeReader{reply: integrationReply()}
	admin := &fakeAdmin{reply: ackReply()}
	svc, _ := adminService(t, domain.AccessAdmin, reader, admin)
	raw := map[string]any{"integrationId": integrationID, "url": "https://hooks.example.test/services/SYNTHETICPATHTOKEN"}

	plan, err := Administer(context.Background(), svc, domain.OpIntegrationsUpdate, strPtr("outline"), raw, Confirmation{DryRun: true})
	if err != nil {
		t.Fatalf("dry run of integrations.update: %v", err)
	}
	if containsValue(plan, "SYNTHETICPATHTOKEN") || containsValue(plan, "https://hooks.example.test/services/SYNTHETICPATHTOKEN") {
		t.Fatalf("the dry run shows the address: %v", plan.Fields())
	}
	if !containsValue(plan, "hooks.example.test") {
		t.Fatalf("the dry run does not show the host: %v", plan.Fields())
	}
}
