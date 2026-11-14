package outlinestub

import (
	"strings"
	"testing"
)

func TestCheckReplyKeysRefusesAKeyTheRouterDoesNotSend(t *testing.T) {
	// The router sends a share's creator as a createdBy object. A flat createdById on a share is a fabricated key.
	body := []byte(`{"ok":true,"data":{"shares":[{"id":"s-1","createdById":"u-1"}]}}`)
	err := CheckReplyKeys("shares.info", body)
	if err == nil || !strings.Contains(err.Error(), `"createdById"`) {
		t.Fatalf("want a refusal naming createdById, got %v", err)
	}
}

func TestCheckReplyKeysAcceptsTheKeysOfThePresenters(t *testing.T) {
	body := []byte(`{"ok":true,"status":200,"data":{"pins":[{"id":"p-1","documentId":"d-1","index":"P"}],` +
		`"documents":[{"id":"d-1","title":"Home Page"}]},"pagination":{"offset":0,"limit":25}}`)
	if err := CheckReplyKeys("pins.list", body); err != nil {
		t.Fatalf("a router reply was refused: %v", err)
	}
}

func TestCheckReplyKeysRefusesANestedFabricatedKey(t *testing.T) {
	// The user presenter has no userId, so a userId inside a file operation's user object is refused.
	body := []byte(`{"ok":true,"data":[{"id":"f-1","user":{"id":"u-1","userId":"u-1"}}]}`)
	err := CheckReplyKeys("fileOperations.list", body)
	if err == nil || !strings.Contains(err.Error(), `"userId"`) {
		t.Fatalf("want a refusal naming the nested userId, got %v", err)
	}
}

func TestCheckReplyKeysDoesNotLookInsideAnOpaqueSettingsObject(t *testing.T) {
	body := []byte(`{"ok":true,"data":{"id":"b-1","name":"Google","settings":{"groupClaim":"groups","anything":"x"}}}`)
	if err := CheckReplyKeys("authenticationProviders.info", body); err != nil {
		t.Fatalf("the settings object is opaque, so its keys are not checked: %v", err)
	}
}

func TestWebhookListHasARouterKeyEntry(t *testing.T) {
	// The webhooks plugin mounts webhookSubscriptions.list at v1.12.0, so the table names its keys and the stub checks it.
	if _, ok := RouterKeys("webhookSubscriptions.list"); !ok {
		t.Fatal("the router key table has no entry for webhookSubscriptions.list")
	}
}

func TestCheckReplyKeysRefusesAFabricatedWebhookKey(t *testing.T) {
	// The webhook presenter sends no key named signingKey, so a stated reply that holds one fails, not the read.
	body := []byte(`{"ok":true,"data":[{"id":"w-1","signingKey":"x"}]}`)
	err := CheckReplyKeys("webhookSubscriptions.list", body)
	if err == nil || !strings.Contains(err.Error(), `"signingKey"`) {
		t.Fatalf("want a refusal naming signingKey, got %v", err)
	}
}

func TestCheckReplyKeysAcceptsTheWebhookPresenterKeys(t *testing.T) {
	// The presenter sends the signing secret on every reply, and the creator as an object when there is one. The router
	// sends both, so the table allows both; Retriever must still drop the secret (decode_2b_test.go).
	body := []byte(`{"ok":true,"status":200,"data":[{"id":"w-1","name":"ops hook","url":"https://hooks.example.com/x",` +
		`"secret":"ol_whs_x","events":["documents.update"],"enabled":true,"createdById":"u-1",` +
		`"createdBy":{"id":"u-1","name":"Fixture User"},"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z"}],` +
		`"pagination":{"offset":0,"limit":25}}`)
	if err := CheckReplyKeys("webhookSubscriptions.list", body); err != nil {
		t.Fatalf("a router reply was refused: %v", err)
	}
}

