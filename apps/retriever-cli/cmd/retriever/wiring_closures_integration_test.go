//go:build integration

package main

import (
	"bytes"
	"context"
	"encoding/pem"
	"net"
	"strings"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/graphql"
	"github.com/wahidyankf/super-agent/libs/retriever-connector-jira/jira"
	"github.com/wahidyankf/super-agent/libs/retriever-connector-kiwi/kiwi"
	"github.com/wahidyankf/super-agent/libs/retriever-connector-outline/outline"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/configpaths"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/envfile"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/sources"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// newSeamFixture starts a loopback TLS server for the test and writes its certificate as the CA file.
func newSeamFixture(t *testing.T) seamFixture {
	t.Helper()
	srv := httptest.NewTLSServer(http.NotFoundHandler())
	t.Cleanup(srv.Close)
	ca := filepath.Join(t.TempDir(), "ca.pem")
	data := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw})
	if err := os.WriteFile(ca, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return seamFixture{base: srv.URL, ca: ca}
}

// client returns an HTTP client that trusts the fixture's certificate, as the seam does. The CA file is read from disk, so
// the method lives in this integration file.
func (f seamFixture) client(t *testing.T) *http.Client {
	t.Helper()
	seam, err := outline.TestSeamFromEnvWithFiles(mapEnv(map[string]string{outline.BaseURLEnv: f.base, outline.CAFileEnv: f.ca}), os.ReadFile)
	if err != nil {
		t.Fatal(err)
	}
	return seam.Client
}

// TestGraphQLServeReportsABusyPort checks, over a real socket, that a port another process holds fails the serve command.
func TestGraphQLServeReportsABusyPort(t *testing.T) {
	held, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = held.Close() })
	port := held.Addr().(*net.TCPAddr).Port
	serve := graphqlServe(sources.Service{}, &bytes.Buffer{}, graphql.Listen)
	if err := serve(t.Context(), port, func(string) {}); err == nil {
		t.Fatal("serve on a busy port returned no error")
	}
}

// TestGraphQLServeBindsLoopbackAndStopsWhenCancelled checks, over a real socket, that the serve function binds a loopback
// address, reports it, and returns cleanly once its context ends.
func TestGraphQLServeBindsLoopbackAndStopsWhenCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var addr string
	serve := graphqlServe(sources.Service{}, &bytes.Buffer{}, graphql.Listen)
	err := serve(ctx, 0, func(bound string) {
		addr = bound
		cancel()
	})
	if err != nil {
		t.Fatalf("serve: %v", err)
	}
	if !strings.HasPrefix(addr, "127.0.0.1:") {
		t.Fatalf("the bound address = %q, want a loopback address", addr)
	}
}

// testCreds returns credentials that read an empty env file, so no key is present.
func testCreds(t *testing.T) *envfile.Credentials {
	t.Helper()
	creds, err := openEnvFile(filepath.Join(t.TempDir(), ".env.local"))
	if err != nil {
		t.Fatal(err)
	}
	return creds
}

// TestReadSeamsTurnsOnEachKindFromItsOwnVariables checks that each seam is read from its own variables, that its timeout
// is kept, and that the seams of the other kinds stay off.
func TestReadSeamsTurnsOnEachKindFromItsOwnVariables(t *testing.T) {
	f := newSeamFixture(t)
	env := mapEnv(map[string]string{
		outline.BaseURLEnv: f.base, outline.CAFileEnv: f.ca, outline.TimeoutEnv: "500ms",
	})
	seams, err := readSeams(env)
	if err != nil {
		t.Fatalf("readSeams: %v", err)
	}
	outlineSeam := seams[domain.KindOutline]
	if !outlineSeam.on || outlineSeam.base != f.base || outlineSeam.timeout != 500*time.Millisecond || outlineSeam.client == nil {
		t.Fatalf("outline seam = %+v, want the fixture base, a 500ms timeout, and a client", outlineSeam)
	}
	if seams[domain.KindKiwi].on || seams[domain.KindJira].on {
		t.Fatal("the kiwi and jira seams must stay off when only the outline variables are set")
	}
	if err := checkSeams(seams); err != nil {
		t.Fatalf("checkSeams: %v", err)
	}
}

// TestReadSeamsStopsOnABadSeam checks that one invalid seam of any kind fails the read, so no command starts with a
// seam that could fall back to a real host.
func TestReadSeamsStopsOnABadSeam(t *testing.T) {
	f := newSeamFixture(t)
	cases := map[string]map[string]string{
		"outline base without a CA file": {outline.BaseURLEnv: f.base},
		"kiwi timeout without a base":    {kiwi.TimeoutEnv: "1s"},
		"jira base without a CA file":    {jira.BaseURLEnv: f.base},
		"jira http base":                 {jira.BaseURLEnv: "http://127.0.0.1:1", jira.CAFileEnv: f.ca},
	}
	for name, vars := range cases {
		if _, err := readSeams(mapEnv(vars)); err == nil {
			t.Fatalf("%s: readSeams returned no error", name)
		}
	}
}

