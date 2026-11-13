package outline

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// The synthetic secrets of the decode fixtures. Each one is a value the redaction rules forbid, so a projection that keeps
// it fails the test. None of them is a real credential.
const (
	secretKeyValue   = "ol_api_SYNTHETICKEYVALUE"
	secretKeySecret  = "ol_api_SYNTHETICKEYSECRET"
	secretWebhook    = "ol_whs_SYNTHETICWEBHOOK"
	secretPath       = "SYNTHETICPATHTOKEN"
	secretClient     = "SYNTHETICCLIENTSECRET"
	secretRegToken   = "SYNTHETICREGTOKEN"
	secretEmail      = "synthetic.person@example.com"
	secretIPAddress  = "203.0.113.77"
	secretStorageURL = "https://storage.example.com/SYNTHETICSTORAGE.png"
)

// flatText returns every text value of a Record, nested records and lists included, joined, so a test can look for a
// forbidden value anywhere in the projection.
func flatText(rec domain.Record) string {
	var b strings.Builder
	var walk func(r domain.Record)
	walk = func(r domain.Record) {
		for _, item := range r.Items() {
			walk(item)
		}
		for _, f := range r.Fields() {
			switch v := f.Value.(type) {
			case string:
				b.WriteString(v)
				b.WriteString(" ")
			case domain.Record:
				walk(v)
			case []any:
				for _, x := range v {
					if s, ok := x.(string); ok {
						b.WriteString(s + " ")
					}
				}
			}
		}
	}
	walk(rec)
	return b.String()
}

// decodeReply decodes a JSON reply, and runs the reply projection of one operation on it.
func decodeReply(t *testing.T, op domain.OperationID, raw string) (domain.Record, error) {
	t.Helper()
	var root any
	if err := json.Unmarshal([]byte(raw), &root); err != nil {
		t.Fatalf("bad fixture: %v", err)
	}
	return endpoints[op].reply(root)
}

// requireNoSecret fails when the projection holds one of the forbidden values.
func requireNoSecret(t *testing.T, rec domain.Record, forbidden ...string) {
	t.Helper()
	text := flatText(rec)
	for _, s := range forbidden {
		if strings.Contains(text, s) {
			t.Fatalf("the projection holds %q, which the redaction rules forbid", s)
		}
	}
}

// TestApiKeyListNeverKeepsTheKeyValueOrSecret checks the redaction of apiKeys.list: the key value, the old secret, the
// hash, and the user object never reach the Record. The last four characters are kept, as the field reference allows.
func TestApiKeyListNeverKeepsTheKeyValueOrSecret(t *testing.T) {
	raw := `{"data":[{"id":"` + uuidOther + `","userId":"` + uuidUser + `","name":"ci","scope":["documents.read"],` +
		`"value":"` + secretKeyValue + `","secret":"` + secretKeySecret + `","hash":"abc123",` +
		`"last4":"WXYZ","user":{"email":"` + secretEmail + `"},"createdAt":"2026-01-01T00:00:00Z",` +
		`"updatedAt":"2026-01-02T00:00:00Z","expiresAt":null,"lastActiveAt":null}]}`
	rec, err := decodeReply(t, domain.OpAPIKeysList, raw)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(rec.Items()) != 1 {
		t.Fatalf("items = %d, want 1", len(rec.Items()))
	}
	for _, name := range []string{"value", "secret", "hash", "user"} {
		if _, ok := rec.Items()[0].Get(name); ok {
			t.Fatalf("the projection keeps %q", name)
		}
	}
	requireNoSecret(t, rec, secretKeyValue, secretKeySecret, secretEmail)
}

