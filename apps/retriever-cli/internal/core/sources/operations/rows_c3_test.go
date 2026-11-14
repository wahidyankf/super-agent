package operations

import (
	"errors"
	"reflect"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// TestSliceC3IntegrationsCreateTakesTheURLAsItsPositionalArgument checks that the URL is the positional argument of the
// creation, so the canonical host is the first thing a person types. A plain HTTP address is refused, and so is a service the
// create does not offer, which the list of integrations offers.
func TestSliceC3IntegrationsCreateTakesTheURLAsItsPositionalArgument(t *testing.T) {
	row := rowOf(t, domain.OpIntegrationsCreate)
	if got := row.Positional(); got != "url" {
		t.Fatalf("positional = %q, want url", got)
	}
	args, err := Bind(row, map[string]any{"url": "https://matomo.example.com/SYNTHETICPATHTOKEN", "type": "analytics", "service": "matomo"})
	if err != nil {
		t.Fatalf("Bind a valid creation: %v", err)
	}
	if args.String("url") != "https://matomo.example.com/SYNTHETICPATHTOKEN" {
		t.Fatalf("the bound url = %q", args.String("url"))
	}
	if _, err := Bind(row, map[string]any{"url": "http://matomo.example.com/x", "type": "analytics", "service": "matomo"}); !errors.Is(err, domain.ErrInvalidURL) {
		t.Fatalf("Bind a plain HTTP address = %v, want ErrInvalidURL", err)
	}
	if _, err := Bind(row, map[string]any{"url": "https://slack.example.com/x", "type": "post", "service": "slack"}); err == nil {
		t.Fatal("Bind a service the create does not offer succeeded, want a refusal")
	}
}

// TestSliceC3StartGroupSyncNamesOnlyThatTheSyncStarted checks the result of the sync start: the redirect address of the
// identity provider is never a field, so the result holds only started.
func TestSliceC3StartGroupSyncNamesOnlyThatTheSyncStarted(t *testing.T) {
	row := rowOf(t, domain.OpAuthenticationProvidersStartGroupSync)
	if len(row.Fields) != 1 || row.Fields[0] != "started" {
		t.Fatalf("fields = %v, want [started]", row.Fields)
	}
}

// TestSliceC3IntegrationsCreateOffersOnlyTheServicesWithAURLSetting checks finding m8 against the endpoint inventory
// (outline-api-operations.md, integrations.create): the service enum holds only the services whose settings shape is {url}.
// The inventory lists the settings shapes without url as measurementId (google-analytics), diagrams, gitlab, and serviceTeamId,
// so those services are refused at bind time, before any request, and the four are not offered.
func TestSliceC3IntegrationsCreateOffersOnlyTheServicesWithAURLSetting(t *testing.T) {
	row := rowOf(t, domain.OpIntegrationsCreate)
	p, ok := row.Param("service")
	if !ok {
		t.Fatal("integrations.create has no service parameter")
	}
	if want := []string{"matomo", "umami"}; !reflect.DeepEqual(p.Values, want) {
		t.Fatalf("service values = %v, want %v", p.Values, want)
	}
	for _, service := range []string{"google-analytics", "diagrams", "gitlab", "grist"} {
		_, err := Bind(row, map[string]any{"url": "https://hooks.example.test/x", "type": "post", "service": service})
		if !errors.Is(err, domain.ErrInvalidEnum) {
			t.Errorf("service %s: Bind error = %v, want ErrInvalidEnum", service, err)
		}
	}
}
