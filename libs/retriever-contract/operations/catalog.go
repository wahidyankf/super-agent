package operations

import (
	"errors"
	"slices"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// Errors that NewCatalog returns. They are start-up errors of a composition root, so no adapter shows them.
var (
	// ErrInvalidRow reports a row with no operation ID.
	ErrInvalidRow = errors.New("an operation row needs an operation ID")
	// ErrDuplicateRow reports two rows with one operation ID, so Lookup would have two answers.
	ErrDuplicateRow = errors.New("duplicate operation row")
	// ErrUnregisteredRowKind reports a row whose command path does not start with a kind the set registers.
	ErrUnregisteredRowKind = errors.New("a row's command path must start with a registered kind")
)

// Catalog is an immutable set of operation rows in catalog order, with the kind set that owns them. Build one with
// NewCatalog to register rows at start-up, or take the built-in catalog with BuiltinCatalog. The package functions Rows,
// Lookup, RowsOf, and Supports read the built-in catalog.
type Catalog struct {
	kinds domain.KindSet
	rows  []Row
}

// NewCatalog checks the rows against the kind set and returns the catalog. Each row needs an operation ID that no other
// row holds, and a command path that starts with a registered kind. The catalog keeps its own copy of the rows.
func NewCatalog(kinds domain.KindSet, rows []Row) (Catalog, error) {
	seen := map[domain.OperationID]bool{}
	for _, row := range rows {
		if row.ID == "" {
			return Catalog{}, ErrInvalidRow
		}
		if _, ok := kindOf(kinds, row); !ok {
			return Catalog{}, ErrUnregisteredRowKind
		}
		if seen[row.ID] {
			return Catalog{}, ErrDuplicateRow
		}
		seen[row.ID] = true
	}
	return Catalog{kinds: kinds, rows: slices.Clone(rows)}, nil
}

// BuiltinCatalog returns the catalog of the built-in rows over the built-in kinds. Every call returns a new value.
func BuiltinCatalog() Catalog {
	return Catalog{kinds: domain.BuiltinKinds(), rows: builtinRows()}
}

// builtinRows returns the built-in rows in catalog order: the Outline reads and writes and the admin rows, then the Kiwi
// and the Jira rows. The order is the order the help text and the tool list show.
func builtinRows() []Row {
	return slices.Concat(
		rows,
		part2bRows,
		writeRows,
		adminRows,
		adminRowsC2,
		adminRowsC3,
		adminRowsC4,
		jiraRows,
		kiwiRows,
	)
}

// Rows returns a copy of the catalog rows, in catalog order.
func (c Catalog) Rows() []Row {
	return slices.Clone(c.rows)
}

// Lookup returns the row of id, and false when the catalog holds no such operation.
func (c Catalog) Lookup(id domain.OperationID) (Row, bool) {
	for _, r := range c.rows {
		if r.ID == id {
			return r, true
		}
	}
	return Row{}, false
}

// KindOf returns the kind that the row belongs to: the first word of its command path, resolved through the kind set.
// It returns false when that word names no registered kind.
func (c Catalog) KindOf(row Row) (domain.SourceKind, bool) {
	return kindOf(c.kinds, row)
}

// kindOf resolves the kind of a row through a kind set.
func kindOf(kinds domain.KindSet, row Row) (domain.SourceKind, bool) {
	path := row.Path()
	if len(path) == 0 {
		return domain.SourceKind{}, false
	}
	kind, err := kinds.Kind(path[0])
	return kind, err == nil
}

// RowsOf returns a copy of the catalog rows of one source kind, in catalog order.
func (c Catalog) RowsOf(kind domain.SourceKind) []Row {
	var out []Row
	for _, row := range c.rows {
		if k, ok := c.KindOf(row); ok && k == kind {
			out = append(out, row)
		}
	}
	return out
}

// Supports reports whether a source kind runs the row. A row runs only on its own kind. The Kiwi and Jira kinds also run
// only the rows of their closed lists, so a Kiwi or a Jira row that the list does not name runs on neither.
func (c Catalog) Supports(kind domain.SourceKind, row Row) bool {
	owner, ok := c.KindOf(row)
	if !ok || owner != kind {
		return false
	}
	switch kind {
	case domain.KindKiwi:
		return slices.Contains(kiwiOperations, row.ID)
	case domain.KindJira:
		return slices.Contains(jiraOperations, row.ID)
	}
	return true
}
