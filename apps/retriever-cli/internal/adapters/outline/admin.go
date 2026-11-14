package outline

import (
	"context"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/sources/operations"
)

// Administer runs one destructive or admin operation against the wiki and returns the Record of its reply. It checks the arguments again with the catalog's rules, then sends the operation's request once.
// The use case has checked the source and its level already; the client checks the level
// again, so a source below admin never sends an admin call.
func (g *Gateway) Administer(ctx context.Context, op domain.OperationID, args domain.Args) (domain.Record, error) {
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
	ep, ok := adminEndpoints[op]
	if !ok {
		return domain.Record{}, domain.ErrUnsupportedOperation
	}
	if ep.current != nil {
		bound, err = ep.current(ctx, g, bound)
		if err != nil {
			return domain.Record{}, err
		}
	}
	return g.send(ctx, op, ep.method, ep.body(bound), ep.reply)
}
