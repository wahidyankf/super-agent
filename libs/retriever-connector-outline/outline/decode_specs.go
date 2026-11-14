package outline

import (
	"slices"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// The object projections of the stage a part 2a reads. Each spec reads one Outline object and keeps only the fields of
// its row's field set, in that order. A spec never reads a field that the redaction rules forbid: the e-mail address of
// a user, the link of a share, the editor data of a template or a comment, and the webhook secret. The reply rules of
// decode.go apply: a missing or null field is the zero value, and an ID that is required and invalid is an unexpected
// reply.

// objectSpec projects one decoded Outline object onto its Record.
type objectSpec func(m map[string]any) (domain.Record, error)

// nonBlank is the ID check of an object whose ID has no format beyond being text.
func nonBlank(raw string) error {
	if raw == "" {
		return domain.ErrUnexpectedResponse
	}
	return nil
}

// intOf returns the whole number at key, or 0 when it is missing, null, or not a number.
func intOf(m map[string]any, key string) int {
	f, _ := m[key].(float64)
	return int(f)
}

// boolOf returns the flag at key, or false when it is missing, null, or not a flag.
func boolOf(m map[string]any, key string) bool {
	b, _ := m[key].(bool)
	return b
}

// projectItems projects each object of a list with the spec. A non-object item is an unexpected reply.
func projectItems(items []any, spec objectSpec) (domain.Record, error) {
	out := make([]domain.Record, 0, len(items))
	for _, raw := range items {
		m, ok := jsonObject(raw)
		if !ok {
			return domain.Record{}, domain.ErrUnexpectedResponse
		}
		rec, err := spec(m)
		if err != nil {
			return domain.Record{}, err
		}
		out = append(out, rec)
	}
	return domain.NewList(out), nil
}

// projectObjectOf returns a projection that reads the data member of the reply as one object, with the spec.
func projectObjectOf(spec objectSpec) func(any) (domain.Record, error) {
	return func(root any) (domain.Record, error) {
		m, err := dataObject(root)
		if err != nil {
			return domain.Record{}, err
		}
		return spec(m)
	}
}

// projectListOf returns a projection that reads the data member of the reply as a list of objects, with the spec.
func projectListOf(spec objectSpec) func(any) (domain.Record, error) {
	return func(root any) (domain.Record, error) {
		data, err := dataList(root)
		if err != nil {
			return domain.Record{}, err
		}
		return projectItems(data, spec)
	}
}

// projectNestedOf returns a projection that reads one list under key of the data object, with the spec. The Outline
// replies of the membership, pin, star, and group calls carry their list this way, next to the users or documents they
// name, and the extra members are ignored.
func projectNestedOf(key string, spec objectSpec) func(any) (domain.Record, error) {
	return func(root any) (domain.Record, error) {
		m, err := dataObject(root)
		if err != nil {
			return domain.Record{}, err
		}
		items, ok := jsonList(m[key])
		if !ok {
			return domain.Record{}, domain.ErrUnexpectedResponse
		}
		return projectItems(items, spec)
	}
}

// projectShareInfo projects shares.info. The Outline reply carries a list of shares under data.shares, and this
// adapter asks for one share by its ID, so the reply holds the one share or nothing. An empty list is ErrNotFound.
func projectShareInfo(root any) (domain.Record, error) {
	m, err := dataObject(root)
	if err != nil {
		return domain.Record{}, err
	}
	items, ok := jsonList(m["shares"])
	if !ok {
		return domain.Record{}, domain.ErrUnexpectedResponse
	}
	if len(items) == 0 {
		return domain.Record{}, domain.ErrNotFound
	}
	first, ok := jsonObject(items[0])
	if !ok {
		return domain.Record{}, domain.ErrUnexpectedResponse
	}
	return shareSpec(first)
}

// projectChildren projects documents.children. The Outline reply names the document, and its children sit under it
// as a tree, so the reply holds one NavNode under data, not a list of them. The children are flattened depth first,
// each naming the document as its parent, and the depth of a child counts from 0.
func projectChildren(root any) (domain.Record, error) {
	top, ok := jsonObject(root)
	if !ok {
		return domain.Record{}, domain.ErrUnexpectedResponse
	}
	if top["data"] == nil {
		// A document that is in no collection has no tree, and the router sends data as undefined: no data member at all.
		// That is no children, so it is an empty list.
		return domain.NewList(nil), nil
	}
	m, err := dataObject(root)
	if err != nil {
		return domain.Record{}, err
	}
	parent, err := requiredID(m, "id", validDocument)
	if err != nil {
		return domain.Record{}, err
	}
	children, _ := jsonList(m["children"])
	var items []domain.Record
	if err := flattenTree(children, &parent, 0, &items); err != nil {
		return domain.Record{}, err
	}
	return domain.NewList(items), nil
}

// membershipSpec projects one user membership of a document or a collection.
func membershipSpec(m map[string]any) (domain.Record, error) {
	id, err := requiredID(m, "id", domain.ValidateUUID)
	if err != nil {
		return domain.Record{}, err
	}
	return domain.NewRecord(
		field("id", id),
		field("userId", textOrNil(m, "userId")),
		field("documentId", textOrNil(m, "documentId")),
		field("collectionId", textOrNil(m, "collectionId")),
		field("permission", textOf(m, "permission")),
		field("createdById", textOrNil(m, "createdById")),
		field("sourceId", textOrNil(m, "sourceId")),
		field("index", textOrNil(m, "index")),
	), nil
}

// groupMembershipSpec projects one group membership of a document or a collection.
func groupMembershipSpec(m map[string]any) (domain.Record, error) {
	id, err := requiredID(m, "id", domain.ValidateUUID)
	if err != nil {
		return domain.Record{}, err
	}
	return domain.NewRecord(
		field("id", id),
		field("groupId", textOrNil(m, "groupId")),
		field("documentId", textOrNil(m, "documentId")),
		field("collectionId", textOrNil(m, "collectionId")),
		field("permission", textOf(m, "permission")),
		field("sourceId", textOrNil(m, "sourceId")),
	), nil
}

// groupUserSpec projects one member of a group. A group member has no ID of its own; the group and the user name it.
func groupUserSpec(m map[string]any) (domain.Record, error) {
	return domain.NewRecord(
		field("groupId", textOf(m, "groupId")),
		field("userId", textOf(m, "userId")),
		field("permission", textOf(m, "permission")),
		field("createdAt", textOrNil(m, "createdAt")),
	), nil
}

// insightSpec projects one day of a document's insights. The day is its date, and its period names the window.
func insightSpec(m map[string]any) (domain.Record, error) {
	return domain.NewRecord(
		field("date", textOf(m, "date")),
		field("period", textOf(m, "period")),
		field("viewCount", intOf(m, "viewCount")),
		field("viewerCount", intOf(m, "viewerCount")),
		field("commentCount", intOf(m, "commentCount")),
		field("reactionCount", intOf(m, "reactionCount")),
		field("revisionCount", intOf(m, "revisionCount")),
		field("editorCount", intOf(m, "editorCount")),
	), nil
}

// userNotificationSpec projects the user that a notification write answers: the user, with the notification settings the
// wiki keeps for it. The settings are a map from an event type to a flag; each flag is kept, in the sorted order of the
// event types, and a value that is not a flag is dropped. The e-mail address is never read.
func userNotificationSpec(m map[string]any) (domain.Record, error) {
	user, err := userSpec(m)
	if err != nil {
		return domain.Record{}, err
	}
	settings := subObject(m, "notificationSettings")
	events := make([]string, 0, len(settings))
	for event := range settings {
		events = append(events, event)
	}
	slices.Sort(events)
	flags := make([]domain.Field, 0, len(events))
	for _, event := range events {
		if on, ok := settings[event].(bool); ok {
			flags = append(flags, field(event, on))
		}
	}
	fields := user.Fields()
	// The user keeps the fields of userSpec, which is the users.info projection, and then its settings.
	return domain.NewRecord(append(fields, field("notificationSettings", domain.NewRecord(flags...)))...), nil
}

// userSpec projects one user. The e-mail address is never read.
func userSpec(m map[string]any) (domain.Record, error) {
	id, err := requiredID(m, "id", domain.ValidateUUID)
	if err != nil {
		return domain.Record{}, err
	}
	return domain.NewRecord(
		field("id", id),
		field("name", textOf(m, "name")),
		field("role", textOf(m, "role")),
		field("isSuspended", boolOf(m, "isSuspended")),
		field("lastActiveAt", textOrNil(m, "lastActiveAt")),
		field("createdAt", textOf(m, "createdAt")),
		field("updatedAt", textOf(m, "updatedAt")),
		field("deletedAt", textOrNil(m, "deletedAt")),
	), nil
}

// groupSpec projects one group. The identity-provider link is never read.
func groupSpec(m map[string]any) (domain.Record, error) {
	id, err := requiredID(m, "id", domain.ValidateUUID)
	if err != nil {
		return domain.Record{}, err
	}
	return domain.NewRecord(
		field("id", id),
		field("name", textOf(m, "name")),
		field("description", textOrNil(m, "description")),
		field("color", textOrNil(m, "color")),
		field("icon", textOrNil(m, "icon")),
		field("memberCount", intOf(m, "memberCount")),
		field("disableMentions", boolOf(m, "disableMentions")),
		field("createdAt", textOf(m, "createdAt")),
		field("updatedAt", textOf(m, "updatedAt")),
	), nil
}

// commentSpec projects one comment. Its text is the plain text of its editor data, so the editor JSON is never kept.
// A reply without editor data keeps the text the server sent, if any.
func commentSpec(m map[string]any) (domain.Record, error) {
	id, err := requiredID(m, "id", domain.ValidateUUID)
	if err != nil {
		return domain.Record{}, err
	}
	text := textOf(m, "text")
	if data := m["data"]; data != nil {
		text = plainText(data)
	}
	return domain.NewRecord(
		field("id", id),
		field("documentId", textOf(m, "documentId")),
		field("parentCommentId", textOrNil(m, "parentCommentId")),
		field("createdById", textOf(m, "createdById")),
		field("resolvedById", textOrNil(m, "resolvedById")),
		field("resolvedAt", textOrNil(m, "resolvedAt")),
		field("createdAt", textOf(m, "createdAt")),
		field("updatedAt", textOf(m, "updatedAt")),
		field("text", text),
		field("anchorText", textOrNil(m, "anchorText")),
	), nil
}

// revisionSpec projects one revision with its Markdown text.
func revisionSpec(m map[string]any) (domain.Record, error) {
	id, err := requiredID(m, "id", domain.ValidateUUID)
	if err != nil {
		return domain.Record{}, err
	}
	return domain.NewRecord(
		field("id", id),
		field("documentId", textOf(m, "documentId")),
		field("title", textOf(m, "title")),
		field("name", textOrNil(m, "name")),
		field("text", textOf(m, "text")),
		field("icon", textOrNil(m, "icon")),
		field("color", textOrNil(m, "color")),
		field("createdAt", textOf(m, "createdAt")),
		field("createdById", textOrNil(m, "createdById")),
		field("deletedAt", textOrNil(m, "deletedAt")),
	), nil
}

// revisionRowSpec projects one revision of a list, which carries no text.
func revisionRowSpec(m map[string]any) (domain.Record, error) {
	id, err := requiredID(m, "id", domain.ValidateUUID)
	if err != nil {
		return domain.Record{}, err
	}
	return domain.NewRecord(
		field("id", id),
		field("documentId", textOf(m, "documentId")),
		field("title", textOf(m, "title")),
		field("name", textOrNil(m, "name")),
		field("icon", textOrNil(m, "icon")),
		field("color", textOrNil(m, "color")),
		field("createdAt", textOf(m, "createdAt")),
		field("createdById", textOrNil(m, "createdById")),
		field("deletedAt", textOrNil(m, "deletedAt")),
	), nil
}

// shareSpec projects one share. The public link (url and urlId) is never read.
func shareSpec(m map[string]any) (domain.Record, error) {
	id, err := requiredID(m, "id", domain.ValidateUUID)
	if err != nil {
		return domain.Record{}, err
	}
	return domain.NewRecord(
		field("id", id),
		field("documentId", textOrNil(m, "documentId")),
		field("collectionId", textOrNil(m, "collectionId")),
		field("documentTitle", textOrNil(m, "documentTitle")),
		field("sourceTitle", textOrNil(m, "sourceTitle")),
		field("published", boolOf(m, "published")),
		field("includeChildDocuments", boolOf(m, "includeChildDocuments")),
		field("allowIndexing", boolOf(m, "allowIndexing")),
		field("allowSubscriptions", boolOf(m, "allowSubscriptions")),
		field("showLastUpdated", boolOf(m, "showLastUpdated")),
		field("showTOC", boolOf(m, "showTOC")),
		field("title", textOrNil(m, "title")),
		field("domain", textOrNil(m, "domain")),
		field("views", intOf(m, "views")),
		field("lastAccessedAt", textOrNil(m, "lastAccessedAt")),
		// The router sends the creator as the createdBy object, and omits it on a public view (presenters/share.ts).
		field("createdById", textOrNil(subObject(m, "createdBy"), "id")),
		field("createdAt", textOf(m, "createdAt")),
		field("updatedAt", textOf(m, "updatedAt")),
	), nil
}

// templateSpec projects one template. Its editor data is never read.
func templateSpec(m map[string]any) (domain.Record, error) {
	id, err := requiredID(m, "id", nonBlank)
	if err != nil {
		return domain.Record{}, err
	}
	return domain.NewRecord(
		field("id", id),
		field("urlId", textOrNil(m, "urlId")),
		field("url", textOrNil(m, "url")),
		field("title", textOf(m, "title")),
		field("icon", textOrNil(m, "icon")),
		field("color", textOrNil(m, "color")),
		field("fullWidth", boolOf(m, "fullWidth")),
		field("collectionId", textOrNil(m, "collectionId")),
		field("createdAt", textOf(m, "createdAt")),
		field("updatedAt", textOf(m, "updatedAt")),
		field("publishedAt", textOrNil(m, "publishedAt")),
		field("deletedAt", textOrNil(m, "deletedAt")),
		// The router sends the creator and the last editor as objects (presenters/template.ts).
		field("createdById", textOrNil(subObject(m, "createdBy"), "id")),
		field("updatedById", textOrNil(subObject(m, "updatedBy"), "id")),
	), nil
}

// pinSpec projects one pin of a document in a collection or on Home.
func pinSpec(m map[string]any) (domain.Record, error) {
	id, err := requiredID(m, "id", domain.ValidateUUID)
	if err != nil {
		return domain.Record{}, err
	}
	return domain.NewRecord(
		field("id", id),
		field("documentId", textOf(m, "documentId")),
		field("collectionId", textOrNil(m, "collectionId")),
		field("index", textOf(m, "index")),
		field("createdAt", textOf(m, "createdAt")),
		field("updatedAt", textOf(m, "updatedAt")),
	), nil
}

// starSpec projects one star of a document or a collection.
func starSpec(m map[string]any) (domain.Record, error) {
	id, err := requiredID(m, "id", domain.ValidateUUID)
	if err != nil {
		return domain.Record{}, err
	}
	return domain.NewRecord(
		field("id", id),
		field("documentId", textOrNil(m, "documentId")),
		field("collectionId", textOrNil(m, "collectionId")),
		field("index", textOf(m, "index")),
		field("createdAt", textOf(m, "createdAt")),
		field("updatedAt", textOf(m, "updatedAt")),
	), nil
}

// subscriptionSpec projects one subscription to the updates of a document or a collection.
func subscriptionSpec(m map[string]any) (domain.Record, error) {
	id, err := requiredID(m, "id", domain.ValidateUUID)
	if err != nil {
		return domain.Record{}, err
	}
	return domain.NewRecord(
		field("id", id),
		field("userId", textOf(m, "userId")),
		field("documentId", textOrNil(m, "documentId")),
		field("collectionId", textOrNil(m, "collectionId")),
		field("event", textOf(m, "event")),
		field("createdAt", textOf(m, "createdAt")),
		field("updatedAt", textOf(m, "updatedAt")),
	), nil
}
