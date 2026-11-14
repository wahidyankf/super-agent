package jira

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// probed is what the client sent in one request, as the scripted transport saw it.
type probed struct {
	method  string
	host    string
	path    string
	query   string
	body    string
	reqHost string
	auth    string
	cookie  string
}

// probe is a round tripper whose reply the test scripts. It numbers each call from 1, records what the client sent, and counts
// the calls in flight, so a test can prove one request at a time.
type probe struct {
	mu          sync.Mutex
	script      func(n int, req *http.Request) (*http.Response, error)
	seen        []probed
	inFlight    int
	maxInFlight int
}

// RoundTrip records the request, then answers with the script. A nil script answers 200 with an empty object.
func (p *probe) RoundTrip(req *http.Request) (*http.Response, error) {
	var body []byte
	if req.Body != nil {
		body, _ = io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewReader(body))
	}
	p.mu.Lock()
	p.inFlight++
	if p.inFlight > p.maxInFlight {
		p.maxInFlight = p.inFlight
	}
	n := len(p.seen) + 1
	p.seen = append(p.seen, probed{
		method:  req.Method,
		host:    req.URL.Host,
		path:    req.URL.Path,
		query:   req.URL.RawQuery,
		body:    string(body),
		reqHost: req.Host,
		auth:    req.Header.Get("Authorization"),
		cookie:  req.Header.Get("Cookie"),
	})
	script := p.script
	p.mu.Unlock()
	defer func() {
		p.mu.Lock()
		p.inFlight--
		p.mu.Unlock()
	}()
	if script == nil {
		return reply(http.StatusOK, "{}"), nil
	}
	resp, err := script(n, req)
	if resp != nil {
		resp.Request = req
	}
	return resp, err
}

// calls returns the number of requests the client sent.
func (p *probe) calls() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.seen)
}

// requests returns what the client sent, in order.
func (p *probe) requests() []probed {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]probed(nil), p.seen...)
}

// peakInFlight returns the most requests that were in flight at one time.
func (p *probe) peakInFlight() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.maxInFlight
}

// reply returns a response with the status, the body, and the headers given as name and value pairs.
func reply(status int, body string, headers ...string) *http.Response {
	h := http.Header{}
	for i := 0; i+1 < len(headers); i += 2 {
		h.Add(headers[i], headers[i+1])
	}
	return &http.Response{
		StatusCode:    status,
		Status:        http.StatusText(status),
		Header:        h,
		Body:          io.NopCloser(strings.NewReader(body)),
		ContentLength: int64(len(body)),
	}
}

// probeGateway builds a gateway whose transport is the probe, with no pacing unless a test passes WithMinDelay.
func probeGateway(t *testing.T, p *probe, clock *fakeClock, secrets Secrets, opts ...Option) *Gateway {
	t.Helper()
	all := append([]Option{WithMinDelay(0), WithClock(clock.Now), WithSleep(clock.Sleep)}, opts...)
	gw, err := NewWithBase(testSource(t), testBase, &http.Client{Transport: p}, secrets, all...)
	if err != nil {
		t.Fatalf("NewWithBase: %v", err)
	}
	return gw
}

// myself is the call the client tests use for one request: the signed-in account, which needs no arguments.
func myself(gw *Gateway) error {
	var out map[string]any
	return gw.call(context.Background(), epMyself, "", nil, nil, &out)
}

// TestTheHostHeaderAndTheRequestHostAreTheEndpointHost proves a request goes to the connection base, and its Host header
// names the endpoint host, so the server sees the right site.
func TestTheHostHeaderAndTheRequestHostAreTheEndpointHost(t *testing.T) {
	p := &probe{}
	gw := probeGateway(t, p, newFakeClock(), newSecrets())
	if err := myself(gw); err != nil {
		t.Fatalf("myself: %v", err)
	}
	got := p.requests()[0]
	if got.host != "jira-stub.invalid" || got.reqHost != testHost || got.method != http.MethodGet || got.path != "/rest/api/3/myself" {
		t.Fatalf("request %+v, want the base host with the endpoint Host header", got)
	}
}

