//go:build integration

package outline

import (
	"context"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"net/http"
	"testing"
)

// TestNoWriteReachesANonLoopbackHost is the proof that no test write leaves the in-process stub for a real host. Every write row is sent through the client; the stub refuses any host
// that is not a loopback address or a .invalid name, and counts it. The count must stay at zero.
func TestNoWriteReachesANonLoopbackHost(t *testing.T) {
	for _, tc := range requestCasesB {
		t.Run(tc.name, func(t *testing.T) {
			r := writeRig(t, sampleOptions())
			method := endpointMethodOf(t, tc.op)
			// The router-faithful synthetic reply of the method, so the write projects and succeeds.
			r.stub.SetReply(method, http.StatusOK, string(loadSynthetic(t, method)))
			// The write must succeed and reach the stub, or the count of refused hosts would prove nothing (finding m6).
			if _, err := r.gw.Write(context.Background(), tc.op, domain.NewArgs(tc.args)); err != nil {
				t.Fatalf("the write did not succeed, so the proof would pass vacuously: %v", err)
			}
			if n := r.stub.Calls(method); n != 1 {
				t.Fatalf("%s calls = %d, want 1", method, n)
			}
			if n := r.stub.Refused(); n != 0 {
				t.Fatalf("a write reached a host that is not loopback or .invalid: %d refused", n)
			}
		})
	}
}
