package engine_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
)

// TestCodeNamesTheAreaAndTheReason checks the stable codes. The area is the source kind, except for a refused network,
// which takes the connector area of the ADR.
func TestCodeNamesTheAreaAndTheReason(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"a timeout", domain.ErrTimedOut, "retriever.example-wiki.timed_out"},
		{"a deadline", context.DeadlineExceeded, "retriever.example-wiki.timed_out"},
		{"a redirect", domain.ErrRedirected, "retriever.example-wiki.redirected"},
		{"a reply over the cap", domain.ErrReplyTooLarge, "retriever.example-wiki.reply_too_large"},
		{"a shape that is refused", domain.ErrUnexpectedResponse, "retriever.example-wiki.unexpected_response"},
		{"a refused network", engine.ErrNetworkRefused, "retriever.connector.network_refused"},
		{"a refused header", engine.ErrInvalidHeader, "retriever.example-wiki.invalid_header"},
		{"a wrapped reason", fmt.Errorf("wrapped: %w", domain.ErrNotFound), "retriever.example-wiki.not_found"},
		{"an error with no reason", errors.New("other"), "retriever.example-wiki.failed"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := engine.Code("example-wiki", c.err); got != c.want {
				t.Fatalf("Code = %q, want %q", got, c.want)
			}
		})
	}
}

// TestExitStatusIsTheContractOfTheCommandLine checks the statuses: a timeout and a rate limit wait, and every other
// failure did not complete.
func TestExitStatusIsTheContractOfTheCommandLine(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"a timeout", domain.ErrTimedOut, 124},
		{"a deadline", context.DeadlineExceeded, 124},
		{"a rate limit", domain.ErrSourceRateLimited, 124},
		{"a refused network", engine.ErrNetworkRefused, 2},
		{"a not found", domain.ErrNotFound, 2},
		{"a refused redirect", domain.ErrRedirected, 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := engine.ExitStatus(c.err); got != c.want {
				t.Fatalf("ExitStatus = %d, want %d", got, c.want)
			}
		})
	}
}