func TestCheckReplyKeysRefusesAMethodTheTableDoesNotName(t *testing.T) {
	err := CheckReplyKeys("no.such.method", []byte(`{"ok":true}`))
	if err == nil || !strings.Contains(err.Error(), "no entry") {
		t.Fatalf("want a refusal for an unknown method, got %v", err)
	}
}

func TestCheckReplyKeysRefusesAReplyThatIsNotJSON(t *testing.T) {
	if err := CheckReplyKeys("pins.list", []byte(`not json`)); err == nil {
		t.Fatal("want a refusal for a reply that is not JSON")
	}
}

func TestEveryTableMethodHasAKnownPresenter(t *testing.T) {
	for method, presenters := range methodPresenters {
		for _, p := range presenters {
			if _, ok := presenterKeys[p]; !ok {
				t.Errorf("%s names the presenter %q, which has no key list", method, p)
			}
		}
	}
}

func TestReplayFixturesUseOnlyTheRouterKeys(t *testing.T) {
	// The replay fixtures are the replies the stub gives when a scenario does not state one. Each one is named for its
	// method, so each is checked against that method's entry. The error fixtures are not replies to a method.
	for _, name := range []string{"collections.documents", "collections.list", "documents.info"} {
		body, err := fixtureFS.ReadFile("testdata/" + name + ".json")
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if err := CheckReplyKeys(name, body); err != nil {
			t.Errorf("replay fixture %s: %v", name, err)
		}
	}
}

// writeMethods are the wire methods of every built write, destructive, and admin row: the stage b writes and the stage c1
// admin rows. Each one has a router key entry and a verified envelope, so a stated or synthetic reply
// of a write is checked against the keys the router sends.
var writeMethods = []string{
	"documents.create", "documents.duplicate", "documents.move", "documents.restore", "documents.templatize",
	"documents.unpublish", "documents.update",
	"collections.create", "collections.duplicate", "collections.export", "collections.move", "collections.restore",
	"collections.update",
	"comments.create", "comments.update", "comments.resolve", "comments.unresolve", "comments.add_reaction",
	"comments.remove_reaction",
	"accessRequests.create", "revisions.update",
	"notifications.update", "notifications.update_all",
	"pins.create", "pins.delete", "pins.update",
	"searches.delete", "searches.update",
	"stars.create", "stars.delete", "stars.update",
	"subscriptions.create", "subscriptions.delete",
	"templates.create", "templates.duplicate", "templates.restore", "templates.update",
	"userMemberships.update",
	"users.notificationsSubscribe", "users.notificationsUnsubscribe",
	// The stage c1 admin rows.
	"documents.add_group", "documents.add_user", "documents.archive", "documents.delete", "documents.empty_trash",
	"documents.remove_group", "documents.remove_user",
	"collections.add_group", "collections.add_user", "collections.archive", "collections.delete",
	"collections.export_all", "collections.remove_group", "collections.remove_user",
}

// dataOnlyWrites are the built writes whose reply is a data member with no policies: a presentation without the policies
// its handler adds (the searches, subscriptions, and notification settings writes, and the membership and group lists).
var dataOnlyWrites = []string{
	"searches.update", "subscriptions.create", "users.notificationsSubscribe", "users.notificationsUnsubscribe",
	"documents.add_user", "documents.add_group", "collections.add_user", "collections.add_group",
}

func TestEveryWriteMethodHasARouterKeyEntry(t *testing.T) {
	for _, m := range writeMethods {
		if _, ok := RouterKeys(m); !ok {
			t.Errorf("the router key table has no entry for the write %s", m)
		}
	}
}

// TestEveryBuiltWriteHasAVerifiedEnvelope checks that every built write has its reply envelope in the table. A write without
// one keeps the shared envelope, which admits ok and status that the wiki never sends for a write.
func TestEveryBuiltWriteHasAVerifiedEnvelope(t *testing.T) {
	for _, m := range writeMethods {
		if _, ok := VerifiedEnvelope(m); !ok {
			t.Errorf("the write %s has no verified envelope, so the shared envelope admits its ok and status", m)
		}
	}
}

