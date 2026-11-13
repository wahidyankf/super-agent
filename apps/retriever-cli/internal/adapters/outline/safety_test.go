package outline

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/tests/support/outlinestub"
)

func TestRateLimitRetriesThreeTimesThenStops(t *testing.T) {
	r := newRig(t, sampleOptions())
	r.stub.SetFault(outlinestub.Fault{Status: 429, RetryAfter: "1"})

	_, err := r.gw.ListCollections(context.Background())
	if !errors.Is(err, domain.ErrSourceRateLimited) {
		t.Fatalf("ListCollections error = %v, want ErrSourceRateLimited", err)
	}
	if n := r.stub.Calls("collections.list"); n != 4 {
		t.Fatalf("collections.list calls = %d, want 4 (one try and three retries)", n)
	}
	want := []time.Duration{time.Second, time.Second, time.Second}
	if got := r.clock.recordedSleeps(); !equalDurations(got, want) {
		t.Fatalf("waits = %v, want %v", got, want)
	}
}

func TestRateLimitRetrySucceedsAfterOneWait(t *testing.T) {
	r := newRig(t, []outlinestub.Option{outlinestub.WithCollections(outlinestub.NumberedCollections(1)...)})
	r.stub.SetFaultFirst(1, outlinestub.Fault{Status: 429, RetryAfter: "1"})

	got, err := r.gw.ListCollections(context.Background())
	if err != nil {
		t.Fatalf("ListCollections: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("ListCollections returned %d collections, want 1", len(got))
	}
	if n := r.stub.Calls("collections.list"); n != 2 {
		t.Fatalf("collections.list calls = %d, want 2", n)
	}
	if w := r.clock.recordedSleeps(); !equalDurations(w, []time.Duration{time.Second}) {
		t.Fatalf("waits = %v, want [1s]", w)
	}
}

func TestRetryAfterOverSixtySecondsFailsWithoutRetry(t *testing.T) {
	r := newRig(t, sampleOptions())
	r.stub.SetFault(outlinestub.Fault{Status: 429, RetryAfter: "61"})

	_, err := r.gw.ListCollections(context.Background())
	if !errors.Is(err, domain.ErrRetryAfterTooLong) {
		t.Fatalf("ListCollections error = %v, want ErrRetryAfterTooLong", err)
	}
	if n := r.stub.Calls("collections.list"); n != 1 {
		t.Fatalf("collections.list calls = %d, want 1 (no retry)", n)
	}
	if w := r.clock.recordedSleeps(); len(w) != 0 {
		t.Fatalf("waits = %v, want none", w)
	}
}

func TestRetryAfterOfExactlySixtySecondsIsWaited(t *testing.T) {
	r := newRig(t, []outlinestub.Option{outlinestub.WithCollections(outlinestub.NumberedCollections(1)...)})
	r.stub.SetFaultFirst(1, outlinestub.Fault{Status: 429, RetryAfter: "60"})

	if _, err := r.gw.ListCollections(context.Background()); err != nil {
		t.Fatalf("ListCollections: %v", err)
	}
	if w := r.clock.recordedSleeps(); !equalDurations(w, []time.Duration{60 * time.Second}) {
		t.Fatalf("waits = %v, want [1m0s]", w)
	}
}

func TestUnusableRetryAfterFailsWithoutRetry(t *testing.T) {
	for _, value := range []string{"", "soon", "-1", "Wed, 04 Nov 2026 09:00:00 GMT"} {
		t.Run(fmt.Sprintf("Retry-After %q", value), func(t *testing.T) {
			r := newRig(t, sampleOptions())
			r.stub.SetFault(outlinestub.Fault{Status: 429, RetryAfter: value})

			_, err := r.gw.ListCollections(context.Background())
			if !errors.Is(err, domain.ErrSourceRateLimited) {
				t.Fatalf("ListCollections error = %v, want ErrSourceRateLimited", err)
			}
			if n := r.stub.Calls("collections.list"); n != 1 {
				t.Fatalf("collections.list calls = %d, want 1 (no retry without a usable wait)", n)
			}
		})
	}
}

func TestRedirectIsNotFollowed(t *testing.T) {
	r := newRig(t, sampleOptions())
	r.stub.SetFault(outlinestub.Fault{Status: 302, Location: "https://evil.example.com/api/collections.list"})

	_, err := r.gw.ListCollections(context.Background())
	if !errors.Is(err, domain.ErrRedirected) {
		t.Fatalf("ListCollections error = %v, want ErrRedirected", err)
	}
	if n := len(r.stub.Requests()); n != 1 {
		t.Fatalf("requests = %d, want 1: the redirect must not be followed", n)
	}
}

func TestRequestsRunOneAtATimeWithTheMinimumDelay(t *testing.T) {
	r := newRig(t,
		[]outlinestub.Option{outlinestub.WithCollections(outlinestub.NumberedCollections(60)...)},
		WithMinDelay(defaultMinDelay),
	)

	if _, err := r.gw.ListCollections(context.Background()); err != nil {
		t.Fatalf("ListCollections: %v", err)
	}
	reqs := r.stub.Requests()
	if len(reqs) != 3 {
		t.Fatalf("requests = %d, want 3", len(reqs))
	}
	for i := 1; i < len(reqs); i++ {
		if gap := reqs[i].At.Sub(reqs[i-1].At); gap < defaultMinDelay {
			t.Fatalf("gap before request %d = %v, want at least %v", i, gap, defaultMinDelay)
		}
	}
	if peak := r.stub.PeakInFlight(); peak != 1 {
		t.Fatalf("peak requests in flight = %d, want 1", peak)
	}
}

func TestPlainHTTPAndAnyNonHTTPSBaseAreRefusedBeforeAnyRequest(t *testing.T) {
	for _, base := range []string{
		"http://docs.example.test",
		"http://127.0.0.1:8080",
		"docs.example.test",
		"ftp://docs.example.test",
	} {
		t.Run(base, func(t *testing.T) {
			stub := outlinestub.New(sampleOptions()...)
			client := &http.Client{Transport: stub.RoundTripper()}
			gw, err := NewWithBase(testSource(t, testHost), base, client, &staticCreds{})
			if !errors.Is(err, domain.ErrHTTPSRequired) {
				t.Fatalf("NewWithBase(%q) error = %v, want ErrHTTPSRequired", base, err)
			}
			if gw != nil {
				t.Fatalf("NewWithBase(%q) returned a gateway with an error", base)
			}
			if n := len(stub.Requests()); n != 0 {
				t.Fatalf("requests = %d, want 0", n)
			}
		})
	}
}

func TestBaseWithoutAHostIsInvalidNotHTTPS(t *testing.T) {
	gw, err := NewWithBase(testSource(t, testHost), "https://", &http.Client{}, &staticCreds{})
	if err == nil || errors.Is(err, domain.ErrHTTPSRequired) {
		t.Fatalf("NewWithBase(https://) error = %v, want an invalid base error", err)
	}
	if gw != nil {
		t.Fatalf("NewWithBase(https://) returned a gateway with an error")
	}
}

// TestTheEndpointTableHoldsExactlyTheStageAReads checks the endpoint table against the stage a reads of the catalog
// (ADR 007 section 3.2, layer 3): the sixty-six catalog reads of parts 1, 2a, and 2b, each with an endpoint, and no other.
func TestTheEndpointTableHoldsExactlyTheStageAReads(t *testing.T) {
	want := []domain.OperationID{
		domain.OpAuthWhoami, domain.OpCollectionsInfo, domain.OpCollectionsList, domain.OpCollectionsDocuments,
		domain.OpDocumentsGet, domain.OpDocumentsList, domain.OpDocumentsSearch,
		domain.OpDocumentsArchived, domain.OpDocumentsChildren, domain.OpDocumentsDeleted, domain.OpDocumentsDrafts,
		domain.OpDocumentsGroupMemberships, domain.OpDocumentsInsights, domain.OpDocumentsMemberships,
		domain.OpDocumentsSearchTitles, domain.OpDocumentsUsers, domain.OpDocumentsViewed,
		domain.OpCollectionsMemberships, domain.OpCollectionsGroupMembers,
		domain.OpCommentsGet, domain.OpCommentsList, domain.OpRevisionsExport, domain.OpRevisionsGet,
		domain.OpRevisionsList, domain.OpSharesGet, domain.OpSharesList, domain.OpUsersGet, domain.OpUsersList,
		domain.OpGroupsGet, domain.OpGroupsList, domain.OpGroupsMembers, domain.OpTemplatesGet, domain.OpTemplatesList,
		domain.OpPinsGet, domain.OpPinsList, domain.OpStarsList, domain.OpSubscriptionsGet, domain.OpSubscriptionsList,
		domain.OpAccessRequestsGet, domain.OpAttachmentsList, domain.OpEmojisGet, domain.OpEmojisList,
		domain.OpEventsList, domain.OpFileOperationsGet, domain.OpFileOperationsList, domain.OpGroupMembershipsList,
		domain.OpInstallationGet, domain.OpNotificationsList, domain.OpOAuthAuthenticationsList, domain.OpReactionsList,
		domain.OpRelationshipsGet, domain.OpRelationshipsList, domain.OpSearchesList, domain.OpSuggestionsMention,
		domain.OpUserMembershipsList, domain.OpViewsList, domain.OpAPIKeysList, domain.OpAuthenticationProvidersGet,
		domain.OpAuthenticationProvidersList, domain.OpImportsGet, domain.OpImportsList, domain.OpIntegrationsGet,
		domain.OpIntegrationsList, domain.OpOAuthClientsGet, domain.OpOAuthClientsList,
		domain.OpWebhookSubscriptionsList,
	}
	got := make([]domain.OperationID, 0, len(endpoints))
	for op := range endpoints {
		got = append(got, op)
	}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("endpoint table = %v, want %v", got, want)
	}
}

