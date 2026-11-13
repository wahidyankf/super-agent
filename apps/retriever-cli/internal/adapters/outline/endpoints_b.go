package outline

import (
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// The Outline method names of the stage b writes. Each one is the wire path /api/<method>, and each is used by one
// write endpoint row below.
const (
	methodDocumentsCreate     = "documents.create"
	methodDocumentsDuplicate  = "documents.duplicate"
	methodDocumentsMove       = "documents.move"
	methodDocumentsRestore    = "documents.restore"
	methodDocumentsTemplatize = "documents.templatize"
	methodDocumentsUnpublish  = "documents.unpublish"
	methodDocumentsUpdate     = "documents.update"

	methodCollectionsCreate    = "collections.create"
	methodCollectionsDuplicate = "collections.duplicate"
	methodCollectionsExport    = "collections.export"
	methodCollectionsMove      = "collections.move"
	methodCollectionsRestore   = "collections.restore"
	methodCollectionsUpdate    = "collections.update"

	methodCommentsCreate         = "comments.create"
	methodCommentsUpdate         = "comments.update"
	methodCommentsResolve        = "comments.resolve"
	methodCommentsUnresolve      = "comments.unresolve"
	methodCommentsAddReaction    = "comments.add_reaction"
	methodCommentsRemoveReaction = "comments.remove_reaction"

	methodAccessRequestsCreate = "accessRequests.create"
	methodRevisionsUpdate      = "revisions.update"

	methodNotificationsUpdate    = "notifications.update"
	methodNotificationsUpdateAll = "notifications.update_all"

	methodPinsCreate = "pins.create"
	methodPinsDelete = "pins.delete"
	methodPinsUpdate = "pins.update"

	methodSearchesDelete = "searches.delete"
	methodSearchesUpdate = "searches.update"

	methodStarsCreate = "stars.create"
	methodStarsDelete = "stars.delete"
	methodStarsUpdate = "stars.update"

	methodSubscriptionsCreate = "subscriptions.create"
	methodSubscriptionsDelete = "subscriptions.delete"

	methodTemplatesCreate    = "templates.create"
	methodTemplatesDuplicate = "templates.duplicate"
	methodTemplatesRestore   = "templates.restore"
	methodTemplatesUpdate    = "templates.update"

	methodUserMembershipsUpdate = "userMemberships.update"

	methodUsersNotificationsSubscribe   = "users.notificationsSubscribe"
	methodUsersNotificationsUnsubscribe = "users.notificationsUnsubscribe"
)

// writeEndpoints maps the stage b writes (ADR 007 section 5, class W) to their endpoints. It is kept apart from the
// read table, so the reader cannot reach a write method. Each body names only the keys of the operation's parameters, and
// an argument that is not set is left out. A write sends its request once; the client retries it only after a 429.
var writeEndpoints = map[domain.OperationID]endpoint{
	domain.OpCollectionsCreate: {
		method: methodCollectionsCreate,
		body: argsBody(false, nil, "name", "description", "permission", "sharing", "commenting",
			"templateManagement"),
		reply: projectCollectionInfo,
	},
	domain.OpCollectionsDuplicate: {
		method: methodCollectionsDuplicate,
		body:   argsBody(false, renamed("collectionId", "id"), "collectionId", "name"),
		reply:  projectCollectionInfo,
	},
	domain.OpCollectionsExport: {
		method: methodCollectionsExport,
		body:   argsBody(false, renamed("collectionId", "id"), "collectionId", "format", "includeAttachments"),
		reply:  projectAckUnder("fileOperation", fileOperationSpec),
	},
	domain.OpCollectionsMove: {
		method: methodCollectionsMove,
		body:   argsBody(false, renamed("collectionId", "id"), "collectionId", "index"),
		reply:  projectAckIndex,
	},
	domain.OpCollectionsRestore: {
		method: methodCollectionsRestore,
		body:   argsBody(false, renamed("collectionId", "id"), "collectionId"),
		reply:  projectCollectionInfo,
	},
	domain.OpCollectionsUpdate: {
		method: methodCollectionsUpdate,
		body: argsBody(false, renamed("collectionId", "id"), "collectionId", "name", "description", "permission",
			"sharing", "commenting", "templateManagement"),
		reply: projectCollectionInfo,
	},
	domain.OpCommentsCreate: {
		method: methodCommentsCreate,
		body:   argsBody(false, nil, "documentId", "text", "parentCommentId"),
		reply:  projectObjectOf(commentSpec),
	},
	domain.OpCommentsUpdate: {
		method: methodCommentsUpdate,
		body:   argsBody(false, renamed("commentId", "id"), "commentId", "text"),
		reply:  projectObjectOf(commentSpec),
	},
	domain.OpCommentsResolve: {
		method: methodCommentsResolve,
		body:   argsBody(false, renamed("commentId", "id"), "commentId"),
		reply:  projectObjectOf(commentSpec),
	},
	domain.OpCommentsUnresolve: {
		method: methodCommentsUnresolve,
		body:   argsBody(false, renamed("commentId", "id"), "commentId"),
		reply:  projectObjectOf(commentSpec),
	},
	domain.OpCommentsAddReaction: {
		method: methodCommentsAddReaction,
		body:   argsBody(false, renamed("commentId", "id"), "commentId", "emoji"),
		reply:  projectAck,
	},
	domain.OpCommentsRemoveReaction: {
		method: methodCommentsRemoveReaction,
		body:   argsBody(false, renamed("commentId", "id"), "commentId", "emoji"),
		reply:  projectAck,
	},
	domain.OpAccessRequestsCreate: {
		method: methodAccessRequestsCreate,
		body:   argsBody(false, nil, "documentId"),
		reply:  projectObjectOf(accessRequestSpec),
	},
	domain.OpRevisionsUpdate: {
		method: methodRevisionsUpdate,
		body:   argsBody(false, renamed("revisionId", "id"), "revisionId", "name"),
		reply:  projectObjectOf(revisionRowSpec),
	},
	domain.OpDocumentsCreate: {
		method: methodDocumentsCreate,
		body: argsBody(false, nil, "title", "text", "collectionId", "parentDocumentId", "templateId",
			"publish"),
		reply: projectObjectOf(documentRevisionRecord),
	},
	domain.OpDocumentsDuplicate: {
		method: methodDocumentsDuplicate,
		body: argsBody(false, renamed("documentId", "id"), "documentId", "title", "recursive", "publish",
			"collectionId", "parentDocumentId"),
		reply: projectNestedOf("documents", documentRevisionRecord),
	},
	domain.OpDocumentsMove: {
		method: methodDocumentsMove,
		body: argsBody(false, renamed("documentId", "id"), "documentId", "collectionId", "parentDocumentId",
			"index"),
		reply: projectNestedOf("documents", documentRevisionRecord),
	},
	domain.OpDocumentsRestore: {
		method: methodDocumentsRestore,
		body:   argsBody(false, renamed("documentId", "id"), "documentId", "collectionId", "revisionId"),
		reply:  projectObjectOf(documentRevisionRecord),
	},
	domain.OpDocumentsTemplatize: {
		method: methodDocumentsTemplatize,
		body:   argsBody(false, renamed("documentId", "id"), "documentId", "publish", "collectionId"),
		reply:  projectObjectOf(templateSpec),
	},
	domain.OpDocumentsUnpublish: {
		method: methodDocumentsUnpublish,
		body:   argsBody(false, renamed("documentId", "id"), "documentId", "detach"),
		reply:  projectObjectOf(documentRevisionRecord),
	},
	domain.OpDocumentsUpdate: {
		method: methodDocumentsUpdate,
		body: argsBody(false, renamed("documentId", "id"), "documentId", "title", "text", "editMode",
			"findText", "lastRevision", "publish", "collectionId"),
		reply: projectObjectOf(documentRevisionRecord),
	},
	domain.OpNotificationsUpdate: {
		method: methodNotificationsUpdate,
		body:   argsBody(false, renamed("notificationId", "id"), "notificationId", "viewedAt", "archivedAt"),
		reply:  projectObjectOf(notificationSpec),
	},
	domain.OpNotificationsUpdateAll: {
		method: methodNotificationsUpdateAll,
		body:   argsBody(false, nil, "viewedAt", "archivedAt"),
		reply:  projectAckTotal,
	},
	domain.OpPinsCreate: {
		method: methodPinsCreate,
		body:   argsBody(false, nil, "documentId", "collectionId", "index"),
		reply:  projectObjectOf(pinSpec),
	},
	domain.OpPinsDelete: {
		method: methodPinsDelete,
		body:   argsBody(false, renamed("pinId", "id"), "pinId"),
		reply:  projectAck,
	},
	domain.OpPinsUpdate: {
		method: methodPinsUpdate,
		body:   argsBody(false, renamed("pinId", "id"), "pinId", "index"),
		reply:  projectObjectOf(pinSpec),
	},
	domain.OpSearchesDelete: {
		method: methodSearchesDelete,
		body:   argsBody(false, renamed("searchId", "id"), "searchId", "query"),
		reply:  projectAck,
	},
	domain.OpSearchesUpdate: {
		method: methodSearchesUpdate,
		body:   argsBody(false, renamed("searchId", "id"), "searchId", "score"),
		reply:  projectObjectOf(searchSpec),
	},
	domain.OpStarsCreate: {
		method: methodStarsCreate,
		body:   argsBody(false, nil, "documentId", "collectionId", "index"),
		reply:  projectObjectOf(starSpec),
	},
	domain.OpStarsDelete: {
		method: methodStarsDelete,
		body:   argsBody(false, renamed("starId", "id"), "starId"),
		reply:  projectAck,
	},
	domain.OpStarsUpdate: {
		method: methodStarsUpdate,
		body:   argsBody(false, renamed("starId", "id"), "starId", "index"),
		reply:  projectObjectOf(starSpec),
	},
	domain.OpSubscriptionsCreate: {
		method: methodSubscriptionsCreate,
		body:   argsBody(false, nil, "event", "documentId", "collectionId"),
		reply:  projectObjectOf(subscriptionSpec),
	},
	domain.OpSubscriptionsDelete: {
		method: methodSubscriptionsDelete,
		body:   argsBody(false, renamed("subscriptionId", "id"), "subscriptionId"),
		reply:  projectAck,
	},
	domain.OpTemplatesCreate: {
		method: methodTemplatesCreate,
		body:   argsBody(false, nil, "title", "collectionId", "icon", "color", "publish"),
		reply:  projectObjectOf(templateSpec),
	},
	domain.OpTemplatesDuplicate: {
		method: methodTemplatesDuplicate,
		body:   argsBody(false, renamed("templateId", "id"), "templateId", "title", "collectionId"),
		reply:  projectObjectOf(templateSpec),
	},
	domain.OpTemplatesRestore: {
		method: methodTemplatesRestore,
		body:   argsBody(false, renamed("templateId", "id"), "templateId"),
		reply:  projectObjectOf(templateSpec),
	},
	domain.OpTemplatesUpdate: {
		method: methodTemplatesUpdate,
		body: argsBody(false, renamed("templateId", "id"), "templateId", "title", "icon", "color", "fullWidth",
			"collectionId", "publish"),
		reply: projectObjectOf(templateSpec),
	},
	domain.OpUserMembershipsUpdate: {
		method: methodUserMembershipsUpdate,
		body:   argsBody(false, renamed("membershipId", "id"), "membershipId", "index"),
		reply:  projectObjectOf(membershipSpec),
	},
	domain.OpUsersNotificationsSubscribe: {
		method: methodUsersNotificationsSubscribe,
		body:   argsBody(false, nil, "eventType"),
		reply:  projectObjectOf(userNotificationSpec),
	},
	domain.OpUsersNotificationsUnsubscribe: {
		method: methodUsersNotificationsUnsubscribe,
		body:   argsBody(false, nil, "eventType"),
		reply:  projectObjectOf(userNotificationSpec),
	},
}

// projectAck projects a plain acknowledgement, the wiki's {success: true} reply of a reaction or a delete. Retriever answers
// ok true, the field the catalog declares, and the reply must say success true.
func projectAck(root any) (domain.Record, error) {
	if _, err := ackObject(root); err != nil {
		return domain.Record{}, err
	}
	return domain.NewRecord(field("ok", true)), nil
}

// ackObject returns the reply object when its success is true. Any other reply is an unexpected reply: a reply that says
// success false, one that says nothing of success, and the {ok: true} shape the wiki never sends for an acknowledgement.
// A write that gets one of them has an unknown outcome (finding m1).
func ackObject(root any) (map[string]any, error) {
	m, ok := jsonObject(root)
	if !ok || m["success"] != true {
		return nil, domain.ErrUnexpectedResponse
	}
	return m, nil
}

// ackData returns the data member of an acknowledgement, or nil when the reply names none. A missing value reads as its
// zero value, as every projection of a tolerant decode does.
func ackData(m map[string]any) map[string]any {
	data, _ := jsonObject(m["data"])
	return data
}

// projectAckTotal projects the answer of notifications.update_all, the wiki's {success: true, data: {total}} reply. Retriever
// answers ok true and the total: the number of notifications the wiki changed.
func projectAckTotal(root any) (domain.Record, error) {
	m, err := ackObject(root)
	if err != nil {
		return domain.Record{}, err
	}
	return domain.NewRecord(field("ok", true), field("total", intOf(ackData(m), "total"))), nil
}

// projectAckIndex projects the answer of collections.move, the wiki's {success: true, data: {index}} reply. Retriever answers
// ok true and the index the wiki kept.
func projectAckIndex(root any) (domain.Record, error) {
	m, err := ackObject(root)
	if err != nil {
		return domain.Record{}, err
	}
	return domain.NewRecord(field("ok", true), field("index", textOf(ackData(m), "index"))), nil
}

// projectAckUnder projects an acknowledgement that also names one object under data, the wiki's {success: true,
// data: {<key>}} reply of collections.export and collections.export_all. The reply must say success true, and the object is
// projected with its spec.
func projectAckUnder(key string, spec objectSpec) func(any) (domain.Record, error) {
	return func(root any) (domain.Record, error) {
		m, err := ackObject(root)
		if err != nil {
			return domain.Record{}, err
		}
		obj, ok := jsonObject(ackData(m)[key])
		if !ok {
			return domain.Record{}, domain.ErrUnexpectedResponse
		}
		return spec(obj)
	}
}
