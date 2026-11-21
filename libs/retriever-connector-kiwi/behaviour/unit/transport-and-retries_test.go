package unit_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-connector-kiwi/kiwi"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-testkit/kiwistub"
)

// @scenario specs/libs/retriever-connector-kiwi/behaviours/transport-and-retries.feature: Every call is one POST to the endpoint with the Bearer key on the request
func TestEveryCallIsOnePOSTWithTheBearerKey(t *testing.T) {
	r := newRig(t, sampleDataset())
	for i := 0; i < 2; i++ {
		_, err := r.gw.ListCollections(context.Background())
		must(t, err)
	}
	reqs := r.stub.Requests()
	if len(reqs) != 2 {
		t.Fatalf("%d requests were sent, want 2", len(reqs))
	}
	for _, q := range reqs {
		if q.Verb != http.MethodPost || q.Path != kiwistub.Endpoint || q.Host != testHost || !q.AuthOK {
			t.Errorf("the request is %s %s to host %q with key accepted=%v", q.Verb, q.Path, q.Host, q.AuthOK)
		}
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/transport-and-retries.feature: A base address that is not HTTPS or has user information is refused as an invalid base
func TestBaseAddressThatIsNotHTTPSOrHasUserInfoIsRefused(t *testing.T) {
	rows := []struct {
		name, base string
	}{
		{"an address with plain HTTP", "http://kiwi.example.test"},
		{"an address with user information", "https://user@kiwi.example.test"},
		{"an address that does not parse", "https://%zz"},
		{"an address with no host", "https:///json-rpc/"},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			_, err := kiwi.NewWithBase(kiwiSource(t), row.base, &http.Client{}, staticCreds{})
			if err == nil {
				t.Fatalf("the base %q was accepted", row.base)
			}
			if strings.HasPrefix(row.base, "http://") && !errors.Is(err, domain.ErrHTTPSRequired) {
				t.Errorf("the plain HTTP base failed with %v, want the HTTPS-required error", err)
			}
		})
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/transport-and-retries.feature: A redirect in a reply is read as a refusal and the client makes one request
func TestRedirectIsReadAsRefusalAndOneRequestIsMade(t *testing.T) {
	r := newRig(t, sampleDataset())
	r.stub.SetFault(kiwistub.Fault{Redirect: true})
	_, err := r.gw.ListCollections(context.Background())
	if !errors.Is(err, domain.ErrRedirected) {
		t.Errorf("the call failed with %v, want the redirect error", err)
	}
	if n := len(r.stub.Requests()); n != 1 {
		t.Errorf("%d requests were sent, want 1", n)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/transport-and-retries.feature: A cookie in a reply is never sent back on the next request
func TestCookieIsNeverSentBackOnTheNextRequest(t *testing.T) {
	srv := &scripted{serve: func(req *http.Request, c call) (*http.Response, error) {
		return rpcReply(req, c, "[]", http.Header{"Set-Cookie": []string{"session=abc123; Path=/"}}), nil
	}}
	gw := hermeticGateway(t, srv, staticCreds{}, newFakeClock())
	for i := 0; i < 2; i++ {
		_, err := gw.ListCollections(context.Background())
		must(t, err)
	}
	reqs := srv.requests()
	if len(reqs) != 2 {
		t.Fatalf("%d requests were sent, want 2", len(reqs))
	}
	if reqs[1].cookie != "" {
		t.Errorf("the second request carries the Cookie header %q", reqs[1].cookie)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/transport-and-retries.feature: A read that times out is a timed-out read and a write that times out is an unknown outcome
func TestReadTimeoutIsTimedOutAndWriteTimeoutIsUnknownOutcome(t *testing.T) {
	srv := &scripted{serve: func(*http.Request, call) (*http.Response, error) { return nil, context.DeadlineExceeded }}
	gw := hermeticGateway(t, srv, staticCreds{}, newFakeClock())
	_, err := gw.ListCollections(context.Background())
	if !errors.Is(err, domain.ErrTimedOut) {
		t.Errorf("the read failed with %v, want the timed-out error", err)
	}
	_, err = gw.Write(context.Background(), domain.OpKiwiPlansAddCase, domain.NewArgs(map[string]any{"planId": "plan-2", "caseId": "case-12"}))
	if !errors.Is(err, domain.ErrWriteOutcomeUnknown) {
		t.Errorf("the write failed with %v, want the unknown-outcome error", err)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/transport-and-retries.feature: A read whose reply is over 10 MiB is refused as too large
func TestReadReplyOverTenMiBIsRefusedAsTooLarge(t *testing.T) {
	big := `{"jsonrpc":"2.0","id":1,"result":"` + strings.Repeat("x", 10<<20) + `"}`
	srv := &scripted{serve: func(req *http.Request, _ call) (*http.Response, error) {
		return httpResponse(req, http.StatusOK, nil, big), nil
	}}
	gw := hermeticGateway(t, srv, staticCreds{}, newFakeClock())
	_, err := gw.ListCollections(context.Background())
	if !errors.Is(err, domain.ErrReplyTooLarge) {
		t.Errorf("the read failed with %v, want the too-large error", err)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/transport-and-retries.feature: A write whose reply is over 10 MiB has an unknown outcome and is not retried
func TestWriteReplyOverTenMiBHasUnknownOutcomeAndIsNotRetried(t *testing.T) {
	big := `{"jsonrpc":"2.0","id":1,"result":"` + strings.Repeat("x", 10<<20) + `"}`
	srv := &scripted{serve: func(req *http.Request, _ call) (*http.Response, error) {
		return httpResponse(req, http.StatusOK, nil, big), nil
	}}
	gw := hermeticGateway(t, srv, staticCreds{}, newFakeClock())
	_, err := gw.Write(context.Background(), domain.OpKiwiPlansAddCase, domain.NewArgs(map[string]any{"planId": "plan-2", "caseId": "case-12"}))
	if !errors.Is(err, domain.ErrWriteOutcomeUnknown) {
		t.Errorf("the write failed with %v, want the unknown-outcome error", err)
	}
	if n := len(srv.requests()); n != 1 {
		t.Errorf("%d requests were sent, want 1", n)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/transport-and-retries.feature: Two calls are never in flight at the same time
func TestTwoCallsAreNeverInFlightAtTheSameTime(t *testing.T) {
	r := newRig(t, sampleDataset())
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := r.gw.ListCollections(context.Background())
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		must(t, err)
	}
	if peak := r.stub.PeakInFlight(); peak != 1 {
		t.Errorf("%d requests were in flight at once, want 1", peak)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/transport-and-retries.feature: The second request waits the full 100 ms pacing delay after the first
func TestSecondRequestWaitsTheFullPacingDelay(t *testing.T) {
	clock := newFakeClock()
	stub := kiwistub.New(sentinelKey, kiwistub.WithDataset(sampleDataset()), kiwistub.WithClock(clock.Now))
	gw := gateway(t, stub.RoundTripper(), staticCreds{}, kiwi.WithClock(clock.Now), kiwi.WithSleep(clock.Sleep))
	for i := 0; i < 2; i++ {
		_, err := gw.ListCollections(context.Background())
		must(t, err)
	}
	if got := clock.Sleeps(); len(got) != 1 || got[0] != 100*time.Millisecond {
		t.Errorf("the waits are %v, want one wait of 100ms", got)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/transport-and-retries.feature: A call that comes after the pacing gap has passed sends with no wait
func TestCallAfterPacingGapSendsWithNoWait(t *testing.T) {
	clock := newFakeClock()
	stub := kiwistub.New(sentinelKey, kiwistub.WithDataset(sampleDataset()), kiwistub.WithClock(clock.Now))
	gw := gateway(t, stub.RoundTripper(), staticCreds{}, kiwi.WithClock(clock.Now), kiwi.WithSleep(clock.Sleep))
	_, err := gw.ListCollections(context.Background())
	must(t, err)
	clock.Advance(200 * time.Millisecond)
	_, err = gw.ListCollections(context.Background())
	must(t, err)
	if got := clock.Sleeps(); len(got) != 0 {
		t.Errorf("the second request waited %v, want no wait", got)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/transport-and-retries.feature: A rate-limited call with a short Retry-After is sent again after the wait
func TestRateLimitedCallWithShortRetryAfterIsSentAgain(t *testing.T) {
	r := newRig(t, sampleDataset())
	r.stub.SetFaultFirst(kiwistub.Fault{Status: http.StatusTooManyRequests, RetryAfter: "1"})
	_, err := r.gw.ListCollections(context.Background())
	must(t, err)
	if n := len(r.stub.Requests()); n != 2 {
		t.Errorf("%d requests were sent, want 2", n)
	}
	if got := r.clock.Sleeps(); len(got) != 1 || got[0] != time.Second {
		t.Errorf("the waits are %v, want one wait of 1s", got)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/transport-and-retries.feature: A source that keeps answering 429 stops after one request and three retries with the rate-limit error
func TestPersistentRateLimitStopsAfterThreeRetries(t *testing.T) {
	r := newRig(t, sampleDataset())
	r.stub.SetFault(kiwistub.Fault{Status: http.StatusTooManyRequests, RetryAfter: "1"})
	_, err := r.gw.ListCollections(context.Background())
	if !errors.Is(err, domain.ErrSourceRateLimited) {
		t.Errorf("the call failed with %v, want the rate-limit error", err)
	}
	if n := len(r.stub.Requests()); n != 4 {
		t.Errorf("%d requests were sent, want 4", n)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/transport-and-retries.feature: A Retry-After longer than 60 seconds is refused without a retry
func TestRetryAfterOverSixtySecondsIsRefusedWithoutRetry(t *testing.T) {
	r := newRig(t, sampleDataset())
	r.stub.SetFault(kiwistub.Fault{Status: http.StatusTooManyRequests, RetryAfter: "61"})
	_, err := r.gw.ListCollections(context.Background())
	if !errors.Is(err, domain.ErrRetryAfterTooLong) {
		t.Errorf("the call failed with %v, want the too-long wait error", err)
	}
	if n := len(r.stub.Requests()); n != 1 {
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
			r := newRig(t, sampleDataset())
			r.stub.SetFault(kiwistub.Fault{Status: http.StatusTooManyRequests, RetryAfter: row.value})
			_, err := r.gw.ListCollections(context.Background())
			if !errors.Is(err, domain.ErrSourceRateLimited) {
				t.Errorf("the call failed with %v, want the rate-limit error", err)
			}
			if n := len(r.stub.Requests()); n != 1 {
				t.Errorf("%d requests were sent, want 1", n)
			}
		})
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/transport-and-retries.feature: A write is retried for a 429 reply and for no other status
func TestWriteIsRetriedForRateLimitAndSucceeds(t *testing.T) {
	r := newRig(t, sampleDataset())
	r.stub.SetMethodFaultFirst("TestPlan.add_case", kiwistub.Fault{Status: http.StatusTooManyRequests, RetryAfter: "1"})
	_, err := r.gw.Write(context.Background(), domain.OpKiwiPlansAddCase, domain.NewArgs(map[string]any{"planId": "plan-2", "caseId": "case-12"}))
	must(t, err)
	if n := len(r.stub.Requests()); n != 2 {
		t.Errorf("%d requests were sent, want 2", n)
	}
	if got := r.clock.Sleeps(); len(got) != 1 || got[0] != time.Second {
		t.Errorf("the waits are %v, want one wait of 1s", got)
	}

	other := newRig(t, sampleDataset())
	other.stub.SetMethodFault("TestPlan.add_case", kiwistub.Fault{Status: http.StatusServiceUnavailable})
	_, err = other.gw.Write(context.Background(), domain.OpKiwiPlansAddCase, domain.NewArgs(map[string]any{"planId": "plan-2", "caseId": "case-12"}))
	if !errors.Is(err, domain.ErrWriteOutcomeUnknown) {
		t.Errorf("the 503 write failed with %v, want the unknown-outcome error", err)
	}
	if n := len(other.stub.Requests()); n != 1 {
		t.Errorf("a 503 write sent %d requests, want 1 with no retry", n)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/transport-and-retries.feature: A wait that is cancelled before the retry stops the call with that fault
func TestCancelledWaitStopsCallWithThatFault(t *testing.T) {
	clock := newFakeClock()
	stub := kiwistub.New(sentinelKey, kiwistub.WithDataset(sampleDataset()), kiwistub.WithClock(clock.Now))
	stub.SetFault(kiwistub.Fault{Status: http.StatusTooManyRequests, RetryAfter: "1"})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cancelWait := func(c context.Context, _ time.Duration) error {
		cancel()
		return c.Err()
	}
	gw := gateway(t, stub.RoundTripper(), staticCreds{}, kiwi.WithMinDelay(0), kiwi.WithClock(clock.Now), kiwi.WithSleep(cancelWait))
	_, err := gw.ListCollections(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("the call returned %v, want the cancellation fault", err)
	}
	if n := len(stub.Requests()); n != 1 {
		t.Errorf("%d requests were sent, want 1 with no retry", n)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/transport-and-retries.feature: A body that fails after a status is unreachable for a read
func TestBodyFailingAfterStatusIsUnreachableForRead(t *testing.T) {
	srv := &scripted{serve: func(req *http.Request, _ call) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: &brokenBody{}, Request: req}, nil
	}}
	gw := hermeticGateway(t, srv, staticCreds{}, newFakeClock())
	_, err := gw.ListCollections(context.Background())
	if !errors.Is(err, domain.ErrUnreachable) {
		t.Errorf("the read failed with %v, want the unreachable error", err)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/transport-and-retries.feature: A TLS handshake that fails sends no request, so a write over it has a known outcome
func TestFailedHandshakeSendsNoRequestSoWriteOutcomeIsKnown(t *testing.T) {
	stub := kiwistub.New(sentinelKey, kiwistub.WithDataset(sampleDataset()))
	gw := hermeticGateway(t, handshakeRT{}, staticCreds{}, newFakeClock())
	_, err := gw.Write(context.Background(), domain.OpKiwiPlansAddCase, domain.NewArgs(map[string]any{"planId": "plan-2", "caseId": "case-12"}))
	if !errors.Is(err, domain.ErrUnreachable) {
		t.Errorf("the write failed with %v, want the unreachable error", err)
	}
	if n := len(stub.Requests()); n != 0 {
		t.Errorf("%d requests reached the gateway, want none", n)
	}
}
