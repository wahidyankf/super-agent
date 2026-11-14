// Package spec holds the connector spec of ADR 001: the model of a declarative connector, the JSON Schema file that
// checks its document, the validation of a document, and the plain-text review that lists what a spec can do. It takes
// bytes and returns a model or findings. It reads no file, environment variable, clock, network, or process, so the
// app reads the spec file and passes the bytes in. It imports only the contract of this repository.
package spec

import (
	"errors"
	"fmt"
	"strings"
)

// CodeSpecInvalid is the stable error code of a spec that fails validation. Its area is connector, so the code is
// retriever.connector.spec_invalid. A finding carries it, and no finding names a value.
const CodeSpecInvalid = "retriever.connector.spec_invalid"

// MaxBytes is the most bytes one spec document may hold: 256 KiB. A longer document is refused before it is parsed.
const MaxBytes = 256 << 10

// Version is the only version of the spec document that this module reads.
const Version = 1

// ErrInvalid is the sentinel that an InvalidError matches with errors.Is. Callers compare with it, never with text.
var ErrInvalid = errors.New("the connector spec is invalid")

// Finding is one reason that a spec is invalid. Path is the JSON pointer of the field at fault, such as
// /operations/0/access, and it is empty for the whole document. Rule states the broken rule in fixed words. No
// finding holds a value of the spec, so a credential-shaped string never reaches a message.
type Finding struct {
	// Code is always CodeSpecInvalid.
	Code string
	// Path is the JSON pointer of the field, or "" for the whole document.
	Path string
	// Rule is the broken rule, in fixed words.
	Rule string
}

// String returns the finding as one line: the code, the field path, and the rule.
func (f Finding) String() string {
	where := f.Path
	if where == "" {
		where = "(document)"
	}
	return fmt.Sprintf("%s: %s: %s", f.Code, where, f.Rule)
}

// Result is the outcome of validating one spec document. Spec is set only when Findings is empty.
type Result struct {
	// Spec is the model of a valid document, or nil when the document is invalid.
	Spec *Spec
	// Findings lists every reason the document is invalid, in path order. It is empty for a valid document.
	Findings []Finding
}

// Valid reports whether the document passed every rule.
func (r Result) Valid() bool {
	return len(r.Findings) == 0
}

// Err returns nil for a valid document, and otherwise an InvalidError that holds the findings.
func (r Result) Err() error {
	if r.Valid() {
		return nil
	}
	return &InvalidError{Findings: r.Findings}
}

// InvalidError reports a spec that failed validation. It matches ErrInvalid with errors.Is, and its text lists the
// findings, one per line, with no value of the spec.
type InvalidError struct {
	// Findings are the reasons the document is invalid.
	Findings []Finding
}

// Error returns the findings, one per line.
func (e *InvalidError) Error() string {
	lines := make([]string, 0, len(e.Findings))
	for _, f := range e.Findings {
		lines = append(lines, f.String())
	}
	return strings.Join(lines, "\n")
}

// Is reports whether target is ErrInvalid.
func (e *InvalidError) Is(target error) bool {
	return target == ErrInvalid
}
