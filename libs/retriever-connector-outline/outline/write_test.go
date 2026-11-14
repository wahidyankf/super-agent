package outline

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/operations"
	"github.com/wahidyankf/super-agent/libs/retriever-testkit/outlinestub"
	"net/http"
	"reflect"
	"slices"
	"testing"
	"time"
)

// requestCasesB are the request-shape cases of the seven documents writes of stage b. Each body is
// the exact JSON the call sends, written by hand from the field reference. A body names only the keys the caller set.
var requestCasesB = []requestCase{
	{"documents.create as a draft", domain.OpDocumentsCreate,
		map[string]any{"title": "Draft"}, "/api/documents.create", `{"title":"Draft"}`},
	{"documents.create in a collection", domain.OpDocumentsCreate,
		map[string]any{"title": "Plan", "text": "# Plan", "collectionId": uuidCollection, "parentDocumentId": uuidOther,
			"templateId": uuidUser, "publish": true},
		"/api/documents.create",
		`{"title":"Plan","text":"# Plan","collectionId":"` + uuidCollection + `","parentDocumentId":"` + uuidOther +
			`","templateId":"` + uuidUser + `","publish":true}`},
	{"documents.duplicate", domain.OpDocumentsDuplicate,
		map[string]any{"documentId": "doc-1", "title": "Copy", "recursive": true, "publish": false,
			"collectionId": uuidCollection, "parentDocumentId": uuidOther},
		"/api/documents.duplicate",
		`{"id":"doc-1","title":"Copy","recursive":true,"publish":false,"collectionId":"` + uuidCollection +
			`","parentDocumentId":"` + uuidOther + `"}`},
	{"documents.move", domain.OpDocumentsMove,
		map[string]any{"documentId": "doc-1", "collectionId": uuidCollection, "parentDocumentId": uuidOther, "index": 2},
		"/api/documents.move",
		`{"id":"doc-1","collectionId":"` + uuidCollection + `","parentDocumentId":"` + uuidOther + `","index":2}`},
	{"documents.restore", domain.OpDocumentsRestore,
		map[string]any{"documentId": "doc-1", "collectionId": uuidCollection, "revisionId": uuidOther},
		"/api/documents.restore",
		`{"id":"doc-1","collectionId":"` + uuidCollection + `","revisionId":"` + uuidOther + `"}`},
	{"documents.templatize", domain.OpDocumentsTemplatize,
		map[string]any{"documentId": "doc-1", "publish": true, "collectionId": uuidCollection},
		"/api/documents.templatize",
		`{"id":"doc-1","publish":true,"collectionId":"` + uuidCollection + `"}`},
	{"documents.unpublish", domain.OpDocumentsUnpublish,
		map[string]any{"documentId": "doc-1", "detach": true},
		"/api/documents.unpublish", `{"id":"doc-1","detach":true}`},
	{"documents.update with a text and a revision", domain.OpDocumentsUpdate,
		map[string]any{"documentId": "doc-1", "title": "New", "text": "Body", "editMode": "append", "findText": "x",
			"lastRevision": 3, "publish": true, "collectionId": uuidCollection},
		"/api/documents.update",
		`{"id":"doc-1","title":"New","text":"Body","editMode":"append","findText":"x","lastRevision":3,"publish":true,` +
			`"collectionId":"` + uuidCollection + `"}`},
	{"documents.update of a title only", domain.OpDocumentsUpdate,
		map[string]any{"documentId": "doc-1", "title": "Only"}, "/api/documents.update", `{"id":"doc-1","title":"Only"}`},
	{"collections.create with its name only", domain.OpCollectionsCreate,
		map[string]any{"name": "Wiki"}, "/api/collections.create", `{"name":"Wiki"}`},
	{"collections.create with every set field", domain.OpCollectionsCreate,
		map[string]any{"name": "Wiki", "description": "About", "permission": "read_write", "sharing": true,
			"commenting": false, "templateManagement": "admin"},
		"/api/collections.create",
		`{"name":"Wiki","description":"About","permission":"read_write","sharing":true,"commenting":false,"templateManagement":"admin"}`},
	{"collections.duplicate", domain.OpCollectionsDuplicate,
		map[string]any{"collectionId": "col-1", "name": "Copy"}, "/api/collections.duplicate", `{"id":"col-1","name":"Copy"}`},
	{"collections.export", domain.OpCollectionsExport,
		map[string]any{"collectionId": "col-1", "format": "outline-markdown", "includeAttachments": false},
		"/api/collections.export", `{"id":"col-1","format":"outline-markdown","includeAttachments":false}`},
	{"collections.move", domain.OpCollectionsMove,
		map[string]any{"collectionId": "col-1", "index": "P"}, "/api/collections.move", `{"id":"col-1","index":"P"}`},
	{"collections.restore", domain.OpCollectionsRestore,
		map[string]any{"collectionId": "col-1"}, "/api/collections.restore", `{"id":"col-1"}`},
	{"collections.update", domain.OpCollectionsUpdate,
		map[string]any{"collectionId": "col-1", "name": "Wiki 2", "sharing": true, "templateManagement": "read_write"},
		"/api/collections.update", `{"id":"col-1","name":"Wiki 2","sharing":true,"templateManagement":"read_write"}`},
	{"comments.create on a document", domain.OpCommentsCreate,
		map[string]any{"documentId": uuidOther, "text": "Looks good"},
		"/api/comments.create", `{"documentId":"` + uuidOther + `","text":"Looks good"}`},
	{"comments.create as a reply", domain.OpCommentsCreate,
		map[string]any{"documentId": uuidOther, "text": "Agreed", "parentCommentId": uuidUser},
		"/api/comments.create", `{"documentId":"` + uuidOther + `","text":"Agreed","parentCommentId":"` + uuidUser + `"}`},
	{"comments.update", domain.OpCommentsUpdate,
		map[string]any{"commentId": uuidOther, "text": "Edited"},
		"/api/comments.update", `{"id":"` + uuidOther + `","text":"Edited"}`},
	{"comments.resolve", domain.OpCommentsResolve,
		map[string]any{"commentId": uuidOther}, "/api/comments.resolve", `{"id":"` + uuidOther + `"}`},
	{"comments.unresolve", domain.OpCommentsUnresolve,
		map[string]any{"commentId": uuidOther}, "/api/comments.unresolve", `{"id":"` + uuidOther + `"}`},
	{"comments.add_reaction", domain.OpCommentsAddReaction,
		map[string]any{"commentId": uuidOther, "emoji": "\U0001F44D"},
		"/api/comments.add_reaction", `{"id":"` + uuidOther + `","emoji":"` + "\U0001F44D" + `"}`},
	{"comments.remove_reaction", domain.OpCommentsRemoveReaction,
		map[string]any{"commentId": uuidOther, "emoji": "\U0001F44D"},
		"/api/comments.remove_reaction", `{"id":"` + uuidOther + `","emoji":"` + "\U0001F44D" + `"}`},
	{"accessRequests.create", domain.OpAccessRequestsCreate,
		map[string]any{"documentId": uuidOther},
		"/api/accessRequests.create", `{"documentId":"` + uuidOther + `"}`},
	{"revisions.update", domain.OpRevisionsUpdate,
		map[string]any{"revisionId": uuidOther, "name": "Plan"},
		"/api/revisions.update", `{"id":"` + uuidOther + `","name":"Plan"}`},
	{"notifications.update", domain.OpNotificationsUpdate,
		map[string]any{"notificationId": uuidOther, "viewedAt": "2026-01-02T03:04:05Z"},
		"/api/notifications.update", `{"id":"` + uuidOther + `","viewedAt":"2026-01-02T03:04:05Z"}`},
	{"notifications.update_all", domain.OpNotificationsUpdateAll,
		map[string]any{"archivedAt": "2026-01-02T03:04:05Z"},
		"/api/notifications.update_all", `{"archivedAt":"2026-01-02T03:04:05Z"}`},
	{"pins.create on a collection", domain.OpPinsCreate,
		map[string]any{"documentId": "doc-1", "collectionId": uuidCollection, "index": "P"},
		"/api/pins.create", `{"documentId":"doc-1","collectionId":"` + uuidCollection + `","index":"P"}`},
	{"pins.create on Home", domain.OpPinsCreate,
		map[string]any{"documentId": "doc-1"}, "/api/pins.create", `{"documentId":"doc-1"}`},
	{"pins.delete", domain.OpPinsDelete,
		map[string]any{"pinId": uuidOther}, "/api/pins.delete", `{"id":"` + uuidOther + `"}`},
	{"pins.update", domain.OpPinsUpdate,
		map[string]any{"pinId": uuidOther, "index": "P"}, "/api/pins.update", `{"id":"` + uuidOther + `","index":"P"}`},
	{"searches.delete by ID", domain.OpSearchesDelete,
		map[string]any{"searchId": uuidOther}, "/api/searches.delete", `{"id":"` + uuidOther + `"}`},
	{"searches.delete by query", domain.OpSearchesDelete,
		map[string]any{"query": "plan"}, "/api/searches.delete", `{"query":"plan"}`},
	{"searches.update", domain.OpSearchesUpdate,
		map[string]any{"searchId": uuidOther, "score": 0.5}, "/api/searches.update", `{"id":"` + uuidOther + `","score":0.5}`},
	{"stars.create on a document", domain.OpStarsCreate,
		map[string]any{"documentId": "doc-1", "index": "P"}, "/api/stars.create", `{"documentId":"doc-1","index":"P"}`},
	{"stars.create on a collection", domain.OpStarsCreate,
		map[string]any{"collectionId": uuidCollection}, "/api/stars.create", `{"collectionId":"` + uuidCollection + `"}`},
	{"stars.delete", domain.OpStarsDelete,
		map[string]any{"starId": uuidOther}, "/api/stars.delete", `{"id":"` + uuidOther + `"}`},
	{"stars.update", domain.OpStarsUpdate,
		map[string]any{"starId": uuidOther, "index": "P"}, "/api/stars.update", `{"id":"` + uuidOther + `","index":"P"}`},
	{"subscriptions.create on a document", domain.OpSubscriptionsCreate,
		map[string]any{"documentId": "doc-1"}, "/api/subscriptions.create", `{"event":"documents.update","documentId":"doc-1"}`},
	{"subscriptions.create on a collection", domain.OpSubscriptionsCreate,
		map[string]any{"collectionId": uuidCollection}, "/api/subscriptions.create",
		`{"event":"documents.update","collectionId":"` + uuidCollection + `"}`},
	{"subscriptions.delete", domain.OpSubscriptionsDelete,
		map[string]any{"subscriptionId": uuidOther}, "/api/subscriptions.delete", `{"id":"` + uuidOther + `"}`},
	{"templates.create", domain.OpTemplatesCreate,
		map[string]any{"title": "Weekly", "collectionId": uuidCollection, "icon": "calendar", "color": "#fa0", "publish": false},
		"/api/templates.create",
		`{"title":"Weekly","collectionId":"` + uuidCollection + `","icon":"calendar","color":"#fa0","publish":false}`},
	{"templates.duplicate", domain.OpTemplatesDuplicate,
		map[string]any{"templateId": "tpl-1", "title": "Copy", "collectionId": uuidCollection},
		"/api/templates.duplicate", `{"id":"tpl-1","title":"Copy","collectionId":"` + uuidCollection + `"}`},
	{"templates.restore", domain.OpTemplatesRestore,
		map[string]any{"templateId": "tpl-1"}, "/api/templates.restore", `{"id":"tpl-1"}`},
	{"templates.update", domain.OpTemplatesUpdate,
		map[string]any{"templateId": "tpl-1", "title": "New", "icon": "star", "color": "#00ff00", "fullWidth": true,
			"collectionId": uuidCollection, "publish": true},
		"/api/templates.update",
		`{"id":"tpl-1","title":"New","icon":"star","color":"#00ff00","fullWidth":true,"collectionId":"` + uuidCollection +
			`","publish":true}`},
	{"userMemberships.update", domain.OpUserMembershipsUpdate,
		map[string]any{"membershipId": uuidOther, "index": "P"},
		"/api/userMemberships.update", `{"id":"` + uuidOther + `","index":"P"}`},
	{"users.notificationsSubscribe with no event type", domain.OpUsersNotificationsSubscribe,
		map[string]any{}, "/api/users.notificationsSubscribe", `{}`},
	{"users.notificationsSubscribe to one event", domain.OpUsersNotificationsSubscribe,
		map[string]any{"eventType": "documents.update"}, "/api/users.notificationsSubscribe",
		`{"eventType":"documents.update"}`},
	{"users.notificationsUnsubscribe from one event", domain.OpUsersNotificationsUnsubscribe,
		map[string]any{"eventType": "documents.update"}, "/api/users.notificationsUnsubscribe",
		`{"eventType":"documents.update"}`},
}

