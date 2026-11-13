package sourcecontract_test

import (
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/ports"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/tests/support/sourcecontract"
)

// TestFakeGatewayPassesTheSourceContract runs the shared contract suite against the
// in-memory fake, the unit-level implementation of ports.SourceGateway.
func TestFakeGatewayPassesTheSourceContract(t *testing.T) {
	sourcecontract.Run(t, func(_ *testing.T, w sourcecontract.Wiring) ports.SourceGateway {
		return sourcecontract.NewFake(w)
	})
}
