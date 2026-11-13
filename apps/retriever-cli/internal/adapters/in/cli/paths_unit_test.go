package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/ports"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/sources"
)

// The tests in this file reach the branches that the command-line tests do not: each one calls a function directly, or runs
// one command line that no other test runs, and checks the status and the message it produces.

const (
	pathsDocumentID = "d0c1d0c1-0000-4000-8000-000000000001"
	pathsMissingID  = "d0c9d0c9-0000-4000-8000-000000000009"
)

func TestExitStatusWithoutAnErrorNamesItsCode(t *testing.T) {
	bare := exitStatus{code: 1}
	if bare.Error() != "exit status 1" || bare.Unwrap() != nil {
		t.Fatalf("a status without an error must name its code, got %q", bare.Error())
	}
	cause := errors.New("the search matched nothing")
	wrapped := exitStatus{code: 2, err: cause}
	if wrapped.Error() != cause.Error() || !errors.Is(wrapped, cause) {
		t.Fatalf("a status with an error must return that error, got %q", wrapped.Error())
	}
}

func TestUsageErrorIsItsMessage(t *testing.T) {
	if got := (usageError{msg: "port must be between 0 and 65535"}).Error(); got != "port must be between 0 and 65535" {
		t.Fatalf("usage error text = %q", got)
	}
}

func TestSignalCauseNamesTheSignal(t *testing.T) {
	if got := SignalCause(os.Interrupt).Error(); got != "received interrupt" {
		t.Fatalf("signal cause text = %q, want the signal name", got)
	}
}

func TestOrDiscardOfNilDiscardsEverything(t *testing.T) {
	if orDiscard(nil) != io.Discard {
		t.Fatal("a nil writer must become io.Discard")
	}
}

func TestRunWithoutAClockOrContextStillRuns(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"--version"}, Deps{Stdout: &out, Stderr: &errOut}); code != 0 {
		t.Fatalf("version with no clock or context = %d, stderr %q", code, errOut.String())
	}
}

func TestAnOutputOptionWithoutItsValueIsAUsageError(t *testing.T) {
	var out, errOut bytes.Buffer
	code := Run([]string{"--output"}, Deps{Stdout: &out, Stderr: &errOut})
	if code != exitFailed || out.Len() != 0 || !strings.Contains(errOut.String(), "missing value for --output") {
		t.Fatalf("want a missing-value usage error, got %q (exit %d)", errOut.String(), code)
	}
}

func TestASwitchNeverTakesAValue(t *testing.T) {
	code, out, errOut := runRetriever(t, []string{"outline", "documents", "delete", pathsDocumentID, "--yes=true"}, outlineSetup{})
	if code != exitFailed || out != "" || !strings.Contains(errOut, "--yes does not take a value") {
		t.Fatalf("want a switch-with-value usage error, got %q (exit %d)", errOut, code)
	}
}

func TestAValuedFlagAtTheEndNamesItsMissingValue(t *testing.T) {
	code, _, errOut := runRetriever(t, []string{"serve", "--port"}, outlineSetup{})
	if code != exitFailed || !strings.Contains(errOut, "missing value for --port") {
		t.Fatalf("want a missing-value error for --port, got %q (exit %d)", errOut, code)
	}
}

func TestAnExtraPositionalArgumentIsUnrecognized(t *testing.T) {
	code, _, errOut := runRetriever(t, []string{"document", "get", pathsDocumentID, "extra"}, outlineSetup{})
	if code != exitFailed || !strings.Contains(errOut, "unrecognized argument: extra") {
		t.Fatalf("want an unrecognized-argument error for the extra word, got %q (exit %d)", errOut, code)
	}
}

func TestDoubleDashPassesTheRestAsPositionalArguments(t *testing.T) {
	l := &leaf{positional: []string{"first", "second"}}
	inv, err := l.parse([]string{"--", "--not-a-flag", "value"})
	if err != nil {
		t.Fatalf("parse after -- failed: %v", err)
	}
	if len(inv.positional) != 2 || inv.positional[0] != "--not-a-flag" || inv.positional[1] != "value" {
		t.Fatalf("positional arguments after -- = %q", inv.positional)
	}
}

func TestGlobalsAfterDoubleDashStayInTheCommandLine(t *testing.T) {
	g, rest, err := splitGlobals([]string{"--output", "json", "--", "--version"})
	if err != nil {
		t.Fatalf("splitGlobals failed: %v", err)
	}
	if g.version || g.output != "json" || len(rest) != 2 || rest[0] != "--" || rest[1] != "--version" {
		t.Fatalf("after -- the rest must pass through unread: globals %+v, rest %q", g, rest)
	}
}

