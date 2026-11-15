package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"syscall"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// errTestUnreadable is the ReadFile failure of the unreadable-spec test.
var errTestUnreadable = errors.New("the test refuses this read")

// connectorWikiSpec is the example wiki spec the connector commands check: one
// list operation with one example. Its endpoint suffix and names are synthetic.
const connectorWikiSpec = `version: 1
kind: example-wiki
summary: Reads documents of an example wiki.
endpoint:
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
    response:
      items: /items
      fields:
        id:
          pointer: /id
        title:
          pointer: /title
    examples:
      - request: {}
        response:
          status: 200
          body:
            items:
              - id: doc-1
                title: First document
        expect:
          id: doc-1
          title: First document
  - id: example-wiki.docs.search
    summary: Search documents.
    access: read
    method: GET
    path: /search
    mcp: false
    params:
      - name: query
        type: Query
        in: query
        required: true
    response:
      items: /items
      fields:
        id:
          pointer: /id
    examples:
      - request:
          query: draft
        response:
          status: 200
          body:
            items:
              - id: doc-2
        expect:
          id: doc-2
`

// connectorNoExampleSpec is the wiki spec with the examples of its operation
// removed, so validate must refuse it naming the operation.
const connectorNoExampleSpec = `version: 1
kind: example-wiki
summary: Reads documents of an example wiki.
endpoint:
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
    response:
      items: /items
      fields:
        id:
          pointer: /id
        title:
          pointer: /title
`

// mustModel parses a spec document for the secrets tests, failing the test
// when the document is refused.
func mustModel(t *testing.T, doc string) *spec.Spec {
	t.Helper()
	res := spec.Parse([]byte(doc))
	if !res.Valid() {
		t.Fatalf("spec is refused: %v", res.Findings)
	}
	return res.Spec
}

// runConnector runs argv with the spec files given, and returns the exit
// status with the output. files maps the file name to its bytes; a name that
// is missing reads as a failure.
func runConnector(t *testing.T, argv []string, files map[string]string, readErr error, env map[string]string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	deps := Deps{
		Stdout: &stdout,
		Stderr: &stderr,
		Getenv: func(name string) string {
			return env[name]
		},
		ReadFile: func(path string) ([]byte, error) {
			if readErr != nil {
				return nil, readErr
			}
			data, ok := files[path]
			if !ok {
				t.Fatalf("ReadFile(%q) reads a file the test did not plant", path)
			}
			return []byte(data), nil
		},
	}
	code := Run(argv, deps)
	return code, stdout.String(), stderr.String()
}

// TestConnectorValidateAcceptsASpec checks the quiet path: validate prints its
// report to standard output and exits 0, with nothing on standard error.
func TestConnectorValidateAcceptsASpec(t *testing.T) {
	code, out, errOut := runConnector(t, []string{"connector", "validate", "wiki.yml"},
		map[string]string{"wiki.yml": connectorWikiSpec}, nil, nil)
	if code != 0 {
		t.Fatalf("code = %d, stderr = %q; want 0", code, errOut)
	}
	if !strings.Contains(out, "validate: ok") || errOut != "" {
		t.Fatalf("stdout = %q, stderr = %q; want the report on stdout only", out, errOut)
	}
}

// TestConnectorValidateNamesAnOperationWithoutAnExample pins gate 1: an
// operation with no example exits 2 naming the operation, with the diagnostics
// on standard error and nothing on standard output.
func TestConnectorValidateNamesAnOperationWithoutAnExample(t *testing.T) {
	code, out, errOut := runConnector(t, []string{"connector", "validate", "wiki.yml"},
		map[string]string{"wiki.yml": connectorNoExampleSpec}, nil, nil)
	if code != 2 {
		t.Fatalf("code = %d, stderr = %q; want 2", code, errOut)
	}
	if out != "" {
		t.Fatalf("stdout = %q, want empty on a failed run", out)
	}
	if !strings.Contains(errOut, "example-wiki.docs.list") {
		t.Fatalf("stderr = %q, want the operation named", errOut)
	}
}

