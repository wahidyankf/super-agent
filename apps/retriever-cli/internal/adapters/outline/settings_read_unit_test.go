package outline

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// TestIntegrationsUpdateFillsTheEventsAndChannelsItRead checks the read-first rule of integrations.update in memory: the
// wiki resets the events when the request leaves them out, so the request carries the events of the integration as read,
// and the channel the caller did not set. The read is sent first, and the address the wiki held is never sent.
func TestIntegrationsUpdateFillsTheEventsAndChannelsItRead(t *testing.T) {
	gw, rec, stub := adminRecorder(t)
	stub.SetReply(endpoints[domain.OpIntegrationsGet].method, http.StatusOK,
		`{"ok":true,"status":200,"data":{"id":"`+uuidOther+`","type":"post","service":"slack",`+
			`"events":["documents.update","documents.publish"],"collectionId":null,"userId":null,`+
			`"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-02T00:00:00Z",`+
			`"settings":{"url":"https://hooks.example.test/SECRETPATH","channel":"general","channelId":"C123"}}}`)
	stub.SetReply(adminEndpoints[domain.OpIntegrationsUpdate].method, http.StatusOK,
		`{"ok":true,"status":200,"data":{"id":"`+uuidOther+`","type":"post","service":"slack",`+
			`"events":["documents.update","documents.publish"]}}`)
	if _, err := gw.Write(context.Background(), domain.OpIntegrationsUpdate, domain.NewArgs(map[string]any{
		"integrationId": uuidOther, "url": "https://new.example.test/x",
	})); err != nil {
		t.Fatalf("integrations.update: %v", err)
	}
	calls := rec.recorded()
	if len(calls) != 2 || calls[0].Path != "/api/integrations.info" || calls[1].Path != "/api/integrations.update" {
		t.Fatalf("calls = %+v, want integrations.info then integrations.update", calls)
	}
	want := bodyOf(t, `{"id":"`+uuidOther+`","settings":{"url":"https://new.example.test/x","channel":"general","channelId":"C123"},`+
		`"events":["documents.update","documents.publish"]}`)
	if !reflect.DeepEqual(calls[1].Body, want) {
		t.Fatalf("body = %v, want %v", calls[1].Body, want)
	}
}

// TestAuthProviderUpdateKeepsTheSettingsItRead checks the read-first rule of authenticationProviders.update in memory: the
// group sync scopes and the group sync setting the caller did not set are sent as the wiki holds them, and a claim the wiki
// holds as null is left out.
func TestAuthProviderUpdateKeepsTheSettingsItRead(t *testing.T) {
	gw, rec, stub := adminRecorder(t)
	stub.SetReply(endpoints[domain.OpAuthenticationProvidersGet].method, http.StatusOK,
		`{"ok":true,"status":200,"data":{"id":"`+uuidOther+`","name":"Google","providerId":"google","isEnabled":false,`+
			`"isConnected":true,"createdAt":"2026-01-01T00:00:00Z",`+
			`"settings":{"groupSyncEnabled":false,"groupClaim":null,"groupSyncScopes":["read","openid"]}}}`)
	stub.SetReply(adminEndpoints[domain.OpAuthenticationProvidersUpdate].method, http.StatusOK,
		`{"ok":true,"status":200,"data":{"id":"`+uuidOther+`","name":"Google","isEnabled":true}}`)
	if _, err := gw.Write(context.Background(), domain.OpAuthenticationProvidersUpdate, domain.NewArgs(map[string]any{
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