// ackRefusals are the 2xx replies an acknowledgement write must never turn into a success (finding m1): the {ok:true}
// shape the wiki does not send for an acknowledgement, a reply that says success false, and a reply that says nothing of
// success. Each one has an unknown outcome, and the write is sent once.
var ackRefusals = []struct {
	name string
	body string
}{
	{"the old ok shape", `{"ok":true,"status":200}`},
	{"success false", `{"success":false}`},
	{"no success", `{"status":200}`},
}

// TestCommentReactionsAnswerAnAcknowledgement checks the acknowledgement reply of the reaction writes: the wiki answers
// {success:true}, and Retriever answers ok true. Every other reply in ackRefusals is an unknown outcome.
func TestCommentReactionsAnswerAnAcknowledgement(t *testing.T) {
	for _, op := range []domain.OperationID{domain.OpCommentsAddReaction, domain.OpCommentsRemoveReaction} {
		r := writeRig(t, nil)
		r.stub.SetReply(string(endpointMethodOf(t, op)), http.StatusOK, `{"success":true}`)
		rec, err := r.gw.Write(context.Background(), op, domain.NewArgs(map[string]any{"commentId": uuidOther, "emoji": "\U0001F44D"}))
		if err != nil {
			t.Fatalf("%s: Write = %v", op, err)
		}
		if ok, _ := rec.Get("ok"); ok != true {
			t.Fatalf("%s: reply = %+v, want ok true", op, rec)
		}

		for _, tc := range ackRefusals {
			r = writeRig(t, nil)
			r.stub.SetReply(string(endpointMethodOf(t, op)), http.StatusOK, tc.body)
			_, err = r.gw.Write(context.Background(), op, domain.NewArgs(map[string]any{"commentId": uuidOther, "emoji": "\U0001F44D"}))
			if !errors.Is(err, domain.ErrWriteOutcomeUnknown) {
				t.Fatalf("%s: %s: error = %v, want ErrWriteOutcomeUnknown (finding m1)", op, tc.name, err)
			}
		}
	}
}

