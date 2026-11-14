package spec

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestParseRefusesAnEmptyDocument checks that a document with no YAML document, not even an empty mapping, is refused
// for the whole document.
func TestParseRefusesAnEmptyDocument(t *testing.T) {
	requireRefused(t, []byte(""), "")
}

// TestParseRefusesARootThatIsNotAMapping checks that a document whose root is a scalar is refused for the whole
// document, and that the refusal names no value.
func TestParseRefusesARootThatIsNotAMapping(t *testing.T) {
	requireRefused(t, []byte("just text\n"), "", "just text")
}

// TestParseAcceptsEachPlainScalarKind checks that a document with a string, an integer, a float, a boolean, and a null
// reads, and that the numbers keep their text, so an integer stays an integer.
func TestParseAcceptsEachPlainScalarKind(t *testing.T) {
	value, fs := parseDocument([]byte("a: text\nb: 3\nc: 1.5\nd: true\ne: null\nf: [1, \"2\"]\n"))
	if len(fs) != 0 {
		t.Fatalf("findings = %v; want none", fs)
	}
	m, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("value = %T; want a map", value)
	}
	if m["a"] != "text" || m["d"] != true || m["e"] != nil {
		t.Fatalf("scalars = %v", m)
	}
	if n, ok := m["b"].(json.Number); !ok || n.String() != "3" {
		t.Fatalf("integer = %#v; want json.Number 3", m["b"])
	}
	if n, ok := m["c"].(json.Number); !ok || n.String() != "1.5" {
		t.Fatalf("float = %#v; want json.Number 1.5", m["c"])
	}
	if list, ok := m["f"].([]any); !ok || len(list) != 2 || list[1] != "2" {
		t.Fatalf("sequence = %#v", m["f"])
	}
}

// TestParseRefusesAKeyNameThatHoldsASecretWithoutShowingIt checks that a field name which is not a plain identifier is
// withheld from the path, so a secret-shaped key never reaches a finding.
func TestParseRefusesAKeyNameThatHoldsASecretWithoutShowingIt(t *testing.T) {
	secret := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdef0123456789"
	doc := fixtureWith("version: 1\n", "version: 1\n"+secret+": 1\n")
	r := Validate([]byte(doc))
	if r.Valid() {
		t.Fatal("an unknown field must be refused")
	}
	if strings.Contains(allText(r), secret) {
		t.Fatalf("a finding shows the field name %q:\n%s", secret, allText(r))
	}
	if !hasFinding(r, "/"+withheldName) {
		t.Fatalf("no finding at the withheld field path; findings:\n%s", allText(r))
	}
}

// TestParseKeepsAPlainFieldNameInThePath checks that an unknown field with a plain name shows its name, so a person
// can find it.
func TestParseKeepsAPlainFieldNameInThePath(t *testing.T) {
	requireRefused(t, []byte(fixtureWith("endpoint:\n", "extra: 1\nendpoint:\n")), "/extra")
}

// TestParseOfAnEmptyNumberOrTextIsNotAFinding checks the empty-value edge: an empty text value is a value, and the
// schema decides whether an empty text is allowed.
func TestParseOfAnEmptyNumberOrTextIsNotAFinding(t *testing.T) {
	value, fs := parseDocument([]byte("a: \"\"\nb: 0\n"))
	if len(fs) != 0 {
		t.Fatalf("findings = %v; want none", fs)
	}
	m, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("value = %T; want a map", value)
	}
	if m["a"] != "" {
		t.Fatalf("a = %#v; want the empty text", m["a"])
	}
}

// TestParseRefusesAMergeKeyAsAliasOrTag checks that the merge key of YAML 1.1, which needs an alias, is refused as a
// tag the spec does not allow.
func TestParseRefusesAMergeKeyAsAliasOrTag(t *testing.T) {
	doc := "base: &b\n  a: 1\nderived:\n  <<: *b\n"
	_, fs := parseDocument([]byte(doc))
	if len(fs) == 0 {
		t.Fatal("a merge key with an alias must be refused")
	}
	for _, f := range fs {
		if f.Code != CodeSpecInvalid {
			t.Fatalf("finding code = %q", f.Code)
		}
	}
}

// TestParseRefusesASyntaxErrorWithoutItsText checks that a document the YAML reader cannot read is refused for the
// whole document, and that the reader's own text, which may hold the document's content, is not shown.
func TestParseRefusesASyntaxErrorWithoutItsText(t *testing.T) {
	requireRefused(t, []byte("version: [1, secret-token-here\n"), "", "secret-token-here")
}

// TestParseRefusesAnAnchorOnAnOperationField checks that an anchor on a value inside an operation is refused at that
// operation's field, not only one at the top of the document.
func TestParseRefusesAnAnchorOnAnOperationField(t *testing.T) {
	doc := fixtureWith("    summary: Read one document.\n", "    summary: &x Read one document.\n")
	requireRefused(t, []byte(doc), "/operations/1/summary")
}
