package outlinestub

import (
	"encoding/json"
	"fmt"
	"sort"
)

// The router key table. For each Outline method Retriever reads, it lists every key the pinned server (v1.12.0) may send in
// a reply: the envelope keys, and the keys of the presenters the handler uses for its objects, nested objects included.
// It is taken from the presenters and handlers of that version (src/server/presenters and src/server/routes/api). A
// scenario that states a reply with SetReply is checked against it, so a key the router never sends (for example a flat
// createdById on a share, which the router sends as a createdBy object) fails the scenario, not the production read.
//
// The table is a list of keys, not of types. A key that a presenter sends only sometimes (an optional field, or a secret
// sent only to a manage-scoped token) is listed, because the router may send it. The check is per method, not per
// object: a key listed for any presenter of a method is allowed on every object of that method.

// editorKeys are the keys of editor JSON. A document, a comment, and a template carry their editor JSON in data, and those
// nodes use these keys.
var editorKeys = []string{"type", "content", "text", "attrs", "marks"}

// envelopeKeys are the keys of the reply envelope, and of the wrappers a list method answers with (data.memberships,
// data.pins, data.documents, data.relationship, and so on). The editor keys are here too, for the objects that carry them.
var envelopeKeys = append([]string{
	"ok", "status", "data", "policies", "abilities", "pagination", "offset", "limit", "total", "unseen",
	"memberships", "groupMemberships", "users", "documents", "groups", "pins", "stars", "relationship", "relationships",
	"notifications", "shares", "sharedTree", "team", "collection", "document", "nodes", "count",
}, editorKeys...)

