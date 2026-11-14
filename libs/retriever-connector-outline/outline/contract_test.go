package outline

import (
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
	"github.com/wahidyankf/super-agent/libs/retriever-testkit/sourcecontract"
)

// TestGatewayPassesTheSourceContractInProcess runs the shared contract suite against
// the Outline gateway, driven by the in-process stub. No socket is opened.
func TestGatewayPassesTheSourceContractInProcess(t *testing.T) {
	sourcecontract.Run(t, func(t *testing.T, w sourcecontract.Wiring) ports.SourceGateway {
		return contractGateway(t, w)
	})
}