// endpointMethodOf returns the wire method of a write, for a test that states its reply.
func endpointMethodOf(t *testing.T, op domain.OperationID) string {
	t.Helper()
	ep, ok := writeEndpoints[op]
	if !ok {
		t.Fatalf("no write endpoint for %s", op)
	}
	return ep.method
}

// TestCollectionsMoveAcknowledgesTheIndexAndNeedsSuccess checks the acknowledgement reply of collections.move: the wiki
// answers {success:true, data:{index}}, Retriever answers ok true and the index the wiki kept. A reply that does not say
// success true is an unknown outcome, whatever index it names.
func TestCollectionsMoveAcknowledgesTheIndexAndNeedsSuccess(t *testing.T) {
	r := writeRig(t, nil)
	r.stub.SetReply("collections.move", http.StatusOK, `{"success":true,"data":{"index":"P"}}`)
	rec, err := r.gw.Write(context.Background(), domain.OpCollectionsMove, domain.NewArgs(map[string]any{"collectionId": "col-1", "index": "P"}))
	if err != nil {
		t.Fatalf("Write = %v", err)
	}
	if ok, _ := rec.Get("ok"); ok != true || rec.Str("index") != "P" {
		t.Fatalf("reply = %+v, want ok true and index P", rec)
	}

	for _, body := range []string{`{"ok":true,"status":200,"data":{"index":"P"}}`, `{"success":false,"data":{"index":"P"}}`,
		`{"data":{"index":"P"}}`} {
		r = writeRig(t, nil)
		r.stub.SetReply("collections.move", http.StatusOK, body)
		_, err = r.gw.Write(context.Background(), domain.OpCollectionsMove, domain.NewArgs(map[string]any{"collectionId": "col-1", "index": "P"}))
		if !errors.Is(err, domain.ErrWriteOutcomeUnknown) {
			t.Fatalf("a move reply %s: error = %v, want ErrWriteOutcomeUnknown (finding m1)", body, err)
		}
	}
}