// TestConnectorValidateReportsAnUnreadableSpec pins the file contract: a spec
// file that cannot be read exits 2, names the base file only, and leaves
// standard output empty.
func TestConnectorValidateReportsAnUnreadableSpec(t *testing.T) {
	code, out, errOut := runConnector(t, []string{"connector", "validate", "/work/specs/wiki.yml"},
		nil, errTestUnreadable, nil)
	if code != 2 {
		t.Fatalf("code = %d, stderr = %q; want 2", code, errOut)
	}
	if out != "" {
		t.Fatalf("stdout = %q, want empty on a failed run", out)
	}
	if !strings.Contains(errOut, "wiki.yml") || strings.Contains(errOut, "/work/specs") {
		t.Fatalf("stderr = %q, want the base name and no path", errOut)
	}
}

// TestConnectorTestReplaysTheExamples checks the quiet path of test: every
// example replays, the report prints the derived texts, and the run exits 0.
func TestConnectorTestReplaysTheExamples(t *testing.T) {
	code, out, errOut := runConnector(t, []string{"connector", "test", "wiki.yml"},
		map[string]string{"wiki.yml": connectorWikiSpec}, nil, nil)
	if code != 0 {
		t.Fatalf("code = %d, stderr = %q; want 0", code, errOut)
	}
	for _, want := range []string{"validate: ok", "replay: ok", "cli:", "mcp:", "readme:"} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout = %q, want %q", out, want)
		}
	}
	if errOut != "" {
		t.Fatalf("stderr = %q, want empty on a passing run", errOut)
	}
}

// TestConnectorTestNamesAnExampleThatExpectsAnUnproducedField pins gate 2: an
// example that expects a declared field the reply does not produce exits 2
// naming the operation, the example, and the field, with nothing on standard
// output.
func TestConnectorTestNamesAnExampleThatExpectsAnUnproducedField(t *testing.T) {
	doc := strings.Replace(connectorWikiSpec,
		"            items:\n              - id: doc-1\n                title: First document\n",
		"            items:\n              - id: doc-1\n", 1)
	code, out, errOut := runConnector(t, []string{"connector", "test", "wiki.yml"},
		map[string]string{"wiki.yml": doc}, nil, nil)
	if code != 2 {
		t.Fatalf("code = %d, stderr = %q; want 2", code, errOut)
	}
	if out != "" {
		t.Fatalf("stdout = %q, want empty on a failed run", out)
	}
	for _, want := range []string{"example-wiki.docs.list", "example 0", "title"} {
		if !strings.Contains(errOut, want) {
			t.Fatalf("stderr = %q, want %q named", errOut, want)
		}
	}
}

// TestConnectorTestRedactsAPlantedCredential pins gate 3: a credential value
// planted in a canned reply reads [REDACTED] in the report, and never in the
// clear.
func TestConnectorTestRedactsAPlantedCredential(t *testing.T) {
	const planted = "planted-credential-value"
	doc := strings.Replace(connectorWikiSpec, "First document", planted, 2)
	code, out, _ := runConnector(t, []string{"connector", "test", "wiki.yml"},
		map[string]string{"wiki.yml": doc}, nil, map[string]string{"EXAMPLE_WIKI_API_KEY": planted})
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if strings.Contains(out, planted) {
		t.Fatalf("stdout holds the planted credential in the clear")
	}
	if !strings.Contains(out, "[REDACTED]") {
		t.Fatalf("stdout = %q, want [REDACTED]", out)
	}
}

