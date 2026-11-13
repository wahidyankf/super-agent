package graphql

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/parser"
)

// Query limits fixed by serve-graphql.feature. A query over a limit is refused before any
// resolver runs, so no gateway is called.
const (
	maxDepth      = 5
	maxComplexity = 100
	// upstreamCost is charged for each root field that calls the source.
	upstreamCost = 40
	// fieldCost is charged for every other field.
	fieldCost = 1
	// nonFieldCost is charged for each fragment spread and inline fragment. It keeps spread-heavy queries bounded.
	nonFieldCost = 1
)

// upstreamFields are the root fields that call the source. A root field with one of these names costs upstreamCost.
var upstreamFields = map[string]bool{
	"collections":    true,
	"collectionTree": true,
	"document":       true,
}

var (
	// errSyntax marks a query the parser rejects. The limit check lets the executor report it.
	errSyntax = errors.New("invalid query syntax")
	// errDepthLimit reports a query nested deeper than maxDepth levels.
	errDepthLimit = fmt.Errorf("query exceeds the depth limit of %d levels", maxDepth)
	// errComplexityLimit reports a query whose cost is over maxComplexity.
	errComplexityLimit = fmt.Errorf("query exceeds the complexity limit of %d points", maxComplexity)
	// errIntrospection reports a query that reads the schema. Introspection is off.
	errIntrospection = errors.New("introspection is disabled")
)

// measureQuery parses query and returns the deepest selection nesting and the total cost of all its
// operations. Depth counts the selection sets on the deepest path, so "{ collections { id } }" has depth 2.
// It stops at the first limit it passes, so a refused query reports no full numbers.
func measureQuery(query string) (depth, cost int, err error) {
	doc, perr := parser.ParseQuery(&ast.Source{Input: query})
	if perr != nil {
		return 0, 0, fmt.Errorf("%w: %v", errSyntax, perr)
	}
	m := &meter{fragments: doc.Fragments}
	for _, op := range doc.Operations {
		if err := m.walk(op.SelectionSet, 1, true); err != nil {
			return m.depth, m.cost, err
		}
	}
	return m.depth, m.cost, nil
}

// meter accumulates the depth and cost of one query.
type meter struct {
	fragments ast.FragmentDefinitionList
	depth     int
	cost      int
}

// walk visits one selection set at nesting level. atRoot is true while the set is the operation's own root,
// including through fragments spread there, so an upstream field cannot hide behind a fragment.
// An empty set is a leaf and adds no level.
func (m *meter) walk(set ast.SelectionSet, level int, atRoot bool) error {
	if len(set) == 0 {
		return nil
	}
	if level > maxDepth {
		return errDepthLimit
	}
	if level > m.depth {
		m.depth = level
	}
	for _, sel := range set {
		switch s := sel.(type) {
		case *ast.Field:
			if err := m.field(s, level, atRoot); err != nil {
				return err
			}
		case *ast.InlineFragment:
			if err := m.charge(nonFieldCost); err != nil {
				return err
			}
			if err := m.walk(s.SelectionSet, level+1, atRoot); err != nil {
				return err
			}
		case *ast.FragmentSpread:
			if err := m.charge(nonFieldCost); err != nil {
				return err
			}
			// An unknown fragment is left for the executor to report.
			if frag := m.fragments.ForName(s.Name); frag != nil {
				if err := m.walk(frag.SelectionSet, level+1, atRoot); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// field charges one field and walks its children. Introspection fields are refused at any level.
func (m *meter) field(f *ast.Field, level int, atRoot bool) error {
	if strings.HasPrefix(f.Name, "__") && f.Name != "__typename" {
		return errIntrospection
	}
	cost := fieldCost
	if atRoot && upstreamFields[f.Name] {
		cost = upstreamCost
	}
	if err := m.charge(cost); err != nil {
		return err
	}
	return m.walk(f.SelectionSet, level+1, false)
}

// charge adds points to the total cost and refuses the query once the total is over maxComplexity.
func (m *meter) charge(points int) error {
	m.cost += points
	if m.cost > maxComplexity {
		return errComplexityLimit
	}
	return nil
}

// limitQuery reads the JSON body, measures its query, and refuses one over the depth, cost, or
// introspection limit with 422. A body that is not one JSON object is refused with 400. A query that
// fails to parse is passed on, so the executor reports the syntax error.
func limitQuery() middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			data, err := readBody(r)
			if err != nil {
				writeErrors(w, http.StatusBadRequest, "could not read request body")
				return
			}
			var body struct {
				Query string `json:"query"`
			}
			if err := json.Unmarshal(data, &body); err != nil {
				writeErrors(w, http.StatusBadRequest, "invalid GraphQL request body")
				return
			}
			if _, _, err := measureQuery(body.Query); err != nil && !errors.Is(err, errSyntax) {
				writeErrors(w, http.StatusUnprocessableEntity, err.Error())
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
