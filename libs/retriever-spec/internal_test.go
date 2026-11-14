package spec

import (
	"errors"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6/kind"
	"go.yaml.in/yaml/v3"
)

// TestRefusingLoaderRefusesEveryURL checks that the loader the schema compiles with reads no document, so a reference
// outside the schema fails instead of reaching a file or the network.
func TestRefusingLoaderRefusesEveryURL(t *testing.T) {
	if _, err := (refusingLoader{}).Load("https://example.test/other.json"); !errors.Is(err, errOutsideSchema) {
		t.Fatalf("Load = %v; want the refusal", err)
	}
}

// TestCompileSchemaRefusesDocumentsItCannotCompile checks the three ways a schema fails to compile: a document that is
// not JSON, a name the compiler cannot take, and a reference to a document outside the schema.
func TestCompileSchemaRefusesDocumentsItCannotCompile(t *testing.T) {
	if _, err := compileSchema(schemaURL, []byte("{")); err == nil {
		t.Fatal("a document that is not JSON must not compile")
	}
	if _, err := compileSchema("%zz", Schema()); err == nil {
		t.Fatal("a name that is not a URL must not compile")
	}
	outside := []byte(`{"$ref": "https://example.test/other.json"}`)
	if _, err := compileSchema(schemaURL, outside); err == nil {
		t.Fatal("a reference to a document outside the schema must not compile")
	}
}

// TestMustCompilePanicsOnABrokenSchema checks that the embedded schema is compiled or the package does not load.
func TestMustCompilePanicsOnABrokenSchema(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("mustCompile of a broken schema must panic")
		}
	}()
	mustCompile([]byte("{"))
}

// TestFindingsOfAnErrorThatIsNotAValidationError checks the fallback: an error that is not a schema failure still
// refuses the document, at the whole document, with the fallback rule.
func TestFindingsOfAnErrorThatIsNotAValidationError(t *testing.T) {
	fs := findingsOf(errors.New("another failure"), nil)
	if len(fs) != 1 || fs[0].Path != "" || fs[0].Rule != fallbackRule {
		t.Fatalf("findings = %v; want one finding for the whole document with the fallback rule", fs)
	}
}

// TestKeywordOfNamesTheKeywordOrNone checks the keyword of a failure: a not names not, a required names required, and a
// kind with no keyword names none.
func TestKeywordOfNamesTheKeywordOrNone(t *testing.T) {
	if got := keywordOf(&kind.Not{}); got != "not" {
		t.Fatalf("keywordOf(not) = %q", got)
	}
	if got := keywordOf(&kind.Required{}); got != "required" {
		t.Fatalf("keywordOf(required) = %q", got)
	}
	if got := keywordOf(&kind.Group{}); got != "" {
		t.Fatalf("keywordOf(group) = %q; want none", got)
	}
}

// TestPointerOfWithholdsAPathThatDoesNotFitTheValue checks the pointer of a schema location: a plain key is kept, and a
// location that does not fit the value is withheld rather than guessed.
func TestPointerOfWithholdsAPathThatDoesNotFitTheValue(t *testing.T) {
	if got := pointerOf(map[string]any{"k": "v"}, []string{"k"}); got != "/k" {
		t.Fatalf("pointer = %q; want /k", got)
	}
	if got := pointerOf([]any{"a"}, []string{"0"}); got != "/0" {
		t.Fatalf("pointer = %q; want /0", got)
	}
	if got := pointerOf([]any{"a"}, []string{"5"}); got != "/"+withheldName {
		t.Fatalf("pointer past the end = %q; want the withheld name", got)
	}
	if got := pointerOf([]any{"a"}, []string{"x"}); got != "/"+withheldName {
		t.Fatalf("pointer with no index = %q; want the withheld name", got)
	}
	if got := pointerOf("text", []string{"a"}); got != "/"+withheldName {
		t.Fatalf("pointer into a scalar = %q; want the withheld name", got)
	}
}

// TestToModelRefusesAValueThatIsNotJSONOrNotASpec checks the two ways a checked value fails to become a model: a value
// that cannot be marshaled, and a value of the wrong type for a field of the model.
func TestToModelRefusesAValueThatIsNotJSONOrNotASpec(t *testing.T) {
	if s, fs := toModel(make(chan int)); s != nil || len(fs) != 1 {
		t.Fatalf("toModel(chan) = %v, %v; want one finding and no model", s, fs)
	}
	if s, fs := toModel(map[string]any{"version": "one"}); s != nil || len(fs) != 1 {
		t.Fatalf("toModel(wrong type) = %v, %v; want one finding and no model", s, fs)
	}
}

// TestNodeRefusesANodeThatIsNotAValue checks the last case of the YAML walk: a node that is neither a mapping, a list, a
// scalar, nor an alias is refused at its path and reads as nothing.
func TestNodeRefusesANodeThatIsNotAValue(t *testing.T) {
	w := &walker{}
	if v := w.node(&yaml.Node{Kind: yaml.DocumentNode}, "/x"); v != nil {
		t.Fatalf("value = %v; want nil", v)
	}
	if len(w.findings) != 1 || w.findings[0].Path != "/x" {
		t.Fatalf("findings = %v; want one finding at /x", w.findings)
	}
}

// TestReviewStatesAReplyLimitThatTheSpecSets checks the limit line when the spec sets the reply size: the value is
// shown, and no cap is named for it.
func TestReviewStatesAReplyLimitThatTheSpecSets(t *testing.T) {
	r := requireValid(t, fixtureWith("  max_pages: 5\noperations:", "  max_pages: 5\n  max_reply_bytes: 1048576\noperations:"))
	want := "limits: timeout 20s, reply 1048576 bytes, at most 5 pages"
	if !strings.Contains(Review(r.Spec), want) {
		t.Fatalf("review does not contain %q:\n%s", want, Review(r.Spec))
	}
}
