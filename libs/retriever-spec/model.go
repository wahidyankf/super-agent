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
	// Summary is one line for help and the MCP description.
	Summary string `json:"summary"`
	// Access is read, write, or admin.
	Access string `json:"access"`
	// Method is the HTTP method of the request.
	Method string `json:"method"`
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

// Response says how the reply of an operation maps to output fields, or to the Markdown of a document.
type Response struct {
	// Items is the JSON pointer to the list of a list operation. It is empty for a whole reply.
	Items string `json:"items,omitempty"`
	// Fields maps each output field to the JSON pointer that reads it.
	Fields map[string]Field `json:"fields"`
	// Markdown names the output field that holds the Markdown body of a document read.
	Markdown string `json:"markdown,omitempty"`
	// Errors maps an HTTP status to the closed reason that the status means.
	Errors map[string]string `json:"errors,omitempty"`
}

// Field is one output field: the JSON pointer that reads it and the converter that cleans it, when a converter is set.
type Field struct {
	// Pointer is the JSON pointer of the value in the reply.
	Pointer string `json:"pointer"`
	// Convert is plain, markdown-from-html, or markdown-from-adf. Empty means plain.
	Convert string `json:"convert,omitempty"`
}

// Pagination says how a read walks the pages. It always names a stop rule: the page cap, which is at most 20.
type Pagination struct {
	// Style is offset, cursor, or none.
	Style string `json:"style"`
	// MaxPages is the page cap of the read, between 1 and 20. It is the stop rule of every paged read.
	MaxPages int `json:"max_pages,omitempty"`
	// LimitParam is the query parameter that sets the page size of an offset read.
	LimitParam string `json:"limit_param,omitempty"`
	// OffsetParam is the query parameter that sets the offset of an offset read.
	OffsetParam string `json:"offset_param,omitempty"`
	// CursorParam is the query parameter that carries the cursor of a cursor read.
	CursorParam string `json:"cursor_param,omitempty"`
	// NextPointer is the JSON pointer to the next cursor in the reply of a cursor read.
	NextPointer string `json:"next_pointer,omitempty"`
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