// verifiedEnvelopes are the top-level keys of the writes whose reply shape the v1.12.0 source fixes, one entry per built
// write, destructive, and admin row (the stage b writes and the stage c admin rows of slices c1 and c2). An acknowledgement answers success
// true and nothing else of its own, and an object reply answers data, with policies where its handler adds them. For these
// methods the envelope is this list, not the shared one, so the ok and the status of a shared envelope are refused: the
// wiki never sends a write as {ok: true}. A write that has no entry keeps the shared envelope, which admits them, so a
// new write row adds its entry here before its fixture is built.
var verifiedEnvelopes = map[string][]string{
	// Acknowledgements: success true, with the data member a value needs.
	"documents.delete":         {"success"},
	"documents.empty_trash":    {"success"},
	"documents.remove_group":   {"success"},
	"documents.remove_user":    {"success"},
	"collections.delete":       {"success"},
	"collections.remove_group": {"success"},
	"collections.remove_user":  {"success"},
	"comments.add_reaction":    {"success"},
	"comments.remove_reaction": {"success"},
	"pins.delete":              {"success"},
	"searches.delete":          {"success"},
	"stars.delete":             {"success"},
	"subscriptions.delete":     {"success"},
	"collections.move":         {"success", "data"},
	"collections.export":       {"success", "data"},
	"collections.export_all":   {"success", "data"},
	"notifications.update_all": {"success", "data"},
	// Object replies whose handler adds policies: the presentation under data, beside the policies.
	"documents.create":       {"data", "policies"},
	"documents.duplicate":    {"data", "policies"},
	"documents.move":         {"data", "policies"},
	"documents.restore":      {"data", "policies"},
	"documents.templatize":   {"data", "policies"},
	"documents.unpublish":    {"data", "policies"},
	"documents.update":       {"data", "policies"},
	"documents.archive":      {"data", "policies"},
	"collections.create":     {"data", "policies"},
	"collections.duplicate":  {"data", "policies"},
	"collections.restore":    {"data", "policies"},
	"collections.update":     {"data", "policies"},
	"collections.archive":    {"data", "policies"},
	"comments.create":        {"data", "policies"},
	"comments.update":        {"data", "policies"},
	"comments.resolve":       {"data", "policies"},
	"comments.unresolve":     {"data", "policies"},
	"accessRequests.create":  {"data", "policies"},
	"revisions.update":       {"data", "policies"},
	"notifications.update":   {"data", "policies"},
	"pins.create":            {"data", "policies"},
	"pins.update":            {"data", "policies"},
	"stars.create":           {"data", "policies"},
	"stars.update":           {"data", "policies"},
	"templates.create":       {"data", "policies"},
	"templates.duplicate":    {"data", "policies"},
	"templates.restore":      {"data", "policies"},
	"templates.update":       {"data", "policies"},
	"userMemberships.update": {"data", "policies"},
	// Object replies without policies: the presentation under data alone, and the membership and group lists.
	"searches.update":                {"data"},
	"subscriptions.create":           {"data"},
	"users.notificationsSubscribe":   {"data"},
	"users.notificationsUnsubscribe": {"data"},
	"documents.add_user":             {"data"},
	"documents.add_group":            {"data"},
	"collections.add_user":           {"data"},
	"collections.add_group":          {"data"},
	// Stage c2: users, groups, and teams. An acknowledgement answers success true. groups.deleteAll is not in the wire
	// facts, so its acknowledgement shape is an assumption, recorded in the report.
	"users.delete":        {"success"},
	"users.requestDelete": {"success"},
	"users.resendInvite":  {"success"},
	"groups.delete":       {"success"},
	"groups.deleteAll":    {"success"},
	"teams.delete":        {"success"},
	"teams.requestDelete": {"success"},
	"users.activate":      {"data", "policies"},
	"users.suspend":       {"data", "policies"},
	"users.update_role":   {"data", "policies"},
	"groups.create":       {"data", "policies"},
	"groups.update":       {"data", "policies"},
	"teams.update":        {"data", "policies"},
	"users.update":        {"data"},
	"users.invite":        {"data"},
	"groups.add_user":     {"data"},
	"groups.update_user":  {"data"},
	"groups.remove_user":  {"data"},
	// Stage c3: access requests, attachments, sign-in providers, comments, emojis, file operations, imports, and integrations.
	// An acknowledgement answers success true. An object reply answers data with policies. startGroupSync answers data alone,
	// with the redirect address under it (methodEnvelopeKeys).
	"accessRequests.approve":                 {"data", "policies"},
	"accessRequests.dismiss":                 {"data", "policies"},
	"attachments.delete":                     {"success"},
	"authenticationProviders.delete":         {"success"},
	"authenticationProviders.startGroupSync": {"data"},
	"authenticationProviders.update":         {"data", "policies"},
	"comments.delete":                        {"success"},
	"emojis.delete":                          {"success"},
	"fileOperations.delete":                  {"success"},
	"imports.cancel":                         {"data", "policies"},
	"imports.delete":                         {"success"},
	"integrations.create":                    {"data", "policies"},
	"integrations.delete":                    {"success"},
	"integrations.update":                    {"data", "policies"},
	// Stage c4: OAuth authorizations and clients, revisions, shares, templates, and webhook subscriptions. An acknowledgement
	// answers success true. An OAuth client's reply carries its secret under data (includeSecret), and a webhook's reply carries
	// its signing secret; both are router keys, and Retriever must drop them. A webhook answers data alone, with no policies.
	"oauthAuthentications.delete": {"success"},
	"oauthClients.create":         {"data", "policies"},
	"oauthClients.delete":         {"success"},
	"oauthClients.rotate_secret":  {"data", "policies"},
	"oauthClients.update":         {"data", "policies"},
	"revisions.delete":            {"success"},
	"shares.create":               {"data", "policies"},
	"shares.revoke":               {"success"},
	"shares.update":               {"data", "policies"},
	"templates.delete":            {"success"},
	"webhookSubscriptions.create": {"data"},
	"webhookSubscriptions.delete": {"success"},
	"webhookSubscriptions.update": {"data"},
}

// editorPresenters are the presenters whose object carries editor JSON in its data member, so a verified reply that holds one
// of them also allows the editor keys.
var editorPresenters = map[string]bool{"document": true, "comment": true, "template": true, "revision": true}

