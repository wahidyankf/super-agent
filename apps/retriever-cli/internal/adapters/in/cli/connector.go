package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/in/mcp"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/conformance"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/connectorspec"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/operations"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/specfolder"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// The stable codes of the connector commands. A refused spec file keeps the
// spec code the loader reports, and an unreadable file and a failed replay
// have their own codes in the connector area.
const (
	codeConnectorUnreadable = "retriever.connector.spec_unreadable"
	codeConnectorTestFailed = "retriever.connector.test_failed"
)

// connectorFailure is a validate or test failure: the stable code and the
// redacted safe text. The CLI prints the text as it is on standard error, and
// the run exits 2 with nothing on standard output.
type connectorFailure struct {
	code string
	msg  string
}

// Error returns the redacted safe text of the failure.
func (e *connectorFailure) Error() string { return e.msg }

// ErrorCode returns the stable code of the failure.
func (e *connectorFailure) ErrorCode() string { return e.code }

// connectorGroup is the connector group of the command tree: the conformance
// commands that check one spec file. The spec kind commands the run registers
// serve the sources; these two only check a file, so they need no source.
func connectorGroup() *command {
	return &command{
		name:    "connector",
		summary: "check a connector spec file",
		children: []*command{
			{
				name:    "validate",
				summary: "check that a connector spec is valid and every operation carries an example",
				leaf: &leaf{
					usage:      "retriever connector validate <spec>",
					positional: []string{"spec"},
					run:        connectorValidate,
				},
			},
			{
				name:    "test",
				summary: "replay the examples of a connector spec and print the conformance report",
				leaf: &leaf{
					usage:      "retriever connector test <spec>",
					positional: []string{"spec"},
					run:        connectorTest,
				},
			},
		},
	}
}

// connectorValidate runs the static half of the suite over one spec file: the
// schema and the semantic rules must pass, and every operation must carry an
// example. The report prints the review a person reads before the engine runs
// the spec (control C1). A failure exits 2 with its diagnostics on standard
// error and nothing on standard output.
func connectorValidate(inv *invocation) error {
	rt := inv.rt
	data, base, ferr := readConnectorSpec(rt, inv.positional[0])
	if ferr != nil {
		return ferr
	}
	model, issues := conformance.Validate(data)
	if len(issues) != 0 {
		return failedValidation(base, issues)
	}
	report := conformance.AssembleValidation(base, model, nil, spec.Review(model), connectorSecrets(rt, model))
	if rt.format == domain.FormatJSON {
		return writeJSON(rt.stdout, connectorJSONDocument(base, model, "pass", report))
	}
	_, err := fmt.Fprint(rt.stdout, report)
	return err
}

// connectorTest runs the whole suite over one spec file: validate, then the
// replay of every example through the engine, then the report with the CLI,
// MCP, and README texts derived from the spec. A failure exits 2 with its
// diagnostics on standard error and nothing on standard output.
func connectorTest(inv *invocation) error {
	rt := inv.rt
	data, base, ferr := readConnectorSpec(rt, inv.positional[0])
	if ferr != nil {
		return ferr
	}
	model, issues := conformance.Validate(data)
	if len(issues) != 0 {
		return failedValidation(base, issues)
	}
	replayed, values := conformance.Test(model)
	if len(replayed) != 0 {
		return failedReplay(base, replayed)
	}
	derived, derr := connectorDerived(model, base, data)
	if derr != nil {
		return derr
	}
	report := conformance.Assemble(base, model, nil, nil, values, derived, connectorSecrets(rt, model))
	if rt.format == domain.FormatJSON {
		return writeJSON(rt.stdout, connectorJSONDocument(base, model, "pass", report))
	}
	_, err := fmt.Fprint(rt.stdout, report)
	return err
}

// readConnectorSpec returns the bytes and the base name of one spec file. A
// file that cannot be read, or that is larger than the spec size limit, is a
// failure that names the base file only, never the path the caller typed.
func readConnectorSpec(rt *runtime, path string) ([]byte, string, error) {
	base := filepath.Base(path)
	if rt.readFile == nil {
		return nil, "", &connectorFailure{code: codeConnectorUnreadable,
			msg: "connector spec file " + base + ": could not be read"}
	}
	data, err := rt.readFile(path)
	if err != nil {
		return nil, "", &connectorFailure{code: codeConnectorUnreadable,
			msg: "connector spec file " + base + ": could not be read"}
	}
	if len(data) > spec.MaxBytes {
		return nil, "", &connectorFailure{code: codeConnectorUnreadable,
			msg: "connector spec file " + base + ": is larger than the size limit"}
	}
	return data, base, nil
}

