package outline

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-testkit/outlinestub"
	"net/http"
	"testing"
)

// requestCasesC3 are the request-shape cases of slice c3: the access requests, attachments, sign-in providers, comments,
// emojis, file operations, imports, and integrations rows. Each body is written by hand from the field
// reference and the wire facts. The two update rows that replace stored values send the values read before the change (the
// pre-reads of adminPreReads), so their bodies carry the values of the synthetic reads of the info methods.
var requestCasesC3 = []requestCase{
	{"access-requests.approve", domain.OpAccessRequestsApprove,
		map[string]any{"accessRequestId": uuidOther, "permission": "read_write"},
		"/api/accessRequests.approve", `{"id":"` + uuidOther + `","permission":"read_write"}`},
	{"access-requests.dismiss", domain.OpAccessRequestsDismiss,
		map[string]any{"accessRequestId": uuidOther}, "/api/accessRequests.dismiss", `{"id":"` + uuidOther + `"}`},
	{"attachments.delete", domain.OpAttachmentsDelete,
		map[string]any{"attachmentId": uuidOther}, "/api/attachments.delete", `{"id":"` + uuidOther + `"}`},
	{"authentication-providers.delete", domain.OpAuthenticationProvidersDelete,
		map[string]any{"authenticationProviderId": uuidOther}, "/api/authenticationProviders.delete",
		`{"id":"` + uuidOther + `"}`},
	{"authentication-providers.start-group-sync", domain.OpAuthenticationProvidersStartGroupSync,
		map[string]any{"authenticationProviderId": uuidOther}, "/api/authenticationProviders.startGroupSync",
		`{"id":"` + uuidOther + `"}`},
	{"authentication-providers.update", domain.OpAuthenticationProvidersUpdate,
		map[string]any{"authenticationProviderId": uuidOther, "isEnabled": false, "groupClaim": "roles"},
		"/api/authenticationProviders.update",
		`{"id":"` + uuidOther + `","isEnabled":false,"settings":{"groupSyncEnabled":true,"groupClaim":"roles","groupSyncScopes":["read"]}}`},
	{"comments.delete", domain.OpCommentsDelete,
		map[string]any{"commentId": uuidOther}, "/api/comments.delete", `{"id":"` + uuidOther + `"}`},
	{"emojis.delete", domain.OpEmojisDelete,
		map[string]any{"emojiId": uuidOther}, "/api/emojis.delete", `{"id":"` + uuidOther + `"}`},
	{"file-operations.delete", domain.OpFileOperationsDelete,
		map[string]any{"fileOperationId": uuidOther}, "/api/fileOperations.delete", `{"id":"` + uuidOther + `"}`},
	{"imports.cancel", domain.OpImportsCancel,
		map[string]any{"importId": uuidOther}, "/api/imports.cancel", `{"id":"` + uuidOther + `"}`},
	{"imports.delete", domain.OpImportsDelete,
		map[string]any{"importId": uuidOther}, "/api/imports.delete", `{"id":"` + uuidOther + `"}`},
	{"integrations.create with a channel", domain.OpIntegrationsCreate,
		map[string]any{"url": "https://matomo.example.com/SYNTHETICPATHTOKEN", "type": "analytics", "service": "matomo", "channel": "general"},
		"/api/integrations.create",
		`{"type":"analytics","service":"matomo","settings":{"url":"https://matomo.example.com/SYNTHETICPATHTOKEN","channel":"general"}}`},
	{"integrations.create without a channel", domain.OpIntegrationsCreate,
		map[string]any{"url": "https://hooks.example.test/x", "type": "post", "service": "umami"},
		"/api/integrations.create", `{"type":"post","service":"umami","settings":{"url":"https://hooks.example.test/x"}}`},
	{"integrations.delete", domain.OpIntegrationsDelete,
		map[string]any{"integrationId": uuidOther}, "/api/integrations.delete", `{"id":"` + uuidOther + `"}`},
	{"integrations.update", domain.OpIntegrationsUpdate,
		map[string]any{"integrationId": uuidOther, "url": "https://new.example.test/x"}, "/api/integrations.update",
		`{"id":"` + uuidOther + `","settings":{"url":"https://new.example.test/x","measurementId":"M-1",` +
			`"instanceUrl":"https://matomo.example.com/SYNTHETICPATHTOKEN","scriptName":"m.js"},"events":[]}`},
}

// adminPreReads names the read that a change of this slice sends first. The wiki replaces some stored values when a request
// leaves them out, so Retriever reads the object and fills the values the caller did not set. The read is an admin read of the same source.
var adminPreReads = map[domain.OperationID]domain.OperationID{
	domain.OpAuthenticationProvidersUpdate: domain.OpAuthenticationProvidersGet,
	domain.OpIntegrationsUpdate:            domain.OpIntegrationsGet,
}

// adminPreReadsOf returns the reads an operation sends before its change: none, or the one of adminPreReads.
func adminPreReadsOf(op domain.OperationID) []domain.OperationID {
	if pre, ok := adminPreReads[op]; ok {
		return []domain.OperationID{pre}
	}
	return nil
}

