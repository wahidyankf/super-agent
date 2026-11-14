package domain

// OperationID names one operation of the Retriever operation catalog, such as documents.get. Every inbound adapter
// uses the same name for an operation; the Outline adapter maps it to its wire method.
type OperationID string

// The read operations of stage a. A new operation is a new constant and one catalog row. The first seven
// are the stage a part 1 reads; the rest are stage a part 2a.
const (
	OpCollectionsList      OperationID = "collections.list"
	OpCollectionsInfo      OperationID = "collections.info"
	OpCollectionsDocuments OperationID = "collections.documents"
	OpDocumentsGet         OperationID = "documents.get"
	OpDocumentsList        OperationID = "documents.list"
	OpDocumentsSearch      OperationID = "documents.search"
	OpAuthWhoami           OperationID = "auth.whoami"

	OpDocumentsArchived         OperationID = "documents.archived"
	OpDocumentsChildren         OperationID = "documents.children"
	OpDocumentsDeleted          OperationID = "documents.deleted"
	OpDocumentsDrafts           OperationID = "documents.drafts"
	OpDocumentsGroupMemberships OperationID = "documents.group-memberships"
	OpDocumentsInsights         OperationID = "documents.insights"
	OpDocumentsMemberships      OperationID = "documents.memberships"
	OpDocumentsSearchTitles     OperationID = "documents.search-titles"
	OpDocumentsUsers            OperationID = "documents.users"
	OpDocumentsViewed           OperationID = "documents.viewed"
	OpCollectionsMemberships    OperationID = "collections.memberships"
	OpCollectionsGroupMembers   OperationID = "collections.group-memberships"
	OpCommentsGet               OperationID = "comments.get"
	OpCommentsList              OperationID = "comments.list"
	OpRevisionsExport           OperationID = "revisions.export"
	OpRevisionsGet              OperationID = "revisions.get"
	OpRevisionsList             OperationID = "revisions.list"
	OpSharesGet                 OperationID = "shares.get"
	OpSharesList                OperationID = "shares.list"
	OpUsersGet                  OperationID = "users.get"
	OpUsersList                 OperationID = "users.list"
	OpGroupsGet                 OperationID = "groups.get"
	OpGroupsList                OperationID = "groups.list"
	OpGroupsMembers             OperationID = "groups.members"
	OpTemplatesGet              OperationID = "templates.get"
	OpTemplatesList             OperationID = "templates.list"
	OpPinsGet                   OperationID = "pins.get"
	OpPinsList                  OperationID = "pins.list"
	OpStarsList                 OperationID = "stars.list"
	OpSubscriptionsGet          OperationID = "subscriptions.get"
	OpSubscriptionsList         OperationID = "subscriptions.list"

	OpAccessRequestsGet        OperationID = "access-requests.get"
	OpAttachmentsList          OperationID = "attachments.list"
	OpEmojisGet                OperationID = "emojis.get"
	OpEmojisList               OperationID = "emojis.list"
	OpEventsList               OperationID = "events.list"
	OpFileOperationsGet        OperationID = "file-operations.get"
	OpFileOperationsList       OperationID = "file-operations.list"
	OpGroupMembershipsList     OperationID = "group-memberships.list"
	OpInstallationGet          OperationID = "installation.get"
	OpNotificationsList        OperationID = "notifications.list"
	OpOAuthAuthenticationsList OperationID = "oauth-authentications.list"
	OpReactionsList            OperationID = "reactions.list"
	OpRelationshipsGet         OperationID = "relationships.get"
	OpRelationshipsList        OperationID = "relationships.list"
	OpSearchesList             OperationID = "searches.list"
	OpSuggestionsMention       OperationID = "suggestions.mention"
	OpUserMembershipsList      OperationID = "user-memberships.list"
	OpViewsList                OperationID = "views.list"

	// The reads of stage a: wiki settings and integration data, read like any other read.
	OpAPIKeysList                 OperationID = "api-keys.list"
	OpAuthenticationProvidersGet  OperationID = "authentication-providers.get"
	OpAuthenticationProvidersList OperationID = "authentication-providers.list"
	OpImportsGet                  OperationID = "imports.get"
	OpImportsList                 OperationID = "imports.list"
	OpIntegrationsGet             OperationID = "integrations.get"
	OpIntegrationsList            OperationID = "integrations.list"
	OpOAuthClientsGet             OperationID = "oauth-clients.get"
	OpOAuthClientsList            OperationID = "oauth-clients.list"
	OpWebhookSubscriptionsList    OperationID = "webhook-subscriptions.list"
)