// TestWebhookListKeepsTheHostOnly checks that a webhook keeps the host of its URL and nothing of the path or the query,
// and that the signing secret never reaches the Record.
func TestWebhookListKeepsTheHostOnly(t *testing.T) {
	raw := `{"data":[{"id":"` + uuidOther + `","name":"ops hook",` +
		`"url":"https://hooks.example.com/services/` + secretPath + `?token=` + secretPath + `",` +
		`"secret":"` + secretWebhook + `","events":["documents.update"],"enabled":true,` +
		`"createdById":"` + uuidUser + `","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z",` +
		`"createdBy":{"email":"` + secretEmail + `"}}]}`
	rec, err := decodeReply(t, domain.OpWebhookSubscriptionsList, raw)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	host, _ := rec.Items()[0].Get("urlHost")
	if host != "hooks.example.com" {
		t.Fatalf("urlHost = %v, want hooks.example.com", host)
	}
	for _, name := range []string{"url", "secret", "createdBy"} {
		if _, ok := rec.Items()[0].Get(name); ok {
			t.Fatalf("the projection keeps %q", name)
		}
	}
	requireNoSecret(t, rec, secretPath, secretWebhook, secretEmail, "/services", "token=")
}

// TestWebhookCreatorIDComesFromTheCreatorObject checks that a webhook whose reply holds the creator only as the createdBy
// object keeps that object's ID as createdById, and that the secret and the object's other fields never reach the Record.
func TestWebhookCreatorIDComesFromTheCreatorObject(t *testing.T) {
	raw := `{"data":[{"id":"` + uuidOther + `","name":"ops hook","url":"https://hooks.example.com/services/x",` +
		`"secret":"` + secretWebhook + `","events":[],"enabled":true,` +
		`"createdBy":{"id":"` + uuidUser + `","email":"` + secretEmail + `"},` +
		`"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z"}]}`
	rec, err := decodeReply(t, domain.OpWebhookSubscriptionsList, raw)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got, _ := rec.Items()[0].Get("createdById"); got != uuidUser {
		t.Fatalf("createdById = %v, want %s", got, uuidUser)
	}
	requireNoSecret(t, rec, secretWebhook, secretEmail)
}

// TestWebhookWithAnUnreadableURLHasNoHost checks that a URL that does not parse gives an empty host, and no error, so the
// listing still shows the other fields.
func TestWebhookWithAnUnreadableURLHasNoHost(t *testing.T) {
	raw := `{"data":[{"id":"` + uuidOther + `","name":"bad","url":"ht tp://[::1","events":[],"enabled":false}]}`
	rec, err := decodeReply(t, domain.OpWebhookSubscriptionsList, raw)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	host, _ := rec.Items()[0].Get("urlHost")
	if host != "" {
		t.Fatalf("urlHost = %v, want empty", host)
	}
}

// TestIntegrationKeepsTheHostOfEachURLOnly checks the redaction of integrations: a chat or embed URL keeps only its host,
// and an analytics instance URL keeps only its host; the paths, the tokens, and the installation objects are dropped.
func TestIntegrationKeepsTheHostOfEachURLOnly(t *testing.T) {
	raw := `{"data":{"id":"` + uuidOther + `","type":"embed","service":"diagrams","events":[],` +
		`"collectionId":"` + uuidCollection + `","userId":"` + uuidUser + `",` +
		`"authenticationId":"auth-1","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z",` +
		`"settings":{"url":"https://grafana.example.com/d/` + secretPath + `","github":{"installation":{"id":1}},` +
		`"gitlab":{"x":1}}}}`
	rec, err := decodeReply(t, domain.OpIntegrationsGet, raw)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	settings := rec.Rec("settings")
	if got, _ := settings.Get("urlHost"); got != "grafana.example.com" {
		t.Fatalf("settings.urlHost = %v, want grafana.example.com", got)
	}
	if _, ok := settings.Get("url"); ok {
		t.Fatal("the projection keeps the full settings.url")
	}
	if _, ok := rec.Get("authenticationId"); ok {
		t.Fatal("the projection keeps the authenticationId")
	}
	requireNoSecret(t, rec, secretPath, "/d/")
}