// TestABuiltWriteRefusesTheSharedEnvelope checks that a reply with the ok and status of the shared envelope is refused for
// every built write. That is the shape a fabricated fixture of the old reading had.
func TestABuiltWriteRefusesTheSharedEnvelope(t *testing.T) {
	for _, m := range writeMethods {
		err := CheckReplyKeys(m, []byte(`{"ok":true,"status":200}`))
		if err == nil || !strings.Contains(err.Error(), `"ok"`) {
			t.Errorf("%s: want a refusal naming ok, got %v", m, err)
		}
	}
}

// TestADataOnlyWriteRefusesPolicies checks that a data-only write refuses the policies of an object reply, and accepts its
// data member alone.
func TestADataOnlyWriteRefusesPolicies(t *testing.T) {
	for _, m := range dataOnlyWrites {
		if err := CheckReplyKeys(m, []byte(`{"data":{"id":"x-1"}}`)); err != nil {
			t.Errorf("%s: the router's data-only reply was refused: %v", m, err)
		}
		err := CheckReplyKeys(m, []byte(`{"data":{"id":"x-1"},"policies":[]}`))
		if err == nil || !strings.Contains(err.Error(), `"policies"`) {
			t.Errorf("%s: want a refusal naming policies, got %v", m, err)
		}
	}
}

// TestDuplicateAnswersItsListUnderData checks the reply of documents.duplicate: the copies are the list under data.documents,
// with the policies beside it. The list key must stay allowed once the write has its own envelope.
func TestDuplicateAnswersItsListUnderData(t *testing.T) {
	body := []byte(`{"data":{"documents":[{"id":"doc-3","title":"Plan (copy)","url":"/doc/plan-copy","collectionId":"col-1",` +
		`"revision":1}]},"policies":[]}`)
	if err := CheckReplyKeys("documents.duplicate", body); err != nil {
		t.Fatalf("the router's duplicate reply was refused: %v", err)
	}
}

// TestNotificationSettingsKeysAreOpaqueButTheUserKeysAreChecked checks that the settings of a user (notificationSettings)
// are keyed by the event types of the server, so their keys are opaque, while a key the user presenter never sends is
// still refused.
func TestNotificationSettingsKeysAreOpaqueButTheUserKeysAreChecked(t *testing.T) {
	ok := []byte(`{"data":{"id":"u-1","notificationSettings":{"documents.update":true}}}`)
	if err := CheckReplyKeys("users.notificationsSubscribe", ok); err != nil {
		t.Fatalf("a settings reply was refused: %v", err)
	}
	bad := []byte(`{"data":{"id":"u-1","userId":"u-1"}}`)
	err := CheckReplyKeys("users.notificationsSubscribe", bad)
	if err == nil || !strings.Contains(err.Error(), `"userId"`) {
		t.Fatalf("want a refusal naming userId, got %v", err)
	}
}

// ackWrites are the acknowledgement writes of the stage b and stage c1 rows. The wiki answers {success: true} (and, for
// four of them, a data member), never {ok: true}, so the table names success and refuses ok and status.
var ackWrites = []string{
	"documents.delete", "documents.empty_trash", "documents.remove_group", "documents.remove_user",
	"collections.delete", "collections.remove_group", "collections.remove_user",
	"comments.add_reaction", "comments.remove_reaction",
	"pins.delete", "searches.delete", "stars.delete", "subscriptions.delete",
	"collections.move", "collections.export", "notifications.update_all", "collections.export_all",
}