// opaqueKeys hold JSON the router does not shape key by key. checkKeys checks the key that holds them, not the keys
// inside them. The provider settings (authenticationProvider.settings, integration.settings) are the configuration the
// admin saved, sent as it was stored.
var opaqueKeys = map[string]bool{
	"settings": true,
	// notificationSettings maps each event type of the server to a flag, so its keys are the event types, not a fixed list.
	"notificationSettings": true,
}

// presenterKeys are the keys each presenter sends, by presenter name.
var presenterKeys = map[string][]string{
	"user": {"id", "name", "avatarUrl", "color", "role", "isSuspended", "createdAt", "updatedAt", "deletedAt",
		"lastActiveAt", "timezone", "email", "language", "preferences", "notificationSettings", "invitedBy"},
	"team": {"id", "name", "description", "avatarUrl", "sharing", "memberCollectionCreate", "memberTeamCreate",
		"defaultCollectionId", "documentEmbeds", "guestSignin", "passkeysEnabled", "subdomain", "domain", "url",
		"defaultUserRole", "inviteRequired", "allowedDomains", "preferences", "guidanceMCP", "plan", "entitlements"},
	"group": {"id", "name", "description", "icon", "color", "externalId", "memberCount", "disableMentions",
		"externalGroup", "createdAt", "updatedAt"},
	"externalGroup": {"id", "externalId", "name", "groupId", "provider", "displayName", "authenticationProviderId",
		"lastSyncedAt", "createdAt", "updatedAt"},
	"groupMembership": {"id", "groupId", "documentId", "collectionId", "permission", "sourceId"},
	"groupUser":       {"id", "userId", "groupId", "permission", "createdById", "createdBy", "createdAt", "user"},
	"membership":      {"id", "userId", "documentId", "collectionId", "permission", "createdById", "sourceId", "index"},
	"navigationNode":  {"id", "title", "url", "children"},
	"document": {"id", "url", "urlId", "title", "data", "icon", "color", "tasks", "language", "createdAt", "createdBy",
		"updatedAt", "updatedBy", "publishedAt", "archivedAt", "deprecatedReason", "deletedAt", "deletedBy",
		"collaboratorIds", "revision", "fullWidth", "preferences", "collectionId", "parentDocumentId", "lastViewedAt",
		"isCollectionDeleted", "backlinkIds", "isPublic"},
	"collection": {"id", "url", "urlId", "name", "data", "description", "sort", "icon", "color", "createdAt",
		"updatedAt", "archivedBy", "index", "sharing", "commenting", "templateManagement", "permission", "ownerId",
		"deletedAt", "archivedAt", "deprecatedReason", "sourceMetadata", "externalId", "externalName", "createdByName",
		// sort is the stored order of the collection: a field and a direction.
		"field", "direction"},
	"comment": {"id", "data", "documentId", "parentCommentId", "createdBy", "createdById", "resolvedAt", "resolvedBy",
		"resolvedById", "createdAt", "updatedAt", "reactions", "anchorText"},
	"revision": {"id", "documentId", "title", "name", "data", "text", "icon", "color", "collaborators",
		"sourceMetadata", "authType", "createdAt", "createdBy", "createdById", "deletedAt"},
	"share": {"id", "sourceTitle", "sourcePath", "collectionId", "documentId", "documentTitle", "documentUrl",
		"published", "url", "urlId", "createdBy", "includeChildDocuments", "allowIndexing", "allowSubscriptions",
		"showLastUpdated", "showTOC", "title", "iconUrl", "lastAccessedAt", "views", "domain", "createdAt", "updatedAt"},
	"template": {"id", "url", "urlId", "title", "data", "icon", "color", "createdAt", "createdBy", "updatedAt",
		"updatedBy", "deletedAt", "publishedAt", "fullWidth", "collectionId"},
	"pin":          {"id", "documentId", "collectionId", "index", "createdAt", "updatedAt"},
	"star":         {"id", "documentId", "collectionId", "index", "createdAt", "updatedAt"},
	"subscription": {"id", "userId", "documentId", "collectionId", "event", "createdAt", "updatedAt"},
	"searchQuery":  {"id", "query", "source", "createdAt", "answer", "score"},
	"searchHit":    {"context", "ranking", "document"},
	"documentInsight": {"date", "period", "viewCount", "viewerCount", "commentCount", "reactionCount",
		"revisionCount", "editorCount"},
	"view": {"id", "documentId", "count", "firstViewedAt", "lastViewedAt", "userId", "user"},
	// value is the secret of the key. The router sends it on apiKeys.list, and Retriever must not print it.
	"apiKey": {"id", "user", "userId", "name", "scope", "value", "last4", "createdAt", "updatedAt", "expiresAt",
		"lastActiveAt"},
	// settings is opaque (see opaqueKeys). The provider has no secret key of its own.
	"authenticationProvider": {"id", "name", "providerId", "createdAt", "isEnabled", "isConnected", "settings"},
	"import": {"id", "name", "service", "state", "documentCount", "error", "createdBy", "createdById", "createdAt",
		"updatedAt"},
	"integration": {"id", "type", "userId", "collectionId", "authenticationId", "service", "events", "settings",
		"url", "channel", "measurementId", "instanceUrl", "scriptName", "createdAt", "updatedAt"},
	// clientSecret is sent only to a token with the manage scope (presentOAuthClient, includeSecret).
	"oauthClient": {"id", "name", "description", "developerName", "developerUrl", "avatarUrl", "clientId",
		"clientSecret", "clientType", "redirectUris", "published", "lastActiveAt", "createdAt", "updatedAt"},
	// The client inside an authorization is the published view (presentPublishedOAuthClient), which has no secret.
	"oauthAuthentication": {"id", "userId", "oauthClientId", "oauthClient", "scope", "lastActiveAt", "createdAt",
		"name", "description", "developerName", "developerUrl", "avatarUrl", "clientId", "published"},
	"emoji":      {"id", "name", "teamId", "url", "createdBy", "createdById", "createdAt", "updatedAt"},
	"attachment": {"userId", "documentId", "contentType", "name", "id", "url", "size"},
	"event": {"id", "name", "modelId", "userId", "actorId", "authType", "actorIpAddress", "collectionId",
		"documentId", "createdAt", "data", "changes", "actor"},
	"fileOperation": {"id", "type", "format", "name", "state", "error", "size", "collectionId", "documentId", "user",
		"createdAt", "updatedAt"},
	"accessRequest": {"id", "documentId", "userId", "user", "teamId", "status", "responderId", "responder",
		"respondedAt", "createdAt", "updatedAt"},
	"notification": {"id", "viewedAt", "accessRequestId", "accessRequestStatus", "archivedAt", "createdAt", "event",
		"userId", "actorId", "actor", "commentId", "comment", "documentId", "document", "revisionId", "collectionId",
		"groupId", "group", "data"},
	"reaction":     {"id", "emoji", "commentId", "user", "userId", "createdAt", "updatedAt"},
	"relationship": {"id", "type", "documentId", "reverseDocumentId", "userId", "createdAt", "updatedAt"},
	"installation": {"version", "latestVersion", "versionsBehind"},
	"suggestions":  {"documents", "users", "groups"},
	// secret is the webhook signing secret. The presenter sends it on every reply, so the router sends it, and Retriever must
	// drop it (decode_2b_specs.go). createdBy is the creator as a user object, present only when the webhook has one.
	"webhook": {"id", "name", "url", "secret", "events", "enabled", "createdBy", "createdById", "createdAt", "updatedAt"},
}