// The write operations of stage b, the documents writes. Each is a non-destructive change. A new write is a new constant and one catalog row.
const (
	OpDocumentsCreate     OperationID = "documents.create"
	OpDocumentsDuplicate  OperationID = "documents.duplicate"
	OpDocumentsMove       OperationID = "documents.move"
	OpDocumentsRestore    OperationID = "documents.restore"
	OpDocumentsTemplatize OperationID = "documents.templatize"
	OpDocumentsUnpublish  OperationID = "documents.unpublish"
	OpDocumentsUpdate     OperationID = "documents.update"

	OpCollectionsCreate    OperationID = "collections.create"
	OpCollectionsDuplicate OperationID = "collections.duplicate"
	OpCollectionsExport    OperationID = "collections.export"
	OpCollectionsMove      OperationID = "collections.move"
	OpCollectionsRestore   OperationID = "collections.restore"
	OpCollectionsUpdate    OperationID = "collections.update"

	OpCommentsCreate         OperationID = "comments.create"
	OpCommentsUpdate         OperationID = "comments.update"
	OpCommentsResolve        OperationID = "comments.resolve"
	OpCommentsUnresolve      OperationID = "comments.unresolve"
	OpCommentsAddReaction    OperationID = "comments.add-reaction"
	OpCommentsRemoveReaction OperationID = "comments.remove-reaction"

	OpAccessRequestsCreate OperationID = "access-requests.create"
	OpRevisionsUpdate      OperationID = "revisions.update"

	OpNotificationsUpdate    OperationID = "notifications.update"
	OpNotificationsUpdateAll OperationID = "notifications.update-all"

	OpPinsCreate OperationID = "pins.create"
	OpPinsDelete OperationID = "pins.delete"
	OpPinsUpdate OperationID = "pins.update"

	OpSearchesDelete OperationID = "searches.delete"
	OpSearchesUpdate OperationID = "searches.update"

	OpStarsCreate OperationID = "stars.create"
	OpStarsDelete OperationID = "stars.delete"
	OpStarsUpdate OperationID = "stars.update"

	OpSubscriptionsCreate OperationID = "subscriptions.create"
	OpSubscriptionsDelete OperationID = "subscriptions.delete"

	OpTemplatesCreate    OperationID = "templates.create"
	OpTemplatesDuplicate OperationID = "templates.duplicate"
	OpTemplatesRestore   OperationID = "templates.restore"
	OpTemplatesUpdate    OperationID = "templates.update"

	OpUserMembershipsUpdate OperationID = "user-memberships.update"

	OpUsersNotificationsSubscribe   OperationID = "users.notifications-subscribe"
	OpUsersNotificationsUnsubscribe OperationID = "users.notifications-unsubscribe"
)

// The destructive and admin operations of stage c. Each one changes the wiki;
// a destructive one removes content.
const (
	OpDocumentsAddGroup    OperationID = "documents.add-group"
	OpDocumentsAddUser     OperationID = "documents.add-user"
	OpDocumentsArchive     OperationID = "documents.archive"
	OpDocumentsDelete      OperationID = "documents.delete"
	OpDocumentsEmptyTrash  OperationID = "documents.empty-trash"
	OpDocumentsRemoveGroup OperationID = "documents.remove-group"
	OpDocumentsRemoveUser  OperationID = "documents.remove-user"

	OpCollectionsAddGroup    OperationID = "collections.add-group"
	OpCollectionsAddUser     OperationID = "collections.add-user"
	OpCollectionsArchive     OperationID = "collections.archive"
	OpCollectionsDelete      OperationID = "collections.delete"
	OpCollectionsExportAll   OperationID = "collections.export-all"
	OpCollectionsRemoveGroup OperationID = "collections.remove-group"
	OpCollectionsRemoveUser  OperationID = "collections.remove-user"
)

// The destructive and admin operations of stage c, slice c2: users, groups, and teams.
const (
	OpUsersActivate      OperationID = "users.activate"
	OpUsersDelete        OperationID = "users.delete"
	OpUsersInvite        OperationID = "users.invite"
	OpUsersRequestDelete OperationID = "users.request-delete"
	OpUsersResendInvite  OperationID = "users.resend-invite"
	OpUsersSuspend       OperationID = "users.suspend"
	OpUsersUpdate        OperationID = "users.update"
	OpUsersUpdateRole    OperationID = "users.update-role"

	OpGroupsAddUser    OperationID = "groups.add-user"
	OpGroupsCreate     OperationID = "groups.create"
	OpGroupsDelete     OperationID = "groups.delete"
	OpGroupsDeleteAll  OperationID = "groups.delete-all"
	OpGroupsRemoveUser OperationID = "groups.remove-user"
	OpGroupsUpdate     OperationID = "groups.update"
	OpGroupsUpdateUser OperationID = "groups.update-user"

	OpTeamsDelete        OperationID = "teams.delete"
	OpTeamsRequestDelete OperationID = "teams.request-delete"
	OpTeamsUpdate        OperationID = "teams.update"
)