// TestCollectionsExportNeedsSuccess checks the reply of collections.export: the wiki answers {success: true, data:
// {fileOperation}}, the acknowledgement that collections.export_all also answers. A reply that holds the file operation
// but does not say success true is an unknown outcome, whatever file operation it names (finding m1).
func TestCollectionsExportNeedsSuccess(t *testing.T) {
	const fileOperation = `"fileOperation":{"id":"a6000000-0000-4000-8000-000000000006","type":"export",` +
		`"format":"outline-markdown","name":"Wiki","state":"creating","error":null,"size":0,"collectionId":"col-1",` +
		`"documentId":null,"user":{"id":"u-1"},"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z"}`
	args := domain.NewArgs(map[string]any{"collectionId": "col-1"})
	for _, body := range []string{
		`{"data":{` + fileOperation + `}}`,
		`{"ok":true,"status":200,"data":{` + fileOperation + `}}`,
		`{"success":false,"data":{` + fileOperation + `}}`,
	} {
		r := writeRig(t, nil)
		r.stub.SetReply("collections.export", http.StatusOK, body)
		_, err := r.gw.Write(context.Background(), domain.OpCollectionsExport, args)
		if !errors.Is(err, domain.ErrWriteOutcomeUnknown) {
			t.Fatalf("a collections.export reply %s: error = %v, want ErrWriteOutcomeUnknown (finding m1)", body, err)
		}
	}
}

// TestCollectionsExportAnswersTheFileOperation checks that collections.export reads data.fileOperation of a reply that says
// success true, and that the download link of the file is never part of the answer (the inventory says none is offered).
func TestCollectionsExportAnswersTheFileOperation(t *testing.T) {
	r := writeRig(t, nil)
	r.stub.SetReply("collections.export", http.StatusOK,
		`{"success":true,"data":{"fileOperation":{"id":"a6000000-0000-4000-8000-000000000006","type":"export","format":"outline-markdown","name":"Wiki","state":"creating","error":null,"size":0,"collectionId":"col-1","documentId":null,"user":{"id":"u-1"},"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z"}}}`)
	rec, err := r.gw.Write(context.Background(), domain.OpCollectionsExport, domain.NewArgs(map[string]any{"collectionId": "col-1"}))
	if err != nil {
		t.Fatalf("Write = %v", err)
	}
	if rec.Str("id") != "a6000000-0000-4000-8000-000000000006" || rec.Str("state") != "creating" {
		t.Fatalf("reply = %+v, want the file operation a6000000-0000-4000-8000-000000000006 in state creating", rec)
	}
}

// updatedDocument is a documents.update reply of the router, with the revision the wiki now holds.
const updatedDocument = `{"data":{"id":"doc-1","title":"New","url":"/doc/new",` +
	`"collectionId":"col-1","revision":4,"text":"Body"},"policies":[]}`

// writeRig is a rig whose source is a plain outline source, so a write row reaches the wire. Its base is the in-process stub's
// .invalid name, so the stub answers it and refuses any host that is not a loopback or .invalid name.
func writeRig(t *testing.T, stubOpts []outlinestub.Option, gwOpts ...Option) rig {
	t.Helper()
	clock := newFakeClock()
	creds := &staticCreds{}
	stub := outlinestub.New(append([]outlinestub.Option{outlinestub.WithClock(clock.Now)}, stubOpts...)...)
	client := &http.Client{Transport: stub.RoundTripper()}
	opts := append([]Option{WithMinDelay(0), WithClock(clock.Now), WithSleep(clock.Sleep)}, gwOpts...)
	gw, err := NewWithBase(testSourceAt(t, testHost), testBase, client, creds, opts...)
	if err != nil {
		t.Fatalf("NewWithBase: %v", err)
	}
	return rig{stub: stub, clock: clock, creds: creds, gw: gw}
}