func TestAGlobalWithoutItsValueAtTheEndIsAnError(t *testing.T) {
	if _, _, err := splitGlobals([]string{"--catalog"}); err == nil || err.Error() != "missing value for --catalog" {
		t.Fatalf("want a missing-value error for --catalog, got %v", err)
	}
}

func TestAGroupThatIsAlsoALeafListsItsSubcommands(t *testing.T) {
	code, out, _ := runRetriever(t, []string{"outline", "collections", "--help"}, outlineSetup{})
	if code != exitOK || !strings.Contains(out, "SUBCOMMANDS:") {
		t.Fatalf("help of a leaf with children must list them, got %q (exit %d)", out, code)
	}
}

func TestUsageCodeNamesTheReasonOfEachUsageMessage(t *testing.T) {
	cases := []struct {
		message, want string
	}{
		{"unrecognized argument: x", "retriever.outline.usage_unrecognized_argument"},
		{"missing arguments: <id>", "retriever.outline.usage_missing_argument"},
		{"--output must be text or json", "retriever.outline.usage_invalid_output"},
		{"--color must be auto, always, or never", "retriever.outline.usage_invalid_color"},
		{"no valid subcommand has been specified", "retriever.outline.usage_no_subcommand"},
		{"port must be between 0 and 65535", "retriever.outline.usage_invalid_argument"},
	}
	for _, c := range cases {
		if got := usageCode("outline", c.message); got != c.want {
			t.Errorf("usageCode(%q) = %q, want %q", c.message, got, c.want)
		}
	}
}

func TestPanicInTextModeNamesTheCodeWithoutItsText(t *testing.T) {
	setup := outlineSetup{serve: func(context.Context, int, func(string)) error { panic("internal detail") }}
	code, out, errOut := runRetriever(t, []string{"serve"}, setup)
	if code != exitFailed || out != "" || !strings.Contains(errOut, "(retriever.panic)") || strings.Contains(errOut, "internal detail") {
		t.Fatalf("want the text internal failure with no panic text, got %q (exit %d)", errOut, code)
	}
}

func TestServeWithoutAServerIsNotConfigured(t *testing.T) {
	var out, errOut bytes.Buffer
	code := Run([]string{"serve"}, Deps{Stdout: &out, Stderr: &errOut})
	if code != exitFailed || out.Len() != 0 || !strings.Contains(errOut.String(), "not configured") {
		t.Fatalf("serve without a server must be not configured, got %q (exit %d)", errOut.String(), code)
	}
}

// TestMCPEndedBySignalNamesTheSignal cancels the session's parent with a signal cause, then returns from the session.
// Whichever arm of the select wins, the status must name the signal.
func TestMCPEndedBySignalNamesTheSignal(t *testing.T) {
	cases := []struct {
		sig  os.Signal
		want int
	}{
		{os.Interrupt, mcpExitInterrupt},
		{syscall.SIGTERM, mcpExitTerm},
	}
	for _, c := range cases {
		parent, cancel := context.WithCancelCause(context.Background())
		var errOut bytes.Buffer
		code := Run([]string{"mcp"}, Deps{
			Context: parent,
			Stdin:   strings.NewReader(""),
			Stdout:  io.Discard,
			Stderr:  &errOut,
			MCP: func(_ context.Context, _ domain.Access, _ []string, _ io.Reader, _ io.Writer) error {
				cancel(SignalCause(c.sig))
				return nil
			},
		})
		cancel(nil)
		if code != c.want {
			t.Fatalf("session ended by %v = %d, want %d (stderr %q)", c.sig, code, c.want, errOut.String())
		}
	}
}

func TestMCPCancelledWithoutASignalExitsTwo(t *testing.T) {
	parent, cancel := context.WithCancelCause(context.Background())
	var errOut bytes.Buffer
	code := Run([]string{"mcp"}, Deps{
		Context: parent,
		Stdin:   strings.NewReader(""),
		Stdout:  io.Discard,
		Stderr:  &errOut,
		MCP: func(ctx context.Context, _ domain.Access, _ []string, _ io.Reader, _ io.Writer) error {
			cancel(errors.New("the parent was dropped"))
			<-ctx.Done()
			return nil
		},
	})
	cancel(nil)
	if code != mcpExitNotDone || strings.Contains(errOut.String(), "the parent was dropped") {
		t.Fatalf("a cancelled session without a signal must exit 2 and keep the cause text private, got %q (exit %d)", errOut.String(), code)
	}
}

