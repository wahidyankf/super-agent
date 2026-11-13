package outline

import (
	"encoding/json"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"testing"
)

// projectionOf runs the reply projection of one operation on a JSON reply.
func projectionOf(t *testing.T, op domain.OperationID, body []byte) domain.Record {
	t.Helper()
	ep, ok := endpointOf(op)
	if !ok {
		t.Fatalf("no endpoint for %s", op)
	}
	var root any
	if err := json.Unmarshal(body, &root); err != nil {
		t.Fatalf("the reply is not JSON: %v", err)
	}
	rec, err := ep.reply(root)
	if err != nil {
		t.Fatalf("the projection failed: %v", err)
	}
	return rec
}

// objectsOf returns the objects a projection holds: the record itself, or each item of a list.
func objectsOf(rec domain.Record) []domain.Record {
	if rec.IsList() {
		return rec.Items()
	}
	return []domain.Record{rec}
}

// fieldNames returns the names of the fields of an object, in order.
func fieldNames(rec domain.Record) []string {
	var names []string
	for _, f := range rec.Fields() {
		names = append(names, f.Name)
	}
	return names
}

// redaction is one value the redaction rules forbid (projection code table of the field reference). Its key is set in
// every object of a hostile reply, so the test does not depend on where a fixture happens to hold the field. Ops limits
// the check to the operations whose reply carries that field; nil checks every read operation.
type redaction struct {
	name      string
	key       string
	value     string
	forbidden string
	ops       []domain.OperationID
}

// redactions lists the fields the field reference forbids, with the sentinel each one carries. A sentinel is invented and
// is no credential, address, or person. A url-like sentinel forbids its secret path token, and its host may show.
var redactions = []redaction{
	{name: "user e-mail address", key: "email", value: "synthetic.person@example.com", forbidden: "synthetic.person@example.com"},
	{name: "user avatar address", key: "avatarUrl", value: "https://storage.example.test/SYNTHETICAVATAR.png", forbidden: "SYNTHETICAVATAR"},
	{name: "event actor address", key: "actorIpAddress", value: "203.0.113.77", forbidden: "203.0.113.77"},
	{name: "OAuth client secret", key: "clientSecret", value: "SYNTHETICCLIENTSECRET", forbidden: "SYNTHETICCLIENTSECRET"},
	{name: "OAuth registration token", key: "registrationAccessToken", value: "SYNTHETICREGTOKEN", forbidden: "SYNTHETICREGTOKEN"},
	{name: "webhook signing secret", key: "secret", value: "ol_whs_SYNTHETICWEBHOOK", forbidden: "ol_whs_SYNTHETICWEBHOOK"},
	{name: "API key value", key: "value", value: "ol_api_SYNTHETICKEYVALUE", forbidden: "ol_api_SYNTHETICKEYVALUE"},
	{name: "collaboration token", key: "collaborationToken", value: "SYNTHETICCOLLABTOKEN", forbidden: "SYNTHETICCOLLABTOKEN"},
	{name: "group identity-provider id", key: "externalId", value: "SYNTHETICEXTERNALID", forbidden: "SYNTHETICEXTERNALID"},
	{name: "revision collaborators", key: "collaborators", value: "SYNTHETICCOLLABORATOR", forbidden: "SYNTHETICCOLLABORATOR"},
	{name: "API key hash", key: "hash", value: "SYNTHETICHASH", forbidden: "SYNTHETICHASH"},
	{name: "share icon address", key: "iconUrl", value: "https://storage.example.test/SYNTHETICSHAREICON", forbidden: "SYNTHETICSHAREICON"},
	{name: "share document address", key: "documentUrl", value: "https://docs.example.test/SYNTHETICDOCURL", forbidden: "SYNTHETICDOCURL"},
	{name: "share source path", key: "sourcePath", value: "/SYNTHETICSOURCEPATH", forbidden: "SYNTHETICSOURCEPATH"},
	{name: "event change set", key: "changes", value: "SYNTHETICCHANGE", forbidden: "SYNTHETICCHANGE"},
	{
		name: "share link", key: "url", value: "https://docs.example.test/share/SYNTHETICSHARELINK", forbidden: "SYNTHETICSHARELINK",
		ops: []domain.OperationID{domain.OpSharesGet, domain.OpSharesList},
	},
	{
		name: "share link id", key: "urlId", value: "SYNTHETICSHAREURLID", forbidden: "SYNTHETICSHAREURLID",
		ops: []domain.OperationID{domain.OpSharesGet, domain.OpSharesList},
	},
	{
		name: "emoji storage address", key: "url", value: "https://storage.example.test/SYNTHETICEMOJI", forbidden: "SYNTHETICEMOJI",
		ops: []domain.OperationID{domain.OpEmojisGet, domain.OpEmojisList},
	},
	{
		name: "attachment storage address", key: "url", value: "https://storage.example.test/SYNTHETICATTACH", forbidden: "SYNTHETICATTACH",
		ops: []domain.OperationID{domain.OpAttachmentsList},
	},
	{
		name: "integration webhook address", key: "url", value: "https://hooks.example.test/SYNTHETICPATHTOKEN", forbidden: "SYNTHETICPATHTOKEN",
		ops: []domain.OperationID{domain.OpIntegrationsGet, domain.OpIntegrationsList},
	},
	{
		name: "webhook address", key: "url", value: "https://hooks.example.test/SYNTHETICWEBHOOKPATH", forbidden: "SYNTHETICWEBHOOKPATH",
		ops: []domain.OperationID{domain.OpWebhookSubscriptionsList},
	},
}
