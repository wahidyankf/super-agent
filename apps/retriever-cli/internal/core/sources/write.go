package sources

import (
	"context"
	"fmt"
	"sort"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/sources/operations"
)

// Write runs one write operation of the catalog against the selected source of the row's kind (ADR 007 section 3.1, ADR 009
// section 1). The order is fixed: the row and its kind are looked up, and an operation its kind does not support is
// refused; an operation of the read effect is refused, because it belongs to the read port; the raw arguments are bound and
// validated; a source is required and selected; the access level is checked; and then a dry run returns the plan, or the
// writer is called last. A refused, invalid, or dry-run call makes no request to the wiki. A reply
// of the wrong shape after the writer was called returns ErrWriteOutcomeUnknown.
func Write(ctx context.Context, svc Service, op domain.OperationID, source *string, raw map[string]any, dryRun bool) (domain.Record, error) {
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
	// The level is checked after the kind is selected, as ADR 007 section 3.1 orders the steps: a source below the row's need
	// is refused with the level error before any call.
	if err := operations.Refuse(row, src); err != nil {
		return domain.Record{}, err
	}
	if dryRun {
		return dryRunPlan(row, src, args), nil
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

// dryRunPlan returns what the write would send, and that it changed nothing. Its arguments are the bound values in name
// order. A text argument is counted and never echoed, so a dry run shows no document body.
func dryRunPlan(row operations.Row, src domain.Source, args domain.Args, extra ...domain.Field) domain.Record {
	values := args.Values()
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	planned := make([]domain.Field, 0, len(names))
	for _, name := range names {
		planned = append(planned, domain.Field{Name: name, Value: planValue(row, name, values[name])})
	}
	fields := []domain.Field{
		{Name: "operation", Value: string(row.ID)},
		{Name: "source", Value: src.ID().String()},
		{Name: "dryRun", Value: true},
		{Name: "changed", Value: false},
	}
	fields = append(fields, extra...)
	fields = append(fields, domain.Field{Name: "arguments", Value: domain.NewRecord(planned...)})
	return domain.NewRecord(fields...)
}

// planValue returns the value a dry run shows for one argument: the byte count for a text, a fixed mark for an e-mail
// address, the host alone for a URL, which may carry
// a secret in its path, and the value itself for any other kind.
func planValue(row operations.Row, name string, value any) any {
	param, ok := row.Param(name)
	text, isText := value.(string)
	if !ok || !isText {
		return value
	}
	switch param.Kind {
	case operations.KindText:
		return fmt.Sprintf("<%d bytes of text>", len(text))
	case operations.KindEmail:
		return "<an e-mail address, not shown>"
	case operations.KindURL:
		return operations.HostOf(text)
	}
	return value
}
