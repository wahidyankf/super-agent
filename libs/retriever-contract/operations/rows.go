package operations

import "github.com/wahidyankf/super-agent/libs/retriever-contract/domain"

// rows is the operation catalog of stage a: the thirty-eight reads of part 1 (seven) and part 2a (thirty-one). Every row
// is a read. A row is added here, with one endpoint entry in the Outline
// adapter, one request case in its endpoint test, and its scenario rows.
var rows = []Row{
	{
		ID:          domain.OpCollectionsList,
		CLI:         "outline collections list",
		Tool:        "list_collections",
		Summary:     "list the collections of the Outline wiki",
		Description: "Lists the collections of one wiki source. Pass source when more than one source of a kind exists.",
		Effect:      domain.EffectRead,
		OpensWorld:  true,
		Fields:      []string{"id", "name"},
		ListKey:     "collections",
		Noun:        "collections",
		Columns:     []string{"id", "name"},
	},
	{
		ID:          domain.OpCollectionsInfo,
		CLI:         "outline collections info",
		Tool:        "collection_info",
		Summary:     "show the ID and name of one Outline collection",
		Description: "Returns the ID and the name of one collection of a wiki source.",
		Effect:      domain.EffectRead,
		OpensWorld:  true,
		Params: []Param{
			{Name: "collectionId", Kind: KindCollectionID, Required: true, Help: "the collection ID, from list_collections"},
		},
		Fields: []string{"id", "name"},
	},
	{
		ID:          domain.OpCollectionsDocuments,
		CLI:         "outline collections documents",
		Tool:        "collection_tree",
		Summary:     "show the document tree of one Outline collection",
		Description: "Lists the documents of one collection as a flat tree. Each node names its parent and its depth; a root document has no parent.",
		Effect:      domain.EffectRead,
		OpensWorld:  true,
		Params: []Param{
			{Name: "collectionId", Kind: KindCollectionID, Required: true, Help: "the collection ID, from list_collections"},
		},
		Fields:  []string{"id", "title", "url", "parentId", "depth"},
		ListKey: "nodes",
		Noun:    "documents",
		Columns: []string{"title", "id", "url"},
		Indent:  "depth",
	},
	{
		ID:          domain.OpDocumentsGet,
		CLI:         "outline documents get",
		Tool:        "get_document",
		Summary:     "print the Markdown of one Outline document",
		Description: "Returns the Markdown of one document with its title and URL.",
		Effect:      domain.EffectRead,
		OpensWorld:  true,
		Params: []Param{
			{Name: "documentId", Kind: KindDocumentID, Required: true, Help: "the document ID, from collection_tree"},
		},
		// The revision is the number documents.update names as last_revision.
		Fields: []string{"id", "title", "url", "collectionId", "revision", "markdown"},
		Body:   "markdown",
	},
	{
		ID:          domain.OpDocumentsList,
		CLI:         "outline documents list",
		Tool:        "list_documents",
		Summary:     "list one page of Outline documents",
		Description: "Lists one page of documents, optionally of one collection. The page holds offset and limit; the limit is from 1 to 100 and defaults to 25.",
		Effect:      domain.EffectRead,
		OpensWorld:  true,
		Params: []Param{
			{Name: "collectionId", Kind: KindCollectionUUID, Help: "only documents of this collection, by its UUID"},
			{Name: "offset", Kind: KindOffset, Default: 0, Help: "the first document to return; 0 when omitted"},
			{Name: "limit", Kind: KindLimit, Default: domain.DefaultPageLimit, Help: "documents per page, from 1 to 100; 25 when omitted"},
		},
		Fields:  []string{"id", "title", "url", "collectionId"},
		ListKey: "documents",
		Noun:    "documents",
		Columns: []string{"id", "title"},
	},
	{
		ID:          domain.OpDocumentsSearch,
		CLI:         "outline documents search",
		Tool:        "search_documents",
		Summary:     "search Outline documents by title and text, one page at a time",
		Description: "Searches the documents by title and text and returns one page of the matches. The query must not be empty; the limit is from 1 to 100 and defaults to 25.",
		Effect:      domain.EffectRead,
		OpensWorld:  true,
		Params: []Param{
			{Name: "query", Kind: KindQuery, Required: true, Help: "the text to find in the document titles and texts"},
			{Name: "offset", Kind: KindOffset, Default: 0, Help: "the first match to return; 0 when omitted"},
			{Name: "limit", Kind: KindLimit, Default: domain.DefaultPageLimit, Help: "matches per page, from 1 to 100; 25 when omitted"},
		},
		Fields:  []string{"id", "title", "url", "collectionId"},
		ListKey: "documents",
		Noun:    "documents",
		Columns: []string{"id", "title"},
		NoMatch: true,
	},
	{
		ID:          domain.OpAuthWhoami,
		CLI:         "outline auth whoami",
		Tool:        "whoami",
		Summary:     "show the user and team that the Outline API key belongs to",
		Description: "Returns the user and the team that the Outline API key of a source belongs to. The user's e-mail address is never returned.",
		Effect:      domain.EffectRead,
		OpensWorld:  true,
		Fields:      []string{"user", "team"},
	},
	{
		ID:          domain.OpDocumentsArchived,
		CLI:         "outline documents archived",
		Tool:        "list_archived_documents",
		Summary:     "list one page of archived Outline documents",
		Description: "Lists one page of the archived documents, optionally of one collection. The page holds offset and limit; the limit is from 1 to 100 and defaults to 25.",
		Effect:      domain.EffectRead,
		OpensWorld:  true,
		Params:      withPage(collectionParam(), sortParam(documentSorts...), directionParam()),
		Fields:      docListFields,
		ListKey:     "documents",
		Noun:        "documents",
		Columns:     []string{"id", "title"},
	},
	{
		ID:          domain.OpDocumentsChildren,
		CLI:         "outline documents children",
		Tool:        "document_children",
		Summary:     "show the documents below one Outline document, as a tree",
		Description: "Lists the child documents of one document as a flat tree. A child names the document as its parent, and its depth counts from 0.",
		Effect:      domain.EffectRead,
		OpensWorld:  true,
		Params: []Param{
			{Name: "documentId", Kind: KindDocumentID, Required: true, Help: "the document ID, from list_documents"},
		},
		Fields:  treeFields,
		ListKey: "nodes",
		Noun:    "documents",
		Columns: []string{"title", "id", "url"},
		Indent:  "depth",
	},
	{
		ID:          domain.OpDocumentsDeleted,
		CLI:         "outline documents deleted",
		Tool:        "list_deleted_documents",
		Summary:     "list one page of deleted Outline documents",
		Description: "Lists one page of the documents in the trash. The page holds offset and limit; the limit is from 1 to 100 and defaults to 25.",
		Effect:      domain.EffectRead,
		OpensWorld:  true,
		Params:      withPage(sortParam(documentSorts...), directionParam()),
		Fields:      docListFields,
		ListKey:     "documents",
		Noun:        "documents",
		Columns:     []string{"id", "title"},
	},
	{
		ID:          domain.OpDocumentsDrafts,
		CLI:         "outline documents drafts",
		Tool:        "list_draft_documents",
		Summary:     "list one page of Outline draft documents",
		Description: "Lists one page of the draft documents, optionally of one collection. The page holds offset and limit; the limit is from 1 to 100 and defaults to 25.",
		Effect:      domain.EffectRead,
		OpensWorld:  true,
		Params:      withPage(collectionParam(), sortParam(documentSorts...), directionParam()),
		Fields:      docListFields,
		ListKey:     "documents",
		Noun:        "documents",
		Columns:     []string{"id", "title"},
	},
	{
		ID:          domain.OpDocumentsGroupMemberships,
		CLI:         "outline documents group-memberships",
		Tool:        "list_document_group_memberships",
		Summary:     "list one page of the groups that have access to one Outline document",
		Description: "Lists one page of the group memberships of one document, optionally filtered by a name query or a permission.",
		Effect:      domain.EffectRead,
		OpensWorld:  true,
		Params: withPage(
			documentIDParam("the document ID, from list_documents"),
			queryParam("the group name to search for"),
			permissionParam(documentPermissions...),
		),
		Fields:  groupMembershipFields,
		ListKey: "groupMemberships",
		Noun:    "group memberships",
		Columns: []string{"id", "groupId", "permission"},
	},
	{
		ID:          domain.OpDocumentsInsights,
		CLI:         "outline documents insights",
		Tool:        "get_document_insights",
		Summary:     "show the daily view and edit counts of one Outline document",
		Description: "Returns one row per day of the insights of one document, for the window from startDate to endDate. The default window is the last 30 days, and the server must have insights enabled.",
		Effect:      domain.EffectRead,
		OpensWorld:  true,
		Params: []Param{
			{Name: "documentId", Kind: KindDocumentID, Required: true, Help: "the document ID, from list_documents"},
			{Name: "startDate", Kind: KindTime, Help: "the first day of the window, RFC 3339 with a zone"},
			{Name: "endDate", Kind: KindTime, Help: "the last day of the window, RFC 3339 with a zone"},
		},
		Fields:  insightFields,
		ListKey: "insights",
		Noun:    "insights",
		Columns: []string{"date", "period", "viewCount"},
	},
	{
		ID:          domain.OpDocumentsMemberships,
		CLI:         "outline documents memberships",
		Tool:        "list_document_memberships",
		Summary:     "list one page of the users who have direct access to one Outline document",
		Description: "Lists one page of the memberships of one document, optionally filtered by a name query or a permission.",
		Effect:      domain.EffectRead,
		OpensWorld:  true,
		Params: withPage(
			documentIDParam("the document ID, from list_documents"),
			queryParam("the user name to search for"),
			permissionParam(documentPermissions...),
		),
		Fields:  membershipFields,
		ListKey: "memberships",
		Noun:    "memberships",
		Columns: []string{"id", "userId", "permission"},
	},
	{
		ID:          domain.OpDocumentsSearchTitles,
		CLI:         "outline documents search-titles",
		Tool:        "search_document_titles",
		Summary:     "search Outline document titles, one page at a time",
		Description: "Returns one page of the documents whose title matches the query. The query must not be empty; the limit is from 1 to 100 and defaults to 25.",
		Effect:      domain.EffectRead,
		OpensWorld:  true,
		Params: withPage(
			Param{Name: "query", Kind: KindQuery, Required: true, Help: "the text to find in the document titles; it must not be blank"},
			collectionParam(),
			uuidParam("documentId", "only documents below this document, by its UUID"),
			uuidParam("userId", "only documents of this user, by the user's UUID"),
			statusFilterParam(),
			sortParam("createdAt", "updatedAt", "title"),
			directionParam(),
		),
		Fields:  docListFields,
		ListKey: "documents",
		Noun:    "documents",
		Columns: []string{"id", "title"},
		NoMatch: true,
	},
	{
		ID:          domain.OpDocumentsUsers,
		CLI:         "outline documents users",
		Tool:        "list_document_users",
		Summary:     "list one page of the users who can see one Outline document",
		Description: "Lists one page of the users who have access to one document, optionally filtered by a name query or by one user. The query matches names only, never e-mail addresses, and no e-mail address is ever returned.",
		Effect:      domain.EffectRead,
		OpensWorld:  true,
		Params: withPage(
			documentIDParam("the document ID, from list_documents"),
			queryParam("the user name to search for"),
			uuidParam("userId", "only this user, by the user's UUID"),
		),
		Fields:  userFields,
		ListKey: "users",
		Noun:    "users",
		Columns: []string{"id", "name", "role"},
	},
	{
		ID:          domain.OpDocumentsViewed,
		CLI:         "outline documents viewed",
		Tool:        "list_viewed_documents",
		Summary:     "list one page of the Outline documents the API key's user has viewed",
		Description: "Lists one page of the documents the user of the API key has viewed, sorted by the time of the view.",
		Effect:      domain.EffectRead,
		OpensWorld:  true,
		Params:      withPage(sortParam("updatedAt", "createdAt"), directionParam()),
		Fields:      docListFields,
		ListKey:     "documents",
		Noun:        "documents",
		Columns:     []string{"id", "title"},
	},
	{
		ID:          domain.OpCollectionsMemberships,
		CLI:         "outline collections memberships",
		Tool:        "list_collection_memberships",
		Summary:     "list one page of the users who have direct access to one Outline collection",
		Description: "Lists one page of the memberships of one collection, optionally filtered by a name query or a permission.",
		Effect:      domain.EffectRead,
		OpensWorld:  true,
		Params: withPage(
			collectionIDParam(),
			queryParam("the user name to search for"),
			permissionParam(documentPermissions...),
		),
		Fields:  membershipFields,
		ListKey: "memberships",
		Noun:    "memberships",
		Columns: []string{"id", "userId", "permission"},
	},
	{
		ID:          domain.OpCollectionsGroupMembers,
		CLI:         "outline collections group-memberships",
		Tool:        "list_collection_group_memberships",
		Summary:     "list one page of the groups that have access to one Outline collection",
		Description: "Lists one page of the group memberships of one collection, optionally filtered by a name query or a permission.",
		Effect:      domain.EffectRead,
		OpensWorld:  true,
		Params: withPage(
			collectionIDParam(),
			queryParam("the group name to search for"),
			permissionParam(documentPermissions...),
		),
		Fields:  groupMembershipFields,
		ListKey: "groupMemberships",
		Noun:    "group memberships",
		Columns: []string{"id", "groupId", "permission"},
	},
	{
		ID:          domain.OpCommentsGet,
		CLI:         "outline comments get",
		Tool:        "get_comment",
		Summary:     "show one Outline comment as plain text",
		Description: "Returns one comment of a document, with its author, its resolution, and its text as plain text. The text is the comment's words, without any formatting.",
		Effect:      domain.EffectRead,
		OpensWorld:  true,
		Params: []Param{
			{Name: "commentId", Kind: KindUUID, Required: true, Help: "the comment ID, from list_comments"},
			{Name: "includeAnchorText", Kind: KindBool, Help: "true to include the text the comment is anchored to"},
		},
		Fields: commentFields,
	},
	{
		ID:          domain.OpCommentsList,
		CLI:         "outline comments list",
		Tool:        "list_comments",
		Summary:     "list one page of Outline comments as plain text",
		Description: "Lists one page of comments, optionally of one document, one collection, or one parent comment. Each comment carries its text as plain text. The page holds offset and limit; the limit is from 1 to 100 and defaults to 25.",
		Effect:      domain.EffectRead,
		OpensWorld:  true,
		Params: withPage(
			Param{Name: "documentId", Kind: KindID, Help: "only comments of this document, by its ID"},
			Param{Name: "collectionId", Kind: KindID, Help: "only comments of this collection, by its ID"},
			uuidParam("parentCommentId", "only replies to this comment, by its UUID"),
			Param{Name: "statusFilter", Kind: KindEnumList, Values: []string{"resolved", "unresolved"}, Help: "only resolved, or only unresolved, comments; comma-separated"},
			Param{Name: "includeAnchorText", Kind: KindBool, Help: "true to include the text the comments are anchored to"},
			sortParam("createdAt", "updatedAt"),
			directionParam(),
		),
		Fields:  commentFields,
		ListKey: "comments",
		Noun:    "comments",
		Columns: []string{"id", "documentId", "createdAt"},
	},
	{
		ID:          domain.OpRevisionsExport,
		CLI:         "outline revisions export",
		Summary:     "print the Markdown of one revision of an Outline document",
		Description: "Returns the Markdown of one revision of a document. It reads the revision only and writes no file.",
		Effect:      domain.EffectRead,
		OpensWorld:  true,
		Params: []Param{
			{Name: "revisionId", Kind: KindUUID, Required: true, Help: "the revision ID, from list_revisions"},
		},
		Fields: []string{"markdown"},
		Body:   "markdown",
	},
	{
		ID:          domain.OpRevisionsGet,
		CLI:         "outline revisions get",
		Tool:        "get_revision",
		Summary:     "print the Markdown of one revision of an Outline document",
		Description: "Returns one revision of a document with its title, name, and Markdown text. The text is cut into pages of at most 32768 bytes.",
		Effect:      domain.EffectRead,
		OpensWorld:  true,
		Params: []Param{
			{Name: "revisionId", Kind: KindUUID, Required: true, Help: "the revision ID, from list_revisions"},
		},
		Fields: revisionFields,
		Body:   "text",
	},
	{
		ID:          domain.OpRevisionsList,
		CLI:         "outline revisions list",
		Tool:        "list_revisions",
		Summary:     "list one page of the revisions of one Outline document",
		Description: "Lists one page of the revisions of one document, without their text. The page holds offset and limit; the limit is from 1 to 100 and defaults to 25.",
		Effect:      domain.EffectRead,
		OpensWorld:  true,
		Params: withPage(
			uuidParamRequired("documentId", "the document UUID, from list_documents"),
			sortParam("createdAt", "updatedAt"),
			directionParam(),
		),
		Fields:  revisionRowFields,
		ListKey: "revisions",
		Noun:    "revisions",
		Columns: []string{"id", "title", "createdAt"},
	},
	{
		ID:          domain.OpSharesGet,
		CLI:         "outline shares get",
		Tool:        "get_share",
		Summary:     "show one Outline share, without its link",
		Description: "Returns one share of a document or a collection, with its settings. The public link of the share is never returned.",
		Effect:      domain.EffectRead,
		OpensWorld:  true,
		Params: []Param{
			{Name: "shareId", Kind: KindShareID, Required: true, Help: "the share ID, from list_shares"},
		},
		Fields: shareFields,
	},
	{
		ID:          domain.OpSharesList,
		CLI:         "outline shares list",
		Tool:        "list_shares",
		Summary:     "list one page of Outline shares, without their links",
		Description: "Lists one page of the shares the API key's user may read. No public link is returned. The page holds offset and limit; the limit is from 1 to 100 and defaults to 25.",
		Effect:      domain.EffectRead,
		OpensWorld:  true,
		Params: withPage(
			queryParam("the text to find in the share titles and documents"),
			sortParam("createdAt", "updatedAt", "lastAccessedAt", "views"),
			directionParam(),
		),
		Fields:  shareFields,
		ListKey: "shares",
		Noun:    "shares",
		Columns: []string{"id", "documentTitle", "published"},
	},
	{
		ID:          domain.OpUsersGet,
		CLI:         "outline users get",
		Tool:        "get_user",
		Summary:     "show one Outline user, without the e-mail address",
		Description: "Returns one user of the wiki: the name, the role, and the activity times. Without a user ID it returns the user of the API key. The e-mail address is never returned.",
		Effect:      domain.EffectRead,
		OpensWorld:  true,
		Params: []Param{
			{Name: "userId", Kind: KindUUID, Help: "the user ID; omitted means the user of the API key"},
		},
		Fields: userFields,
	},
	{
		ID:          domain.OpUsersList,
		CLI:         "outline users list",
		Tool:        "list_users",
		Summary:     "list one page of Outline users, without e-mail addresses",
		Description: "Lists one page of the users, optionally filtered by a name query, a role, a status, or a list of IDs. The page holds offset and limit; the limit is from 1 to 100 and defaults to 25. Where the team's e-mail display setting lets the caller read addresses, the query also matches e-mail addresses, so a result can confirm that an address belongs to a user; no e-mail address is ever returned.",
		Effect:      domain.EffectRead,
		OpensWorld:  true,
		Params: withPage(
			queryParam("the user name to search for"),
			Param{Name: "role", Kind: KindEnum, Values: userRoles, Help: "only users with this role"},
			Param{Name: "filter", Kind: KindEnum, Values: userFilters, Help: "only users with this status"},
			Param{Name: "ids", Kind: KindUUIDList, Help: "only these users, by their UUIDs; comma-separated"},
			sortParam("createdAt", "updatedAt", "name", "lastActiveAt"),
			directionParam(),
		),
		Fields:  userFields,
		ListKey: "users",
		Noun:    "users",
		Columns: []string{"id", "name", "role"},
	},
	{
		ID:          domain.OpGroupsGet,
		CLI:         "outline groups get",
		Tool:        "get_group",
		Summary:     "show one Outline group",
		Description: "Returns one group of the wiki: its name, its description, and its member count. The identity-provider link of the group is never returned.",
		Effect:      domain.EffectRead,
		OpensWorld:  true,
		Params: []Param{
			{Name: "groupId", Kind: KindUUID, Required: true, Help: "the group ID, from list_groups"},
		},
		Fields: groupFields,
	},
	{
		ID:          domain.OpGroupsList,
		CLI:         "outline groups list",
		Tool:        "list_groups",
		Summary:     "list one page of Outline groups",
		Description: "Lists one page of the groups, optionally of one user, of one source, or matching a name query. The page holds offset and limit; the limit is from 1 to 100 and defaults to 25.",
		Effect:      domain.EffectRead,
		OpensWorld:  true,
		Params: withPage(
			uuidParam("userId", "only groups of this user, by the user's UUID"),
			Param{Name: "groupSource", Kind: KindName, Help: "only groups of this source: manual, or the name of an identity provider"},
			queryParam("the group name to search for"),
			sortParam("createdAt", "updatedAt", "name"),
			directionParam(),
		),
		Fields:  groupFields,
		ListKey: "groups",
		Noun:    "groups",
		Columns: []string{"id", "name", "memberCount"},
	},
	{
		ID:          domain.OpGroupsMembers,
		CLI:         "outline groups members",
		Tool:        "list_group_members",
		Summary:     "list one page of the members of one Outline group",
		Description: "Lists one page of the members of one group with their permission, optionally filtered by a permission. The page holds offset and limit; the limit is from 1 to 100 and defaults to 25.",
		Effect:      domain.EffectRead,
		OpensWorld:  true,
		Params: withPage(
			uuidParamRequired("groupId", "the group ID, from list_groups"),
			queryParam("the group member name to search for"),
			Param{Name: "permission", Kind: KindEnum, Values: groupPermissions, Help: "only members with this permission"},
		),
		Fields:  groupUserFields,
		ListKey: "groupMemberships",
		Noun:    "members",
		Columns: []string{"userId", "permission", "createdAt"},
	},
	{
		ID:          domain.OpTemplatesGet,
		CLI:         "outline templates get",
		Tool:        "get_template",
		Summary:     "show one Outline template, without its body",
		Description: "Returns one template: its title, its collection, and its icon and color. The body of a template is editor data, and it is never returned.",
		Effect:      domain.EffectRead,
		OpensWorld:  true,
		Params: []Param{
			{Name: "templateId", Kind: KindID, Required: true, Help: "the template ID, from list_templates"},
		},
		Fields: templateFields,
	},
	{
		ID:          domain.OpTemplatesList,
		CLI:         "outline templates list",
		Tool:        "list_templates",
		Summary:     "list one page of Outline templates, without their bodies",
		Description: "Lists one page of the templates, optionally of one collection or matching a title query. The page holds offset and limit; the limit is from 1 to 100 and defaults to 25.",
		Effect:      domain.EffectRead,
		OpensWorld:  true,
		Params: withPage(
			collectionParam(),
			queryParam("the template title to search for"),
			sortParam("createdAt", "updatedAt", "publishedAt", "title", "collectionId"),
			directionParam(),
		),
		Fields:  templateFields,
		ListKey: "templates",
		Noun:    "templates",
		Columns: []string{"id", "title", "collectionId"},
	},
	{
		ID:          domain.OpPinsGet,
		CLI:         "outline pins get",
		Tool:        "get_pin",
		Summary:     "show the pin of one Outline document",
		Description: "Returns the pin of one document in one collection. Without a collection it returns the pin of the document on Home.",
		Effect:      domain.EffectRead,
		OpensWorld:  true,
		Params: []Param{
			{Name: "documentId", Kind: KindDocumentID, Required: true, Help: "the pinned document ID, from list_documents"},
			optionalCollectionParam("the collection of the pin, by its UUID; Home when omitted"),
		},
		Fields: pinFields,
	},
	{
		ID:          domain.OpPinsList,
		CLI:         "outline pins list",
		Tool:        "list_pins",
		Summary:     "list one page of the Outline pins of a collection or of Home",
		Description: "Lists one page of the pins of one collection. Without a collection it lists the pins on Home. The page holds offset and limit; the limit is from 1 to 100 and defaults to 25.",
		Effect:      domain.EffectRead,
		OpensWorld:  true,
		Params:      withPage(optionalCollectionParam("the collection of the pins, by its UUID; Home when omitted")),
		Fields:      pinFields,
		ListKey:     "pins",
		Noun:        "pins",
		Columns:     []string{"id", "documentId", "index"},
	},
	{
		ID:          domain.OpStarsList,
		CLI:         "outline stars list",
		Tool:        "list_stars",
		Summary:     "list one page of the Outline stars of the API key's user",
		Description: "Lists one page of the documents and collections the user of the API key has starred. The page holds offset and limit; the limit is from 1 to 100 and defaults to 25.",
		Effect:      domain.EffectRead,
		OpensWorld:  true,
		Params:      withPage(),
		Fields:      starFields,
		ListKey:     "stars",
		Noun:        "stars",
		Columns:     []string{"id", "documentId", "index"},
	},
	{
		ID:          domain.OpSubscriptionsGet,
		CLI:         "outline subscriptions get",
		Tool:        "get_subscription",
		Summary:     "show the subscription of the API key's user to one Outline document or collection",
		Description: "Returns the subscription to document updates of the user of the API key, for one document or one collection. Exactly one of documentId and collectionId is required.",
		Effect:      domain.EffectRead,
		OpensWorld:  true,
		Params:      subscriptionParams(),
		OneOf:       []string{"documentId", "collectionId"},
		Fields:      subscriptionFields,
	},
	{
		ID:          domain.OpSubscriptionsList,
		CLI:         "outline subscriptions list",
		Tool:        "list_subscriptions",
		Summary:     "list one page of the Outline subscriptions to one document or collection",
		Description: "Lists one page of the subscriptions to document updates for one document or one collection. Exactly one of documentId and collectionId is required. The page holds offset and limit; the limit is from 1 to 100 and defaults to 25.",
		Effect:      domain.EffectRead,
		OpensWorld:  true,
		Params:      withPage(subscriptionParams()...),
		OneOf:       []string{"documentId", "collectionId"},
		Fields:      subscriptionFields,
		ListKey:     "subscriptions",
		Noun:        "subscriptions",
		Columns:     []string{"id", "documentId", "event"},
	},
}

