package spec

import (
	"bytes"
	"encoding/json"
)

// Spec is one connector spec as validated. It holds names and rules, never a value: the key of a source is read from
// the environment by name, and no field of a spec carries it. A zero Limits field means the cap of the built-in
// connectors applies.
type Spec struct {
	// Version is the spec version, which is 1.
	Version int `json:"version"`
	// Kind is the catalog kind that the spec declares, a lower-case slug.
	Kind string `json:"kind"`
	// Summary is one line of help text.
	Summary string `json:"summary"`
	// Endpoint holds the fixed path prefix and the host suffixes. The host itself is never in a spec.
	Endpoint Endpoint `json:"endpoint"`
	// Auth says how the key travels to the source.
	Auth Auth `json:"auth"`
	// Limits lower the caps of the engine. Each may only lower a cap.
	Limits Limits `json:"limits,omitempty"`
	// Operations are the operations of the source, in declaration order.
	Operations []Operation `json:"operations"`
}

// Endpoint is the path prefix and the host suffixes of a spec.
type Endpoint struct {
	// BasePath is a fixed path prefix, such as /v1, placed before every operation path.
	BasePath string `json:"base_path,omitempty"`
	// HostSuffix lists the DNS suffixes, each with its leading dot, that a catalog host must end with to bind to the spec.
	HostSuffix []string `json:"host_suffix,omitempty"`
}

// Auth says how the key of a source travels. The engine sets the header itself, and the spec names only variables.
type Auth struct {
	// Type is bearer, basic, or header.
	Type string `json:"type"`
	// CredentialEnv is the name of the variable that holds the key.
	CredentialEnv string `json:"credential_env"`
	// IdentityEnv is the name of the variable that holds the account e-mail. Only basic auth names it.
	IdentityEnv string `json:"identity_env,omitempty"`
	// HeaderName is the header that carries the key. Only header auth names it.
	HeaderName string `json:"header_name,omitempty"`
}

// Limits lower the caps of the engine. A zero field means the cap applies.
type Limits struct {
	// TimeoutSeconds is the time limit of one request, at most 30.
	TimeoutSeconds int `json:"timeout_seconds,omitempty"`
	// MaxReplyBytes is the size limit of one reply, at most 10 MiB.
	MaxReplyBytes int `json:"max_reply_bytes,omitempty"`
	// MaxPages is the most pages one read may take, at most 20.
	MaxPages int `json:"max_pages,omitempty"`
}

// Operation is one operation of the source. Access is a label for the review and the MCP hints, never a ceiling.
type Operation struct {
	// ID is <kind>.<group>.<verb>, unique in the spec.
	ID string `json:"id"`
	// CatalogID names the operation row of the built-in catalog that a bundled operation serves, such as
	// documents.get. The catalog row stays the source of the CLI name, the MCP tool name, and the parameters, and
	// the spec supplies only the request and the reply rules. Only a bundled spec may name it.
	CatalogID string `json:"catalog_id,omitempty"`
	// Summary is one line for help and the MCP description.
	Summary string `json:"summary"`
	// Access is read, write, or admin.
	Access string `json:"access"`
	// Method is the HTTP method of the request.
	Method string `json:"method"`
	// ReadOverPost is the marker of a read that is sent as POST. Only a read over POST takes it, and it is true or absent.
	ReadOverPost bool `json:"read_over_post,omitempty"`
	// Path is the operation path under the base path. A {name} slot fills one whole segment.
	Path string `json:"path"`
	// Params are the typed parameters of the operation.
	Params []Param `json:"params,omitempty"`
	// Headers maps each header of the request to a constant, or to a whole {name} parameter slot.
	Headers map[string]string `json:"headers,omitempty"`
	// Body is the JSON object of the request body. A string that is exactly {{name}} takes the value of a body parameter.
	Body map[string]any `json:"body,omitempty"`
	// Response says how the reply maps to the output fields.
	Response Response `json:"response"`
	// Pagination says how a read walks the pages, and always names a stop rule.
	Pagination *Pagination `json:"pagination,omitempty"`
	// Fallback is the declared second read of the operation, run when the first reply triggers it. Only a read of
	// one object takes one, and the target is a read of the same spec with no fallback of its own.
	Fallback *Fallback `json:"fallback,omitempty"`
	// Role names one of the gateway reads that the operation serves, when it serves one.
	Role string `json:"role,omitempty"`
	// CLI says whether the command-line tool offers the operation, and under which name when it is not the default.
	CLI Surface `json:"cli"`
	// MCP says whether the MCP server offers the operation, and under which name when it is not the default.
	MCP Surface `json:"mcp"`
	// Examples are canned exchanges that the conformance suite replays. At least one is required.
	Examples []Example `json:"examples"`
}

