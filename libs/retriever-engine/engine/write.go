package engine

import (
	"context"
	"net/http"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/operations"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// The engine serves the write port of the contract.
var _ ports.SourceWriter = (*Engine)(nil)

// Write runs one write or admin operation of the spec with its validated arguments. It binds the arguments with the
// catalog's rules, resolves each declared name to its ID with one read, and then sends the write once. A failure before
// the request leaves is a known failure. A failure after it is an unknown outcome, which is never retried, except that a
// 429 reply is waited out and sent again, as a read is. A reply that reads as a success but holds no mappable field is an
// unknown outcome too, as the Go connectors decide it.
func (e *Engine) Write(ctx context.Context, op domain.OperationID, args domain.Args) (domain.Record, error) {
	o, ok := e.ops[op]
	if !ok {
		return domain.Record{}, domain.ErrUnsupportedOperation
	}
	if o.Access == "read" {
		return domain.Record{}, domain.ErrWrongOperationClass
	}
	if err := unsupportedWrite(o); err != nil {
		return domain.Record{}, err
	}
	raw := args.Values()
	bound, err := operations.Bind(rowOf(o), raw)
	if err != nil {
		return domain.Record{}, err
	}
	vals, err := e.resolveNames(ctx, o, raw, bound.Values())
	if err != nil {
		return domain.Record{}, err
	}
	req, err := renderRequest(e.model.Endpoint.BasePath, o, domain.NewArgs(vals), nil)
	if err != nil {
		return domain.Record{}, err
	}
	root, err := e.send(ctx, o, req)
	if err != nil {
		return domain.Record{}, err
	}
	budget := e.maxReply
	rec, err := mapReply(o, root, &budget)
	if err != nil {
		return domain.Record{}, domain.ErrWriteOutcomeUnknown
	}
	return rec, nil
}

// unsupportedWrite refuses a write that the engine does not serve. A write takes one request, so a paging rule on it would
// do nothing and is refused. A reply that needs a Markdown converter is refused too.
func unsupportedWrite(o spec.Operation) error {
	if pagingStyle(o) != "none" {
		return domain.ErrUnsupportedOperation
	}
	return converterUnsupported(o)
}

// resolveNames replaces the name of each parameter that declares a lookup with the ID that the lookup finds for it. A
// parameter that the caller left out has no name to resolve. The lookups run in parameter order, and the first failure
// stops them, so no write is sent after a failed lookup.
func (e *Engine) resolveNames(ctx context.Context, o spec.Operation, raw, vals map[string]any) (map[string]any, error) {
	out := make(map[string]any, len(vals))
	for name, v := range vals {
		out[name] = v
	}
	for _, p := range o.Params {
		if p.Resolve == nil {
			continue
		}
		name, given := vals[p.Name]
		if !given {
			continue
		}
		id, err := e.lookup(ctx, p.Resolve, raw, textOf(name))
		if err != nil {
			return nil, err
		}
		out[p.Name] = id
	}
	return out, nil
}

// lookup finds the ID of the one entry that a read names by a name. The read runs once through the engine, with the
// caller's arguments that the read declares, and with the name under the parameter of the field's own name when the read
// declares one. An entry of the reply is a match when its field equals the name exactly. Only one match gives an ID, so
// zero matches and two matches both fail with the unresolved-name error, and the write is never sent.
func (e *Engine) lookup(ctx context.Context, r *spec.Resolve, raw map[string]any, name string) (any, error) {
	read, ok := e.ops[domain.OperationID(r.Operation)]
	if !ok || !lookupServable(read, r.Field) {
		return nil, domain.ErrUnsupportedOperation
	}
	args := make(map[string]any)
	for _, p := range read.Params {
		if v, set := raw[p.Name]; set {
			args[p.Name] = v
		}
	}
	if declares(read, r.Field) {
		args[r.Field] = name
	}
	rec, err := e.Read(ctx, domain.OperationID(read.ID), domain.NewArgs(args))
	if err != nil {
		return nil, err
	}
	var matches []domain.Record
	for _, item := range rec.Items() {
		if item.Str(r.Field) == name {
			matches = append(matches, item)
		}
	}
	if len(matches) != 1 {
		return nil, domain.ErrUnresolvedName
	}
	id, _ := matches[0].Get("id")
	if !usableID(id) {
		return nil, domain.ErrUnexpectedResponse
	}
	return id, nil
}

// lookupServable reports whether a read can resolve a name. The read is a read whose reply is a list, it names the field
// that the name must match and an ID field, and it is not paged. A paged read can hide a second match past its last page,
// so the engine never resolves a name over one.
func lookupServable(read spec.Operation, field string) bool {
	return read.Access == "read" && read.Response.Items != "" && pagingStyle(read) == "none" &&
		hasField(read, field) && hasField(read, "id")
}

// declares reports whether an operation declares a parameter of the name.
func declares(o spec.Operation, name string) bool {
	for _, p := range o.Params {
		if p.Name == name {
			return true
		}
	}
	return false
}

// hasField reports whether the reply of an operation maps an output field of the name.
func hasField(o spec.Operation, name string) bool {
	_, ok := o.Response.Fields[name]
	return ok
}

// usableID reports whether a value can serve as an ID in a write: a positive whole number, or a text that is not empty.
func usableID(v any) bool {
	switch x := v.(type) {
	case int:
		return x > 0
	case string:
		return x != ""
	}
	return false
}

// unusableWriteReply reports whether a reply to a write says nothing usable about the change: a status below 300, which is
// no final answer or a 2xx that the client cannot read, or a 5xx. A 3xx, a 4xx, and a 429 are answers that the source gave
// about the request, so each keeps its own error.
func unusableWriteReply(status int) bool {
	return status < http.StatusMultipleChoices || status >= http.StatusInternalServerError
}
