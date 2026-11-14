package main

import (
	"testing"
	"testing/fstest"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/in/cli"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/catalog"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/configpaths"
)

// exampleCatalogYAML is a catalog with one source of each kind, all on example hosts.
const exampleCatalogYAML = `version: 1
sources:
  - id: docs
    kind: outline
    endpoint: https://docs.example.test
    credential_env: EXAMPLE_OUTLINE_API_KEY
  - id: wiki
    kind: kiwi
    endpoint: https://wiki.example.test
    credential_env: EXAMPLE_KIWI_API_KEY
  - id: tracker
    kind: jira
    endpoint: https://tracker.example.test
    credential_env: EXAMPLE_JIRA_API_KEY
    identity_env: EXAMPLE_JIRA_EMAIL
`

// memoryHost returns the unit-test host whose catalog file holds exampleCatalogYAML in an in-memory folder.
func memoryHost() host {
	files := fstest.MapFS{"sources.yml": &fstest.MapFile{Data: []byte(exampleCatalogYAML)}}
	h := testHost()
	h.openCatalog = func(path string) (*catalog.Loader, error) {
		return catalog.NewWithFiles(path, catalog.Files{FS: files, SameFile: sameName})
	}
	return h
}

// examplePaths are the configuration paths of memoryHost.
var examplePaths = configpaths.Paths{Catalog: "/work/project/sources.yml", EnvFile: "/work/project/.env.local"}

// TestRunReturnsZeroForVersionWithFakes checks the success path of run: the command line is parsed, the wiring runs over
// fakes, and the version line goes to the host's stdout.
func TestRunReturnsZeroForVersionWithFakes(t *testing.T) {
	if status := run([]string{"--version"}, emptyEnv, testHost()); status != 0 {
		t.Fatalf("run(--version) = %d, want 0", status)
	}
}

// TestAttachSourcesWiresTheCatalogOfMemory checks that attachSourcesFrom resolves the paths and attaches the services.
func TestAttachSourcesWiresTheCatalogOfMemory(t *testing.T) {
	var deps cli.Deps
	globals := cli.Globals{Catalog: examplePaths.Catalog, EnvFile: examplePaths.EnvFile}
	if err := attachSourcesFrom(&deps, globals, seamSet{}, emptyEnv, memoryHost()); err != nil {
		t.Fatalf("attachSourcesFrom error = %v, want nil", err)
	}
	if deps.Serve == nil || deps.MCP == nil {
		t.Fatal("attachSourcesFrom left the serve or the MCP function unset")
	}
}

// TestEveryKindOfTheCatalogBuildsItsGateway checks each gateway builder: the resolver builds a gateway for each source of
// the example catalog, with the seams off.
func TestEveryKindOfTheCatalogBuildsItsGateway(t *testing.T) {
	w, err := wireSources(examplePaths, seamSet{}, memoryHost())
	if err != nil {
		t.Fatalf("wireSources error = %v, want nil", err)
	}
	all := w.service.Catalog.All()
	if len(all) != 3 {
		t.Fatalf("catalog holds %d sources, want 3", len(all))
	}
	for _, src := range all {
		if _, err := w.service.Gateways.For(src); err != nil {
			t.Fatalf("gateway for %s: %v", src.ID(), err)
		}
	}
}

// TestWireSourcesReportsAFailingEnvFileOpen checks the error path of the env file open.
func TestWireSourcesReportsAFailingEnvFileOpen(t *testing.T) {
	h := memoryHost()
	h.openEnvFile = failingEnvFile
	if _, err := wireSources(examplePaths, seamSet{}, h); err == nil {
		t.Fatal("wireSources accepted a failing env file open")
	}
}

// TestAttachSourcesReportsAWireFailure checks that a wiring failure is returned to the CLI as the setup error.
func TestAttachSourcesReportsAWireFailure(t *testing.T) {
	var deps cli.Deps
	h := testHost()
	globals := cli.Globals{Catalog: examplePaths.Catalog, EnvFile: examplePaths.EnvFile}
	if err := attachSourcesFrom(&deps, globals, seamSet{}, emptyEnv, h); err == nil {
		t.Fatal("attachSourcesFrom returned no setup error for a missing catalog")
	}
}

// TestAttachSourcesReportsAPathFailure checks that a failure to resolve the paths is returned before any file is opened.
func TestAttachSourcesReportsAPathFailure(t *testing.T) {
	var deps cli.Deps
	h := testHostWithGetwd(func() (string, error) { return "", errFakeGetwd })
	if err := attachSourcesFrom(&deps, cli.Globals{}, seamSet{}, emptyEnv, h); err == nil {
		t.Fatal("attachSourcesFrom returned no error when the working directory cannot be read")
	}
}