// The field sets of the stage a part 2a reads. Each one is the projection of the Outline object, in the order of the
// reference. The Outline adapter returns exactly these names.
var (
	docListFields         = []string{"id", "title", "url", "collectionId"}
	treeFields            = []string{"id", "title", "url", "parentId", "depth"}
	membershipFields      = []string{"id", "userId", "documentId", "collectionId", "permission", "createdById", "sourceId", "index"}
	groupMembershipFields = []string{"id", "groupId", "documentId", "collectionId", "permission", "sourceId"}
	groupUserFields       = []string{"groupId", "userId", "permission", "createdAt"}
	insightFields         = []string{"date", "period", "viewCount", "viewerCount", "commentCount", "reactionCount", "revisionCount", "editorCount"}
	userFields            = []string{"id", "name", "role", "isSuspended", "lastActiveAt", "createdAt", "updatedAt", "deletedAt"}
	groupFields           = []string{"id", "name", "description", "color", "icon", "memberCount", "disableMentions", "createdAt", "updatedAt"}
	commentFields         = []string{"id", "documentId", "parentCommentId", "createdById", "resolvedById", "resolvedAt", "createdAt", "updatedAt", "text", "anchorText"}
	revisionFields        = []string{"id", "documentId", "title", "name", "text", "icon", "color", "createdAt", "createdById", "deletedAt"}
	revisionRowFields     = []string{"id", "documentId", "title", "name", "icon", "color", "createdAt", "createdById", "deletedAt"}
	shareFields           = []string{"id", "documentId", "collectionId", "documentTitle", "sourceTitle", "published", "includeChildDocuments", "allowIndexing", "allowSubscriptions", "showLastUpdated", "showTOC", "title", "domain", "views", "lastAccessedAt", "createdById", "createdAt", "updatedAt"}
	templateFields        = []string{"id", "urlId", "url", "title", "icon", "color", "fullWidth", "collectionId", "createdAt", "updatedAt", "publishedAt", "deletedAt", "createdById", "updatedById"}
	pinFields             = []string{"id", "documentId", "collectionId", "index", "createdAt", "updatedAt"}
	starFields            = []string{"id", "documentId", "collectionId", "index", "createdAt", "updatedAt"}
	subscriptionFields    = []string{"id", "userId", "documentId", "collectionId", "event", "createdAt", "updatedAt"}
)