// TestCheckSeamsAcceptsAnOnSeamWithAGoodBase checks that a seam that is on with an HTTPS loopback base passes setup.
func TestCheckSeamsAcceptsAnOnSeamWithAGoodBase(t *testing.T) {
	f := newSeamFixture(t)
	seams := seamSet{}
	for _, kind := range seamKinds {
		seams[kind] = seamConfig{on: true, base: f.base}
	}
	if err := checkSeams(seams); err != nil {
		t.Fatalf("checkSeams: %v", err)
	}
}

// TestWireSourcesStopsOnABadSeam checks that a bad seam stops the wiring before any catalog is read.
func TestWireSourcesStopsOnABadSeam(t *testing.T) {
	paths := configpaths.Paths{Catalog: absPath(t, exampleCatalog), EnvFile: filepath.Join(t.TempDir(), ".env.local")}
	seams := seamSet{domain.KindJira: seamConfig{on: true, base: "http://127.0.0.1:1"}}
	if _, err := wireSources(paths, seams, realHost); err == nil {
		t.Fatal("wireSources accepted a bad seam")
	}
}

// TestWireSourcesRefusesARelativePath checks that the loaders refuse a path that is not absolute.
func TestWireSourcesRefusesARelativePath(t *testing.T) {
	absolute := absPath(t, exampleCatalog)
	envPath := filepath.Join(t.TempDir(), ".env.local")
	if _, err := wireSources(configpaths.Paths{Catalog: absolute, EnvFile: "relative.env"}, seamSet{}, realHost); err == nil {
		t.Fatal("wireSources accepted a relative env file path")
	}
	if _, err := wireSources(configpaths.Paths{Catalog: "sources.yml", EnvFile: envPath}, seamSet{}, realHost); err == nil {
		t.Fatal("wireSources accepted a relative catalog path")
	}
}

// TestWireSourcesReportsAMissingCatalog checks that a catalog file that does not exist fails the wiring.
func TestWireSourcesReportsAMissingCatalog(t *testing.T) {
	paths := configpaths.Paths{
		Catalog: filepath.Join(t.TempDir(), "sources.yml"),
		EnvFile: filepath.Join(t.TempDir(), ".env.local"),
	}
	if _, err := wireSources(paths, seamSet{}, realHost); err == nil {
		t.Fatal("wireSources accepted a missing catalog")
	}
}

// TestWireSourcesBuildsAGatewayForEverySourceOfTheExampleCatalog checks that every source kind of the example catalog
// resolves to a gateway through the registry, with every seam off.
func TestWireSourcesBuildsAGatewayForEverySourceOfTheExampleCatalog(t *testing.T) {
	paths := configpaths.Paths{Catalog: absPath(t, exampleCatalog), EnvFile: filepath.Join(t.TempDir(), ".env.local")}
	w, err := wireSources(paths, seamSet{}, realHost)
	if err != nil {
		t.Fatalf("wireSources: %v", err)
	}
	if w.serve == nil || w.mcp == nil {
		t.Fatal("the serve and MCP functions must be wired")
	}
	for _, src := range w.service.Catalog.All() {
		if _, err := w.service.Gateways.For(src); err != nil {
			t.Fatalf("gateway for %s: %v", src.ID(), err)
		}
	}
}

// TestWireSourcesBuildsSeamGatewaysForEveryKind checks that the gateways of every kind build when their seams are on.
func TestWireSourcesBuildsSeamGatewaysForEveryKind(t *testing.T) {
	f := newSeamFixture(t)
	paths := configpaths.Paths{Catalog: absPath(t, exampleCatalog), EnvFile: filepath.Join(t.TempDir(), ".env.local")}
	seams := seamSet{}
	for _, kind := range seamKinds {
		seams[kind] = seamConfig{on: true, base: f.base, client: f.client(t), timeout: time.Second}
	}
	w, err := wireSources(paths, seams, realHost)
	if err != nil {
		t.Fatalf("wireSources: %v", err)
	}
	for _, src := range w.service.Catalog.All() {
		if _, err := w.service.Gateways.For(src); err != nil {
			t.Fatalf("seam gateway for %s: %v", src.ID(), err)
		}
	}
}

// TestBuildGatewaysWithoutASeamUseTheSourceEndpoint checks that a gateway built with the seam off is returned for each kind.
func TestBuildGatewaysWithoutASeamUseTheSourceEndpoint(t *testing.T) {
	creds := testCreds(t)
	byKind := map[domain.SourceKind]domain.Source{}
	for _, src := range exampleSources(t) {
		byKind[src.Kind()] = src
	}
	if gw, err := buildOutline(byKind[domain.KindOutline], creds, seamConfig{}); err != nil || gw == nil {
		t.Fatalf("buildOutline without a seam = %v, %v", gw, err)
	}
	if gw, err := buildKiwi(byKind[domain.KindKiwi], creds, seamConfig{}); err != nil || gw == nil {
		t.Fatalf("buildKiwi without a seam = %v, %v", gw, err)
	}
	if gw, err := buildJira(byKind[domain.KindJira], creds, seamConfig{}); err != nil || gw == nil {
		t.Fatalf("buildJira without a seam = %v, %v", gw, err)
	}
}

