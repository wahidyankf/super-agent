package outline

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"sync"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/tests/support/outlinestub"
)

// wireCall is one request the gateway sent: the wire path and the decoded JSON body.
type wireCall struct {
	Path string
	Body any
}

// wireRecorder is a transport that records each request's path and body, then hands the request to the stub. The
// stub answers the call, and the recorder keeps what went out.
type wireRecorder struct {
	next  http.RoundTripper
	mu    sync.Mutex
	calls []wireCall
}

// RoundTrip records the request and delegates it.
func (w *wireRecorder) RoundTrip(req *http.Request) (*http.Response, error) {
	var raw []byte
	if req.Body != nil {
		raw, _ = io.ReadAll(req.Body)
	}
	req.Body = io.NopCloser(bytes.NewReader(raw))
	var body any
	_ = json.Unmarshal(raw, &body)
	w.mu.Lock()
	w.calls = append(w.calls, wireCall{Path: req.URL.Path, Body: body})
	w.mu.Unlock()
	return w.next.RoundTrip(req)
}

// recorded returns the requests sent so far.
func (w *wireRecorder) recorded() []wireCall {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]wireCall(nil), w.calls...)
}

// recordingGateway builds the gateway over a recording transport and an in-process stub.
func recordingGateway(t *testing.T) (*Gateway, *wireRecorder) {
	t.Helper()
	return recordingGatewayAt(t)
}

// recordingGatewayAt is recordingGateway for a source. Every request of the source reaches the wire.
func recordingGatewayAt(t *testing.T) (*Gateway, *wireRecorder) {
	t.Helper()
	clock := newFakeClock()
	stub := outlinestub.New(outlinestub.WithClock(clock.Now))
	rec := &wireRecorder{next: stub.RoundTripper()}
	client := &http.Client{Transport: rec}
	gw, err := NewWithBase(testSourceAt(t, testHost), testBase, client, &staticCreds{},
		WithMinDelay(0), WithClock(clock.Now), WithSleep(clock.Sleep))
	if err != nil {
		t.Fatalf("NewWithBase: %v", err)
	}
	return gw, rec
}

// Sample IDs of the request-shape cases.
const (
	uuidCollection = "11111111-1111-4111-8111-111111111111"
	uuidOther      = "22222222-2222-4222-8222-222222222222"
	uuidUser       = "33333333-3333-4333-8333-333333333333"
	uuidComment    = "44444444-4444-4444-8444-444444444444"
	uuidRevision   = "55555555-5555-4555-8555-555555555555"
	uuidDocument   = "66666666-6666-4666-8666-666666666666"
	uuidGroup      = "77777777-7777-4777-8777-777777777777"
)

// requestCase is one catalog read with the arguments it is given and the exact request it must send. The body is JSON,
// so a key that is missing, extra, or of the wrong type fails the comparison.
type requestCase struct {
	name string
	op   domain.OperationID
	args map[string]any
	path string
	body string
}