// The closed lists of the stage a part 2a parameters, from the v1.12.0 server.
var (
	documentSorts       = []string{"createdAt", "updatedAt", "publishedAt", "index", "title", "popularityScore"}
	documentPermissions = []string{"read", "read_write", "admin"}
	groupPermissions    = []string{"member", "admin"}
	userRoles           = []string{"admin", "member", "viewer", "guest"}
	userFilters         = []string{"invited", "viewers", "admins", "members", "active", "all", "suspended"}
	documentStatuses    = []string{"published", "archived", "draft"}
	subscriptionEvents  = []string{"documents.update"}
)

// withPage returns the parameters followed by the page parameters, offset and limit, in that order.
func withPage(params ...Param) []Param {
	return append(params, Param{Name: "offset", Kind: KindOffset, Default: 0, Help: "the first item to return; 0 when omitted"},
		Param{Name: "limit", Kind: KindLimit, Default: domain.DefaultPageLimit, Help: "items per page, from 1 to 100; 25 when omitted"})
}

// collectionParam is the optional collection filter of a list, by its UUID.
func collectionParam() Param {
	return Param{Name: "collectionId", Kind: KindCollectionUUID, Help: "only documents of this collection, by its UUID"}
}

// collectionIDParam is the required collection of a read of one collection's members, by its ID.
func collectionIDParam() Param {
	return Param{Name: "collectionId", Kind: KindCollectionID, Required: true, Help: "the collection ID, from list_collections"}
}

