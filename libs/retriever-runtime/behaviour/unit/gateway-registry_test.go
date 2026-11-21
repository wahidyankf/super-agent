package unit_test

import (
	"context"
	"errors"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/registry"
)

// errBuild is the failure of a build that cannot make a gateway.
var errBuild = errors.New("the build failed")

// @scenario specs/libs/retriever-runtime/behaviours/gateway-registry.feature: Two requests for the same source get the same gateway
func TestTwoRequestsForTheSameSourceGetTheSameGateway(t *testing.T) {
	builds := 0
	gw := &wiki{id: "docs"}
	r := registry.New(map[domain.SourceKind]registry.Builder{
		domain.KindOutline: func(domain.Source) (ports.SourceGateway, error) {
			builds++
			return gw, nil
		},
	})
	src := outlineSource(t, "docs")
	first, err := r.For(src)
	if err != nil {
		t.Fatalf("For error = %v", err)
	}
	second, err := r.For(src)
	if err != nil {
		t.Fatalf("For error = %v", err)
	}
	if builds != 1 {
		t.Fatalf("builder ran %d times, want once", builds)
	}
	if first != second {
		t.Fatalf("the two answers are different gateways")
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/gateway-registry.feature: Two sources get two gateways
func TestTwoSourcesGetTwoGateways(t *testing.T) {
	var asked []string
	r := registry.New(map[domain.SourceKind]registry.Builder{
		domain.KindOutline: func(src domain.Source) (ports.SourceGateway, error) {
			return &wiki{id: src.ID().String(), asked: &asked}, nil
		},
	})
	docs, err := r.For(outlineSource(t, "docs"))
	if err != nil {
		t.Fatalf("For(docs) error = %v", err)
	}
	wikiGW, err := r.For(outlineSource(t, "wiki"))
	if err != nil {
		t.Fatalf("For(wiki) error = %v", err)
	}
	if docs == wikiGW {
		t.Fatalf("docs and wiki share one gateway, want each its own")
	}
	if _, err := docs.ListCollections(context.Background()); err != nil {
		t.Fatalf("ListCollections error = %v", err)
	}
	if len(asked) != 1 || asked[0] != "docs" {
		t.Fatalf("the docs gateway answered %v, want only docs", asked)
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/gateway-registry.feature: A source of a kind with no builder is refused as unsupported
func TestASourceOfAKindWithNoBuilderIsRefusedAsUnsupported(t *testing.T) {
	r := registry.New(map[domain.SourceKind]registry.Builder{
		domain.KindOutline: builderFor(&wiki{id: "docs"}),
	})
	gw, err := r.For(kiwiSource(t, "tests"))
	wantError(t, err, "unsupported source")
	if gw != nil {
		t.Fatalf("gateway = %v, want none", gw)
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/gateway-registry.feature: A failed build is not kept and does not stop the other sources
func TestAFailedBuildIsNotKeptAndDoesNotStopTheOtherSources(t *testing.T) {
	wikiGW := &wiki{id: "wiki"}
	r := registry.New(map[domain.SourceKind]registry.Builder{
		domain.KindOutline: func(src domain.Source) (ports.SourceGateway, error) {
			if src.ID().String() == "docs" {
				return nil, errBuild
			}
			return wikiGW, nil
		},
	})
	if _, err := r.For(outlineSource(t, "docs")); !errors.Is(err, errBuild) {
		t.Fatalf("For(docs) error = %v, want the build error", err)
	}
	gw, err := r.For(outlineSource(t, "wiki"))
	if err != nil || gw == nil {
		t.Fatalf("For(wiki) = %v, %v; want a gateway", gw, err)
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/gateway-registry.feature: A gateway that cannot read operations is refused as a reader
func TestAGatewayThatCannotReadOperationsIsRefusedAsAReader(t *testing.T) {
	r := registry.New(map[domain.SourceKind]registry.Builder{
		domain.KindOutline: builderFor(plainGateway{w: &wiki{id: "docs"}}),
	})
	reader, err := r.Reader(outlineSource(t, "docs"))
	wantError(t, err, "unsupported source")
	if reader != nil {
		t.Fatalf("reader = %v, want none", reader)
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/gateway-registry.feature: The writer of a source is the gateway that writes
func TestTheWriterOfASourceIsTheGatewayThatWrites(t *testing.T) {
	gw := &wiki{id: "docs"}
	r := registry.New(map[domain.SourceKind]registry.Builder{
		domain.KindOutline: builderFor(gw),
	})
	src := outlineSource(t, "docs")
	kept, err := r.For(src)
	if err != nil {
		t.Fatalf("For error = %v", err)
	}
	writer, err := r.Writer(src)
	if err != nil {
		t.Fatalf("Writer error = %v", err)
	}
	if any(writer) != any(kept) {
		t.Fatalf("the writer is not the gateway the registry keeps for docs")
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/gateway-registry.feature: A source whose kind has no builder has no writer
func TestASourceWhoseKindHasNoBuilderHasNoWriter(t *testing.T) {
	r := registry.New(map[domain.SourceKind]registry.Builder{
		domain.KindOutline: builderFor(&wiki{id: "docs"}),
	})
	writer, err := r.Writer(kiwiSource(t, "tests"))
	wantError(t, err, "unsupported source")
	if writer != nil {
		t.Fatalf("writer = %v, want none", writer)
	}
}