// TestConnectorFailureBodyHoldsTheCommandAndCode pins the failure contract in
// JSON mode: the diagnostics go to standard error as the stable failure body,
// and standard output stays empty.
func TestConnectorFailureBodyHoldsTheCommandAndCode(t *testing.T) {
	code, out, errOut := runConnector(t, []string{"--output", "json", "connector", "validate", "wiki.yml"},
		map[string]string{"wiki.yml": connectorNoExampleSpec}, nil, nil)
	if code != 2 {
		t.Fatalf("code = %d, stderr = %q; want 2", code, errOut)
	}
	if out != "" {
		t.Fatalf("stdout = %q, want empty on a failed run", out)
	}
	var body struct {
		SchemaVersion int    `json:"schemaVersion"`
		Command       string `json:"command"`
		ExitCode      int    `json:"exitCode"`
		Error         struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(errOut), &body); err != nil {
		t.Fatalf("stderr = %q, want the JSON failure body: %v", errOut, err)
	}
	if body.SchemaVersion != 1 || body.Command != "retriever connector validate" || body.ExitCode != 2 {
		t.Fatalf("body = %+v, want schema 1, the command, and exit 2", body)
	}
	if !strings.HasPrefix(body.Error.Code, "retriever.connector.") {
		t.Fatalf("code = %q, want the connector area", body.Error.Code)
	}
}

// TestConnectorSuccessJSONIsAVersionedDocument pins the success shape in JSON
// mode: one versioned document on standard output.
func TestConnectorSuccessJSONIsAVersionedDocument(t *testing.T) {
	code, out, errOut := runConnector(t, []string{"--output", "json", "connector", "validate", "wiki.yml"},
		map[string]string{"wiki.yml": connectorWikiSpec}, nil, nil)
	if code != 0 {
		t.Fatalf("code = %d, stderr = %q; want 0", code, errOut)
	}
	var body struct {
		SchemaVersion int    `json:"schemaVersion"`
		Status        string `json:"status"`
	}
	if err := json.Unmarshal([]byte(out), &body); err != nil {
		t.Fatalf("stdout = %q, want JSON: %v", out, err)
	}
	if body.SchemaVersion != 1 || body.Status != "pass" || errOut != "" {
		t.Fatalf("stdout = %q, stderr = %q; want a passing versioned document", out, errOut)
	}
}

// epipeWriter is a writer whose writes meet a closed reader, so the run ends
// as a broken pipe does.
type epipeWriter struct{}

// Write fails every write with EPIPE.
func (epipeWriter) Write([]byte) (int, error) { return 0, syscall.EPIPE }

// TestConnectorBrokenPipeExits141 pins the pipe contract for the new commands:
// output with no reader ends the run with status 141, as the shell reports
// SIGPIPE.
func TestConnectorBrokenPipeExits141(t *testing.T) {
	var stderr bytes.Buffer
	deps := Deps{
		Stdout:   epipeWriter{},
		Stderr:   &stderr,
		Getenv:   func(string) string { return "" },
		ReadFile: func(string) ([]byte, error) { return []byte(connectorWikiSpec), nil },
	}
	if code := Run([]string{"connector", "validate", "wiki.yml"}, deps); code != 141 {
		t.Fatalf("code = %d, want 141", code)
	}
}

// TestConnectorTestRefusesABuiltinKind pins the user-folder rule at the
// command: a spec of a built-in kind exits 2 naming the kind rule, with
// nothing on standard output.
func TestConnectorTestRefusesABuiltinKind(t *testing.T) {
	doc := strings.Replace(connectorWikiSpec, "kind: example-wiki", "kind: outline", 1)
	doc = strings.ReplaceAll(doc, "example-wiki.", "outline.")
	code, out, errOut := runConnector(t, []string{"connector", "test", "outline.yml"},
		map[string]string{"outline.yml": doc}, nil, nil)
	if code != 2 {
		t.Fatalf("code = %d, stderr = %q; want 2", code, errOut)
	}
	if out != "" {
		t.Fatalf("stdout = %q, want empty on a failed run", out)
	}
	if !strings.Contains(errOut, "is the kind of a built-in source") {
		t.Fatalf("stderr = %q, want the kind rule", errOut)
	}
}

