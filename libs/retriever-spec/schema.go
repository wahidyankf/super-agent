package spec

import (
	"bytes"
	_ "embed"
	"errors"
	"regexp"
	"strconv"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
)

// schemaSource is the JSON Schema file of the connector spec, version 1. It is embedded, so no file is read.
//
//go:embed schema/connector-spec-v1.schema.json
var schemaSource []byte

// schemaURL names the embedded schema inside the compiler. It is a fixed name that is never fetched.
const schemaURL = "https://example.test/retriever-spec/connector-spec-v1.schema.json"

// specSchema is the compiled schema. It compiles once, when the package loads.
var specSchema = mustCompile(schemaSource)

// safeNamePattern matches a field name that a finding path may show. A name outside it is withheld.
var safeNamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_.-]{0,30}$`)

// rules maps a failing JSON Schema keyword to its fixed rule text. A keyword outside the map falls back to fallbackRule.
var rules = map[string]string{
	"required":             "is missing a field that the spec requires",
	"additionalProperties": "is a field that this spec version does not allow",
	"type":                 "has the wrong kind of value",
	"enum":                 "is not one of the values this spec version allows",
	"const":                "is not the value this spec version requires",
	"pattern":              "does not have the form this field allows",
	"not":                  "has a form that this spec refuses",
	"minLength":            "is shorter than this field allows",
	"maxLength":            "is longer than this field allows",
	"minimum":              "is below the lowest value this field allows",
	"maximum":              "is above the highest value this field allows",
	"minItems":             "has fewer items than this field allows",
	"maxItems":             "has more items than this field allows",
	"minProperties":        "has fewer fields than this field allows",
	"maxProperties":        "has more fields than this field allows",
	"uniqueItems":          "repeats an item",
	"anyOf":                "does not match any form this field allows",
	"oneOf":                "does not match exactly one form this field allows",
}

// Schema returns a copy of the JSON Schema file of the connector spec, version 1.
func Schema() []byte {
	return append([]byte(nil), schemaSource...)
}

// errOutsideSchema is what the loader returns for a reference to a document outside the schema.
var errOutsideSchema = errors.New("the schema refers to a document outside itself")

// refusingLoader refuses every reference to a document outside the schema, so the compiler reads no file and no network.
type refusingLoader struct{}

// Load refuses the URL.
func (refusingLoader) Load(string) (any, error) {
	return nil, errOutsideSchema
}

// compileSchema compiles a JSON Schema document under the name that url gives it. The compiler has no loader that reads
// outside the document, so no file or network is read.
func compileSchema(url string, src []byte) (*jsonschema.Schema, error) {
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(src))
	if err != nil {
		return nil, err
	}
	c := jsonschema.NewCompiler()
	c.UseLoader(refusingLoader{})
	if err := c.AddResource(url, doc); err != nil {
		return nil, err
	}
	return c.Compile(url)
}

// mustCompile compiles the embedded schema and panics when it does not compile, which a test of the package catches.
func mustCompile(src []byte) *jsonschema.Schema {
	s, err := compileSchema(schemaURL, src)
	if err != nil {
		panic("the embedded connector spec schema does not compile: " + err.Error())
	}
	return s
}

// checkSchema checks a value that the parser read against the JSON Schema. It returns one finding per broken field.
func checkSchema(value any) []Finding {
	err := specSchema.Validate(value)
	if err == nil {
		return nil
	}
	return findingsOf(err, value)
}

// findingsOf turns the error of the schema into findings, one for each broken field, at the path of that field.
func findingsOf(err error, value any) []Finding {
	var verr *jsonschema.ValidationError
	if !errors.As(err, &verr) {
		return []Finding{finding("", fallbackRule)}
	}
	var out []Finding
	collect(verr, value, &out)
	return out
}

// collect walks the error tree and appends a finding for each leaf. An any-of failure descends into the branches that
// fail for a reason other than the JSON type, so a broken field inside an object is named at its own path. When every
// branch fails on the type, the any-of failure is one finding at its field.
func collect(e *jsonschema.ValidationError, value any, out *[]Finding) {
	at := pointerOf(value, e.InstanceLocation)
	switch k := e.ErrorKind.(type) {
	case *kind.AnyOf, *kind.OneOf:
		var branches []*jsonschema.ValidationError
		for _, cause := range e.Causes {
			if !onlyTypeMismatch(cause) {
				branches = append(branches, cause)
			}
		}
		if len(branches) == 0 {
			*out = append(*out, finding(at, ruleFor(keywordOf(k))))
			return
		}
		for _, cause := range branches {
			collect(cause, value, out)
		}
		return
	case *kind.Required:
		for _, name := range k.Missing {
			*out = append(*out, finding(at+"/"+safeName(name), ruleFor("required")))
		}
		return
	case *kind.AdditionalProperties:
		for _, name := range k.Properties {
			*out = append(*out, finding(at+"/"+safeName(name), ruleFor("additionalProperties")))
		}
		return
	}
	if len(e.Causes) > 0 {
		for _, cause := range e.Causes {
			collect(cause, value, out)
		}
		return
	}
	*out = append(*out, finding(at, ruleFor(keywordOf(e.ErrorKind))))
}

// onlyTypeMismatch reports whether a failure is only the wrong JSON type of the value, which is every leaf under it being
// a type check. A branch of an any-of that fails for that reason alone says nothing about the field.
func onlyTypeMismatch(e *jsonschema.ValidationError) bool {
	if _, ok := e.ErrorKind.(*kind.Type); ok {
		return true
	}
	for _, cause := range e.Causes {
		if !onlyTypeMismatch(cause) {
			return false
		}
	}
	return len(e.Causes) > 0
}

// keywordOf returns the JSON Schema keyword of a failure, or "" when the kind names none.
func keywordOf(k jsonschema.ErrorKind) string {
	if _, ok := k.(*kind.Not); ok {
		return "not"
	}
	if path := k.KeywordPath(); len(path) > 0 {
		return path[0]
	}
	return ""
}

// pointerOf returns the JSON pointer of the value at the location that the schema error names. A key that is not a
// plain name is withheld, and an index is kept as it is.
func pointerOf(value any, location []string) string {
	var at string
	cur := value
	for _, seg := range location {
		switch v := cur.(type) {
		case []any:
			i, err := strconv.Atoi(seg)
			if err != nil || i < 0 || i >= len(v) {
				return at + "/" + withheldName
			}
			at += "/" + seg
			cur = v[i]
		case map[string]any:
			at += "/" + safeName(seg)
			cur = v[seg]
		default:
			return at + "/" + withheldName
		}
	}
	return at
}

// safeName returns a field name that a finding path may show, or withheldName when the name is not a plain identifier.
func safeName(name string) string {
	if safeNamePattern.MatchString(name) {
		return name
	}
	return withheldName
}

// ruleFor returns the fixed rule text of a failing keyword, or fallbackRule when the keyword has none.
func ruleFor(keyword string) string {
	if rule, ok := rules[keyword]; ok {
		return rule
	}
	return fallbackRule
}
