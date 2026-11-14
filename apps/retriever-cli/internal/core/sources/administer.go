package sources

import (
	"context"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/sources/operations"
)

// Administer runs one destructive or admin operation of the catalog against the selected source. The order is fixed: a
// read is refused; the arguments are bound; a source is required and selected; the access level is checked; and then the
// admin port is called once. A refused or invalid call makes no request to the wiki.
func Administer(ctx context.Context, svc Service, op domain.OperationID, source *string, raw map[string]any) (domain.Record, error) {
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
	if err := operations.Refuse(row, src); err != nil {
		return domain.Record{}, err
	}
	admin, err := svc.Gateways.Admin(src)
	if err != nil {
		return domain.Record{}, err
	}
	reply, err := admin.Administer(ctx, op, args)
	if err != nil {
		return domain.Record{}, err
	}
	if !matchesRow(row, reply) {
		// The wiki answered, so the change may have run, and its reply cannot be read: this is the
		// same outcome as a 2xx reply that is not a readable success, so the caller never retries it.
		return domain.Record{}, domain.ErrWriteOutcomeUnknown
	}
	return reply, nil
}
