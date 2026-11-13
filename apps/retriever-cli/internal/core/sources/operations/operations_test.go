package operations

import (
	"errors"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

var toolPattern = regexp.MustCompile(`^[a-z]+(_[a-z]+)*$`)

// rowOf returns the catalog row of id, failing the test when the catalog does not hold it.
func rowOf(t *testing.T, id domain.OperationID) Row {
	t.Helper()
	row, ok := Lookup(id)
	if !ok {
		t.Fatalf("the catalog holds no row for %s", id)
	}
	return row
}

// stageAReads is every read of stage a: the seven built reads of part 1, the thirty-one reads of part 2a, the eighteen
// reads of part 2b (access read), then the ten admin-level reads of part 2b. The order is the catalog order. The vault
// export (documents.export) is not a row, so it is not here.
var stageAReads = []domain.OperationID{
	domain.OpCollectionsList, domain.OpCollectionsInfo, domain.OpCollectionsDocuments,
	domain.OpDocumentsGet, domain.OpDocumentsList, domain.OpDocumentsSearch, domain.OpAuthWhoami,
	domain.OpDocumentsArchived, domain.OpDocumentsChildren, domain.OpDocumentsDeleted, domain.OpDocumentsDrafts,
	domain.OpDocumentsGroupMemberships, domain.OpDocumentsInsights, domain.OpDocumentsMemberships,
	domain.OpDocumentsSearchTitles, domain.OpDocumentsUsers, domain.OpDocumentsViewed,
	domain.OpCollectionsMemberships, domain.OpCollectionsGroupMembers,
	domain.OpCommentsGet, domain.OpCommentsList,
	domain.OpRevisionsExport, domain.OpRevisionsGet, domain.OpRevisionsList,
	domain.OpSharesGet, domain.OpSharesList,
	domain.OpUsersGet, domain.OpUsersList,
	domain.OpGroupsGet, domain.OpGroupsList, domain.OpGroupsMembers,
	domain.OpTemplatesGet, domain.OpTemplatesList,
	domain.OpPinsGet, domain.OpPinsList, domain.OpStarsList,
	domain.OpSubscriptionsGet, domain.OpSubscriptionsList,
	domain.OpAccessRequestsGet, domain.OpAttachmentsList, domain.OpEmojisGet, domain.OpEmojisList,
	domain.OpEventsList, domain.OpFileOperationsGet, domain.OpFileOperationsList,
	domain.OpGroupMembershipsList, domain.OpInstallationGet, domain.OpNotificationsList,
	domain.OpOAuthAuthenticationsList, domain.OpReactionsList, domain.OpRelationshipsGet,
	domain.OpRelationshipsList, domain.OpSearchesList, domain.OpSuggestionsMention,
	domain.OpUserMembershipsList, domain.OpViewsList,
	domain.OpAPIKeysList, domain.OpAuthenticationProvidersGet, domain.OpAuthenticationProvidersList,
	domain.OpImportsGet, domain.OpImportsList, domain.OpIntegrationsGet, domain.OpIntegrationsList,
	domain.OpOAuthClientsGet, domain.OpOAuthClientsList, domain.OpWebhookSubscriptionsList,
}

// stageAAdminReads is the ten reads that need an admin source.
var stageAAdminReads = []domain.OperationID{
	domain.OpAPIKeysList, domain.OpAuthenticationProvidersGet, domain.OpAuthenticationProvidersList,
	domain.OpImportsGet, domain.OpImportsList, domain.OpIntegrationsGet, domain.OpIntegrationsList,
	domain.OpOAuthClientsGet, domain.OpOAuthClientsList, domain.OpWebhookSubscriptionsList,
}

// stageBWrites is the seven documents writes of stage b, class W, in catalog order. They follow the
// stage a reads.
var stageBWrites = []domain.OperationID{
	domain.OpDocumentsCreate, domain.OpDocumentsDuplicate, domain.OpDocumentsMove, domain.OpDocumentsRestore,
	domain.OpDocumentsTemplatize, domain.OpDocumentsUnpublish, domain.OpDocumentsUpdate,
	domain.OpCollectionsCreate, domain.OpCollectionsDuplicate, domain.OpCollectionsExport, domain.OpCollectionsMove,
	domain.OpCollectionsRestore, domain.OpCollectionsUpdate,
	domain.OpCommentsCreate, domain.OpCommentsUpdate, domain.OpCommentsResolve, domain.OpCommentsUnresolve,
	domain.OpCommentsAddReaction, domain.OpCommentsRemoveReaction,
	domain.OpAccessRequestsCreate,
	domain.OpRevisionsUpdate,
	domain.OpNotificationsUpdate, domain.OpNotificationsUpdateAll,
	domain.OpPinsCreate, domain.OpPinsDelete, domain.OpPinsUpdate,
	domain.OpSearchesDelete, domain.OpSearchesUpdate,
	domain.OpStarsCreate, domain.OpStarsDelete, domain.OpStarsUpdate,
	domain.OpSubscriptionsCreate, domain.OpSubscriptionsDelete,
	domain.OpTemplatesCreate, domain.OpTemplatesDuplicate, domain.OpTemplatesRestore, domain.OpTemplatesUpdate,
	domain.OpUserMembershipsUpdate,
	domain.OpUsersNotificationsSubscribe, domain.OpUsersNotificationsUnsubscribe,
}

// TestCatalogHoldsEveryStageAReadInOrder checks that the catalog holds the built reads and the part 2a reads, once each,
// then the stage b writes, and nothing else (the admin reads and the stage 2b reads are not rows yet).
func TestCatalogHoldsEveryStageAReadInOrder(t *testing.T) {
	var got []domain.OperationID
	for _, r := range RowsOf(domain.KindOutline) {
		got = append(got, r.ID)
	}
	want := append(append(slices.Clone(stageAReads), stageBWrites...), stageCIDs()...)
	if !slices.Equal(got, want) {
		t.Fatalf("catalog IDs = %v, want %v", got, want)
	}
}

// TestWriteRowsNeedWriteAccessAndAnEffectThatChanges checks the write rows: each one changes the
// wiki, so each needs write access and an effect that is not read. Only documents.update repeats a different change,
// so only it is NotIdempotent. Every write tool says that it changes data.
func TestWriteRowsNeedWriteAccessAndAnEffectThatChanges(t *testing.T) {
	for _, id := range stageBWrites {
		r := rowOf(t, id)
		if r.Effect == domain.EffectRead {
			t.Errorf("%s: a write row needs an effect that changes data, not read", id)
		}
		if r.Access != domain.AccessWrite {
			t.Errorf("%s: access = %s, want write", id, r.Access)
		}
		if r.Tool == "" || !strings.Contains(r.Description, "This tool changes data in the wiki.") {
			t.Errorf("%s: the tool %q must say that it changes data", id, r.Tool)
		}
		if want := id == domain.OpDocumentsUpdate; r.NotIdempotent != want {
			t.Errorf("%s: NotIdempotent = %v, want %v", id, r.NotIdempotent, want)
		}
	}
}

// TestWriteToolNamesAreTheFixedNames checks the seven write tool names of stage b.
func TestWriteToolNamesAreTheFixedNames(t *testing.T) {
	want := map[domain.OperationID]string{
		domain.OpDocumentsCreate:               "create_document",
		domain.OpDocumentsDuplicate:            "duplicate_document",
		domain.OpDocumentsMove:                 "move_document",
		domain.OpDocumentsRestore:              "restore_document",
		domain.OpDocumentsTemplatize:           "templatize_document",
		domain.OpDocumentsUnpublish:            "unpublish_document",
		domain.OpDocumentsUpdate:               "update_document",
		domain.OpCollectionsCreate:             "create_collection",
		domain.OpCollectionsDuplicate:          "duplicate_collection",
		domain.OpCollectionsExport:             "export_collection",
		domain.OpCollectionsMove:               "move_collection",
		domain.OpCollectionsRestore:            "restore_collection",
		domain.OpCollectionsUpdate:             "update_collection",
		domain.OpCommentsCreate:                "create_comment",
		domain.OpCommentsUpdate:                "update_comment",
		domain.OpCommentsResolve:               "resolve_comment",
		domain.OpCommentsUnresolve:             "unresolve_comment",
		domain.OpCommentsAddReaction:           "add_comment_reaction",
		domain.OpCommentsRemoveReaction:        "remove_comment_reaction",
		domain.OpAccessRequestsCreate:          "create_access_request",
		domain.OpRevisionsUpdate:               "update_revision",
		domain.OpNotificationsUpdate:           "update_notification",
		domain.OpNotificationsUpdateAll:        "update_all_notifications",
		domain.OpPinsCreate:                    "create_pin",
		domain.OpPinsDelete:                    "delete_pin",
		domain.OpPinsUpdate:                    "update_pin",
		domain.OpSearchesDelete:                "delete_search",
		domain.OpSearchesUpdate:                "update_search",
		domain.OpStarsCreate:                   "create_star",
		domain.OpStarsDelete:                   "delete_star",
		domain.OpStarsUpdate:                   "update_star",
		domain.OpSubscriptionsCreate:           "create_subscription",
		domain.OpSubscriptionsDelete:           "delete_subscription",
		domain.OpTemplatesCreate:               "create_template",
		domain.OpTemplatesDuplicate:            "duplicate_template",
		domain.OpTemplatesRestore:              "restore_template",
		domain.OpTemplatesUpdate:               "update_template",
		domain.OpUserMembershipsUpdate:         "update_user_membership",
		domain.OpUsersNotificationsSubscribe:   "subscribe_user_notifications",
		domain.OpUsersNotificationsUnsubscribe: "unsubscribe_user_notifications",
	}
	for id, tool := range want {
		if got := rowOf(t, id).Tool; got != tool {
			t.Errorf("%s tool = %q, want %q", id, got, tool)
		}
	}
}

// TestWritesFollowTheInventoryClass checks the CLI command, the effect, and the access of the collections and comments
// writes against the inventory (outline-api-operations.md, collections and comments tables). The creates, the duplicate,
// the export, and the add-reaction are additive; the move, the restore, the updates, the resolves, and the remove-reaction
// are updates. Each one is a write.
func TestWritesFollowTheInventoryClass(t *testing.T) {
	want := map[domain.OperationID]struct {
		cli    string
		effect domain.Effect
	}{
		domain.OpCollectionsCreate:             {"outline collections create", domain.EffectAdditive},
		domain.OpCollectionsDuplicate:          {"outline collections duplicate", domain.EffectAdditive},
		domain.OpCollectionsExport:             {"outline collections export", domain.EffectAdditive},
		domain.OpCollectionsMove:               {"outline collections move", domain.EffectUpdate},
		domain.OpCollectionsRestore:            {"outline collections restore", domain.EffectUpdate},
		domain.OpCollectionsUpdate:             {"outline collections update", domain.EffectUpdate},
		domain.OpCommentsCreate:                {"outline comments create", domain.EffectAdditive},
		domain.OpCommentsUpdate:                {"outline comments update", domain.EffectUpdate},
		domain.OpCommentsResolve:               {"outline comments resolve", domain.EffectUpdate},
		domain.OpCommentsUnresolve:             {"outline comments unresolve", domain.EffectUpdate},
		domain.OpCommentsAddReaction:           {"outline comments add-reaction", domain.EffectAdditive},
		domain.OpCommentsRemoveReaction:        {"outline comments remove-reaction", domain.EffectUpdate},
		domain.OpAccessRequestsCreate:          {"outline access-requests create", domain.EffectAdditive},
		domain.OpRevisionsUpdate:               {"outline revisions update", domain.EffectUpdate},
		domain.OpNotificationsUpdate:           {"outline notifications update", domain.EffectUpdate},
		domain.OpNotificationsUpdateAll:        {"outline notifications update-all", domain.EffectUpdate},
		domain.OpPinsCreate:                    {"outline pins create", domain.EffectAdditive},
		domain.OpPinsDelete:                    {"outline pins delete", domain.EffectUpdate},
		domain.OpPinsUpdate:                    {"outline pins update", domain.EffectUpdate},
		domain.OpSearchesDelete:                {"outline searches delete", domain.EffectUpdate},
		domain.OpSearchesUpdate:                {"outline searches update", domain.EffectUpdate},
		domain.OpStarsCreate:                   {"outline stars create", domain.EffectAdditive},
		domain.OpStarsDelete:                   {"outline stars delete", domain.EffectUpdate},
		domain.OpStarsUpdate:                   {"outline stars update", domain.EffectUpdate},
		domain.OpSubscriptionsCreate:           {"outline subscriptions create", domain.EffectAdditive},
		domain.OpSubscriptionsDelete:           {"outline subscriptions delete", domain.EffectUpdate},
		domain.OpTemplatesCreate:               {"outline templates create", domain.EffectAdditive},
		domain.OpTemplatesDuplicate:            {"outline templates duplicate", domain.EffectAdditive},
		domain.OpTemplatesRestore:              {"outline templates restore", domain.EffectUpdate},
		domain.OpTemplatesUpdate:               {"outline templates update", domain.EffectUpdate},
		domain.OpUserMembershipsUpdate:         {"outline user-memberships update", domain.EffectUpdate},
		domain.OpUsersNotificationsSubscribe:   {"outline users notifications-subscribe", domain.EffectAdditive},
		domain.OpUsersNotificationsUnsubscribe: {"outline users notifications-unsubscribe", domain.EffectAdditive},
	}
	for id, w := range want {
		r := rowOf(t, id)
		if r.CLI != w.cli || r.Effect != w.effect || r.Access != domain.AccessWrite {
			t.Errorf("%s = (%q, %s, %s), want (%q, %s, write)", id, r.CLI, r.Effect, r.Access, w.cli, w.effect)
		}
	}
}

// TestEveryRowIsWellFormed checks the invariants a row must keep for the generated adapters to be correct.
func TestEveryRowIsWellFormed(t *testing.T) {
	cli := map[string]bool{}
	tools := map[string]bool{}
	for _, r := range Rows() {
		kind, _ := r.Kind()
		if len(r.Path()) < 2 || r.Path()[0] != kind.String() {
			t.Errorf("%s: the CLI path %q must start with its kind %s", r.ID, r.CLI, kind)
		}
		if cli[r.CLI] {
			t.Errorf("%s: CLI path %q is listed twice", r.ID, r.CLI)
		}
		cli[r.CLI] = true
		if r.Tool != "" {
			if !toolPattern.MatchString(r.Tool) {
				t.Errorf("%s: tool name %q does not match ^[a-z]+(_[a-z]+)*$", r.ID, r.Tool)
			}
			if tools[r.Tool] {
				t.Errorf("%s: tool name %q is listed twice", r.ID, r.Tool)
			}
			tools[r.Tool] = true
		}
		if r.Summary == "" || r.Description == "" {
			t.Errorf("%s: a summary and a description are required", r.ID)
		}
		if r.Access.String() == "" || r.Effect == "" {
			t.Errorf("%s: an access level and an effect are required", r.ID)
		}
		// A read changes nothing, so it needs read access, or admin access when it shows admin data.
		if r.Effect == domain.EffectRead && r.Access != domain.AccessRead && r.Access != domain.AccessAdmin {
			t.Errorf("%s: a read effect needs read access, or admin access for an admin read", r.ID)
		}
		names := map[string]bool{}
		for _, p := range r.Params {
			if names[p.Name] {
				t.Errorf("%s: parameter %s is listed twice", r.ID, p.Name)
			}
			names[p.Name] = true
			if p.Help == "" {
				t.Errorf("%s: parameter %s needs help", r.ID, p.Name)
			}
		}
		if r.Positional() != "" && r.Params[0].Name != r.Positional() {
			t.Errorf("%s: the positional parameter must be the first parameter", r.ID)
		}
		if r.IsList() {
			if r.Noun == "" || len(r.Columns) == 0 {
				t.Errorf("%s: a list needs a noun and text columns", r.ID)
			}
			for _, c := range r.Columns {
				if !slices.Contains(r.Fields, c) {
					t.Errorf("%s: column %s is not a field", r.ID, c)
				}
			}
		}
		if r.Indent != "" && !slices.Contains(r.Fields, r.Indent) {
			t.Errorf("%s: indent field %s is not a field", r.ID, r.Indent)
		}
		if r.Body != "" && !slices.Contains(r.Fields, r.Body) {
			t.Errorf("%s: body field %s is not a field", r.ID, r.Body)
		}
	}
}

func TestToolNamesMatchTheFixedNames(t *testing.T) {
	want := map[domain.OperationID]string{
		domain.OpCollectionsList:       "list_collections",
		domain.OpCollectionsInfo:       "collection_info",
		domain.OpCollectionsDocuments:  "collection_tree",
		domain.OpDocumentsGet:          "get_document",
		domain.OpDocumentsList:         "list_documents",
		domain.OpDocumentsSearch:       "search_documents",
		domain.OpAuthWhoami:            "whoami",
		domain.OpDocumentsArchived:     "list_archived_documents",
		domain.OpDocumentsChildren:     "document_children",
		domain.OpDocumentsSearchTitles: "search_document_titles",
		domain.OpCommentsGet:           "get_comment",
		domain.OpSubscriptionsGet:      "get_subscription",
		domain.OpStarsList:             "list_stars",
	}
	for id, tool := range want {
		if got := rowOf(t, id).Tool; got != tool {
			t.Errorf("%s tool = %q, want %q", id, got, tool)
		}
	}
}

// TestRevisionsExportHasNoMCPTool checks that the revision export is a CLI command only. Its Markdown is the same as
// get_revision's text, cut into pages of a size the model should not read, so the MCP surface offers no tool for it
// (finding a2). The CLI command keeps it.
func TestRevisionsExportHasNoMCPTool(t *testing.T) {
	row := rowOf(t, domain.OpRevisionsExport)
	if row.Tool != "" {
		t.Fatalf("revisions.export tool = %q, want no MCP tool", row.Tool)
	}
	if row.CLI != "outline revisions export" {
		t.Fatalf("revisions.export CLI = %q, want the CLI command kept", row.CLI)
	}
}

func TestBindFillsDefaultsAndKeepsTheSetValues(t *testing.T) {
	row := rowOf(t, domain.OpDocumentsList)
	args, err := Bind(row, map[string]any{"collectionId": "123e4567-e89b-12d3-a456-426614174000", "limit": "10"})
	if err != nil {
		t.Fatalf("Bind = %v", err)
	}
	if args.String("collectionId") != "123e4567-e89b-12d3-a456-426614174000" || args.Int("limit") != 10 || args.Int("offset") != 0 {
		t.Fatalf("values = collectionId %q limit %d offset %d", args.String("collectionId"), args.Int("limit"), args.Int("offset"))
	}
	defaults, err := Bind(row, nil)
	if err != nil {
		t.Fatalf("Bind(nil) = %v", err)
	}
	if defaults.Int("limit") != domain.DefaultPageLimit {
		t.Fatalf("default limit = %d, want %d", defaults.Int("limit"), domain.DefaultPageLimit)
	}
}

func TestBindAcceptsTypedAndTextValues(t *testing.T) {
	row := rowOf(t, domain.OpDocumentsList)
	typed, err := Bind(row, map[string]any{"limit": float64(5), "offset": 3})
	if err != nil || typed.Int("limit") != 5 || typed.Int("offset") != 3 {
		t.Fatalf("typed numbers = %v, %+v", err, typed)
	}
	if _, err := Bind(row, map[string]any{"limit": "abc"}); !errors.Is(err, domain.ErrInvalidLimit) {
		t.Fatalf("a text limit that is not a number error = %v, want ErrInvalidLimit", err)
	}
}

func TestBindRefusesBadInputBeforeAnyCall(t *testing.T) {
	cases := []struct {
		name string
		op   domain.OperationID
		raw  map[string]any
		want error
	}{
		{"a missing required argument", domain.OpDocumentsGet, nil, domain.ErrMissingArgument},
		{"a document ID with a dot", domain.OpDocumentsGet, map[string]any{"documentId": "a.b"}, domain.ErrInvalidDocumentID},
		{"a collection ID with a dot", domain.OpCollectionsInfo, map[string]any{"collectionId": "col.1"}, domain.ErrInvalidCollectionID},
		{"a urlId slug as a list collection", domain.OpDocumentsList, map[string]any{"collectionId": "col-1"}, domain.ErrInvalidCollectionID},
		{"a limit of zero", domain.OpDocumentsList, map[string]any{"limit": 0}, domain.ErrInvalidLimit},
		{"a limit above one hundred", domain.OpDocumentsList, map[string]any{"limit": 101}, domain.ErrInvalidLimit},
		{"a negative offset", domain.OpDocumentsList, map[string]any{"offset": -1}, domain.ErrInvalidOffset},
		{"a blank query", domain.OpDocumentsSearch, map[string]any{"query": "  "}, domain.ErrInvalidQuery},
		{"a query over 1000 characters", domain.OpDocumentsSearch, map[string]any{"query": strings.Repeat("q", 1001)}, domain.ErrInvalidQuery},
		{"an argument the row does not declare", domain.OpAuthWhoami, map[string]any{"email": "x"}, domain.ErrUnknownArgument},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			row := rowOf(t, tc.op)
			if _, err := Bind(row, tc.raw); !errors.Is(err, tc.want) {
				t.Fatalf("Bind error = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestBindIgnoresTheSourceArgument(t *testing.T) {
	row := rowOf(t, domain.OpAuthWhoami)
	if _, err := Bind(row, map[string]any{"source": "outline"}); err != nil {
		t.Fatalf("source argument error = %v, want nil", err)
	}
}

func TestValidateRechecksTheArgumentsOfAnArgsValue(t *testing.T) {
	row := rowOf(t, domain.OpDocumentsGet)
	args := domain.NewArgs(map[string]any{"documentId": "bad.id"})
	if err := Validate(row, args); !errors.Is(err, domain.ErrInvalidDocumentID) {
		t.Fatalf("Validate error = %v, want ErrInvalidDocumentID", err)
	}
	if err := Validate(row, domain.NewArgs(map[string]any{"documentId": "doc-1"})); err != nil {
		t.Fatalf("Validate of a valid value = %v", err)
	}
}

func TestBindSplitsAnIDListGivenAsText(t *testing.T) {
	kind := Param{Name: "ids", Kind: KindIDList, Help: "ids"}
	got, err := convert(kind, "a,b,c")
	if err != nil {
		t.Fatalf("convert = %v", err)
	}
	if !slices.Equal(got.([]string), []string{"a", "b", "c"}) {
		t.Fatalf("list = %v", got)
	}
}

// sourceAt returns an outline source with the given access level.
func sourceAt(t *testing.T, access domain.Access) domain.Source {
	t.Helper()
	id, err := domain.NewSourceID("outline")
	if err != nil {
		t.Fatalf("NewSourceID: %v", err)
	}
	ep, err := domain.NewEndpoint("https://docs.example.com")
	if err != nil {
		t.Fatalf("NewEndpoint: %v", err)
	}
	ref, err := domain.NewCredentialRef("OUTLINE_API_KEY")
	if err != nil {
		t.Fatalf("NewCredentialRef: %v", err)
	}
	return domain.NewSource(id, domain.KindOutline, domain.TagSet{}, access, ep, ref)
}

func TestRefuseNamesTheSourceAndBothLevels(t *testing.T) {
	src := sourceAt(t, domain.AccessRead)
	err := Refuse(Row{Access: domain.AccessWrite}, src)
	if !errors.Is(err, domain.ErrAccessLevelTooLow) {
		t.Fatalf("Refuse error = %v, want ErrAccessLevelTooLow", err)
	}
	if err.Error() != `source "outline" allows read only; this operation needs write` {
		t.Fatalf("Refuse text = %q", err.Error())
	}
	if err := Refuse(Row{Access: domain.AccessRead}, src); err != nil {
		t.Fatalf("Refuse of a read by a read source = %v", err)
	}
}

func TestSupportsTheOutlineKindForEveryRow(t *testing.T) {
	for _, r := range RowsOf(domain.KindOutline) {
		if !Supports(domain.KindOutline, r) {
			t.Errorf("%s is not supported by the outline kind", r.ID)
		}
	}
}

// TestBindChoosesExactlyOneOfARowsNamedArguments checks the rule of a row that selects one target from two arguments:
// one of them must be set, and never both.
func TestBindChoosesExactlyOneOfARowsNamedArguments(t *testing.T) {
	row := rowOf(t, domain.OpSubscriptionsGet)
	cases := []struct {
		name string
		raw  map[string]any
		want error
	}{
		{"a document target", map[string]any{"documentId": "doc-1"}, nil},
		{"a collection target", map[string]any{"collectionId": "123e4567-e89b-12d3-a456-426614174000"}, nil},
		{"no target", nil, domain.ErrExactlyOneArgument},
		{"both targets", map[string]any{"documentId": "doc-1", "collectionId": "123e4567-e89b-12d3-a456-426614174000"}, domain.ErrExactlyOneArgument},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			args, err := Bind(row, tc.raw)
			if !errors.Is(err, tc.want) {
				t.Fatalf("Bind error = %v, want %v", err, tc.want)
			}
			if tc.want == nil && args.String("event") != "documents.update" {
				t.Fatalf("the event default = %q, want documents.update", args.String("event"))
			}
		})
	}
}
