package kiwi

import (
	"context"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/sources/operations"
)

// Write runs one of the three Kiwi write operations and returns the Record of its reply. The use case
// has checked the source before it calls a writer.
func (g *Gateway) Write(ctx context.Context, op domain.OperationID, args domain.Args) (domain.Record, error) {
	rec, err := g.write(ctx, op, args)
	if err != nil {
		return domain.Record{}, kindErr(err)
	}
	return rec, nil
}

// write binds the arguments with the catalog's rules and runs the operation. A name is turned into an ID before the one
// write is sent, so a name that matches nothing, or too much, sends no write at all.
func (g *Gateway) write(ctx context.Context, op domain.OperationID, args domain.Args) (domain.Record, error) {
	row, ok := operations.Lookup(op)
	if !ok {
		return domain.Record{}, domain.ErrUnsupportedOperation
	}
	if row.Effect == domain.EffectRead {
		return domain.Record{}, domain.ErrWrongOperationClass
	}
	bound, err := operations.Bind(row, args.Values())
	if err != nil {
		return domain.Record{}, err
	}
	switch op {
	case domain.OpKiwiTestCasesCreate:
		return g.createCase(ctx, bound)
	case domain.OpKiwiTestCasesUpdate:
		return g.updateCase(ctx, bound)
	case domain.OpKiwiPlansAddCase:
		return g.addCase(ctx, bound)
	}
	return domain.Record{}, domain.ErrUnsupportedOperation
}

// createCase resolves the category, priority, and optional status names inside the product, then creates the case. Its
// reply must name the new case, or the outcome is unknown.
func (g *Gateway) createCase(ctx context.Context, a domain.Args) (domain.Record, error) {
	product, err := parseProductID(a.String("product"))
	if err != nil {
		return domain.Record{}, err
	}
	category, err := resolveName[wireCategory](ctx, g, methodCategoryFilter,
		map[string]any{"product": product, "name": a.String("category")},
		func(r wireCategory) int { return r.ID })
	if err != nil {
		return domain.Record{}, err
	}
	priority, err := resolveName[wirePriority](ctx, g, methodPriorityFilter,
		map[string]any{"value": a.String("priority")},
		func(r wirePriority) int { return r.ID })
	if err != nil {
		return domain.Record{}, err
	}
	values := map[string]any{"summary": a.String("summary"), "category": category, "priority": priority}
	if a.Has("status") {
		status, err := resolveName[wireStatus](ctx, g, methodTestCaseStatusFilter,
			map[string]any{"name": a.String("status")},
			func(r wireStatus) int { return r.ID })
		if err != nil {
			return domain.Record{}, err
		}
		values["case_status"] = status
	}
	if a.Has("text") {
		values["text"] = a.String("text")
	}
	raw, err := g.call(ctx, methodTestCaseCreate, map[string]any{"values": values})
	if err != nil {
		return domain.Record{}, err
	}
	reply, err := decodeOne[wireWrite](raw)
	if err != nil || !validNumber(reply.ID) {
		return domain.Record{}, domain.ErrWriteOutcomeUnknown
	}
	return domain.NewRecord(
		field("id", caseID(reply.ID)),
		field("title", title(a.String("summary"))),
		field("url", g.caseURL(reply.ID)),
	), nil
}

// updateCase changes the named fields of one case. The case must exist, and a name must resolve to one ID, before the
// write is sent. The reply must name the same case, or the outcome is unknown.
func (g *Gateway) updateCase(ctx context.Context, a domain.Args) (domain.Record, error) {
	n, err := parseCaseID(a.String("documentId"))
	if err != nil {
		return domain.Record{}, err
	}
	existing, err := filterRows[wireCase](ctx, g, methodTestCaseFilter, map[string]any{"id": n})
	if err != nil {
		return domain.Record{}, err
	}
	if len(existing) == 0 {
		return domain.Record{}, domain.ErrDocumentNotFound
	}
	values := map[string]any{}
	summary := existing[0].Summary
	if a.Has("summary") {
		summary = a.String("summary")
		values["summary"] = summary
	}
	if a.Has("text") {
		values["text"] = a.String("text")
	}
	if a.Has("status") {
		status, err := resolveName[wireStatus](ctx, g, methodTestCaseStatusFilter,
			map[string]any{"name": a.String("status")},
			func(r wireStatus) int { return r.ID })
		if err != nil {
			return domain.Record{}, err
		}
		values["case_status"] = status
	}
	if a.Has("priority") {
		priority, err := resolveName[wirePriority](ctx, g, methodPriorityFilter,
			map[string]any{"value": a.String("priority")},
			func(r wirePriority) int { return r.ID })
		if err != nil {
			return domain.Record{}, err
		}
		values["priority"] = priority
	}
	raw, err := g.call(ctx, methodTestCaseUpdate, map[string]any{"case_id": n, "values": values})
	if err != nil {
		return domain.Record{}, err
	}
	reply, err := decodeOne[wireWrite](raw)
	if err != nil || reply.ID != n {
		return domain.Record{}, domain.ErrWriteOutcomeUnknown
	}
	return domain.NewRecord(
		field("id", caseID(n)),
		field("title", title(summary)),
		field("url", g.caseURL(n)),
	), nil
}

// addCase adds one case to one plan. Kiwi's reply is only checked for its presence, since the names of both ends are the
// request's own.
func (g *Gateway) addCase(ctx context.Context, a domain.Args) (domain.Record, error) {
	planNo, err := parsePlanID(a.String("planId"))
	if err != nil {
		return domain.Record{}, err
	}
	caseNo, err := parseCaseID(a.String("caseId"))
	if err != nil {
		return domain.Record{}, err
	}
	if _, err := g.call(ctx, methodTestPlanAddCase, map[string]any{"plan_id": planNo, "case_id": caseNo}); err != nil {
		return domain.Record{}, err
	}
	return domain.NewRecord(
		field("planId", planID(planNo)),
		field("caseId", caseID(caseNo)),
	), nil
}