// nestedPresenters names the presenters an object's own presenter calls for its nested objects (a document's createdBy
// is a user, a share's collection is a collection). RouterKeys adds them, so a method that sends a document also
// allows the keys of the users the document names.
var nestedPresenters = map[string][]string{
	"document":   {"user"},
	"collection": {"user"},
	"template":   {"user"},
	"import":     {"user"},
	"emoji":      {"user"},
	"comment":    {"user"},
	"revision":   {"user"},
	"share":      {"user"},
}

// methodPresenters names the presenters of the objects each method answers with.
var methodPresenters = map[string][]string{
	"auth.info":                      {"user", "team"},
	"collections.list":               {"collection"},
	"collections.info":               {"collection"},
	"collections.documents":          {"navigationNode"},
	"collections.memberships":        {"membership", "user"},
	"collections.group_memberships":  {"groupMembership", "group"},
	"documents.info":                 {"document"},
	"documents.export":               {},
	"documents.list":                 {"document"},
	"documents.search":               {"searchHit", "document"},
	"documents.archived":             {"document"},
	"documents.documents":            {"navigationNode"},
	"documents.deleted":              {"document"},
	"documents.drafts":               {"document"},
	"documents.group_memberships":    {"groupMembership", "group"},
	"documents.insights":             {"documentInsight"},
	"documents.memberships":          {"membership", "user"},
	"documents.search_titles":        {"document"},
	"documents.users":                {"user"},
	"documents.viewed":               {"view", "user", "document"},
	"comments.info":                  {"comment", "user", "reaction"},
	"comments.list":                  {"comment", "user", "reaction"},
	"revisions.export":               {},
	"revisions.info":                 {"revision", "user"},
	"revisions.list":                 {"revision", "user"},
	"shares.info":                    {"share", "user", "collection", "document"},
	"shares.list":                    {"share", "user"},
	"users.info":                     {"user"},
	"users.list":                     {"user"},
	"groups.info":                    {"group", "externalGroup"},
	"groups.list":                    {"group", "externalGroup"},
	"groups.memberships":             {"groupUser", "user"},
	"templates.info":                 {"template", "user"},
	"documents.create":               {"document"},
	"documents.duplicate":            {"document"},
	"documents.move":                 {"document"},
	"documents.restore":              {"document"},
	"documents.templatize":           {"template", "user"},
	"documents.unpublish":            {"document"},
	"documents.update":               {"document"},
	"templates.list":                 {"template", "user"},
	"pins.info":                      {"pin"},
	"pins.list":                      {"pin", "document"},
	"stars.list":                     {"star", "document"},
	"subscriptions.info":             {"subscription"},
	"subscriptions.list":             {"subscription"},
	"accessRequests.info":            {"accessRequest", "user"},
	"attachments.list":               {"attachment"},
	"emojis.info":                    {"emoji"},
	"emojis.list":                    {"emoji"},
	"events.list":                    {"event", "user"},
	"fileOperations.info":            {"fileOperation", "user"},
	"fileOperations.list":            {"fileOperation", "user"},
	"groupMemberships.list":          {"groupMembership", "group", "document"},
	"installation.info":              {"installation"},
	"notifications.list":             {"notification", "document", "comment", "user", "group"},
	"oauthAuthentications.list":      {"oauthAuthentication"},
	"reactions.list":                 {"reaction", "user"},
	"relationships.info":             {"relationship", "document"},
	"relationships.list":             {"relationship", "document"},
	"searches.list":                  {"searchQuery"},
	"suggestions.mention":            {"suggestions", "document", "user", "group"},
	"userMemberships.list":           {"membership"},
	"views.list":                     {"view", "user"},
	"apiKeys.list":                   {"apiKey", "user"},
	"authenticationProviders.info":   {"authenticationProvider"},
	"authenticationProviders.list":   {"authenticationProvider"},
	"imports.info":                   {"import", "user"},
	"imports.list":                   {"import", "user"},
	"integrations.info":              {"integration"},
	"integrations.list":              {"integration"},
	"oauthClients.info":              {"oauthClient"},
	"oauthClients.list":              {"oauthClient"},
	"webhookSubscriptions.list":      {"webhook", "user"},
	"collections.create":             {"collection"},
	"collections.duplicate":          {"collection"},
	"collections.export":             {"fileOperation"},
	"collections.move":               {},
	"collections.restore":            {"collection"},
	"collections.update":             {"collection"},
	"comments.create":                {"comment", "user", "reaction"},
	"comments.update":                {"comment", "user", "reaction"},
	"comments.resolve":               {"comment", "user", "reaction"},
	"comments.unresolve":             {"comment", "user", "reaction"},
	"comments.add_reaction":          {},
	"comments.remove_reaction":       {},
	"accessRequests.create":          {"accessRequest", "user"},
	"revisions.update":               {"revision", "user"},
	"notifications.update":           {"notification", "document", "comment", "user", "group"},
	"notifications.update_all":       {},
	"pins.create":                    {"pin"},
	"pins.delete":                    {},
	"pins.update":                    {"pin"},
	"searches.delete":                {},
	"searches.update":                {"searchQuery"},
	"stars.create":                   {"star"},
	"stars.delete":                   {},
	"stars.update":                   {"star"},
	"subscriptions.create":           {"subscription"},
	"subscriptions.delete":           {},
	"templates.create":               {"template", "user"},
	"templates.duplicate":            {"template", "user"},
	"templates.restore":              {"template", "user"},
	"templates.update":               {"template", "user"},
	"userMemberships.update":         {"membership"},
	"users.notificationsSubscribe":   {"user"},
	"users.notificationsUnsubscribe": {"user"},
	"documents.add_group":            {"groupMembership", "group"},
	"documents.add_user":             {"membership", "user"},
	"documents.archive":              {"document"},
	"documents.delete":               {},
	"documents.empty_trash":          {},
	"documents.remove_group":         {},
	"documents.remove_user":          {},
	"collections.add_group":          {"groupMembership", "group"},
	"collections.add_user":           {"membership", "user"},
	"collections.archive":            {"collection"},
	"collections.delete":             {},
	"collections.export_all":         {"fileOperation"},
	"collections.remove_group":       {},
	"collections.remove_user":        {},
	"users.activate":                 {"user"},
	"users.delete":                   {},
	"users.invite":                   {"user"},
	"users.requestDelete":            {},
	"users.resendInvite":             {},
	"users.suspend":                  {"user"},
	"users.update":                   {"user"},
	"users.update_role":              {"user"},
	"groups.add_user":                {"groupUser", "user", "group"},
	"groups.create":                  {"group"},
	"groups.delete":                  {},
	"groups.deleteAll":               {},
	"groups.remove_user":             {"group"},
	"groups.update":                  {"group"},
	"groups.update_user":             {"groupUser", "user", "group"},
	"teams.delete":                   {},
	"teams.requestDelete":            {},
	"teams.update":                   {"team"},
	// Stage c3 writes (slice c3): the presenter of each object reply, and none for an acknowledgement.
	"accessRequests.approve":                 {"accessRequest", "user"},
	"accessRequests.dismiss":                 {"accessRequest", "user"},
	"attachments.delete":                     {},
	"authenticationProviders.delete":         {},
	"authenticationProviders.startGroupSync": {},
	"authenticationProviders.update":         {"authenticationProvider"},
	"comments.delete":                        {},
	"emojis.delete":                          {},
	"fileOperations.delete":                  {},
	"imports.cancel":                         {"import"},
	"imports.delete":                         {},
	"integrations.create":                    {"integration"},
	"integrations.delete":                    {},
	"integrations.update":                    {"integration"},
	// Stage c4 writes: the presenter of each object reply, and none for an acknowledgement.
	"oauthAuthentications.delete": {},
	"oauthClients.create":         {"oauthClient"},
	"oauthClients.delete":         {},
	"oauthClients.rotate_secret":  {"oauthClient"},
	"oauthClients.update":         {"oauthClient"},
	"revisions.delete":            {},
	"shares.create":               {"share", "user"},
	"shares.revoke":               {},
	"shares.update":               {"share", "user"},
	"templates.delete":            {},
	"webhookSubscriptions.create": {"webhook", "user"},
	"webhookSubscriptions.delete": {},
	"webhookSubscriptions.update": {"webhook", "user"},
}

