package outline

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// decodeJSON decodes a reply body the way the client does, into a generic value.
func decodeJSON(t *testing.T, body string) any {
	t.Helper()
	var root any
	if err := json.Unmarshal([]byte(body), &root); err != nil {
		t.Fatalf("decode %q: %v", body, err)
	}
	return root
}

// TestInterpretMapsEachStatusToItsSentinel is a table of the HTTP replies the client maps (finding M3). A reply that
// names a sentinel carries no body text, and a 404 is told apart by its message: the router's catch-all means the
// method is not offered, and any other 404 is the missing document or collection of the method.
func TestInterpretMapsEachStatusToItsSentinel(t *testing.T) {
	const catchAll = `{"ok":false,"error":"not_found","status":404,"message":"Endpoint not found"}`
	const missing = `{"ok":false,"error":"not_found","status":404,"message":"Document not found"}`
	cases := []struct {
		name       string
		method     string
		status     int
		retryAfter string
		body       string
		want       error
	}{
		{"catch-all 404 is an unavailable endpoint", methodCollectionsList, 404, "", catchAll,
			domain.OperationUnavailableError{Method: methodCollectionsList}},
		// The webhooks plugin mounts this method at v1.12.0. An older server or one without the plugin answers the same
		// catch-all, so the method reports itself as unavailable.
		{"catch-all 404 on webhookSubscriptions.list is an unavailable endpoint", methodWebhookSubscriptionsList, 404, "", catchAll,
			domain.OperationUnavailableError{Method: methodWebhookSubscriptionsList}},
		{"404 on collections.info is an unknown collection", methodCollectionsInfo, 404, "", missing, domain.ErrCollectionNotFound},
		{"404 on collections.documents is an unknown collection", methodCollectionsDocuments, 404, "", missing, domain.ErrCollectionNotFound},
		{"404 on documents.info is an unknown document", methodDocumentsInfo, 404, "", missing, domain.ErrDocumentNotFound},
		{"404 on documents.export is an unknown document", methodDocumentsExport, 404, "", missing, domain.ErrDocumentNotFound},
		// ADR 007 section 6.2: any 404 that is not the catch-all is not found, on a list as on an object, never unexpected.
		{"404 on a list with another message is not found", methodCollectionsList, 404, "", missing, domain.ErrNotFound},
		{"404 on documents.list with another message is not found", methodDocumentsList, 404, "", missing, domain.ErrNotFound},
		{"404 on a revision read is not found", methodRevisionsInfo, 404, "", missing, domain.ErrNotFound},
		// The status decides a 404 that is not the catch-all: its body is never read for the sentinel, so an HTML 404 on a
		// document method is an unknown document, as a JSON 404 is.
		{"404 with an HTML body on documents.info is an unknown document", methodDocumentsInfo, 404, "", "<html>not found</html>", domain.ErrDocumentNotFound},
		// Outline answers 204 with no body when a pin or a subscription does not exist (routes/api/pins and
		// routes/api/subscriptions, v1.12.0). That is an empty answer, so it is not found, never an unexpected reply.
		{"204 on pins.info with no body is not found", methodPinsInfo, 204, "", "", domain.ErrNotFound},
		{"204 on subscriptions.info with no body is not found", methodSubscriptionsInfo, 204, "", "", domain.ErrNotFound},
		{"204 on documents.info is an unknown document", methodDocumentsInfo, 204, "", "", domain.ErrDocumentNotFound},
		{"204 on a list is not found, never an empty success", methodCollectionsList, 204, "", "", domain.ErrNotFound},
		{"400 is a bad request", methodDocumentsList, 400, "", `{"message":"invalid"}`, domain.ErrBadRequest},
		{"401 is a rejected key", methodAuthInfo, 401, "", "", domain.ErrKeyRejected},
		{"403 is denied access", methodAuthInfo, 403, "", "", domain.ErrAccessDenied},
		{"429 without Retry-After is rate limited", methodDocumentsList, 429, "", "", domain.ErrSourceRateLimited},
		{"429 with a Retry-After that is not a number is rate limited", methodDocumentsList, 429, "soon", "", domain.ErrSourceRateLimited},
		{"429 with a Retry-After over 60 seconds is refused", methodDocumentsList, 429, "61", "", domain.ErrRetryAfterTooLong},
		{"500 is an upstream failure", methodDocumentsSearch, 500, "", "", domain.ErrUpstreamFailure},
		{"503 is an upstream failure", methodDocumentsSearch, 503, "", "", domain.ErrUpstreamFailure},
		{"302 is a redirect that is not followed", methodAuthInfo, 302, "", "", domain.ErrRedirected},
		{"an unlisted status is unexpected", methodAuthInfo, 418, "", "", domain.ErrUnexpectedResponse},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := interpret(tc.method, tc.status, tc.retryAfter, []byte(tc.body), new(any))
			if !errors.Is(err, tc.want) {
				t.Fatalf("interpret(%d) error = %v, want %v", tc.status, err, tc.want)
			}
		})
	}
}

