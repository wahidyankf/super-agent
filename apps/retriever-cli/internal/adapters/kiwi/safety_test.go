package kiwi

import (
	"context"
	"errors"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/ports"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/tests/support/kiwistub"
)

// addCaseArgs are the arguments of one write that a safety test can repeat.
func addCaseArgs() domain.Args {
	return domain.NewArgs(map[string]any{"planId": "plan-2", "caseId": "case-13"})
}

// TestEveryRequestIsAPostToTheEndpointWithTheBearerKey checks the transport rules on the wire: one POST path, the endpoint
// host, and a right Bearer key on each request.
func TestEveryRequestIsAPostToTheEndpointWithTheBearerKey(t *testing.T) {
	r := newRig(t, sampleDataset())
	if _, err := r.gw.ListCollections(context.Background()); err != nil {
		t.Fatalf("ListCollections: %v", err)
	}
	reqs := r.stub.Requests()
	if len(reqs) != 1 {
		t.Fatalf("requests = %d, want 1", len(reqs))
	}
	got := reqs[0]
	if got.Verb != http.MethodPost || got.Path != "/json-rpc/" || got.Host != testHost || !got.AuthOK {
		t.Fatalf("request = %+v, want a POST to /json-rpc/ on %s with the Bearer key", got, testHost)
	}
	if r.creds.count() != 1 {
		t.Fatalf("the key was read %d times, want once per request", r.creds.count())
	}
}

// TestRedirectIsNotFollowed checks that a 3xx reply is read as a refusal, and that the client makes one request only.
func TestRedirectIsNotFollowed(t *testing.T) {
	r := newRig(t, sampleDataset())
	r.stub.SetFault(kiwistub.Fault{Redirect: true})
	_, err := r.gw.ListCollections(context.Background())
	if !errors.Is(err, domain.ErrRedirected) {
		t.Fatalf("error = %v, want ErrRedirected", err)
	}
	if r.stub.Calls("Product.filter") != 1 {
		t.Fatalf("a redirect was followed")
	}
}

// TestBaseMustBeHTTPSAndHaveNoUserInformation checks the base checks of NewWithBase.
func TestBaseMustBeHTTPSAndHaveNoUserInformation(t *testing.T) {
	src := testSourceAt(t)
	if _, err := NewWithBase(src, "http://127.0.0.1:9", &http.Client{}, &staticCreds{}); !errors.Is(err, domain.ErrHTTPSRequired) {
		t.Fatalf("plain HTTP base = %v, want ErrHTTPSRequired", err)
	}
	if _, err := NewWithBase(src, "https://reader@127.0.0.1:9", &http.Client{}, &staticCreds{}); err == nil {
		t.Fatalf("a base with user information was accepted")
	}
}

// TestTransportHasNoProxyAndTheEndpointTLSName checks the production transport: no proxy, and the TLS name of the endpoint.
func TestTransportHasNoProxyAndTheEndpointTLSName(t *testing.T) {
	gw := New(testSourceAt(t), &staticCreds{})
	tr, ok := gw.client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport is %T, want *http.Transport", gw.client.Transport)
	}
	if tr.Proxy != nil {
		t.Fatalf("the transport reads a proxy from the environment")
	}
	if tr.TLSClientConfig == nil || tr.TLSClientConfig.ServerName != testHost {
		t.Fatalf("TLS server name = %v, want %s", tr.TLSClientConfig, testHost)
	}
	if gw.client.CheckRedirect == nil {
		t.Fatalf("the redirect policy is not replaced")
	}
	if gw.timeout != 30*time.Second {
		t.Fatalf("timeout = %v, want 30s", gw.timeout)
	}
}

// TestNewWithBaseClearsTheCallersProxyAndJarOnACopy checks that the gateway changes a copy of the caller's client: the
// copy has no proxy and no cookie jar, and the caller's own client keeps its proxy, jar, and redirect policy.
func TestNewWithBaseClearsTheCallersProxyAndJarOnACopy(t *testing.T) {
	proxyURL, err := url.Parse("http://proxy.invalid:3128")
	if err != nil {
		t.Fatalf("url.Parse: %v", err)
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar.New: %v", err)
	}
	callerTransport := &http.Transport{Proxy: http.ProxyURL(proxyURL)}
	caller := &http.Client{Transport: callerTransport, Jar: jar}

	gw, err := NewWithBase(testSourceAt(t), testBase, caller, &staticCreds{})
	if err != nil {
		t.Fatalf("NewWithBase: %v", err)
	}
	gatewayTransport, ok := gw.client.Transport.(*http.Transport)
	if !ok || gatewayTransport.Proxy != nil {
		t.Fatalf("the gateway transport still reads a proxy")
	}
	if gw.client.Jar != nil {
		t.Fatalf("the gateway client keeps the caller's cookie jar")
	}
	if caller.Jar != jar || callerTransport.Proxy == nil || caller.CheckRedirect != nil {
		t.Fatalf("the caller's client was changed; the gateway must work on a copy")
	}
}

