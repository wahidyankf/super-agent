package outline

import (
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// specTestID is a well-formed object ID for the projections that require one.
const specTestID = "3f1c2b4a-5d6e-4f70-8a9b-0c1d2e3f4a5b"

// TestRecordSpecsKeepTheIDOfTheObjectTheyProject checks that each projection that starts from the object ID returns a record
// that keeps that ID, from an in-memory object.
func TestRecordSpecsKeepTheIDOfTheObjectTheyProject(t *testing.T) {
	obj := map[string]any{"id": specTestID}
	specs := map[string]func(map[string]any) (domain.Record, error){
		"revision":       revisionSpec,
		"revision row":   revisionRowSpec,
		"star":           starSpec,
		"subscription":   subscriptionSpec,
		"import":         importSpec,
		"access request": accessRequestSpec,
		"reaction":       reactionSpec,
		"search":         searchSpec,
	}
	for name, spec := range specs {
		rec, err := spec(obj)
		if err != nil {
			t.Fatalf("%s spec: %v", name, err)
		}
		if rec.Str("id") != specTestID {
			t.Fatalf("%s spec id = %q, want %q", name, rec.Str("id"), specTestID)
		}
	}
}

// TestProjectStartedReportsTheSyncOnceTheReplyNamesTheRedirect checks that the group sync start is reported as started only
// when the reply names the redirect address, and that a reply without it is refused.
func TestProjectStartedReportsTheSyncOnceTheReplyNamesTheRedirect(t *testing.T) {
	rec, err := projectStarted(map[string]any{"data": map[string]any{"url": "https://idp.example.test/redirect"}})
	if err != nil {
		t.Fatalf("projectStarted: %v", err)
	}
	if started, _ := rec.Get("started"); started != true {
		t.Fatalf("started = %v, want true", started)
	}
	if _, err := projectStarted(map[string]any{"data": map[string]any{}}); err != domain.ErrUnexpectedResponse {
		t.Fatalf("a reply without the redirect address: err = %v, want ErrUnexpectedResponse", err)
	}
}

// TestInviteBodySendsSuppressEmailOnlyWhenTheCallerSetsIt checks that the invitation carries the suppress flag only when the
// caller set it.
func TestInviteBodySendsSuppressEmailOnlyWhenTheCallerSetsIt(t *testing.T) {
	plain := inviteBody(domain.NewArgs(map[string]any{"email": "user@example.test", "name": "Example", "role": "member"}))
	if _, has := plain["suppressEmail"]; has {
		t.Fatal("an invitation that sets no suppress flag must not send one")
	}
	quiet := inviteBody(domain.NewArgs(map[string]any{"email": "user@example.test", "suppressEmail": true}))
	if quiet["suppressEmail"] != true {
		t.Fatalf("suppressEmail = %v, want true", quiet["suppressEmail"])
	}
}

// TestIntegrationSettingsKeepsTheChannelsTheCallerSet checks that the settings of an integration carry the channel and the
// channel ID when the caller set them, and carry neither when the caller did not.
func TestIntegrationSettingsKeepsTheChannelsTheCallerSet(t *testing.T) {
	set := integrationSettings(domain.NewArgs(map[string]any{
		"url": "https://hooks.example.test/hook", "channel": "general", "channelId": "C-EXAMPLE1",
	}))
	if set["channel"] != "general" || set["channelId"] != "C-EXAMPLE1" {
		t.Fatalf("settings = %v, want the channel and the channel ID the caller set", set)
	}
	plain := integrationSettings(domain.NewArgs(map[string]any{"url": "https://hooks.example.test/hook"}))
	if _, has := plain["channel"]; has {
		t.Fatal("settings the caller did not set must not be sent")
	}
}

// TestCurrentIntegrationSpecKeepsTheEventsAndSettingsButNotTheAddress checks that the read before an integration update keeps
// the events and the settings of the wiki, and leaves the address out of the record.
func TestCurrentIntegrationSpecKeepsTheEventsAndSettingsButNotTheAddress(t *testing.T) {
	reply := map[string]any{"data": map[string]any{
		"events":   []any{"documents.create"},
		"settings": map[string]any{"url": "https://hooks.example.test/secret-path", "measurementId": "G-EXAMPLE1"},
	}}
	rec, err := currentIntegrationSpec(reply)
	if err != nil {
		t.Fatalf("currentIntegrationSpec: %v", err)
	}
	events, _ := rec.Get("events")
	if list, ok := events.([]any); !ok || len(list) != 1 || list[0] != "documents.create" {
		t.Fatalf("events = %v, want the event of the integration", events)
	}
	settings, _ := rec.Get("settings")
	m, ok := settings.(map[string]any)
	if !ok || m["measurementId"] != "G-EXAMPLE1" {
		t.Fatalf("settings = %v, want the measurement ID of the wiki", settings)
	}
	if _, has := m["url"]; has {
		t.Fatal("the address of the integration must not be read into the record")
	}
	if _, err := currentIntegrationSpec("not an object"); err == nil {
		t.Fatal("a reply that is not an object must be refused")
	}
}

// TestCurrentAuthProviderSpecKeepsOnlyTheGroupSettingsTheReplyHolds checks that the read before a sign-in provider update
// keeps the group sync settings the reply holds, and leaves out the ones it does not.
func TestCurrentAuthProviderSpecKeepsOnlyTheGroupSettingsTheReplyHolds(t *testing.T) {
	reply := map[string]any{"data": map[string]any{
		"settings": map[string]any{"groupSyncEnabled": true, "groupClaim": "groups"},
	}}
	rec, err := currentAuthProviderSpec(reply)
	if err != nil {
		t.Fatalf("currentAuthProviderSpec: %v", err)
	}
	if rec.Str("groupClaim") != "groups" {
		t.Fatalf("groupClaim = %q, want groups", rec.Str("groupClaim"))
	}
	if enabled, _ := rec.Get("groupSyncEnabled"); enabled != true {
		t.Fatalf("groupSyncEnabled = %v, want true", enabled)
	}
	if _, ok := rec.Get("groupSyncScopes"); ok {
		t.Fatal("a setting the reply does not hold must not be in the record")
	}
	if _, err := currentAuthProviderSpec(nil); err == nil {
		t.Fatal("a reply that is not an object must be refused")
	}
}
