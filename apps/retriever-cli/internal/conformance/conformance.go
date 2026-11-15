// Package conformance checks a connector spec the way PR21 (ADR 001 §10 via
// ADR 002 §21) requires: the schema is valid, every operation carries
// examples, each example replays through the engine and matches its expected
// fields, and derived texts equal their stored form. It reads no file, no
// environment variable, no clock, and no network: the caller passes the spec
// bytes in, and the replay answers from the canned exchanges of the examples.
// No finding, issue, or report line holds a value of the spec, so a
// credential-shaped string never reaches a message; Redact replaces the
// secrets the caller names with [REDACTED].
package conformance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"strings"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine/replay"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
	"go.yaml.in/yaml/v3"
)

// redacted is the only text a secret is ever replaced with.
const redacted = "[REDACTED]"

// ExampleDomains are the only DNS domains a bundled spec may name: the three
// documentation domains of C3 (ADR 001, 003-plugin-and-supply-chain-threats).
// A user spec may name the host it really talks to, so CheckHosts runs only
// in the suite over the bundled specs, never in connector validate or test.
var ExampleDomains = []string{"example.test", "example.invalid", "example.com"}

// Rules are the fixed words of an issue. No rule holds a value of the spec.
const (
	ruleNoExample       = "has no example"
	ruleOutsideExample  = "names a host outside the example domains"
	ruleNoReplay        = "does not replay through the engine"
	ruleFieldUnproduced = "expects a field the reply does not produce"
	ruleFieldMismatch   = "expects a field value the reply does not produce"
)

// Issue is one way a spec fails the suite. Op names the operation, Example
// is the index of the example at fault (-1 when no example is at fault),
// Field names the expected field at fault ("" when none is), Path is the JSON
// pointer of the field at fault, and Rule states the broken rule in fixed
// words. No issue holds a value of the spec. The zero value names no example:
// leave Example at NoExample unless an example is at fault.
type Issue struct {
	Op      string
	Example int
	Field   string
	Path    string
	Rule    string
}

// NoExample marks an issue that faults no example: the operation, the
// document, or the host is at fault, not one example of it.
const NoExample = -1

// String returns the issue as one line: the operation, the example, the path,
// and the rule, naming the field when one is at fault.
func (i Issue) String() string {
	where := i.Path
	if where == "" {
		where = "(document)"
	}
	if i.Example >= 0 && i.Field != "" {
		return fmt.Sprintf("op %s example %d field %s: %s: %s", i.Op, i.Example, i.Field, where, i.Rule)
	}
	if i.Example >= 0 {
		return fmt.Sprintf("op %s example %d: %s: %s", i.Op, i.Example, where, i.Rule)
	}
	return fmt.Sprintf("op %s: %s: %s", i.Op, where, i.Rule)
}

// Validate parses one spec document of the user folder and checks the
// static half of the suite: the schema and the semantic rules must pass, and
// every operation must carry at least one example. The schema already requires
// the examples array, but its finding names only the index, so the explicit
// check names the operation as gate 1 requires. It returns the model when no
// issue stops it, and nil with the issues otherwise.
func Validate(data []byte) (*spec.Spec, []Issue) {
	res := spec.Parse(data)
	if !res.Valid() {
		return nil, findingsAsIssues(res.Findings, data)
	}
	return res.Spec, CheckModel(res.Spec)
}

// findingsAsIssues turns spec findings into issues, naming the operation each
// finding points at.
func findingsAsIssues(findings []spec.Finding, data []byte) []Issue {
	issues := make([]Issue, 0, len(findings))
	for _, f := range findings {
		issues = append(issues, Issue{Op: opAt(f.Path, data), Example: NoExample, Path: f.Path, Rule: f.Rule})
	}
	return issues
}

// CheckModel checks the static rule a parsed model must still meet on its
// own: every operation carries at least one example, naming the operation as
// gate 1 requires.
func CheckModel(s *spec.Spec) []Issue {
	var issues []Issue
	for i, op := range s.Operations {
		if len(op.Examples) == 0 {
			issues = append(issues, Issue{
				Op:      op.ID,
				Example: NoExample,
				Path:    fmt.Sprintf("/operations/%d/examples", i),
				Rule:    ruleNoExample,
			})
		}
	}
	return issues
}