// cookieEcho wraps a round tripper: it answers every request with a Set-Cookie header and records the Cookie header that
// each request carried.
type cookieEcho struct {
	inner   http.RoundTripper
	mu      sync.Mutex
	cookies []string
}

func (c *cookieEcho) RoundTrip(req *http.Request) (*http.Response, error) {
	c.mu.Lock()
	c.cookies = append(c.cookies, req.Header.Get("Cookie"))
	c.mu.Unlock()
	resp, err := c.inner.RoundTrip(req)
	if err == nil {
		resp.Header.Add("Set-Cookie", "session=abc; Path=/")
	}
	return resp, err
}

// TestACookieIsNeverSentBack checks that a Set-Cookie reply does not add a Cookie header to the next request, even when the
// caller's client holds a cookie jar.
func TestACookieIsNeverSentBack(t *testing.T) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar.New: %v", err)
	}
	stub := kiwistub.New(sentinelKey, kiwistub.WithDataset(sampleDataset()))
	echo := &cookieEcho{inner: stub.RoundTripper()}
	caller := &http.Client{Transport: echo, Jar: jar}
	gw, err := NewWithBase(testSourceAt(t), testBase, caller, &staticCreds{}, WithMinDelay(0))
	if err != nil {
		t.Fatalf("NewWithBase: %v", err)
	}
	for i := 0; i < 2; i++ {
		if _, err := gw.ListCollections(context.Background()); err != nil {
			t.Fatalf("request %d: %v", i+1, err)
		}
	}
	if len(echo.cookies) != 2 || echo.cookies[1] != "" {
		t.Fatalf("Cookie headers = %q, want none on the second request; the client keeps no cookies", echo.cookies)
	}
}

// TestTimeoutIsATimedOutReadAndAnUnknownWrite checks the timeout: a read times out, and a write has an unknown outcome.
func TestTimeoutIsATimedOutReadAndAnUnknownWrite(t *testing.T) {
	r := newRig(t, sampleDataset(), WithTimeout(20*time.Millisecond))
	r.stub.SetFault(kiwistub.Fault{Never: true})
	if _, err := r.gw.ListCollections(context.Background()); !errors.Is(err, domain.ErrTimedOut) {
		t.Fatalf("read timeout = %v, want ErrTimedOut", err)
	}
	if _, err := r.gw.Write(context.Background(), domain.OpKiwiPlansAddCase, addCaseArgs()); !errors.Is(err, domain.ErrWriteOutcomeUnknown) {
		t.Fatalf("write timeout = %v, want ErrWriteOutcomeUnknown", err)
	}
	if r.stub.Calls("TestPlan.add_case") != 1 {
		t.Fatalf("a timed-out write was sent %d times, want 1", r.stub.Calls("TestPlan.add_case"))
	}
}

// TestOversizeReplyIsRefusedForARead checks that a reply over the 10 MiB cap is refused as too large.
func TestOversizeReplyIsRefusedForARead(t *testing.T) {
	r := newRig(t, sampleDataset())
	r.stub.SetFault(kiwistub.Fault{Oversize: true})
	if _, err := r.gw.ListCollections(context.Background()); !errors.Is(err, domain.ErrReplyTooLarge) {
		t.Fatalf("oversize read = %v, want ErrReplyTooLarge", err)
	}
}

// TestOversizeReplyIsUnknownForAWrite checks that an oversize reply to a write is an unknown outcome, and is not retried.
func TestOversizeReplyIsUnknownForAWrite(t *testing.T) {
	r := newRig(t, sampleDataset())
	r.stub.SetFault(kiwistub.Fault{Oversize: true})
	if _, err := r.gw.Write(context.Background(), domain.OpKiwiPlansAddCase, addCaseArgs()); !errors.Is(err, domain.ErrWriteOutcomeUnknown) {
		t.Fatalf("oversize write = %v, want ErrWriteOutcomeUnknown", err)
	}
	if r.stub.Calls("TestPlan.add_case") != 1 {
		t.Fatalf("an oversize write was retried")
	}
}

