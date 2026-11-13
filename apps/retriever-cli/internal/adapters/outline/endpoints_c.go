package outline

import (
	"context"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// The Outline method names of the stage c operations of this slice. Each one is the wire path /api/<method>, and each is
// used by one admin endpoint row below.
const (
	methodDocumentsAddGroup    = "documents.add_group"
	methodDocumentsAddUser     = "documents.add_user"
	methodDocumentsArchive     = "documents.archive"
	methodDocumentsDelete      = "documents.delete"
	methodDocumentsEmptyTrash  = "documents.empty_trash"
	methodDocumentsRemoveGroup = "documents.remove_group"
	methodDocumentsRemoveUser  = "documents.remove_user"

	methodCollectionsAddGroup    = "collections.add_group"
	methodCollectionsAddUser     = "collections.add_user"
	methodCollectionsArchive     = "collections.archive"
	methodCollectionsDelete      = "collections.delete"
	methodCollectionsExportAll   = "collections.export_all"
	methodCollectionsRemoveGroup = "collections.remove_group"
	methodCollectionsRemoveUser  = "collections.remove_user"

	methodUsersActivate      = "users.activate"
	methodUsersDelete        = "users.delete"
	methodUsersInvite        = "users.invite"
	methodUsersRequestDelete = "users.requestDelete"
	methodUsersResendInvite  = "users.resendInvite"
	methodUsersSuspend       = "users.suspend"
	methodUsersUpdate        = "users.update"
	methodUsersUpdateRole    = "users.update_role"

	methodGroupsAddUser    = "groups.add_user"
	methodGroupsCreate     = "groups.create"
	methodGroupsDelete     = "groups.delete"
	methodGroupsDeleteAll  = "groups.deleteAll"
	methodGroupsRemoveUser = "groups.remove_user"
	methodGroupsUpdate     = "groups.update"
	methodGroupsUpdateUser = "groups.update_user"

	methodTeamsDelete        = "teams.delete"
	methodTeamsRequestDelete = "teams.requestDelete"
	methodTeamsUpdate        = "teams.update"

	methodAccessRequestsApprove                 = "accessRequests.approve"
	methodAccessRequestsDismiss                 = "accessRequests.dismiss"
	methodAttachmentsDelete                     = "attachments.delete"
	methodAuthenticationProvidersDelete         = "authenticationProviders.delete"
	methodAuthenticationProvidersStartGroupSync = "authenticationProviders.startGroupSync"
	methodAuthenticationProvidersUpdate         = "authenticationProviders.update"
	methodCommentsDelete                        = "comments.delete"
	methodEmojisDelete                          = "emojis.delete"
	methodFileOperationsDelete                  = "fileOperations.delete"
	methodImportsCancel                         = "imports.cancel"
	methodImportsDelete                         = "imports.delete"
	methodIntegrationsCreate                    = "integrations.create"
	methodIntegrationsDelete                    = "integrations.delete"
	methodIntegrationsUpdate                    = "integrations.update"

	// Stage c, slice c4: OAuth authorizations and clients, revisions, shares, templates, and webhook subscriptions.
	methodOAuthAuthenticationsDelete = "oauthAuthentications.delete"
	methodOAuthClientsCreate         = "oauthClients.create"
	methodOAuthClientsDelete         = "oauthClients.delete"
	methodOAuthClientsRotateSecret   = "oauthClients.rotate_secret"
	methodOAuthClientsUpdate         = "oauthClients.update"
	methodRevisionsDelete            = "revisions.delete"
	methodSharesCreate               = "shares.create"
	methodSharesRevoke               = "shares.revoke"
	methodSharesUpdate               = "shares.update"
	methodTemplatesDelete            = "templates.delete"
	methodWebhookSubscriptionsCreate = "webhookSubscriptions.create"
	methodWebhookSubscriptionsDelete = "webhookSubscriptions.delete"
	methodWebhookSubscriptionsUpdate = "webhookSubscriptions.update"
)

// adminEndpoints maps the stage c operations of this slice to their endpoints. It is
// kept apart from the read and write tables, so neither the reader nor the writer can reach an admin method. The access of
// each row comes from the catalog, and the client's method allowlist checks it. Each body
// names only the keys of the operation's parameters, and an argument that is not set is left out. A call is sent once; the
// client retries it only after a 429.
var adminEndpoints = map[domain.OperationID]endpoint{
	domain.OpDocumentsAddGroup: {
		method: methodDocumentsAddGroup,
		body:   argsBody(false, renamed("documentId", "id"), "documentId", "groupId", "permission"),
		reply:  projectNestedOf("groupMemberships", groupMembershipSpec),
	},
	domain.OpDocumentsAddUser: {
		method: methodDocumentsAddUser,
		body:   argsBody(false, renamed("documentId", "id"), "documentId", "userId", "permission"),
		reply:  projectNestedOf("memberships", membershipSpec),
	},
	domain.OpDocumentsArchive: {
		method: methodDocumentsArchive,
		body:   argsBody(false, renamed("documentId", "id"), "documentId", "reason"),
		reply:  projectObjectOf(documentRecord),
	},
	domain.OpDocumentsDelete: {
		method: methodDocumentsDelete,
		body:   argsBody(false, renamed("documentId", "id"), "documentId", "permanent", "reason"),
		reply:  projectAck,
	},
	domain.OpDocumentsEmptyTrash: {
		method: methodDocumentsEmptyTrash,
		body:   emptyBody,
		reply:  projectAck,
	},
	domain.OpDocumentsRemoveGroup: {
		method: methodDocumentsRemoveGroup,
		body:   argsBody(false, renamed("documentId", "id"), "documentId", "groupId"),
		reply:  projectAck,
	},
	domain.OpDocumentsRemoveUser: {
		method: methodDocumentsRemoveUser,
		body:   argsBody(false, renamed("documentId", "id"), "documentId", "userId"),
		reply:  projectAck,
	},
	domain.OpCollectionsAddGroup: {
		method: methodCollectionsAddGroup,
		body:   argsBody(false, renamed("collectionId", "id"), "collectionId", "groupId", "permission"),
		reply:  projectNestedOf("groupMemberships", groupMembershipSpec),
	},
	domain.OpCollectionsAddUser: {
		method: methodCollectionsAddUser,
		body:   argsBody(false, renamed("collectionId", "id"), "collectionId", "userId", "permission"),
		reply:  projectNestedOf("memberships", membershipSpec),
	},
	domain.OpCollectionsArchive: {
		method: methodCollectionsArchive,
		body:   argsBody(false, renamed("collectionId", "id"), "collectionId", "reason"),
		reply:  projectCollectionInfo,
	},
	domain.OpCollectionsDelete: {
		method: methodCollectionsDelete,
		body:   argsBody(false, renamed("collectionId", "id"), "collectionId", "reason"),
		reply:  projectAck,
	},
	domain.OpCollectionsExportAll: {
		method: methodCollectionsExportAll,
		body:   argsBody(false, nil, "format", "includeAttachments", "includePrivate"),
		reply:  projectAckUnder("fileOperation", fileOperationSpec),
	},
	domain.OpCollectionsRemoveGroup: {
		method: methodCollectionsRemoveGroup,
		body:   argsBody(false, renamed("collectionId", "id"), "collectionId", "groupId"),
		reply:  projectAck,
	},
	domain.OpCollectionsRemoveUser: {
		method: methodCollectionsRemoveUser,
		body:   argsBody(false, renamed("collectionId", "id"), "collectionId", "userId"),
		reply:  projectAck,
	},
	domain.OpUsersActivate: {
		method: methodUsersActivate,
		body:   argsBody(false, renamed("userId", "id"), "userId"),
		reply:  projectObjectOf(userSpec),
	},
	domain.OpUsersDelete: {
		method: methodUsersDelete,
		body:   argsBody(false, renamed("userId", "id"), "userId"),
		reply:  projectAck,
	},
	domain.OpUsersInvite: {
		method: methodUsersInvite,
		body:   inviteBody,
		reply:  projectNestedOf("users", userSpec),
	},
	domain.OpUsersRequestDelete: {
		method: methodUsersRequestDelete,
		body:   emptyBody,
		reply:  projectAck,
	},
	domain.OpUsersResendInvite: {
		method: methodUsersResendInvite,
		body:   argsBody(false, renamed("userId", "id"), "userId"),
		reply:  projectAck,
	},
	domain.OpUsersSuspend: {
		method: methodUsersSuspend,
		body:   argsBody(false, renamed("userId", "id"), "userId"),
		reply:  projectObjectOf(userSpec),
	},
	domain.OpUsersUpdate: {
		method: methodUsersUpdate,
		body:   argsBody(false, renamed("userId", "id"), "userId", "name", "language", "timezone"),
		reply:  projectObjectOf(userSpec),
	},
	domain.OpUsersUpdateRole: {
		method: methodUsersUpdateRole,
		body:   argsBody(false, renamed("userId", "id"), "userId", "role"),
		reply:  projectObjectOf(userSpec),
	},
	domain.OpGroupsAddUser: {
		method: methodGroupsAddUser,
		body:   argsBody(false, renamed("groupId", "id"), "groupId", "userId", "permission"),
		reply:  projectNestedOf("groupMemberships", groupUserSpec),
	},
	domain.OpGroupsCreate: {
		method: methodGroupsCreate,
		body:   argsBody(false, nil, "name", "description", "color", "icon", "disableMentions"),
		reply:  projectObjectOf(groupSpec),
	},
	domain.OpGroupsDelete: {
		method: methodGroupsDelete,
		body:   argsBody(false, renamed("groupId", "id"), "groupId"),
		reply:  projectAck,
	},
	domain.OpGroupsDeleteAll: {
		method: methodGroupsDeleteAll,
		body:   argsBody(false, nil, "authenticationProviderId"),
		reply:  projectAck,
	},
	domain.OpGroupsRemoveUser: {
		method: methodGroupsRemoveUser,
		body:   argsBody(false, renamed("groupId", "id"), "groupId", "userId"),
		reply:  projectNestedOf("groups", groupSpec),
	},
	domain.OpGroupsUpdate: {
		method: methodGroupsUpdate,
		body:   argsBody(false, renamed("groupId", "id"), "groupId", "name", "description", "color", "icon", "disableMentions"),
		reply:  projectObjectOf(groupSpec),
	},
	domain.OpGroupsUpdateUser: {
		method: methodGroupsUpdateUser,
		body:   argsBody(false, renamed("groupId", "id"), "groupId", "userId", "permission"),
		reply:  projectNestedOf("groupMemberships", groupUserSpec),
	},
	domain.OpTeamsDelete: {
		method: methodTeamsDelete,
		body:   argsBody(false, nil, "code"),
		reply:  projectAck,
	},
	domain.OpTeamsRequestDelete: {
		method: methodTeamsRequestDelete,
		body:   emptyBody,
		reply:  projectAck,
	},
	domain.OpTeamsUpdate: {
		method: methodTeamsUpdate,
		body:   argsBody(false, nil, "name", "description", "sharing", "documentEmbeds", "memberCollectionCreate", "defaultUserRole"),
		reply:  projectObjectOf(teamSpec),
	},
	domain.OpAccessRequestsApprove: {
		method: methodAccessRequestsApprove,
		body:   argsBody(false, renamed("accessRequestId", "id"), "accessRequestId", "permission"),
		reply:  projectObjectOf(accessRequestSpec),
	},
	domain.OpAccessRequestsDismiss: {
		method: methodAccessRequestsDismiss,
		body:   argsBody(false, renamed("accessRequestId", "id"), "accessRequestId"),
		reply:  projectObjectOf(accessRequestSpec),
	},
	domain.OpAttachmentsDelete: {
		method: methodAttachmentsDelete,
		body:   argsBody(false, renamed("attachmentId", "id"), "attachmentId"),
		reply:  projectAck,
	},
	domain.OpAuthenticationProvidersDelete: {
		method: methodAuthenticationProvidersDelete,
		body:   argsBody(false, renamed("authenticationProviderId", "id"), "authenticationProviderId"),
		reply:  projectAck,
	},
	domain.OpAuthenticationProvidersStartGroupSync: {
		method: methodAuthenticationProvidersStartGroupSync,
		body:   argsBody(false, renamed("authenticationProviderId", "id"), "authenticationProviderId"),
		reply:  projectStarted,
	},
	domain.OpAuthenticationProvidersUpdate: {
		method:  methodAuthenticationProvidersUpdate,
		body:    authProviderUpdateBody,
		reply:   projectObjectOf(authProviderSpec),
		current: currentAuthProvider,
	},
	domain.OpCommentsDelete: {
		method: methodCommentsDelete,
		body:   argsBody(false, renamed("commentId", "id"), "commentId"),
		reply:  projectAck,
	},
	domain.OpEmojisDelete: {
		method: methodEmojisDelete,
		body:   argsBody(false, renamed("emojiId", "id"), "emojiId"),
		reply:  projectAck,
	},
	domain.OpFileOperationsDelete: {
		method: methodFileOperationsDelete,
		body:   argsBody(false, renamed("fileOperationId", "id"), "fileOperationId"),
		reply:  projectAck,
	},
	domain.OpImportsCancel: {
		method: methodImportsCancel,
		body:   argsBody(false, renamed("importId", "id"), "importId"),
		reply:  projectObjectOf(importSpec),
	},
	domain.OpImportsDelete: {
		method: methodImportsDelete,
		body:   argsBody(false, renamed("importId", "id"), "importId"),
		reply:  projectAck,
	},
	domain.OpIntegrationsCreate: {
		method: methodIntegrationsCreate,
		body:   integrationCreateBody,
		reply:  projectObjectOf(integrationSpec),
	},
	domain.OpIntegrationsDelete: {
		method: methodIntegrationsDelete,
		body:   argsBody(false, renamed("integrationId", "id"), "integrationId"),
		reply:  projectAck,
	},
	domain.OpIntegrationsUpdate: {
		method:  methodIntegrationsUpdate,
		body:    integrationUpdateBody,
		reply:   projectObjectOf(integrationSpec),
		current: currentIntegration,
	},
	domain.OpOAuthAuthenticationsDelete: {
		method: methodOAuthAuthenticationsDelete,
		body:   argsBody(false, nil, "oauthClientId"),
		reply:  projectAck,
	},
	domain.OpOAuthClientsCreate: {
		method: methodOAuthClientsCreate,
		body:   argsBody(false, nil, "name", "redirectUris", "clientType", "description", "published"),
		reply:  projectObjectOf(oauthClientSpec),
	},
	domain.OpOAuthClientsDelete: {
		method: methodOAuthClientsDelete,
		body:   argsBody(false, renamed("oauthClientId", "id"), "oauthClientId"),
		reply:  projectAck,
	},
	domain.OpOAuthClientsRotateSecret: {
		method: methodOAuthClientsRotateSecret,
		body:   argsBody(false, renamed("oauthClientId", "id"), "oauthClientId"),
		reply:  projectObjectOf(oauthClientSpec),
	},
	domain.OpOAuthClientsUpdate: {
		method: methodOAuthClientsUpdate,
		body:   argsBody(false, renamed("oauthClientId", "id"), "oauthClientId", "name", "description", "published", "clientType"),
		reply:  projectObjectOf(oauthClientSpec),
	},
	domain.OpRevisionsDelete: {
		method: methodRevisionsDelete,
		body:   argsBody(false, renamed("revisionId", "id"), "revisionId"),
		reply:  projectAck,
	},
	domain.OpSharesCreate: {
		method: methodSharesCreate,
		body: argsBody(false, nil, "documentId", "collectionId", "published", "includeChildDocuments", "allowIndexing",
			"allowSubscriptions", "showLastUpdated", "showTOC"),
		reply: projectObjectOf(shareSpec),
	},
	domain.OpSharesRevoke: {
		method: methodSharesRevoke,
		body:   argsBody(false, renamed("shareId", "id"), "shareId"),
		reply:  projectAck,
	},
	domain.OpSharesUpdate: {
		method: methodSharesUpdate,
		body: argsBody(false, renamed("shareId", "id"), "shareId", "published", "includeChildDocuments", "allowIndexing",
			"allowSubscriptions", "showLastUpdated", "showTOC", "title"),
		reply: projectObjectOf(shareSpec),
	},
	domain.OpTemplatesDelete: {
		method: methodTemplatesDelete,
		body:   argsBody(false, renamed("templateId", "id"), "templateId"),
		reply:  projectAck,
	},
	domain.OpWebhookSubscriptionsCreate: {
		method: methodWebhookSubscriptionsCreate,
		body:   argsBody(false, nil, "name", "url", "events"),
		reply:  projectObjectOf(webhookSpec),
	},
	domain.OpWebhookSubscriptionsDelete: {
		method: methodWebhookSubscriptionsDelete,
		body:   argsBody(false, renamed("webhookSubscriptionId", "id"), "webhookSubscriptionId"),
		reply:  projectAck,
	},
	domain.OpWebhookSubscriptionsUpdate: {
		method: methodWebhookSubscriptionsUpdate,
		body:   argsBody(false, renamed("webhookSubscriptionId", "id"), "webhookSubscriptionId", "name", "url", "events"),
		reply:  projectObjectOf(webhookSpec),
	},
}

// inviteBody is the body of users.invite. Outline takes a list of invitations, and Retriever sends one per call, so the list
// holds one invitation with the e-mail address, the name, and the role. suppressEmail is sent only when the caller set it.
func inviteBody(a domain.Args) map[string]any {
	invite := map[string]any{"email": a.String("email"), "name": a.String("name"), "role": a.String("role")}
	body := map[string]any{"invites": []any{invite}}
	if a.Has("suppressEmail") {
		body["suppressEmail"] = a.Bool("suppressEmail")
	}
	return body
}

// currentSettingsKey names the argument that carries the settings object read before an integration update. It is set only
// by currentIntegration, after Bind has checked the caller's arguments, so no caller can send it.
const currentSettingsKey = "currentSettings"

// integrationSettings returns the settings of an integration request. The wiki replaces the settings of an integration, so
// the request resends every key of the settings read before an update (the measurement ID, the instance address, and any
// other key the wiki holds), and replaces only the address, and the channel and the channel ID when the caller set them.
// A create has no settings read before it, so its request holds the address and the values the caller gave.
func integrationSettings(a domain.Args) map[string]any {
	settings := map[string]any{}
	for k, v := range currentSettingsOf(a) {
		settings[k] = v
	}
	settings["url"] = a.String("url")
	values := a.Values()
	for _, name := range []string{"channel", "channelId"} {
		if v, ok := values[name]; ok {
			settings[name] = v
		}
	}
	return settings
}

// currentSettingsOf returns the settings object read before an integration update, or nil when none was read.
func currentSettingsOf(a domain.Args) map[string]any {
	settings, _ := a.Values()[currentSettingsKey].(map[string]any)
	return settings
}

// integrationCreateBody is the body of integrations.create: the type, the service, and the settings with the address.
func integrationCreateBody(a domain.Args) map[string]any {
	return map[string]any{"type": a.String("type"), "service": a.String("service"), "settings": integrationSettings(a)}
}

// integrationUpdateBody is the body of integrations.update. The wiki replaces the settings, and it resets the events to none
// when the request leaves them out, so the request carries the events and the channels read before the change
// (currentIntegration; the inventory, Finding 9).
func integrationUpdateBody(a domain.Args) map[string]any {
	body := map[string]any{"id": a.String("integrationId"), "settings": integrationSettings(a)}
	if a.Has("events") {
		body["events"] = a.Values()["events"]
	}
	return body
}

// authProviderUpdateBody is the body of authenticationProviders.update: the ID, isEnabled when the caller set it, and the
// group sync settings. The settings are the ones the caller set, and the ones read before the change (currentAuthProvider),
// so the request never drops a value the wiki holds.
func authProviderUpdateBody(a domain.Args) map[string]any {
	body := map[string]any{"id": a.String("authenticationProviderId")}
	if a.Has("isEnabled") {
		body["isEnabled"] = a.Bool("isEnabled")
	}
	values := a.Values()
	settings := map[string]any{}
	for _, name := range []string{"groupSyncEnabled", "groupClaim", "groupSyncScopes"} {
		if v, ok := values[name]; ok {
			settings[name] = v
		}
	}
	if len(settings) > 0 {
		body["settings"] = settings
	}
	return body
}

// currentIntegration reads the integration before its change. It fills the events the caller did not set, and it carries the
// settings object of the wiki, so the update resends every key of it (integrationSettings).
func currentIntegration(ctx context.Context, g *Gateway, a domain.Args) (domain.Args, error) {
	cur, err := g.readCurrent(ctx, domain.OpIntegrationsGet, a, currentIntegrationSpec)
	if err != nil {
		return domain.Args{}, err
	}
	filled := withMissing(a, cur, "events").Values()
	if v, ok := cur.Get("settings"); ok {
		filled[currentSettingsKey] = v
	}
	return domain.NewArgs(filled), nil
}

// currentAuthProvider reads the sign-in provider before its change, and fills the group sync settings the caller did not set
// from it. A setting the reply does not hold is not filled, so the request sends no default for it (m1).
func currentAuthProvider(ctx context.Context, g *Gateway, a domain.Args) (domain.Args, error) {
	cur, err := g.readCurrent(ctx, domain.OpAuthenticationProvidersGet, a, currentAuthProviderSpec)
	if err != nil {
		return domain.Args{}, err
	}
	return withMissing(a, cur, "groupSyncEnabled", "groupClaim", "groupSyncScopes"), nil
}

// readCurrent reads one object with its information method, through the same client and with the reply projection the
// caller gives. The read is an admin read, so it is checked like any other call of the source.
func (g *Gateway) readCurrent(ctx context.Context, op domain.OperationID, a domain.Args,
	reply func(any) (domain.Record, error)) (domain.Record, error) {
	ep, ok := endpoints[op]
	if !ok {
		return domain.Record{}, domain.ErrUnsupportedOperation
	}
	return g.send(ctx, op, ep.method, ep.body(a), reply)
}

// withMissing returns the arguments with each named value of rec that the caller did not set. A value the wiki holds as null
// is not filled, so the request leaves it out.
func withMissing(a domain.Args, rec domain.Record, names ...string) domain.Args {
	values := a.Values()
	for _, name := range names {
		if _, set := values[name]; set {
			continue
		}
		if v, ok := rec.Get(name); ok && v != nil {
			values[name] = v
		}
	}
	return domain.NewArgs(values)
}

// currentIntegrationSpec projects the parts of an integration that the update resends: its events, and a copy of its settings
// object. The address is left out of the copy, because the update replaces it, so the old address is never read into the
// Record.
func currentIntegrationSpec(root any) (domain.Record, error) {
	m, err := dataObject(root)
	if err != nil {
		return domain.Record{}, err
	}
	settings := make(map[string]any)
	for k, v := range subObject(m, "settings") {
		if k != "url" {
			settings[k] = v
		}
	}
	return domain.NewRecord(
		field("events", stringsOf(m, "events")),
		field("settings", settings),
	), nil
}

// currentAuthProviderSpec projects the group sync settings of a sign-in provider, which the update resends. A setting the
// reply does not hold is left out of the Record, so no default reaches the request (m1); a setting the wiki holds as null
// is left out as well, by withMissing.
func currentAuthProviderSpec(root any) (domain.Record, error) {
	m, err := dataObject(root)
	if err != nil {
		return domain.Record{}, err
	}
	settings := subObject(m, "settings")
	var fields []domain.Field
	for _, name := range []string{"groupSyncEnabled", "groupClaim", "groupSyncScopes"} {
		if v, ok := settings[name]; ok {
			fields = append(fields, field(name, v))
		}
	}
	return domain.NewRecord(fields...), nil
}