// requestCases covers every endpoint row of parts 1 and 2a. Each body names exactly the keys the call sends.
var requestCases = []requestCase{
	{"collections.list", domain.OpCollectionsList,
		map[string]any{}, "/api/collections.list", `{"offset":0,"limit":25}`},
	{"collections.info", domain.OpCollectionsInfo,
		map[string]any{"collectionId": "col-1"}, "/api/collections.info", `{"id":"col-1"}`},
	{"collections.documents", domain.OpCollectionsDocuments,
		map[string]any{"collectionId": "col-1"}, "/api/collections.documents", `{"id":"col-1"}`},
	{"documents.get", domain.OpDocumentsGet,
		map[string]any{"documentId": "doc-1"}, "/api/documents.info", `{"id":"doc-1"}`},
	{"documents.list", domain.OpDocumentsList,
		map[string]any{"offset": 0, "limit": 25}, "/api/documents.list", `{"offset":0,"limit":25}`},
	{"documents.search", domain.OpDocumentsSearch,
		map[string]any{"query": "plan", "offset": 0, "limit": 25}, "/api/documents.search",
		`{"offset":0,"limit":25,"query":"plan"}`},
	{"documents.archived", domain.OpDocumentsArchived,
		map[string]any{"collectionId": uuidCollection, "sort": "title", "direction": "ASC", "offset": 0, "limit": 25},
		"/api/documents.archived",
		`{"offset":0,"limit":25,"collectionId":"` + uuidCollection + `","sort":"title","direction":"ASC"}`},
	{"documents.children", domain.OpDocumentsChildren,
		map[string]any{"documentId": "doc-1"},
		"/api/documents.documents", `{"id":"doc-1"}`},
	{"documents.deleted", domain.OpDocumentsDeleted,
		map[string]any{"sort": "updatedAt", "direction": "DESC", "offset": 25, "limit": 10},
		"/api/documents.deleted", `{"offset":25,"limit":10,"sort":"updatedAt","direction":"DESC"}`},
	{"documents.drafts", domain.OpDocumentsDrafts,
		map[string]any{"collectionId": uuidCollection, "offset": 0, "limit": 25},
		"/api/documents.drafts", `{"offset":0,"limit":25,"collectionId":"` + uuidCollection + `"}`},
	{"documents.group-memberships", domain.OpDocumentsGroupMemberships,
		map[string]any{"documentId": "doc-1", "query": "eng", "permission": "admin", "offset": 0, "limit": 25},
		"/api/documents.group_memberships",
		`{"offset":0,"limit":25,"id":"doc-1","query":"eng","permission":"admin"}`},
	{"documents.insights", domain.OpDocumentsInsights,
		map[string]any{"documentId": "doc-1", "startDate": "2026-01-01T00:00:00Z", "endDate": "2026-01-31T00:00:00Z"},
		"/api/documents.insights",
		`{"id":"doc-1","startDate":"2026-01-01T00:00:00Z","endDate":"2026-01-31T00:00:00Z"}`},
	{"documents.memberships", domain.OpDocumentsMemberships,
		map[string]any{"documentId": "doc-1", "offset": 0, "limit": 25},
		"/api/documents.memberships", `{"offset":0,"limit":25,"id":"doc-1"}`},
	{"documents.search-titles", domain.OpDocumentsSearchTitles,
		map[string]any{
			"query": "plan", "collectionId": uuidCollection, "documentId": uuidOther, "userId": uuidUser,
			"statusFilter": []string{"published"}, "sort": "title", "direction": "ASC", "offset": 0, "limit": 25,
		},
		"/api/documents.search_titles",
		`{"offset":0,"limit":25,"query":"plan","collectionId":"` + uuidCollection + `","documentId":"` + uuidOther +
			`","userId":"` + uuidUser + `","statusFilter":["published"],"sort":"title","direction":"ASC"}`},
	{"documents.users", domain.OpDocumentsUsers,
		map[string]any{"documentId": "doc-1", "query": "ann", "userId": uuidUser, "offset": 0, "limit": 25},
		"/api/documents.users", `{"offset":0,"limit":25,"id":"doc-1","query":"ann","userId":"` + uuidUser + `"}`},
	{"documents.viewed", domain.OpDocumentsViewed,
		map[string]any{"sort": "updatedAt", "direction": "ASC", "offset": 0, "limit": 25},
		"/api/documents.viewed", `{"offset":0,"limit":25,"sort":"updatedAt","direction":"ASC"}`},
	{"collections.memberships", domain.OpCollectionsMemberships,
		map[string]any{"collectionId": "col-1", "query": "ann", "permission": "read", "offset": 0, "limit": 25},
		"/api/collections.memberships", `{"offset":0,"limit":25,"id":"col-1","query":"ann","permission":"read"}`},
	{"collections.group-memberships", domain.OpCollectionsGroupMembers,
		map[string]any{"collectionId": "col-1", "offset": 0, "limit": 25},
		"/api/collections.group_memberships", `{"offset":0,"limit":25,"id":"col-1"}`},
	{"comments.get", domain.OpCommentsGet,
		map[string]any{"commentId": uuidComment, "includeAnchorText": true},
		"/api/comments.info", `{"id":"` + uuidComment + `","includeAnchorText":true}`},
	{"comments.list", domain.OpCommentsList,
		map[string]any{
			"documentId": "doc-1", "statusFilter": []string{"resolved"}, "includeAnchorText": true,
			"sort": "createdAt", "direction": "DESC", "offset": 0, "limit": 25,
		},
		"/api/comments.list",
		`{"offset":0,"limit":25,"documentId":"doc-1","statusFilter":["resolved"],"includeAnchorText":true,` +
			`"sort":"createdAt","direction":"DESC"}`},
	{"revisions.export", domain.OpRevisionsExport,
		map[string]any{"revisionId": uuidRevision}, "/api/revisions.export", `{"id":"` + uuidRevision + `"}`},
	{"revisions.get", domain.OpRevisionsGet,
		map[string]any{"revisionId": uuidRevision}, "/api/revisions.info", `{"id":"` + uuidRevision + `"}`},
	{"revisions.list", domain.OpRevisionsList,
		map[string]any{"documentId": uuidDocument, "sort": "createdAt", "direction": "ASC", "offset": 0, "limit": 25},
		"/api/revisions.list",
		`{"offset":0,"limit":25,"documentId":"` + uuidDocument + `","sort":"createdAt","direction":"ASC"}`},
	{"shares.get", domain.OpSharesGet,
		map[string]any{"shareId": "share-1"}, "/api/shares.info", `{"id":"share-1"}`},
	{"shares.list", domain.OpSharesList,
		map[string]any{"query": "plan", "sort": "views", "direction": "DESC", "offset": 0, "limit": 25},
		"/api/shares.list", `{"offset":0,"limit":25,"query":"plan","sort":"views","direction":"DESC"}`},
	{"users.get with a user", domain.OpUsersGet,
		map[string]any{"userId": uuidUser}, "/api/users.info", `{"id":"` + uuidUser + `"}`},
	{"users.get without a user", domain.OpUsersGet,
		map[string]any{}, "/api/users.info", `{}`},
	{"users.list", domain.OpUsersList,
		map[string]any{
			"query": "ann", "role": "admin", "filter": "active", "ids": []string{uuidUser, uuidComment},
			"sort": "name", "direction": "ASC", "offset": 0, "limit": 25,
		},
		"/api/users.list",
		`{"offset":0,"limit":25,"query":"ann","role":"admin","filter":"active","ids":["` + uuidUser + `","` +
			uuidComment + `"],"sort":"name","direction":"ASC"}`},
	{"groups.get", domain.OpGroupsGet,
		map[string]any{"groupId": uuidGroup}, "/api/groups.info", `{"id":"` + uuidGroup + `"}`},
	{"groups.list", domain.OpGroupsList,
		map[string]any{
			"userId": uuidUser, "groupSource": "manual", "query": "eng", "sort": "name", "direction": "ASC",
			"offset": 0, "limit": 25,
		},
		"/api/groups.list",
		`{"offset":0,"limit":25,"userId":"` + uuidUser + `","source":"manual","query":"eng","sort":"name",` +
			`"direction":"ASC"}`},
	{"groups.members", domain.OpGroupsMembers,
		map[string]any{"groupId": uuidGroup, "query": "ann", "permission": "admin", "offset": 0, "limit": 25},
		"/api/groups.memberships",
		`{"offset":0,"limit":25,"id":"` + uuidGroup + `","query":"ann","permission":"admin"}`},
	{"templates.get", domain.OpTemplatesGet,
		map[string]any{"templateId": "tpl-1"}, "/api/templates.info", `{"id":"tpl-1"}`},
	{"templates.list", domain.OpTemplatesList,
		map[string]any{
			"collectionId": uuidCollection, "query": "weekly", "sort": "title", "direction": "ASC",
			"offset": 0, "limit": 25,
		},
		"/api/templates.list",
		`{"offset":0,"limit":25,"collectionId":"` + uuidCollection + `","query":"weekly","sort":"title","direction":"ASC"}`},
	{"pins.get on a collection", domain.OpPinsGet,
		map[string]any{"documentId": "doc-1", "collectionId": uuidCollection},
		"/api/pins.info", `{"documentId":"doc-1","collectionId":"` + uuidCollection + `"}`},
	{"pins.get on Home", domain.OpPinsGet,
		map[string]any{"documentId": "doc-1"}, "/api/pins.info", `{"documentId":"doc-1"}`},
	{"pins.list on a collection", domain.OpPinsList,
		map[string]any{"collectionId": uuidCollection, "offset": 0, "limit": 25},
		"/api/pins.list", `{"offset":0,"limit":25,"collectionId":"` + uuidCollection + `"}`},
	{"pins.list on Home", domain.OpPinsList,
		map[string]any{"offset": 0, "limit": 25}, "/api/pins.list", `{"offset":0,"limit":25}`},
	{"stars.list", domain.OpStarsList,
		map[string]any{"offset": 0, "limit": 25}, "/api/stars.list", `{"offset":0,"limit":25}`},
	{"subscriptions.get on a document", domain.OpSubscriptionsGet,
		map[string]any{"event": "documents.update", "documentId": "doc-1"},
		"/api/subscriptions.info", `{"event":"documents.update","documentId":"doc-1"}`},
	{"subscriptions.list on a collection", domain.OpSubscriptionsList,
		map[string]any{"event": "documents.update", "collectionId": uuidCollection, "offset": 0, "limit": 25},
		"/api/subscriptions.list",
		`{"offset":0,"limit":25,"event":"documents.update","collectionId":"` + uuidCollection + `"}`},
	{"auth.whoami", domain.OpAuthWhoami, map[string]any{}, "/api/auth.info", `{}`},
}