// TestTheWriteTableHoldsExactlyTheStageBWrites checks the write endpoint table against the stage b writes: one endpoint
// per write, and no other.
func TestTheWriteTableHoldsExactlyTheStageBWrites(t *testing.T) {
	want := []domain.OperationID{
		domain.OpDocumentsCreate, domain.OpDocumentsDuplicate, domain.OpDocumentsMove, domain.OpDocumentsRestore,
		domain.OpDocumentsTemplatize, domain.OpDocumentsUnpublish, domain.OpDocumentsUpdate,
		domain.OpCollectionsCreate, domain.OpCollectionsDuplicate, domain.OpCollectionsExport, domain.OpCollectionsMove,
		domain.OpCollectionsRestore, domain.OpCollectionsUpdate,
		domain.OpCommentsCreate, domain.OpCommentsUpdate, domain.OpCommentsResolve, domain.OpCommentsUnresolve,
		domain.OpCommentsAddReaction, domain.OpCommentsRemoveReaction,
		domain.OpAccessRequestsCreate, domain.OpRevisionsUpdate,
		domain.OpNotificationsUpdate, domain.OpNotificationsUpdateAll,
		domain.OpPinsCreate, domain.OpPinsDelete, domain.OpPinsUpdate,
		domain.OpSearchesDelete, domain.OpSearchesUpdate,
		domain.OpStarsCreate, domain.OpStarsDelete, domain.OpStarsUpdate,
		domain.OpSubscriptionsCreate, domain.OpSubscriptionsDelete,
		domain.OpTemplatesCreate, domain.OpTemplatesDuplicate, domain.OpTemplatesRestore, domain.OpTemplatesUpdate,
		domain.OpUserMembershipsUpdate,
		domain.OpUsersNotificationsSubscribe, domain.OpUsersNotificationsUnsubscribe,
	}
	got := make([]domain.OperationID, 0, len(writeEndpoints))
	for op := range writeEndpoints {
		got = append(got, op)
	}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("write endpoint table = %v, want %v", got, want)
	}
}

