package kiwi

import (
	"context"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// lookupTables holds the category, priority, and status tables of one Kiwi server, keyed by ID. A case names its
// category, priority, and status by ID, and a document shows their names (ADR 009 section 5).
type lookupTables struct {
	categories map[int]wireCategory
	priorities map[int]wirePriority
	statuses   map[int]wireStatus
}

// filterRows runs one filter method with its query and decodes the rows. The query is the named "query" object.
func filterRows[T any](ctx context.Context, g *Gateway, method string, query map[string]any) ([]T, error) {
	raw, err := g.call(ctx, method, map[string]any{"query": query})
	if err != nil {
		return nil, err
	}
	return decodeList[T](raw)
}

// loadLookups reads the three tables with an unfiltered query, once per gateway, and keeps them. A failed read is not
// kept, so the next call reads again.
func (g *Gateway) loadLookups(ctx context.Context) (*lookupTables, error) {
	g.lookupMu.Lock()
	defer g.lookupMu.Unlock()
	if g.lookups != nil {
		return g.lookups, nil
	}
	categories, err := filterRows[wireCategory](ctx, g, methodCategoryFilter, map[string]any{})
	if err != nil {
		return nil, err
	}
	priorities, err := filterRows[wirePriority](ctx, g, methodPriorityFilter, map[string]any{})
	if err != nil {
		return nil, err
	}
	statuses, err := filterRows[wireStatus](ctx, g, methodTestCaseStatusFilter, map[string]any{})
	if err != nil {
		return nil, err
	}
	tables := &lookupTables{
		categories: make(map[int]wireCategory, len(categories)),
		priorities: make(map[int]wirePriority, len(priorities)),
		statuses:   make(map[int]wireStatus, len(statuses)),
	}
	for _, row := range categories {
		tables.categories[row.ID] = row
	}
	for _, row := range priorities {
		tables.priorities[row.ID] = row
	}
	for _, row := range statuses {
		tables.statuses[row.ID] = row
	}
	g.lookups = tables
	return tables, nil
}

// resolveName returns the ID of the one row that a name query matches. Zero or several matches are ErrUnresolvedName, and
// the caller sends no write in that case.
func resolveName[T any](ctx context.Context, g *Gateway, method string, query map[string]any, id func(T) int) (int, error) {
	rows, err := filterRows[T](ctx, g, method, query)
	if err != nil {
		return 0, err
	}
	if len(rows) != 1 {
		return 0, domain.ErrUnresolvedName
	}
	n := id(rows[0])
	if !validNumber(n) {
		return 0, domain.ErrUnexpectedResponse
	}
	return n, nil
}
