//go:build integration

package outline

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
	"github.com/wahidyankf/super-agent/libs/retriever-testkit/outlinestub"
	"github.com/wahidyankf/super-agent/libs/retriever-testkit/sourcecontract"
)

// These tests open real TLS loopback listeners through httptest. They run with the
// integration tag, so the unit layer stays free of sockets.

// TestGatewayPassesTheSourceContractOverTLS runs the shared contract suite against the
// Outline gateway on a real loopback TLS listener.
func TestGatewayPassesTheSourceContractOverTLS(t *testing.T) {
	sourcecontract.Run(t, func(t *testing.T, w sourcecontract.Wiring) ports.SourceGateway {
		stub := outlinestub.Start(t, contractStubOptions(w)...)
		stubFault(stub, w.Fault)
		gw, err := NewWithBase(testSource(t, testHost), stub.URL(), stub.Client(), contractCreds(t, w), WithMinDelay(0), WithSleep(noSleep))
		if err != nil {
			t.Fatalf("NewWithBase: %v", err)
		}
		return gw
	})
}

// TestSeamReadsTheCAFileAndSendsTheLogicalHost runs the seam against a loopback stub and
// checks that the request still names the logical host in its Host header.
func TestSeamReadsTheCAFileAndSendsTheLogicalHost(t *testing.T) {
	stub := outlinestub.Start(t, outlinestub.WithCollections(outlinestub.NumberedCollections(3)...))
	caPath, err := stub.WriteCAFile(t.TempDir())
	if err != nil {
		t.Fatalf("WriteCAFile: %v", err)
	}
	env := map[string]string{BaseURLEnv: stub.URL(), CAFileEnv: caPath}
	seam, err := seamFromEnvDisk(func(name string) string { return env[name] })
	if err != nil || !seam.Enabled {
		t.Fatalf("TestSeamFromEnv enabled=%v err=%v, want the seam on", seam.Enabled, err)
	}

	gw, err := NewWithBase(testSource(t, testHost), seam.BaseURL, seam.Client, &staticCreds{}, WithMinDelay(0), WithSleep(noSleep))
	if err != nil {
		t.Fatalf("NewWithBase: %v", err)
	}
	got, err := gw.ListCollections(context.Background())
	if err != nil {
		t.Fatalf("ListCollections over the seam: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("ListCollections returned %d collections, want 3", len(got))
	}
	for i, req := range stub.Requests() {
		if req.Host != testHost {
			t.Fatalf("request %d Host = %q, want %q", i, req.Host, testHost)
		}
	}
}

// TestSeamRejectsACAFileWithNoCertificate refuses a CA file that holds no PEM certificate.
func TestSeamRejectsACAFileWithNoCertificate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-a-certificate.pem")
	if err := os.WriteFile(path, []byte("not a certificate"), 0o600); err != nil {
		t.Fatalf("write CA file: %v", err)
	}
	env := map[string]string{BaseURLEnv: "https://127.0.0.1:8443", CAFileEnv: path}
	seam, err := seamFromEnvDisk(func(name string) string { return env[name] })
	if !errors.Is(err, errSeamNoCertificate) || seam.Enabled {
		t.Fatalf("TestSeamFromEnv enabled=%v err=%v, want errSeamNoCertificate", seam.Enabled, err)
	}
}

// TestStubRecordsTheTLSFlagOverTheListener checks that requests over the TLS listener are
// recorded with their HTTP method and as secure.
func TestStubRecordsTheTLSFlagOverTheListener(t *testing.T) {
	stub := outlinestub.Start(t, outlinestub.WithCollections(outlinestub.NumberedCollections(1)...))
	client := stub.Client()

	resp, err := client.Post(stub.URL()+"/api/collections.list", "application/json", strings.NewReader(`{"offset":0,"limit":25}`))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	resp.Body.Close()
	resp, err = client.Get(stub.URL() + "/api/collections.list")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	resp.Body.Close()

	reqs := stub.Requests()
	if len(reqs) != 2 {
		t.Fatalf("recorded %d requests, want 2", len(reqs))
	}
	for i, req := range reqs {
		if !req.Secure {
			t.Fatalf("request %d over the TLS listener is recorded as not secure", i)
		}
	}
	if reqs[0].HTTPMethod != http.MethodPost || reqs[1].HTTPMethod != http.MethodGet {
		t.Fatalf("recorded methods = %q, %q, want POST then GET", reqs[0].HTTPMethod, reqs[1].HTTPMethod)
	}
}