// TestPacingWaitsOneDelayBetweenRequests checks the 100 ms rule on a fake clock: the second request waits the full delay.
func TestPacingWaitsOneDelayBetweenRequests(t *testing.T) {
	r := newRig(t, sampleDataset(), WithMinDelay(100*time.Millisecond))
	if _, err := r.gw.ListCollections(context.Background()); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if _, err := r.gw.ListCollections(context.Background()); err != nil {
		t.Fatalf("second call: %v", err)
	}
	if got := r.clock.recordedSleeps(); len(got) != 1 || got[0] != 100*time.Millisecond {
		t.Fatalf("pacing sleeps = %v, want one 100ms wait", got)
	}
}

// TestOneRequestIsInFlightAtATime checks that concurrent calls go out one at a time.
func TestOneRequestIsInFlightAtATime(t *testing.T) {
	r := newRig(t, sampleDataset())
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = r.gw.ListCollections(context.Background())
		}()
	}
	wg.Wait()
	if r.stub.PeakInFlight() != 1 {
		t.Fatalf("peak in flight = %d, want 1", r.stub.PeakInFlight())
	}
}

// TestRateLimitIsRetriedAfterTheWait checks a 429 with a short Retry-After: the request is sent again after the wait.
func TestRateLimitIsRetriedAfterTheWait(t *testing.T) {
	r := newRig(t, sampleDataset())
	r.stub.SetFaultFirst(kiwistub.Fault{Status: http.StatusTooManyRequests, RetryAfter: "1"})
	if _, err := r.gw.ListCollections(context.Background()); err != nil {
		t.Fatalf("ListCollections after one 429: %v", err)
	}
	if got := r.clock.recordedSleeps(); len(got) != 1 || got[0] != time.Second {
		t.Fatalf("sleeps = %v, want one 1s wait", got)
	}
	if r.stub.Calls("Product.filter") != 2 {
		t.Fatalf("calls = %d, want 2", r.stub.Calls("Product.filter"))
	}
}

// TestRateLimitGivesUpAfterThreeRetries checks the retry cap: one request and three retries, then ErrSourceRateLimited.
func TestRateLimitGivesUpAfterThreeRetries(t *testing.T) {
	r := newRig(t, sampleDataset())
	r.stub.SetFault(kiwistub.Fault{Status: http.StatusTooManyRequests, RetryAfter: "1"})
	if _, err := r.gw.ListCollections(context.Background()); !errors.Is(err, domain.ErrSourceRateLimited) {
		t.Fatalf("error = %v, want ErrSourceRateLimited", err)
	}
	if r.stub.Calls("Product.filter") != 4 {
		t.Fatalf("calls = %d, want 4 (one request and three retries)", r.stub.Calls("Product.filter"))
	}
}

// TestRetryAfterOverSixtySecondsIsNotWaited checks that a long wait is refused without a retry.
func TestRetryAfterOverSixtySecondsIsNotWaited(t *testing.T) {
	r := newRig(t, sampleDataset())
	r.stub.SetFault(kiwistub.Fault{Status: http.StatusTooManyRequests, RetryAfter: "61"})
	if _, err := r.gw.ListCollections(context.Background()); !errors.Is(err, domain.ErrRetryAfterTooLong) {
		t.Fatalf("error = %v, want ErrRetryAfterTooLong", err)
	}
	if r.stub.Calls("Product.filter") != 1 || len(r.clock.recordedSleeps()) != 0 {
		t.Fatalf("a long Retry-After was retried or waited")
	}
}

// TestAWriteIsRetriedOnlyForA429 checks that a write is retried for a 429 and for nothing else.
func TestAWriteIsRetriedOnlyForA429(t *testing.T) {
	r := newRig(t, sampleDataset())
	r.stub.SetFaultFirst(kiwistub.Fault{Status: http.StatusTooManyRequests, RetryAfter: "1"})
	if _, err := r.gw.Write(context.Background(), domain.OpKiwiPlansAddCase, addCaseArgs()); err != nil {
		t.Fatalf("write after one 429: %v", err)
	}
	if r.stub.Calls("TestPlan.add_case") != 2 {
		t.Fatalf("a 429 write was sent %d times, want 2", r.stub.Calls("TestPlan.add_case"))
	}
	r.stub.SetFault(kiwistub.Fault{Status: http.StatusInternalServerError})
	if _, err := r.gw.Write(context.Background(), domain.OpKiwiPlansAddCase, addCaseArgs()); !errors.Is(err, domain.ErrWriteOutcomeUnknown) {
		t.Fatalf("write after 500 = %v, want ErrWriteOutcomeUnknown", err)
	}
	if r.stub.Calls("TestPlan.add_case") != 3 {
		t.Fatalf("a 500 write was retried")
	}
}