// methodEnvelopeKeys are the keys of one method's reply that the shared envelope list does not hold. auth.info answers
// data.user and data.team: the presenters are named user and team, but those names are keys of the reply itself.
// collections.export answers data.fileOperation, collections.export_all the same under data beside success, and
// collections.move answers the index the wiki kept under data. The membership and group replies name their wrapper lists
// under data, which the verified envelopes do not hold. documents.duplicate and documents.move answer the copies or the
// moved documents as a list under data.documents, with the collections they name beside it.
var methodEnvelopeKeys = map[string][]string{
	"auth.info":                {"user", "team"},
	"collections.export":       {"fileOperation"},
	"collections.export_all":   {"fileOperation"},
	"collections.move":         {"index"},
	"notifications.update_all": {"total"},
	"documents.duplicate":      {"documents"},
	"documents.move":           {"documents", "collections"},
	"documents.add_user":       {"memberships", "users"},
	"documents.add_group":      {"groupMemberships"},
	"collections.add_user":     {"memberships", "users"},
	"collections.add_group":    {"groupMemberships"},
	"users.invite":             {"sent", "unsent", "users"},
	"groups.add_user":          {"users", "groupMemberships", "groups"},
	"groups.update_user":       {"users", "groupMemberships", "groups"},
	"groups.remove_user":       {"groups"},
	// The redirect address of startGroupSync is under data, and the router sends it there.
	"authenticationProviders.startGroupSync": {"url"},
}