// opAt names the operation a finding path points at, or "" when the path
// names no operation. A finding of an invalid document carries no model, so
// the ID is read from the document itself with a light decode that takes only
// the operation IDs, never a value. A document that does not decode, or an
// index past its operations, names no operation.
func opAt(path string, data []byte) string {
	var idx int
	if _, err := fmt.Sscanf(path, "/operations/%d", &idx); err != nil {
		return ""
	}
	var doc struct {
		Operations []struct {
			ID string `yaml:"id"`
		} `yaml:"operations"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return ""
	}
	if idx < 0 || idx >= len(doc.Operations) {
		return ""
	}
	return doc.Operations[idx].ID
}

// CheckHosts refuses a spec whose endpoint suffix names a host outside the
// example domains. It runs only in the suite over the bundled specs (C3): a
// user spec names the host it really talks to.
func CheckHosts(s *spec.Spec) []Issue {
	var issues []Issue
	for i, suffix := range s.Endpoint.HostSuffix {
		if !withinExampleDomains(suffix) {
			issues = append(issues, Issue{
				Example: NoExample,
				Path:    fmt.Sprintf("/endpoint/host_suffix/%d", i),
				Rule:    ruleOutsideExample,
			})
		}
	}
	return issues
}

// withinExampleDomains reports whether a host suffix stays inside the example
// domains: the suffix with its leading dot stripped equals one of them, or
// ends with a dot and one of them, so sub.example.test passes and
// example.test.evil.test fails.
func withinExampleDomains(suffix string) bool {
	host := strings.TrimPrefix(suffix, ".")
	for _, d := range ExampleDomains {
		if host == d || strings.HasSuffix(host, "."+d) {
			return true
		}
	}
	return false
}

// Replayed is the mapped output of one example that replayed without an
// engine failure: the operation, the example index, and the values of the
// expected fields. The report prints them, so a credential planted in a canned
// reply is visible and must read [REDACTED].
type Replayed struct {
	Op      string
	Example int
	Fields  map[string]any
}

// Test replays every example of a validated spec through the engine and
// checks the mapped output against the expected fields. The replay answers
// from the canned exchanges, so no socket is opened. Each example runs
// against its own canned reply, even when two examples share a method and a
// target: the script names the reply of the example that runs, so a read over
// POST never answers with the reply of another operation. A list read matches
// when its first item carries the expected fields; an empty list with a
// non-empty expectation is an issue. Every issue names the operation, the
// example, and the field at fault. Every example that runs without an engine
// failure also returns its mapped values, even when an expected field
// mismatches.
func Test(s *spec.Spec) ([]Issue, []Replayed) {
	stub, err := replay.New(s)
	if err != nil {
		return []Issue{{Example: NoExample, Path: "(document)", Rule: ruleNoReplay}}, nil
	}
	eng, err := testEngine(s, stub)
	if err != nil {
		return []Issue{{Example: NoExample, Path: "(document)", Rule: ruleNoReplay}}, nil
	}
	var issues []Issue
	var replayed []Replayed
	for i, op := range s.Operations {
		for j, ex := range op.Examples {
			if scriptErr := scriptExample(stub, s, op, ex); scriptErr != nil {
				issues = append(issues, Issue{Op: op.ID, Example: j,
					Path: fmt.Sprintf("/operations/%d/examples/%d", i, j), Rule: ruleNoReplay})
				continue
			}
			ii, rr := replayExample(eng, i, op, j, ex)
			issues = append(issues, ii...)
			if rr != nil {
				replayed = append(replayed, *rr)
			}
		}
	}
	return issues, replayed
}

// scriptExample names the canned reply of one example for the run: the method
// and the target its request renders, with the status, the headers, and the
// body of its canned reply. A later page of a paged read answers with the
// same reply, the way the shared canned map answers it.
func scriptExample(stub *replay.Stub, s *spec.Spec, op spec.Operation, ex spec.Example) error {
	method, target, err := engine.ExampleTarget(s, op, plainRequestMap(ex.Request))
	if err != nil {
		return err
	}
	body, err := json.Marshal(ex.Response.Body)
	if err != nil {
		return err
	}
	if ex.Response.Body == nil {
		body = nil
	}
	header := make(http.Header, len(ex.Response.Headers))
	for name, value := range ex.Response.Headers {
		header.Set(name, value)
	}
	stub.Script(method, target, replay.Reply{Status: ex.Response.Status, Header: header, Body: body})
	return nil
}

// replayExample runs one example and compares its mapped output with the
// expectation. A replay failure, and every expected field the reply does not
// produce, is an issue naming the operation and the example. The mapped
// values return with it when the example runs, so the report shows what
// replayed.
func replayExample(eng *engine.Engine, i int, op spec.Operation, j int, ex spec.Example) ([]Issue, *Replayed) {
	base := fmt.Sprintf("/operations/%d/examples/%d/expect", i, j)
	rec, err := runExample(eng, op, ex)
	if err != nil {
		return []Issue{{Op: op.ID, Example: j, Path: base, Rule: ruleNoReplay}}, nil
	}
	out := &Replayed{Op: op.ID, Example: j, Fields: map[string]any{}}
	var issues []Issue
	for _, key := range sortedKeys(ex.Expect) {
		got, ok := lookupField(rec, key)
		if !ok {
			issues = append(issues, Issue{Op: op.ID, Example: j, Field: key, Path: base, Rule: ruleFieldUnproduced})
			continue
		}
		out.Fields[key] = got
		if !equalValue(got, ex.Expect[key]) {
			issues = append(issues, Issue{Op: op.ID, Example: j, Field: key, Path: base, Rule: ruleFieldMismatch})
		}
	}
	return issues, out
}

// runExample sends one example through the engine: a read runs Read, and a
// write or admin operation runs Write once, as production does. The request
// values go through the number normalization the engine applies to an
// example: a JSON number reads as a Go number, so the binder sees the value
// the document means.
func runExample(eng *engine.Engine, op spec.Operation, ex spec.Example) (domain.Record, error) {
	args := domain.NewArgs(plainRequestMap(ex.Request))
	if op.Access == "read" {
		return eng.Read(context.Background(), domain.OperationID(op.ID), args)
	}
	return eng.Write(context.Background(), domain.OperationID(op.ID), args)
}

// plainRequestMap returns the request values the engine binds: every JSON
// number as a Go number, recursing into maps and lists.
func plainRequestMap(request map[string]any) map[string]any {
	out := make(map[string]any, len(request))
	for k, v := range request {
		out[k] = plainRequest(v)
	}
	return out
}

// plainRequest returns one request value the engine binds: a JSON number as
// an integer when it reads as one and as a float otherwise, recursing into
// maps and lists. Any other value passes through.
func plainRequest(v any) any {
	switch n := v.(type) {
	case json.Number:
		if i, err := n.Int64(); err == nil {
			return int(i)
		}
		if f, err := n.Float64(); err == nil {
			return f
		}
		return v
	case map[string]any:
		out := make(map[string]any, len(n))
		for k, e := range n {
			out[k] = plainRequest(e)
		}
		return out
	case []any:
		out := make([]any, len(n))
		for i, e := range n {
			out[i] = plainRequest(e)
		}
		return out
	}
	return v
}

// lookupField returns the mapped output for one expected field. A list read
// answers with its first item, because the examples pin one item each; an
// empty list produces no field. A nested expectation descends into nested
// records.
func lookupField(rec domain.Record, key string) (any, bool) {
	if rec.IsList() {
		items := rec.Items()
		if len(items) == 0 {
			return nil, false
		}
		rec = items[0]
	}
	return lookupRecord(rec, key)
}

// lookupRecord reads one field of one record, descending into a nested record
// when the value holds one.
func lookupRecord(rec domain.Record, key string) (any, bool) {
	v, ok := rec.Get(key)
	if !ok {
		return nil, false
	}
	if nested, isRecord := v.(domain.Record); isRecord {
		return recordMap(nested), true
	}
	return v, true
}

// recordMap returns the fields of a record as a map, with nested records as
// maps too.
func recordMap(rec domain.Record) map[string]any {
	out := map[string]any{}
	for _, f := range rec.Fields() {
		if nested, ok := f.Value.(domain.Record); ok {
			out[f.Name] = recordMap(nested)
			continue
		}
		out[f.Name] = f.Value
	}
	return out
}

// equalValue reports whether a mapped value equals an expected one. Both go
// through JSON, so a YAML integer and a decoded float compare by value, and
// objects compare by content. A nil expectation matches JSON null only.
func equalValue(got, want any) bool {
	g, err := json.Marshal(got)
	if err != nil {
		return false
	}
	w, err := json.Marshal(want)
	if err != nil {
		return false
	}
	// g and w are produced by Marshal above, so they always parse. The errors
	// cannot fire and are not checked; an unparsable value would leave a nil,
	// which never equals a mapped field.
	var gv, wv any
	_ = json.Unmarshal(g, &gv)
	_ = json.Unmarshal(w, &wv)
	return reflect.DeepEqual(gv, wv)
}

// sortedKeys returns the keys of a map in order, so the issues of one example
// read the same way on every run.
func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Redact replaces every occurrence of each non-empty secret in text with
// [REDACTED]. An empty secret changes nothing.
func Redact(text string, secrets []string) string {
	for _, secret := range secrets {
		if secret == "" {
			continue
		}
		text = strings.ReplaceAll(text, secret, redacted)
	}
	return text
}

// AssembleValidation returns the validation report of one spec: the file's
// base name, the counts, the validation issues, and the review a person reads
// before the engine runs the spec (control C1). The caller passes the secrets
// the spec declares, and every occurrence reads [REDACTED].
func AssembleValidation(file string, s *spec.Spec, issues []Issue, review string, secrets []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "conformance %s, kind %s\n", file, s.Kind)
	fmt.Fprintf(&b, "operations: %d\n", len(s.Operations))
	fmt.Fprintf(&b, "examples: %d\n", countExamples(s))
	writeIssues(&b, "validate", issues)
	fmt.Fprintf(&b, "readme:\n%s", indent(review))
	return Redact(b.String(), secrets)
}

// Derived holds the three texts the suite derives from a spec: the help of
// its command-line surface, the JSON of its MCP tools, and the plain-text
// review a person reads. The caller renders them with the real adapters; this
// package only assembles and redacts them.
type Derived struct {
	CLI    string
	MCP    string
	README string
}

// countExamples returns the examples of every operation of a spec.
func countExamples(s *spec.Spec) int {
	examples := 0
	for _, op := range s.Operations {
		examples += len(op.Examples)
	}
	return examples
}

// Assemble returns the conformance report of one spec: the file's base name,
// the counts, the validation and replay issues, the mapped values of every
// example that ran, and the three derived texts. The caller passes the secrets
// the spec declares, and every occurrence reads [REDACTED]. The report is
// deterministic: issues print in operation order, and expected fields in name
// order with their keys sorted by the JSON encoding.
func Assemble(file string, s *spec.Spec, validateIssues, replayIssues []Issue, replayed []Replayed, derived Derived, secrets []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "conformance %s, kind %s\n", file, s.Kind)
	fmt.Fprintf(&b, "operations: %d\n", len(s.Operations))
	fmt.Fprintf(&b, "examples: %d\n", countExamples(s))
	writeIssues(&b, "validate", validateIssues)
	writeIssues(&b, "replay", replayIssues)
	writeReplayed(&b, replayed)
	fmt.Fprintf(&b, "cli:\n%s", indent(derived.CLI))
	fmt.Fprintf(&b, "mcp:\n%s", indent(derived.MCP))
	fmt.Fprintf(&b, "readme:\n%s", indent(derived.README))
	return Redact(b.String(), secrets)
}

// writeReplayed writes the mapped values of every example that ran, one line
// per example with its fields as canonical JSON. An example that never ran
// prints no line.
func writeReplayed(b *strings.Builder, replayed []Replayed) {
	if len(replayed) == 0 {
		return
	}
	for _, r := range replayed {
		fields, err := json.Marshal(r.Fields)
		if err != nil {
			fields = []byte("{}")
		}
		fmt.Fprintf(b, "  op %s example %d: %s\n", r.Op, r.Example, fields)
	}
}

// writeIssues writes one section of issues: ok when empty, else one line per
// issue.
func writeIssues(b *strings.Builder, name string, issues []Issue) {
	if len(issues) == 0 {
		fmt.Fprintf(b, "%s: ok\n", name)
		return
	}
	fmt.Fprintf(b, "%s: %d issues\n", name, len(issues))
	for _, issue := range issues {
		fmt.Fprintf(b, "  %s\n", issue.String())
	}
}

// indent indents every non-empty line of text by two spaces, so the derived
// texts sit under their section, and ends the section with a newline.
func indent(text string) string {
	if text == "" {
		return "\n"
	}
	var b strings.Builder
	for _, line := range strings.Split(text, "\n") {
		if line == "" {
			b.WriteString("\n")
			continue
		}
		b.WriteString("  " + line + "\n")
	}
	return b.String()
}