// TestInterpretAcceptsARetryAfterWithinTheLimit checks that a rate limit with a short Retry-After is retried after the
// stated wait, and that a success decodes the reply.
func TestInterpretAcceptsARetryAfterWithinTheLimit(t *testing.T) {
	wait, retry, err := interpret(methodDocumentsList, 429, "2", nil, new(any))
	if err != nil || !retry || wait.Seconds() != 2 {
		t.Fatalf("interpret(429, 2) = (%v, %v, %v), want a 2 second retry", wait, retry, err)
	}
	var out any
	if _, _, err := interpret(methodAuthInfo, 200, "", []byte(`{"data":{}}`), &out); err != nil {
		t.Fatalf("interpret(200) error = %v", err)
	}
	if _, _, err := interpret(methodAuthInfo, 200, "", []byte("not json"), &out); !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("interpret(200, not JSON) error = %v, want ErrUnexpectedResponse", err)
	}
}

// TestProjectionsTolerateNullAndMissingFields checks the tolerant projections (finding M3): a null or missing name is
// empty text, and a null child list is no child.
func TestProjectionsTolerateNullAndMissingFields(t *testing.T) {
	t.Run("a null name is empty text", func(t *testing.T) {
		rec, err := projectCollectionInfo(decodeJSON(t, `{"data":{"id":"col-1","name":null}}`))
		if err != nil || rec.Str("name") != "" || rec.Str("id") != "col-1" {
			t.Fatalf("projectCollectionInfo = %+v, %v; want id col-1 and an empty name", rec, err)
		}
	})
	t.Run("a missing name is empty text", func(t *testing.T) {
		rec, err := projectCollectionInfo(decodeJSON(t, `{"data":{"id":"col-1"}}`))
		if err != nil || rec.Str("name") != "" {
			t.Fatalf("projectCollectionInfo = %+v, %v; want an empty name", rec, err)
		}
	})
	t.Run("a null children list is a leaf", func(t *testing.T) {
		rec, err := projectTree(decodeJSON(t, `{"data":[{"id":"doc-1","title":"A","url":"/a","children":null}]}`))
		if err != nil || len(rec.Items()) != 1 || len(rec.Items()[0].Fields()) != 5 {
			t.Fatalf("projectTree = %+v, %v; want one node", rec, err)
		}
	})
	t.Run("an account without a user has empty names", func(t *testing.T) {
		rec, err := projectAccount(decodeJSON(t, `{"data":{"team":{"name":"T"}}}`))
		if err != nil || rec.Rec("user").Str("name") != "" || rec.Rec("team").Str("name") != "T" {
			t.Fatalf("projectAccount = %+v, %v; want an empty user name and team T", rec, err)
		}
	})
	t.Run("a document without text has a nil markdown", func(t *testing.T) {
		rec, err := projectDocument(decodeJSON(t, `{"data":{"id":"doc-1","title":"A","url":"/a","collectionId":"col-1"}}`))
		if err != nil {
			t.Fatalf("projectDocument error = %v", err)
		}
		if v, ok := rec.Get("markdown"); !ok || v != nil {
			t.Fatalf("markdown = %v (present %v), want nil", v, ok)
		}
	})
}

// TestProjectionsRefuseAWrongShape checks that a reply of the wrong type is an unexpected response, never a guessed value
// (finding M3).
func TestProjectionsRefuseAWrongShape(t *testing.T) {
	cases := []struct {
		name    string
		project func(any) (domain.Record, error)
		body    string
	}{
		{"collections.info with a string data", projectCollectionInfo, `{"data":"oops"}`},
		{"collections.info with a numeric id", projectCollectionInfo, `{"data":{"id":5,"name":"x"}}`},
		{"collections.list with an object data", projectCollectionList, `{"data":{}}`},
		{"collections.list with a non-object item", projectCollectionList, `{"data":[1]}`},
		{"collections.documents with an object data", projectTree, `{"data":{}}`},
		{"documents.list with an object data", projectDocumentList, `{"data":{}}`},
		{"documents.search with a hit that has no document", projectSearch, `{"data":[{"context":"x"}]}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := tc.project(decodeJSON(t, tc.body)); err == nil {
				t.Fatalf("projection of %s returned no error", tc.body)
			}
		})
	}
}