// adminRecorder returns an admin gateway whose requests are recorded, with the stub that answers them. A test scripts the stub's
// replies, such as the pre-read of an update.
func adminRecorder(t *testing.T) (*Gateway, *wireRecorder, *outlinestub.Stub) {
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
	return gw, rec, stub
}

// bodyOf decodes a JSON text to the value a recorded request body holds, for a deep comparison.
func bodyOf(t *testing.T, text string) any {
	t.Helper()
	var v any
	if err := json.Unmarshal([]byte(text), &v); err != nil {
		t.Fatalf("bad JSON %s: %v", text, err)
	}
	return v
}

// TestSliceC3StartGroupSyncRefusesAReplyWithoutItsRedirect checks that a sync start whose reply names no redirect address is
// an unexpected reply, so the call never reports a start it cannot see.
func TestSliceC3StartGroupSyncRefusesAReplyWithoutItsRedirect(t *testing.T) {
	for _, reply := range []string{`{"data":{}}`, `{"data":{"url":""}}`, `{"success":true}`} {
		r := adminRig(t)
		r.stub.SetReply(adminEndpoints[domain.OpAuthenticationProvidersStartGroupSync].method, http.StatusOK, reply)
		_, err := r.gw.Write(context.Background(), domain.OpAuthenticationProvidersStartGroupSync,
			domain.NewArgs(map[string]any{"authenticationProviderId": uuidOther}))
		if err == nil {
			t.Fatalf("a sync start with %s was accepted", reply)
		}
	}
}

// TestSliceC3AckWithoutSuccessIsAnOutcomeUnknown checks that an acknowledgement of the slice that the wiki did not confirm is an
// unknown outcome, and that the change is never retried (finding m1).
func TestSliceC3AckWithoutSuccessIsAnOutcomeUnknown(t *testing.T) {
	cases := []struct {
		op    domain.OperationID
		reply string
	}{
		{domain.OpAttachmentsDelete, `{"ok":true}`},
		{domain.OpCommentsDelete, `{"status":200}`},
		{domain.OpEmojisDelete, `{"success":false}`},
		{domain.OpFileOperationsDelete, `{"ok":true}`},
		{domain.OpImportsDelete, `{"ok":true}`},
		{domain.OpIntegrationsDelete, `{"status":200}`},
		{domain.OpAuthenticationProvidersDelete, `{"success":false}`},
	}
	for _, tc := range cases {
		r := adminRig(t)
		method := adminEndpoints[tc.op].method
		r.stub.SetReply(method, http.StatusOK, tc.reply)
		_, err := r.gw.Write(context.Background(), tc.op, domain.NewArgs(adminArgsFor(tc.op)))
		if !errors.Is(err, domain.ErrWriteOutcomeUnknown) {
			t.Fatalf("%s with %s: error = %v, want ErrWriteOutcomeUnknown", tc.op, tc.reply, err)
		}
		if n := r.stub.Calls(method); n != 1 {
			t.Fatalf("%s: calls = %d, want 1", tc.op, n)
		}
	}
}

// TestSliceC3IntegrationsUpdateSendsNothingWhenTheReadFails checks that a change whose pre-read fails is never sent: the read
// is needed for the events, so an update without it would reset them.
func TestSliceC3IntegrationsUpdateSendsNothingWhenTheReadFails(t *testing.T) {
	gw, rec, stub := adminRecorder(t)
	stub.SetReply(endpoints[domain.OpIntegrationsGet].method, http.StatusNotFound, `{"ok":false,"status":404,"error":"not_found"}`)
	_, err := gw.Write(context.Background(), domain.OpIntegrationsUpdate, domain.NewArgs(map[string]any{
		"integrationId": uuidOther, "url": "https://new.example.test/x",
	}))
	if err == nil {
		t.Fatal("an update whose read failed succeeded")
	}
	for _, c := range rec.recorded() {
		if c.Path == "/api/integrations.update" {
			t.Fatal("the update was sent after its read failed")
		}
	}
}

// integrationInfoFixture is the reply of integrations.info for the integration of the update tests. Its settings hold keys that
// the update must resend unchanged (measurementId, instanceUrl, scriptName, and a nested object), the channel and channel ID
// the update keeps, and the address the update replaces. Its events are kept too. The values are synthetic.
const integrationInfoFixture = `{"ok":true,"status":200,"data":{"id":"` + uuidOther + `","type":"analytics","service":"matomo",` +
	`"events":["documents.create","documents.update"],"collectionId":null,"userId":"33333333-3333-4333-8333-333333333333",` +
	`"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-02T00:00:00Z",` +
	`"settings":{"url":"https://old.example.test/OLDPATHTOKEN","channel":"general","channelId":"C-1",` +
	`"measurementId":"M-7","instanceUrl":"https://matomo.example.test/KEEPPATH","scriptName":"m.js","extra":{"nested":true}}}}`