// RouterKeys returns the keys the router may send in a reply of one method, and false when the table has no entry for
// the method. The envelope is the shared one, or the method's verified envelope when the table holds one, so an
// acknowledgement's ok or status is refused.
func RouterKeys(method string) (map[string]bool, bool) {
	presenters, ok := methodPresenters[method]
	if !ok {
		return nil, false
	}
	envelope, verified := verifiedEnvelopes[method]
	if !verified {
		envelope = envelopeKeys
	}
	keys := map[string]bool{}
	for _, k := range envelope {
		keys[k] = true
	}
	// methodEnvelopeKeys are the keys of one method's reply that the envelope list does not hold: auth.info answers
	// data.user and data.team, which the presenters name but which are keys of the reply itself.
	for _, k := range methodEnvelopeKeys[method] {
		keys[k] = true
	}
	for _, p := range presenters {
		addPresenter(keys, p, verified)
		for _, nested := range nestedPresenters[p] {
			addPresenter(keys, nested, verified)
		}
	}
	return keys, true
}

// addPresenter adds the keys one presenter sends. A verified envelope has no editor keys of its own, so an editor-bearing
// presenter adds them when the method is verified.
func addPresenter(keys map[string]bool, presenter string, verified bool) {
	for _, k := range presenterKeys[presenter] {
		keys[k] = true
	}
	if verified && editorPresenters[presenter] {
		for _, k := range editorKeys {
			keys[k] = true
		}
	}
}

