package main

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/connectorspec"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine/replay"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/specfolder"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// minimalSpec is a valid spec of one read operation, with one example. It is the smallest spec that the run can list and
// read end to end over the replay stub.
const minimalSpec = `version: 1
kind: example-wiki
summary: Reads documents of an example wiki.
endpoint:
  base_path: /v1
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
operations:
  - id: example-wiki.docs.list
    summary: List documents.
    access: read
    method: GET
    path: /docs
    params:
      - name: limit
        type: Limit
        in: query
        required: false
        default: 20
    response:
      items: /items
      fields:
        id:
          pointer: /id
        title:
          pointer: /title
    examples:
      - request:
          limit: 20
        response:
          status: 200
          body:
            items:
              - id: doc-1
                title: First document
        expect:
          id: doc-1
          title: First document
`

// minimalCatalog is a catalog of one source of the minimal spec's kind, whose key comes from the env file.
const minimalCatalog = `version: 1
sources:
  - id: wiki
    kind: example-wiki
    endpoint: https://wiki.example.test
    credential_env: EXAMPLE_WIKI_API_KEY
`

// missingFS is a folder that does not exist: listing it fails with fs.ErrNotExist.
type missingFS struct{}

// Open fails with fs.ErrNotExist for every name.
func (missingFS) Open(string) (fs.File, error) {
	return nil, fs.ErrNotExist
}

// brokenFS is a folder that exists and cannot be read: listing it fails with a fault that is not a missing name.
type brokenFS struct{}

// Open fails with a fault for every name.
func (brokenFS) Open(string) (fs.File, error) {
	return nil, errors.New("the test folder cannot be read")
}

// TestAMissingDefaultFolderLoadsNoSpec checks the default: a connectors folder that the user did not name and that does
// not exist is not an error, and it loads no spec.
func TestAMissingDefaultFolderLoadsNoSpec(t *testing.T) {
	h := hostOver(func() (string, error) { return "/work/project", nil }, fstest.MapFS{}, missingFS{})
	set, err := loadConnectors(h, "/var/example/.config/retriever/connectors", false)
	if err != nil {
		t.Fatalf("loadConnectors: %v", err)
	}
	if len(set.Specs) != 0 || len(set.CLIRows) != 0 {
		t.Fatalf("set = %d specs and %d CLI rows, want none", len(set.Specs), len(set.CLIRows))
	}
}

// TestAMissingNamedFolderIsRefusedWithItsCodeAndNoPath checks an explicit --connectors-dir that does not exist: it is a
// failure with one stable code, and its text names no path.
func TestAMissingNamedFolderIsRefusedWithItsCodeAndNoPath(t *testing.T) {
	h := hostOver(func() (string, error) { return "/work/project", nil }, fstest.MapFS{}, missingFS{})
	_, err := loadConnectors(h, "/work/elsewhere/connectors", true)
	var coded folderError
	if !errors.As(err, &coded) {
		t.Fatalf("err = %v, want a folderError", err)
	}
	var code interface{ ErrorCode() string }
	if !errors.As(err, &code) || code.ErrorCode() != "retriever.connector.folder_unavailable" {
		t.Fatalf("code = %v, want retriever.connector.folder_unavailable", err)
	}
	if strings.Contains(err.Error(), "/work") {
		t.Fatalf("error text %q names a path", err.Error())
	}
}

// TestAnUnreadableFolderIsRefused checks a folder that exists and cannot be read, for the default and the named folder
// alike: it is a folder failure, and it is never taken for an empty folder.
func TestAnUnreadableFolderIsRefused(t *testing.T) {
	h := hostOver(func() (string, error) { return "/work/project", nil }, fstest.MapFS{}, brokenFS{})
	for _, named := range []bool{false, true} {
		_, err := loadConnectors(h, "/work/connectors", named)
		var coded folderError
		if !errors.As(err, &coded) || coded.rule != "cannot be read" {
			t.Fatalf("named=%v: err = %v, want a folderError that cannot be read", named, err)
		}
	}
}