// TestAnalyticsIntegrationKeepsTheInstanceHostOnly checks that an analytics instance URL is cut to its host.
func TestAnalyticsIntegrationKeepsTheInstanceHostOnly(t *testing.T) {
	raw := `{"data":[{"id":"` + uuidOther + `","type":"analytics","service":"matomo","events":[],` +
		`"settings":{"measurementId":"M-1","instanceUrl":"https://matomo.example.com/` + secretPath + `","scriptName":"m.js"}}]}`
	rec, err := decodeReply(t, domain.OpIntegrationsList, raw)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	settings := rec.Items()[0].Rec("settings")
	if got, _ := settings.Get("instanceUrl"); got != "matomo.example.com" {
		t.Fatalf("settings.instanceUrl = %v, want matomo.example.com", got)
	}
	if got, _ := settings.Get("scriptName"); got != "m.js" {
		t.Fatalf("settings.scriptName = %v, want m.js", got)
	}
	requireNoSecret(t, rec, secretPath)
}

// TestOAuthClientNeverKeepsTheSecret checks that a client's secret and registration token are dropped, on the info reply
// and on the list reply, which the server also answers with the secret for a manager.
func TestOAuthClientNeverKeepsTheSecret(t *testing.T) {
	obj := `{"data":{"id":"` + uuidOther + `","name":"ci app","clientId":"client-1","clientType":"confidential",` +
		`"redirectUris":["https://app.example.com/cb"],"published":false,"createdAt":"2026-01-01T00:00:00Z",` +
		`"updatedAt":"2026-01-01T00:00:00Z","clientSecret":"` + secretClient + `",` +
		`"registrationAccessToken":"` + secretRegToken + `","avatarUrl":"https://cdn.example.com/a.png"}}`
	rec, err := decodeReply(t, domain.OpOAuthClientsGet, obj)
	if err != nil {
		t.Fatalf("info decode: %v", err)
	}
	requireNoSecret(t, rec, secretClient, secretRegToken, "cdn.example.com")
	if _, ok := rec.Get("clientSecret"); ok {
		t.Fatal("the info projection keeps clientSecret")
	}
	list := strings.Replace(obj, `{"data":{`, `{"data":[{`, 1)
	list = strings.Replace(list, `}}`, `}]}`, 1)
	recs, err := decodeReply(t, domain.OpOAuthClientsList, list)
	if err != nil {
		t.Fatalf("list decode: %v", err)
	}
	requireNoSecret(t, recs, secretClient, secretRegToken)
}

// TestAuthProviderKeepsOnlyTheGroupSyncSettings checks that a sign-in provider keeps the three group-sync settings and
// no other setting key.
func TestAuthProviderKeepsOnlyTheGroupSyncSettings(t *testing.T) {
	raw := `{"data":{"id":"` + uuidOther + `","name":"Google","providerId":"google","isEnabled":true,` +
		`"isConnected":true,"createdAt":"2026-01-01T00:00:00Z","settings":{"groupSyncEnabled":true,` +
		`"groupClaim":"groups","groupSyncScopes":["read"],"clientSecret":"` + secretClient + `"}}}`
	rec, err := decodeReply(t, domain.OpAuthenticationProvidersGet, raw)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	settings := rec.Rec("settings")
	if n := len(settings.Fields()); n != 3 {
		t.Fatalf("settings holds %d fields, want 3", n)
	}
	requireNoSecret(t, rec, secretClient)
}

// TestSuggestionsFlattenTheThreeListsIntoKinds checks the mention suggestions: documents, users, and groups become one
// list, each item with its kind, and the e-mail address of a user is never kept.
func TestSuggestionsFlattenTheThreeListsIntoKinds(t *testing.T) {
	raw := `{"data":{"documents":[{"id":"doc-1","title":"Onboarding"}],` +
		`"users":[{"id":"` + uuidUser + `","name":"Ann","email":"` + secretEmail + `","role":"member"}],` +
		`"groups":[{"id":"` + uuidGroup + `","name":"Team"}]}}`
	rec, err := decodeReply(t, domain.OpSuggestionsMention, raw)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	kinds := []string{}
	for _, item := range rec.Items() {
		k, _ := item.Get("kind")
		kinds = append(kinds, k.(string))
	}
	if strings.Join(kinds, ",") != "document,user,group" {
		t.Fatalf("kinds = %v, want document, user, group", kinds)
	}
	if name, _ := rec.Items()[0].Get("name"); name != "Onboarding" {
		t.Fatalf("the document suggestion is named %v, want its title", name)
	}
	requireNoSecret(t, rec, secretEmail)
}

