package registry

import (
	"context"
	"errors"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
)

// stubGateway is a gateway with no behaviour. Each build makes a new one, so identity shows whether the cache worked.
type stubGateway struct{ ports.SourceGateway }

func source(t *testing.T, id, host, cred string) domain.Source {
	t.Helper()
	return sourceAt(t, id, host, cred)
}

func sourceAt(t *testing.T, id, host, cred string) domain.Source {
	t.Helper()
	sid, err := domain.NewSourceID(id)
	if err != nil {
		t.Fatalf("NewSourceID: %v", err)
	}
	ep, err := domain.NewEndpoint("https://" + host)
	if err != nil {
		t.Fatalf("NewEndpoint: %v", err)
	}
	ref, err := domain.NewCredentialRef(cred)
	if err != nil {
		t.Fatalf("NewCredentialRef: %v", err)
	}
	tags, _ := domain.NewTagSet()
	return domain.NewSource(sid, domain.KindOutline, tags, ep, ref)
}

func TestTheSameSourceGetsTheSameGateway(t *testing.T) {
	built := 0
	r := New(map[domain.SourceKind]Builder{
		domain.KindOutline: func(domain.Source) (ports.SourceGateway, error) {
			built++
			return &stubGateway{}, nil
		},
	})
	src := source(t, "outline", "docs.example.com", "A_API_KEY")
	first, err := r.For(src)
	if err != nil {
		t.Fatalf("For error = %v", err)
	}
	second, err := r.For(src)
	if err != nil || first != second {
		t.Fatalf("the same source returned a different gateway: %v, %v", first, err)
	}
	if built != 1 {
		t.Fatalf("builder ran %d times, want 1", built)
	}
}

func TestTwoSourcesGetTwoGateways(t *testing.T) {
	r := New(map[domain.SourceKind]Builder{
		domain.KindOutline: func(domain.Source) (ports.SourceGateway, error) { return &stubGateway{}, nil },
	})
	a, _ := r.For(source(t, "outline", "docs.example.com", "A_API_KEY"))
	b, _ := r.For(source(t, "outline-engineering", "wiki.example.com", "B_API_KEY"))
	if a == b {
		t.Fatalf("two sources share one gateway")
	}
}

func TestAnUnknownKindIsUnsupported(t *testing.T) {
	r := New(nil)
	if _, err := r.For(source(t, "outline", "docs.example.com", "A_API_KEY")); !errors.Is(err, domain.ErrUnsupportedSource) {
		t.Fatalf("error = %v, want ErrUnsupportedSource", err)
	}
}

func TestAFailedBuildIsNotCachedAndDoesNotStopOthers(t *testing.T) {
	calls := 0
	r := New(map[domain.SourceKind]Builder{
		domain.KindOutline: func(src domain.Source) (ports.SourceGateway, error) {
			calls++
			if src.ID().String() == "outline" && calls == 1 {
				return nil, errors.New("missing key")
			}
			return &stubGateway{}, nil
		},
	})
	if _, err := r.For(source(t, "outline", "docs.example.com", "A_API_KEY")); err == nil {
		t.Fatalf("a failed build returned no error")
	}
	if _, err := r.For(source(t, "outline-engineering", "wiki.example.com", "B_API_KEY")); err != nil {
		t.Fatalf("another source was stopped by a failed build: %v", err)
	}
	if _, err := r.For(source(t, "outline", "docs.example.com", "A_API_KEY")); err != nil {
		t.Fatalf("the failed source was not built on retry: %v", err)
	}
}

// readingGateway is a gateway that also reads catalog operations, so Reader must return it.
type readingGateway struct{ stubGateway }

func (readingGateway) Read(context.Context, domain.OperationID, domain.Args) (domain.Record, error) {
	return domain.Record{}, nil
}

// TestAGatewayThatCannotReadIsUnsupportedForReads checks that Reader refuses a gateway without the catalog reads, and
// returns the same gateway for a gateway that has them.
func TestAGatewayThatCannotReadIsUnsupportedForReads(t *testing.T) {
	plain := New(map[domain.SourceKind]Builder{
		domain.KindOutline: func(domain.Source) (ports.SourceGateway, error) { return &stubGateway{}, nil },
	})
	if _, err := plain.Reader(source(t, "outline", "docs.example.com", "A_API_KEY")); !errors.Is(err, domain.ErrUnsupportedSource) {
		t.Fatalf("Reader error = %v, want ErrUnsupportedSource", err)
	}

	reading := &readingGateway{}
	r := New(map[domain.SourceKind]Builder{
		domain.KindOutline: func(domain.Source) (ports.SourceGateway, error) { return reading, nil },
	})
	src := source(t, "outline", "docs.example.com", "A_API_KEY")
	reader, err := r.Reader(src)
	gw, _ := r.For(src)
	if err != nil || reader != ports.SourceReader(reading) || gw != ports.SourceGateway(reading) {
		t.Fatalf("Reader = %v, %v; want the gateway that For returns", reader, err)
	}
}
