package spec

import (
	"bytes"
	"encoding/json"
	"sort"
)

// withheldName stands in for a field name that is not a plain identifier, so a secret-shaped key never reaches a path.
const withheldName = "(name withheld)"

// fallbackRule is the rule of a failing check that has no fixed rule text of its own.
const fallbackRule = "does not match the rule for this field"

// finding builds a finding of the stable code at a path, with a fixed rule.
func finding(path, rule string) Finding {
	return Finding{Code: CodeSpecInvalid, Path: path, Rule: rule}
}

// Validate reads one spec document and returns its model, or the findings that refuse it. The document goes through
// three checks in order: the YAML reader, the JSON Schema, and the Go rules that compare two fields. A later check runs
// only when the earlier one passes, so each finding comes from the first layer that refuses the document.
func Validate(data []byte) Result {
	value, fs := parseDocument(data)
	if len(fs) == 0 {
		fs = checkSchema(value)
	}
	if len(fs) == 0 {
		s, modelFs := toModel(value)
		if len(modelFs) == 0 {
			modelFs = semanticFindings(s)
		}
		if len(modelFs) == 0 {
			return Result{Spec: s}
		}
		fs = modelFs
	}
	return Result{Findings: sortFindings(fs)}
}

// sortFindings orders findings by path and then by rule, and drops a repeat, so one document always gives one list.
func sortFindings(fs []Finding) []Finding {
	sort.SliceStable(fs, func(i, j int) bool {
		if fs[i].Path != fs[j].Path {
			return fs[i].Path < fs[j].Path
		}
		return fs[i].Rule < fs[j].Rule
	})
	out := make([]Finding, 0, len(fs))
	for i, f := range fs {
		if i > 0 && f == fs[i-1] {
			continue
		}
		out = append(out, f)
	}
	return out
}

// toModel decodes a checked value into the model. Each number is read as a json.Number, so an integer keeps its digits.
func toModel(value any) (*Spec, []Finding) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, []Finding{finding("", "is not a value that the spec can hold")}
	}
	var s Spec
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&s); err != nil {
		return nil, []Finding{finding("", "does not fit the spec model")}
	}
	return &s, nil
}
