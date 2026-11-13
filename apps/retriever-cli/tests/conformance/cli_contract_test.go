// Package conformance runs the published command-line contract checks of specs/fixtures/cli-conformance. Each check runs
// the CLI in process, with no source wired, and compares the exit status, the streams, and the failure body with the
// fixture. The runner refuses a check whose status is outside the closed set the fixture names.
package conformance

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/in/cli"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/ports"
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

// loadFixture reads and decodes the contract fixture.
func loadFixture(t *testing.T) fixture {
	t.Helper()
	raw, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read the conformance fixture: %v", err)
	}
	var fx fixture
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatalf("decode the conformance fixture: %v", err)
	}
	return fx
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
		Now:      time.Now,
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

// TestContractChecks runs every published check. A check that fails is a failure, unless the fixture lists it under
// knownGaps. A listed gap that passes is also a failure, so a fixed gap must be removed from the list.
func TestContractChecks(t *testing.T) {
	fx := loadFixture(t)
	if fx.SchemaVersion != 1 {
		t.Fatalf("the fixture schemaVersion is %d, want 1", fx.SchemaVersion)
	}
	closed := map[int]bool{}
	for _, status := range fx.ClosedExitStatuses {
		closed[status] = true
	}
	ids := map[string]bool{}
	for _, a := range fx.Assertions {
		ids[a.ID] = true
	}
	gaps := map[string]bool{}
	for _, g := range fx.KnownGaps {
		if !ids[g] {
			t.Errorf("knownGaps names %q, which no assertion has", g)
		}
		gaps[g] = true
	}
	for _, a := range fx.Assertions {
		t.Run(a.ID, func(t *testing.T) {
			if !closed[a.ExitStatus] {
				t.Fatalf("exit status %d is outside the closed set %v", a.ExitStatus, fx.ClosedExitStatuses)
			}
			fails := runCheck(a)
			if gaps[a.ID] {
				if len(fails) == 0 {
					t.Errorf("%s is listed in knownGaps but passes; remove it from the list", a.ID)
				}
				return
			}
			for _, f := range fails {
				t.Error(f)
			}
		})
	}
}
