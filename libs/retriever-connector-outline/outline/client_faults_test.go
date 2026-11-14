package outline

import (
	"context"
	"errors"
	"io"
	"math"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// errClientFault is the fault a test makes the transport, the wait, or the body return.
var errClientFault = errors.New("injected client fault")

// clientFaultRT is a round tripper that answers each request through a function the test sets. It never touches the
// network.
type clientFaultRT func(*http.Request) (*http.Response, error)

func (f clientFaultRT) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

// clientFaultBody is a reply body that fails on the first read.
type clientFaultBody struct{ err error }

func (b clientFaultBody) Read([]byte) (int, error) { return 0, b.err }
func (clientFaultBody) Close() error               { return nil }

// clientFaultTimeout is a network error that reports a timeout, as a deadline on the connection does.
type clientFaultTimeout struct{}

func (clientFaultTimeout) Error() string   { return "i/o timeout" }
func (clientFaultTimeout) Timeout() bool   { return true }
func (clientFaultTimeout) Temporary() bool { return true }

// clientFaultReply is a reply with the status and the JSON body the test gives.
func clientFaultReply(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}
}

// clientFaultGateway builds a gateway over rt, with no pacing wait and a fake clock. Options given here replace
// the defaults.
func clientFaultGateway(t *testing.T, rt http.RoundTripper, opts ...Option) *Gateway {
	t.Helper()
	clock := newFakeClock()
	all := append([]Option{WithMinDelay(0), WithClock(clock.Now), WithSleep(clock.Sleep)}, opts...)
	gw, err := NewWithBase(testSourceAt(t, testHost), testBase, &http.Client{Transport: rt}, &staticCreds{}, all...)
	if err != nil {
		t.Fatalf("NewWithBase: %v", err)
	}
	return gw
}

// TestAnUnencodableRequestIsRefusedBeforeAnyRequest checks that a request the JSON encoder cannot hold is refused before a
// request is sent.
func TestAnUnencodableRequestIsRefusedBeforeAnyRequest(t *testing.T) {
	sent := 0
	gw := clientFaultGateway(t, clientFaultRT(func(*http.Request) (*http.Response, error) {
		sent++
		return clientFaultReply(http.StatusOK, `{"data":{}}`), nil
	}))
	var out any
	if err := gw.call(context.Background(), domain.OpCollectionsInfo, methodCollectionsInfo, math.NaN(), &out); err == nil {
		t.Fatalf("call with an unencodable request returned no error")
	}
	if sent != 0 {
		t.Fatalf("an unencodable request sent %d requests", sent)
	}
}

// TestAFailedRetryWaitStopsTheCallWithItsFault checks that a fault in the wait before a 429 retry is returned, and that no
// retry is sent.
func TestAFailedRetryWaitStopsTheCallWithItsFault(t *testing.T) {
	sent := 0
	rt := clientFaultRT(func(*http.Request) (*http.Response, error) {
		sent++
		r := clientFaultReply(http.StatusTooManyRequests, `{}`)
		r.Header.Set("Retry-After", "1")
		return r, nil
	})
	gw := clientFaultGateway(t, rt, WithSleep(func(context.Context, time.Duration) error { return errClientFault }))
	var out any
	if err := gw.call(context.Background(), domain.OpCollectionsInfo, methodCollectionsInfo, map[string]any{}, &out); !errors.Is(err, errClientFault) {
		t.Fatalf("call error = %v, want the wait fault", err)
	}
	if sent != 1 {
		t.Fatalf("requests = %d, want 1 with no retry after the wait fault", sent)
	}
}

// TestAFailedPacingWaitStopsTheNextRequest checks that a fault in the pacing wait stops the second request, which is never
// sent.
func TestAFailedPacingWaitStopsTheNextRequest(t *testing.T) {
	sent := 0
	rt := clientFaultRT(func(*http.Request) (*http.Response, error) {
		sent++
		return clientFaultReply(http.StatusOK, `{"data":{}}`), nil
	})
	gw := clientFaultGateway(t, rt,
		WithMinDelay(time.Second),
		WithSleep(func(context.Context, time.Duration) error { return errClientFault }))
	var out any
	if err := gw.call(context.Background(), domain.OpCollectionsInfo, methodCollectionsInfo, map[string]any{}, &out); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if err := gw.call(context.Background(), domain.OpCollectionsInfo, methodCollectionsInfo, map[string]any{}, &out); !errors.Is(err, errClientFault) {
		t.Fatalf("second call error = %v, want the pacing fault", err)
	}
	if sent != 1 {
		t.Fatalf("requests = %d, want 1 when pacing stops the second", sent)
	}
}