// failedValidation returns the failure of a spec the static checks refuse.
// Every issue names the file, the field, and the rule, never a value.
func failedValidation(base string, issues []conformance.Issue) error {
	lines := make([]string, 0, len(issues))
	for _, issue := range issues {
		lines = append(lines, "connector spec file "+base+": "+issueSuffix(issue))
	}
	return &connectorFailure{code: spec.CodeSpecInvalid, msg: strings.Join(lines, "\n")}
}

// failedReplay returns the failure of a spec whose examples do not replay.
// Every issue names the operation, the example, and the field at fault.
func failedReplay(base string, issues []conformance.Issue) error {
	lines := make([]string, 0, len(issues))
	for _, issue := range issues {
		lines = append(lines, "connector spec file "+base+": "+issueSuffix(issue))
	}
	return &connectorFailure{code: codeConnectorTestFailed, msg: strings.Join(lines, "\n")}
}

// issueSuffix returns the path and the rule of an issue, naming the operation,
// the example, and the field when one is at fault.
func issueSuffix(issue conformance.Issue) string {
	return issue.String()
}

// connectorDerived renders the three texts the suite derives from a validated
// user spec with the real adapters: the help of its command-line surface, the
// JSON of its MCP tools, and the plain-text review. A spec of a built-in kind
// is refused here as it is in the user folder: only the bundled suite checks
// those, so a deliberate copy can never pass as a user spec.
func connectorDerived(model *spec.Spec, base string, data []byte) (conformance.Derived, error) {
	sum := sha256.Sum256(data)
	set, err := connectorspec.Load([]specfolder.File{{
		Name:   base,
		Data:   data,
		Digest: hex.EncodeToString(sum[:]),
	}})
	if err != nil {
		return conformance.Derived{}, err
	}
	tools := mcp.ToolsJSON(toolRowsOf(set, model))
	return conformance.Derived{
		CLI:    specCLIText(set, model),
		MCP:    string(tools),
		README: spec.Review(model),
	}, nil
}

// toolRowsOf returns the CLI rows of a set whose operation keeps its MCP
// tool, in order. The row alone cannot tell: the loader clears the tool of a
// row with the tool off only in the catalog rows, so the model decides.
func toolRowsOf(set connectorspec.Set, model *spec.Spec) []operations.Row {
	on := make(map[string]bool, len(model.Operations))
	for _, op := range model.Operations {
		if op.MCP.Enabled {
			on[op.ID] = true
		}
	}
	var rows []operations.Row
	for _, row := range set.CLIRows {
		if on[string(row.ID)] {
			rows = append(rows, row)
		}
	}
	return rows
}

// specCLIText returns the help of the command-line surface of one loaded spec
// of one kind: the help of its kind group, then the help of each of its
// commands in row order. The text comes from the real command tree, so it is
// the text the run would show. Every row resolves, because the tree is built
// from the same rows.
func specCLIText(set connectorspec.Set, model *spec.Spec) string {
	tree := commandTreeWith(set.Operations, set.CLIRows)
	group, path, _ := walk(tree, []string{model.Kind})
	var b strings.Builder
	writeHelp(&b, path, group)
	for _, row := range set.CLIRows {
		words := strings.Split(row.CLI, " ")
		node, lpath, _ := walk(tree, words)
		b.WriteString("\n")
		writeHelp(&b, lpath, node)
	}
	return b.String()
}

// connectorSecrets returns the secret values the spec declares: the values of
// its credential and identity variables. The replay never reads them, but the
// report redacts every occurrence, so a credential planted in a canned reply
// reads [REDACTED]. An empty environment reads no secret.
func connectorSecrets(rt *runtime, model *spec.Spec) []string {
	if rt.getenv == nil {
		return nil
	}
	var secrets []string
	if v := rt.getenv(model.Auth.CredentialEnv); v != "" {
		secrets = append(secrets, v)
	}
	if model.Auth.IdentityEnv != "" {
		if v := rt.getenv(model.Auth.IdentityEnv); v != "" {
			secrets = append(secrets, v)
		}
	}
	return secrets
}

// connectorJSONDocument returns the JSON document of a passing connector run.
// The shape is versioned with the schema version every JSON document carries,
// so a caller that parses it pins the version it understands.
type connectorJSON struct {
	SchemaVersion int    `json:"schemaVersion"`
	Spec          string `json:"spec"`
	Kind          string `json:"kind"`
	Operations    int    `json:"operations"`
	Examples      int    `json:"examples"`
	Status        string `json:"status"`
	Report        string `json:"report"`
}

// connectorJSONDocument builds the passing document of one spec.
func connectorJSONDocument(base string, model *spec.Spec, status, report string) connectorJSON {
	examples := 0
	for _, op := range model.Operations {
		examples += len(op.Examples)
	}
	return connectorJSON{
		SchemaVersion: schemaVersion,
		Spec:          base,
		Kind:          model.Kind,
		Operations:    len(model.Operations),
		Examples:      examples,
		Status:        status,
		Report:        report,
	}
}