func TestOutlineWiringWithoutGatewaysIsNotConfigured(t *testing.T) {
	_, err := (&runtime{}).outlineWiring()
	if !errors.Is(err, errNotConfigured) {
		t.Fatalf("wiring with no gateways = %v, want not configured", err)
	}
}

func TestExportWithoutASinkIsNotConfigured(t *testing.T) {
	setup, _ := wikiWith(treeDataset)
	setup.sink = nil
	code, out, errOut := runRetriever(t, []string{"outline", "export", pathsDocumentID, "--out", "exported"}, setup)
	if code != exitFailed || out != "" || !strings.Contains(errOut, "not configured") {
		t.Fatalf("export without a sink must be not configured, got %q (exit %d)", errOut, code)
	}
}

func TestDocumentGetRejectsAnUnknownKind(t *testing.T) {
	code, out, errOut := runRetriever(t, []string{"document", "get", pathsDocumentID, "--kind", "nope"}, outlineSetup{})
	if code != exitFailed || out != "" || errOut == "" {
		t.Fatalf("an unknown kind must be a usage failure, got %q (exit %d)", errOut, code)
	}
}

func TestDocumentGetWithoutWiringNamesTheSetupFailure(t *testing.T) {
	setup := outlineSetup{setupErr: errors.New("the wiring is missing")}
	code, out, errOut := runRetriever(t, []string{"document", "get", pathsDocumentID, "--kind", "outline"}, setup)
	if code != exitFailed || out != "" || !strings.Contains(errOut, "the wiring is missing") {
		t.Fatalf("document get without wiring must name the setup failure, got %q (exit %d)", errOut, code)
	}
}

func TestDocumentGetOfAMissingDocumentFails(t *testing.T) {
	setup, _ := wikiWith(treeDataset)
	code, out, errOut := runRetriever(t, []string{"document", "get", pathsMissingID}, setup)
	if code != exitFailed || out != "" || errOut == "" {
		t.Fatalf("a missing document must fail with no standard output, got %q (exit %d)", errOut, code)
	}
}

func TestWriteWithoutWiringNamesTheSetupFailure(t *testing.T) {
	setup := outlineSetup{setupErr: errors.New("the wiring is missing")}
	code, out, errOut := runRetriever(t, []string{"outline", "documents", "update", pathsDocumentID,
		"--title", "Renamed", "--source", "example-outline"}, setup)
	if code != exitFailed || out != "" || !strings.Contains(errOut, "the wiring is missing") {
		t.Fatalf("a write without wiring must name the setup failure, got %q (exit %d)", errOut, code)
	}
}

func TestAdminWithoutWiringNamesTheSetupFailure(t *testing.T) {
	setup := outlineSetup{setupErr: errors.New("the wiring is missing")}
	code, out, errOut := runRetriever(t, []string{"outline", "documents", "delete", pathsDocumentID,
		"--permanent", "true", "--yes", "--confirm", pathsDocumentID, "--source", "example-outline"}, setup)
	if code != exitFailed || out != "" || !strings.Contains(errOut, "the wiring is missing") {
		t.Fatalf("an admin call without wiring must name the setup failure, got %q (exit %d)", errOut, code)
	}
}

// writableWiki returns the wiki setup with its one source granted write access, so a write command reaches its reply.
func writableWiki(t *testing.T) outlineSetup {
	t.Helper()
	setup, gw := wikiWith(treeDataset)
	id, err := domain.NewSourceID("outline")
	if err != nil {
		t.Fatal(err)
	}
	ep, err := domain.NewEndpoint("https://docs.example.test")
	if err != nil {
		t.Fatal(err)
	}
	ref, err := domain.NewCredentialRef("EXAMPLE_OUTLINE_API_KEY")
	if err != nil {
		t.Fatal(err)
	}
	tags, err := domain.NewTagSet()
	if err != nil {
		t.Fatal(err)
	}
	cat, err := domain.NewCatalog([]domain.Source{
		domain.NewSource(id, domain.KindOutline, tags, domain.AccessWrite, ep, ref),
	})
	if err != nil {
		t.Fatal(err)
	}
	setup.service = sources.Service{Catalog: cat, Gateways: fixedResolver{gw: gw}}
	return setup
}