// TestBuildGatewaysWithASeamUseTheSeamBase checks that a gateway built with the seam on is returned for each kind, with
// and without a seam timeout.
func TestBuildGatewaysWithASeamUseTheSeamBase(t *testing.T) {
	f := newSeamFixture(t)
	creds := testCreds(t)
	src := exampleSources(t)[0]
	for _, timeout := range []time.Duration{0, time.Second} {
		seam := seamConfig{on: true, base: f.base, client: f.client(t), timeout: timeout}
		if gw, err := buildOutline(src, creds, seam); err != nil || gw == nil {
			t.Fatalf("buildOutline with a seam (timeout %v) = %v, %v", timeout, gw, err)
		}
		if gw, err := buildKiwi(src, creds, seam); err != nil || gw == nil {
			t.Fatalf("buildKiwi with a seam (timeout %v) = %v, %v", timeout, gw, err)
		}
		if gw, err := buildJira(src, creds, seam); err != nil || gw == nil {
			t.Fatalf("buildJira with a seam (timeout %v) = %v, %v", timeout, gw, err)
		}
	}
}

// TestBuildGatewaysRefuseAnOnSeamWithANonHTTPSBase checks that each builder refuses a seam base that is not HTTPS, even
// when it is called without the setup check.
func TestBuildGatewaysRefuseAnOnSeamWithANonHTTPSBase(t *testing.T) {
	creds := testCreds(t)
	src := exampleSources(t)[0]
	bad := seamConfig{on: true, base: "http://127.0.0.1:1"}
	if _, err := buildOutline(src, creds, bad); err == nil {
		t.Fatal("buildOutline accepted an http base")
	}
	if _, err := buildKiwi(src, creds, bad); err == nil {
		t.Fatal("buildKiwi accepted an http base")
	}
	if _, err := buildJira(src, creds, bad); err == nil {
		t.Fatal("buildJira accepted an http base")
	}
}

// TestRunReturnsTheStatusOfTheCommandLine checks the composition root end to end for a command that needs no source:
// --version writes its line and returns 0, with the configuration paths pinned so the test reads no user file.
func TestRunReturnsTheStatusOfTheCommandLine(t *testing.T) {
	getenv := envOf(map[string]string{
		"RETRIEVER_CATALOG":  absPath(t, exampleCatalog),
		"RETRIEVER_ENV_FILE": filepath.Join(t.TempDir(), ".env.local"),
	})
	if status := run([]string{"--version"}, getenv, realHost); status != 0 {
		t.Fatalf("run(--version) = %d, want 0", status)
	}
}

// TestRunStopsWhenASeamVariableIsInvalid checks that an invalid test seam stops the process with status 2 before any
// command runs.
func TestRunStopsWhenASeamVariableIsInvalid(t *testing.T) {
	getenv := envOf(map[string]string{"RETRIEVER_JIRA_TEST_TIMEOUT": "1s"})
	if status := run([]string{"--version"}, getenv, realHost); status != 2 {
		t.Fatalf("run with an invalid seam = %d, want 2", status)
	}
}

// TestMainExitsWithTheStatusOfRun checks the process boundary: the binary exits 0 for --version. The test runs itself
// as a child process, so os.Exit does not end the test run.
func TestMainExitsWithTheStatusOfRun(t *testing.T) {
	if os.Getenv("RETRIEVER_MAIN_CHILD") == "1" {
		os.Args = []string{"retriever", "--version"}
		main()
		return
	}
	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestMainExitsWithTheStatusOfRun$")
	cmd.Env = append(os.Environ(),
		"RETRIEVER_MAIN_CHILD=1",
		"RETRIEVER_CATALOG="+absPath(t, exampleCatalog),
		"RETRIEVER_ENV_FILE="+filepath.Join(t.TempDir(), ".env.local"),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("main child: %v\n%s", err, out)
	}
}

// exampleCatalog is the example catalog the repository ships. Its path is absolute, so a test reads the same file the
// composition root reads with --catalog. Reading it is a real-disk read, so these helpers live in this integration file.
const exampleCatalog = "../../examples/sources.example.yml"

// absPath returns the absolute path of a repository file, so a test can pass it to the path rules.
func absPath(t *testing.T, rel string) string {
	t.Helper()
	abs, err := filepath.Abs(rel)
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

// exampleSources loads the example catalog and returns its sources.
func exampleSources(t *testing.T) []domain.Source {
	t.Helper()
	loader, err := openCatalog(absPath(t, exampleCatalog))
	if err != nil {
		t.Fatal(err)
	}
	cat, err := loader.Load()
	if err != nil {
		t.Fatal(err)
	}
	return cat.All()
}
