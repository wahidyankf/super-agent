package cli

import (
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/ports"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/sources"
)

// fixedResolver serves one gateway for every source. It stands in for the registry in the CLI tests.
type fixedResolver struct {
	gw ports.SourceGateway
}

// For returns the fixed gateway, or ErrUnsupportedSource when none was given.
func (r fixedResolver) For(domain.Source) (ports.SourceGateway, error) {
	if r.gw == nil {
		return nil, domain.ErrUnsupportedSource
	}
	return r.gw, nil
}

// Reader returns the fixed gateway as a reader, when the gateway can read operations.
func (r fixedResolver) Reader(domain.Source) (ports.SourceReader, error) {
	reader, ok := r.gw.(ports.SourceReader)
	if !ok {
		return nil, domain.ErrUnsupportedSource
	}
	return reader, nil
}

// Writer returns the fixed gateway as a writer, when the gateway can write operations.
func (r fixedResolver) Writer(domain.Source) (ports.SourceWriter, error) {
	writer, ok := r.gw.(ports.SourceWriter)
	if !ok {
		return nil, domain.ErrUnsupportedSource
	}
	return writer, nil
}

// oneOutlineService returns a service whose only source is the outline source, served by gw.
func oneOutlineService(gw ports.SourceGateway) sources.Service {
	id, err := domain.NewSourceID("outline")
	if err != nil {
		panic(err)
	}
	ep, err := domain.NewEndpoint("https://docs.example.test")
	if err != nil {
		panic(err)
	}
	ref, err := domain.NewCredentialRef("EXAMPLE_OUTLINE_API_KEY")
	if err != nil {
		panic(err)
	}
	tags, err := domain.NewTagSet()
	if err != nil {
		panic(err)
	}
	cat, err := domain.NewCatalog([]domain.Source{
		domain.NewSource(id, domain.KindOutline, tags, ep, ref),
	})
	if err != nil {
		panic(err)
	}
	return sources.Service{Catalog: cat, Gateways: fixedResolver{gw: gw}}
}
