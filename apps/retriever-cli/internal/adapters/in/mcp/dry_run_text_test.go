package mcp

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/sources/operations"
)

// schemaDryRunText returns the description of the dry_run input of a tool as the client receives it.
func schemaDryRunText(t *testing.T, schema any) (string, bool) {
	t.Helper()
	raw, err := json.Marshal(schema)
	if err != nil {
		t.Fatalf("marshal input schema: %v", err)
	}
	var s struct {
		Properties map[string]struct {
			Description string `json:"description"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatalf("unmarshal input schema: %v", err)
	}
	p, ok := s.Properties[dryRunParam]
	return p.Description, ok
}

// TestDryRunTextMatchesWhatEachWriteToolSends checks the dry_run description of each write tool (finding m6). A write row's
// dry run sends nothing and reads nothing, so its text says that no request is sent. An admin row's dry run sends no change,
// but it may read the target and, for a change to a user, the key's own user, so its text must
// not say that no request is sent, and it must say that a read may happen.
func TestDryRunTextMatchesWhatEachWriteToolSends(t *testing.T) {
	svc := serviceWith(t, &fakeGateway{}, mustSource(t, "outline", "admin")).Capped(domain.AccessAdmin)
	var logs bytes.Buffer
	res, err := connect(t, svc, &logs).ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	rows := map[string]operations.Row{}
	for _, row := range operations.Rows() {
		if row.Tool != "" {
			rows[row.Tool] = row
		}
	}
	writes, admins := 0, 0
	for _, tl := range res.Tools {
		row, ok := rows[tl.Name]
		if !ok || row.Effect == domain.EffectRead {
			continue
		}
		desc, has := schemaDryRunText(t, tl.InputSchema)
		if !has {
			t.Errorf("%s: a write tool has no dry_run input", tl.Name)
			continue
		}
		switch row.Access {
		case domain.AccessWrite:
			writes++
			if !strings.Contains(desc, "no request is sent") {
				t.Errorf("%s: dry_run = %q, want the write text that says no request is sent", tl.Name, desc)
			}
		case domain.AccessAdmin:
			admins++
			if strings.Contains(desc, "no request is sent") {
				t.Errorf("%s: dry_run = %q says no request is sent, but an admin dry run may read the target", tl.Name, desc)
			}
			if !strings.Contains(desc, "may read") {
				t.Errorf("%s: dry_run = %q does not say that the dry run may read the target and the key's user", tl.Name, desc)
			}
			if !strings.Contains(desc, "sends no change") {
				t.Errorf("%s: dry_run = %q does not say that the dry run sends no change", tl.Name, desc)
			}
		}
	}
	if writes == 0 || admins == 0 {
		t.Fatalf("the check covered %d write tools and %d admin tools; both must be checked", writes, admins)
	}
}
