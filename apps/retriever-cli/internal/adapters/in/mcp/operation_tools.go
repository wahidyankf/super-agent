package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/operations"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/sources"
)

// The MCP tools of the operation catalog are generated. A row with a tool name becomes one tool: its
// name, description, and input schema come from the row, its call goes through the one binder, and its answer is the
// Record rendered in snake case. A list row is cut by fitRows, and a row with a body is paged by cutPage, and the
// model reads the next page with the offset that the answer names.

// bodyParam is the MCP-only paging input of a row with a body. It is not a catalog parameter, so the binder never sees it.
const bodyParam = "offset"

// sourceSchemaText describes the source input of every generated read tool.
const sourceSchemaText = "the source ID; may be omitted when exactly one outline source exists"

// sourceWriteText describes the source input of every generated write tool, which must name its source.
const sourceWriteText = "the source ID; required, a change names its source"

// operationTools returns the MCP tools of the catalog, one per row that names a tool, in catalog order.
func operationTools() []operations.Row {
	var out []operations.Row
	for _, row := range operations.Rows() {
		if row.Tool != "" {
			out = append(out, row)
		}
	}
	return out
}

// toolFor returns the SDK tool of one row: its name, its description ending with the untrusted-data sentence, its
// input schema, and its hints.
func toolFor(row operations.Row) *sdk.Tool {
	desc := row.Description
	switch {
	case row.IsList():
		desc += " The list is cut at " + strconv.Itoa(pageCap) + " bytes; when truncated is true, part of it was left out."
	case row.Body != "":
		desc += " The text is cut at " + strconv.Itoa(pageCap) + " bytes; when truncated is true, pass next_offset as offset to read on."
	}
	return &sdk.Tool{
		Name:        row.Tool,
		Description: describe(desc),
		InputSchema: inputSchema(row),
		Annotations: rowAnnotations(row),
	}
}

// rowAnnotations derives the hints of a catalog row. A read is read-only and idempotent. A write is
// not read-only, and it is never destructive in stage b. An update repeats the same change, so it is idempotent, unless
// the row says that its repeat is a different change (NotIdempotent). A create is never idempotent.
func rowAnnotations(row operations.Row) *sdk.ToolAnnotations {
	world := row.OpensWorld
	switch row.Effect {
	case domain.EffectAdditive:
		return &sdk.ToolAnnotations{IdempotentHint: row.SameOnRepeat, DestructiveHint: ptr(false), OpenWorldHint: &world}
	case domain.EffectUpdate:
		return &sdk.ToolAnnotations{IdempotentHint: !row.NotIdempotent, DestructiveHint: ptr(false), OpenWorldHint: &world}
	}
	return annotationsOf(row.Effect, row.OpensWorld)
}

// isWrite reports whether a row changes the wiki, so its tool takes the write input and runs through the write use case.
func isWrite(row operations.Row) bool {
	return row.Effect != domain.EffectRead
}

// inputSchema returns the JSON schema of the tool input: one property per catalog parameter in snake case, the source,
// and, for a row with a body, the paging offset. The schema is closed, so a property it does not name is refused.
func inputSchema(row operations.Row) map[string]any {
	props := map[string]any{
		"source": map[string]any{"type": "string", "description": sourceSchemaText},
	}
	var required []string
	if isWrite(row) {
		// A change always names its source.
		props["source"] = map[string]any{"type": "string", "description": sourceWriteText}
		required = append(required, "source")
	}
	for _, p := range row.Params {
		props[snakeCase(p.Name)] = propertySchema(p)
		if p.Required {
			required = append(required, snakeCase(p.Name))
		}
	}
	if row.Body != "" {
		props[bodyParam] = map[string]any{
			"type":        "integer",
			"minimum":     0,
			"description": "the byte offset to start the text from; 0 when omitted, and next_offset of the last answer",
		}
	}
	schema := map[string]any{"type": "object", "properties": props, "additionalProperties": false}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

// propertySchema returns the JSON schema of one catalog parameter, from its kind.
func propertySchema(p operations.Param) map[string]any {
	switch p.Kind {
	case operations.KindLimit:
		return map[string]any{"type": "integer", "minimum": 1, "maximum": domain.MaxPageLimit, "description": p.Help}
	case operations.KindOffset:
		return map[string]any{"type": "integer", "minimum": 0, "description": p.Help}
	case operations.KindInt:
		return map[string]any{"type": "integer", "minimum": p.Min, "maximum": p.Max, "description": p.Help}
	case operations.KindBool:
		return map[string]any{"type": "boolean", "description": p.Help}
	case operations.KindScore:
		return map[string]any{"type": "number", "minimum": -1, "maximum": 1, "description": p.Help}
	case operations.KindIDList, operations.KindUUIDList, operations.KindEventList:
		return map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "maxItems": domain.MaxIDList, "description": p.Help}
	case operations.KindURLList:
		return map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "minItems": 1, "maxItems": domain.MaxURLList, "description": p.Help}
	case operations.KindEnum:
		return map[string]any{"type": "string", "enum": p.Values, "description": p.Help}
	case operations.KindEnumList:
		return map[string]any{"type": "array", "items": map[string]any{"type": "string", "enum": p.Values}, "description": p.Help}
	}
	return map[string]any{"type": "string", "description": p.Help}
}

