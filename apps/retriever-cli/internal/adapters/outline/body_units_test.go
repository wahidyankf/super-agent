package outline

import (
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// TestInsightSpecProjectsTheDayOfAnInsight checks that the day projection keeps the date and the period of one day of a
// document's insights, from an in-memory reply.
func TestInsightSpecProjectsTheDayOfAnInsight(t *testing.T) {
	rec, err := insightSpec(map[string]any{"date": "2026-11-01", "period": "day", "viewCount": 4})
	if err != nil {
		t.Fatalf("insightSpec: %v", err)
	}
	if rec.Str("date") != "2026-11-01" || rec.Str("period") != "day" {
		t.Fatalf("insight = %+v, want the date and the period", rec)
	}
}

// TestIntegrationUpdateBodyCarriesTheEventsOnlyWhenTheCallerSetsThem checks that the update resends the events of the
// integration only when the request names them, so an unset list is never sent as an empty one.
func TestIntegrationUpdateBodyCarriesTheEventsOnlyWhenTheCallerSetsThem(t *testing.T) {
	without := integrationUpdateBody(domain.NewArgs(map[string]any{"integrationId": "int-1"}))
	if without["id"] != "int-1" {
		t.Fatalf("id = %v, want int-1", without["id"])
	}
	if _, has := without["events"]; has {
		t.Fatal("an update that names no events must not send events")
	}
	with := integrationUpdateBody(domain.NewArgs(map[string]any{"integrationId": "int-1", "events": []any{"documents.create"}}))
	if _, has := with["events"]; !has {
		t.Fatal("an update that names events must send them")
	}
}

// TestAuthProviderUpdateBodySendsOnlyTheSettingsTheCallerSet checks that the sign-in provider update sends isEnabled and the
// group sync settings the caller set, and no settings object when none is set.
func TestAuthProviderUpdateBodySendsOnlyTheSettingsTheCallerSet(t *testing.T) {
	body := authProviderUpdateBody(domain.NewArgs(map[string]any{
		"authenticationProviderId": "ap-1", "isEnabled": true, "groupClaim": "groups",
	}))
	if body["id"] != "ap-1" || body["isEnabled"] != true {
		t.Fatalf("body = %v, want the id and isEnabled", body)
	}
	settings, ok := body["settings"].(map[string]any)
	if !ok || settings["groupClaim"] != "groups" {
		t.Fatalf("settings = %v, want the group claim the caller set", body["settings"])
	}
	if _, has := settings["groupSyncEnabled"]; has {
		t.Fatal("a setting the caller did not set must not be sent")
	}
	plain := authProviderUpdateBody(domain.NewArgs(map[string]any{"authenticationProviderId": "ap-1"}))
	if _, has := plain["settings"]; has {
		t.Fatal("an update that sets no group sync value must send no settings object")
	}
}

// TestWithMissingFillsOnlyTheValuesTheCallerDidNotSet checks that a value the caller set is kept, and that a value the reply
// holds is filled only when the caller left it out and the reply does not hold it as null.
func TestWithMissingFillsOnlyTheValuesTheCallerDidNotSet(t *testing.T) {
	args := domain.NewArgs(map[string]any{"groupClaim": "mine"})
	rec := domain.NewRecord(field("groupClaim", "theirs"), field("groupSyncEnabled", true))
	values := withMissing(args, rec, "groupClaim", "groupSyncEnabled", "groupSyncScopes").Values()
	if values["groupClaim"] != "mine" {
		t.Fatalf("groupClaim = %v, want the value the caller set", values["groupClaim"])
	}
	if values["groupSyncEnabled"] != true {
		t.Fatalf("groupSyncEnabled = %v, want the value the reply holds", values["groupSyncEnabled"])
	}
	if _, has := values["groupSyncScopes"]; has {
		t.Fatal("a value the reply does not hold must not be filled")
	}
}
