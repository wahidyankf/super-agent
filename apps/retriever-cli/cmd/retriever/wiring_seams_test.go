package main

import (
	"errors"
	"io/fs"
	"net/http"
	"testing"
	"testing/fstest"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/configpaths"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/envfile"
)

// seamSourceOf returns one source of the given kind and endpoint, built through the domain constructors. No file is read.
func seamSourceOf(t *testing.T, kind, endpoint, credential string) domain.Source {
	t.Helper()
	id, err := domain.NewSourceID(kind)
	if err != nil {
		t.Fatalf("NewSourceID: %v", err)
	}
	k, err := domain.NewSourceKind(kind)
	if err != nil {
		t.Fatalf("NewSourceKind: %v", err)
	}
	tags, err := domain.NewTagSet()
	if err != nil {
		t.Fatalf("NewTagSet: %v", err)
	}
	ep, err := domain.NewEndpoint(endpoint)
	if err != nil {
		t.Fatalf("NewEndpoint: %v", err)
	}
	cred, err := domain.NewCredentialRef(credential)
	if err != nil {
		t.Fatalf("NewCredentialRef: %v", err)
	}
	return domain.NewSource(id, k, tags, ep, cred)
}

// seamCredentialsOf returns credentials over an absolute env file path. The path is never read, because the builders do not
// read a key until a request needs one.
func seamCredentialsOf(t *testing.T) *envfile.Credentials {
	t.Helper()
	creds, err := envfile.NewWithFiles("/work/project/.env.local", func(string) string { return "" }, envfile.Files{
		FS:       fstest.MapFS{},
		SameFile: func(a, b fs.FileInfo) bool { return false },
	})
	if err != nil {
		t.Fatalf("envfile.NewWithFiles: %v", err)
	}
	return creds
}

// noNetwork is the transport of a fake client. It sends nothing, so a builder that tries a request gets an error.
type noNetwork struct{}

// RoundTrip refuses every request.
func (noNetwork) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("the fake client sends no request")
}

// fakeClient returns an HTTP client whose transport is noNetwork.
func fakeClient() *http.Client { return &http.Client{Transport: noNetwork{}} }

// TestSeamOptionsAreEmptyWithoutATimeout checks that each seam adds a gateway option only when it names a timeout.
func TestSeamOptionsAreEmptyWithoutATimeout(t *testing.T) {
	if seamOptions(seamConfig{}) != nil || kiwiSeamOptions(seamConfig{}) != nil || jiraSeamOptions(seamConfig{}) != nil {
		t.Fatal("a seam without a timeout must add no option")
	}
	timed := seamConfig{timeout: time.Second}
	if len(seamOptions(timed)) != 1 || len(kiwiSeamOptions(timed)) != 1 || len(jiraSeamOptions(timed)) != 1 {
		t.Fatal("a seam with a timeout must add one option")
	}
}

// TestSeamBuildersReturnAGatewayWithTheSeamOff checks that each builder returns the gateway of its own endpoint when its seam
// is off.
func TestSeamBuildersReturnAGatewayWithTheSeamOff(t *testing.T) {
	creds := seamCredentialsOf(t)
	outlineSrc := seamSourceOf(t, "outline", "https://docs.example.test", "EXAMPLE_OUTLINE_API_KEY")
	kiwiSrc := seamSourceOf(t, "kiwi", "https://kiwi.example.test", "EXAMPLE_KIWI_API_KEY")
	jiraSrc := seamSourceOf(t, "jira", "https://jira.example.test", "EXAMPLE_JIRA_API_TOKEN")
	if gw, err := buildOutline(outlineSrc, creds, seamConfig{}); err != nil || gw == nil {
		t.Fatalf("buildOutline with the seam off = %v, %v", gw, err)
	}
	if gw, err := buildKiwi(kiwiSrc, creds, seamConfig{}); err != nil || gw == nil {
		t.Fatalf("buildKiwi with the seam off = %v, %v", gw, err)
	}
	if gw, err := buildJira(jiraSrc, creds, seamConfig{}); err != nil || gw == nil {
		t.Fatalf("buildJira with the seam off = %v, %v", gw, err)
	}
}

