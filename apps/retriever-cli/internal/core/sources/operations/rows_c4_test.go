package operations

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// TestSliceC4CriticalRowsNameTheirCanonicalTarget checks the four critical rows of slice c4 against the table of canonical
// targets. The OAuth client name is the new name itself, the OAuth client rotation names the client ID,
// and the two webhook rows name the host of their destination URL. Each is critical on every call, needs --yes, and has no
// MCP tool at any ceiling.
func TestSliceC4CriticalRowsNameTheirCanonicalTarget(t *testing.T) {
	cases := []struct {
		op    domain.OperationID
		kind  string
		host  string
		text  string
		read  domain.OperationID
		field string
		hasTg bool
	}{
		{domain.OpOAuthClientsCreate, "the new client name", "", "name", "", "", false},
		{domain.OpOAuthClientsRotateSecret, "the OAuth client ID", "", "", domain.OpOAuthClientsGet, "id", true},
		{domain.OpWebhookSubscriptionsCreate, "the host of the destination URL", "url", "", "", "", false},
		{domain.OpWebhookSubscriptionsUpdate, "the host of the destination URL", "url", "", "", "", false},
	}
	for _, tc := range cases {
		row := rowOf(t, tc.op)
		if row.Critical == nil || row.Critical.When != "" || row.Critical.Kind != tc.kind || row.Critical.Host != tc.host || row.Critical.Text != tc.text {
			t.Errorf("%s: Critical = %+v, want critical on every call with kind %q, host %q, and text %q",
				tc.op, row.Critical, tc.kind, tc.host, tc.text)
		}
		if row.Tool != "" {
			t.Errorf("%s: critical, but it has the MCP tool %q", tc.op, row.Tool)
		}
		if !row.NeedsYes() {
			t.Errorf("%s: NeedsYes false, want true", tc.op)
		}
		if !tc.hasTg {
			if row.Target != nil {
				t.Errorf("%s: Target = %+v, want none", tc.op, row.Target)
			}
			continue
		}
		if row.Target == nil || row.Target.Read != tc.read || row.Target.Field != tc.field {
			t.Errorf("%s: Target = %+v, want read %s and field %q", tc.op, row.Target, tc.read, tc.field)
		}
	}
}

// TestSliceC4ToolsAndCriticalRowsCount checks the counts of slice c4: 13 rows, 9 with an MCP tool, and 4 critical rows with
// none. With the rows of slices a to c3, the admin ceiling offers 154 tools.
func TestSliceC4ToolsAndCriticalRowsCount(t *testing.T) {
	if len(adminRowsC4) != 13 {
		t.Fatalf("slice c4 holds %d rows, want 13", len(adminRowsC4))
	}
	tools, critical := 0, 0
	for _, r := range adminRowsC4 {
		if r.Tool != "" {
			tools++
		}
		if r.Critical != nil {
			critical++
		}
	}
	if tools != 9 || critical != 4 {
		t.Fatalf("slice c4 has %d tools and %d critical rows, want 9 and 4", tools, critical)
	}
}

// TestCriticalRowsAreExactlyTheEleven pins the complete set of critical operations: eleven rows
// that are critical on every call, and none of them has an MCP tool at any ceiling. documents.delete is the one row critical
// only with permanent true, so it keeps its tool; a call of it without permanent has no critical value.
func TestCriticalRowsAreExactlyTheEleven(t *testing.T) {
	want := []domain.OperationID{
		domain.OpDocumentsEmptyTrash,
		domain.OpUsersDelete,
		domain.OpTeamsDelete,
		domain.OpGroupsDeleteAll,
		domain.OpAuthenticationProvidersDelete,
		domain.OpIntegrationsCreate,
		domain.OpIntegrationsUpdate,
		domain.OpOAuthClientsCreate,
		domain.OpOAuthClientsRotateSecret,
		domain.OpWebhookSubscriptionsCreate,
		domain.OpWebhookSubscriptionsUpdate,
	}
	var got []domain.OperationID
	for _, row := range Rows() {
		if row.Critical != nil && row.Critical.When == "" {
			got = append(got, row.ID)
			if row.Tool != "" {
				t.Errorf("%s is critical, but it has the MCP tool %q at a ceiling", row.ID, row.Tool)
			}
		}
	}
	slices.Sort(want)
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Fatalf("critical operations = %v, want exactly %v", got, want)
	}
	if len(got) != 11 {
		t.Fatalf("%d critical operations, want 11", len(got))
	}
	perm := rowOf(t, domain.OpDocumentsDelete)
	if perm.Critical == nil || perm.Critical.When != "permanent" || perm.Tool == "" {
		t.Fatalf("documents.delete: Critical = %+v, Tool = %q; want critical only with permanent and a tool", perm.Critical, perm.Tool)
	}
}

