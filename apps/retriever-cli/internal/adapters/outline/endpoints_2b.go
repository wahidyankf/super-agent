package outline

import (
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// The Outline method names of the stage a part 2b reads. Each one is the wire path /api/<method>. Each is used by one
// endpoint row below. The eighteen reads of access read come first, then the ten admin-level reads.
const (
	methodAccessRequestsInfo          = "accessRequests.info"
	methodAttachmentsList             = "attachments.list"
	methodEmojisInfo                  = "emojis.info"
	methodEmojisList                  = "emojis.list"
	methodEventsList                  = "events.list"
	methodFileOperationsInfo          = "fileOperations.info"
	methodFileOperationsList          = "fileOperations.list"
	methodGroupMembershipsList        = "groupMemberships.list"
	methodInstallationInfo            = "installation.info"
	methodNotificationsList           = "notifications.list"
	methodOAuthAuthenticationsList    = "oauthAuthentications.list"
	methodReactionsList               = "reactions.list"
	methodRelationshipsInfo           = "relationships.info"
	methodRelationshipsList           = "relationships.list"
	methodSearchesList                = "searches.list"
	methodSuggestionsMention          = "suggestions.mention"
	methodUserMembershipsList         = "userMemberships.list"
	methodViewsList                   = "views.list"
	methodAPIKeysList                 = "apiKeys.list"
	methodAuthenticationProvidersInfo = "authenticationProviders.info"
	methodAuthenticationProvidersList = "authenticationProviders.list"
	methodImportsInfo                 = "imports.info"
	methodImportsList                 = "imports.list"
	methodIntegrationsInfo            = "integrations.info"
	methodIntegrationsList            = "integrations.list"
	methodOAuthClientsInfo            = "oauthClients.info"
	methodOAuthClientsList            = "oauthClients.list"
	methodWebhookSubscriptionsList    = "webhookSubscriptions.list"
)

// part2bEndpoints maps the stage a part 2b operations to their endpoints. init merges them into endpoints, so the table
// of the stage a part 2a stays in endpoints.go. The bodies name only the keys of each row's parameters; an argument
// that is not set is left out.
var part2bEndpoints = map[domain.OperationID]endpoint{
	domain.OpAccessRequestsGet: {
		method: methodAccessRequestsInfo,
		body:   idBody("accessRequestId"),
		reply:  projectObjectOf(accessRequestSpec),
	},
	domain.OpAttachmentsList: {
		method: methodAttachmentsList,
		body:   argsBody(true, nil, "documentId", "userId"),
		reply:  projectListOf(attachmentSpec),
	},
	domain.OpEmojisGet: {
		method: methodEmojisInfo,
		body:   idBody("emojiId"),
		reply:  projectObjectOf(emojiSpec),
	},
	domain.OpEmojisList: {
		method: methodEmojisList,
		body:   argsBody(true, nil, "query"),
		reply:  projectListOf(emojiSpec),
	},
	domain.OpEventsList: {
		method: methodEventsList,
		body: argsBody(true, nil, "actorId", "documentId", "collectionId", "auditLog", "sort",
			"direction"),
		reply: projectListOf(eventSpec),
	},
	domain.OpFileOperationsGet: {
		method: methodFileOperationsInfo,
		body:   idBody("fileOperationId"),
		reply:  projectObjectOf(fileOperationSpec),
	},
	domain.OpFileOperationsList: {
		method: methodFileOperationsList,
		body:   argsBody(true, nil, "type", "sort", "direction"),
		reply:  projectListOf(fileOperationSpec),
	},
	domain.OpGroupMembershipsList: {
		method: methodGroupMembershipsList,
		body:   argsBody(true, nil, "groupId"),
		reply:  projectNestedOf("groupMemberships", groupMembershipSpec),
	},
	domain.OpInstallationGet: {
		method: methodInstallationInfo,
		body:   emptyBody,
		reply:  projectObjectOf(installationSpec),
	},
	domain.OpNotificationsList: {
		method: methodNotificationsList,
		body:   argsBody(true, nil, "eventType", "archived"),
		reply:  projectNestedOf("notifications", notificationSpec),
	},
	domain.OpOAuthAuthenticationsList: {
		method: methodOAuthAuthenticationsList,
		body:   pageBody,
		reply:  projectListOf(oauthAuthenticationSpec),
	},
	domain.OpReactionsList: {
		method: methodReactionsList,
		body:   argsBody(true, nil, "commentId"),
		reply:  projectListOf(reactionSpec),
	},
	domain.OpRelationshipsGet: {
		method: methodRelationshipsInfo,
		body:   idBody("relationshipId"),
		reply:  projectUnder("relationship", relationshipSpec),
	},
	domain.OpRelationshipsList: {
		method: methodRelationshipsList,
		body:   argsBody(true, nil, "type", "documentId", "reverseDocumentId"),
		reply:  projectNestedOf("relationships", relationshipSpec),
	},
	domain.OpSearchesList: {
		method: methodSearchesList,
		body:   argsBody(true, renamed("searchSource", "source"), "searchSource"),
		reply:  projectListOf(searchSpec),
	},
	domain.OpSuggestionsMention: {
		method: methodSuggestionsMention,
		body:   argsBody(true, nil, "query"),
		reply:  projectSuggestions,
	},
	domain.OpUserMembershipsList: {
		method: methodUserMembershipsList,
		body:   pageBody,
		reply:  projectNestedOf("memberships", membershipSpec),
	},
	domain.OpViewsList: {
		method: methodViewsList,
		body:   argsBody(false, nil, "documentId", "includeSuspended"),
		reply:  projectListOf(viewSpec),
	},
	domain.OpAPIKeysList: {
		method: methodAPIKeysList,
		body:   argsBody(true, nil, "userId", "query", "sort", "direction"),
		reply:  projectListOf(apiKeySpec),
	},
	domain.OpAuthenticationProvidersGet: {
		method: methodAuthenticationProvidersInfo,
		body:   idBody("authenticationProviderId"),
		reply:  projectObjectOf(authProviderSpec),
	},
	domain.OpAuthenticationProvidersList: {
		method: methodAuthenticationProvidersList,
		body:   emptyBody,
		reply:  projectListOf(authProviderSpec),
	},
	domain.OpImportsGet: {
		method: methodImportsInfo,
		body:   idBody("importId"),
		reply:  projectObjectOf(importSpec),
	},
	domain.OpImportsList: {
		method: methodImportsList,
		body:   argsBody(true, nil, "service", "sort", "direction"),
		reply:  projectListOf(importSpec),
	},
	domain.OpIntegrationsGet: {
		method: methodIntegrationsInfo,
		body:   idBody("integrationId"),
		reply:  projectObjectOf(integrationSpec),
	},
	domain.OpIntegrationsList: {
		method: methodIntegrationsList,
		body:   argsBody(true, nil, "type", "service", "sort", "direction"),
		reply:  projectListOf(integrationSpec),
	},
	domain.OpOAuthClientsGet: {
		method: methodOAuthClientsInfo,
		body:   idBody("oauthClientId"),
		reply:  projectObjectOf(oauthClientSpec),
	},
	domain.OpOAuthClientsList: {
		method: methodOAuthClientsList,
		body:   pageBody,
		reply:  projectListOf(oauthClientSpec),
	},
	domain.OpWebhookSubscriptionsList: {
		method: methodWebhookSubscriptionsList,
		body:   argsBody(true, nil, "query", "sort", "direction"),
		reply:  projectListOf(webhookSpec),
	},
}

// init merges the part 2b endpoints into the table of the reader.
func init() {
	for op, ep := range part2bEndpoints {
		endpoints[op] = ep
	}
}
