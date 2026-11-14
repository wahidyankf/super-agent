//go:build integration

package outline

import (
	"io/fs"
	"slices"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/operations"
	"github.com/wahidyankf/super-agent/libs/retriever-testkit/outlinestub"
	"github.com/wahidyankf/super-agent/libs/retriever-testkit/synthetic"
)

// loadSynthetic returns the synthetic reply of one wire method. The replies are embedded in the synthetic package, so a
// missing reply fails the test, and every read operation needs one.
func loadSynthetic(t *testing.T, method string) []byte {
	t.Helper()
	body, err := fs.ReadFile(synthetic.Replies(), method+".json")
	if err != nil {
		t.Fatalf("no synthetic reply for %s: %v", method, err)
	}
	return body
}

// TestEveryReadProjectionHoldsExactlyItsFields is the projection tripwire. For every read
// operation of the catalog it projects the router-faithful synthetic reply, checks that the reply holds only keys the
// router sends, and checks that the projected fields are exactly the row's Fields: no extra field and no missing one.
// A row and a projection that drift apart fail here, before any scenario runs.
func TestEveryReadProjectionHoldsExactlyItsFields(t *testing.T) {
	for _, row := range operations.RowsOf(domain.KindOutline) {
		t.Run(string(row.ID), func(t *testing.T) {
			ep, ok := endpointOf(row.ID)
			if !ok {
				t.Fatalf("no endpoint for %s", row.ID)
			}
			body := loadSynthetic(t, ep.method)
			if err := outlinestub.CheckReplyKeys(ep.method, body); err != nil {
				t.Fatalf("the fixture is not router-faithful: %v", err)
			}
			objects := objectsOf(projectionOf(t, row.ID, body))
			if len(objects) == 0 {
				t.Fatalf("the fixture holds no object to project")
			}
			for i, obj := range objects {
				if got := fieldNames(obj); !slices.Equal(got, row.Fields) {
					t.Errorf("object %d projects %v, want exactly the row fields %v", i, got, row.Fields)
				}
			}
		})
	}
}

// TestRedactedFieldsNeverReachAProjection feeds each forbidden field, with its sentinel, into every object of the reply of
// every operation that can carry it, and checks that the sentinel is not in the projection. A projection that keeps the
// field fails here. The same reply is also run through the CLI and the MCP scenarios, which check the output streams.
func TestRedactedFieldsNeverReachAProjection(t *testing.T) {
	for _, r := range redactions {
		for _, row := range operations.RowsOf(domain.KindOutline) {
			if r.ops != nil && !slices.Contains(r.ops, row.ID) {
				continue
			}
			t.Run(r.name+"/"+string(row.ID), func(t *testing.T) {
				ep, _ := endpointOf(row.ID)
				hostile, err := outlinestub.HostileReply(loadSynthetic(t, ep.method), r.key, r.value)
				if err != nil {
					t.Fatalf("hostile reply: %v", err)
				}
				rec := projectionOf(t, row.ID, hostile)
				if text := flatText(rec); strings.Contains(text, r.forbidden) {
					t.Fatalf("the projection carries the %s", r.name)
				}
			})
		}
	}
}