// TestAnAcknowledgementAnswersSuccessAndNeverOk checks the wire reply of every acknowledgement write: {success: true}
// passes, and the {ok: true} shape the old reading expected is refused, so a fabricated reply fails the scenario that
// states it.
func TestAnAcknowledgementAnswersSuccessAndNeverOk(t *testing.T) {
	for _, m := range ackWrites {
		if err := CheckReplyKeys(m, []byte(`{"success":true}`)); err != nil {
			t.Errorf("%s: the router's acknowledgement was refused: %v", m, err)
		}
		err := CheckReplyKeys(m, []byte(`{"ok":true}`))
		if err == nil || !strings.Contains(err.Error(), `"ok"`) {
			t.Errorf("%s: want a refusal naming ok, got %v", m, err)
		}
		err = CheckReplyKeys(m, []byte(`{"success":true,"status":200}`))
		if err == nil || !strings.Contains(err.Error(), `"status"`) {
			t.Errorf("%s: want a refusal naming status, got %v", m, err)
		}
	}
}

// TestAckWritesWithAValueAnswerTheirDataMember checks the four acknowledgements that also name a value: a move names the
// index the wiki kept, an update-all names its total, and an export (one collection, or every collection) names its file
// operation, all under data.
func TestAckWritesWithAValueAnswerTheirDataMember(t *testing.T) {
	const fileOperation = `"fileOperation":{"id":"a6000000-0000-4000-8000-000000000006",` +
		`"type":"export","format":"json","name":"All","state":"creating","error":null,"size":0,"collectionId":null,` +
		`"documentId":null,"user":{"id":"u-1","name":"Ann"},"createdAt":"2026-01-01T00:00:00Z",` +
		`"updatedAt":"2026-01-01T00:00:00Z"}`
	ok := map[string]string{
		"collections.move":         `{"success":true,"data":{"index":"P"}}`,
		"notifications.update_all": `{"success":true,"data":{"total":3}}`,
		"collections.export":       `{"success":true,"data":{` + fileOperation + `}}`,
		"collections.export_all":   `{"success":true,"data":{` + fileOperation + `}}`,
	}
	for m, body := range ok {
		if err := CheckReplyKeys(m, []byte(body)); err != nil {
			t.Errorf("%s: the router's reply was refused: %v", m, err)
		}
	}
	// A move's ok shape with the index at the top level is not the router's reply.
	err := CheckReplyKeys("collections.move", []byte(`{"ok":true,"status":200,"index":"P"}`))
	if err == nil || !strings.Contains(err.Error(), `"ok"`) {
		t.Fatalf("want a refusal naming ok, got %v", err)
	}
}

// TestAVerifiedObjectWriteAnswersDataAndPoliciesNotOk checks the object replies whose shape the v1.12.0 source fixes:
// documents.archive and documents.move answer data and policies, and the add-user and add-group replies answer data.
// The ok and status of the shared envelope are refused for them.
func TestAVerifiedObjectWriteAnswersDataAndPoliciesNotOk(t *testing.T) {
	good := map[string]string{
		"documents.archive": `{"data":{"id":"doc-2","title":"Plan","url":"/doc/plan","collectionId":"col-1"},"policies":[]}`,
		"documents.move":    `{"data":{"documents":[{"id":"doc-1","title":"Plan","url":"/doc/plan","collectionId":"col-2","revision":2}],"collections":[]},"policies":[]}`,
		"documents.add_user": `{"data":{"memberships":[{"id":"a7000000-0000-4000-8000-000000000002","userId":"u-1",` +
			`"documentId":"doc-2","permission":"read_write"}],"users":[{"id":"u-1","name":"Ann"}]}}`,
		"collections.add_group": `{"data":{"groupMemberships":[{"id":"a7000000-0000-4000-8000-000000000001",` +
			`"groupId":"g-1","collectionId":"col-1","permission":"read_write"}]}}`,
	}
	for m, body := range good {
		if err := CheckReplyKeys(m, []byte(body)); err != nil {
			t.Errorf("%s: the router's reply was refused: %v", m, err)
		}
	}
	err := CheckReplyKeys("documents.archive", []byte(`{"ok":true,"status":200,"data":{"id":"doc-2","title":"Plan"}}`))
	if err == nil || !strings.Contains(err.Error(), `"ok"`) {
		t.Fatalf("want a refusal naming ok on an archive reply, got %v", err)
	}
}