// TestEveryWriteSendsItsExactRequest checks each write's method path and JSON body, sent by an outline source.
func TestEveryWriteSendsItsExactRequest(t *testing.T) {
	for _, tc := range requestCasesB {
		t.Run(tc.name, func(t *testing.T) {
			gw, rec := recordingGatewayAt(t)
			_, _ = gw.Write(context.Background(), tc.op, domain.NewArgs(tc.args))
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

// TestEveryWriteIsAPostOfJSONWithTheAPIVersion checks the transport of a write: a POST with the JSON content type and
// X-API-Version 2, and no X-Client-Version header, which Retriever never sends.
func TestEveryWriteIsAPostOfJSONWithTheAPIVersion(t *testing.T) {
	r := writeRig(t, []outlinestub.Option{outlinestub.WithDocuments(outlinestub.Document{ID: "doc-1", Title: "Plan"})})
	r.stub.SetReply("documents.update", http.StatusOK, updatedDocument)
	if _, err := r.gw.Write(context.Background(), domain.OpDocumentsUpdate, domain.NewArgs(map[string]any{"documentId": "doc-1", "title": "New"})); err != nil {
		t.Fatalf("Write = %v", err)
	}
	reqs := r.stub.Requests()
	if len(reqs) != 1 {
		t.Fatalf("requests = %d, want 1", len(reqs))
	}
	req := reqs[0]
	if req.HTTPMethod != http.MethodPost || req.ContentType != "application/json" || req.APIVersion != "2" || req.ClientVersion != "" {
		t.Fatalf("request = %s, content type %q, X-API-Version %q, X-Client-Version %q; want a POST of JSON with version 2 and no client version",
			req.HTTPMethod, req.ContentType, req.APIVersion, req.ClientVersion)
	}
}

// TestAWriteIsNeverRetriedAfterATimeoutOrADroppedConnection checks the retry rule: after a timeout or a dropped
// connection the outcome is unknown, so the write returns ErrWriteOutcomeUnknown after one request, never a retry.
func TestAWriteIsNeverRetriedAfterATimeoutOrADroppedConnection(t *testing.T) {
	for _, fault := range []outlinestub.Fault{{Hang: true}, {Drop: true}} {
		r := writeRig(t, nil)
		r.stub.SetFault(fault)
		_, err := r.gw.Write(context.Background(), domain.OpDocumentsCreate, domain.NewArgs(map[string]any{"title": "Plan"}))
		if !errors.Is(err, domain.ErrWriteOutcomeUnknown) {
			t.Fatalf("fault %+v: error = %v, want ErrWriteOutcomeUnknown", fault, err)
		}
		if n := r.stub.Calls("documents.create"); n != 1 {
			t.Fatalf("fault %+v: documents.create calls = %d, want 1 (no retry of a write)", fault, n)
		}
		if w := r.clock.recordedSleeps(); len(w) != 0 {
			t.Fatalf("fault %+v: waits = %v, want none", fault, w)
		}
	}
}

// TestAWriteRetriesOnlyAfterA429 checks the retry rules: a 429 means the wiki refused the write before it
// ran, so the client waits Retry-After and sends it again, once, and then succeeds.
func TestAWriteRetriesOnlyAfterA429(t *testing.T) {
	r := writeRig(t, nil)
	r.stub.SetFaultFirst(1, outlinestub.Fault{Status: http.StatusTooManyRequests, RetryAfter: "1"})
	r.stub.SetReply("documents.update", http.StatusOK, updatedDocument)

	rec, err := r.gw.Write(context.Background(), domain.OpDocumentsUpdate, domain.NewArgs(map[string]any{"documentId": "doc-1", "title": "New"}))
	if err != nil {
		t.Fatalf("Write = %v", err)
	}
	if got := rec.Str("title"); got != "New" {
		t.Fatalf("reply title = %q", got)
	}
	if n := r.stub.Calls("documents.update"); n != 2 {
		t.Fatalf("documents.update calls = %d, want 2 (one 429 and one retry)", n)
	}
	if w := r.clock.recordedSleeps(); !equalDurations(w, []time.Duration{time.Second}) {
		t.Fatalf("waits = %v, want [1s]", w)
	}
}

// TestAWriteStopsAfterThreeRetriesOf429 checks that a write that keeps meeting 429 stops after the same bounded retries
// as a read, and reports the rate limit.
func TestAWriteStopsAfterThreeRetriesOf429(t *testing.T) {
	r := writeRig(t, nil)
	r.stub.SetFault(outlinestub.Fault{Status: http.StatusTooManyRequests, RetryAfter: "1"})
	_, err := r.gw.Write(context.Background(), domain.OpDocumentsUpdate, domain.NewArgs(map[string]any{"documentId": "doc-1", "title": "New"}))
	if !errors.Is(err, domain.ErrSourceRateLimited) {
		t.Fatalf("error = %v, want ErrSourceRateLimited", err)
	}
	if n := r.stub.Calls("documents.update"); n != 4 {
		t.Fatalf("documents.update calls = %d, want 4 (one try and three retries)", n)
	}
}

// TestA409DocumentConflictIsItsOwnErrorAndIsNotRetried checks the last_revision refusal: a
// 409 with document_conflict is ErrDocumentConflict after one request, and any other 409 is an unexpected reply.
func TestA409DocumentConflictIsItsOwnErrorAndIsNotRetried(t *testing.T) {
	cases := []struct {
		name string
		body string
		want error
	}{
		{"document_conflict", `{"ok":false,"status":409,"error":"document_conflict","message":"the document changed"}`, domain.ErrDocumentConflict},
		{"another conflict", `{"ok":false,"status":409,"error":"validation_error","message":"no"}`, domain.ErrUnexpectedResponse},
		{"a conflict with no body", ``, domain.ErrUnexpectedResponse},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := writeRig(t, nil)
			r.stub.SetReply("documents.update", http.StatusConflict, tc.body)
			_, err := r.gw.Write(context.Background(), domain.OpDocumentsUpdate, domain.NewArgs(map[string]any{"documentId": "doc-1", "title": "New", "lastRevision": 3}))
			if !errors.Is(err, tc.want) {
				t.Fatalf("error = %v, want %v", err, tc.want)
			}
			if n := r.stub.Calls("documents.update"); n != 1 {
				t.Fatalf("documents.update calls = %d, want 1 (a conflict is not retried)", n)
			}
		})
	}
}

// TestTheWrongClassIsRefusedInBothDirections checks that a read sent to the write path, a write sent to the read path,
// and an unknown operation are each refused before any request.
func TestTheWrongClassIsRefusedInBothDirections(t *testing.T) {
	r := writeRig(t, sampleOptions())
	ctx := context.Background()
	if _, err := r.gw.Write(ctx, domain.OpDocumentsList, domain.NewArgs(nil)); !errors.Is(err, domain.ErrWrongOperationClass) {
		t.Fatalf("Write of a read: error = %v, want ErrWrongOperationClass", err)
	}
	if _, err := r.gw.Read(ctx, domain.OpDocumentsUpdate, domain.NewArgs(map[string]any{"documentId": "doc-1"})); !errors.Is(err, domain.ErrWrongOperationClass) {
		t.Fatalf("Read of a write: error = %v, want ErrWrongOperationClass", err)
	}
	if _, err := r.gw.Write(ctx, domain.OperationID("nope.nope"), domain.NewArgs(nil)); !errors.Is(err, domain.ErrUnsupportedOperation) {
		t.Fatalf("Write of an unknown operation: error = %v, want ErrUnsupportedOperation", err)
	}
	if n := len(r.stub.Requests()); n != 0 {
		t.Fatalf("requests = %d, want 0", n)
	}
}

// TestAWriteRefusesBadArgumentsBeforeAnyRequest checks that the write path binds its arguments with the catalog rules.
func TestAWriteRefusesBadArgumentsBeforeAnyRequest(t *testing.T) {
	cases := []struct {
		name string
		raw  map[string]any
		want error
	}{
		{"a document ID with a dot", map[string]any{"documentId": "a.b", "title": "New"}, domain.ErrInvalidDocumentID},
		{"no document", map[string]any{"title": "New"}, domain.ErrMissingArgument},
		{"an edit mode outside the list", map[string]any{"documentId": "doc-1", "editMode": "merge"}, domain.ErrInvalidEnum},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := writeRig(t, nil)
			if _, err := r.gw.Write(context.Background(), domain.OpDocumentsUpdate, domain.NewArgs(tc.raw)); !errors.Is(err, tc.want) {
				t.Fatalf("error = %v, want %v", err, tc.want)
			}
			if n := len(r.stub.Requests()); n != 0 {
				t.Fatalf("requests = %d, want 0", n)
			}
		})
	}
}

