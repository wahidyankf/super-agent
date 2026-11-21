//go:build integration

package integration_test

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-connector-kiwi/kiwi"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-testkit/kiwistub"
)

// addCase sends the one write the transport bindings use: a case added to a plan.
func addCase(gw *kiwi.Gateway) error {
	_, err := gw.Write(context.Background(), domain.OpKiwiPlansAddCase, domain.NewArgs(map[string]any{"planId": "plan-2", "caseId": "case-12"}))
	return err
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/transport-and-retries.feature: Every call is one POST to the endpoint with the Bearer key on the request
func TestEveryCallIsOnePOSTToTheEndpointWithTheKey(t *testing.T) {
	srv := serveKiwi(t, newStub(sampleDataset()), nil)
	gw := gatewayAt(t, srv, staticCreds{})
	for i := 0; i < 2; i++ {
		_, err := gw.ListCollections(context.Background())
		must(t, err)
	}
	reqs := srv.Stub.Requests()
	if len(reqs) != 2 {
		t.Fatalf("%d requests were sent, want 2", len(reqs))
	}
	for _, q := range reqs {
		if q.Verb != http.MethodPost || q.Path != kiwistub.Endpoint || q.Host != testHost || !q.AuthOK {
			t.Errorf("the request is %s %s to host %q with key accepted=%v", q.Verb, q.Path, q.Host, q.AuthOK)
		}
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/transport-and-retries.feature: A redirect in a reply is read as a refusal and the client makes one request
func TestRedirectIsReadAsRefusalAndOneRequestIsMade(t *testing.T) {
	srv := serveKiwi(t, newStub(sampleDataset()), nil)
	srv.Stub.SetFault(kiwistub.Fault{Redirect: true})
	gw := gatewayAt(t, srv, staticCreds{})
	_, err := gw.ListCollections(context.Background())
	if !errors.Is(err, domain.ErrRedirected) {
		t.Errorf("the call failed with %v, want the redirect error", err)
	}
	if n := len(srv.Stub.Requests()); n != 1 {
		t.Errorf("%d requests were sent, want 1", n)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/transport-and-retries.feature: A cookie in a reply is never sent back on the next request
func TestCookieIsNeverSentBackOnTheNextRequest(t *testing.T) {
	setCookie := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Set-Cookie", "session=abc123; Path=/")
			next.ServeHTTP(w, r)
		})
	}
	srv := serveKiwi(t, newStub(sampleDataset()), setCookie)
	gw := gatewayAt(t, srv, staticCreds{})
	for i := 0; i < 2; i++ {
		_, err := gw.ListCollections(context.Background())
		must(t, err)
	}
	seen := srv.Log.all()
	if len(seen) != 2 {
		t.Fatalf("%d requests reached the server, want 2", len(seen))
	}
	if seen[1].Cookie != "" {
		t.Errorf("the second request carries the Cookie header %q", seen[1].Cookie)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/transport-and-retries.feature: A read that times out is a timed-out read and a write that times out is an unknown outcome
func TestReadTimeoutIsTimedOutAndWriteTimeoutIsUnknownOutcome(t *testing.T) {
	srv := serveKiwi(t, newStub(sampleDataset()), nil)
	gw := gatewayAt(t, srv, staticCreds{}, kiwi.WithTimeout(100*time.Millisecond))
	srv.Stub.SetFault(kiwistub.Fault{Never: true})
	_, err := gw.ListCollections(context.Background())
	if !errors.Is(err, domain.ErrTimedOut) {
		t.Errorf("the read failed with %v, want the timed-out error", err)
	}
	srv.Stub.ClearFault()
	srv.Stub.SetMethodFault("TestPlan.add_case", kiwistub.Fault{Never: true})
	err = addCase(gw)
	if !errors.Is(err, domain.ErrWriteOutcomeUnknown) {
		t.Errorf("the write failed with %v, want the unknown-outcome error", err)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/transport-and-retries.feature: A read whose reply is over 10 MiB is refused as too large
func TestReadReplyOverTenMiBIsRefusedAsTooLarge(t *testing.T) {
	srv := serveKiwi(t, newStub(sampleDataset()), nil)
	srv.Stub.SetFault(kiwistub.Fault{Oversize: true})
	gw := gatewayAt(t, srv, staticCreds{})
	_, err := gw.ListCollections(context.Background())
	if !errors.Is(err, domain.ErrReplyTooLarge) {
		t.Errorf("the read failed with %v, want the too-large error", err)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/transport-and-retries.feature: A write whose reply is over 10 MiB has an unknown outcome and is not retried
func TestWriteReplyOverTenMiBHasUnknownOutcomeAndIsNotRetried(t *testing.T) {
	srv := serveKiwi(t, newStub(sampleDataset()), nil)
	srv.Stub.SetMethodFault("TestPlan.add_case", kiwistub.Fault{Oversize: true})
	gw := gatewayAt(t, srv, staticCreds{})
	err := addCase(gw)
	if !errors.Is(err, domain.ErrWriteOutcomeUnknown) {
		t.Errorf("the write failed with %v, want the unknown-outcome error", err)
	}
	if n := srv.Stub.Calls("TestPlan.add_case"); n != 1 {
		t.Errorf("%d add requests were sent, want 1", n)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/transport-and-retries.feature: Two calls are never in flight at the same time
func TestTwoCallsAreNeverInFlightAtTheSameTime(t *testing.T) {
	srv := serveKiwi(t, newStub(sampleDataset()), nil)
	gw := gatewayAt(t, srv, staticCreds{})
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := gw.ListCollections(context.Background())
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		must(t, err)
	}
	if peak := srv.Stub.PeakInFlight(); peak != 1 {
		t.Errorf("%d requests were in flight at once, want 1", peak)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/transport-and-retries.feature: The second request waits the full 100 ms pacing delay after the first
func TestSecondRequestWaitsTheFullPacingDelay(t *testing.T) {
	srv := serveKiwi(t, newStub(sampleDataset()), nil)
	gw, err := kiwi.NewWithBase(kiwiSource(t), srv.Base, srv.Client, staticCreds{})
	must(t, err)
	for i := 0; i < 2; i++ {
		_, err := gw.ListCollections(context.Background())
		must(t, err)
	}
	reqs := srv.Stub.Requests()
	if len(reqs) != 2 {
		t.Fatalf("%d requests were sent, want 2", len(reqs))
	}
	if gap := reqs[1].At.Sub(reqs[0].At); gap < 100*time.Millisecond {
		t.Errorf("the second request came %v after the first, want at least 100ms", gap)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/transport-and-retries.feature: A rate-limited call with a short Retry-After is sent again after the wait
func TestRateLimitedCallWithShortRetryAfterIsSentAgain(t *testing.T) {
	srv := serveKiwi(t, newStub(sampleDataset()), nil)
	srv.Stub.SetFaultFirst(kiwistub.Fault{Status: http.StatusTooManyRequests, RetryAfter: "1"})
	rec := &recorder{}
	gw := gatewayAt(t, srv, staticCreds{}, kiwi.WithSleep(rec.Sleep))
	_, err := gw.ListCollections(context.Background())
	must(t, err)
	if n := len(srv.Stub.Requests()); n != 2 {
		t.Errorf("%d requests were sent, want 2", n)
	}
	if got := rec.Waits(); len(got) != 1 || got[0] != time.Second {
		t.Errorf("the waits are %v, want one wait of 1s", got)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/transport-and-retries.feature: A source that keeps answering 429 stops after one request and three retries with the rate-limit error
func TestPersistentRateLimitStopsAfterThreeRetries(t *testing.T) {
	srv := serveKiwi(t, newStub(sampleDataset()), nil)
	srv.Stub.SetFault(kiwistub.Fault{Status: http.StatusTooManyRequests, RetryAfter: "1"})
	rec := &recorder{}
	gw := gatewayAt(t, srv, staticCreds{}, kiwi.WithSleep(rec.Sleep))
	_, err := gw.ListCollections(context.Background())
	if !errors.Is(err, domain.ErrSourceRateLimited) {
		t.Errorf("the call failed with %v, want the rate-limit error", err)
	}
	if n := len(srv.Stub.Requests()); n != 4 {
		t.Errorf("%d requests were sent, want 4", n)
	}
	if got := rec.Waits(); len(got) != 3 {
		t.Errorf("the waits are %v, want three", got)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/transport-and-retries.feature: A Retry-After longer than 60 seconds is refused without a retry
func TestRetryAfterOverSixtySecondsIsRefusedWithoutRetry(t *testing.T) {
	srv := serveKiwi(t, newStub(sampleDataset()), nil)
	srv.Stub.SetFault(kiwistub.Fault{Status: http.StatusTooManyRequests, RetryAfter: "61"})
	gw := gatewayAt(t, srv, staticCreds{})
	_, err := gw.ListCollections(context.Background())
	if !errors.Is(err, domain.ErrRetryAfterTooLong) {
		t.Errorf("the call failed with %v, want the too-long wait error", err)
	}
	if n := len(srv.Stub.Requests()); n != 1 {
		t.Errorf("%d requests were sent, want 1", n)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/transport-and-retries.feature: A 429 with a missing or unusable Retry-After is not retried
func TestRateLimitWithMissingOrUnusableRetryAfterIsNotRetried(t *testing.T) {
	rows := []struct{ name, value string }{
		{"no Retry-After", ""},
		{"an unusable value", "soon"},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			srv := serveKiwi(t, newStub(sampleDataset()), nil)
			srv.Stub.SetFault(kiwistub.Fault{Status: http.StatusTooManyRequests, RetryAfter: row.value})
			gw := gatewayAt(t, srv, staticCreds{})
			_, err := gw.ListCollections(context.Background())
			if !errors.Is(err, domain.ErrSourceRateLimited) {
				t.Errorf("the call failed with %v, want the rate-limit error", err)
			}
			if n := len(srv.Stub.Requests()); n != 1 {
				t.Errorf("%d requests were sent, want 1", n)
			}
		})
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/transport-and-retries.feature: A write is retried for a 429 reply and for no other status
func TestWriteIsRetriedForRateLimitAndNotForOtherStatus(t *testing.T) {
	srv := serveKiwi(t, newStub(sampleDataset()), nil)
	srv.Stub.SetMethodFaultFirst("TestPlan.add_case", kiwistub.Fault{Status: http.StatusTooManyRequests, RetryAfter: "1"})
	rec := &recorder{}
	gw := gatewayAt(t, srv, staticCreds{}, kiwi.WithSleep(rec.Sleep))
	must(t, addCase(gw))
	if n := len(srv.Stub.Requests()); n != 2 {
		t.Errorf("%d requests were sent for the 429 write, want 2", n)
	}
	if got := rec.Waits(); len(got) != 1 || got[0] != time.Second {
		t.Errorf("the waits are %v, want one wait of 1s", got)
	}

	other := serveKiwi(t, newStub(sampleDataset()), nil)
	other.Stub.SetMethodFault("TestPlan.add_case", kiwistub.Fault{Status: http.StatusServiceUnavailable})
	otherGW := gatewayAt(t, other, staticCreds{}, kiwi.WithSleep(rec.Sleep))
	err := addCase(otherGW)
	if !errors.Is(err, domain.ErrWriteOutcomeUnknown) {
		t.Errorf("the 503 write failed with %v, want the unknown-outcome error", err)
	}
	if n := len(other.Stub.Requests()); n != 1 {
		t.Errorf("a 503 write sent %d requests, want 1 with no retry", n)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/transport-and-retries.feature: A wait that is cancelled before the retry stops the call with that fault
func TestCancelledWaitStopsCallWithThatFault(t *testing.T) {
	srv := serveKiwi(t, newStub(sampleDataset()), nil)
	srv.Stub.SetFault(kiwistub.Fault{Status: http.StatusTooManyRequests, RetryAfter: "1"})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cancelWait := func(c context.Context, _ time.Duration) error {
		cancel()
		return c.Err()
	}
	gw := gatewayAt(t, srv, staticCreds{}, kiwi.WithSleep(cancelWait))
	_, err := gw.ListCollections(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("the call returned %v, want the cancellation fault", err)
	}
	if n := len(srv.Stub.Requests()); n != 1 {
		t.Errorf("%d requests were sent, want 1 with no retry", n)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/transport-and-retries.feature: A body that fails after a status is unreachable for a read
func TestBodyFailingAfterStatusIsUnreachableForRead(t *testing.T) {
	srv := serveKiwi(t, newStub(sampleDataset()), breakBody)
	gw := gatewayAt(t, srv, staticCreds{})
	_, err := gw.ListCollections(context.Background())
	if !errors.Is(err, domain.ErrUnreachable) {
		t.Errorf("the read failed with %v, want the unreachable error", err)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/transport-and-retries.feature: A TLS handshake that fails sends no request, so a write over it has a known outcome
func TestFailedHandshakeSendsNoRequestSoWriteOutcomeIsKnown(t *testing.T) {
	srv := serveKiwi(t, newStub(sampleDataset()), nil)
	gw := untrustingGateway(t, srv.Base)
	err := addCase(gw)
	if !errors.Is(err, domain.ErrUnreachable) {
		t.Errorf("the write failed with %v, want the unreachable error", err)
	}
	if n := srv.Log.count(); n != 0 {
		t.Errorf("%d requests reached the server, want none", n)
	}
}
