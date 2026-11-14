package sources

import (
	"context"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/sources/operations"
)

// Write runs one write operation of the catalog against the selected source of the row's kind. The order is fixed: the
// row and its kind are looked up, and an operation its kind does not support is refused; an operation of the read effect
// is refused, because it belongs to the read port; the raw arguments are bound and validated; a source is required and
// selected; the access level is checked; and then the writer is called once. A refused or invalid call makes no request
// to the wiki. A reply of the wrong shape after the writer was called returns ErrWriteOutcomeUnknown.
func Write(ctx context.Context, svc Service, op domain.OperationID, source *string, raw map[string]any) (domain.Record, error) {
	row, kind, err := rowOf(op)
	if err != nil {
		return domain.Record{}, err
	}
	if row.Effect == domain.EffectRead {
		return domain.Record{}, domain.ErrWrongOperationClass
	}
	args, err := operations.Bind(row, raw)
	if err != nil {
		return domain.Record{}, err
	}
	if source == nil {
		return domain.Record{}, domain.ErrSourceRequired
	}
	src, err := selectSource(svc, kind, source)
	if err != nil {
		return domain.Record{}, err
	}
	// The level is checked after the kind is selected, in the order the steps are listed: a source below the row's need
	// is refused with the level error before any call.
	if err := operations.Refuse(row, src); err != nil {
		return domain.Record{}, err
	}
	writer, err := svc.Gateways.Writer(src)
	if err != nil {
		return domain.Record{}, err
	}
	reply, err := writer.Write(ctx, op, args)
	if err != nil {
		return domain.Record{}, err
	}
	if !matchesRow(row, reply) {
		// The request was sent, so the change may have run, and its reply is unusable: the outcome is unknown, as a reply
		// that the adapter could not read is. The write is never retried.
		return domain.Record{}, domain.ErrWriteOutcomeUnknown
	}
	return reply, nil
}
