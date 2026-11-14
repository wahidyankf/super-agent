package outline

import (
	"context"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/operations"
)

// Write runs one change against the wiki and returns the Record of its reply. A change is a write, a destructive
// operation, or an admin operation of the catalog. It checks the arguments again with the catalog's rules, so a value
// that reaches a request has passed them, then sends the operation's request once. The use case has checked the source
// already. When the endpoint reads the object before its change, the read values fill the arguments the caller did not set.
func (g *Gateway) Write(ctx context.Context, op domain.OperationID, args domain.Args) (domain.Record, error) {
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
	ep, ok := writeEndpoints[op]
	if !ok {
		// The destructive and admin changes of stage c have their own table, kept apart from the stage b writes.
		ep, ok = adminEndpoints[op]
	}
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
