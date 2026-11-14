//go:build integration

package outline

import (
	"context"
	"fmt"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

// TestSliceC4EveryRequestIsSentOnceWithItsExactBody checks, for each case of slice c4, the method path and the exact body of
// the change, and that the change is the only request (the rows of slice c4 read no target at the gateway; the critical
// check is the use case's, and it reads the target there).
func TestSliceC4EveryRequestIsSentOnceWithItsExactBody(t *testing.T) {
	for _, tc := range requestCasesC4 {
		t.Run(tc.name, func(t *testing.T) {
			gw, rec := recordingAdminCall(t, tc.op)
			if _, err := gw.Write(context.Background(), tc.op, domain.NewArgs(tc.args)); err != nil {
				t.Fatalf("Write = %v", err)
			}
			calls := rec.recorded()
			if want := 1 + len(adminPreReadsOf(tc.op)); len(calls) != want {
				t.Fatalf("requests = %d, want %d", len(calls), want)
			}
			change := calls[len(calls)-1]
			if change.Path != tc.path {
				t.Fatalf("path = %q, want %q", change.Path, tc.path)
			}
			if !reflect.DeepEqual(change.Body, bodyOf(t, tc.body)) {
				t.Fatalf("body = %v, want %s", change.Body, tc.body)
			}
		})
	}
}

// TestSliceC4RepliesAreReadLikeTheirRows checks the field each row of the slice returns from its reply: an acknowledgement
// holds ok, and an object reply holds its own fields, with the secret and the link of the reply left out.
func TestSliceC4RepliesAreReadLikeTheirRows(t *testing.T) {
	cases := []struct {
		op     domain.OperationID
		field  string
		expect string
	}{
		{domain.OpOAuthAuthenticationsDelete, "ok", "true"},
		{domain.OpOAuthClientsCreate, "name", "CI app"},
		{domain.OpOAuthClientsDelete, "ok", "true"},
		{domain.OpOAuthClientsRotateSecret, "clientId", "client-1"},
		{domain.OpOAuthClientsUpdate, "name", "CI app renamed"},
		{domain.OpRevisionsDelete, "ok", "true"},
		{domain.OpSharesCreate, "documentId", "doc-1"},
		{domain.OpSharesRevoke, "ok", "true"},
		{domain.OpSharesUpdate, "title", "Plan for review"},
		{domain.OpTemplatesDelete, "ok", "true"},
		{domain.OpWebhookSubscriptionsCreate, "name", "CI events"},
		{domain.OpWebhookSubscriptionsDelete, "ok", "true"},
		{domain.OpWebhookSubscriptionsUpdate, "urlHost", "new.example.test"},
	}
	for _, tc := range cases {
		t.Run(string(tc.op), func(t *testing.T) {
			r := adminRig(t)
			method := adminEndpoints[tc.op].method
			r.stub.SetReply(method, http.StatusOK, string(loadSynthetic(t, method)))
			setPreReadReplies(t, r.stub, tc.op)
			out, err := r.gw.Write(context.Background(), tc.op, domain.NewArgs(caseArgsC4(t, tc.op)))
			if err != nil {
				t.Fatalf("Write = %v", err)
			}
			if v, _ := out.Get(tc.field); fmt.Sprint(v) != tc.expect {
				t.Fatalf("field %s = %v, want %q", tc.field, v, tc.expect)
			}
		})
	}
}

// TestSliceC4SecretsAndLinksAreNeverProjected is the proof of the redaction of slice c4 at the adapter. Each reply of the
// slice carries a secret or a public link (the OAuth client secret of create, rotate, and update, the signing secret of a
// webhook, and the link of a share). The result of each call must hold none of them, in any field or nested object, and
// its fields must not name them. The sentinels are the ones of the synthetic replies.
func TestSliceC4SecretsAndLinksAreNeverProjected(t *testing.T) {
	sentinels := []string{"SYNTHETICCLIENTSECRET", "SYNTHETICROTATEDSECRET", "SYNTHETICWEBHOOKSECRET", "SYNTHETICSHARELINK"}
	ops := []domain.OperationID{
		domain.OpOAuthClientsCreate, domain.OpOAuthClientsRotateSecret, domain.OpOAuthClientsUpdate,
		domain.OpWebhookSubscriptionsCreate, domain.OpWebhookSubscriptionsUpdate, domain.OpSharesCreate, domain.OpSharesUpdate,
	}
	for _, op := range ops {
		t.Run(string(op), func(t *testing.T) {
			r := adminRig(t)
			method := adminEndpoints[op].method
			r.stub.SetReply(method, http.StatusOK, string(loadSynthetic(t, method)))
			out, err := r.gw.Write(context.Background(), op, domain.NewArgs(caseArgsC4(t, op)))
			if err != nil {
				t.Fatalf("the call did not succeed, so the proof would pass vacuously: %v", err)
			}
			text := fmt.Sprintf("%#v", out)
			for _, s := range sentinels {
				if strings.Contains(text, s) {
					t.Fatalf("the result of %s holds %s", op, s)
				}
			}
			for _, name := range []string{"clientSecret", "secret", "url", "urlId"} {
				if _, has := out.Get(name); has {
					t.Fatalf("the result of %s has the field %s", op, name)
				}
			}
		})
	}
}
