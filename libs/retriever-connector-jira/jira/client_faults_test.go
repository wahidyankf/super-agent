package jira

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// jiraFault is the fault a test makes the wait, the transport, or the body return.
var jiraFault = errors.New("injected jira client fault")

// jiraFaultRT is a round tripper that answers each request through a function the test sets. It never touches the network.
type jiraFaultRT func(*http.Request) (*http.Response, error)

func (f jiraFaultRT) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

// jiraFaultBody is a reply body that fails on the first read.
type jiraFaultBody struct{ err error }

func (b jiraFaultBody) Read([]byte) (int, error) { return 0, b.err }
func (jiraFaultBody) Close() error               { return nil }

// jiraFaultTimeout is a network error that reports a timeout, as a deadline on the connection does.
type jiraFaultTimeout struct{}

func (jiraFaultTimeout) Error() string   { return "i/o timeout" }
func (jiraFaultTimeout) Timeout() bool   { return true }
func (jiraFaultTimeout) Temporary() bool { return true }

// jiraFaultReply is a reply with the status and body the test gives.
func jiraFaultReply(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}
}

// jiraFaultGateway builds a gateway over rt, with no pacing wait and a fake clock. Options given here replace the defaults.
func jiraFaultGateway(t *testing.T, rt http.RoundTripper, opts ...Option) *Gateway {
	t.Helper()
	clock := newFakeClock()
	all := append([]Option{WithMinDelay(0), WithClock(clock.Now), WithSleep(clock.Sleep)}, opts...)
	gw, err := NewWithBase(testSource(t), testBase, &http.Client{Transport: rt}, newSecrets(), all...)
	if err != nil {
		t.Fatalf("NewWithBase: %v", err)
	}
	return gw
}

// TestJiraCallRefusesARowOutsideTheTable checks that a request whose row is not in the table is refused before any
// request is sent.
func TestJiraCallRefusesARowOutsideTheTable(t *testing.T) {
	sent := 0
	gw := jiraFaultGateway(t, jiraFaultRT(func(*http.Request) (*http.Response, error) {
		sent++
		return jiraFaultReply(http.StatusOK, `{}`), nil
	}))
	var out any
	if err := gw.call(context.Background(), endpoint{}, "", nil, nil, &out); !errors.Is(err, errMethodNotAllowed) {
		t.Fatalf("call error = %v, want errMethodNotAllowed", err)
	}
	if sent != 0 {
		t.Fatalf("a row outside the table sent %d requests", sent)
	}
}

// TestAFailedRetryWaitStopsTheCallWithItsFault checks that a fault in the wait before a 429 retry is returned, and that
// no retry is sent.
func TestAFailedRetryWaitStopsTheCallWithItsFault(t *testing.T) {
	sent := 0
	rt := jiraFaultRT(func(*http.Request) (*http.Response, error) {
		sent++
		r := jiraFaultReply(http.StatusTooManyRequests, `{}`)
		r.Header.Set("Retry-After", "1")
		return r, nil
	})
	gw := jiraFaultGateway(t, rt, WithSleep(func(context.Context, time.Duration) error { return jiraFault }))
	var out any
	if err := gw.call(context.Background(), epMyself, "", nil, nil, &out); !errors.Is(err, jiraFault) {
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
	rt := jiraFaultRT(func(*http.Request) (*http.Response, error) {
		sent++
		return jiraFaultReply(http.StatusOK, `{}`), nil
	})
	gw := jiraFaultGateway(t, rt,
		WithMinDelay(time.Second),
		WithSleep(func(context.Context, time.Duration) error { return jiraFault }))
	var out any
	if err := gw.call(context.Background(), epMyself, "", nil, nil, &out); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if err := gw.call(context.Background(), epMyself, "", nil, nil, &out); !errors.Is(err, jiraFault) {
		t.Fatalf("second call error = %v, want the pacing fault", err)
	}
	if sent != 1 {
		t.Fatalf("requests = %d, want 1 when pacing stops the second", sent)
	}
}

// TestJiraPacingNeverWaitsWhenTheGapHasPassed checks the pacing rule when the gap has already passed: the second request
// goes out with no wait at all.
func TestJiraPacingNeverWaitsWhenTheGapHasPassed(t *testing.T) {
	now := epoch
	clock := func() time.Time {
		now = now.Add(time.Hour)
		return now
	}
	slept := 0
	rt := jiraFaultRT(func(*http.Request) (*http.Response, error) {
		return jiraFaultReply(http.StatusOK, `{}`), nil
	})
	gw := jiraFaultGateway(t, rt,
		WithMinDelay(time.Second),
		WithClock(clock),
		WithSleep(func(context.Context, time.Duration) error { slept++; return nil }))
	var out any
	for i := 0; i < 2; i++ {
		if err := gw.call(context.Background(), epMyself, "", nil, nil, &out); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}
	if slept != 0 {
		t.Fatalf("pacing slept %d times, want none when the gap has passed", slept)
	}
}

// TestAReplyBodyFaultIsUnreachable checks a reply whose body fails: the read is not complete, so the call is unreachable.
func TestAReplyBodyFaultIsUnreachable(t *testing.T) {
	gw := jiraFaultGateway(t, jiraFaultRT(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: jiraFaultBody{err: jiraFault}}, nil
	}))
	var out any
	if err := gw.call(context.Background(), epMyself, "", nil, nil, &out); !errors.Is(err, domain.ErrUnreachable) {
		t.Fatalf("call error = %v, want ErrUnreachable", err)
	}
}

// TestAnInvalidBaseIsRefusedBeforeAnyRequest checks that a base the request builder rejects is an invalid base, and no
// request is sent.
func TestAnInvalidBaseIsRefusedBeforeAnyRequest(t *testing.T) {
	sent := 0
	gw := jiraFaultGateway(t, jiraFaultRT(func(*http.Request) (*http.Response, error) {
		sent++
		return jiraFaultReply(http.StatusOK, `{}`), nil
	}))
	gw.base = "https://bad\x7fhost"
	var out any
	if err := gw.call(context.Background(), epMyself, "", nil, nil, &out); !errors.Is(err, errInvalidBaseURL) {
		t.Fatalf("call error = %v, want errInvalidBaseURL", err)
	}
	if sent != 0 {
		t.Fatalf("an invalid base sent %d requests", sent)
	}
}

// TestJiraTransportErrorMapsEachFailureToItsSentinel checks transportError: a cancelled caller gets its own error, a
// deadline and a network timeout are timed out, and any other fault is unreachable.
func TestJiraTransportErrorMapsEachFailureToItsSentinel(t *testing.T) {
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	tests := []struct {
		name   string
		parent context.Context
		err    error
		want   error
	}{
		{name: "a cancelled caller gets its cancellation", parent: cancelled, err: jiraFault, want: context.Canceled},
		{name: "a deadline is timed out", parent: context.Background(), err: context.DeadlineExceeded, want: domain.ErrTimedOut},
		{name: "a network timeout is timed out", parent: context.Background(), err: jiraFaultTimeout{}, want: domain.ErrTimedOut},
		{name: "any other fault is unreachable", parent: context.Background(), err: jiraFault, want: domain.ErrUnreachable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := transportError(tt.parent, tt.err); !errors.Is(got, tt.want) {
				t.Fatalf("transportError = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestJiraSleepContextHonoursItsContext checks sleepContext: a cancelled context ends the wait at once, a short wait ends
// normally, and a zero wait returns at once.
func TestJiraSleepContextHonoursItsContext(t *testing.T) {
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

// Ensure the net import is used by the timeout fault above.
var _ net.Error = jiraFaultTimeout{}
