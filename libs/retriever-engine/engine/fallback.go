package engine

import (
	"context"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/operations"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// runWithFallback runs a read with a declared fallback. Both requests run under one shared time limit, so the pair
// never takes longer than the limit of one request, and both run through the same engine, so the target keeps the
// host, the key, the redirect refusal, and the caps of the first request. The target is picked from the document,
// never from reply data, and at most one fallback request runs per call, because the target has no fallback of its
// own and this function never calls itself. A first reply the mapper refuses ends the read with that failure: a
// failed read is never retried through the fallback.
func (e *Engine) runWithFallback(ctx context.Context, o spec.Operation, args domain.Args, budget *int) (domain.Record, error) {
	fb := o.Fallback
	fctx, cancel := e.ports.Clock.WithTimeout(ctx, e.timeout)
	defer cancel()
	root, status, err := e.fetchFull(fctx, o, args, nil)
	if err != nil {
		if status != 0 && triggersOnStatus(fb, status) {
			return e.runFallbackTarget(fctx, o, args, domain.Record{}, false, budget)
		}
		return domain.Record{}, err
	}
	if triggersOnStatus(fb, status) {
		return e.runFallbackTarget(fctx, o, args, domain.Record{}, false, budget)
	}
	rec, err := mapReply(o, args, root, budget)
	if err != nil {
		return domain.Record{}, err
	}
	if fb.When.Empty != "" {
		v, _ := rec.Get(fb.When.Empty)
		if v == nil || v == "" {
			return e.runFallbackTarget(fctx, o, args, rec, true, budget)
		}
	}
	return rec, nil
}

// triggersOnStatus reports whether the trigger of the fallback lists the status of the first reply. A fallback with
// the empty trigger lists no status, so it never fires here.
func triggersOnStatus(fb *spec.Fallback, status int) bool {
	for _, s := range fb.When.Status {
		if s == status {
			return true
		}
	}
	return false
}

// runFallbackTarget runs the one fallback request of a read and fills the named output fields with its values. The
// target declares only parameters the first operation also declares, and it receives their values; when it needs a
// value the caller gave none, the read fails with the missing argument. The output keeps the shape and the other
// fields of the first record when the first reply mapped, and null fields otherwise. When the fallback fails, the
// read fails with the fallback's error.
func (e *Engine) runFallbackTarget(ctx context.Context, o spec.Operation, args domain.Args, first domain.Record, hasFirst bool, budget *int) (domain.Record, error) {
	target, ok := e.ops[domain.OperationID(o.Fallback.Operation)]
	if !ok {
		return domain.Record{}, domain.ErrUnsupportedOperation
	}
	if err := unsupported(target); err != nil {
		return domain.Record{}, err
	}
	vals := args.Values()
	targs := make(map[string]any, len(target.Params))
	for _, p := range target.Params {
		if v, given := vals[p.Name]; given {
			targs[p.Name] = v
		}
	}
	bound, err := operations.Bind(rowOf(target), targs)
	if err != nil {
		return domain.Record{}, err
	}
	root, _, err := e.fetchFull(ctx, target, bound, nil)
	if err != nil {
		return domain.Record{}, err
	}
	trec, err := mapReply(target, bound, root, budget)
	if err != nil {
		return domain.Record{}, err
	}
	out := first
	if !hasFirst {
		out = emptyRecordOf(o)
	}
	for _, name := range o.Fallback.Fill {
		v, _ := trec.Get(name)
		out, err = withComputedField(out, name, v, budget)
		if err != nil {
			return domain.Record{}, err
		}
	}
	return out, nil
}

// emptyRecordOf returns a record with one null field for each declared field of the operation, so a fallback that
// fires on a status carries the shape of the first read with the fill fields set and every other field null.
func emptyRecordOf(o spec.Operation) domain.Record {
	names := sortedKeys(o.Response.Fields)
	fields := make([]domain.Field, 0, len(names))
	for _, name := range names {
		fields = append(fields, domain.Field{Name: name})
	}
	return domain.NewRecord(fields...)
}