// TestRequestsRunOneAtATime proves the lock holds: several callers at once never have two requests in flight.
func TestRequestsRunOneAtATime(t *testing.T) {
	p := &probe{script: func(int, *http.Request) (*http.Response, error) {
		// A yield, not a wait: it lets another caller reach the lock while this request is in flight, and it reads no clock.
		for i := 0; i < 8; i++ {
			runtime.Gosched()
		}
		return reply(http.StatusOK, "{}"), nil
	}}
	gw := probeGateway(t, p, newFakeClock(), newSecrets())
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = myself(gw)
		}()
	}
	wg.Wait()
	if peak := p.peakInFlight(); peak != 1 {
		t.Fatalf("peak in flight = %d, want 1", peak)
	}
	if p.calls() != 8 {
		t.Fatalf("calls = %d, want 8", p.calls())
	}
}

// TestRequestsArePacedByTheMinimumDelay proves the least time between one request's end and the next request's start is
// the configured delay: three requests make two waits of 100 ms, measured on the fake clock.
func TestRequestsArePacedByTheMinimumDelay(t *testing.T) {
	clock := newFakeClock()
	p := &probe{}
	gw := probeGateway(t, p, clock, newSecrets(), WithMinDelay(100*time.Millisecond))
	for i := 0; i < 3; i++ {
		if err := myself(gw); err != nil {
			t.Fatalf("request %d: %v", i+1, err)
		}
	}
	waits := clock.recordedSleeps()
	if len(waits) != 2 || waits[0] != 100*time.Millisecond || waits[1] != 100*time.Millisecond {
		t.Fatalf("waits = %v, want two waits of 100ms", waits)
	}
}

// TestA429IsRetriedAfterItsWaitAtMostThreeTimes proves a 429 waits for Retry-After and is sent again, three times at most.
func TestA429IsRetriedAfterItsWaitAtMostThreeTimes(t *testing.T) {
	clock := newFakeClock()
	p := &probe{script: func(int, *http.Request) (*http.Response, error) {
		return reply(http.StatusTooManyRequests, "{}", "Retry-After", "2"), nil
	}}
	gw := probeGateway(t, p, clock, newSecrets())
	if err := myself(gw); !errors.Is(err, domain.ErrSourceRateLimited) {
		t.Fatalf("err = %v, want ErrSourceRateLimited after the retries", err)
	}
	if p.calls() != 4 {
		t.Fatalf("calls = %d, want 4 (one try and three retries)", p.calls())
	}
	waits := clock.recordedSleeps()
	if len(waits) != 3 || waits[0] != 2*time.Second {
		t.Fatalf("waits = %v, want three waits of 2s", waits)
	}
}

// TestA429WithRetryAfterOfExactlySixtySecondsIsWaitedOut proves the limit is inclusive: 60 seconds is waited, then the
// request succeeds.
func TestA429WithRetryAfterOfExactlySixtySecondsIsWaitedOut(t *testing.T) {
	clock := newFakeClock()
	p := &probe{script: func(n int, _ *http.Request) (*http.Response, error) {
		if n == 1 {
			return reply(http.StatusTooManyRequests, "{}", "Retry-After", "60"), nil
		}
		return reply(http.StatusOK, "{}"), nil
	}}
	gw := probeGateway(t, p, clock, newSecrets())
	if err := myself(gw); err != nil {
		t.Fatalf("myself: %v", err)
	}
	if waits := clock.recordedSleeps(); len(waits) != 1 || waits[0] != 60*time.Second {
		t.Fatalf("waits = %v, want one wait of 60s", waits)
	}
}