// snakeCase turns a camelCase catalog name into the snake_case name of the MCP input, such as collectionId into
// collection_id.
func snakeCase(name string) string {
	var b strings.Builder
	for i, r := range name {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(r + ('a' - 'A'))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// resourceSet is the set of resources a server offers. A nil set offers every resource.
type resourceSet map[string]bool

// newResourceSet returns the set of the named resources. A nil list means that no filter was named, so it returns a nil
// set, which offers every resource.
func newResourceSet(resources []string) resourceSet {
	if resources == nil {
		return nil
	}
	set := resourceSet{}
	for _, r := range resources {
		set[r] = true
	}
	return set
}

// offers reports whether the set offers the resource.
func (s resourceSet) offers(resource string) bool {
	return s == nil || s[resource]
}

// registerOperations adds the generated tools whose resource the set offers. It returns the names it added.
func registerOperations(server *sdk.Server, svc sources.Service, log *slog.Logger, offers resourceSet) []string {
	c := caller{svc: svc, log: log}
	var names []string
	for _, row := range operationTools() {
		if !offers.offers(row.Resource()) {
			continue
		}
		server.AddTool(toolFor(row), operationHandler(c, row))
		names = append(names, row.Tool)
	}
	return names
}

// operationHandler returns the call handler of one row. It decodes the arguments, binds them through the catalog, runs
// the one use case, renders the answer, and maps a failure to the public text. Its outcome is one log line, as the
// typed tools write.
func operationHandler(c caller, row operations.Row) sdk.ToolHandler {
	return func(ctx context.Context, req *sdk.CallToolRequest) (result *sdk.CallToolResult, err error) {
		start := time.Now()
		out, source, class, public := runOperation(ctx, c.svc, row, req)
		if class == outcomeUnexpected {
			c.record(row.Tool, source, outcomeUnexpected, start)
			return toolError(errUnexpected), nil
		}
		if public != nil {
			c.record(row.Tool, source, class, start)
			return toolError(public), nil
		}
		c.record(row.Tool, source, outcomeOK, start)
		return toolResult(out), nil
	}
}

// runOperation runs one row for one call. It recovers a panic, so no panic value reaches the model, and it returns the
// answer, the resolved source ID, the outcome class, and the public error when the call failed.
func runOperation(ctx context.Context, svc sources.Service, row operations.Row, req *sdk.CallToolRequest) (
	out map[string]any, source, class string, public error) {
	defer func() {
		if r := recover(); r != nil {
			out, source, class, public = nil, "", outcomeUnexpected, errUnexpected
		}
	}()
	args, err := decodeArgs(req)
	if err != nil {
		return nil, "", "bad arguments", errors.New("arguments must be a JSON object")
	}
	if err := refuseUndeclared(row, args); err != nil {
		return nil, "", "bad arguments", err
	}
	sourceArg, err := sourceOf(args)
	if err != nil {
		return nil, "", "bad arguments", err
	}
	if isWrite(row) {
		return runWriteOperation(ctx, svc, row, args, sourceArg)
	}
	src, err := sources.Resolve(svc, domain.KindOutline, sourceArg)
	if err != nil {
		public, class := publicError(err)
		return nil, "", class, public
	}
	id := src.ID().String()
	offset, err := bodyOffset(row, args)
	if err != nil {
		return nil, id, "negative offset", err
	}
	rec, err := sources.Read(ctx, svc, row.ID, sourceArg, catalogArgs(row, args))
	if err != nil {
		public, class := publicError(err)
		return nil, id, class, public
	}
	answer, err := render(row, id, rec, offset)
	if err != nil {
		public, class := publicError(err)
		return nil, id, class, public
	}
	return answer, id, outcomeOK, nil
}

// runWriteOperation runs one write row for one call. The source is required, and the use case refuses a call without
// one before any request.
func runWriteOperation(ctx context.Context, svc sources.Service, row operations.Row, args map[string]any, sourceArg *string) (
	out map[string]any, source, class string, public error) {
	id := resolvedID(svc, sourceArg)
	rec, err := sources.Write(ctx, svc, row.ID, sourceArg, catalogArgs(row, args))
	if err != nil {
		public, class := publicError(err)
		return nil, id, class, public
	}
	return renderObject(id, rec), id, outcomeOK, nil
}

// resolvedID returns the ID of the source a call names, for the answer and the log line. It returns "" when the call
// names no source or names one the catalog does not hold, so the use case reports that refusal instead.
func resolvedID(svc sources.Service, sourceArg *string) string {
	src, err := sources.Resolve(svc, domain.KindOutline, sourceArg)
	if err != nil {
		return ""
	}
	return src.ID().String()
}

// errUnknownInput refuses a call that names an input the tool does not list. The text names no argument, so it is the
// same for every name the caller tried.
var errUnknownInput = errors.New("invalid arguments: use the input names the tool lists, in snake_case")

// refuseUndeclared refuses a call that names a key the tool does not list before anything is bound (finding M3). The
// tool lists its snake_case inputs, the source, and the offset of a row with a body. A
// camelCase catalog name, or a second spelling of a parameter, is not one of them.
func refuseUndeclared(row operations.Row, args map[string]any) error {
	allowed := map[string]bool{"source": true}
	if row.Body != "" {
		allowed[bodyParam] = true
	}
	for _, p := range row.Params {
		allowed[snakeCase(p.Name)] = true
	}
	for key := range args {
		if !allowed[key] {
			return errUnknownInput
		}
	}
	return nil
}

// decodeArgs decodes the JSON object of the tool arguments. No arguments at all is an empty object.
func decodeArgs(req *sdk.CallToolRequest) (map[string]any, error) {
	args := map[string]any{}
	if req == nil || req.Params == nil || len(req.Params.Arguments) == 0 {
		return args, nil
	}
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return nil, err
	}
	return args, nil
}

// sourceOf returns the source argument as a pointer, or nil when the caller omitted it. A source that is not text is
// refused: it is not the same as omitting it, because an omitted source selects the one default source.
func sourceOf(args map[string]any) (*string, error) {
	v, ok := args["source"]
	if !ok {
		return nil, nil
	}
	s, ok := v.(string)
	if !ok {
		return nil, errBadSource
	}
	return &s, nil
}

// catalogArgs returns the arguments of the row in catalog names. It walks the parameters of the row in catalog order, so
// each snake_case input maps to exactly one catalog name, whatever the order of the call's keys. The source and the
// MCP-only switches are not catalog parameters, so they are left out. refuseUndeclared has checked the keys already.
func catalogArgs(row operations.Row, args map[string]any) map[string]any {
	out := map[string]any{}
	for _, p := range row.Params {
		if v, ok := args[snakeCase(p.Name)]; ok {
			out[p.Name] = v
		}
	}
	return out
}

// bodyOffset returns the paging offset of a row with a body. A negative offset is refused before the first request.
func bodyOffset(row operations.Row, args map[string]any) (int, error) {
	if row.Body == "" {
		return 0, nil
	}
	v, ok := args[bodyParam]
	if !ok || v == nil {
		return 0, nil
	}
	f, ok := v.(float64)
	if !ok || f != float64(int(f)) {
		return 0, errBadOffset
	}
	if f < 0 {
		return 0, errNegativeOffset
	}
	return int(f), nil
}

// render turns the Record of a call into the answer of the tool, with snake_case names and the source it ran against.
// A list is cut to the rows that fit pageCap, with truncated set when rows were left out. A body is cut to one page.
func render(row operations.Row, source string, rec domain.Record, offset int) (map[string]any, error) {
	if row.IsList() {
		return renderList(row, source, rec)
	}
	if row.Body != "" {
		return renderBody(row, source, rec, offset)
	}
	return renderObject(source, rec), nil
}

// renderObject answers one record, which is neither a list nor a body. It is cut to the page cap when it is too large,
// with truncated set. Every row that is not a list or a body is an object, so the write and admin answers use it too.
func renderObject(source string, rec domain.Record) map[string]any {
	out := snakeRecord(rec)
	out["source"] = source
	out["truncated"] = false
	if fitsCap(out) {
		return out
	}
	out["truncated"] = true
	cut, _ := cutToCap(out, func(m map[string]any) any { return m })
	return cut
}

// renderList answers a list row. The rows that fit pageCap are kept, and truncated is set when rows were left out. A
// paged list also says whether more rows may follow (has_more) and the offset to ask for next (next_offset), so an
// answer never stops at a page with no way forward. A single row that is larger than the cap is returned cut, with its
// longest strings shortened, so the count is never 0 while rows exist.
func renderList(row operations.Row, source string, rec domain.Record) (map[string]any, error) {
	key := snakeCase(row.ListKey)
	page, paged := rec.Page()
	rows := make([]map[string]any, 0, len(rec.Items()))
	for _, item := range rec.Items() {
		rows = append(rows, snakeRecord(item))
	}
	answer := func(kept []map[string]any, truncated bool) map[string]any {
		out := map[string]any{"source": source, key: kept, "count": len(kept), "truncated": truncated}
		if paged {
			hasMore := page.HasMore || truncated
			out["has_more"] = hasMore
			if hasMore {
				out["next_offset"] = page.Offset + len(kept)
			}
		}
		return out
	}
	kept, truncated, err := fitRows(rows, func(kept []map[string]any, tr bool) any { return answer(kept, tr) })
	if err != nil {
		return nil, err
	}
	if len(kept) == 0 && len(rows) > 0 {
		cut, _ := cutToCap(rows[0], func(m map[string]any) any { return answer([]map[string]any{m}, true) })
		kept, truncated = []map[string]any{cut}, true
	}
	return answer(kept, truncated), nil
}

// renderBody answers a row with a body: its fields, the page of the body, and the paging facts. The body is named text
// in the answer, as the get_document tool has always named it.
func renderBody(row operations.Row, source string, rec domain.Record, offset int) (map[string]any, error) {
	full := rec.Str(row.Body)
	if offset > len(full) {
		return nil, errOffsetPastEnd
	}
	page, next, truncated := cutPage(full, offset)
	out := map[string]any{"source": source, "text": page, "truncated": truncated, "total_bytes": len(full)}
	for _, f := range rec.Fields() {
		if f.Name != row.Body {
			out[snakeCase(f.Name)] = f.Value
		}
	}
	if truncated {
		out["next_offset"] = next
	}
	return out, nil
}

// snakeRecord returns the fields of a Record as a map with snake_case names, with nested records converted too.
func snakeRecord(rec domain.Record) map[string]any {
	out := map[string]any{}
	for _, f := range rec.Fields() {
		if nested, ok := f.Value.(domain.Record); ok {
			out[snakeCase(f.Name)] = snakeRecord(nested)
			continue
		}
		out[snakeCase(f.Name)] = f.Value
	}
	return out
}

// toolResult wraps an answer as a successful tool result: the JSON text, and the same object as structured content.
func toolResult(out map[string]any) *sdk.CallToolResult {
	raw, err := json.Marshal(out)
	if err != nil {
		return toolError(errUnexpected)
	}
	return &sdk.CallToolResult{
		Content:           []sdk.Content{&sdk.TextContent{Text: string(raw)}},
		StructuredContent: out,
	}
}

// toolError wraps a public error as a tool error, whose text the model sees.
func toolError(err error) *sdk.CallToolResult {
	return &sdk.CallToolResult{
		IsError: true,
		Content: []sdk.Content{&sdk.TextContent{Text: fmt.Sprint(err)}},
	}
}