// TestEveryReadSendsItsExactRequest checks each endpoint row: the method path and the JSON body that goes out. A
// body names only the keys the caller set (plus the page for a paged list), so a missing or extra key fails the case.
// The reply is not under test here, so the call's error is not checked.
func TestEveryReadSendsItsExactRequest(t *testing.T) {
	for _, tc := range requestCases {
		t.Run(tc.name, func(t *testing.T) {
			gw, rec := recordingGatewayAt(t)
			_, _ = gw.Read(context.Background(), tc.op, domain.NewArgs(tc.args))
			calls := rec.recorded()
			if len(calls) != 1 {
				t.Fatalf("requests = %d, want 1", len(calls))
			}
			if calls[0].Path != tc.path {
				t.Fatalf("path = %q, want %q", calls[0].Path, tc.path)
			}
			var want any
			if err := json.Unmarshal([]byte(tc.body), &want); err != nil {
				t.Fatalf("bad expected body: %v", err)
			}
			if !reflect.DeepEqual(calls[0].Body, want) {
				t.Fatalf("body = %v, want %v", calls[0].Body, want)
			}
		})
	}
}

// TestEveryCatalogReadHasARequestCase checks that the request cases name every served operation, so a new endpoint row
// without a body case fails here.
func TestEveryCatalogReadHasARequestCase(t *testing.T) {
	covered := map[domain.OperationID]bool{}
	for _, tc := range append(append(append([]requestCase(nil), requestCases...), requestCasesB...), requestCasesC...) {
		covered[tc.op] = true
	}
	for _, op := range ServedOperations() {
		if !covered[op] {
			t.Fatalf("operation %s has no request case", op)
		}
	}
}

