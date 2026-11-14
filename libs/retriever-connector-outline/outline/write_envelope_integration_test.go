//go:build integration

package outline

import (
	"encoding/json"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/operations"
	"github.com/wahidyankf/super-agent/libs/retriever-testkit/outlinestub"
	"slices"
	"testing"
)

// TestEveryBuiltWriteFixtureHasExactlyItsVerifiedEnvelope checks, for every built write, destructive, and admin row, that the
// synthetic reply of its method holds exactly the top-level keys its verified envelope names: {success} for an
// acknowledgement, {success, data} for one that names a value, {data, policies} for an object reply, and {data} for an
// object reply the wiki sends without policies. A fixture with the shared ok and status, or without the policies of an
// object reply, fails here, before any scenario runs.
func TestEveryBuiltWriteFixtureHasExactlyItsVerifiedEnvelope(t *testing.T) {
	for _, row := range operations.RowsOf(domain.KindOutline) {
		if row.Effect == domain.EffectRead {
			continue
		}
		t.Run(string(row.ID), func(t *testing.T) {
			ep, ok := endpointOf(row.ID)
			if !ok {
				t.Fatalf("no endpoint for %s", row.ID)
			}
			envelope, ok := outlinestub.VerifiedEnvelope(ep.method)
			if !ok {
				t.Fatalf("the write %s has no verified envelope", ep.method)
			}
			var top map[string]json.RawMessage
			if err := json.Unmarshal(loadSynthetic(t, ep.method), &top); err != nil {
				t.Fatalf("the synthetic reply of %s is not a JSON object: %v", ep.method, err)
			}
			got := make([]string, 0, len(top))
			for k := range top {
				got = append(got, k)
			}
			slices.Sort(got)
			want := slices.Clone(envelope)
			slices.Sort(want)
			if !slices.Equal(got, want) {
				t.Fatalf("the synthetic reply of %s has the top-level keys %v, want exactly its verified envelope %v", ep.method, got, want)
			}
		})
	}
}