// TestAFileThatIsNotARegularFileIsASpecFailureOfThatFile checks a spec name that is a link: the folder is read, and the
// failure names the file and the rule, and the code is the spec code.
func TestAFileThatIsNotARegularFileIsASpecFailureOfThatFile(t *testing.T) {
	folder := fstest.MapFS{"linked.yml": &fstest.MapFile{Mode: fs.ModeSymlink}}
	h := hostOver(func() (string, error) { return "/work/project", nil }, fstest.MapFS{}, folder)
	_, err := loadConnectors(h, "/work/connectors", true)
	var specErr *connectorspec.Error
	if !errors.As(err, &specErr) {
		t.Fatalf("err = %v, want a connectorspec.Error", err)
	}
	if specErr.File != "linked.yml" || specErr.ErrorCode() != "retriever.connector.spec_invalid" {
		t.Fatalf("file = %q, code = %q; want linked.yml and retriever.connector.spec_invalid", specErr.File, specErr.ErrorCode())
	}
}

// TestAValidSpecLoadsItsKindAndItsCLIRow checks a folder with one valid spec: the run registers its kind, its digest, and
// its command row.
func TestAValidSpecLoadsItsKindAndItsCLIRow(t *testing.T) {
	folder := fstest.MapFS{"example-wiki.yml": &fstest.MapFile{Data: []byte(minimalSpec)}}
	h := hostOver(func() (string, error) { return "/work/project", nil }, fstest.MapFS{}, folder)
	set, err := loadConnectors(h, "/work/connectors", true)
	if err != nil {
		t.Fatalf("loadConnectors: %v", err)
	}
	if len(set.Specs) != 1 || set.Specs[0].Kind.String() != "example-wiki" {
		t.Fatalf("specs = %+v, want one example-wiki spec", set.Specs)
	}
	if _, ok := set.Digests["example-wiki"]; !ok {
		t.Fatalf("digests = %v, want the example-wiki digest", set.Digests)
	}
	if len(set.CLIRows) != 1 || set.CLIRows[0].CLI != "example-wiki docs list" {
		t.Fatalf("CLI rows = %+v, want example-wiki docs list", set.CLIRows)
	}
}

// TestAnInvalidSpecStopsLoadingNamingItsFile checks a spec with a fault: the load stops, and the failure names the file and
// never a value of the spec.
func TestAnInvalidSpecStopsLoadingNamingItsFile(t *testing.T) {
	bad := strings.Replace(minimalSpec, "kind: example-wiki", "kind: outline", 1)
	folder := fstest.MapFS{"bad.yml": &fstest.MapFile{Data: []byte(bad)}}
	h := hostOver(func() (string, error) { return "/work/project", nil }, fstest.MapFS{}, folder)
	_, err := loadConnectors(h, "/work/connectors", true)
	if err == nil || !strings.Contains(err.Error(), "bad.yml") {
		t.Fatalf("err = %v, want a failure that names bad.yml", err)
	}
	if strings.Contains(err.Error(), "Reads documents") {
		t.Fatalf("error text %q carries a value of the spec", err.Error())
	}
}

// TestASpecCommandRunsOverTheReplayStub checks the whole run of a spec command: the catalog names the spec's kind, the
// spec's engine reads through the replay stub, and the command prints the mapped document with exit 0.
func TestASpecCommandRunsOverTheReplayStub(t *testing.T) {
	model := mustSpec(t, minimalSpec)
	stub, err := replay.New(model)
	if err != nil {
		t.Fatalf("replay.New: %v", err)
	}
	root := fstest.MapFS{
		"sources.yml": &fstest.MapFile{Data: []byte(minimalCatalog)},
		"keys.env":    &fstest.MapFile{Data: []byte("EXAMPLE_WIKI_API_KEY=example-key\n")},
	}
	folder := fstest.MapFS{"example-wiki.yml": &fstest.MapFile{Data: []byte(minimalSpec)}}
	h := hostOver(func() (string, error) { return "/work/project", nil }, root, folder)
	h.engine = testEngineSeam()
	h.engine.transport = stub

	status := run([]string{"--catalog", "/work/project/sources.yml", "--env-file", "/work/project/keys.env",
		"--connectors-dir", "/work/connectors", "example-wiki", "docs", "list"}, func(string) string { return "" }, h)
	out := h.stdout.(*bytes.Buffer).String()
	if status != 0 || !strings.Contains(out, "doc-1") {
		t.Fatalf("status = %d, stdout = %q; want 0 and doc-1", status, out)
	}
}