// documentIDParam is the required document of a read of one document's members, by its ID.
func documentIDParam(help string) Param {
	return Param{Name: "documentId", Kind: KindDocumentID, Required: true, Help: help}
}

// uuidParam is an optional UUID parameter.
func uuidParam(name, help string) Param {
	return Param{Name: name, Kind: KindUUID, Help: help}
}

// uuidParamRequired is a required UUID parameter.
func uuidParamRequired(name, help string) Param {
	return Param{Name: name, Kind: KindUUID, Required: true, Help: help}
}

// queryParam is an optional text to search for, from 1 to 1000 characters.
func queryParam(help string) Param {
	return Param{Name: "query", Kind: KindQuery, Help: help}
}

// sortParam is an optional sort field from a closed list.
func sortParam(values ...string) Param {
	return Param{Name: "sort", Kind: KindEnum, Values: values, Help: "the field to sort by; the server's default when omitted"}
}

// directionParam is the optional sort direction, ASC or DESC.
func directionParam() Param {
	return Param{Name: "direction", Kind: KindEnum, Values: []string{"ASC", "DESC"}, Help: "ASC or DESC; DESC when omitted"}
}

// permissionParam is an optional permission from a closed list.
func permissionParam(values ...string) Param {
	return Param{Name: "permission", Kind: KindEnum, Values: values, Help: "only items with this permission"}
}

// statusFilterParam is an optional list of document statuses, comma-separated on the command line.
func statusFilterParam() Param {
	return Param{Name: "statusFilter", Kind: KindEnumList, Values: documentStatuses, Help: "only published, archived, or draft documents; comma-separated"}
}

// subscriptionParams are the parameters that select one subscription target. The event is fixed by the server, so it
// defaults to documents.update.
func subscriptionParams() []Param {
	return []Param{
		{Name: "event", Kind: KindEnum, Values: subscriptionEvents, Default: "documents.update", Help: "the event to follow; documents.update is the only one"},
		optionalDocumentParam("the document to follow, by its ID"),
		optionalCollectionParam("the collection to follow, by its UUID"),
	}
}

// optionalDocumentParam is an optional document, by its ID.
func optionalDocumentParam(help string) Param {
	return Param{Name: "documentId", Kind: KindDocumentID, Help: help}
}

// optionalCollectionParam is an optional collection, by its UUID.
func optionalCollectionParam(help string) Param {
	return Param{Name: "collectionId", Kind: KindCollectionUUID, Help: help}
}
