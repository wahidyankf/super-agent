package mcp

import (
	"bytes"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/sources/operations"
)

// stageBIdempotent is the expected idempotent hint of every write tool of stage b, written out by hand (finding A3). It is
// the ceiling rule applied to each row: an additive row is not idempotent, an update is idempotent, and
// documents.update is not, because its append and patch modes repeat a different change. The test reads no value from the
// catalog's own annotation code, so a wrong predicate cannot agree with itself. The table holds the 40 write tools of stage
// b (inventory: stage b 40, class W). None is critical, so the write ceiling offers exactly these tools.
var stageBIdempotent = map[string]bool{
	"create_access_request":          false,
	"create_collection":              false,
	"duplicate_collection":           false,
	"export_collection":              false,
	"move_collection":                true,
	"restore_collection":             true,
	"update_collection":              true,
	"add_comment_reaction":           false,
	"create_comment":                 false,
	"remove_comment_reaction":        true,
	"resolve_comment":                true,
	"unresolve_comment":              true,
	"update_comment":                 true,
	"create_document":                false,
	"duplicate_document":             false,
	"move_document":                  true,
	"restore_document":               true,
	"templatize_document":            false,
	"unpublish_document":             true,
	"update_document":                false,
	"update_all_notifications":       true,
	"update_notification":            true,
	"create_pin":                     false,
	"delete_pin":                     true,
	"update_pin":                     true,
	"update_revision":                true,
	"delete_search":                  true,
	"update_search":                  true,
	"create_star":                    false,
	"delete_star":                    true,
	"update_star":                    true,
	"create_subscription":            false,
	"delete_subscription":            true,
	"create_template":                false,
	"duplicate_template":             false,
	"restore_template":               true,
	"update_template":                true,
	"update_user_membership":         true,
	"subscribe_user_notifications":   false,
	"unsubscribe_user_notifications": false,
}

// TestStageBWriteToolsCarryTheHintsOfTheirCatalogRows pins the MCP set of stage b. At the write
// ceiling, with a write source, the tools are exactly the write rows of the catalog, and each tool carries the hints the
// hand-written table gives: not read-only, never destructive, open to the world, and idempotent as the table says.
func TestStageBWriteToolsCarryTheHintsOfTheirCatalogRows(t *testing.T) {
	svc := serviceWith(t, &fakeGateway{}, mustSource(t, "outline", "write")).Capped(domain.AccessWrite)
	var logs bytes.Buffer
	res, err := connect(t, svc, &logs).ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}

	byName := map[string]*sdk.Tool{}
	for _, tl := range res.Tools {
		byName[tl.Name] = tl
		if a := tl.Annotations; a != nil && a.DestructiveHint != nil && *a.DestructiveHint {
			t.Errorf("%s carries a destructive hint, and stage b has no destructive tool", tl.Name)
		}
	}

	writeTools := map[string]bool{}
	for _, row := range operations.RowsOf(domain.KindOutline) {
		if row.Access == domain.AccessWrite {
			writeTools[row.Tool] = true
		}
	}
	if len(writeTools) != len(stageBIdempotent) {
		t.Fatalf("the catalog holds %d write tools, want the %d of stage b", len(writeTools), len(stageBIdempotent))
	}
	for name, wantIdempotent := range stageBIdempotent {
		if !writeTools[name] {
			t.Errorf("%s is in the table but the catalog has no write row with that tool", name)
			continue
		}
		tl, ok := byName[name]
		if !ok {
			t.Errorf("%s: the write ceiling offers no such tool", name)
			continue
		}
		a := tl.Annotations
		if a == nil || a.ReadOnlyHint || a.DestructiveHint == nil || *a.DestructiveHint || a.OpenWorldHint == nil ||
			!*a.OpenWorldHint || a.IdempotentHint != wantIdempotent {
			t.Errorf("%s: hints = %+v, want read-only false, destructive false, idempotent %v, open world true",
				name, a, wantIdempotent)
		}
	}
}