// TestAnExplicitMissingConnectorsFolderExitsTwoWithItsCode checks a run that names a folder which does not exist: the
// sources command exits 2, and the JSON body carries the folder code.
func TestAnExplicitMissingConnectorsFolderExitsTwoWithItsCode(t *testing.T) {
	root := fstest.MapFS{"sources.yml": &fstest.MapFile{Data: []byte(minimalCatalog)}}
	h := hostOver(func() (string, error) { return "/work/project", nil }, root, missingFS{})
	status := run([]string{"--output", "json", "--catalog", "/work/project/sources.yml", "--env-file", "/work/project/keys.env",
		"--connectors-dir", "/work/nope", "sources", "list"}, func(string) string { return "" }, h)
	out := h.stdout.(*bytes.Buffer).String() + h.stderr.(*bytes.Buffer).String()
	if status != 2 || !strings.Contains(out, "retriever.connector.folder_unavailable") {
		t.Fatalf("status = %d, output = %q; want 2 and the folder code", status, out)
	}
}

// TestTheRealClockEndsAContextAndWaitsNoLonger checks the real clock on a context that has already ended: a wait returns at
// once with the context's error, and a time limit gives a context that the release function ends.
func TestTheRealClockEndsAContextAndWaitsNoLonger(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := (realClock{}).Sleep(ctx, time.Hour); !errors.Is(err, context.Canceled) {
		t.Fatalf("Sleep on an ended context = %v, want context.Canceled", err)
	}
	limited, release := (realClock{}).WithTimeout(context.Background(), time.Hour)
	release()
	if limited.Err() == nil {
		t.Fatalf("a released time limit context has no error")
	}
	if err := (realClock{}).Sleep(context.Background(), 0); err != nil {
		t.Fatalf("Sleep for no time = %v, want nil", err)
	}
}

// TestASpecEngineWithoutItsPortsIsRefused checks the builder: an engine that the host gives no clock is refused by the
// engine's own rule, and the refusal reaches the caller.
func TestASpecEngineWithoutItsPortsIsRefused(t *testing.T) {
	folder := fstest.MapFS{"example-wiki.yml": &fstest.MapFile{Data: []byte(minimalSpec)}}
	h := hostOver(func() (string, error) { return "/work/project", nil }, fstest.MapFS{}, folder)
	set, err := loadConnectors(h, "/work/connectors", true)
	if err != nil {
		t.Fatalf("loadConnectors: %v", err)
	}
	kind, err := set.Kinds.Kind("example-wiki")
	if err != nil {
		t.Fatalf("Kind: %v", err)
	}
	src := testSource(t, kind)
	if _, err := buildSpec(set.Specs[0].Model, src, engineSeam{}, nil); !errors.Is(err, engine.ErrNotConfigured) {
		t.Fatalf("buildSpec without a clock = %v, want engine.ErrNotConfigured", err)
	}
}

// testSource returns a source of kind whose endpoint is the example host and whose key variable is the example's.
func testSource(t *testing.T, kind domain.SourceKind) domain.Source {
	t.Helper()
	id, err := domain.NewSourceID("wiki")
	if err != nil {
		t.Fatalf("NewSourceID: %v", err)
	}
	ep, err := domain.NewEndpoint("https://wiki.example.test")
	if err != nil {
		t.Fatalf("NewEndpoint: %v", err)
	}
	ref, err := domain.NewCredentialRef("EXAMPLE_WIKI_API_KEY")
	if err != nil {
		t.Fatalf("NewCredentialRef: %v", err)
	}
	tags, _ := domain.NewTagSet()
	return domain.NewSource(id, kind, tags, ep, ref)
}

// TestTheSpecFileCapIsTheSpecModuleCap checks that the folder reader caps each file at the spec module's size, so the two
// never disagree about how large a spec may be.
func TestTheSpecFileCapIsTheSpecModuleCap(t *testing.T) {
	big := strings.Repeat("a", spec.MaxBytes+1)
	folder := fstest.MapFS{"big.yml": &fstest.MapFile{Data: []byte(big)}}
	_, err := specfolder.Read(folder, spec.MaxBytes)
	var file specfolder.FileError
	if !errors.As(err, &file) {
		t.Fatalf("Read over the cap = %v, want a FileError", err)
	}
}

// mustSpec validates src and returns its model, failing the test when the spec is refused.
func mustSpec(t *testing.T, src string) *spec.Spec {
	t.Helper()
	res := spec.Validate([]byte(src))
	if !res.Valid() {
		t.Fatalf("spec refused: %v", res.Findings)
	}
	return res.Spec
}
