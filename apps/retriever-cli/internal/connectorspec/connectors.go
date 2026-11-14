// Package connectorspec turns the connector spec files of one folder into what a run registers at start-up: the kinds, the
// operation rows that the CLI and the MCP tools are built from, and the validated models that the engine runs. It is the
// only app code that joins a spec to the contract kinds, the operation catalog, and the engine. It reads no file: the
// composition root passes in the files that the runtime read, so the rules here stay pure.
package connectorspec

import (
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"slices"
	"strings"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/operations"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/specfolder"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// commandWords are the first words of the built-in commands. A spec kind may not take one, because the kind is the first
// word of a command path.
var commandWords = []string{"document", "help", "mcp", "retriever", "serve", "sources"}

// Rules of a kind that a spec may not use. They are fixed words, so the text never names a value.
const (
	ruleKindReserved   = "collides with a built-in kind or a command word"
	ruleKindDuplicate  = "collides with another spec's kind"
	ruleCommandTaken   = "collides with another command"
	ruleToolTaken      = "collides with another tool"
	ruleListFieldEmpty = "a list must map at least one field"
)

// Error reports a spec file that stops start-up. File is the file name, never a path, Path is the JSON pointer of the
// field at fault, and Rule is the broken rule in fixed words. No text of the error holds a value of the spec.
type Error struct {
	File string
	Path string
	Rule string
}

// Error returns the file, the field path, and the rule, such as "connector spec file x.yml: /kind: is not a valid kind
// name". A whole-document finding names "(document)" in place of the path.
func (e *Error) Error() string {
	where := e.Path
	if where == "" {
		where = "(document)"
	}
	return fmt.Sprintf("connector spec file %s: %s: %s", e.File, where, e.Rule)
}

// ErrorCode returns the stable code of a refused spec, retriever.connector.spec_invalid.
func (e *Error) ErrorCode() string {
	return spec.CodeSpecInvalid
}

// Spec is one loaded spec: the name of its file, the digest of the bytes that were parsed, and its validated model.
type Spec struct {
	// File is the name of the spec file, without its folder.
	File string
	// Digest is the lower-case hex SHA-256 of the bytes that Model was parsed from.
	Digest string
	// Model is the validated spec.
	Model *spec.Spec
	// Kind is the source kind the spec declares, which the engine serves.
	Kind domain.SourceKind
}

// Set is what a folder of specs registers at start-up. Kinds holds the three built-in kinds and every spec kind.
// Operations holds the built-in operation rows and the spec rows, and it is the catalog the runtime reads. CLIRows holds
// the spec rows that the command-line tool offers, and a row whose tool is off has an empty Tool. Digests maps each spec
// kind to the digest of its file, which a catalog spec_sha256 pin is checked against.
type Set struct {
	Specs      []Spec
	Kinds      domain.KindSet
	Operations operations.Catalog
	CLIRows    []operations.Row
	Digests    map[string]string
}

// builtinKinds lists the three built-in kinds with the fields a kind definition takes.
func builtinKinds() []domain.SourceKind {
	return []domain.SourceKind{domain.KindOutline, domain.KindKiwi, domain.KindJira}
}

// isReserved reports whether a kind name is the first word of a built-in command. The spec module already refuses a
// built-in kind name, so only the command words need this check.
func isReserved(kind string) bool {
	return slices.Contains(commandWords, kind)
}

// Load validates each spec file in the order given, and it builds the Set from the files that pass. The first failure
// stops the load and is returned as an *Error, which names the file and the field, and never a value.
func Load(files []specfolder.File) (Set, error) {
	set := Set{Digests: map[string]string{}}
	var defs []domain.KindDefinition
	var rows []operations.Row
	cliPaths := map[string]bool{}
	toolNames := map[string]bool{}
	for _, file := range files {
		res := spec.Validate(file.Data)
		if !res.Valid() {
			first := res.Findings[0]
			return Set{}, &Error{File: file.Name, Path: first.Path, Rule: first.Rule}
		}
		m := res.Spec
		if isReserved(m.Kind) {
			return Set{}, &Error{File: file.Name, Path: "/kind", Rule: ruleKindReserved}
		}
		if _, taken := set.Digests[m.Kind]; taken {
			return Set{}, &Error{File: file.Name, Path: "/kind", Rule: ruleKindDuplicate}
		}
		specRows, cliRows, err := rowsOf(file.Name, m, cliPaths, toolNames)
		if err != nil {
			return Set{}, err
		}
		rows = append(rows, specRows...)
		set.CLIRows = append(set.CLIRows, cliRows...)
		set.Specs = append(set.Specs, Spec{File: file.Name, Digest: file.Digest, Model: m})
		set.Digests[m.Kind] = file.Digest
		defs = append(defs, domain.KindDefinition{Name: m.Kind, UsesIdentity: m.Auth.Type == "basic"})
	}
	set.Kinds = kindSet(defs)
	for i := range set.Specs {
		set.Specs[i].Kind = mustKind(set.Kinds, set.Specs[i].Model.Kind)
	}
	set.Operations = buildCatalog(set.Kinds, rows)
	return set, nil
}

// mustKind returns the kind of a validated spec name from the set the run registers. The spec module checks the grammar of
// the name, and Load registers every spec kind, so a failure here is a broken invariant, and it fails loudly.
func mustKind(kinds domain.KindSet, name string) domain.SourceKind {
	kind, err := kinds.Kind(name)
	if err != nil {
		panic(err)
	}
	return kind
}

// kindSet returns the set of the built-in kinds and the spec kinds. The spec module refuses a kind name outside the
// contract grammar and a built-in name, and Load refuses a repeated name, so the set cannot fail on input. A failure here
// is a broken invariant, so it fails loudly.
func kindSet(defs []domain.KindDefinition) domain.KindSet {
	all := make([]domain.KindDefinition, 0, len(builtinKinds())+len(defs))
	for _, b := range builtinKinds() {
		all = append(all, domain.KindDefinition{Name: b.String(), Display: b.Display(), UsesIdentity: b.UsesIdentity()})
	}
	kinds, err := domain.NewKindSet(append(all, defs...)...)
	if err != nil {
		panic(fmt.Errorf("connectors: the kind set broke its invariant: %w", err))
	}
	return kinds
}

// buildCatalog checks the rows against the kind set and returns the catalog that holds them. The rows of the built-in
// catalog come first, so the built-in order stays the order the help text shows. Operation IDs are unique within a spec,
// and every spec kind is unique, so the catalog cannot refuse the rows on input. A refusal here is a broken invariant, so
// it fails loudly.
func buildCatalog(kinds domain.KindSet, specRows []operations.Row) operations.Catalog {
	catalog, err := operations.NewCatalog(kinds, append(operations.Rows(), specRows...))
	if err != nil {
		panic(fmt.Errorf("connectors: the operation catalog broke its invariant: %w", err))
	}
	return catalog
}

// rowsOf returns the catalog rows of one spec and the rows its command-line tool offers. It refuses an operation whose ID
// does not start with the kind, a command path or a tool name that another operation already holds, and a list that maps
// no field. cliPaths and toolNames hold the names of the operations loaded so far, and they are updated here.
func rowsOf(file string, m *spec.Spec, cliPaths, toolNames map[string]bool) ([]operations.Row, []operations.Row, error) {
	var rows, cliRows []operations.Row
	for i, op := range m.Operations {
		base := fmt.Sprintf("/operations/%d", i)
		row := rowOf(m.Kind, op)
		if row.IsList() && len(row.Fields) == 0 {
			return nil, nil, &Error{File: file, Path: base + "/response/fields", Rule: ruleListFieldEmpty}
		}
		if op.CLI.Enabled {
			if cliPaths[row.CLI] {
				return nil, nil, &Error{File: file, Path: base + "/cli", Rule: ruleCommandTaken}
			}
			cliPaths[row.CLI] = true
			cliRows = append(cliRows, row)
		}
		if op.MCP.Enabled {
			if toolNames[row.Tool] {
				return nil, nil, &Error{File: file, Path: base + "/mcp", Rule: ruleToolTaken}
			}
			toolNames[row.Tool] = true
		} else {
			row.Tool = ""
		}
		rows = append(rows, row)
	}
	return rows, cliRows, nil
}

// rowOf returns the catalog row of one spec operation. Its command path starts with the kind, so the catalog can resolve
// the kind of every row, even one whose command the spec turns off. Its tool name is set here, and the caller clears it
// when the spec turns the tool off.
func rowOf(kind string, op spec.Operation) operations.Row {
	row := operations.Row{
		ID:          domain.OperationID(op.ID),
		CLI:         kind + " " + commandPath(kind, op),
		Tool:        toolName(op),
		Summary:     op.Summary,
		Description: op.Summary,
		Effect:      effectOf(op),
		OpensWorld:  true,
		Fields:      slices.Sorted(maps.Keys(op.Response.Fields)),
	}
	for _, p := range op.Params {
		row.Params = append(row.Params, paramOf(p))
	}
	if op.Response.Items != "" {
		row.ListKey = "items"
		row.Noun = "items"
		row.Columns = row.Fields
	}
	return row
}

// commandPath returns the words of the command path below the kind. A custom CLI name replaces the default, which is the
// group and the verb of the operation ID.
func commandPath(kind string, op spec.Operation) string {
	if op.CLI.Name != "" {
		return op.CLI.Name
	}
	return strings.ReplaceAll(strings.TrimPrefix(op.ID, kind+"."), ".", " ")
}

// toolName returns the MCP tool name of an operation: its custom name, or its ID with dots and hyphens made underscores.
func toolName(op spec.Operation) string {
	if op.MCP.Name != "" {
		return op.MCP.Name
	}
	return strings.NewReplacer(".", "_", "-", "_").Replace(op.ID)
}

// effectOf returns the effect of an operation from its access label and its method, so the MCP hints follow the label.
// A read has no effect. A delete and an admin operation are destructive. A POST adds, and a PUT or a PATCH updates.
func effectOf(op spec.Operation) domain.Effect {
	switch {
	case op.Access == "read":
		return domain.EffectRead
	case op.Access == "admin", op.Method == "DELETE":
		return domain.EffectDestructive
	case op.Method == "POST":
		return domain.EffectAdditive
	}
	return domain.EffectUpdate
}

// paramOf returns the catalog parameter of one spec parameter. A missing bound is the widest one, as the engine has it.
func paramOf(p spec.Param) operations.Param {
	param := operations.Param{
		Name:     p.Name,
		Kind:     operations.Kind(p.Type),
		Required: p.Required || p.In == "path",
		Min:      math.MinInt,
		Max:      math.MaxInt,
		Values:   p.Values,
		Help:     p.Help,
	}
	if p.Default != nil {
		param.Default = plainValue(p.Default)
	}
	if p.Min != nil {
		param.Min = *p.Min
	}
	if p.Max != nil {
		param.Max = *p.Max
	}
	return param
}

// plainValue returns a default as a plain Go value. The spec reader keeps numbers as json.Number, so an integer default
// becomes an int and any other number becomes a float.
func plainValue(v any) any {
	n, ok := v.(json.Number)
	if !ok {
		return v
	}
	if i, err := n.Int64(); err == nil {
		return int(i)
	}
	if f, err := n.Float64(); err == nil {
		return f
	}
	return v
}
