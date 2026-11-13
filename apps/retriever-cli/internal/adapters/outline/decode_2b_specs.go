package outline

import (
	"net/url"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// The object projections of the stage a part 2b reads. Each spec keeps only
// the fields of its row and drops the redacted ones: a key value and a secret, a signing secret, a client secret, the
// full address of a webhook or an integration (only its host stays), an e-mail address, a storage address, and an actor's
// IP address. The reply rules of decode.go apply: a missing or null optional field is the zero value, and a required ID
// that is invalid is an unexpected reply.

// hostOnly returns the host name of an address, without the path, the query, the port, or the user name. An address that
// does not parse gives "", so a broken address never reaches the answer.
func hostOnly(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return ""
	}
	return u.Hostname()
}

// stringsOf returns the list of text values at key, as a list that is never nil, so an absent list is an empty list.
// Items that are not text are left out.
func stringsOf(m map[string]any, key string) []any {
	items, _ := jsonList(m[key])
	out := make([]any, 0, len(items))
	for _, raw := range items {
		if s, ok := raw.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// requireObjectID reads the ID of an object that the server always names, and refuses a reply without one.
func requireObjectID(m map[string]any) (string, error) {
	return requiredID(m, "id", domain.ValidateUUID)
}

// projectUnder returns a projection that reads one object under key of the data object, with the spec. A relationship
// reply holds its object beside the documents it names, and this reads the object alone.
func projectUnder(key string, spec objectSpec) func(any) (domain.Record, error) {
	return func(root any) (domain.Record, error) {
		m, err := dataObject(root)
		if err != nil {
			return domain.Record{}, err
		}
		obj, ok := jsonObject(m[key])
		if !ok {
			return domain.Record{}, domain.ErrUnexpectedResponse
		}
		return spec(obj)
	}
}

// subObject returns the object at key, or an empty object when the key is missing or null. A nested object of a
// projection is optional, so a missing one is not an error.
func subObject(m map[string]any, key string) map[string]any {
	obj, _ := jsonObject(m[key])
	return obj
}

// apiKeySpec projects one API key. The key value, the old secret, the hash, and the user object are never read. The last
// four characters are kept, as the field reference allows.
func apiKeySpec(m map[string]any) (domain.Record, error) {
	id, err := requireObjectID(m)
	if err != nil {
		return domain.Record{}, err
	}
	return domain.NewRecord(
		field("id", id),
		field("userId", textOrNil(m, "userId")),
		field("name", textOf(m, "name")),
		field("scope", stringsOf(m, "scope")),
		field("last4", textOrNil(m, "last4")),
		field("createdAt", textOf(m, "createdAt")),
		field("updatedAt", textOf(m, "updatedAt")),
		field("expiresAt", textOrNil(m, "expiresAt")),
		field("lastActiveAt", textOrNil(m, "lastActiveAt")),
	), nil
}

// authProviderSpec projects one sign-in provider. Only the three group-sync settings are kept of its settings.
func authProviderSpec(m map[string]any) (domain.Record, error) {
	id, err := requireObjectID(m)
	if err != nil {
		return domain.Record{}, err
	}
	settings := subObject(m, "settings")
	return domain.NewRecord(
		field("id", id),
		field("name", textOf(m, "name")),
		field("providerId", textOf(m, "providerId")),
		field("isEnabled", boolOf(m, "isEnabled")),
		field("isConnected", boolOf(m, "isConnected")),
		field("createdAt", textOf(m, "createdAt")),
		field("settings", domain.NewRecord(
			field("groupSyncEnabled", boolOf(settings, "groupSyncEnabled")),
			field("groupClaim", textOrNil(settings, "groupClaim")),
			field("groupSyncScopes", stringsOf(settings, "groupSyncScopes")),
		)),
	), nil
}

// importSpec projects one import of documents. The creator object is not read.
func importSpec(m map[string]any) (domain.Record, error) {
	id, err := requireObjectID(m)
	if err != nil {
		return domain.Record{}, err
	}
	return domain.NewRecord(
		field("id", id),
		field("name", textOrNil(m, "name")),
		field("service", textOf(m, "service")),
		field("state", textOf(m, "state")),
		field("documentCount", intOf(m, "documentCount")),
		field("error", textOrNil(m, "error")),
		field("createdById", textOrNil(m, "createdById")),
		field("createdAt", textOf(m, "createdAt")),
		field("updatedAt", textOf(m, "updatedAt")),
	), nil
}

// integrationSpec projects one integration. Its settings keep the host of each address and nothing else of an address:
// the full URL of a chat or embed integration carries a secret in its path.
func integrationSpec(m map[string]any) (domain.Record, error) {
	id, err := requireObjectID(m)
	if err != nil {
		return domain.Record{}, err
	}
	settings := subObject(m, "settings")
	return domain.NewRecord(
		field("id", id),
		field("type", textOf(m, "type")),
		field("service", textOrNil(m, "service")),
		field("events", stringsOf(m, "events")),
		field("collectionId", textOrNil(m, "collectionId")),
		field("userId", textOrNil(m, "userId")),
		field("createdAt", textOf(m, "createdAt")),
		field("updatedAt", textOf(m, "updatedAt")),
		field("settings", domain.NewRecord(
			field("urlHost", hostOnly(textOf(settings, "url"))),
			field("channel", textOrNil(settings, "channel")),
			field("measurementId", textOrNil(settings, "measurementId")),
			field("instanceUrl", hostOnly(textOf(settings, "instanceUrl"))),
			field("scriptName", textOrNil(settings, "scriptName")),
		)),
	), nil
}

// oauthClientSpec projects one OAuth client. Its client secret and its registration token are never read.
func oauthClientSpec(m map[string]any) (domain.Record, error) {
	id, err := requireObjectID(m)
	if err != nil {
		return domain.Record{}, err
	}
	return domain.NewRecord(
		field("id", id),
		field("name", textOf(m, "name")),
		field("description", textOrNil(m, "description")),
		field("developerName", textOrNil(m, "developerName")),
		field("developerUrl", textOrNil(m, "developerUrl")),
		field("clientId", textOf(m, "clientId")),
		field("clientType", textOf(m, "clientType")),
		field("redirectUris", stringsOf(m, "redirectUris")),
		field("published", boolOf(m, "published")),
		field("lastActiveAt", textOrNil(m, "lastActiveAt")),
		field("createdAt", textOf(m, "createdAt")),
		field("updatedAt", textOf(m, "updatedAt")),
	), nil
}

// creatorID returns the ID of the creator of an object. The flat createdById is read first, and when the reply holds only
// the creator as the createdBy object, the ID of that object is read.
func creatorID(m map[string]any) any {
	if id := textOrNil(m, "createdById"); id != nil {
		return id
	}
	return textOrNil(subObject(m, "createdBy"), "id")
}

// webhookSpec projects one webhook subscription. Its URL is cut to the host, and its signing secret is never read.
func webhookSpec(m map[string]any) (domain.Record, error) {
	id, err := requireObjectID(m)
	if err != nil {
		return domain.Record{}, err
	}
	return domain.NewRecord(
		field("id", id),
		field("name", textOf(m, "name")),
		field("urlHost", hostOnly(textOf(m, "url"))),
		field("events", stringsOf(m, "events")),
		field("enabled", boolOf(m, "enabled")),
		field("createdById", creatorID(m)),
		field("createdAt", textOf(m, "createdAt")),
		field("updatedAt", textOf(m, "updatedAt")),
	), nil
}

// emojiSpec projects one custom emoji. Its storage address and team are never read.
func emojiSpec(m map[string]any) (domain.Record, error) {
	id, err := requireObjectID(m)
	if err != nil {
		return domain.Record{}, err
	}
	return domain.NewRecord(
		field("id", id),
		field("name", textOf(m, "name")),
		field("createdById", textOrNil(m, "createdById")),
		field("createdAt", textOf(m, "createdAt")),
		field("updatedAt", textOf(m, "updatedAt")),
	), nil
}

// attachmentSpec projects one attachment. Its storage address is never read.
func attachmentSpec(m map[string]any) (domain.Record, error) {
	id, err := requireObjectID(m)
	if err != nil {
		return domain.Record{}, err
	}
	return domain.NewRecord(
		field("id", id),
		field("name", textOf(m, "name")),
		field("contentType", textOrNil(m, "contentType")),
		field("size", intOf(m, "size")),
		field("documentId", textOrNil(m, "documentId")),
		field("userId", textOrNil(m, "userId")),
	), nil
}

// eventSpec projects one event. The actor's IP address, the change data, and the actor object are never read.
func eventSpec(m map[string]any) (domain.Record, error) {
	id, err := requireObjectID(m)
	if err != nil {
		return domain.Record{}, err
	}
	return domain.NewRecord(
		field("id", id),
		field("name", textOf(m, "name")),
		field("modelId", textOrNil(m, "modelId")),
		field("userId", textOrNil(m, "userId")),
		field("actorId", textOrNil(m, "actorId")),
		field("collectionId", textOrNil(m, "collectionId")),
		field("documentId", textOrNil(m, "documentId")),
		field("authType", textOrNil(m, "authType")),
		field("createdAt", textOf(m, "createdAt")),
	), nil
}

// fileOperationSpec projects one import or export file operation. Its reply has no download link, and none is read.
func fileOperationSpec(m map[string]any) (domain.Record, error) {
	id, err := requireObjectID(m)
	if err != nil {
		return domain.Record{}, err
	}
	return domain.NewRecord(
		field("id", id),
		field("type", textOf(m, "type")),
		field("format", textOrNil(m, "format")),
		field("name", textOrNil(m, "name")),
		field("state", textOf(m, "state")),
		field("error", textOrNil(m, "error")),
		field("size", intOf(m, "size")),
		field("collectionId", textOrNil(m, "collectionId")),
		field("documentId", textOrNil(m, "documentId")),
		// The router names the user by its user object (presenters/fileOperation.ts), not by a flat userId.
		field("userId", textOrNil(subObject(m, "user"), "id")),
		field("createdAt", textOf(m, "createdAt")),
		field("updatedAt", textOf(m, "updatedAt")),
	), nil
}

// accessRequestSpec projects one access request. The nested user and responder objects are never read.
func accessRequestSpec(m map[string]any) (domain.Record, error) {
	id, err := requireObjectID(m)
	if err != nil {
		return domain.Record{}, err
	}
	return domain.NewRecord(
		field("id", id),
		field("documentId", textOrNil(m, "documentId")),
		field("userId", textOrNil(m, "userId")),
		field("status", textOrNil(m, "status")),
		field("responderId", textOrNil(m, "responderId")),
		field("respondedAt", textOrNil(m, "respondedAt")),
		field("createdAt", textOf(m, "createdAt")),
		field("updatedAt", textOf(m, "updatedAt")),
	), nil
}

// notificationSpec projects one notification. The nested actor, comment, document, and group objects and the data are
// never read.
func notificationSpec(m map[string]any) (domain.Record, error) {
	id, err := requireObjectID(m)
	if err != nil {
		return domain.Record{}, err
	}
	return domain.NewRecord(
		field("id", id),
		field("event", textOf(m, "event")),
		field("createdAt", textOf(m, "createdAt")),
		field("viewedAt", textOrNil(m, "viewedAt")),
		field("archivedAt", textOrNil(m, "archivedAt")),
		field("actorId", textOrNil(m, "actorId")),
		field("documentId", textOrNil(m, "documentId")),
		field("collectionId", textOrNil(m, "collectionId")),
		field("commentId", textOrNil(m, "commentId")),
		field("groupId", textOrNil(m, "groupId")),
		field("revisionId", textOrNil(m, "revisionId")),
		field("accessRequestId", textOrNil(m, "accessRequestId")),
		field("accessRequestStatus", textOrNil(m, "accessRequestStatus")),
	), nil
}

// oauthAuthenticationSpec projects one OAuth sign-in of the user. Of its client only the name is kept: the router's
// client object (presentPublishedOAuthClient) sends no id, so none is read.
func oauthAuthenticationSpec(m map[string]any) (domain.Record, error) {
	id, err := requireObjectID(m)
	if err != nil {
		return domain.Record{}, err
	}
	client := subObject(m, "oauthClient")
	return domain.NewRecord(
		field("id", id),
		field("userId", textOrNil(m, "userId")),
		field("oauthClientId", textOrNil(m, "oauthClientId")),
		field("scope", stringsOf(m, "scope")),
		field("lastActiveAt", textOrNil(m, "lastActiveAt")),
		field("createdAt", textOf(m, "createdAt")),
		field("oauthClient", domain.NewRecord(
			field("name", textOrNil(client, "name")),
		)),
	), nil
}

// reactionSpec projects one reaction to a comment. The nested user object is never read.
func reactionSpec(m map[string]any) (domain.Record, error) {
	id, err := requireObjectID(m)
	if err != nil {
		return domain.Record{}, err
	}
	return domain.NewRecord(
		field("id", id),
		field("emoji", textOf(m, "emoji")),
		field("commentId", textOrNil(m, "commentId")),
		field("userId", textOrNil(m, "userId")),
		field("createdAt", textOf(m, "createdAt")),
	), nil
}

// relationshipSpec projects one relationship between two documents.
func relationshipSpec(m map[string]any) (domain.Record, error) {
	id, err := requireObjectID(m)
	if err != nil {
		return domain.Record{}, err
	}
	return domain.NewRecord(
		field("id", id),
		field("type", textOf(m, "type")),
		field("documentId", textOrNil(m, "documentId")),
		field("reverseDocumentId", textOrNil(m, "reverseDocumentId")),
		field("userId", textOrNil(m, "userId")),
		field("createdAt", textOf(m, "createdAt")),
		field("updatedAt", textOf(m, "updatedAt")),
	), nil
}

// searchSpec projects one search of the user. The answer is never read. The server's own search source is named
// searchSource, so it does not clash with the Retriever source of a tool call.
func searchSpec(m map[string]any) (domain.Record, error) {
	id, err := requireObjectID(m)
	if err != nil {
		return domain.Record{}, err
	}
	score, _ := m["score"].(float64)
	return domain.NewRecord(
		field("id", id),
		field("query", textOf(m, "query")),
		field("searchSource", textOrNil(m, "source")),
		field("score", score),
		field("createdAt", textOf(m, "createdAt")),
	), nil
}

// viewSpec projects one view of a document by a user. The user keeps its ID and name, and no e-mail address.
func viewSpec(m map[string]any) (domain.Record, error) {
	user := subObject(m, "user")
	return domain.NewRecord(
		field("documentId", textOrNil(m, "documentId")),
		field("userId", textOrNil(m, "userId")),
		field("user", domain.NewRecord(
			field("id", textOrNil(user, "id")),
			field("name", textOrNil(user, "name")),
		)),
		field("count", intOf(m, "count")),
		field("firstViewedAt", textOrNil(m, "firstViewedAt")),
		field("lastViewedAt", textOrNil(m, "lastViewedAt")),
	), nil
}

// installationSpec projects the version facts of the Outline server.
func installationSpec(m map[string]any) (domain.Record, error) {
	return domain.NewRecord(
		field("version", textOf(m, "version")),
		field("latestVersion", textOrNil(m, "latestVersion")),
		field("versionsBehind", intOf(m, "versionsBehind")),
	), nil
}

// projectSuggestions projects suggestions.mention. The server answers three arrays, documents, users, and groups, and
// the projection flattens them into one list, each item with its kind. A document is named by its title, and the other
// two by their name. The e-mail addresses that the server may add are never read.
func projectSuggestions(root any) (domain.Record, error) {
	m, err := dataObject(root)
	if err != nil {
		return domain.Record{}, err
	}
	kinds := []struct {
		key, kind, label string
	}{
		{"documents", "document", "title"},
		{"users", "user", "name"},
		{"groups", "group", "name"},
	}
	var items []domain.Record
	for _, k := range kinds {
		list, ok := jsonList(m[k.key])
		if !ok {
			return domain.Record{}, domain.ErrUnexpectedResponse
		}
		for _, raw := range list {
			obj, ok := jsonObject(raw)
			if !ok {
				return domain.Record{}, domain.ErrUnexpectedResponse
			}
			id, err := requiredID(obj, "id", nonBlank)
			if err != nil {
				return domain.Record{}, err
			}
			items = append(items, domain.NewRecord(
				field("kind", k.kind),
				field("id", id),
				field("name", textOf(obj, k.label)),
			))
		}
	}
	return domain.NewList(items), nil
}