// TestA429WithRetryAfterOverSixtySecondsIsNotWaitedOut proves a longer wait fails at once, with no retry.
func TestA429WithRetryAfterOverSixtySecondsIsNotWaitedOut(t *testing.T) {
	clock := newFakeClock()
	p := &probe{script: func(int, *http.Request) (*http.Response, error) {
		return reply(http.StatusTooManyRequests, "{}", "Retry-After", "61"), nil
	}}
	gw := probeGateway(t, p, clock, newSecrets())
	if err := myself(gw); !errors.Is(err, domain.ErrRetryAfterTooLong) {
		t.Fatalf("err = %v, want ErrRetryAfterTooLong", err)
	}
	if p.calls() != 1 || len(clock.recordedSleeps()) != 0 {
		t.Fatalf("calls %d, waits %v; want one call and no wait", p.calls(), clock.recordedSleeps())
	}
}

// TestAnUnusableRetryAfterFailsWithoutRetry proves a missing, negative, or non-numeric Retry-After is not waited for.
func TestAnUnusableRetryAfterFailsWithoutRetry(t *testing.T) {
	for _, value := range []string{"", "-1", "soon", "1.5"} {
		p := &probe{script: func(int, *http.Request) (*http.Response, error) {
			return reply(http.StatusTooManyRequests, "{}", "Retry-After", value), nil
		}}
		gw := probeGateway(t, p, newFakeClock(), newSecrets())
		if err := myself(gw); !errors.Is(err, domain.ErrSourceRateLimited) {
			t.Fatalf("Retry-After %q: err = %v, want ErrSourceRateLimited", value, err)
		}
		if p.calls() != 1 {
			t.Fatalf("Retry-After %q: calls = %d, want 1", value, p.calls())
		}
	}
}

// TestA401And403AreNeverRetried proves a refused token or a denied account is final: one request, and no second attempt.
func TestA401And403AreNeverRetried(t *testing.T) {
	cases := []struct {
		status int
		want   error
	}{
		{http.StatusUnauthorized, domain.ErrKeyRejected},
		{http.StatusForbidden, domain.ErrAccessDenied},
	}
	for _, c := range cases {
		p := &probe{script: func(int, *http.Request) (*http.Response, error) {
			return reply(c.status, `{"errorMessages":["no"]}`), nil
		}}
		gw := probeGateway(t, p, newFakeClock(), newSecrets())
		if err := myself(gw); !errors.Is(err, c.want) {
			t.Fatalf("status %d: err = %v, want %v", c.status, err, c.want)
		}
		if p.calls() != 1 {
			t.Fatalf("status %d: calls = %d, want 1 (never retried)", c.status, p.calls())
		}
	}
}

// TestARedirectIsReadAndNeverFollowed proves a 3xx reply is refused, and the client does not send a second request to the
// Location, so a credential never goes to where a redirect points.
func TestARedirectIsReadAndNeverFollowed(t *testing.T) {
	p := &probe{script: func(int, *http.Request) (*http.Response, error) {
		return reply(http.StatusFound, "", "Location", "https://evil.invalid/login"), nil
	}}
	gw := probeGateway(t, p, newFakeClock(), newSecrets())
	if err := myself(gw); !errors.Is(err, domain.ErrRedirected) {
		t.Fatalf("err = %v, want ErrRedirected", err)
	}
	if p.calls() != 1 {
		t.Fatalf("calls = %d, want 1: the redirect target must not be requested", p.calls())
	}
}

// TestACookieIsNeverSentBack proves a Set-Cookie reply does not add a Cookie header to the next request.
func TestACookieIsNeverSentBack(t *testing.T) {
	p := &probe{script: func(n int, _ *http.Request) (*http.Response, error) {
		if n == 1 {
			return reply(http.StatusOK, "{}", "Set-Cookie", "atlassian.xsrf=abc; Path=/"), nil
		}
		return reply(http.StatusOK, "{}"), nil
	}}
	gw := probeGateway(t, p, newFakeClock(), newSecrets())
	for i := 0; i < 2; i++ {
		if err := myself(gw); err != nil {
			t.Fatalf("request %d: %v", i+1, err)
		}
	}
	if got := p.requests()[1].cookie; got != "" {
		t.Fatalf("second request sent Cookie %q; the client keeps no cookies", got)
	}
}

