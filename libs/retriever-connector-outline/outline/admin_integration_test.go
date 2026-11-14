//go:build integration

package outline

import (
	"context"
	"encoding/json"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"net/http"
	"reflect"
	"testing"
)

func TestEveryAdminSendsItsExactRequest(t *testing.T) {
	for _, tc := range requestCasesC {
		t.Run(tc.name, func(t *testing.T) {
			gw, rec := recordingAdminCall(t, tc.op)
			_, _ = gw.Write(context.Background(), tc.op, domain.NewArgs(tc.args))
			calls := rec.recorded()
			// A change that reads the object first (adminPreReads) sends that read before the change, so the change is the last call.
			if want := 1 + len(adminPreReadsOf(tc.op)); len(calls) != want {
				t.Fatalf("requests = %d, want %d", len(calls), want)
			}
			change := calls[len(calls)-1]
			if change.Path != tc.path {
				t.Fatalf("path = %q, want %q", change.Path, tc.path)
			}
			var want any
			if err := json.Unmarshal([]byte(tc.body), &want); err != nil {
				t.Fatalf("bad expected body: %v", err)
			}
			if !reflect.DeepEqual(change.Body, want) {
				t.Fatalf("body = %v, want %v", change.Body, want)
			}
		})
	}
}

// TestNoAdminReachesANonLoopbackHost is the proof that no test admin call leaves the in-process stub for a real host. Every admin row of this slice is sent through the client;
// the stub refuses any host that is not a loopback address or a .invalid name, and counts it. Each call must
// succeed and reach the stub once, and the count of refused hosts must stay at zero.
func TestNoAdminReachesANonLoopbackHost(t *testing.T) {
	for _, tc := range requestCasesC {
		t.Run(tc.name, func(t *testing.T) {
			r := adminRig(t)
			method := adminEndpoints[tc.op].method
			// The router-faithful synthetic reply of the method, so the call projects and succeeds.
			r.stub.SetReply(method, http.StatusOK, string(loadSynthetic(t, method)))
			setPreReadReplies(t, r.stub, tc.op)
			// The call must succeed and reach the stub, or the count of refused hosts would prove nothing (finding m6).
			if _, err := r.gw.Write(context.Background(), tc.op, domain.NewArgs(tc.args)); err != nil {
				t.Fatalf("the admin call did not succeed, so the proof would pass vacuously: %v", err)
			}
			if n := r.stub.Calls(method); n != 1 {
				t.Fatalf("%s calls = %d, want 1", method, n)
			}
			if n := r.stub.Refused(); n != 0 {
				t.Fatalf("an admin call reached a host that is not loopback or .invalid: %d refused", n)
			}
		})
	}
}