// TestAWriteReplyIsProjectedToTheRowFields checks the reply of documents.update: the document with its revision, which
// last_revision names on the next update.
func TestAWriteReplyIsProjectedToTheRowFields(t *testing.T) {
	r := writeRig(t, nil)
	r.stub.SetReply("documents.update", http.StatusOK, updatedDocument)
	rec, err := r.gw.Write(context.Background(), domain.OpDocumentsUpdate, domain.NewArgs(map[string]any{"documentId": "doc-1", "title": "New"}))
	if err != nil {
		t.Fatalf("Write = %v", err)
	}
	var names []string
	for _, f := range rec.Fields() {
		names = append(names, f.Name)
	}
	if want := fieldsOf(t, domain.OpDocumentsUpdate); !slices.Equal(names, want) {
		t.Fatalf("fields = %v, want exactly the row fields %v", names, want)
	}
	if v, _ := rec.Get("revision"); v != 4 {
		t.Fatalf("revision = %v, want 4", v)
	}
}

// TestAListWriteAnswersTheDocumentsThatMoved checks that documents.move reads its list under data.documents.
func TestAListWriteAnswersTheDocumentsThatMoved(t *testing.T) {
	r := writeRig(t, nil)
	r.stub.SetReply("documents.move", http.StatusOK,
		`{"data":{"documents":[{"id":"doc-1","title":"Plan","url":"/doc/plan","collectionId":"col-2","revision":2}],"collections":[]},"policies":[]}`)
	rec, err := r.gw.Write(context.Background(), domain.OpDocumentsMove, domain.NewArgs(map[string]any{"documentId": "doc-1", "index": 0}))
	if err != nil {
		t.Fatalf("Write = %v", err)
	}
	if !rec.IsList() || len(rec.Items()) != 1 || rec.Items()[0].Str("collectionId") != "col-2" {
		t.Fatalf("reply = %+v, want one moved document in col-2", rec)
	}
}

// TestDuplicateAnswersTheCopiesAsAList checks that documents.duplicate reads its copies as a list under data.documents, with
// the document fields of a write and the policies beside them. The copies are a list because the wiki copies a document
// and, with recursive true, its children, so the row declares a list and never one document.
func TestDuplicateAnswersTheCopiesAsAList(t *testing.T) {
	r := writeRig(t, nil)
	r.stub.SetReply("documents.duplicate", http.StatusOK,
		`{"data":{"documents":[`+
			`{"id":"doc-3","title":"Plan (copy)","url":"/doc/plan-copy","collectionId":"col-1","revision":1},`+
			`{"id":"doc-4","title":"Child (copy)","url":"/doc/child-copy","collectionId":"col-1","revision":1}]},"policies":[]}`)
	rec, err := r.gw.Write(context.Background(), domain.OpDocumentsDuplicate, domain.NewArgs(map[string]any{"documentId": "doc-1", "recursive": true}))
	if err != nil {
		t.Fatalf("Write = %v", err)
	}
	if !rec.IsList() || len(rec.Items()) != 2 {
		t.Fatalf("reply = %+v, want the two copies as a list", rec)
	}
	if got := rec.Items()[1].Str("id"); got != "doc-4" {
		t.Fatalf("the second copy is %q, want doc-4", got)
	}
	if want := fieldsOf(t, domain.OpDocumentsDuplicate); !slices.Equal(fieldNames(rec.Items()[0]), want) {
		t.Fatalf("fields = %v, want exactly the row fields %v", fieldNames(rec.Items()[0]), want)
	}
}

// TestTheNonLoopbackGuardCatchesAWrite is the control of the proof above: a write to a real-looking host is refused by
// the stub and counted, so the proof test cannot pass by accident.
func TestTheNonLoopbackGuardCatchesAWrite(t *testing.T) {
	stub := outlinestub.New(sampleOptions()...)
	client := &http.Client{Transport: stub.RoundTripper()}
	gw, err := NewWithBase(testSourceAt(t, testHost), "https://"+testHost, client, &staticCreds{}, WithMinDelay(0))
	if err != nil {
		t.Fatalf("NewWithBase: %v", err)
	}
	if _, err := gw.Write(context.Background(), domain.OpDocumentsCreate, domain.NewArgs(map[string]any{"title": "Plan"})); err == nil {
		t.Fatal("a write to a real-looking host returned no error")
	}
	if n := stub.Refused(); n != 1 {
		t.Fatalf("refused = %d, want 1", n)
	}
	if n := len(stub.Requests()); n != 0 {
		t.Fatalf("requests answered = %d, want 0", n)
	}
}

