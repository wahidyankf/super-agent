package spec

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// TestSurfaceDecodesABooleanOrACustomName checks the three forms of a surface: on, off, and a custom name, which turns
// the surface on under that name. A number is refused.
func TestSurfaceDecodesABooleanOrACustomName(t *testing.T) {
	cases := []struct {
		doc  string
		want Surface
		ok   bool
	}{
		{`true`, Surface{Enabled: true}, true},
		{`false`, Surface{Enabled: false}, true},
		{`"docs-list"`, Surface{Enabled: true, Name: "docs-list"}, true},
		{`5`, Surface{}, false},
	}
	for _, c := range cases {
		var s Surface
		err := json.Unmarshal([]byte(c.doc), &s)
		if (err == nil) != c.ok {
			t.Fatalf("decode %s: err = %v; want ok %v", c.doc, err, c.ok)
		}
		if c.ok && s != c.want {
			t.Fatalf("decode %s = %+v; want %+v", c.doc, s, c.want)
		}
	}
}

// TestOperationDecodesWithBothSurfacesOnByDefault checks that an operation with no cli or mcp key offers both.
func TestOperationDecodesWithBothSurfacesOnByDefault(t *testing.T) {
	var op Operation
	if err := json.Unmarshal([]byte(`{"id":"a.b.c"}`), &op); err != nil {
		t.Fatal(err)
	}
	if !op.CLI.Enabled || !op.MCP.Enabled || op.ID != "a.b.c" {
		t.Fatalf("operation = %+v; want an id with both surfaces on", op)
	}
}

// TestOperationDecodeFailsOnAMismatchedField checks that a field of the wrong JSON type makes the operation fail to
// decode, so no partial operation escapes.
func TestOperationDecodeFailsOnAMismatchedField(t *testing.T) {
	var op Operation
	if err := json.Unmarshal([]byte(`{"id":5}`), &op); err == nil {
		t.Fatal("an id that is a number must fail to decode")
	}
}

// TestFindingStringNamesTheWholeDocumentForAnEmptyPath checks the one-line form of a finding: a path is shown, and an
// empty path names the document.
func TestFindingStringNamesTheWholeDocumentForAnEmptyPath(t *testing.T) {
	whole := Finding{Code: CodeSpecInvalid, Path: "", Rule: "is too big"}
	if got := whole.String(); got != "retriever.connector.spec_invalid: (document): is too big" {
		t.Fatalf("String() = %q", got)
	}
	field := Finding{Code: CodeSpecInvalid, Path: "/kind", Rule: "is a built-in kind"}
	if got := field.String(); got != "retriever.connector.spec_invalid: /kind: is a built-in kind" {
		t.Fatalf("String() = %q", got)
	}
}

// TestResultErrIsNilForAValidDocumentAndMatchesTheSentinel checks the error of a result: none for a valid document,
// and for an invalid one an error that matches ErrInvalid and lists every finding.
func TestResultErrIsNilForAValidDocumentAndMatchesTheSentinel(t *testing.T) {
	if err := (Result{}).Err(); err != nil {
		t.Fatalf("Err() of a valid result = %v; want nil", err)
	}
	r := Result{Findings: []Finding{
		{Code: CodeSpecInvalid, Path: "/a", Rule: "one"},
		{Code: CodeSpecInvalid, Path: "/b", Rule: "two"},
	}}
	err := r.Err()
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("errors.Is(err, ErrInvalid) = false for %v", err)
	}
	if lines := strings.Count(err.Error(), "\n"); lines != 1 {
		t.Fatalf("error text has %d line breaks; want one between two findings:\n%s", lines, err.Error())
	}
	if errors.Is(err, errors.New("another")) {
		t.Fatal("an invalid spec error must not match an unrelated error")
	}
}

// TestInvalidErrorWithNoFindingsIsEmpty checks the edge of the error text: an error with no finding has no text.
func TestInvalidErrorWithNoFindingsIsEmpty(t *testing.T) {
	if got := (&InvalidError{}).Error(); got != "" {
		t.Fatalf("Error() = %q; want empty", got)
	}
}