// UnmarshalJSON decodes an operation. The cli and mcp surfaces are on unless the document turns them off.
func (o *Operation) UnmarshalJSON(data []byte) error {
	type plain Operation
	p := plain{CLI: Surface{Enabled: true}, MCP: Surface{Enabled: true}}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if err := dec.Decode(&p); err != nil {
		return err
	}
	*o = Operation(p)
	return nil
}

// Param is one typed parameter of an operation. Its type is a parameter kind of the operation catalog.
type Param struct {
	// Name is the parameter name, a lower-case identifier.
	Name string `json:"name"`
	// Type is a parameter kind of the operation catalog, such as ID or Limit.
	Type string `json:"type"`
	// In is path, query, or body.
	In string `json:"in"`
	// Required says whether the caller must give the value. A path parameter is always required.
	Required bool `json:"required"`
	// Default is the value that an optional parameter takes when the caller gives none.
	Default any `json:"default,omitempty"`
	// Min is the lowest value of an Int parameter, when the spec sets one.
	Min *int `json:"min,omitempty"`
	// Max is the highest value of an Int parameter, when the spec sets one.
	Max *int `json:"max,omitempty"`
	// Values is the closed list of an Enum or EnumList parameter.
	Values []string `json:"values,omitempty"`
	// Help is one line of help for the parameter.
	Help string `json:"help,omitempty"`
	// Resolve turns a name into an ID by a read operation of the same spec.
	Resolve *Resolve `json:"resolve,omitempty"`
}

// Resolve names the read operation and the output field that match a name to exactly one entry.
type Resolve struct {
	// Operation is the ID of a read operation of the same spec.
	Operation string `json:"operation"`
	// Field is the output field of that operation that the name must match.
	Field string `json:"field"`
}

// Fallback is a declared second read of one operation. When the first reply triggers it, the engine runs the target
// read once with the values of the first call and fills the named output fields with the target's values. At most
// one fallback request runs per call, and the target has no fallback of its own.
type Fallback struct {
	// Operation is the ID of a read operation of the same spec. It is picked in the document, never from reply data.
	Operation string `json:"operation"`
	// When is the one closed trigger of the fallback: a mapped output field that is empty, or a list of HTTP
	// statuses. The triggers empty and status are owner proposals that the owner can still veto at review.
	When FallbackWhen `json:"when"`
	// Fill names the output fields that take the target's values.
	Fill []string `json:"fill"`
}

// FallbackWhen is the one closed trigger of a fallback: Empty names a mapped output field that is empty, and Status
// lists the HTTP statuses that run the fallback. Exactly one of them is set.
type FallbackWhen struct {
	// Empty is the mapped output field whose empty value runs the fallback. Empty means missing, null, or empty text.
	Empty string `json:"empty,omitempty"`
	// Status lists the HTTP statuses of the first reply that run the fallback.
	Status []int `json:"status,omitempty"`
}

// Response says how the reply of an operation maps to output fields, or to the Markdown of a document. A merge
// response names merge instead of fields: each list of the merge names its own fields. A tree response names tree
// with items: the tree flattens the nested nodes of the list, computing the parent and the depth of each node.
type Response struct {
	// Items is the JSON pointer to the list of a list operation. It is empty for a whole reply.
	Items string `json:"items,omitempty"`
	// First is the JSON pointer to the list whose first item a first-item read maps. It is empty unless the read
	// maps item 0, and a response holds at most one of items, first, and merge.
	First string `json:"first,omitempty"`
	// Merge merges two to eight lists with constant labels. It is nil unless the read merges lists, and a response
	// holds at most one of items, first, and merge.
	Merge *Merge `json:"merge,omitempty"`
	// Tree flattens the nested nodes of the items list depth first. It is nil unless the read is a tree, and it
	// rides on items, so a response with a tree names items and neither first nor merge.
	Tree *Tree `json:"tree,omitempty"`
	// Fields maps each output field to the JSON pointer that reads it. A merge response names no top-level field:
	// each list of the merge names its own fields.
	Fields map[string]Field `json:"fields,omitempty"`
	// Markdown names the output field that holds the Markdown body of a document read.
	Markdown string `json:"markdown,omitempty"`
	// Truncated is the computed per-item flag of a cursor read with the page-end rule. It is nil unless the read
	// marks each item truncated when a next-page token is present and the reply is not last.
	Truncated *Truncated `json:"truncated,omitempty"`
	// Errors maps an HTTP status to the closed reason that the status means.
	Errors map[string]string `json:"errors,omitempty"`
	// UnavailableMessage is the exact body message that proves the method is missing: a 404 carrying exactly this
	// message fails as operation_unavailable instead of the mapped reason. Empty means no body rule, and the 404 body
	// is never read. The match is equality with this constant only.
	UnavailableMessage string `json:"unavailable_message,omitempty"`
}