// TestUnboundPageParametersTakeTheBinderDefaults checks that a read whose caller set no page sends the page the binder
// gives it: offset 0 and the default limit of 25. The reader must use the bound arguments, not the ones it was given,
// or it sends "limit": 0 and the server answers an empty page.
func TestUnboundPageParametersTakeTheBinderDefaults(t *testing.T) {
	gw, rec := recordingGateway(t)
	_, _ = gw.Read(context.Background(), domain.OpDocumentsList, domain.NewArgs(map[string]any{}))
	calls := rec.recorded()
	if len(calls) != 1 {
		t.Fatalf("requests = %d, want 1", len(calls))
	}
	body, ok := calls[0].Body.(map[string]any)
	if !ok {
		t.Fatalf("body is not an object: %v", calls[0].Body)
	}
	if body["offset"] != float64(0) || body["limit"] != float64(domain.DefaultPageLimit) {
		t.Fatalf("page = offset %v, limit %v; want offset 0 and limit %d", body["offset"], body["limit"], domain.DefaultPageLimit)
	}
}

// TestPageLessReadsSendNoPage checks that a read whose endpoint is not paged sends no offset or limit.
func TestPageLessReadsSendNoPage(t *testing.T) {
	gw, rec := recordingGateway(t)
	_, _ = gw.Read(context.Background(), domain.OpRevisionsGet, domain.NewArgs(map[string]any{"revisionId": uuidRevision}))
	body, ok := rec.recorded()[0].Body.(map[string]any)
	if !ok {
		t.Fatalf("body is not an object: %v", rec.recorded()[0].Body)
	}
	for _, key := range []string{"offset", "limit"} {
		if _, has := body[key]; has {
			t.Fatalf("body has %q, want none for a page-less read", key)
		}
	}
}
