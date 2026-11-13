package operations

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// TestAdminReadsAreExactlyTheTenAdminRows checks that the rows of access admin are the ten admin-level reads, and that
// every other row needs read access. An admin read is a read in effect, so its effect stays read.
func TestAdminReadsAreExactlyTheTenAdminRows(t *testing.T) {
	var admin []domain.OperationID
	for _, r := range Rows() {
		// The stage c rows are admin too, and they are not reads; their own test checks them.
		if r.Access == domain.AccessAdmin && r.Effect == domain.EffectRead {
			admin = append(admin, r.ID)
		}
	}
	if !slices.Equal(admin, stageAAdminReads) {
		t.Fatalf("admin rows = %v, want %v", admin, stageAAdminReads)
	}
}

// TestAdminRowsNameTheAdminLevelInTheirText checks that the summary of an admin read says it needs an admin source, so a
// person or a model reads the level from the catalog text.
func TestAdminRowsNameTheAdminLevelInTheirText(t *testing.T) {
	for _, id := range stageAAdminReads {
		row := rowOf(t, id)
		if !strings.Contains(row.Description, "admin") {
			t.Errorf("%s: the description must say that it needs an admin source, got %q", id, row.Description)
		}
	}
}

// TestRefuseAdminReadOnAReadSourceNamesBothLevels checks the safe text of the admin refusal: the
// source ID and the two levels, and the sentinel ErrAccessLevelTooLow.
func TestRefuseAdminReadOnAReadSourceNamesBothLevels(t *testing.T) {
	src := sourceAt(t, domain.AccessRead)
	row := rowOf(t, domain.OpAPIKeysList)
	err := Refuse(row, src)
	if err == nil {
		t.Fatal("a read source must not run an admin read")
	}
	const want = `source "outline" allows read only; this operation needs admin`
	if err.Error() != want {
		t.Fatalf("Refuse text = %q, want %q", err.Error(), want)
	}
	if !errors.Is(err, domain.ErrAccessLevelTooLow) {
		t.Fatalf("Refuse error does not match ErrAccessLevelTooLow: %v", err)
	}
}

// TestRefuseAdminReadAllowsAnAdminSource checks that an admin source may run an admin read.
func TestRefuseAdminReadAllowsAnAdminSource(t *testing.T) {
	src := sourceAt(t, domain.AccessAdmin)
	for _, id := range stageAAdminReads {
		if err := Refuse(rowOf(t, id), src); err != nil {
			t.Fatalf("%s on an admin source = %v, want nil", id, err)
		}
	}
}

// TestEveryNewReadHasItsFixedToolName pins the MCP names of the eighteen part 2b reads and the ten admin reads, so a
// rename is a visible change.
func TestEveryNewReadHasItsFixedToolName(t *testing.T) {
	want := map[domain.OperationID]string{
		domain.OpAccessRequestsGet:           "get_access_request",
		domain.OpAttachmentsList:             "list_attachments",
		domain.OpEmojisGet:                   "get_emoji",
		domain.OpEmojisList:                  "list_emojis",
		domain.OpEventsList:                  "list_events",
		domain.OpFileOperationsGet:           "get_file_operation",
		domain.OpFileOperationsList:          "list_file_operations",
		domain.OpGroupMembershipsList:        "list_group_memberships",
		domain.OpInstallationGet:             "get_installation",
		domain.OpNotificationsList:           "list_notifications",
		domain.OpOAuthAuthenticationsList:    "list_oauth_authentications",
		domain.OpReactionsList:               "list_reactions",
		domain.OpRelationshipsGet:            "get_relationship",
		domain.OpRelationshipsList:           "list_relationships",
		domain.OpSearchesList:                "list_searches",
		domain.OpSuggestionsMention:          "suggest_mentions",
		domain.OpUserMembershipsList:         "list_user_memberships",
		domain.OpViewsList:                   "list_views",
		domain.OpAPIKeysList:                 "list_api_keys",
		domain.OpAuthenticationProvidersGet:  "get_authentication_provider",
		domain.OpAuthenticationProvidersList: "list_authentication_providers",
		domain.OpImportsGet:                  "get_import",
		domain.OpImportsList:                 "list_imports",
		domain.OpIntegrationsGet:             "get_integration",
		domain.OpIntegrationsList:            "list_integrations",
		domain.OpOAuthClientsGet:             "get_oauth_client",
		domain.OpOAuthClientsList:            "list_oauth_clients",
		domain.OpWebhookSubscriptionsList:    "list_webhook_subscriptions",
	}
	for id, tool := range want {
		if got := rowOf(t, id).Tool; got != tool {
			t.Errorf("%s tool = %q, want %q", id, got, tool)
		}
	}
}

// TestSearchSourceIsNotNamedSource checks that the search list does not use the name source for its own filter. The
// name source is the Retriever source ID of --source and of the MCP source input, so the server's search source is
// searchSource in the catalog (its wire name stays source).
func TestSearchSourceIsNotNamedSource(t *testing.T) {
	row := rowOf(t, domain.OpSearchesList)
	if _, clash := row.Param("source"); clash {
		t.Fatal("searches.list must not declare a parameter named source")
	}
	if _, ok := row.Param("searchSource"); !ok {
		t.Fatal("searches.list must declare the parameter searchSource")
	}
}

// TestRelationshipsListNeedsExactlyOneDocument checks that relationships.list takes exactly one of documentId and
// reverseDocumentId, as the server requires.
func TestRelationshipsListNeedsExactlyOneDocument(t *testing.T) {
	row := rowOf(t, domain.OpRelationshipsList)
	if !slices.Equal(row.OneOf, []string{"documentId", "reverseDocumentId"}) {
		t.Fatalf("OneOf = %v, want documentId and reverseDocumentId", row.OneOf)
	}
}

// TestFileOperationsListIsReadButNeedsTheAdminRole records the owner decision of ADR 007 (decision 17): the class stays
// read, and the row says the server needs an admin user.
func TestFileOperationsListIsReadButNeedsTheAdminRole(t *testing.T) {
	row := rowOf(t, domain.OpFileOperationsList)
	if row.Access != domain.AccessRead {
		t.Fatalf("file-operations.list access = %v, want read (ADR 007 decision 17)", row.Access)
	}
	if !strings.Contains(row.Description, "admin") {
		t.Fatalf("file-operations.list must say that the server needs an admin user, got %q", row.Description)
	}
}