// TestRelationshipGetReadsTheRelationshipOnly checks that relationships.get reads the relationship object and does not
// project the documents beside it.
func TestRelationshipGetReadsTheRelationshipOnly(t *testing.T) {
	raw := `{"data":{"relationship":{"id":"` + uuidRevision + `","type":"similar","documentId":"doc-1",` +
		`"reverseDocumentId":"doc-2","userId":"` + uuidUser + `","createdAt":"2026-01-01T00:00:00Z",` +
		`"updatedAt":"2026-01-01T00:00:00Z"},"documents":[{"id":"doc-1","text":"` + secretPath + `"}]}}`
	rec, err := decodeReply(t, domain.OpRelationshipsGet, raw)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got, _ := rec.Get("type"); got != "similar" {
		t.Fatalf("type = %v, want similar", got)
	}
	if _, ok := rec.Get("documents"); ok {
		t.Fatal("the projection keeps the documents")
	}
	requireNoSecret(t, rec, secretPath)
}

// TestEventProjectionDropsTheActorAddress checks that an event never keeps the IP address of its actor or its change data.
func TestEventProjectionDropsTheActorAddress(t *testing.T) {
	raw := `{"data":[{"id":"` + uuidOther + `","name":"documents.update","modelId":"` + uuidDocument + `",` +
		`"userId":"` + uuidUser + `","actorId":"` + uuidUser + `","documentId":"doc-1",` +
		`"authType":"api","createdAt":"2026-01-01T00:00:00Z","actorIpAddress":"` + secretIPAddress + `",` +
		`"changes":{"text":"` + secretPath + `"},"data":{"x":1},"actor":{"email":"` + secretEmail + `"}}]}`
	rec, err := decodeReply(t, domain.OpEventsList, raw)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	requireNoSecret(t, rec, secretIPAddress, secretPath, secretEmail)
	if _, ok := rec.Items()[0].Get("actorIpAddress"); ok {
		t.Fatal("the projection keeps actorIpAddress")
	}
}

// TestEmojiAndAttachmentDropTheStorageAddress checks that neither a custom emoji nor an attachment keeps the storage
// address of its file.
func TestEmojiAndAttachmentDropTheStorageAddress(t *testing.T) {
	emoji := `{"data":[{"id":"` + uuidOther + `","name":"party","url":"` + secretStorageURL + `",` +
		`"teamId":"t-1","createdById":"` + uuidUser + `","createdAt":"2026-01-01T00:00:00Z",` +
		`"updatedAt":"2026-01-01T00:00:00Z"}]}`
	rec, err := decodeReply(t, domain.OpEmojisList, emoji)
	if err != nil {
		t.Fatalf("emoji decode: %v", err)
	}
	requireNoSecret(t, rec, secretStorageURL, "storage.example.com")

	att := `{"data":[{"id":"` + uuidOther + `","name":"plan.pdf","contentType":"application/pdf","size":42,` +
		`"documentId":"` + uuidDocument + `","userId":"` + uuidUser + `","url":"` + secretStorageURL + `"}]}`
	arec, err := decodeReply(t, domain.OpAttachmentsList, att)
	if err != nil {
		t.Fatalf("attachment decode: %v", err)
	}
	requireNoSecret(t, arec, secretStorageURL, "storage.example.com")
}