// TestTheCallRefusesEveryOperationOutsideTheTable checks that a method the table does not name, or an empty name,
// never reaches the network.
func TestTheCallRefusesEveryOperationOutsideTheTable(t *testing.T) {
	r := newRig(t, sampleOptions())
	for _, op := range []domain.OperationID{"documents.delete", "collections.delete", "documents.update", "", "../collections.list"} {
		t.Run(fmt.Sprintf("operation %q", op), func(t *testing.T) {
			var out any
			err := r.gw.call(context.Background(), op, string(op), struct{}{}, &out)
			if !errors.Is(err, errMethodNotAllowed) {
				t.Fatalf("call error = %v, want errMethodNotAllowed", err)
			}
		})
	}
	if n := len(r.stub.Requests()); n != 0 {
		t.Fatalf("requests = %d, want 0 for refused methods", n)
	}
}

// TestTheReadSourceMayMakeOnlyReadCalls checks the level rule of the adapter itself: an operation the source does not
// reach is refused before the request, and the export call is a read.
func TestTheReadSourceMayMakeOnlyReadCalls(t *testing.T) {
	r := newRig(t, sampleOptions())
	if !r.gw.permits(domain.OpDocumentsGet) || !r.gw.permits(opExport) {
		t.Fatal("a read source is refused a read call")
	}
	r.gw.access = domain.Access{}
	if r.gw.permits(domain.OpDocumentsGet) {
		t.Fatal("a source with no access level is permitted a read")
	}
}

