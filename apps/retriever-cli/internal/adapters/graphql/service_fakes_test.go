package graphql

import (
	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
	usecases "github.com/wahidyankf/super-agent/libs/retriever-runtime/sources"
)

// fixedResolver serves one gateway for every source, as the registry would for a single outline source.
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

// mustSource builds one outline source of the test catalogs. The constant inputs are valid, so a constructor
// failure is a programming error and panics.
func mustSource(id, host, credential string, tags []string) domain.Source {
	sid, err := domain.NewSourceID(id)
	if err != nil {
		panic(err)
	}
	ep, err := domain.NewEndpoint("https://" + host)
	if err != nil {
		panic(err)
	}
	ref, err := domain.NewCredentialRef(credential)
	if err != nil {
		panic(err)
	}
	set, err := domain.NewTagSet(tags...)
	if err != nil {
		panic(err)
	}
	return domain.NewSource(sid, domain.KindOutline, set, ep, ref)
}

// catalogServices returns the source service of a catalog with the given sources, served by gw.
func catalogServices(gw ports.SourceGateway, sources ...domain.Source) usecases.Service {
	cat, err := domain.NewCatalog(sources)
	if err != nil {
		panic(err)
	}
	return usecases.Service{Catalog: cat, Gateways: fixedResolver{gw: gw}}
}

// oneOutlineSources returns the source service of a catalog with the single outline source, served by gw.
func oneOutlineSources(gw ports.SourceGateway) usecases.Service {
	return catalogServices(gw, mustSource("outline", "docs.example.test", "EXAMPLE_OUTLINE_API_KEY",
		nil))
}

// twoOutlineSources returns the source service of the sources feature: a read source with tags company and wiki, and
// a write source with tags engineering and wiki. Both are served by gw.
func twoOutlineSources(gw ports.SourceGateway) usecases.Service {
	return catalogServices(gw,
		mustSource("outline", "docs.example.test", "EXAMPLE_OUTLINE_API_KEY",
			[]string{"company", "wiki"}),
		mustSource("outline-engineering", "wiki.engineering.example.com", "ENGINEERING_OUTLINE_API_KEY",
			[]string{"engineering", "wiki"}),
	)
}