// TestSeamBuildersReturnAGatewayWithTheSeamOn checks that each builder uses a seam base and client when its seam is on, with
// and without a seam timeout. The base is a loopback HTTPS address, and the fake client sends no request.
func TestSeamBuildersReturnAGatewayWithTheSeamOn(t *testing.T) {
	creds := seamCredentialsOf(t)
	src := seamSourceOf(t, "outline", "https://docs.example.test", "EXAMPLE_OUTLINE_API_KEY")
	kiwiSrc := seamSourceOf(t, "kiwi", "https://kiwi.example.test", "EXAMPLE_KIWI_API_KEY")
	jiraSrc := seamSourceOf(t, "jira", "https://jira.example.test", "EXAMPLE_JIRA_API_TOKEN")
	for _, timeout := range []time.Duration{0, time.Second} {
		seam := seamConfig{on: true, base: "https://127.0.0.1:9443", client: fakeClient(), timeout: timeout}
		if gw, err := buildOutline(src, creds, seam); err != nil || gw == nil {
			t.Fatalf("buildOutline with a seam (timeout %v) = %v, %v", timeout, gw, err)
		}
		if gw, err := buildKiwi(kiwiSrc, creds, seam); err != nil || gw == nil {
			t.Fatalf("buildKiwi with a seam (timeout %v) = %v, %v", timeout, gw, err)
		}
		if gw, err := buildJira(jiraSrc, creds, seam); err != nil || gw == nil {
			t.Fatalf("buildJira with a seam (timeout %v) = %v, %v", timeout, gw, err)
		}
	}
}

// TestSeamBuildersRefuseAPlainHTTPBase checks that each builder refuses a seam base that is not HTTPS, so a seam can never
// send a key to a plain connection, even when the setup check is skipped.
func TestSeamBuildersRefuseAPlainHTTPBase(t *testing.T) {
	creds := seamCredentialsOf(t)
	src := seamSourceOf(t, "outline", "https://docs.example.test", "EXAMPLE_OUTLINE_API_KEY")
	kiwiSrc := seamSourceOf(t, "kiwi", "https://kiwi.example.test", "EXAMPLE_KIWI_API_KEY")
	jiraSrc := seamSourceOf(t, "jira", "https://jira.example.test", "EXAMPLE_JIRA_API_TOKEN")
	plain := seamConfig{on: true, base: "http://127.0.0.1:1", client: fakeClient()}
	if _, err := buildOutline(src, creds, plain); err == nil {
		t.Fatal("buildOutline accepted a plain HTTP base")
	}
	if _, err := buildKiwi(kiwiSrc, creds, plain); err == nil {
		t.Fatal("buildKiwi accepted a plain HTTP base")
	}
	if _, err := buildJira(jiraSrc, creds, plain); err == nil {
		t.Fatal("buildJira accepted a plain HTTP base")
	}
}

// TestWireSourcesRefusesAnInvalidSeamAndARelativePath checks that the wiring stops before any file is read, when a seam is
// invalid or when a configuration path is not absolute.
func TestWireSourcesRefusesAnInvalidSeamAndARelativePath(t *testing.T) {
	bad := seamSet{domain.KindJira: seamConfig{on: true, base: "http://127.0.0.1:1"}}
	if _, err := wireSources(configpaths.Paths{Catalog: "/work/sources.yml", EnvFile: "/work/.env.local"}, bad, testHost()); err == nil {
		t.Fatal("wireSources accepted an invalid seam")
	}
	if _, err := wireSources(configpaths.Paths{Catalog: "sources.yml", EnvFile: "relative.env"}, seamSet{}, testHost()); err == nil {
		t.Fatal("wireSources accepted a relative env file path")
	}
	if _, err := wireSources(configpaths.Paths{Catalog: "sources.yml", EnvFile: "/work/.env.local"}, seamSet{}, testHost()); err == nil {
		t.Fatal("wireSources accepted a relative catalog path")
	}
}

// TestRunStopsWhenASeamVariableIsInvalidBeforeAnyFileIsRead checks that an invalid seam variable stops the run with status 2,
// before the configuration paths are resolved.
func TestRunStopsWhenASeamVariableIsInvalidBeforeAnyFileIsRead(t *testing.T) {
	getenv := func(key string) string {
		if key == "RETRIEVER_JIRA_TEST_TIMEOUT" {
			return "1s"
		}
		return ""
	}
	if status := run([]string{"--version"}, getenv, testHost()); status != 2 {
		t.Fatalf("run with an invalid seam = %d, want 2", status)
	}
}

// TestExportSinkIsReturnedForAFolder checks that the export sink the command gets is built for the folder it names. No file
// is written by this check.
func TestExportSinkIsReturnedForAFolder(t *testing.T) {
	if exportSink("/work/exports") == nil {
		t.Fatal("exportSink returned no sink")
	}
}
