// Package registry implements ports.GatewayResolver. It builds one gateway per source ID, on first use, and keeps it.
// The cache is required: each gateway holds the lock and the delay that give "one request at a time", so a new
// gateway per call would lose that rule.
package registry

import (
	"sync"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/ports"
)

// Builder makes the gateway for one source. The composition root supplies one per kind.
type Builder func(source domain.Source) (ports.SourceGateway, error)

// Resolver implements ports.GatewayResolver. Build it with New.
type Resolver struct {
	mu       sync.Mutex
	builders map[domain.SourceKind]Builder
	cache    map[domain.SourceID]ports.SourceGateway
}

var _ ports.GatewayResolver = (*Resolver)(nil)

// New returns a resolver that uses builders, one per kind. A kind without a builder is unsupported.
func New(builders map[domain.SourceKind]Builder) *Resolver {
	return &Resolver{builders: builders, cache: map[domain.SourceID]ports.SourceGateway{}}
}

// For returns the cached gateway of the source, and builds it on the first call. A failed build is not cached, so
// a missing key for one source does not stop the others.
func (r *Resolver) For(source domain.Source) (ports.SourceGateway, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if gw, ok := r.cache[source.ID()]; ok {
		return gw, nil
	}
	build, ok := r.builders[source.Kind()]
	if !ok {
		return nil, domain.ErrUnsupportedSource
	}
	gw, err := build(source)
	if err != nil {
		return nil, err
	}
	r.cache[source.ID()] = gw
	return gw, nil
}

// Reader returns the source's gateway as a SourceReader. A gateway that cannot read operations is unsupported. The
// gateway is the same one For returns, so the request lock and the pacing rule apply to every call.
func (r *Resolver) Reader(source domain.Source) (ports.SourceReader, error) {
	gw, err := r.For(source)
	if err != nil {
		return nil, err
	}
	reader, ok := gw.(ports.SourceReader)
	if !ok {
		return nil, domain.ErrUnsupportedSource
	}
	return reader, nil
}

// Writer returns the source's gateway as a SourceWriter. A gateway that cannot write operations is unsupported. As with
// Reader, it is the gateway For returns, so the request lock and the pacing rule apply to every write too.
func (r *Resolver) Writer(source domain.Source) (ports.SourceWriter, error) {
	gw, err := r.For(source)
	if err != nil {
		return nil, err
	}
	writer, ok := gw.(ports.SourceWriter)
	if !ok {
		return nil, domain.ErrUnsupportedSource
	}
	return writer, nil
}
