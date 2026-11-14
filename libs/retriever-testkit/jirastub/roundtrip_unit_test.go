package jirastub_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-testkit/jirastub"
)

func TestAHangFaultOverTheRoundTripperTimesOutAtOnce(t *testing.T) {
	stub := newStub(t)
	stub.SetFault(jirastub.Fault{Hang: true})
	client := &http.Client{Transport: stub.RoundTripper()}
	_, err := client.Get("https://jira-stub.invalid/rest/api/3/myself")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("hang error = %v, want the deadline the client would reach", err)
	}
	if got := stub.Requests(); len(got) != 1 || got[0].Status != 0 {
		t.Fatalf("recorded %+v, want the one hanging request with no status", got)
	}
}

func TestADropFaultOverTheRoundTripperReportsAClosedConnection(t *testing.T) {
	stub := newStub(t)
	stub.SetFault(jirastub.Fault{Drop: true})
	client := &http.Client{Transport: stub.RoundTripper()}
	_, err := client.Get("https://jira-stub.invalid/rest/api/3/myself")
	if err == nil || !strings.Contains(err.Error(), "closed before a reply") {
		t.Fatalf("drop error = %v, want the connection-closed report", err)
	}
}
