package sources

import (
	"context"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/sources/operations"
)

// Read runs one operation of the catalog against the selected source of the row's kind (ADR 007 section 3.1, ADR 009
// section 1). The order is fixed: the row and its kind are looked up, and an operation its kind does not support is
// refused; the raw arguments are bound and validated; the source is selected by the default rule for the kind; the
// source's access level is checked; and the reader is called last. A refused or invalid call makes no request to the wiki.
func Read(ctx context.Context, svc Service, op domain.OperationID, source *string, raw map[string]any) (domain.Record, error) {
	row, kind, err := rowOf(op)
	if err != nil {
		return domain.Record{}, err
	}
	args, err := operations.Bind(row, raw)
	if err != nil {
		return domain.Record{}, err
	}
	src, err := selectSource(svc, kind, source)
	if err != nil {
		return domain.Record{}, err
	}
	if err := operations.Refuse(row, src); err != nil {
		return domain.Record{}, err
	}
	reader, err := svc.Gateways.Reader(src)
	if err != nil {
		return domain.Record{}, err
	}
	reply, err := reader.Read(ctx, op, args)
	if err != nil {
		return domain.Record{}, err
	}
	if !matchesRow(row, reply) {
		return domain.Record{}, domain.ErrUnexpectedResponse
	}
	return withPage(row, args, reply), nil
}

// withPage returns a paged list with the continuation of its page, from the offset and the limit the call asked for and
// the number of rows the reply holds. A row without an offset and a limit, or a reply that is not a list, is returned as
// it is.
func withPage(row operations.Row, args domain.Args, reply domain.Record) domain.Record {
	if !reply.IsList() {
		return reply
	}
	_, hasOffset := row.Param("offset")
	_, hasLimit := row.Param("limit")
	if !hasOffset || !hasLimit {
		return reply
	}
	return reply.WithPage(domain.NewPage(args.Int("offset"), args.Int("limit"), len(reply.Items())))
}

// matchesRow reports whether a reply has the shape its row declares: a list for a list row, and an object that holds
// every declared field for any other row. A reader that returns a part of the wrong shape, or with a field missing,
// is a defect the caller must not see as a nil value.
func matchesRow(row operations.Row, reply domain.Record) bool {
	if row.IsList() != reply.IsList() {
		return false
	}
	if reply.IsList() {
		for _, item := range reply.Items() {
			if !holdsFields(item, row.Fields) {
				return false
			}
		}
		return true
	}
	return holdsFields(reply, row.Fields)
}

// holdsFields reports whether an object record holds every one of the named fields.
func holdsFields(rec domain.Record, names []string) bool {
	for _, name := range names {
		if _, ok := rec.Get(name); !ok {
			return false
		}
	}
	return true
}