// Merge merges two to eight lists of one reply into one list, in the declared list order and then in the reply
// order. Each list is tagged with its constant label under the label field, which beats a reply field of the same
// name. The merged output shares the output cap of any list.
type Merge struct {
	// LabelField is the output field that carries the constant label of each list.
	LabelField string `json:"label_field"`
	// Lists holds two to eight lists to merge, in the order their items join the output.
	Lists []MergeList `json:"lists"`
}

// Truncated is the computed per-item flag of a cursor read with the page-end rule: it names the reply pointer of the
// next-page token and the reply pointer of the last flag. Each item gains truncated, which is true exactly when the
// token is present and non-empty and last is false.
type Truncated struct {
	// Token is the JSON pointer of the next-page token in the reply. A missing token counts as empty.
	Token string `json:"token"`
	// Last is the JSON pointer of the last flag in the reply. A missing last counts as false.
	Last string `json:"last"`
}

// MergeList is one list of a merge: the pointer to the list in the reply, the constant label of its items, and the
// fields that each of its items maps.
type MergeList struct {
	// Items is the JSON pointer to the list in the reply.
	Items string `json:"items"`
	// Label is the constant label of the items, of lowercase letters and hyphens, at most 32 characters.
	Label string `json:"label"`
	// Fields maps each output field of the list to the JSON pointer that reads it.
	Fields map[string]Field `json:"fields"`
}

// treeDepthCap is the deepest node a tree read keeps when the spec leaves max_depth unset.
const treeDepthCap = 64

// Tree flattens the nested nodes of a list reply depth first, computing the parent and the depth of each node, and
// only these two fields. The parent of a node is the id of its parent node: null for a top node, or the value of
// the root parent parameter when the read names one. The depth counts from 0. A node deeper than the depth cap,
// and the nodes below it, is left out.
type Tree struct {
	// Children is the JSON pointer of the child nodes of one node, read relative to the node.
	Children string `json:"children"`
	// ParentField is the output field that carries the id of the parent node.
	ParentField string `json:"parent_field"`
	// DepthField is the output field that carries the depth of the node.
	DepthField string `json:"depth_field"`
	// RootParentParam names the read parameter whose value is the parent of the top nodes. Empty means null.
	RootParentParam string `json:"root_parent_param,omitempty"`
	// MaxDepth is the deepest node the read keeps. Zero means the default of 64, and a spec may only lower it.
	MaxDepth int `json:"max_depth,omitempty"`
}

// EffectiveMaxDepth returns the deepest node the tree keeps: max_depth, or the cap of 64 when the spec leaves it
// unset. A value above the cap is refused at load, so it never reaches the engine.
func (t *Tree) EffectiveMaxDepth() int {
	if t == nil || t.MaxDepth <= 0 {
		return treeDepthCap
	}
	return t.MaxDepth
}

