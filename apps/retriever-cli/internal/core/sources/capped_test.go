package sources

import (
	"context"
	"errors"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/ports"
)

// adminOutlineService returns a service over one outline source at admin access, whose reader is reader. It is the
// source an admin-level read needs, so a cap below admin is the only thing that can refuse it.
func adminOutlineService(t *testing.T, reader *fakeReader) (Service, *fakeResolver) {
	t.Helper()
	id, err := domain.NewSourceID("outline")
	if err != nil {
		t.Fatalf("NewSourceID: %v", err)
	}
	ep, err := domain.NewEndpoint("https://wiki.example.com")
	if err != nil {
		t.Fatalf("NewEndpoint: %v", err)
	}
	ref, err := domain.NewCredentialRef("OUTLINE_API_KEY")
	if err != nil {
		t.Fatalf("NewCredentialRef: %v", err)
	}
	tags, _ := domain.NewTagSet()
	cat, err := domain.NewCatalog([]domain.Source{
		domain.NewSource(id, domain.KindOutline, tags, domain.AccessAdmin, ep, ref),
	})
	if err != nil {
		t.Fatalf("NewCatalog: %v", err)
	}
	res := &fakeResolver{gateways: map[string]ports.SourceGateway{"outline": &fakeGateway{}}, reader: reader}
	return Service{Catalog: cat, Gateways: res}, res
}

// TestCappedServiceRefusesAnAdminReadBeforeTheSourceIsAsked checks the cap (ADR 007 section 3.3): an admin source capped
// to read refuses an admin-level read with the public text, and the source is never asked. The same read on the
// uncapped service is not refused for its level.
func TestCappedServiceRefusesAnAdminReadBeforeTheSourceIsAsked(t *testing.T) {
	reader := &fakeReader{reply: domain.NewList(nil)}
	svc, res := adminOutlineService(t, reader)
	capped := svc.Capped(domain.AccessRead)

	_, err := Read(context.Background(), capped, domain.OpAPIKeysList, nil, map[string]any{})
	const want = `source "outline" allows read only; this operation needs admin`
	if err == nil || err.Error() != want {
		t.Fatalf("error = %v, want %q", err, want)
	}
	if !errors.Is(err, domain.ErrAccessLevelTooLow) {
		t.Fatalf("error does not match ErrAccessLevelTooLow: %v", err)
	}
	if len(res.asked) != 0 || len(reader.ops) != 0 {
		t.Fatalf("a refused read reached the source: asked %v, reader ran %v", res.asked, reader.ops)
	}

	if _, err := Read(context.Background(), svc, domain.OpAPIKeysList, nil, map[string]any{}); errors.Is(err, domain.ErrAccessLevelTooLow) {
		t.Fatalf("the uncapped service refused an admin read for its level: %v", err)
	}
}

// TestCappedServiceLeavesTheReceiverAndItsResolverAlone checks that Capped returns a new service: the original keeps
// its admin catalog and the same resolver.
func TestCappedServiceLeavesTheReceiverAndItsResolverAlone(t *testing.T) {
	svc, res := adminOutlineService(t, &fakeReader{})
	capped := svc.Capped(domain.AccessRead)
	if got := svc.Catalog.MaxAccess(); got != domain.AccessAdmin {
		t.Fatalf("the original catalog max = %v, want admin", got)
	}
	if got := capped.Catalog.MaxAccess(); got != domain.AccessRead {
		t.Fatalf("the capped catalog max = %v, want read", got)
	}
	if capped.Gateways != svc.Gateways || capped.Gateways != ports.GatewayResolver(res) {
		t.Fatalf("Capped changed the gateway resolver")
	}
}
