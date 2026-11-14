package operations

import (
	"slices"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

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

// TestFileOperationsListIsReadButNeedsTheAdminRole records the owner decision 17: the class stays
// read, and the row says the server needs an admin user.
func TestFileOperationsListIsReadButNeedsTheAdminRole(t *testing.T) {
	row := rowOf(t, domain.OpFileOperationsList)
	if row.Effect != domain.EffectRead {
		t.Fatalf("file-operations.list effect = %v, want read", row.Effect)
	}
	if !strings.Contains(row.Description, "admin") {
		t.Fatalf("file-operations.list must say that the server needs an admin user, got %q", row.Description)
	}
}