// TestPacingNeverWaitsWhenTheGapHasPassed checks the pacing rule when the gap has already passed: the second request goes
// out with no wait at all.
func TestPacingNeverWaitsWhenTheGapHasPassed(t *testing.T) {
	now := epoch
	clock := func() time.Time {
		now = now.Add(time.Hour)
		return now
	}
	slept := 0
	rt := clientFaultRT(func(*http.Request) (*http.Response, error) {
		return clientFaultReply(http.StatusOK, `{"data":{}}`), nil
	})
	gw := clientFaultGateway(t, rt,
		WithMinDelay(time.Second),
		WithClock(clock),
		WithSleep(func(context.Context, time.Duration) error { slept++; return nil }))
	var out any
	for i := 0; i < 2; i++ {
		if err := gw.call(context.Background(), domain.OpCollectionsInfo, methodCollectionsInfo, map[string]any{}, &out); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}
	if slept != 0 {
		t.Fatalf("pacing slept %d times, want none when the gap has passed", slept)
	}
}

// TestAnInvalidBaseIsRefusedBeforeAnyRequest checks that a base URL the request builder rejects is an invalid base, and no
// request is sent.
func TestAnInvalidBaseIsRefusedBeforeAnyRequest(t *testing.T) {
	sent := 0
	gw := clientFaultGateway(t, clientFaultRT(func(*http.Request) (*http.Response, error) {
		sent++
		return clientFaultReply(http.StatusOK, `{}`), nil
	}))
	gw.base = "https://bad\x7fhost"
	var out any
	if err := gw.call(context.Background(), domain.OpCollectionsInfo, methodCollectionsInfo, map[string]any{}, &out); !errors.Is(err, errInvalidBaseURL) {
		t.Fatalf("call error = %v, want errInvalidBaseURL", err)
	}
	if sent != 0 {
		t.Fatalf("an invalid base sent %d requests", sent)
	}
}

// TestAReplyBodyFaultIsUnknownForAWriteAndUnreachableForARead checks a reply whose body fails after its status. Outline
// answered the status, so a write may have run, and its outcome is unknown; a read is unreachable.
func TestAReplyBodyFaultIsUnknownForAWriteAndUnreachableForARead(t *testing.T) {
	broken := clientFaultRT(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: clientFaultBody{err: errClientFault}}, nil
	})
	var out any

	write := clientFaultGateway(t, broken)
	if err := write.call(context.Background(), domain.OpCollectionsMove, "collections.move", map[string]any{}, &out); !errors.Is(err, domain.ErrWriteOutcomeUnknown) {
		t.Fatalf("write error = %v, want ErrWriteOutcomeUnknown", err)
	}

	read := clientFaultGateway(t, broken)
	if err := read.call(context.Background(), domain.OpCollectionsInfo, methodCollectionsInfo, map[string]any{}, &out); !errors.Is(err, domain.ErrUnreachable) {
		t.Fatalf("read error = %v, want ErrUnreachable", err)
	}
}

// TestRetryAfterTooLargeToHoldIsOneOverTheLimit checks parseRetryAfter on a value too large for an int: it is reported one
// over the limit, which then fails without a retry.
func TestRetryAfterTooLargeToHoldIsOneOverTheLimit(t *testing.T) {
	got, ok := parseRetryAfter("99999999999999999999999")
	if !ok || got != maxRetryAfter+1 {
		t.Fatalf("parseRetryAfter = %d, %v; want %d, true", got, ok, maxRetryAfter+1)
	}
	if _, ok := parseRetryAfter("-1"); ok {
		t.Fatalf("parseRetryAfter accepted a negative value")
	}
}

// TestTransportErrorMapsEachFailureToItsSentinel checks transportError: a cancelled caller gets its own error, a deadline
// and a network timeout are timed out, and any other fault is unreachable.
func TestTransportErrorMapsEachFailureToItsSentinel(t *testing.T) {
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	tests := []struct {
		name   string
		parent context.Context
		err    error
		want   error
	}{
		{name: "a cancelled caller gets its cancellation", parent: cancelled, err: errClientFault, want: context.Canceled},
		{name: "a deadline is timed out", parent: context.Background(), err: context.DeadlineExceeded, want: domain.ErrTimedOut},
		{name: "a network timeout is timed out", parent: context.Background(), err: clientFaultTimeout{}, want: domain.ErrTimedOut},
		{name: "any other fault is unreachable", parent: context.Background(), err: errClientFault, want: domain.ErrUnreachable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := transportError(tt.parent, tt.err); !errors.Is(got, tt.want) {
				t.Fatalf("transportError = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestSleepContextHonoursItsContext checks sleepContext: a cancelled context ends the wait at once, a short wait ends
// normally, and a zero wait returns at once.
func TestSleepContextHonoursItsContext(t *testing.T) {
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := sleepContext(cancelled, time.Hour); !errors.Is(err, context.Canceled) {
		t.Fatalf("sleepContext with a cancelled context = %v, want context.Canceled", err)
	}
	if err := sleepContext(context.Background(), time.Millisecond); err != nil {
		t.Fatalf("sleepContext of a short wait = %v, want nil", err)
	}
	if err := sleepContext(context.Background(), 0); err != nil {
		t.Fatalf("sleepContext of zero = %v, want nil", err)
	}
}
