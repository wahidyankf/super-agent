package connectorspec

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"io/fs"
	"maps"
	"slices"
	"strings"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/operations"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/specfolder"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// bundledSpecs is the tree of the embedded bundled specs. E9 carries no spec yet, so the tree holds only the note of
// the folder; the ports add one spec file per bundled kind. The embedded bytes come from go:embed, so no file, flag,
// or catalog field reaches the exception of ParseBundled.
//
//go:embed bundled
var bundledSpecs embed.FS

// Rules of the embedded set. They are fixed words, so the text never names a value.
const (
	ruleKindShadow        = "shadows a bundled spec"
	ruleBundledUnreadable = "could not be read"
	ruleBundledMissing    = "leaves a row of the kind without an operation"
)

// LoadBundled loads the embedded bundled specs in name order, and it builds the set the run registers for them. E9
// carries no spec, so it loads an empty set.
func LoadBundled() (Set, error) {
	return loadBundledFrom(bundledSpecs, "bundled")
}

// loadBundledFrom reads the bundled specs of one file system, so tests feed it in-memory files. The embedded tree is
// compiled in; a failure to read it breaks the build invariant and stops start-up.
func loadBundledFrom(fsys fs.FS, dir string) (Set, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return Set{}, &Error{File: dir, Rule: ruleBundledUnreadable}
	}
	var files []specfolder.File
	for _, entry := range entries {
		if !isBundledSpecName(entry.Name()) {
			continue
		}
		data, err := fs.ReadFile(fsys, dir+"/"+entry.Name())
		if err != nil {
			return Set{}, &Error{File: entry.Name(), Rule: ruleBundledUnreadable}
		}
		sum := sha256.Sum256(data)
		files = append(files, specfolder.File{Name: entry.Name(), Data: data, Digest: hex.EncodeToString(sum[:])})
	}
	return loadBundled(files)
}

// isBundledSpecName reports whether a file name has the extension of a spec file.
func isBundledSpecName(name string) bool {
	return strings.HasSuffix(name, ".yml") || strings.HasSuffix(name, ".yaml")
}

// loadBundled validates the bundled spec files and builds the set the run registers for them. Each file goes through
// ParseBundled, so a bundled kind keeps every rule of a user spec except the built-in kind. One file holds each
// bundled kind, so a spec may not claim a row another spec of the kind already claims: a second file of the kind is
// refused, and a repeated claim inside one file never leaves ParseBundled. The claimed operations of each kind cover
// every read row of that kind in the built-in catalog, so the catalog row stays the single source of each surface. A
// bundled kind is a built-in kind, which every user set registers, so the set needs no row of its own.
func loadBundled(files []specfolder.File) (Set, error) {
	set := Set{Digests: map[string]string{}}
	var defs []domain.KindDefinition
	var models []*spec.Spec
	var names []string
	for _, file := range files {
		res := spec.ParseBundled(file.Data)
		if !res.Valid() {
			first := res.Findings[0]
			return Set{}, &Error{File: file.Name, Path: first.Path, Rule: first.Rule}
		}
		m := res.Spec
		if _, taken := set.Digests[m.Kind]; taken {
			return Set{}, &Error{File: file.Name, Path: "/kind", Rule: ruleKindDuplicate}
		}
		if _, err := domain.BuiltinKinds().Kind(m.Kind); err != nil {
			defs = append(defs, domain.KindDefinition{Name: m.Kind, UsesIdentity: m.Auth.Type == "basic"})
		}
		set.Specs = append(set.Specs, Spec{File: file.Name, Digest: file.Digest, Model: m})
		set.Digests[m.Kind] = file.Digest
		models = append(models, m)
		names = append(names, file.Name)
	}
	set.Kinds = kindSet(defs)
	for i := range set.Specs {
		set.Specs[i].Kind = mustKind(set.Kinds, set.Specs[i].Model.Kind)
	}
	set.Operations = buildCatalog(set.Kinds, nil)
	if err := checkBundledCoverage(names, models, set.Kinds); err != nil {
		return Set{}, err
	}
	return set, nil
}

// checkBundledCoverage refuses a bundled kind that leaves a catalog read row of the kind without an operation. The
// rows come from the built-in catalog in catalog order, and the first file of the kind answers for the gap, so the
// text names no value and stays put while rows move. A change operation may stay with the Go gateway until its port:
// the composite builder routes it there, so only reads must be claimed. Pull request 17 claims the Outline writes.
func checkBundledCoverage(names []string, models []*spec.Spec, kinds domain.KindSet) *Error {
	claimed := map[string]map[string]bool{}
	first := map[string]string{}
	for f, m := range models {
		if _, ok := claimed[m.Kind]; !ok {
			claimed[m.Kind] = map[string]bool{}
			first[m.Kind] = names[f]
		}
		for _, op := range m.Operations {
			if op.CatalogID != "" {
				claimed[m.Kind][op.CatalogID] = true
			}
		}
	}
	for _, kind := range slices.Sorted(maps.Keys(claimed)) {
		for _, row := range operations.RowsOf(mustKind(kinds, kind)) {
			if row.Effect != domain.EffectRead {
				continue
			}
			if !claimed[kind][string(row.ID)] {
				return &Error{File: first[kind], Rule: ruleBundledMissing}
			}
		}
	}
	return nil
}

// Merge joins the user-folder set with the embedded bundled set. A user spec whose kind the embedded set claims
// stops start-up, because the embedded bytes are the source for that kind. Otherwise the merged set holds every
// spec of both sets: the embedded specs claim built-in kinds, which the user set registers with its rows, so the
// merged set keeps the user's kinds, rows, and command rows, and adds the bundled specs and digests.
func Merge(user, bundled Set) (Set, error) {
	claimed := make(map[string]bool, len(bundled.Specs))
	for _, sp := range bundled.Specs {
		claimed[sp.Model.Kind] = true
	}
	for _, sp := range user.Specs {
		if claimed[sp.Model.Kind] {
			return Set{}, &Error{File: sp.File, Path: "/kind", Rule: ruleKindShadow}
		}
	}
	out := Set{
		Specs:      append(append([]Spec{}, user.Specs...), bundled.Specs...),
		Kinds:      user.Kinds,
		Operations: user.Operations,
		CLIRows:    append([]operations.Row{}, user.CLIRows...),
		Digests:    maps.Clone(user.Digests),
	}
	for kind, digest := range bundled.Digests {
		out.Digests[kind] = digest
	}
	return out, nil
}