// TestNotificationsUpdateAllAnswersItsTotal checks the reply of notifications.update_all: the wiki answers
// {success:true, data:{total}}, Retriever answers ok true and the total it names. A reply that does not say success true is an
// unknown outcome, whatever total it names.
func TestNotificationsUpdateAllAnswersItsTotal(t *testing.T) {
	args := domain.NewArgs(map[string]any{"archivedAt": "2026-01-02T03:04:05Z"})
	r := writeRig(t, nil)
	r.stub.SetReply("notifications.update_all", http.StatusOK, `{"success":true,"data":{"total":3}}`)
	rec, err := r.gw.Write(context.Background(), domain.OpNotificationsUpdateAll, args)
	if err != nil {
		t.Fatalf("Write = %v", err)
	}
	if ok, _ := rec.Get("ok"); ok != true {
		t.Fatalf("reply ok = %v, want true", ok)
	}
	if total, _ := rec.Get("total"); total != 3 {
		t.Fatalf("reply total = %v, want 3", total)
	}

	for _, body := range []string{`{"ok":true,"status":200,"data":{"total":3}}`, `{"success":false,"data":{"total":3}}`,
		`{"data":{"total":3}}`} {
		r = writeRig(t, nil)
		r.stub.SetReply("notifications.update_all", http.StatusOK, body)
		_, err = r.gw.Write(context.Background(), domain.OpNotificationsUpdateAll, args)
		if !errors.Is(err, domain.ErrWriteOutcomeUnknown) {
			t.Fatalf("an update_all reply %s: error = %v, want ErrWriteOutcomeUnknown (finding m1)", body, err)
		}
	}
}

// TestDeletesOfAUsersOwnObjectsAnswerAnAcknowledgement checks the deletes that remove a bookmark-like object of the
// user (a pin, a saved search, a star, a subscription): each answers {success:true}, and every reply in ackRefusals is
// an unknown outcome, never a success.
func TestDeletesOfAUsersOwnObjectsAnswerAnAcknowledgement(t *testing.T) {
	cases := []struct {
		op   domain.OperationID
		args map[string]any
	}{
		{domain.OpPinsDelete, map[string]any{"pinId": uuidOther}},
		{domain.OpSearchesDelete, map[string]any{"searchId": uuidOther}},
		{domain.OpStarsDelete, map[string]any{"starId": uuidOther}},
		{domain.OpSubscriptionsDelete, map[string]any{"subscriptionId": uuidOther}},
	}
	for _, tc := range cases {
		r := writeRig(t, nil)
		r.stub.SetReply(endpointMethodOf(t, tc.op), http.StatusOK, `{"success":true}`)
		rec, err := r.gw.Write(context.Background(), tc.op, domain.NewArgs(tc.args))
		if err != nil {
			t.Fatalf("%s: Write = %v", tc.op, err)
		}
		if ok, _ := rec.Get("ok"); ok != true {
			t.Fatalf("%s: reply = %+v, want ok true", tc.op, rec)
		}

		for _, refusal := range ackRefusals {
			r = writeRig(t, nil)
			r.stub.SetReply(endpointMethodOf(t, tc.op), http.StatusOK, refusal.body)
			_, err = r.gw.Write(context.Background(), tc.op, domain.NewArgs(tc.args))
			if !errors.Is(err, domain.ErrWriteOutcomeUnknown) {
				t.Fatalf("%s: %s: error = %v, want ErrWriteOutcomeUnknown (finding m1)", tc.op, refusal.name, err)
			}
		}
	}
}

// TestUserNotificationSubscriptionsAnswerTheSettingsButNoEmail checks the reply of users.notificationsSubscribe: the
// user with its notification settings, which the inventory keeps for this reply, and never the e-mail address.
func TestUserNotificationSubscriptionsAnswerTheSettingsButNoEmail(t *testing.T) {
	r := writeRig(t, nil)
	r.stub.SetReply("users.notificationsSubscribe", http.StatusOK,
		`{"data":{"id":"`+uuidUser+`","name":"Ann","role":"member","email":"ann@example.invalid",`+
			`"isSuspended":false,"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z",`+
			`"notificationSettings":{"documents.update":true,"comments.create":false}}}`)
	rec, err := r.gw.Write(context.Background(), domain.OpUsersNotificationsSubscribe,
		domain.NewArgs(map[string]any{"eventType": "documents.update"}))
	if err != nil {
		t.Fatalf("Write = %v", err)
	}
	var names []string
	for _, f := range rec.Fields() {
		names = append(names, f.Name)
	}
	if want := fieldsOf(t, domain.OpUsersNotificationsSubscribe); !slices.Equal(names, want) {
		t.Fatalf("fields = %v, want exactly the row fields %v", names, want)
	}
	if _, found := rec.Get("email"); found {
		t.Fatal("the reply names an e-mail address, which no projection may hold")
	}
	if v, _ := rec.Rec("notificationSettings").Get("documents.update"); v != true {
		t.Fatalf("notificationSettings documents.update = %v, want true", v)
	}
}

// endpointOf returns the endpoint of an operation, a read or a write, for the tests that walk the whole catalog.
func endpointOf(op domain.OperationID) (endpoint, bool) {
	if ep, ok := endpoints[op]; ok {
		return ep, true
	}
	if ep, ok := writeEndpoints[op]; ok {
		return ep, true
	}
	ep, ok := adminEndpoints[op]
	return ep, ok
}

// fieldsOf returns the field names of the row of op, for the message of a failed projection check.
func fieldsOf(t *testing.T, op domain.OperationID) []string {
	t.Helper()
	row, ok := operations.Lookup(op)
	if !ok {
		t.Fatalf("no row for %s", op)
	}
	return row.Fields
}