// TestConnectorValidateRefusesAnOversizedSpec pins the size contract: a spec
// larger than the size limit exits 2 naming the limit, with nothing on
// standard output.
func TestConnectorValidateRefusesAnOversizedSpec(t *testing.T) {
	big := strings.Repeat("x", 257<<10)
	code, out, errOut := runConnector(t, []string{"connector", "validate", "big.yml"},
		map[string]string{"big.yml": big}, nil, nil)
	if code != 2 {
		t.Fatalf("code = %d, stderr = %q; want 2", code, errOut)
	}
	if out != "" {
		t.Fatalf("stdout = %q, want empty on a failed run", out)
	}
	if !strings.Contains(errOut, "is larger than the size limit") {
		t.Fatalf("stderr = %q, want the size rule", errOut)
	}
}

// TestConnectorWithoutAReaderRefusesEverySpec pins the wiring contract: with
// no file reader the commands exit 2 and read nothing.
func TestConnectorWithoutAReaderRefusesEverySpec(t *testing.T) {
	var stdout, stderr bytes.Buffer
	deps := Deps{
		Stdout: &stdout,
		Stderr: &stderr,
		Getenv: func(string) string { return "" },
	}
	if code := Run([]string{"connector", "validate", "wiki.yml"}, deps); code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty on a failed run", stdout.String())
	}
}

// TestConnectorTestDerivesTwoCommandsAndOneTool checks the derived texts of a
// two-operation spec: the CLI text holds both commands, and the MCP text holds
// only the operation whose tool stays on.
func TestConnectorTestDerivesTwoCommandsAndOneTool(t *testing.T) {
	code, out, errOut := runConnector(t, []string{"connector", "test", "wiki.yml"},
		map[string]string{"wiki.yml": connectorWikiSpec}, nil, nil)
	if code != 0 {
		t.Fatalf("code = %d, stderr = %q; want 0", code, errOut)
	}
	for _, want := range []string{"example-wiki docs list", "example-wiki docs search", "example_wiki_docs_list"} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout lacks %q", want)
		}
	}
	if strings.Contains(out, "example_wiki_docs_search") {
		t.Fatalf("stdout holds the tool of the operation with MCP off")
	}
}

// TestConnectorGroupShowsItsSubcommands checks the help of the new group: the
// connector group lists validate and test.
func TestConnectorGroupShowsItsSubcommands(t *testing.T) {
	code, out, _ := runConnector(t, []string{"connector", "--help"}, nil, nil, nil)
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	for _, want := range []string{"validate", "test"} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout = %q, want %q", out, want)
		}
	}
}

// TestConnectorFailureHoldsItsCodeAndText checks the failure shape directly:
// the code and the redacted safe text the run reports.
func TestConnectorFailureHoldsItsCodeAndText(t *testing.T) {
	failure := &connectorFailure{code: codeConnectorTestFailed, msg: "connector spec file wiki.yml: failed"}
	if failure.Error() != "connector spec file wiki.yml: failed" {
		t.Fatalf("Error() = %q", failure.Error())
	}
	if failure.ErrorCode() != codeConnectorTestFailed {
		t.Fatalf("ErrorCode() = %q", failure.ErrorCode())
	}
}

// TestConnectorTestReportsAnUnreadableSpec pins the file contract for test:
// a spec file that cannot be read exits 2 with nothing on standard output.
func TestConnectorTestReportsAnUnreadableSpec(t *testing.T) {
	code, out, errOut := runConnector(t, []string{"connector", "test", "missing.yml"},
		nil, errTestUnreadable, nil)
	if code != 2 {
		t.Fatalf("code = %d, stderr = %q; want 2", code, errOut)
	}
	if out != "" {
		t.Fatalf("stdout = %q, want empty on a failed run", out)
	}
	if !strings.Contains(errOut, "missing.yml") {
		t.Fatalf("stderr = %q, want the base name", errOut)
	}
}