// TestSliceC4WebhookEventsAndRedirectsBindLikeTheirFieldReference checks the two list kinds of slice c4. The event names are
// `*` or a dotted name, at most 50 of them; the redirect addresses are one to ten HTTPS addresses of at most 1024 characters.
// A list that breaks a rule is refused, so no request is sent with it.
func TestSliceC4WebhookEventsAndRedirectsBindLikeTheirFieldReference(t *testing.T) {
	hook := rowOf(t, domain.OpWebhookSubscriptionsCreate)
	base := map[string]any{"url": "https://hooks.example.test/x", "name": "CI"}
	ok := map[string]any{"url": base["url"], "name": "CI", "events": "documents.create,*"}
	args, err := Bind(hook, ok)
	if err != nil {
		t.Fatalf("Bind valid events: %v", err)
	}
	if got := args.Values()["events"]; !slices.Equal(got.([]string), []string{"documents.create", "*"}) {
		t.Fatalf("events = %v, want [documents.create *]", got)
	}
	for _, bad := range []any{"bad event!", []any{"documents.create", "a/b"}, "x"} {
		in := map[string]any{"url": base["url"], "name": "CI", "events": bad}
		if _, err := Bind(hook, in); !errors.Is(err, domain.ErrInvalidName) {
			t.Errorf("events %v: Bind = %v, want ErrInvalidName", bad, err)
		}
	}

	client := rowOf(t, domain.OpOAuthClientsCreate)
	good := map[string]any{"name": "Reader app", "redirectUris": "https://app.example.test/cb,https://app.example.test/cb2"}
	if _, err := Bind(client, good); err != nil {
		t.Fatalf("Bind two redirect addresses: %v", err)
	}
	if _, err := Bind(client, map[string]any{"name": "x", "redirectUris": "http://app.example.test/cb"}); !errors.Is(err, domain.ErrInvalidURL) {
		t.Errorf("a plain HTTP redirect: Bind = %v, want ErrInvalidURL", err)
	}
	eleven := make([]string, 11)
	for i := range eleven {
		eleven[i] = "https://app.example.test/cb"
	}
	if _, err := Bind(client, map[string]any{"name": "x", "redirectUris": eleven}); !errors.Is(err, domain.ErrInvalidURL) {
		t.Errorf("11 redirect addresses: Bind = %v, want ErrInvalidURL", err)
	}
	if _, err := Bind(client, map[string]any{"name": "x", "redirectUris": []string{}}); !errors.Is(err, domain.ErrInvalidURL) {
		t.Errorf("no redirect address: Bind = %v, want ErrInvalidURL", err)
	}
	long := "https://app.example.test/" + strings.Repeat("a", domain.MaxURLLength)
	if _, err := Bind(client, map[string]any{"name": "x", "redirectUris": []string{long}}); !errors.Is(err, domain.ErrInvalidURL) {
		t.Errorf("a redirect address over 1024 characters: Bind = %v, want ErrInvalidURL", err)
	}
}

// TestSliceC4SharesCreateRepeatsTheSameShare checks the one row of the slice whose repeat answers the same object: a share
// of a document that is already shared is the same share, so its tool announces idempotent. It is the
// only row of the catalog with the flag.
func TestSliceC4SharesCreateRepeatsTheSameShare(t *testing.T) {
	for _, row := range Rows() {
		if row.SameOnRepeat != (row.ID == domain.OpSharesCreate) {
			t.Errorf("%s: SameOnRepeat = %v, want true for shares.create only", row.ID, row.SameOnRepeat)
		}
	}
}

// TestSliceC4SharesCreateNeedsExactlyOneTarget checks that a share names its document or its collection, not both and not
// neither, through the one-of rule of the catalog.
func TestSliceC4SharesCreateNeedsExactlyOneTarget(t *testing.T) {
	row := rowOf(t, domain.OpSharesCreate)
	if !slices.Equal(row.OneOf, []string{"documentId", "collectionId"}) {
		t.Fatalf("OneOf = %v, want documentId and collectionId", row.OneOf)
	}
	if _, err := Bind(row, map[string]any{"documentId": "11111111-1111-4111-8111-111111111111", "collectionId": "11111111-1111-4111-8111-111111111111"}); err == nil {
		t.Error("a share of a document and a collection was bound")
	}
	if _, err := Bind(row, map[string]any{"published": true}); err == nil {
		t.Error("a share of neither a document nor a collection was bound")
	}
}

// TestSliceC4SecretBearingRepliesHaveNoSecretField checks that the fields of the rows that receive a client secret or a
// signing secret do not name it, and that a share's fields name no public link (the field references, oauthClient, webhook,
// and share). Those values are never a field of a result, so no caller can print them.
func TestSliceC4SecretBearingRepliesHaveNoSecretField(t *testing.T) {
	for _, op := range []domain.OperationID{
		domain.OpOAuthClientsCreate, domain.OpOAuthClientsRotateSecret, domain.OpOAuthClientsUpdate,
		domain.OpWebhookSubscriptionsCreate, domain.OpWebhookSubscriptionsUpdate,
	} {
		row := rowOf(t, op)
		for _, f := range row.Fields {
			if f == "clientSecret" || f == "secret" || f == "registrationToken" {
				t.Errorf("%s: field %q would carry a secret", op, f)
			}
		}
	}
	for _, op := range []domain.OperationID{domain.OpSharesCreate, domain.OpSharesUpdate} {
		row := rowOf(t, op)
		for _, f := range row.Fields {
			if f == "url" || f == "urlId" {
				t.Errorf("%s: field %q would carry the public link", op, f)
			}
		}
	}
}