// TestSeamTimeoutBoundsEachRequestOverTLS reads the timeout from the seam, then points the
// gateway at a listener that never answers. The request must fail with ErrTimedOut at the
// seam's bound, far below the default 30 seconds.
func TestSeamTimeoutBoundsEachRequestOverTLS(t *testing.T) {
	stub := outlinestub.Start(t, outlinestub.WithCollections(outlinestub.NumberedCollections(1)...))
	stub.SetFault(outlinestub.Fault{Hang: true})
	caPath, err := stub.WriteCAFile(t.TempDir())
	if err != nil {
		t.Fatalf("WriteCAFile: %v", err)
	}
	env := map[string]string{BaseURLEnv: stub.URL(), CAFileEnv: caPath, TimeoutEnv: "150ms"}
	seam, err := seamFromEnvDisk(func(name string) string { return env[name] })
	if err != nil || !seam.Enabled {
		t.Fatalf("TestSeamFromEnv enabled=%v err=%v, want the seam on", seam.Enabled, err)
	}
	if seam.Timeout != 150*time.Millisecond {
		t.Fatalf("seam timeout = %v, want 150ms", seam.Timeout)
	}

	gw, err := NewWithBase(testSource(t, testHost), seam.BaseURL, seam.Client, &staticCreds{}, WithTimeout(seam.Timeout), WithMinDelay(0), WithSleep(noSleep))
	if err != nil {
		t.Fatalf("NewWithBase: %v", err)
	}
	start := time.Now()
	if _, err := gw.ListCollections(context.Background()); !errors.Is(err, domain.ErrTimedOut) {
		t.Fatalf("ListCollections error = %v, want ErrTimedOut", err)
	}
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Fatalf("request ran %v, want the seam's 150ms bound", elapsed)
	}
}

// TestRealTimeoutMapsToErrTimedOut uses a listener that never answers and a short timeout.
func TestRealTimeoutMapsToErrTimedOut(t *testing.T) {
	stub := outlinestub.Start(t, outlinestub.WithCollections(outlinestub.NumberedCollections(1)...))
	stub.SetFault(outlinestub.Fault{Hang: true})
	gw, err := NewWithBase(testSource(t, testHost), stub.URL(), stub.Client(), &staticCreds{}, WithTimeout(200*time.Millisecond), WithMinDelay(0))
	if err != nil {
		t.Fatalf("NewWithBase: %v", err)
	}
	if _, err := gw.ListCollections(context.Background()); !errors.Is(err, domain.ErrTimedOut) {
		t.Fatalf("ListCollections error = %v, want ErrTimedOut", err)
	}
}

// TestRealDroppedConnectionMapsToErrUnreachable closes the connection before any reply.
func TestRealDroppedConnectionMapsToErrUnreachable(t *testing.T) {
	stub := outlinestub.Start(t, outlinestub.WithCollections(outlinestub.NumberedCollections(1)...))
	stub.SetFault(outlinestub.Fault{Drop: true})
	gw, err := NewWithBase(testSource(t, testHost), stub.URL(), stub.Client(), &staticCreds{}, WithMinDelay(0))
	if err != nil {
		t.Fatalf("NewWithBase: %v", err)
	}
	if _, err := gw.ListCollections(context.Background()); !errors.Is(err, domain.ErrUnreachable) {
		t.Fatalf("ListCollections error = %v, want ErrUnreachable", err)
	}
}