// The destructive and admin operations of stage c, slice c3: access requests, attachments, sign-in providers, comments,
// emojis, file operations, imports, and integrations.
const (
	OpAccessRequestsApprove                 OperationID = "access-requests.approve"
	OpAccessRequestsDismiss                 OperationID = "access-requests.dismiss"
	OpAttachmentsDelete                     OperationID = "attachments.delete"
	OpAuthenticationProvidersDelete         OperationID = "authentication-providers.delete"
	OpAuthenticationProvidersStartGroupSync OperationID = "authentication-providers.start-group-sync"
	OpAuthenticationProvidersUpdate         OperationID = "authentication-providers.update"
	OpCommentsDelete                        OperationID = "comments.delete"
	OpEmojisDelete                          OperationID = "emojis.delete"
	OpFileOperationsDelete                  OperationID = "file-operations.delete"
	OpImportsCancel                         OperationID = "imports.cancel"
	OpImportsDelete                         OperationID = "imports.delete"
	OpIntegrationsCreate                    OperationID = "integrations.create"
	OpIntegrationsDelete                    OperationID = "integrations.delete"
	OpIntegrationsUpdate                    OperationID = "integrations.update"
)

// The Kiwi TCMS operations: three reads and three writes. Each names one row of rows_kiwi.go. Kiwi has
// no admin and no destructive operation.
const (
	OpKiwiProductsList    OperationID = "kiwi.products.list"
	OpKiwiProductsTree    OperationID = "kiwi.products.tree"
	OpKiwiDocumentsGet    OperationID = "kiwi.documents.get"
	OpKiwiTestCasesCreate OperationID = "kiwi.test-cases.create"
	OpKiwiTestCasesUpdate OperationID = "kiwi.test-cases.update"
	OpKiwiPlansAddCase    OperationID = "kiwi.plans.add-case"
)

// The Jira Cloud operations: five reads, and no write, admin, destructive, or MCP operation. Each names one
// row of rows_jira.go.
const (
	OpJiraWhoami       OperationID = "jira.whoami"
	OpJiraProjectsList OperationID = "jira.projects.list"
	OpJiraIssuesSearch OperationID = "jira.issues.search"
	OpJiraIssuesGet    OperationID = "jira.issues.get"
	OpJiraCommentsList OperationID = "jira.comments.list"
)

// The destructive and admin operations of stage c, slice c4: OAuth authorizations, OAuth clients, revisions, shares,
// templates, and webhook subscriptions.
const (
	OpOAuthAuthenticationsDelete OperationID = "oauth-authentications.delete"
	OpOAuthClientsCreate         OperationID = "oauth-clients.create"
	OpOAuthClientsDelete         OperationID = "oauth-clients.delete"
	OpOAuthClientsRotateSecret   OperationID = "oauth-clients.rotate-secret"
	OpOAuthClientsUpdate         OperationID = "oauth-clients.update"
	OpRevisionsDelete            OperationID = "revisions.delete"
	OpSharesCreate               OperationID = "shares.create"
	OpSharesRevoke               OperationID = "shares.revoke"
	OpSharesUpdate               OperationID = "shares.update"
	OpTemplatesDelete            OperationID = "templates.delete"
	OpWebhookSubscriptionsCreate OperationID = "webhook-subscriptions.create"
	OpWebhookSubscriptionsDelete OperationID = "webhook-subscriptions.delete"
	OpWebhookSubscriptionsUpdate OperationID = "webhook-subscriptions.update"
)

// Effect says what an operation does to the data it touches. Only read operations exist in
// stage a.
type Effect string

// The four effects, from the least to the most changing.
const (
	EffectRead        Effect = "read"
	EffectAdditive    Effect = "additive"
	EffectUpdate      Effect = "update"
	EffectDestructive Effect = "destructive"
)

// OperationUnavailableError reports that the server answered 404 "Endpoint not found" for one wire method. Its text
// names the method and the version hint, and it never carries the reply body.
type OperationUnavailableError struct {
	Method string
}

// Error returns the public text for the missing method.
func (e OperationUnavailableError) Error() string {
	return "this Outline server does not offer " + e.Method + "; its version may be older than v1.12.0"
}

// Is makes the error match ErrOperationUnavailable.
func (e OperationUnavailableError) Is(target error) bool {
	return target == ErrOperationUnavailable
}