// Field is one output field: either the JSON pointer that reads a scalar, with whether the reply must hold it, the
// closed format that checks it, the converter that cleans it, when a converter is set, and the fallback pointer that
// is read when the pointer finds nothing, so one field maps two reply shapes; one level of object with the
// sub-fields that each read their own pointer; or one open-key map with closed bool values, which keeps every key
// the reply holds, sorted, with each value checked as a flag. An object field takes no scalar keyword of its own,
// and a bool map takes no format, converter, or object of its own. A strings field
// reads a list of strings instead of one scalar.
type Field struct {
	// Pointer is the JSON pointer of the value in the reply.
	Pointer string `json:"pointer"`
	// FallbackPointer is the second JSON pointer of the value, read when the pointer finds nothing. Empty means the
	// field has one shape only.
	FallbackPointer string `json:"fallback_pointer,omitempty"`
	// Required says whether a missing, null, or empty value fails the read. A missing optional field maps to null.
	Required bool `json:"required,omitempty"`
	// Format is the closed name that checks the value: id, uuid, share-id, or strings. Empty means no check.
	Format string `json:"format,omitempty"`
	// Convert is plain, markdown-from-html, markdown-from-adf, or hostname. Empty means plain. Hostname keeps the
	// host of an address, so a reply never shows a secret path.
	Convert string `json:"convert,omitempty"`
	// Object holds the sub-fields of a nested record field, each a scalar field of its own pointer. A sub-field holds
	// no object of its own, so a record nests one level only.
	Object map[string]Field `json:"object,omitempty"`
	// BoolMap marks an open-key map with closed bool values: the reply object at the pointer is kept whole, every
	// key sorted, and each value must be a flag. Empty means the field is a scalar or a nested record.
	BoolMap bool `json:"bool_map,omitempty"`
}

// Pagination says how a read walks the pages. It always names a stop rule: the page cap, which is at most 20.
type Pagination struct {
	// Style is offset, cursor, page-all, or none.
	Style string `json:"style"`
	// In is where the paging parameters travel: query, or body. Empty means query.
	In string `json:"in,omitempty"`
	// MaxPages is the page cap of the read, between 1 and 20. It is the stop rule of every paged read.
	MaxPages int `json:"max_pages,omitempty"`
	// PageSize is the fixed page size of a page-all read, between 1 and 100. Only a page-all rule names it.
	PageSize int `json:"page_size,omitempty"`
	// LimitParam is the query parameter that sets the page size of an offset read.
	LimitParam string `json:"limit_param,omitempty"`
	// OffsetParam is the query parameter that sets the offset of an offset read.
	OffsetParam string `json:"offset_param,omitempty"`
	// CursorParam is the query parameter that carries the cursor of a cursor read.
	CursorParam string `json:"cursor_param,omitempty"`
	// NextPointer is the JSON pointer to the next cursor in the reply of a cursor read.
	NextPointer string `json:"next_pointer,omitempty"`
	// LastPointer is the JSON pointer to the reply flag that decides last: when the flag is true the read claims no
	// further page, beating the short-page rule. Empty means the short-page rule decides.
	LastPointer string `json:"last_pointer,omitempty"`
	// CutToLimit names the Limit or Int parameter whose value cuts the page: a page holding more items than that
	// value keeps the first items and claims more. Empty means no cut.
	CutToLimit string `json:"cut_to_limit,omitempty"`
}

// Surface says whether the command-line tool or the MCP server offers an operation. A document sets it to a boolean,
// or to a custom name, which turns the surface on under that name.
type Surface struct {
	// Enabled says whether the surface offers the operation.
	Enabled bool
	// Name is the custom name of the command or tool, or "" for the default name.
	Name string
}

// UnmarshalJSON decodes a boolean or a custom name. A name turns the surface on.
func (s *Surface) UnmarshalJSON(data []byte) error {
	var on bool
	if err := json.Unmarshal(data, &on); err == nil {
		*s = Surface{Enabled: on}
		return nil
	}
	var name string
	if err := json.Unmarshal(data, &name); err != nil {
		return err
	}
	*s = Surface{Enabled: true, Name: name}
	return nil
}

// Example is one canned exchange: the parameter values of a request, the reply, and the output fields that the reply
// must map to.
type Example struct {
	// Request holds the parameter values of the request.
	Request map[string]any `json:"request"`
	// Response is the canned reply.
	Response ExampleResponse `json:"response"`
	// Expect holds the output fields that the reply must map to.
	Expect map[string]any `json:"expect"`
}

// ExampleResponse is the canned reply of an example.
type ExampleResponse struct {
	// Status is the HTTP status, from 100 to 599.
	Status int `json:"status"`
	// Headers are the constant headers of the reply. A reserved header name is refused.
	Headers map[string]string `json:"headers,omitempty"`
	// Body is the JSON body of the reply.
	Body any `json:"body,omitempty"`
}