// TestRealPacingLeavesAtLeastOneHundredMillisecondsBetweenRequests uses the default
// real delay and clock, and checks the arrival gaps the listener saw.
func TestRealPacingLeavesAtLeastOneHundredMillisecondsBetweenRequests(t *testing.T) {
	stub := outlinestub.Start(t, outlinestub.WithCollections(outlinestub.NumberedCollections(60)...))
	gw, err := NewWithBase(testSource(t, testHost), stub.URL(), stub.Client(), &staticCreds{})
	if err != nil {
		t.Fatalf("NewWithBase: %v", err)
	}
	if _, err := gw.ListCollections(context.Background()); err != nil {
		t.Fatalf("ListCollections: %v", err)
	}
	reqs := stub.Requests()
	if len(reqs) != 3 {
		t.Fatalf("requests = %d, want 3", len(reqs))
	}
	for i := 1; i < len(reqs); i++ {
		if gap := reqs[i].At.Sub(reqs[i-1].At); gap < defaultMinDelay {
			t.Fatalf("gap before request %d = %v, want at least %v", i, gap, defaultMinDelay)
		}
	}
}

// TestRealRetryAfterWaitsItsSecondsBeforeRetrying checks the real arrival gap after a 429.
func TestRealRetryAfterWaitsItsSecondsBeforeRetrying(t *testing.T) {
	stub := outlinestub.Start(t, outlinestub.WithCollections(outlinestub.NumberedCollections(1)...))
	stub.SetFaultFirst(1, outlinestub.Fault{Status: 429, RetryAfter: "1"})
	gw, err := NewWithBase(testSource(t, testHost), stub.URL(), stub.Client(), &staticCreds{})
	if err != nil {
		t.Fatalf("NewWithBase: %v", err)
	}
	if _, err := gw.ListCollections(context.Background()); err != nil {
		t.Fatalf("ListCollections: %v", err)
	}
	reqs := stub.Requests()
	if len(reqs) != 2 {
		t.Fatalf("requests = %d, want 2", len(reqs))
	}
	if gap := reqs[1].At.Sub(reqs[0].At); gap < time.Second {
		t.Fatalf("retry came %v after the 429, want at least 1s", gap)
	}
}

// TestRealConcurrentCallsNeverOverlap sends concurrent calls at a listener that can
// answer in parallel, and checks that the gateway still keeps one request in flight.
func TestRealConcurrentCallsNeverOverlap(t *testing.T) {
	stub := outlinestub.Start(t, outlinestub.WithCollections(outlinestub.NumberedCollections(3)...))
	gw, err := NewWithBase(testSource(t, testHost), stub.URL(), stub.Client(), &staticCreds{}, WithMinDelay(0))
	if err != nil {
		t.Fatalf("NewWithBase: %v", err)
	}

	const callers = 5
	var wg sync.WaitGroup
	errs := make(chan error, callers)
	for i := 0; i < callers; i++ {
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
		if err != nil {
			t.Fatalf("concurrent ListCollections: %v", err)
		}
	}
	if peak := stub.PeakInFlight(); peak != 1 {
		t.Fatalf("peak requests in flight = %d, want 1", peak)
	}
}

// TestRealNoOtherRequestGoesOutDuringARetryAfterWait starts a second call while the first
// waits out a 429 Retry-After. The second call must wait for the retry, so the listener
// sees the retry before any other method.
func TestRealNoOtherRequestGoesOutDuringARetryAfterWait(t *testing.T) {
	stub := outlinestub.Start(t, outlinestub.WithCollections(outlinestub.NumberedCollections(1)...))
	stub.SetFaultFirst(1, outlinestub.Fault{Status: 429, RetryAfter: "1"})
	gw, err := NewWithBase(testSource(t, testHost), stub.URL(), stub.Client(), &staticCreds{}, WithMinDelay(0))
	if err != nil {
		t.Fatalf("NewWithBase: %v", err)
	}
	ctx := context.Background()
	colID := mustColID(t, "col-001")
	errs := make(chan error, 2)

	go func() {
		_, err := gw.ListCollections(ctx)
		errs <- err
	}()
	waitUntil(t, func() bool {
		reqs := stub.Requests()
		return len(reqs) == 1 && reqs[0].Status == http.StatusTooManyRequests
	})
	go func() {
		_, err := gw.CollectionTree(ctx, colID)
		errs <- err
	}()
	for i := 0; i < 2; i++ {
		if err := <-errs; err != nil {
			t.Fatalf("concurrent call: %v", err)
		}
	}

	reqs := stub.Requests()
	want := []string{"collections.list", "collections.list", "collections.documents"}
	if len(reqs) != len(want) {
		t.Fatalf("requests = %d, want %d", len(reqs), len(want))
	}
	for i, method := range want {
		if reqs[i].Method != method {
			t.Fatalf("request %d method = %q, want %q: a call went out during the Retry-After wait", i, reqs[i].Method, method)
		}
	}
}