func TestTheKeyIsReadPerRequestAndSentAsBearerToTheLogicalHost(t *testing.T) {
	r := newRig(t, sampleOptions())
	ctx := context.Background()

	if _, err := r.gw.ListCollections(ctx); err != nil {
		t.Fatalf("ListCollections: %v", err)
	}
	if n := r.creds.count(); n != 1 {
		t.Fatalf("key reads after one request = %d, want 1 (read lazily per request)", n)
	}
	if _, err := r.gw.Document(ctx, mustDocID(t, "doc-1")); err != nil {
		t.Fatalf("Document: %v", err)
	}
	if n := r.creds.count(); n != 2 {
		t.Fatalf("key reads after one list and one document call = %d, want 2 (one per request)", n)
	}
	for i, req := range r.stub.Requests() {
		if req.Authorization != "Bearer "+sentinelKey {
			t.Fatalf("request %d Authorization was not the bearer key", i)
		}
		if req.Host != "docs.example.test" {
			t.Fatalf("request %d Host = %q, want docs.example.test", i, req.Host)
		}
	}
}

func TestMissingKeyStopsBeforeAnyRequestAndNamesTheVariable(t *testing.T) {
	stub := outlinestub.New(sampleOptions()...)
	client := &http.Client{Transport: stub.RoundTripper()}
	gw, err := NewWithBase(testSource(t, testHost), testBase, client, missingCreds{}, WithMinDelay(0))
	if err != nil {
		t.Fatalf("NewWithBase: %v", err)
	}

	_, err = gw.ListCollections(context.Background())
	var missing domain.MissingKeyError
	if !errors.As(err, &missing) {
		t.Fatalf("ListCollections error = %v, want a MissingKeyError", err)
	}
	if missing.Variable != keyVariable {
		t.Fatalf("missing variable = %q, want %q", missing.Variable, keyVariable)
	}
	if !strings.Contains(err.Error(), keyVariable) {
		t.Fatalf("error %q does not name %s", err, keyVariable)
	}
	if n := len(stub.Requests()); n != 0 {
		t.Fatalf("requests = %d, want 0 without a key", n)
	}
}