func TestJSONWritePrintsOneDocument(t *testing.T) {
	setup := writableWiki(t)
	code, out, errOut := runRetriever(t, []string{"--output", "json", "outline", "documents", "update", pathsDocumentID,
		"--title", "Renamed", "--dry-run", "--source", "outline"}, setup)
	if code != exitOK || !strings.Contains(out, `"schemaVersion":1`) {
		t.Fatalf("a JSON write must print one document, got %q (stderr %q, exit %d)", out, errOut, code)
	}
}

func TestSourceRowsAsJSONAreNeverNull(t *testing.T) {
	rows := sourceRowsJSON([]domain.SourceSummary{{ID: "example-outline", Kind: "outline", Access: "read"}})
	if len(rows) != 1 || rows[0].Tags == nil || len(rows[0].Tags) != 0 {
		t.Fatalf("an untagged source must read as an empty list, got %+v", rows)
	}
}

func TestSecretLookupReadsEveryKeyTheToolAccepts(t *testing.T) {
	values := map[string]string{
		"EXAMPLE_OUTLINE_API_KEY": "synthetic-outline",
		"EXAMPLE_KIWI_API_KEY":    "synthetic-kiwi",
		"EXAMPLE_JIRA_API_TOKEN":  "synthetic-token",
		"EXAMPLE_JIRA_EMAIL":      "synthetic@example.test",
	}
	lookup := SecretLookup(envOf(values))
	for key, value := range values {
		if got := lookup(key); got != value {
			t.Errorf("SecretLookup(%s) = %q, want the set value", key, got)
		}
	}
	if got := lookup("EXAMPLE_NOT_A_KEY"); got != "" {
		t.Errorf("an unknown key must read as empty, got %q", got)
	}
}

func TestOutlineLeafPathsNameEveryOutlineLeaf(t *testing.T) {
	paths := OutlineLeafPaths()
	found := false
	for _, p := range paths {
		if !strings.HasPrefix(p, "outline ") {
			t.Fatalf("leaf path %q is outside the outline group", p)
		}
		if p == "outline documents get" {
			found = true
		}
	}
	if !found {
		t.Fatalf("the outline leaf paths must include outline documents get, got %q", paths)
	}
}

func TestUnavailableOperationHasItsOwnTextAndCode(t *testing.T) {
	unavailable := domain.OperationUnavailableError{Method: "documents.example"}
	if got := publicMessage(unavailable); !strings.Contains(got, "documents.example") {
		t.Fatalf("public text of an unavailable operation = %q, want the method", got)
	}
	if got := errorCode(unavailable, "outline"); got != "retriever.outline.operation_unavailable" {
		t.Fatalf("code of an unavailable operation = %q", got)
	}
}

func TestRefusedAndCatalogFailuresHaveTheirOwnCodes(t *testing.T) {
	if got := errorCode(refusedError{err: errors.New("the port is busy")}, "serve"); got != "retriever.serve.refused" {
		t.Fatalf("code of a refused setup = %q", got)
	}
	if got := errorCode(domain.CatalogError{Message: "the catalog is malformed"}, "sources"); got != "retriever.sources.catalog_invalid" {
		t.Fatalf("code of a catalog failure = %q", got)
	}
}

// refusingWriter fails every write, as a full disk or a closed file does.
type refusingWriter struct{}

func (refusingWriter) Write([]byte) (int, error) { return 0, errors.New("the disk is full") }

// TestAFailedWriteOfTheReplyIsAFailure checks that a reply the stream refuses ends the command with status 2, and that
// the failure body names the failure without its text.
func TestAFailedWriteOfTheReplyIsAFailure(t *testing.T) {
	setup, _ := wikiWith(collectionsOf("Synthetic"))
	var errOut bytes.Buffer
	code := Run([]string{"--output", "json", "outline", "collections"}, Deps{
		Ports:   ports.Ports{FS: stubFS{}},
		Stdout:  refusingWriter{},
		Stderr:  &errOut,
		Getenv:  func(string) string { return "" },
		Now:     func() time.Time { return fixedNow },
		Service: setup.service,
	})
	if code != exitFailed || !strings.Contains(errOut.String(), `"code":"retriever.outline.failed"`) || strings.Contains(errOut.String(), "disk is full") {
		t.Fatalf("a refused reply must be a failure with no stream text, got %q (exit %d)", errOut.String(), code)
	}
}
