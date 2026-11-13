//go:build integration

package registry

import (
	"context"
	"errors"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/ports"
)

// administeringGateway is a gateway that also runs admin operations.
type administeringGateway struct{ stubGateway }

func (administeringGateway) Administer(context.Context, domain.OperationID, domain.Args) (domain.Record, error) {
	return domain.Record{}, nil
}

// adminResolver returns a resolver whose only builder returns gw for every outline source.
func adminResolver(gw ports.SourceGateway) *Resolver {
	return New(map[domain.SourceKind]Builder{
		domain.KindOutline: func(domain.Source) (ports.SourceGateway, error) { return gw, nil },
	})
}

// TestAdminReturnsTheGatewayThatAdministers checks the admin port: a gateway with admin operations is returned, a gateway
// without them is refused, and a source that cannot be built is reported.
func TestAdminReturnsTheGatewayThatAdministers(t *testing.T) {
	src := sourceAt(t, "outline", "docs.example.com", "A_API_KEY", domain.AccessAdmin)
	admin, err := adminResolver(administeringGateway{}).Admin(src)
	if err != nil || admin == nil {
		t.Fatalf("Admin = %v, %v; want the gateway", admin, err)
	}
	if _, err := adminResolver(&stubGateway{}).Admin(src); !errors.Is(err, domain.ErrUnsupportedSource) {
		t.Fatalf("Admin on a gateway without admin error = %v, want ErrUnsupportedSource", err)
	}
	if _, err := New(map[domain.SourceKind]Builder{}).Admin(src); err == nil {
		t.Fatal("Admin returned no error for a kind with no builder")
	}
}

// TestReaderReportsABuildFailure checks that a source which cannot be built is reported by Reader, not hidden as a missing
// capability.
func TestReaderReportsABuildFailure(t *testing.T) {
	src := sourceAt(t, "outline", "docs.example.com", "A_API_KEY", domain.AccessRead)
	if _, err := New(map[domain.SourceKind]Builder{}).Reader(src); !errors.Is(err, domain.ErrUnsupportedSource) {
		t.Fatalf("Reader error = %v, want ErrUnsupportedSource", err)
	}
}
