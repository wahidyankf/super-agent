package outline

import (
	"context"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/sources/operations"
)

// Read runs one catalog operation against the wiki and returns its Record. It first checks
// the arguments again with the catalog's rules, so a value that reaches a request has passed them, then sends the
// operation's endpoint request. The use case has already checked the access level.
func (g *Gateway) Read(ctx context.Context, op domain.OperationID, args domain.Args) (domain.Record, error) {
	row, ok := operations.Lookup(op)
	if !ok {
		return domain.Record{}, domain.ErrUnsupportedOperation
	}
	if row.Effect != domain.EffectRead {
		return domain.Record{}, domain.ErrWrongOperationClass
	}
	// The request is built from the bound arguments, so a parameter the caller did not set takes its default.
	bound, err := operations.Bind(row, args.Values())
	if err != nil {
		return domain.Record{}, err
	}
	args = bound
	ep, ok := endpoints[op]
	if !ok {
		return domain.Record{}, domain.ErrUnsupportedOperation
	}
	if ep.pageAll {
		return g.readAllPages(ctx, op, ep, args)
	}
	rec, err := g.send(ctx, op, ep.method, ep.body(args), ep.reply)
	if err != nil {
		return domain.Record{}, err
	}
	if ep.after != nil {
		return ep.after(ctx, g, args, rec)
	}
	return rec, nil
}

// send makes one request of an endpoint and projects its reply.
func (g *Gateway) send(ctx context.Context, op domain.OperationID, method string, body map[string]any,
	reply func(any) (domain.Record, error)) (domain.Record, error) {
	var root any
	if err := g.call(ctx, op, method, body, &root); err != nil {
		return domain.Record{}, err
	}
	rec, err := reply(root)
	if err != nil && writes(op) {
		// The wiki answered 2xx, so the change may have run, and the reply could not be read or did not say success true (m1).
		return domain.Record{}, domain.ErrWriteOutcomeUnknown
	}
	return rec, err
}

// readAllPages reads every page of a list operation, pageSize items at a time, and stops at the first short page or at
// the page cap. Only collections.list reads this way; a caller that wants one page uses the page parameters of a row.
func (g *Gateway) readAllPages(ctx context.Context, op domain.OperationID, ep endpoint, args domain.Args) (domain.Record, error) {
	var items []domain.Record
	for page := 0; ; page++ {
		if page == pageCap {
			return domain.Record{}, domain.ErrPageCapReached
		}
		values := args.Values()
		values["offset"] = page * pageSize
		values["limit"] = pageSize
		rec, err := g.send(ctx, op, ep.method, ep.body(domain.NewArgs(values)), ep.reply)
		if err != nil {
			return domain.Record{}, err
		}
		items = append(items, rec.Items()...)
		if len(rec.Items()) < pageSize {
			return domain.NewList(items), nil
		}
	}
}

// exportText reads the Markdown of one document with documents.export. The vault export and the text fallback of
// documents.get use it.
func (g *Gateway) exportText(ctx context.Context, rawID string) (string, error) {
	args := domain.NewArgs(map[string]any{"documentId": rawID})
	rec, err := g.send(ctx, opExport, exportEndpoint.method, exportEndpoint.body(args), exportEndpoint.reply)
	if err != nil {
		return "", err
	}
	return rec.Str("markdown"), nil
}