// duplicateCommandSpec is the wiki spec with two operations under one custom
// command name: the loader refuses the collision.
const duplicateCommandSpec = `version: 1
kind: example-wiki
summary: Reads documents of an example wiki.
endpoint:
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
    cli: search
    response:
      items: /items
      fields:
        id:
          pointer: /id
    examples:
      - request: {}
        response:
          status: 200
          body:
            items:
              - id: doc-1
        expect:
          id: doc-1
  - id: example-wiki.docs.other
    summary: List documents again.
    access: read
    method: GET
    path: /other
    cli: search
    response:
      items: /items
      fields:
        id:
          pointer: /id
    examples:
      - request: {}
        response:
          status: 200
          body:
            items:
              - id: doc-9
        expect:
          id: doc-9
`

// TestConnectorTestRefusesACollidingCommand pins the loader rule at the
// command: two operations under one command name exit 2 naming the collision,
// with nothing on standard output.
func TestConnectorTestRefusesACollidingCommand(t *testing.T) {
	code, out, errOut := runConnector(t, []string{"connector", "test", "dup.yml"},
		map[string]string{"dup.yml": duplicateCommandSpec}, nil, nil)
	if code != 2 {
		t.Fatalf("code = %d, stderr = %q; want 2", code, errOut)
	}
	if out != "" {
		t.Fatalf("stdout = %q, want empty on a failed run", out)
	}
	if !strings.Contains(errOut, "collides with another command") {
		t.Fatalf("stderr = %q, want the collision rule", errOut)
	}
}

// TestConnectorTestJSONPrintsAVersionedDocument checks the success shape of
// test in JSON mode: one versioned document on standard output.
func TestConnectorTestJSONPrintsAVersionedDocument(t *testing.T) {
	code, out, errOut := runConnector(t, []string{"--output", "json", "connector", "test", "wiki.yml"},
		map[string]string{"wiki.yml": connectorWikiSpec}, nil, nil)
	if code != 0 {
		t.Fatalf("code = %d, stderr = %q; want 0", code, errOut)
	}
	var body struct {
		SchemaVersion int    `json:"schemaVersion"`
		Kind          string `json:"kind"`
		Operations    int    `json:"operations"`
		Status        string `json:"status"`
	}
	if err := json.Unmarshal([]byte(out), &body); err != nil {
		t.Fatalf("stdout = %q, want JSON: %v", out, err)
	}
	if body.SchemaVersion != 1 || body.Kind != "example-wiki" || body.Operations != 2 || body.Status != "pass" {
		t.Fatalf("body = %+v, want the passing document", body)
	}
	if errOut != "" {
		t.Fatalf("stderr = %q, want empty on a passing run", errOut)
	}
}

// TestConnectorSecretsReadsOnlyTheDeclaredVariables checks the redaction
// inputs: with no reader there is no secret, and the identity variable joins
// the credential variable when the spec names one.
func TestConnectorSecretsReadsOnlyTheDeclaredVariables(t *testing.T) {
	model := mustModel(t, connectorWikiSpec)
	if secrets := connectorSecrets(&runtime{}, model); len(secrets) != 0 {
		t.Fatalf("connectorSecrets() = %v, want none without a reader", secrets)
	}
	basic := strings.Replace(connectorWikiSpec, "  type: bearer", "  type: basic", 1)
	basic = strings.Replace(basic, "  credential_env: EXAMPLE_WIKI_API_KEY",
		"  credential_env: EXAMPLE_WIKI_API_KEY\n  identity_env: EXAMPLE_WIKI_EMAIL", 1)
	rt := &runtime{getenv: func(name string) string {
		return map[string]string{"EXAMPLE_WIKI_API_KEY": "key-1", "EXAMPLE_WIKI_EMAIL": "a@example.test"}[name]
	}}
	secrets := connectorSecrets(rt, mustModel(t, basic))
	if len(secrets) != 2 {
		t.Fatalf("connectorSecrets() = %v, want the credential and the identity", secrets)
	}
}
