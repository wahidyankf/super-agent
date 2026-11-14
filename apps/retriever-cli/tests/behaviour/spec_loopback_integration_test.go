//go:build integration

package behaviour

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/tests/specstub"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// The integration level answers a spec source from the loopback stub of tests/specstub, over real HTTPS on the loopback
// address. The stub's certificate is valid for the stub host, the resolver answers that host only, and the dialer reaches
// only the address the engine checked. The engine's own address checks run unchanged, and the private-network option lets
// the loopback address through, as the composition root's test wiring does.
func init() {
	loopbackSpecRuntime = func(t *testing.T, model *spec.Spec) (specRuntime, error) {
		srv := specstub.Start(t, model)
		var dialer net.Dialer
		nw := engine.Network{
			Lookup: func(_ context.Context, host string) ([]netip.Addr, error) {
				if host != specstub.Host {
					return nil, errors.New("the test resolver knows only the stub host")
				}
				return []netip.Addr{netip.MustParseAddr("127.0.0.1")}, nil
			},
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				if address != "127.0.0.1:443" {
					return nil, errors.New("the test dialer reaches only the checked loopback address")
				}
				return dialer.DialContext(ctx, network, srv.Addr())
			},
		}
		return specRuntime{
			network: nw,
			roots:   srv.Roots,
			options: []engine.Option{engine.AllowPrivateNetwork()},
			hits:    srv.Hits,
		}, nil
	}
}
