package outline

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/tests/support/outlinestub"
)

// TestAWriteWhoseReplyIsNotAUsableSuccessHasAnUnknownOutcome covers finding m1. The request of a write was sent, so the
// wiki may have run it. A 5xx reply, a 2xx body that is not JSON, a 2xx reply whose success is not true, and a body over
// 10 MiB all map to ErrWriteOutcomeUnknown, and the write is sent once. The {ok:true} shape of the old reading is not a
// success either. A client error keeps its own sentinel (next test).
func TestAWriteWhoseReplyIsNotAUsableSuccessHasAnUnknownOutcome(t *testing.T) {
	big := strings.Repeat("x", maxBodyBytes+1)
	cases := []struct {
		name   string
		status int
		body   string
	}{
		{"a 500 reply", http.StatusInternalServerError, `{"ok":false,"status":500}`},
		{"a 503 reply", http.StatusServiceUnavailable, ``},
		{"a 200 body that is not JSON", http.StatusOK, `not json`},
		{"a 200 reply whose success is false", http.StatusOK, `{"success":false,"data":{"total":1}}`},
		{"a 200 reply with no success", http.StatusOK, `{"status":200,"data":{"total":1}}`},
		{"a 200 reply with the old ok shape", http.StatusOK, `{"ok":true,"status":200,"data":{"total":1}}`},
		{"a 200 body over 10 MiB", http.StatusOK, big},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := writeRig(t, nil)
			r.stub.SetReply("notifications.update_all", tc.status, tc.body)
			_, err := r.gw.Write(context.Background(), domain.OpNotificationsUpdateAll,
				domain.NewArgs(map[string]any{"archivedAt": "2026-01-02T03:04:05Z"}))
			if !errors.Is(err, domain.ErrWriteOutcomeUnknown) {
				t.Fatalf("error = %v, want ErrWriteOutcomeUnknown", err)
			}
			if strings.Contains(err.Error(), "smaller limit") {
				t.Fatalf("a write's error %q tells the caller to use a smaller limit", err)
			}
			if n := r.stub.Calls("notifications.update_all"); n != 1 {
				t.Fatalf("calls = %d, want 1 (a write is never retried on a bad reply)", n)
			}
		})
	}
}

// TestAWriteReplyOverTheCapDoesNotAskForASmallerLimit checks that the 10 MiB cap of a write does not say "use a smaller
// limit": that advice belongs to a read, and a write has no limit to lower. A read of the same size keeps its own error.
func TestAWriteReplyOverTheCapDoesNotAskForASmallerLimit(t *testing.T) {
	big := strings.Repeat("x", maxBodyBytes+1)
	r := writeRig(t, nil)
	r.stub.SetReply("documents.update", http.StatusOK, big)
	_, err := r.gw.Write(context.Background(), domain.OpDocumentsUpdate, domain.NewArgs(map[string]any{"documentId": "doc-1", "title": "New"}))
	if !errors.Is(err, domain.ErrWriteOutcomeUnknown) {
		t.Fatalf("write error = %v, want ErrWriteOutcomeUnknown", err)
	}

	read := newRig(t, sampleOptions())
	read.stub.SetReply("documents.info", http.StatusOK, big)
	_, err = read.gw.Read(context.Background(), domain.OpDocumentsGet, domain.NewArgs(map[string]any{"documentId": "doc-1"}))
	if !errors.Is(err, domain.ErrReplyTooLarge) {
		t.Fatalf("read error = %v, want ErrReplyTooLarge", err)
	}
}

// TestAClientErrorOnAWriteKeepsItsOwnSentinel checks that the 4xx replies of a write are unchanged (finding m1): they are
// refusals the wiki made, so the change did not happen and the caller is told why.
func TestAClientErrorOnAWriteKeepsItsOwnSentinel(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		want   error
	}{
		{"a 400 reply", http.StatusBadRequest, `{"ok":false,"error":"validation_error"}`, domain.ErrBadRequest},
		{"a 401 reply", http.StatusUnauthorized, `{"ok":false}`, domain.ErrKeyRejected},
		{"a 403 reply", http.StatusForbidden, `{"ok":false}`, domain.ErrAccessDenied},
		{"a 404 reply", http.StatusNotFound, `{"ok":false,"message":"not found"}`, domain.ErrNotFound},
		{"a 409 reply that is not a conflict", http.StatusConflict, `{"ok":false,"error":"validation_error"}`, domain.ErrUnexpectedResponse},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := writeRig(t, nil)
			r.stub.SetReply("notifications.update_all", tc.status, tc.body)
			_, err := r.gw.Write(context.Background(), domain.OpNotificationsUpdateAll,
				domain.NewArgs(map[string]any{"archivedAt": "2026-01-02T03:04:05Z"}))
			if !errors.Is(err, tc.want) || errors.Is(err, domain.ErrWriteOutcomeUnknown) {
				t.Fatalf("error = %v, want %v and not an unknown outcome", err, tc.want)
			}
		})
	}
}

// TestATLSHandshakeFailureOnAWriteIsUnreachable covers finding m2. A certificate the client does not trust fails the TLS
// handshake before any byte of the request is sent, so the write's outcome is known: the wiki never saw it, and the
// error is the same unreachable error a dial failure gives, not ErrWriteOutcomeUnknown.
func TestATLSHandshakeFailureOnAWriteIsUnreachable(t *testing.T) {
	stub := outlinestub.New(sampleOptions()...)
	server, err := outlinestub.NewTLSServer(stub)
	if err != nil {
		t.Fatalf("NewTLSServer: %v", err)
	}
	defer server.Close()
	// The client uses the system roots, which do not hold the stub certificate, so the handshake fails.
	client := &http.Client{}
	gw, err := NewWithBase(testSourceAt(t, testHost, domain.AccessWrite), server.URL, client, &staticCreds{}, WithMinDelay(0))
	if err != nil {
		t.Fatalf("NewWithBase: %v", err)
	}
	_, err = gw.Write(context.Background(), domain.OpDocumentsCreate, domain.NewArgs(map[string]any{"title": "Plan"}))
	if !errors.Is(err, domain.ErrUnreachable) {
		t.Fatalf("error = %v, want ErrUnreachable", err)
	}
	if errors.Is(err, domain.ErrWriteOutcomeUnknown) {
		t.Fatal("a handshake failure reported an unknown outcome, but no request was sent")
	}
	if n := len(stub.Requests()); n != 0 {
		t.Fatalf("requests received = %d, want 0", n)
	}
}
