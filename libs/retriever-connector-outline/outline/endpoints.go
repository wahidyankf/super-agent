package outline

import (
	"context"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// The Outline method names. Each one is the wire path /api/<method>. Each is used by one endpoint row below, except
// documents.export, which is the vault export's method and the fallback of documents.get.
const (
	methodCollectionsList             = "collections.list"
	methodCollectionsInfo             = "collections.info"
	methodCollectionsDocuments        = "collections.documents"
	methodCollectionsMemberships      = "collections.memberships"
	methodCollectionsGroupMemberships = "collections.group_memberships"
	methodDocumentsInfo               = "documents.info"
	methodDocumentsList               = "documents.list"
	methodDocumentsSearch             = "documents.search"
	methodDocumentsExport             = "documents.export"
	methodDocumentsArchived           = "documents.archived"
	methodDocumentsDocuments          = "documents.documents"
	methodDocumentsDeleted            = "documents.deleted"
	methodDocumentsDrafts             = "documents.drafts"
	methodDocumentsGroupMemberships   = "documents.group_memberships"
	methodDocumentsInsights           = "documents.insights"
	methodDocumentsMemberships        = "documents.memberships"
	methodDocumentsSearchTitles       = "documents.search_titles"
	methodDocumentsUsers              = "documents.users"
	methodDocumentsViewed             = "documents.viewed"
	methodCommentsInfo                = "comments.info"
	methodCommentsList                = "comments.list"
	methodRevisionsExport             = "revisions.export"
	methodRevisionsInfo               = "revisions.info"
	methodRevisionsList               = "revisions.list"
	methodSharesInfo                  = "shares.info"
	methodSharesList                  = "shares.list"
	methodUsersInfo                   = "users.info"
	methodUsersList                   = "users.list"
	methodGroupsInfo                  = "groups.info"
	methodGroupsList                  = "groups.list"
	methodGroupsMemberships           = "groups.memberships"
	methodTemplatesInfo               = "templates.info"
	methodTemplatesList               = "templates.list"
	methodPinsInfo                    = "pins.info"
	methodPinsList                    = "pins.list"
	methodStarsList                   = "stars.list"
	methodSubscriptionsInfo           = "subscriptions.info"
	methodSubscriptionsList           = "subscriptions.list"
	methodAuthInfo                    = "auth.info"
)

// opExport is the operation of the one call that is not in the catalog: the vault export reads documents.export as
// the Markdown of a document. It is a read, so a read source may make it.
const opExport domain.OperationID = "documents.export"

// endpoint is one row of the Outline endpoint table. It names the Outline method of one
// catalog operation, builds the JSON body from the operation's declared arguments only, and projects the decoded reply
// onto the operation's Record, tolerant of null, missing, and extra fields.
type endpoint struct {
	method string
	// body returns the request body. It names every key the request sends; a test checks that no other key appears.
	body func(a domain.Args) map[string]any
	// reply projects the decoded reply body onto the operation's Record. A reply of the wrong shape is
	// domain.ErrUnexpectedResponse.
	reply func(root any) (domain.Record, error)
	// pageAll marks an operation that reads every page of its list, one page of pageSize at a time, until a short page.
	pageAll bool
	// after completes a Record that the reply left partial, such as a document whose text is read by its export.
	after func(ctx context.Context, g *Gateway, a domain.Args, rec domain.Record) (domain.Record, error)
	// current, when set, reads the object before its change and returns the arguments the request carries. The wiki replaces
	// some stored values when the request leaves them out, so the read values fill the arguments the caller did not set.
	current func(ctx context.Context, g *Gateway, a domain.Args) (domain.Args, error)
}

// endpoints maps each catalog operation the reader serves to its endpoint. The method allowlist is built from this
// table, so only a method named here ever reaches the network.
var endpoints = map[domain.OperationID]endpoint{
	domain.OpCollectionsList: {
		method:  methodCollectionsList,
		body:    pageBody,
		reply:   projectCollectionList,
		pageAll: true,
	},
	domain.OpCollectionsInfo: {
		method: methodCollectionsInfo,
		body:   idBody("collectionId"),
		reply:  projectCollectionInfo,
	},
	domain.OpCollectionsDocuments: {
		method: methodCollectionsDocuments,
		body:   idBody("collectionId"),
		reply:  projectTree,
	},
	domain.OpCollectionsMemberships: {
		method: methodCollectionsMemberships,
		body:   argsBody(true, renamed("collectionId", "id"), "collectionId", "query", "permission"),
		reply:  projectNestedOf("memberships", membershipSpec),
	},
	domain.OpCollectionsGroupMembers: {
		method: methodCollectionsGroupMemberships,
		body:   argsBody(true, renamed("collectionId", "id"), "collectionId", "query", "permission"),
		reply:  projectNestedOf("groupMemberships", groupMembershipSpec),
	},
	domain.OpDocumentsGet: {
		method: methodDocumentsInfo,
		body:   idBody("documentId"),
		reply:  projectDocument,
		after:  exportMissingText,
	},
	domain.OpDocumentsList: {
		method: methodDocumentsList,
		body:   documentListBody,
		reply:  projectDocumentList,
	},
	domain.OpDocumentsSearch: {
		method: methodDocumentsSearch,
		body:   searchBody,
		reply:  projectSearch,
	},
	domain.OpDocumentsArchived: {
		method: methodDocumentsArchived,
		body:   argsBody(true, nil, "collectionId", "sort", "direction"),
		reply:  projectDocumentList,
	},
	domain.OpDocumentsChildren: {
		method: methodDocumentsDocuments,
		body:   argsBody(false, renamed("documentId", "id"), "documentId"),
		reply:  projectChildren,
	},
	domain.OpDocumentsDeleted: {
		method: methodDocumentsDeleted,
		body:   argsBody(true, nil, "sort", "direction"),
		reply:  projectDocumentList,
	},
	domain.OpDocumentsDrafts: {
		method: methodDocumentsDrafts,
		body:   argsBody(true, nil, "collectionId", "sort", "direction"),
		reply:  projectDocumentList,
	},
	domain.OpDocumentsGroupMemberships: {
		method: methodDocumentsGroupMemberships,
		body:   argsBody(true, renamed("documentId", "id"), "documentId", "query", "permission"),
		reply:  projectNestedOf("groupMemberships", groupMembershipSpec),
	},
	domain.OpDocumentsInsights: {
		method: methodDocumentsInsights,
		body:   argsBody(false, renamed("documentId", "id"), "documentId", "startDate", "endDate"),
		reply:  projectListOf(insightSpec),
	},
	domain.OpDocumentsMemberships: {
		method: methodDocumentsMemberships,
		body:   argsBody(true, renamed("documentId", "id"), "documentId", "query", "permission"),
		reply:  projectNestedOf("memberships", membershipSpec),
	},
	domain.OpDocumentsSearchTitles: {
		method: methodDocumentsSearchTitles,
		body: argsBody(true, nil, "query", "collectionId", "documentId", "userId", "statusFilter", "sort",
			"direction"),
		reply: projectDocumentList,
	},
	domain.OpDocumentsUsers: {
		method: methodDocumentsUsers,
		body:   argsBody(true, renamed("documentId", "id"), "documentId", "query", "userId"),
		reply:  projectListOf(userSpec),
	},
	domain.OpDocumentsViewed: {
		method: methodDocumentsViewed,
		body:   argsBody(true, nil, "sort", "direction"),
		reply:  projectDocumentList,
	},
	domain.OpCommentsGet: {
		method: methodCommentsInfo,
		body:   argsBody(false, renamed("commentId", "id"), "commentId", "includeAnchorText"),
		reply:  projectObjectOf(commentSpec),
	},
	domain.OpCommentsList: {
		method: methodCommentsList,
		body: argsBody(true, nil, "documentId", "collectionId", "parentCommentId", "statusFilter",
			"includeAnchorText", "sort", "direction"),
		reply: projectListOf(commentSpec),
	},
	domain.OpRevisionsExport: {
		method: methodRevisionsExport,
		body:   argsBody(false, renamed("revisionId", "id"), "revisionId"),
		reply:  projectExport,
	},
	domain.OpRevisionsGet: {
		method: methodRevisionsInfo,
		body:   argsBody(false, renamed("revisionId", "id"), "revisionId"),
		reply:  projectObjectOf(revisionSpec),
	},
	domain.OpRevisionsList: {
		method: methodRevisionsList,
		body:   argsBody(true, nil, "documentId", "sort", "direction"),
		reply:  projectListOf(revisionRowSpec),
	},
	domain.OpSharesGet: {
		method: methodSharesInfo,
		body:   argsBody(false, renamed("shareId", "id"), "shareId"),
		reply:  projectShareInfo,
	},
	domain.OpSharesList: {
		method: methodSharesList,
		body:   argsBody(true, nil, "query", "sort", "direction"),
		reply:  projectListOf(shareSpec),
	},
	domain.OpUsersGet: {
		method: methodUsersInfo,
		body:   argsBody(false, renamed("userId", "id"), "userId"),
		reply:  projectObjectOf(userSpec),
	},
	domain.OpUsersList: {
		method: methodUsersList,
		body:   argsBody(true, nil, "query", "role", "filter", "ids", "sort", "direction"),
		reply:  projectListOf(userSpec),
	},
	domain.OpGroupsGet: {
		method: methodGroupsInfo,
		body:   argsBody(false, renamed("groupId", "id"), "groupId"),
		reply:  projectObjectOf(groupSpec),
	},
	domain.OpGroupsList: {
		method: methodGroupsList,
		body:   argsBody(true, renamed("groupSource", "source"), "userId", "groupSource", "query", "sort", "direction"),
		reply:  projectNestedOf("groups", groupSpec),
	},
	domain.OpGroupsMembers: {
		method: methodGroupsMemberships,
		body:   argsBody(true, renamed("groupId", "id"), "groupId", "query", "permission"),
		reply:  projectNestedOf("groupMemberships", groupUserSpec),
	},
	domain.OpTemplatesGet: {
		method: methodTemplatesInfo,
		body:   argsBody(false, renamed("templateId", "id"), "templateId"),
		reply:  projectObjectOf(templateSpec),
	},
	domain.OpTemplatesList: {
		method: methodTemplatesList,
		body:   argsBody(true, nil, "collectionId", "query", "sort", "direction"),
		reply:  projectListOf(templateSpec),
	},
	domain.OpPinsGet: {
		method: methodPinsInfo,
		body:   argsBody(false, nil, "documentId", "collectionId"),
		reply:  projectObjectOf(pinSpec),
	},
	domain.OpPinsList: {
		method: methodPinsList,
		body:   argsBody(true, nil, "collectionId"),
		reply:  projectNestedOf("pins", pinSpec),
	},
	domain.OpStarsList: {
		method: methodStarsList,
		body:   argsBody(true, nil),
		reply:  projectNestedOf("stars", starSpec),
	},
	domain.OpSubscriptionsGet: {
		method: methodSubscriptionsInfo,
		body:   argsBody(false, nil, "event", "documentId", "collectionId"),
		reply:  projectObjectOf(subscriptionSpec),
	},
	domain.OpSubscriptionsList: {
		method: methodSubscriptionsList,
		body:   argsBody(true, nil, "event", "documentId", "collectionId"),
		reply:  projectListOf(subscriptionSpec),
	},
	domain.OpAuthWhoami: {
		method: methodAuthInfo,
		body:   emptyBody,
		reply:  projectAccount,
	},
}

// exportEndpoint is the documents.export call. It is outside the catalog, so it has no row of its own, and it is used
// only by the vault export and by documents.get when the reply has no text.
var exportEndpoint = endpoint{
	method: methodDocumentsExport,
	body:   idBody("documentId"),
	reply:  projectExport,
}

// methodOf returns the Outline method of a catalog operation, or of the export call.
func methodOf(op domain.OperationID) (string, bool) {
	if op == opExport {
		return exportEndpoint.method, true
	}
	ep, ok := endpoints[op]
	return ep.method, ok
}

// idBody returns a body builder that sends the operation's ID argument as "id", which is Outline's name for it.
func idBody(param string) func(domain.Args) map[string]any {
	return func(a domain.Args) map[string]any {
		return map[string]any{"id": a.String(param)}
	}
}

// renamed maps one parameter name to the wire key Outline reads it under.
func renamed(name, wire string) map[string]string {
	return map[string]string{name: wire}
}

// argsBody returns a body builder that sends each named argument that is set, under its own name or, when renames maps
// it, under the wire name the map gives. An argument that is not set is left out, so the body names only what the
// caller chose. With paged, the body also carries the page: offset and limit, as Outline's pagination middleware reads
// them.
func argsBody(paged bool, renames map[string]string, names ...string) func(domain.Args) map[string]any {
	return func(a domain.Args) map[string]any {
		body := map[string]any{}
		if paged {
			body = pageBody(a)
		}
		values := a.Values()
		for _, name := range names {
			v, ok := values[name]
			if !ok {
				continue
			}
			key := name
			if wire, ok := renames[name]; ok {
				key = wire
			}
			body[key] = v
		}
		return body
	}
}

// pageBody returns the page of a list operation: offset and limit, which Outline's pagination middleware reads.
func pageBody(a domain.Args) map[string]any {
	return map[string]any{"offset": a.Int("offset"), "limit": a.Int("limit")}
}

// documentListBody is the body of documents.list. The collection filter is sent only when a collection is named.
func documentListBody(a domain.Args) map[string]any {
	body := pageBody(a)
	if a.Has("collectionId") {
		body["collectionId"] = a.String("collectionId")
	}
	return body
}

// searchBody is the body of documents.search.
func searchBody(a domain.Args) map[string]any {
	body := pageBody(a)
	body["query"] = a.String("query")
	return body
}

// ServedOperations returns the catalog operations that the endpoint tables serve, the reads and the writes, in no
// particular order. The catalog sync test checks that the tables serve every row of the catalog and nothing else.
func ServedOperations() []domain.OperationID {
	out := make([]domain.OperationID, 0, len(endpoints)+len(writeEndpoints)+len(adminEndpoints))
	for op := range endpoints {
		out = append(out, op)
	}
	for op := range writeEndpoints {
		out = append(out, op)
	}
	for op := range adminEndpoints {
		out = append(out, op)
	}
	return out
}

// Methods returns the Outline method of every served operation and of the export call, in no particular order. Each
// one is a method the client may call, so the sync test checks this list against the allowed methods.
func Methods() []string {
	out := []string{exportEndpoint.method}
	for _, ep := range endpoints {
		out = append(out, ep.method)
	}
	for _, ep := range writeEndpoints {
		out = append(out, ep.method)
	}
	for _, ep := range adminEndpoints {
		out = append(out, ep.method)
	}
	return out
}

// emptyBody is the empty body of auth.info.
func emptyBody(domain.Args) map[string]any {
	return map[string]any{}
}