// TestStatusFaultsMapToTheirSentinels checks the status rules, and that no error carries the key, even when the body
// echoes it.
func TestStatusFaultsMapToTheirSentinels(t *testing.T) {
	cases := []struct {
		name  string
		fault kiwistub.Fault
		want  error
	}{
		{"401", kiwistub.Fault{Status: http.StatusUnauthorized, EchoKey: true}, domain.ErrKeyRejected},
		{"403", kiwistub.Fault{Status: http.StatusForbidden, EchoKey: true}, domain.ErrAccessDenied},
		{"404", kiwistub.Fault{Status: http.StatusNotFound, EchoKey: true}, domain.ErrUnexpectedResponse},
		{"405", kiwistub.Fault{Status: http.StatusMethodNotAllowed}, domain.ErrUnexpectedResponse},
		{"500", kiwistub.Fault{Status: http.StatusInternalServerError, EchoKey: true}, domain.ErrUpstreamFailure},
		{"503", kiwistub.Fault{Status: http.StatusServiceUnavailable}, domain.ErrUpstreamFailure},
		{"not JSON", kiwistub.Fault{NotJSON: true, EchoKey: true}, domain.ErrUnexpectedResponse},
		{"wrong id", kiwistub.Fault{WrongID: true}, domain.ErrUnexpectedResponse},
		{"neither result nor error", kiwistub.Fault{NoResultError: true}, domain.ErrUnexpectedResponse},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newRig(t, sampleDataset())
			r.stub.SetFault(tc.fault)
			_, err := r.gw.ListCollections(context.Background())
			if !errors.Is(err, tc.want) {
				t.Fatalf("error = %v, want %v", err, tc.want)
			}
			assertNoKey(t, err)
		})
	}
}

// TestMissingKeyIsNamedBeforeAnyRequest checks that a missing key is reported with its variable, and nothing is sent.
func TestMissingKeyIsNamedBeforeAnyRequest(t *testing.T) {
	stub := kiwistub.New(sentinelKey, kiwistub.WithDataset(sampleDataset()))
	client := &http.Client{Transport: stub.RoundTripper()}
	var creds ports.Credentials = missingCreds{}
	gw, err := NewWithBase(testSourceAt(t), testBase, client, creds, WithMinDelay(0), WithSleep(noSleep))
	if err != nil {
		t.Fatalf("NewWithBase: %v", err)
	}
	_, err = gw.ListCollections(context.Background())
	var missing domain.MissingKeyError
	if !errors.As(err, &missing) || missing.Variable != keyVariable {
		t.Fatalf("error = %v, want a MissingKeyError naming %s", err, keyVariable)
	}
	if len(stub.Requests()) != 0 {
		t.Fatalf("a request was sent without a key")
	}
}

// TestAMethodOutsideTheAllowlistNeverReachesTheNetwork checks the closed method table: a method that is not on the
// allowlist is refused before any request is sent.
func TestAMethodOutsideTheAllowlistNeverReachesTheNetwork(t *testing.T) {
	r := newRig(t, sampleDataset())
	_, err := r.gw.call(context.Background(), "TestCase.remove", nil)
	if !errors.Is(err, errMethodNotAllowed) {
		t.Fatalf("call outside the allowlist = %v, want errMethodNotAllowed", err)
	}
	if len(r.stub.Requests()) != 0 {
		t.Fatalf("a request reached the network for a method outside the allowlist")
	}
}

// TestEveryFailureNamesKiwi checks that the boundary wraps each failure in the Kiwi kind.
func TestEveryFailureNamesKiwi(t *testing.T) {
	r := newRig(t, sampleDataset())
	r.stub.SetFault(kiwistub.Fault{Status: http.StatusUnauthorized})
	_, err := r.gw.ListCollections(context.Background())
	var kindErr domain.KindError
	if !errors.As(err, &kindErr) || kindErr.Kind != domain.KindKiwi {
		t.Fatalf("error = %#v, want a KindError of kiwi", err)
	}
	if strings.Contains(err.Error(), "Outline") {
		t.Fatalf("error text %q names Outline", err.Error())
	}
}

// assertNoKey fails the test when the error text holds the sentinel key.
func assertNoKey(t *testing.T, err error) {
	t.Helper()
	if err != nil && strings.Contains(err.Error(), sentinelKey) {
		t.Fatalf("error %q carries the key", err.Error())
	}
}

// noSleep is a wait that returns at once, for the tests that do not test pacing.
func noSleep(ctx context.Context, _ time.Duration) error {
	return ctx.Err()
}