// waitUntil polls cond for up to five seconds and fails the test if it never holds.
func waitUntil(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("condition did not hold within five seconds")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// TestRealRequestsCarryTheBearerKeyAndTheLogicalHost checks what a real TLS listener sees:
// the URL names the loopback address, and the Host header names the logical host.
func TestRealRequestsCarryTheBearerKeyAndTheLogicalHost(t *testing.T) {
	stub := outlinestub.Start(t, outlinestub.WithCollections(outlinestub.NumberedCollections(1)...))
	gw, err := NewWithBase(testSource(t, testHost), stub.URL(), stub.Client(), &staticCreds{}, WithMinDelay(0))
	if err != nil {
		t.Fatalf("NewWithBase: %v", err)
	}
	if _, err := gw.ListCollections(context.Background()); err != nil {
		t.Fatalf("ListCollections: %v", err)
	}
	for i, req := range stub.Requests() {
		if req.Host != testHost {
			t.Fatalf("request %d Host = %q, want %q", i, req.Host, testHost)
		}
		if req.Authorization != "Bearer "+sentinelKey {
			t.Fatalf("request %d Authorization was not the bearer key", i)
		}
	}
	if stub.Calls("collections.list") != 1 {
		t.Fatalf("collections.list calls = %d, want 1", stub.Calls("collections.list"))
	}
}

// TestSeamSendsEachSourceItsOwnHostAndTLSName serves a second source, whose endpoint is not the built-in host,
// through the seam. The connection goes to the seam base, but the Host header and the TLS name come from that
// source's own endpoint.
func TestSeamSendsEachSourceItsOwnHostAndTLSName(t *testing.T) {
	stub := outlinestub.Start(t, outlinestub.WithCollections(outlinestub.NumberedCollections(2)...))
	caPath, err := stub.WriteCAFile(t.TempDir())
	if err != nil {
		t.Fatalf("WriteCAFile: %v", err)
	}
	env := map[string]string{BaseURLEnv: stub.URL(), CAFileEnv: caPath}
	seam, err := seamFromEnvDisk(func(name string) string { return env[name] })
	if err != nil || !seam.Enabled {
		t.Fatalf("TestSeamFromEnv enabled=%v err=%v, want the seam on", seam.Enabled, err)
	}

	const engineeringHost = "wiki.engineering.example.com"
	gw, err := NewWithBase(testSource(t, engineeringHost), seam.BaseURL, seam.Client, &staticCreds{}, WithMinDelay(0), WithSleep(noSleep))
	if err != nil {
		t.Fatalf("NewWithBase: %v", err)
	}
	got, err := gw.ListCollections(context.Background())
	if err != nil {
		t.Fatalf("ListCollections over the seam: %v; the TLS name must be the source endpoint", err)
	}
	if len(got) != 2 {
		t.Fatalf("ListCollections returned %d collections, want 2", len(got))
	}
	for i, req := range stub.Requests() {
		if req.Host != engineeringHost {
			t.Fatalf("request %d Host = %q, want the source endpoint %q", i, req.Host, engineeringHost)
		}
	}
}

// seamFromEnvDisk reads the seam through getenv with the real CA reader. Only an integration test calls it.
func seamFromEnvDisk(getenv func(string) string) (Seam, error) {
	return TestSeamFromEnvWithFiles(getenv, os.ReadFile)
}
