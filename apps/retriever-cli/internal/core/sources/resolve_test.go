package sources

import (
	"errors"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// TestResolveNamesTheSourceWithoutAGatewayCall checks that Resolve applies the default rule and reads no gateway.
func TestResolveNamesTheSourceWithoutAGatewayCall(t *testing.T) {
	gw := &fakeGateway{}
	svc, resolver := outlineService(t, gw)
	src, err := Resolve(svc, domain.KindOutline, nil)
	if err != nil || src.ID().String() != "outline" {
		t.Fatalf("Resolve(nil) = %v, %v", src.ID(), err)
	}
	if len(resolver.asked) != 0 {
		t.Fatalf("Resolve asked the gateways for %v, want none", resolver.asked)
	}
}

// TestResolveRefusesABadOrUnknownSource checks the same rules the reads apply, with the same sentinel errors.
func TestResolveRefusesABadOrUnknownSource(t *testing.T) {
	svc, _ := outlineService(t, &fakeGateway{})
	if _, err := Resolve(svc, domain.KindOutline, strPtr("Outline")); !errors.Is(err, domain.ErrInvalidSourceID) {
		t.Fatalf("invalid ID error = %v", err)
	}
	if _, err := Resolve(svc, domain.KindOutline, strPtr("rnd")); !errors.Is(err, domain.ErrUnknownSource) {
		t.Fatalf("unknown ID error = %v", err)
	}
}
