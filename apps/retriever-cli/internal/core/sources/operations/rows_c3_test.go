package operations

import (
	"errors"
	"reflect"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// TestSliceC3CriticalRowsHaveNoToolAndNameTheirCanonicalTarget checks the three critical rows of slice c3 against the table of
// canonical targets. The sign-in provider deletion names the provider ID, and the two integration
// changes name the host of their URL setting. Each is critical on every call, needs --yes, and has no MCP tool at any ceiling.
func TestSliceC3CriticalRowsHaveNoToolAndNameTheirCanonicalTarget(t *testing.T) {
	cases := []struct {
		op     domain.OperationID
		kind   string
		host   string
		read   domain.OperationID
		field  string
		hasTgt bool
	}{
		{domain.OpAuthenticationProvidersDelete, "the authentication provider ID", "", domain.OpAuthenticationProvidersGet, "id", true},
		{domain.OpIntegrationsCreate, "the host of the URL setting", "url", "", "", false},
		{domain.OpIntegrationsUpdate, "the host of the URL setting", "url", domain.OpIntegrationsGet, "id", true},
	}
	for _, tc := range cases {
		row := rowOf(t, tc.op)
		if row.Critical == nil || row.Critical.When != "" || row.Critical.Kind != tc.kind || row.Critical.Host != tc.host {
			t.Errorf("%s: Critical = %+v, want critical on every call with kind %q and host %q", tc.op, row.Critical, tc.kind, tc.host)
		}
		if row.Tool != "" {
			t.Errorf("%s: critical, but it has the MCP tool %q", tc.op, row.Tool)
		}
		if !row.NeedsYes() {
			t.Errorf("%s: NeedsYes false, want true", tc.op)
		}
		if !tc.hasTgt {
			if row.Target != nil {
				t.Errorf("%s: Target = %+v, want none: the object does not exist before the call", tc.op, row.Target)
			}
			continue
		}
		if row.Target == nil || row.Target.Read != tc.read || row.Target.Field != tc.field {
			t.Errorf("%s: Target = %+v, want read %s and field %q", tc.op, row.Target, tc.read, tc.field)
		}
	}
}

// TestSliceC3ToolsAndCriticalRowsCount checks the counts of slice c3: 14 rows, 11 with an MCP tool, and 3 critical rows with
// none. With the rows of slices c1 and c2, they make 145 tools at the admin ceiling.
func TestSliceC3ToolsAndCriticalRowsCount(t *testing.T) {
	if len(adminRowsC3) != 14 {
		t.Fatalf("slice c3 holds %d rows, want 14", len(adminRowsC3))
	}
	tools, critical := 0, 0
	for _, r := range adminRowsC3 {
		if r.Tool != "" {
			tools++
		}
		if r.Critical != nil {
			critical++
		}
	}
	if tools != 11 || critical != 3 {
		t.Fatalf("slice c3 has %d tools and %d critical rows, want 11 and 3", tools, critical)
	}
}

// TestSliceC3AttachmentsDeleteHasNoTargetRead checks the one row of slice c3 with no target read: the catalog holds no read
// of one attachment by its ID, so its dry run shows the plan only.
func TestSliceC3AttachmentsDeleteHasNoTargetRead(t *testing.T) {
	row := rowOf(t, domain.OpAttachmentsDelete)
	if row.Target != nil {
		t.Fatalf("attachments.delete: Target = %+v, want none", row.Target)
	}
	if row.Positional() != "attachmentId" {
		t.Fatalf("attachments.delete: positional = %q, want attachmentId", row.Positional())
	}
}

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

// TestSliceC3HostOfKeepsOnlyTheLowerCaseHost checks the canonical value of a URL: its host, lower-cased,
// without the path, the query, the port, or the user name. An address without a host gives the empty text.
func TestSliceC3HostOfKeepsOnlyTheLowerCaseHost(t *testing.T) {
	cases := map[string]string{
		"https://Matomo.Example.com/SECRET/path?token=x": "matomo.example.com",
		"https://user:pw@hooks.example.test:8443/a":      "hooks.example.test",
		"https:///no-host": "",
		"not an address":   "",
	}
	for raw, want := range cases {
		if got := HostOf(raw); got != want {
			t.Errorf("HostOf(%q) = %q, want %q", raw, got, want)
		}
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