// VerifiedEnvelope returns the top-level keys the router sends in the reply of a write whose envelope the table verifies,
// and false when the method has no verified envelope. A test of a synthetic fixture checks its top-level keys against it.
func VerifiedEnvelope(method string) ([]string, bool) {
	envelope, ok := verifiedEnvelopes[method]
	return append([]string(nil), envelope...), ok
}

// CheckReplyKeys reports the first key of a JSON reply that the router does not send for the method. A reply that is not
// JSON, or a method the table does not name, is an error too. It walks every object of the reply, nested ones included,
// except the opaque keys.
func CheckReplyKeys(method string, body []byte) error {
	allowed, ok := RouterKeys(method)
	if !ok {
		return fmt.Errorf("the router key table has no entry for %s", method)
	}
	var root any
	if err := json.Unmarshal(body, &root); err != nil {
		return fmt.Errorf("the reply for %s is not JSON: %w", method, err)
	}
	return checkKeys(method, root, allowed)
}

// checkKeys walks v and returns an error for the first object key that allowed does not hold.
func checkKeys(method string, v any, allowed map[string]bool) error {
	switch x := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if !allowed[k] {
				return fmt.Errorf("the reply for %s has the key %q, which the router does not send for that method", method, k)
			}
			if opaqueKeys[k] {
				continue
			}
			if err := checkKeys(method, x[k], allowed); err != nil {
				return err
			}
		}
	case []any:
		for _, item := range x {
			if err := checkKeys(method, item, allowed); err != nil {
				return err
			}
		}
	}
	return nil
}