func TestDefaultsMatchTheSafetyRules(t *testing.T) {
	g := New(testSource(t, testHost), missingCreds{})
	if want := "https://" + testHost; g.base != want {
		t.Fatalf("base = %q, want %q", g.base, want)
	}
	if g.minDelay != 100*time.Millisecond {
		t.Fatalf("minDelay = %v, want 100ms", g.minDelay)
	}
	if g.timeout != 30*time.Second {
		t.Fatalf("timeout = %v, want 30s", g.timeout)
	}
	if g.client == nil || g.client.CheckRedirect == nil {
		t.Fatalf("client must be set and must refuse redirects")
	}
}

// TestTheProductionClientNeverUsesAnEnvironmentProxy checks the transport the production
// constructor gives its client. Its Proxy is nil, so HTTPS_PROXY and the other proxy
// variables are never read. It is a copy with its own timeouts, so the shared default
// transport keeps the standard library's proxy setting.
func TestTheProductionClientNeverUsesAnEnvironmentProxy(t *testing.T) {
	g := New(testSource(t, testHost), missingCreds{})
	tr, ok := g.client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("production client transport is %T, want its own *http.Transport", g.client.Transport)
	}
	if tr.Proxy != nil {
		t.Fatalf("production transport has a proxy function, want nil so no environment proxy is used")
	}
	if tr.DialContext == nil || tr.TLSHandshakeTimeout <= 0 {
		t.Fatalf("production transport lacks a dialer or a TLS handshake timeout")
	}
	def, ok := http.DefaultTransport.(*http.Transport)
	if !ok || def.Proxy == nil {
		t.Fatalf("http.DefaultTransport was changed, want the production transport to be a copy")
	}
}

// replacedTransport stands in for a default transport that another package swapped for a non-HTTP one.
type replacedTransport struct{}

func (replacedTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("no network in this test")
}

// TestNewSurvivesAReplacedDefaultTransport checks that the production constructor still builds a gateway that
// never reads a proxy, when http.DefaultTransport is not an *http.Transport.
func TestNewSurvivesAReplacedDefaultTransport(t *testing.T) {
	saved := http.DefaultTransport
	http.DefaultTransport = replacedTransport{}
	t.Cleanup(func() { http.DefaultTransport = saved })

	g := New(testSource(t, testHost), missingCreds{})
	tr, ok := g.client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("production client transport is %T, want a *http.Transport", g.client.Transport)
	}
	if tr.Proxy != nil || tr.DialContext == nil || tr.TLSHandshakeTimeout <= 0 {
		t.Fatalf("fallback transport lacks the no-proxy setting or the dial and TLS timeouts")
	}
}

