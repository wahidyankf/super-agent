package graphql

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// failingWriter is a response writer whose body write always fails, as a client that has gone away does.
type failingWriter struct {
	header http.Header
	status int
}

func (w *failingWriter) Header() http.Header {
	if w.header == nil {
		w.header = http.Header{}
	}
	return w.header
}

func (w *failingWriter) WriteHeader(status int) { w.status = status }

func (w *failingWriter) Write([]byte) (int, error) { return 0, errors.New("client gone") }

// errReader is a request body that fails on every read.
type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("body unreadable") }

// TestWriteErrorsSurvivesAClientThatHasGone checks that a refusal written to a client that has gone away returns without
// a panic, and that the status is still set.
func TestWriteErrorsSurvivesAClientThatHasGone(t *testing.T) {
	w := &failingWriter{}
	writeErrors(w, http.StatusBadRequest, "refused")
	if w.status != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.status)
	}
}

// TestNewHandlerWithoutALoggerStillServes checks that a handler built with no logger uses a discarding one.
func TestNewHandlerWithoutALoggerStillServes(t *testing.T) {
	h, err := NewHandler(
		Deps{Service: oneOutlineSources(&countingGateway{})},
		WithListener(tcpListener("127.0.0.1", testPort)),
	)
	if err != nil || h == nil {
		t.Fatalf("NewHandler without a logger = %v, %v", h, err)
	}
	rec := sendTo(t, h, graphqlPOST(`{ collections { id } }`))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

// TestMeasureQueryRefusesFragmentsPastTheLimits checks that the cost and depth rules also hold inside inline fragments and
// fragment spreads, where the meter charges and walks separately.
func TestMeasureQueryRefusesFragmentsPastTheLimits(t *testing.T) {
	plain := strings.Repeat(" __typename", 100)
	cases := []struct {
		name    string
		query   string
		wantErr error
	}{
		{
			name:    "an inline fragment past the cost limit is refused",
			query:   "{" + plain + " ... on Query { __typename } }",
			wantErr: errComplexityLimit,
		},
		{
			name:    "a fragment spread past the cost limit is refused",
			query:   "{" + plain + " ...F } fragment F on Query { __typename }",
			wantErr: errComplexityLimit,
		},
		{
			name:    "an inline fragment nested past the depth limit is refused",
			query:   `{ a { b { c { d { ... on X { e } } } } } }`,
			wantErr: errDepthLimit,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := measureQuery(tc.query); !errors.Is(err, tc.wantErr) {
				t.Fatalf("measureQuery error = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

// TestLimitQueryRefusesAnUnreadableBody checks that a body the server cannot read is refused with 400, and that the
// next handler is not reached.
func TestLimitQueryRefusesAnUnreadableBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", errReader{})
	rec, next := runGuard(limitQuery(), req)
	assertRefused(t, rec, next, http.StatusBadRequest, "could not read request body")
}

// TestLimitBodyRefusesAnUnreadableBody checks that a body that fails while it is read is refused with 400 when its length
// is unknown.
func TestLimitBodyRefusesAnUnreadableBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", errReader{})
	req.ContentLength = -1
	rec, next := runGuard(limitBody(64), req)
	assertRefused(t, rec, next, http.StatusBadRequest, "could not read request body")
}

// TestLimitBodyRefusesALongBodyWithAnUnknownLength checks that a body with no declared length is still capped at the
// limit, and refused with 413 once it reads past it.
func TestLimitBodyRefusesALongBodyWithAnUnknownLength(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(strings.Repeat("x", 50)))
	req.ContentLength = -1
	rec, next := runGuard(limitBody(10), req)
	assertRefused(t, rec, next, http.StatusRequestEntityTooLarge, "size limit")
}

// TestResolverRefusesAnUnknownKindBeforeAnyRequest checks that a collectionTree or document query with a kind the code
// does not know is refused before any gateway is called.
func TestResolverRefusesAnUnknownKindBeforeAnyRequest(t *testing.T) {
	queries := []string{
		`{ collectionTree(id: "c", kind: "notion") { id } }`,
		`{ document(id: "d", kind: "notion") { id } }`,
	}
	for _, query := range queries {
		gw := &countingGateway{}
		r := sendServed(t, oneOutlineSources(gw), query)
		assertErrorContains(t, r, "unknown source kind")
		if gw.calls.Load() != 0 {
			t.Fatalf("%s reached a gateway", query)
		}
	}
}

// TestCollectionTreeReportsAnUpstreamFailureWithoutItsText checks that a failing source call is reported as one generic
// message, with none of the upstream text.
func TestCollectionTreeReportsAnUpstreamFailureWithoutItsText(t *testing.T) {
	gw := &countingGateway{err: errors.New("upstream said secret-text")}
	r := sendServed(t, oneOutlineSources(gw), `{ collectionTree(id: "c") { id } }`)
	if msgs := r.messages(); len(msgs) != 1 || strings.Contains(msgs[0], "secret-text") {
		t.Fatalf("messages = %v, want one generic message without the upstream text", msgs)
	}
}

// TestServeReportsAListenerThatFailsBeforeCancel checks that a listener which stops serving before the context ends is
// reported as an error, not a clean shutdown.
func TestServeReportsAListenerThatFailsBeforeCancel(t *testing.T) {
	l, err := Listen(0)
	if err != nil {
		t.Fatal(err)
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	if err := Serve(t.Context(), l, http.NotFoundHandler()); err == nil {
		t.Fatal("Serve returned nil for a listener that was closed before it started")
	}
}

// TestServeReportsAShutdownThatTimesOut checks that a request still running when the context ends is reported as a
// shutdown failure. The handler blocks until the test releases it, so the graceful shutdown runs out of its time limit,
// which is five seconds.
func TestServeReportsAShutdownThatTimesOut(t *testing.T) {
	l, err := Listen(0)
	if err != nil {
		t.Fatal(err)
	}
	entered := make(chan struct{})
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	handler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		close(entered)
		<-release
	})
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	serveErr := make(chan error, 1)
	go func() { serveErr <- Serve(ctx, l, handler) }()
	go func() {
		req, reqErr := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+l.Addr().String()+"/", nil)
		if reqErr != nil {
			return
		}
		resp, doErr := (&http.Client{}).Do(req)
		if doErr == nil {
			_ = resp.Body.Close()
		}
	}()
	<-entered
	cancel()
	err = <-serveErr
	if err == nil || !strings.Contains(err.Error(), "shutdown of the GraphQL server") {
		t.Fatalf("Serve error = %v, want the shutdown failure", err)
	}
}
