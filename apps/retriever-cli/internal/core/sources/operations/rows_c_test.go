package operations

import (
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// stageCRows is the expected shape of each stage c row of this slice.
// Each entry holds the operation, its CLI path, its MCP tool ("" for none), its effect, and whether it is critical.
var stageCRows = []struct {
	op     domain.OperationID
	cli    string
	tool   string
	effect domain.Effect
}{
	{domain.OpDocumentsAddGroup, "outline documents add-group", "add_document_group", domain.EffectUpdate},
	{domain.OpDocumentsAddUser, "outline documents add-user", "add_document_user", domain.EffectUpdate},
	{domain.OpDocumentsArchive, "outline documents archive", "archive_document", domain.EffectDestructive},
	{domain.OpDocumentsDelete, "outline documents delete", "delete_document", domain.EffectDestructive},
	{domain.OpDocumentsEmptyTrash, "outline documents empty-trash", "empty_trash", domain.EffectDestructive},
	{domain.OpDocumentsRemoveGroup, "outline documents remove-group", "remove_document_group", domain.EffectDestructive},
	{domain.OpDocumentsRemoveUser, "outline documents remove-user", "remove_document_user", domain.EffectDestructive},
	{domain.OpCollectionsAddGroup, "outline collections add-group", "add_collection_group", domain.EffectUpdate},
	{domain.OpCollectionsAddUser, "outline collections add-user", "add_collection_user", domain.EffectUpdate},
	{domain.OpCollectionsArchive, "outline collections archive", "archive_collection", domain.EffectDestructive},
	{domain.OpCollectionsDelete, "outline collections delete", "delete_collection", domain.EffectDestructive},
	{domain.OpCollectionsExportAll, "outline collections export-all", "export_collection_all", domain.EffectAdditive},
	{domain.OpCollectionsRemoveGroup, "outline collections remove-group", "remove_collection_group", domain.EffectDestructive},
	{domain.OpCollectionsRemoveUser, "outline collections remove-user", "remove_collection_user", domain.EffectDestructive},
	// Slice c2: the users, groups, and teams rows. The three critical ones have no tool.
	{domain.OpUsersActivate, "outline users activate", "activate_user", domain.EffectUpdate},
	{domain.OpUsersDelete, "outline users delete", "delete_user", domain.EffectDestructive},
	{domain.OpUsersInvite, "outline users invite", "invite_users", domain.EffectAdditive},
	{domain.OpUsersRequestDelete, "outline users request-delete", "request_user_delete", domain.EffectAdditive},
	{domain.OpUsersResendInvite, "outline users resend-invite", "resend_user_invite", domain.EffectUpdate},
	{domain.OpUsersSuspend, "outline users suspend", "suspend_user", domain.EffectDestructive},
	{domain.OpUsersUpdate, "outline users update", "update_user", domain.EffectUpdate},
	{domain.OpUsersUpdateRole, "outline users update-role", "update_user_role", domain.EffectUpdate},
	{domain.OpGroupsAddUser, "outline groups add-user", "add_group_user", domain.EffectUpdate},
	{domain.OpGroupsCreate, "outline groups create", "create_group", domain.EffectAdditive},
	{domain.OpGroupsDelete, "outline groups delete", "delete_group", domain.EffectDestructive},
	{domain.OpGroupsDeleteAll, "outline groups delete-all", "delete_all_groups", domain.EffectDestructive},
	{domain.OpGroupsRemoveUser, "outline groups remove-user", "remove_group_user", domain.EffectDestructive},
	{domain.OpGroupsUpdate, "outline groups update", "update_group", domain.EffectUpdate},
	{domain.OpGroupsUpdateUser, "outline groups update-user", "update_group_user", domain.EffectUpdate},
	{domain.OpTeamsDelete, "outline teams delete", "delete_team", domain.EffectDestructive},
	{domain.OpTeamsRequestDelete, "outline teams request-delete", "request_team_delete", domain.EffectAdditive},
	{domain.OpTeamsUpdate, "outline teams update", "update_team", domain.EffectUpdate},
	// Slice c3: the access requests, attachments, sign-in providers, comments, emojis, file operations, imports, and integrations
	// rows. The three critical ones, the provider delete and the two integration changes, have no tool.
	{domain.OpAccessRequestsApprove, "outline access-requests approve", "approve_access_request", domain.EffectUpdate},
	{domain.OpAccessRequestsDismiss, "outline access-requests dismiss", "dismiss_access_request", domain.EffectDestructive},
	{domain.OpAttachmentsDelete, "outline attachments delete", "delete_attachment", domain.EffectDestructive},
	{domain.OpAuthenticationProvidersDelete, "outline authentication-providers delete", "delete_authentication_provider", domain.EffectDestructive},
	{domain.OpAuthenticationProvidersStartGroupSync, "outline authentication-providers start-group-sync", "start_authentication_provider_group_sync", domain.EffectUpdate},
	{domain.OpAuthenticationProvidersUpdate, "outline authentication-providers update", "update_authentication_provider", domain.EffectUpdate},
	{domain.OpCommentsDelete, "outline comments delete", "delete_comment", domain.EffectDestructive},
	{domain.OpEmojisDelete, "outline emojis delete", "delete_emoji", domain.EffectDestructive},
	{domain.OpFileOperationsDelete, "outline file-operations delete", "delete_file_operation", domain.EffectDestructive},
	{domain.OpImportsCancel, "outline imports cancel", "cancel_import", domain.EffectUpdate},
	{domain.OpImportsDelete, "outline imports delete", "delete_import", domain.EffectDestructive},
	{domain.OpIntegrationsCreate, "outline integrations create", "create_integration", domain.EffectAdditive},
	{domain.OpIntegrationsDelete, "outline integrations delete", "delete_integration", domain.EffectDestructive},
	{domain.OpIntegrationsUpdate, "outline integrations update", "update_integration", domain.EffectUpdate},
	// Slice c4: the OAuth authorizations and clients, revisions, shares, templates, and webhook subscriptions rows. The
	// four critical ones, the two OAuth client changes and the two webhook changes, have no tool.
	{domain.OpOAuthAuthenticationsDelete, "outline oauth-authentications delete", "delete_oauth_authentication", domain.EffectDestructive},
	{domain.OpOAuthClientsCreate, "outline oauth-clients create", "create_oauth_client", domain.EffectAdditive},
	{domain.OpOAuthClientsDelete, "outline oauth-clients delete", "delete_oauth_client", domain.EffectDestructive},
	{domain.OpOAuthClientsRotateSecret, "outline oauth-clients rotate-secret", "rotate_oauth_client_secret", domain.EffectUpdate},
	{domain.OpOAuthClientsUpdate, "outline oauth-clients update", "update_oauth_client", domain.EffectUpdate},
	{domain.OpRevisionsDelete, "outline revisions delete", "delete_revision", domain.EffectDestructive},
	{domain.OpSharesCreate, "outline shares create", "create_share", domain.EffectAdditive},
	{domain.OpSharesRevoke, "outline shares revoke", "revoke_share", domain.EffectDestructive},
	{domain.OpSharesUpdate, "outline shares update", "update_share", domain.EffectUpdate},
	{domain.OpTemplatesDelete, "outline templates delete", "delete_template", domain.EffectDestructive},
	{domain.OpWebhookSubscriptionsCreate, "outline webhook-subscriptions create", "create_webhook_subscription", domain.EffectAdditive},
	{domain.OpWebhookSubscriptionsDelete, "outline webhook-subscriptions delete", "delete_webhook_subscription", domain.EffectDestructive},
	{domain.OpWebhookSubscriptionsUpdate, "outline webhook-subscriptions update", "update_webhook_subscription", domain.EffectUpdate},
}

// stageCIDs returns the operation IDs of the stage c rows of this slice, in catalog order.
func stageCIDs() []domain.OperationID {
	ids := make([]domain.OperationID, 0, len(stageCRows))
	for _, row := range stageCRows {
		ids = append(ids, row.op)
	}
	return ids
}

func TestStageCRowsHaveTheInventoryCliToolAndEffect(t *testing.T) {
	for _, want := range stageCRows {
		row, ok := Lookup(want.op)
		if !ok {
			t.Fatalf("the catalog has no row %s", want.op)
		}
		if row.CLI != want.cli {
			t.Errorf("%s: CLI %q, want %q", want.op, row.CLI, want.cli)
		}
		if row.Tool != want.tool {
			t.Errorf("%s: tool %q, want %q", want.op, row.Tool, want.tool)
		}
		if row.Effect != want.effect {
			t.Errorf("%s: effect %s, want %s", want.op, row.Effect, want.effect)
		}
		if len(row.Fields) == 0 {
			t.Errorf("%s: no result fields", want.op)
		}
	}
}