// TestNotificationsDropTheNestedObjectsAndTheUnseenCount checks that a notification keeps its flat fields only, and the
// unseen count of the reply is not a field of the list.
func TestNotificationsDropTheNestedObjectsAndTheUnseenCount(t *testing.T) {
	raw := `{"data":{"notifications":[{"id":"` + uuidOther + `","event":"documents.update",` +
		`"createdAt":"2026-01-01T00:00:00Z","viewedAt":null,"archivedAt":null,"actorId":"` + uuidUser + `",` +
		`"documentId":"doc-1","collectionId":null,"commentId":null,"groupId":null,"revisionId":null,` +
		`"accessRequestId":null,"accessRequestStatus":null,"data":{"x":"` + secretPath + `"},` +
		`"actor":{"email":"` + secretEmail + `"},"document":{"text":"` + secretPath + `"}}],"unseen":3}}`
	rec, err := decodeReply(t, domain.OpNotificationsList, raw)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(rec.Items()) != 1 {
		t.Fatalf("items = %d, want 1", len(rec.Items()))
	}
	for _, name := range []string{"data", "actor", "document", "unseen"} {
		if _, ok := rec.Items()[0].Get(name); ok {
			t.Fatalf("the projection keeps %q", name)
		}
	}
	requireNoSecret(t, rec, secretPath, secretEmail)
}

// TestViewKeepsTheUserIDAndNameOnly checks that a view's user keeps its ID and name and no e-mail address.
func TestViewKeepsTheUserIDAndNameOnly(t *testing.T) {
	raw := `{"data":[{"documentId":"` + uuidDocument + `","userId":"` + uuidUser + `",` +
		`"user":{"id":"` + uuidUser + `","name":"Ann","email":"` + secretEmail + `"},"count":4,` +
		`"firstViewedAt":"2026-01-01T00:00:00Z","lastViewedAt":"2026-01-02T00:00:00Z"}]}`
	rec, err := decodeReply(t, domain.OpViewsList, raw)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	user := rec.Items()[0].Rec("user")
	if n := len(user.Fields()); n != 2 {
		t.Fatalf("user holds %d fields, want 2 (id and name)", n)
	}
	requireNoSecret(t, rec, secretEmail)
}

// TestInstallationKeepsTheVersionFields checks the installation reply, which holds three version fields.
func TestInstallationKeepsTheVersionFields(t *testing.T) {
	raw := `{"data":{"version":"1.12.0","latestVersion":"1.12.1","versionsBehind":1,"other":"x"}}`
	rec, err := decodeReply(t, domain.OpInstallationGet, raw)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got, _ := rec.Get("version"); got != "1.12.0" {
		t.Fatalf("version = %v, want 1.12.0", got)
	}
	if n := len(rec.Fields()); n != 3 {
		t.Fatalf("installation holds %d fields, want 3", n)
	}
}

// TestRelationshipWithoutItsObjectIsUnexpected checks that a relationship reply with no relationship object is refused.
func TestRelationshipWithoutItsObjectIsUnexpected(t *testing.T) {
	_, err := decodeReply(t, domain.OpRelationshipsGet, `{"data":{"documents":[]}}`)
	if !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
	}
}

// TestAdminListWithoutItsDataIsUnexpected checks that an admin list reply with no data array is refused, not read as empty.
func TestAdminListWithoutItsDataIsUnexpected(t *testing.T) {
	_, err := decodeReply(t, domain.OpAPIKeysList, `{"ok":true}`)
	if !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
	}
}

// TestUnknownFieldsAreIgnoredByTheNewProjections checks the decode tolerance of the part 2b projections: an extra field
// is ignored, and a null optional field is the zero value.
func TestUnknownFieldsAreIgnoredByTheNewProjections(t *testing.T) {
	raw := `{"data":{"groupMemberships":[{"id":"` + uuidOther + `","groupId":"` + uuidGroup + `","documentId":null,` +
		`"collectionId":null,"permission":"member","sourceId":null,"futureField":true}]}}`
	rec, err := decodeReply(t, domain.OpGroupMembershipsList, raw)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := rec.Items()[0].Get("futureField"); ok {
		t.Fatal("an unknown field reached the projection")
	}
	if got, _ := rec.Items()[0].Get("documentId"); got != nil {
		t.Fatalf("a null documentId = %v, want nil", got)
	}
}