// TestATimeoutIsTimedOut proves a request that gets no reply within the limit is ErrTimedOut.
func TestATimeoutIsTimedOut(t *testing.T) {
	p := &probe{script: func(_ int, req *http.Request) (*http.Response, error) {
		<-req.Context().Done()
		return nil, req.Context().Err()
	}}
	gw := probeGateway(t, p, newFakeClock(), newSecrets(), WithTimeout(20*time.Millisecond))
	if err := myself(gw); !errors.Is(err, domain.ErrTimedOut) {
		t.Fatalf("err = %v, want ErrTimedOut", err)
	}
}

// TestAReplyOverTheCapIsTooLarge proves a body over 10 MiB is refused, not read to the end.
func TestAReplyOverTheCapIsTooLarge(t *testing.T) {
	p := &probe{script: func(int, *http.Request) (*http.Response, error) {
		return reply(http.StatusOK, strings.Repeat("a", maxBodyBytes+1)), nil
	}}
	gw := probeGateway(t, p, newFakeClock(), newSecrets())
	if err := myself(gw); !errors.Is(err, domain.ErrReplyTooLarge) {
		t.Fatalf("err = %v, want ErrReplyTooLarge", err)
	}
}

// TestAFailedConnectionIsUnreachable proves a network failure before a reply is ErrUnreachable, and its text is not shown.
func TestAFailedConnectionIsUnreachable(t *testing.T) {
	p := &probe{script: func(int, *http.Request) (*http.Response, error) {
		return nil, errors.New("dial tcp 203.0.113.9:443: connection refused")
	}}
	gw := probeGateway(t, p, newFakeClock(), newSecrets())
	err := myself(gw)
	if !errors.Is(err, domain.ErrUnreachable) {
		t.Fatalf("err = %v, want ErrUnreachable", err)
	}
	if strings.Contains(err.Error(), "203.0.113.9") {
		t.Fatalf("the error shows the network detail: %q", err.Error())
	}
}

// TestNoFailureShowsTheTokenOrTheEmail proves that no reply, however it fails, puts the token or the e-mail address into an
// error, even when the server echoes them in its body.
func TestNoFailureShowsTheTokenOrTheEmail(t *testing.T) {
	echo := "token " + sentinelToken + " for " + sentinelEmail
	scripts := map[string]func(int, *http.Request) (*http.Response, error){
		"401": func(int, *http.Request) (*http.Response, error) { return reply(401, echo), nil },
		"403": func(int, *http.Request) (*http.Response, error) { return reply(403, echo), nil },
		"400": func(int, *http.Request) (*http.Response, error) { return reply(400, echo), nil },
		"404": func(int, *http.Request) (*http.Response, error) { return reply(404, echo), nil },
		"410": func(int, *http.Request) (*http.Response, error) { return reply(410, echo), nil },
		"500": func(int, *http.Request) (*http.Response, error) { return reply(500, echo), nil },
		"302": func(int, *http.Request) (*http.Response, error) {
			return reply(302, echo, "Location", "https://x.invalid/"), nil
		},
		"non-JSON":   func(int, *http.Request) (*http.Response, error) { return reply(200, "<html>"+echo), nil },
		"429 long":   func(int, *http.Request) (*http.Response, error) { return reply(429, echo, "Retry-After", "999"), nil },
		"transport":  func(int, *http.Request) (*http.Response, error) { return nil, errors.New("tls: " + echo) },
		"unexpected": func(int, *http.Request) (*http.Response, error) { return reply(418, echo), nil },
	}
	for name, script := range scripts {
		p := &probe{script: script}
		gw := probeGateway(t, p, newFakeClock(), newSecrets())
		err := myself(gw)
		if err == nil {
			t.Fatalf("%s: no error", name)
		}
		text := err.Error()
		if strings.Contains(text, sentinelToken) || strings.Contains(text, sentinelEmail) {
			t.Fatalf("%s: the error shows a credential: %q", name, text)
		}
	}
}