// TestDroppedConnectionIsUnreachableForEveryMethod checks the stub's no-network fault: a
// dropped connection reaches every gateway method as ErrUnreachable.
func TestDroppedConnectionIsUnreachableForEveryMethod(t *testing.T) {
	r := newRig(t, sampleOptions())
	r.stub.SetFault(outlinestub.Fault{Drop: true})
	ctx := context.Background()

	_, listErr := r.gw.ListCollections(ctx)
	_, treeErr := r.gw.CollectionTree(ctx, mustColID(t, "col-1"))
	_, docErr := r.gw.Document(ctx, mustDocID(t, "doc-1"))
	for _, err := range []error{listErr, treeErr, docErr} {
		if !errors.Is(err, domain.ErrUnreachable) {
			t.Fatalf("error = %v, want ErrUnreachable for a dropped connection", err)
		}
	}
}

func TestBodyOverTheCapIsUnexpected(t *testing.T) {
	r := newRig(t, sampleOptions())
	r.stub.SetFault(outlinestub.Fault{Status: 200, Body: strings.Repeat("x", maxBodyBytes+1)})

	if _, err := r.gw.ListCollections(context.Background()); !errors.Is(err, domain.ErrReplyTooLarge) {
		t.Fatalf("ListCollections error = %v, want ErrReplyTooLarge for a body over the cap", err)
	}
}

// TestNoFailureEverExposesTheKey runs every failure path and checks each error, and
// the gateway itself, through the formats a log or a terminal would use.
func TestNoFailureEverExposesTheKey(t *testing.T) {
	cases := append(failureCases(),
		statusCase{name: "401 that echoes the key", fault: outlinestub.Fault{Status: 401, EchoAuthorization: true}},
		statusCase{name: "500 that echoes the key", fault: outlinestub.Fault{Status: 500, EchoAuthorization: true}},
		statusCase{name: "200 that is not JSON and echoes the key", fault: outlinestub.Fault{Status: 200, EchoAuthorization: true}},
		statusCase{name: "429 until the retries run out", fault: outlinestub.Fault{Status: 429, RetryAfter: "1"}},
	)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newRig(t, sampleOptions())
			r.stub.SetFault(tc.fault)
			ctx := context.Background()
			_, listErr := r.gw.ListCollections(ctx)
			_, treeErr := r.gw.CollectionTree(ctx, mustColID(t, "col-1"))
			_, docErr := r.gw.Document(ctx, mustDocID(t, "doc-1"))
			for _, err := range []error{listErr, treeErr, docErr} {
				if err == nil {
					t.Fatalf("expected a failure for %s", tc.name)
				}
				assertNoKey(t, err)
			}
			assertNoKey(t, r.gw)
		})
	}
}

func TestNoMissingKeyErrorExposesAKey(t *testing.T) {
	stub := outlinestub.New(sampleOptions()...)
	client := &http.Client{Transport: stub.RoundTripper()}
	gw, err := NewWithBase(testSource(t, testHost), testBase, client, missingCreds{})
	if err != nil {
		t.Fatalf("NewWithBase: %v", err)
	}
	_, err = gw.ListCollections(context.Background())
	if err == nil {
		t.Fatalf("expected a missing-key error")
	}
	assertNoKey(t, err)
	assertNoKey(t, gw)
	if n := len(stub.Requests()); n != 0 {
		t.Fatalf("requests = %d, want 0 without a key", n)
	}
}

// assertNoKey fails when the value, in any common format, contains the sentinel key.
func assertNoKey(t *testing.T, v any) {
	t.Helper()
	for _, form := range []string{"%v", "%+v", "%#v", "%s"} {
		if out := fmt.Sprintf(form, v); strings.Contains(out, sentinelKey) {
			t.Fatalf("key leaked through %s: %q", form, out)
		}
	}
}

// equalDurations reports whether two wait lists match in order.
func equalDurations(a, b []time.Duration) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
