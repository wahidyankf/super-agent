package outline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/tests/support/outlinestub"
)

// requestCasesC3 are the request-shape cases of slice c3: the access requests, attachments, sign-in providers, comments,
// emojis, file operations, imports, and integrations rows (ADR 007 section 3.6). Each body is written by hand from the field
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
// leaves them out, so Retriever reads the object and fills the values the caller did not set (ADR 007 section 3.4; the
// inventory, Finding 9). The read is an admin read of the same source.
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

// setPreReadReplies answers the pre-read of an operation with its synthetic reply, so the change can go ahead.
func setPreReadReplies(t *testing.T, stub *outlinestub.Stub, op domain.OperationID) {
	t.Helper()
	for _, pre := range adminPreReadsOf(op) {
		method := endpoints[pre].method
		stub.SetReply(method, http.StatusOK, string(loadSynthetic(t, method)))
	}
}

// adminRecorder returns an admin gateway whose requests are recorded, with the stub that answers them. A test scripts the stub's
// replies, such as the pre-read of an update.
func adminRecorder(t *testing.T) (*Gateway, *wireRecorder, *outlinestub.Stub) {
	t.Helper()
	clock := newFakeClock()
	stub := outlinestub.New(outlinestub.WithClock(clock.Now))
	rec := &wireRecorder{next: stub.RoundTripper()}
	client := &http.Client{Transport: rec}
	gw, err := NewWithBase(testSourceAt(t, testHost, domain.AccessAdmin), testBase, client, &staticCreds{},
		WithMinDelay(0), WithClock(clock.Now), WithSleep(clock.Sleep))
	if err != nil {
		t.Fatalf("NewWithBase: %v", err)
	}
	return gw, rec, stub
}

