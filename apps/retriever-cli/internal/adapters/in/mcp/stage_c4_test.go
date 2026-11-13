package mcp

import (
	"bytes"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// stageCHints is the expected hint pair of one MCP tool of stage c.
type stageCHints struct {
	destructive bool
	idempotent  bool
}

// stageCTools is the expected annotation of each of the 48 MCP tools of stage c, written out by hand from ADR 007 section 3.3
// (finding m3). The test reads no value from the catalog's annotation code, so a wrong predicate there cannot agree with
// itself. The rule applied: a destructive tool announces destructive and not idempotent; an update announces idempotent,
// because no stage c update repeats another change (the only non-idempotent update is in stage b); an additive tool is not
// idempotent, except shares.create, whose repeat answers the same share. No stage c tool is read-only.
var stageCTools = map[string]stageCHints{
	// Updates: 18 tools, not destructive and idempotent.
	"add_document_group":                       {false, true},
	"add_document_user":                        {false, true},
	"add_collection_group":                     {false, true},
	"add_collection_user":                      {false, true},
	"activate_user":                            {false, true},
	"resend_user_invite":                       {false, true},
	"update_user":                              {false, true},
	"update_user_role":                         {false, true},
	"add_group_user":                           {false, true},
	"update_group":                             {false, true},
	"update_group_user":                        {false, true},
	"update_team":                              {false, true},
	"approve_access_request":                   {false, true},
	"start_authentication_provider_group_sync": {false, true},
	"update_authentication_provider":           {false, true},
	"cancel_import":                            {false, true},
	"update_oauth_client":                      {false, true},
	"update_share":                             {false, true},

	// Additive: 6 tools, not destructive; only create_share is idempotent (a repeat answers the same share).
	"export_collection_all": {false, false},
	"invite_users":          {false, false},
	"request_user_delete":   {false, false},
	"create_group":          {false, false},
	"request_team_delete":   {false, false},
	"create_share":          {false, true},

	// Destructive: 24 tools, destructive and not idempotent.
	"archive_document":            {true, false},
	"delete_document":             {true, false},
	"remove_document_group":       {true, false},
	"remove_document_user":        {true, false},
	"archive_collection":          {true, false},
	"delete_collection":           {true, false},
	"remove_collection_group":     {true, false},
	"remove_collection_user":      {true, false},
	"suspend_user":                {true, false},
	"delete_group":                {true, false},
	"remove_group_user":           {true, false},
	"dismiss_access_request":      {true, false},
	"delete_attachment":           {true, false},
	"delete_comment":              {true, false},
	"delete_emoji":                {true, false},
	"delete_file_operation":       {true, false},
	"delete_import":               {true, false},
	"delete_integration":          {true, false},
	"delete_oauth_authentication": {true, false},
	"delete_oauth_client":         {true, false},
	"delete_revision":             {true, false},
	"revoke_share":                {true, false},
	"delete_template":             {true, false},
	"delete_webhook_subscription": {true, false},
}

// stageCToolCount is the number of MCP tools of stage c, and adminCeilingToolCount the number of tools an admin source offers
// at the admin ceiling, with list_sources (ADR 007 section 3.3; specs mcp-destructive-tools.feature).
const (
	stageCToolCount       = 48
	adminCeilingToolCount = 154
)

// TestStageCToolsCarryTheHandWrittenHints lists the tools of an admin source at the admin ceiling and checks each stage c tool
// against the literal table: the table holds 48 tools, every one is offered, the listing holds 154 tools in all, and every
// stage c tool is not read-only, with the destructive and idempotent hints the table gives.
func TestStageCToolsCarryTheHandWrittenHints(t *testing.T) {
	if len(stageCTools) != stageCToolCount {
		t.Fatalf("the table holds %d stage c tools, want %d", len(stageCTools), stageCToolCount)
	}
	svc := serviceWith(t, &fakeGateway{}, mustSource(t, "outline", "admin")).Capped(domain.AccessAdmin)
	var logs bytes.Buffer
	res, err := connect(t, svc, &logs).ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Tools) != adminCeilingToolCount {
		t.Fatalf("the admin ceiling offers %d tools, want %d with list_sources", len(res.Tools), adminCeilingToolCount)
	}

	byName := map[string]bool{}
	offered := 0
	for _, tl := range res.Tools {
		byName[tl.Name] = true
		want, stage := stageCTools[tl.Name]
		if !stage {
			continue
		}
		offered++
		a := tl.Annotations
		if a == nil {
			t.Errorf("%s: the tool carries no annotations", tl.Name)
			continue
		}
		if a.ReadOnlyHint {
			t.Errorf("%s: read-only is set, but every stage c tool changes the wiki", tl.Name)
		}
		if a.DestructiveHint == nil || *a.DestructiveHint != want.destructive {
			t.Errorf("%s: destructiveHint = %v, want %v", tl.Name, a.DestructiveHint, want.destructive)
		}
		if a.IdempotentHint != want.idempotent {
			t.Errorf("%s: idempotentHint = %v, want %v", tl.Name, a.IdempotentHint, want.idempotent)
		}
	}
	if offered != stageCToolCount {
		t.Errorf("the admin ceiling offers %d stage c tools, want %d", offered, stageCToolCount)
	}
	for name := range stageCTools {
		if !byName[name] {
			t.Errorf("%s: the admin ceiling offers no such tool", name)
		}
	}
}
