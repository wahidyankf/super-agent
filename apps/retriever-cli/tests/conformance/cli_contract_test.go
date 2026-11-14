package conformance

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/in/cli"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
	"strings"
	"time"
)

// fixturePath is the contract fixture, relative to this package at apps/retriever-cli/tests/conformance.
const fixturePath = "../../../../specs/fixtures/cli-conformance/assertions.json"

// errNoSource is the setup error of every check. The contract checks need no source, so none is wired.
var errNoSource = errors.New("no source is wired in the conformance run")

// fixture is the decoded assertions.json.
type fixture struct {
	SchemaVersion      int         `json:"schemaVersion"`
	ClosedExitStatuses []int       `json:"closedExitStatuses"`
	KnownGaps          []string    `json:"knownGaps"`
	Assertions         []assertion `json:"assertions"`
}

// assertion is one published check.
type assertion struct {
	ID         string      `json:"id"`
	Argv       []string    `json:"argv"`
	ExitStatus int         `json:"exitStatus"`
	Stdout     streamCheck `json:"stdout"`
	Stderr     streamCheck `json:"stderr"`
}

// streamCheck states what one stream must hold. Each field is optional.
type streamCheck struct {
	Contains  []string       `json:"contains"`
	Empty     *bool          `json:"empty"`
	NotEmpty  *bool          `json:"notEmpty"`
	JSONError *jsonErrorSpec `json:"jsonError"`
}

// jsonErrorSpec states the failure body. The top-level fields are compared with the body, and code and retryable are
// compared with the error object inside it.
type jsonErrorSpec struct {
	SchemaVersion int    `json:"schemaVersion"`
	Command       string `json:"command"`
	ExitCode      int    `json:"exitCode"`
	Code          string `json:"code"`
	Retryable     bool   `json:"retryable"`
}

// conformanceNow is the fixed instant the checks read as the clock, so no check reads the real clock.
var conformanceNow = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

// failureBody is the decoded failure body of the CLI, as domain documentation names it.
type failureBody struct {
	SchemaVersion int    `json:"schemaVersion"`
	Command       string `json:"command"`
	ExitCode      int    `json:"exitCode"`
	Error         struct {
		Code      string `json:"code"`
		Retryable bool   `json:"retryable"`
	} `json:"error"`
}

// runCheck runs one check in process and returns every way its result differs from the fixture. An empty list means the
// check passes.
func runCheck(a assertion) []string {
	var out, errb bytes.Buffer
	deps := cli.Deps{
		Ports:    ports.Ports{},
		Stdin:    strings.NewReader(""),
		Stdout:   &out,
		Stderr:   &errb,
		Getenv:   func(string) string { return "" },
		Now:      func() time.Time { return conformanceNow },
		SetupErr: errNoSource,
	}
	status := cli.Run(a.Argv, deps)
	var fails []string
	if status != a.ExitStatus {
		fails = append(fails, fmt.Sprintf("exit status is %d, want %d", status, a.ExitStatus))
	}
	fails = append(fails, checkStream("stdout", out.String(), a.Stdout)...)
	fails = append(fails, checkStream("stderr", errb.String(), a.Stderr)...)
	return fails
}

// checkStream compares one stream with its check.
func checkStream(name, got string, want streamCheck) []string {
	var fails []string
	for _, text := range want.Contains {
		if !strings.Contains(got, text) {
			fails = append(fails, fmt.Sprintf("%s does not contain %q", name, text))
		}
	}
	if want.Empty != nil && *want.Empty && got != "" {
		fails = append(fails, name+" is not empty")
	}
	if want.NotEmpty != nil && *want.NotEmpty && got == "" {
		fails = append(fails, name+" is empty")
	}
	if want.JSONError != nil {
		fails = append(fails, checkJSONError(name, got, *want.JSONError)...)
	}
	return fails
}

// checkJSONError compares the failure body of a stream with the expected body.
func checkJSONError(name, got string, want jsonErrorSpec) []string {
	var body failureBody
	if err := json.Unmarshal([]byte(got), &body); err != nil {
		return []string{fmt.Sprintf("%s is not a failure body: %v", name, err)}
	}
	var fails []string
	if body.SchemaVersion != want.SchemaVersion {
		fails = append(fails, fmt.Sprintf("%s schemaVersion is %d, want %d", name, body.SchemaVersion, want.SchemaVersion))
	}
	if body.Command != want.Command {
		fails = append(fails, fmt.Sprintf("%s command is %q, want %q", name, body.Command, want.Command))
	}
	if body.ExitCode != want.ExitCode {
		fails = append(fails, fmt.Sprintf("%s exitCode is %d, want %d", name, body.ExitCode, want.ExitCode))
	}
	if body.Error.Code != want.Code {
		fails = append(fails, fmt.Sprintf("%s error code is %q, want %q", name, body.Error.Code, want.Code))
	}
	if body.Error.Retryable != want.Retryable {
		fails = append(fails, fmt.Sprintf("%s error retryable is %v, want %v", name, body.Error.Retryable, want.Retryable))
	}
	return fails
}