// recordingAdminCall returns the recorded admin gateway of one operation, with the synthetic reply of its change and the
// synthetic pre-read of the operation answered, so the call goes ahead.
func recordingAdminCall(t *testing.T, op domain.OperationID) (*Gateway, *wireRecorder) {
	t.Helper()
	gw, rec, stub := adminRecorder(t)
	method := adminEndpoints[op].method
	stub.SetReply(method, http.StatusOK, string(loadSynthetic(t, method)))
	setPreReadReplies(t, stub, op)
	return gw, rec
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

// TestSliceC3EveryRequestIsSentOnceWithItsExactBody checks, for each case of slice c3, the method path and the exact body of
// the change, and that the only other request is the pre-read of an update (TestEveryAdminSendsItsExactRequest covers the
// whole of stage c; this names the slice).
func TestSliceC3EveryRequestIsSentOnceWithItsExactBody(t *testing.T) {
	for _, tc := range requestCasesC3 {
		t.Run(tc.name, func(t *testing.T) {
			gw, rec := recordingAdminCall(t, tc.op)
			if _, err := gw.Administer(context.Background(), tc.op, domain.NewArgs(tc.args)); err != nil {
				t.Fatalf("Administer = %v", err)
			}
			calls := rec.recorded()
			if want := 1 + len(adminPreReadsOf(tc.op)); len(calls) != want {
				t.Fatalf("requests = %d, want %d", len(calls), want)
			}
			change := calls[len(calls)-1]
			if change.Path != tc.path {
				t.Fatalf("path = %q, want %q", change.Path, tc.path)
			}
			if !reflect.DeepEqual(change.Body, bodyOf(t, tc.body)) {
				t.Fatalf("body = %v, want %s", change.Body, tc.body)
			}
		})
	}
}

// TestSliceC3EveryRowIsRefusedAtAWriteSourceBeforeAnyRequest checks layer 3 of ADR 007 section 3.2 for each row of the slice: a
// source below admin never sends the change or the pre-read.
func TestSliceC3EveryRowIsRefusedAtAWriteSourceBeforeAnyRequest(t *testing.T) {
	for _, tc := range requestCasesC3 {
		t.Run(tc.name, func(t *testing.T) {
			gw, rec := recordingGatewayAt(t, domain.AccessWrite)
			if _, err := gw.Administer(context.Background(), tc.op, domain.NewArgs(tc.args)); err == nil {
				t.Fatal("an admin call on a write source succeeded")
			}
			if n := len(rec.recorded()); n != 0 {
				t.Fatalf("requests = %d, want 0", n)
			}
		})
	}
}

// TestSliceC3RepliesAreReadLikeTheirRows checks the field each row of the slice returns from its reply: an acknowledgement
// holds ok, an object reply holds its own field, and the sync start holds started.
func TestSliceC3RepliesAreReadLikeTheirRows(t *testing.T) {
	cases := []struct {
		op     domain.OperationID
		field  string
		expect string
	}{
		{domain.OpAccessRequestsApprove, "status", "approved"},
		{domain.OpAccessRequestsDismiss, "status", "dismissed"},
		{domain.OpAttachmentsDelete, "ok", "true"},
		{domain.OpAuthenticationProvidersDelete, "ok", "true"},
		{domain.OpAuthenticationProvidersStartGroupSync, "started", "true"},
		{domain.OpAuthenticationProvidersUpdate, "name", "Google"},
		{domain.OpCommentsDelete, "ok", "true"},
		{domain.OpEmojisDelete, "ok", "true"},
		{domain.OpFileOperationsDelete, "ok", "true"},
		{domain.OpImportsCancel, "state", "canceled"},
		{domain.OpImportsDelete, "ok", "true"},
		{domain.OpIntegrationsCreate, "service", "matomo"},
		{domain.OpIntegrationsDelete, "ok", "true"},
		{domain.OpIntegrationsUpdate, "service", "matomo"},
	}
	for _, tc := range cases {
		t.Run(string(tc.op), func(t *testing.T) {
			r := adminRig(t)
			method := adminEndpoints[tc.op].method
			r.stub.SetReply(method, http.StatusOK, string(loadSynthetic(t, method)))
			setPreReadReplies(t, r.stub, tc.op)
			out, err := r.gw.Administer(context.Background(), tc.op, domain.NewArgs(adminArgsFor(tc.op)))
			if err != nil {
				t.Fatalf("Administer = %v", err)
			}
			if v, _ := out.Get(tc.field); fmt.Sprint(v) != tc.expect {
				t.Fatalf("field %s = %v, want %q", tc.field, v, tc.expect)
			}
		})
	}
}

// TestSliceC3ProjectionsDropTheAddressAndTheRedirect checks the redaction rules of the slice: an integration keeps the host of
// its address, and the redirect address of a sign-in sync start is never a field of the result.
func TestSliceC3ProjectionsDropTheAddressAndTheRedirect(t *testing.T) {
	r := adminRig(t)
	r.stub.SetReply(adminEndpoints[domain.OpIntegrationsCreate].method, http.StatusOK,
		string(loadSynthetic(t, adminEndpoints[domain.OpIntegrationsCreate].method)))
	out, err := r.gw.Administer(context.Background(), domain.OpIntegrationsCreate, domain.NewArgs(map[string]any{
		"url": "https://matomo.example.com/SYNTHETICPATHTOKEN", "type": "analytics", "service": "matomo",
	}))
	if err != nil {
		t.Fatalf("integrations.create: %v", err)
	}
	if strings.Contains(fmt.Sprint(out), "SYNTHETICPATHTOKEN") {
		t.Fatalf("the created integration shows the path of its address: %+v", out)
	}
	if got := out.Rec("settings").Str("urlHost"); got != "matomo.example.com" {
		t.Fatalf("settings.urlHost = %q, want the host only", got)
	}

	r = adminRig(t)
	r.stub.SetReply(adminEndpoints[domain.OpAuthenticationProvidersStartGroupSync].method, http.StatusOK,
		string(loadSynthetic(t, adminEndpoints[domain.OpAuthenticationProvidersStartGroupSync].method)))
	started, err := r.gw.Administer(context.Background(), domain.OpAuthenticationProvidersStartGroupSync,
		domain.NewArgs(map[string]any{"authenticationProviderId": uuidOther}))
	if err != nil {
		t.Fatalf("startGroupSync: %v", err)
	}
	if strings.Contains(fmt.Sprint(started), "SYNTHETICREDIRECT") {
		t.Fatalf("the sync start shows the redirect address: %+v", started)
	}
	if _, has := started.Get("url"); has {
		t.Fatal("the sync start holds a url field, want only started")
	}
}

// TestSliceC3StartGroupSyncRefusesAReplyWithoutItsRedirect checks that a sync start whose reply names no redirect address is
// an unexpected reply, so the call never reports a start it cannot see.
func TestSliceC3StartGroupSyncRefusesAReplyWithoutItsRedirect(t *testing.T) {
	for _, reply := range []string{`{"data":{}}`, `{"data":{"url":""}}`, `{"success":true}`} {
		r := adminRig(t)
		r.stub.SetReply(adminEndpoints[domain.OpAuthenticationProvidersStartGroupSync].method, http.StatusOK, reply)
		_, err := r.gw.Administer(context.Background(), domain.OpAuthenticationProvidersStartGroupSync,
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
		_, err := r.gw.Administer(context.Background(), tc.op, domain.NewArgs(adminArgsFor(tc.op)))
		if !errors.Is(err, domain.ErrWriteOutcomeUnknown) {
			t.Fatalf("%s with %s: error = %v, want ErrWriteOutcomeUnknown", tc.op, tc.reply, err)
		}
		if n := r.stub.Calls(method); n != 1 {
			t.Fatalf("%s: calls = %d, want 1", tc.op, n)
		}
	}
}

// TestSliceC3IntegrationsUpdateSendsTheEventsAndChannelsItRead checks the read-first rule of integrations.update: the wiki
// resets the events when the request leaves them out, so the request carries the events of the integration as read, and
// the channel the caller did not set. The read is sent first.
func TestSliceC3IntegrationsUpdateSendsTheEventsAndChannelsItRead(t *testing.T) {
	gw, rec, stub := adminRecorder(t)
	stub.SetReply(endpoints[domain.OpIntegrationsGet].method, http.StatusOK,
		`{"ok":true,"status":200,"data":{"id":"`+uuidOther+`","type":"post","service":"slack","events":["documents.update","documents.publish"],"collectionId":null,"userId":null,"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-02T00:00:00Z","settings":{"url":"https://hooks.example.test/SECRETPATH","channel":"general","channelId":"C123"}}}`)
	stub.SetReply(adminEndpoints[domain.OpIntegrationsUpdate].method, http.StatusOK, string(loadSynthetic(t, "integrations.update")))
	if _, err := gw.Administer(context.Background(), domain.OpIntegrationsUpdate, domain.NewArgs(map[string]any{
		"integrationId": uuidOther, "url": "https://new.example.test/x",
	})); err != nil {
		t.Fatalf("integrations.update: %v", err)
	}
	calls := rec.recorded()
	if len(calls) != 2 || calls[0].Path != "/api/integrations.info" || calls[1].Path != "/api/integrations.update" {
		t.Fatalf("calls = %+v, want integrations.info then integrations.update", calls)
	}
	want := bodyOf(t, `{"id":"`+uuidOther+`","settings":{"url":"https://new.example.test/x","channel":"general","channelId":"C123"},"events":["documents.update","documents.publish"]}`)
	if !reflect.DeepEqual(calls[1].Body, want) {
		t.Fatalf("body = %v, want %v", calls[1].Body, want)
	}
	if strings.Contains(fmt.Sprint(calls[0].Body), "SECRETPATH") {
		t.Fatal("the read sent the address back")
	}
}

// TestSliceC3IntegrationsUpdateSendsNothingWhenTheReadFails checks that a change whose pre-read fails is never sent: the read
// is needed for the events, so an update without it would reset them.
func TestSliceC3IntegrationsUpdateSendsNothingWhenTheReadFails(t *testing.T) {
	gw, rec, stub := adminRecorder(t)
	stub.SetReply(endpoints[domain.OpIntegrationsGet].method, http.StatusNotFound, `{"ok":false,"status":404,"error":"not_found"}`)
	_, err := gw.Administer(context.Background(), domain.OpIntegrationsUpdate, domain.NewArgs(map[string]any{
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

// TestSliceC3AuthProviderUpdateKeepsTheSettingsItRead checks the read-first rule of authenticationProviders.update: the group
// sync scopes and the group sync setting the caller did not set are sent as the wiki holds them, and a claim the wiki holds as
// null is left out.
func TestSliceC3AuthProviderUpdateKeepsTheSettingsItRead(t *testing.T) {
	gw, rec, stub := adminRecorder(t)
	stub.SetReply(endpoints[domain.OpAuthenticationProvidersGet].method, http.StatusOK,
		`{"ok":true,"status":200,"data":{"id":"`+uuidOther+`","name":"Google","providerId":"google","isEnabled":false,"isConnected":true,"createdAt":"2026-01-01T00:00:00Z","settings":{"groupSyncEnabled":false,"groupClaim":null,"groupSyncScopes":["read","openid"]}}}`)
	stub.SetReply(adminEndpoints[domain.OpAuthenticationProvidersUpdate].method, http.StatusOK, string(loadSynthetic(t, "authenticationProviders.update")))
	if _, err := gw.Administer(context.Background(), domain.OpAuthenticationProvidersUpdate, domain.NewArgs(map[string]any{
		"authenticationProviderId": uuidOther, "isEnabled": true,
	})); err != nil {
		t.Fatalf("authenticationProviders.update: %v", err)
	}
	calls := rec.recorded()
	if len(calls) != 2 {
		t.Fatalf("requests = %d, want the read and the update", len(calls))
	}
	want := bodyOf(t, `{"id":"`+uuidOther+`","isEnabled":true,"settings":{"groupSyncEnabled":false,"groupSyncScopes":["read","openid"]}}`)
	if !reflect.DeepEqual(calls[1].Body, want) {
		t.Fatalf("body = %v, want %v", calls[1].Body, want)
	}
}

// integrationInfoFixture is the reply of integrations.info for the integration of the update tests. Its settings hold keys that
// the update must resend unchanged (measurementId, instanceUrl, scriptName, and a nested object), the channel and channel ID
// the update keeps, and the address the update replaces. Its events are kept too. The values are synthetic (ADR 007 section 3.4).
const integrationInfoFixture = `{"ok":true,"status":200,"data":{"id":"` + uuidOther + `","type":"analytics","service":"matomo",` +
	`"events":["documents.create","documents.update"],"collectionId":null,"userId":"33333333-3333-4333-8333-333333333333",` +
	`"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-02T00:00:00Z",` +
	`"settings":{"url":"https://old.example.test/OLDPATHTOKEN","channel":"general","channelId":"C-1",` +
	`"measurementId":"M-7","instanceUrl":"https://matomo.example.test/KEEPPATH","scriptName":"m.js","extra":{"nested":true}}}}`

// TestIntegrationsUpdateResendsEverySettingsKeyOfTheWiki checks the settings rule of integrations.update (m3 of the review of
// stage c): the wiki replaces the settings of an integration, so the request carries every key of the settings read before
// the change, with only the url replaced by the new address, and the events read with them. A channel the caller sets
// replaces the channel the wiki holds. The old address is never sent.
func TestIntegrationsUpdateResendsEverySettingsKeyOfTheWiki(t *testing.T) {
	cases := []struct {
		name string
		raw  map[string]any
		want string
	}{
		{"no channel given", map[string]any{"integrationId": uuidOther, "url": "https://new.example.test/x"},
			`{"id":"` + uuidOther + `","settings":{"url":"https://new.example.test/x","channel":"general","channelId":"C-1",` +
				`"measurementId":"M-7","instanceUrl":"https://matomo.example.test/KEEPPATH","scriptName":"m.js","extra":{"nested":true}},` +
				`"events":["documents.create","documents.update"]}`},
		{"a channel given", map[string]any{"integrationId": uuidOther, "url": "https://new.example.test/x", "channel": "ops"},
			`{"id":"` + uuidOther + `","settings":{"url":"https://new.example.test/x","channel":"ops","channelId":"C-1",` +
				`"measurementId":"M-7","instanceUrl":"https://matomo.example.test/KEEPPATH","scriptName":"m.js","extra":{"nested":true}},` +
				`"events":["documents.create","documents.update"]}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gw, rec, stub := adminRecorder(t)
			stub.SetReply(endpoints[domain.OpIntegrationsGet].method, http.StatusOK, integrationInfoFixture)
			stub.SetReply(adminEndpoints[domain.OpIntegrationsUpdate].method, http.StatusOK, string(loadSynthetic(t, "integrations.update")))
			if _, err := gw.Administer(context.Background(), domain.OpIntegrationsUpdate, domain.NewArgs(tc.raw)); err != nil {
				t.Fatalf("integrations.update: %v", err)
			}
			calls := rec.recorded()
			if len(calls) != 2 {
				t.Fatalf("requests = %d, want the read and the update", len(calls))
			}
			if !reflect.DeepEqual(calls[1].Body, bodyOf(t, tc.want)) {
				t.Fatalf("body = %v, want %s", calls[1].Body, tc.want)
			}
		})
	}
}

// TestAuthProvidersUpdateFillsOnlyTheKeysTheReplyHolds checks m1 of the review of stage c: the pre-read of
// authenticationProviders.update fills a value only when its key exists in the reply. A key the reply does not hold gets no
// default, so the request never sends groupSyncEnabled false or an empty groupSyncScopes the wiki did not report.
func TestAuthProvidersUpdateFillsOnlyTheKeysTheReplyHolds(t *testing.T) {
	cases := []struct {
		name  string
		reply string
		want  string
	}{
		{"a settings object with only the claim",
			`{"ok":true,"status":200,"data":{"id":"` + uuidOther + `","name":"Google","providerId":"google","isEnabled":true,"settings":{"groupClaim":"groups"}}}`,
			`{"id":"` + uuidOther + `","isEnabled":false,"settings":{"groupClaim":"groups"}}`},
		{"no settings object at all",
			`{"ok":true,"status":200,"data":{"id":"` + uuidOther + `","name":"Google","providerId":"google","isEnabled":true}}`,
			`{"id":"` + uuidOther + `","isEnabled":false}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gw, rec, stub := adminRecorder(t)
			stub.SetReply(endpoints[domain.OpAuthenticationProvidersGet].method, http.StatusOK, tc.reply)
			stub.SetReply(adminEndpoints[domain.OpAuthenticationProvidersUpdate].method, http.StatusOK, string(loadSynthetic(t, "authenticationProviders.update")))
			if _, err := gw.Administer(context.Background(), domain.OpAuthenticationProvidersUpdate, domain.NewArgs(map[string]any{
				"authenticationProviderId": uuidOther, "isEnabled": false,
			})); err != nil {
				t.Fatalf("authenticationProviders.update: %v", err)
			}
			calls := rec.recorded()
			if len(calls) != 2 {
				t.Fatalf("requests = %d, want the read and the update", len(calls))
			}
			if !reflect.DeepEqual(calls[1].Body, bodyOf(t, tc.want)) {
				t.Fatalf("body = %v, want %s", calls[1].Body, tc.want)
			}
		})
	}
}
