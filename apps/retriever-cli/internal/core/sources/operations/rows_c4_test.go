package operations

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

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
