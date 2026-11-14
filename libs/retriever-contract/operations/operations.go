// Package operations is the Retriever operation catalog. One Row describes one operation completely: its ID,
// the command and tool that expose it, its effect, its parameters, and the shape of its result. The CLI
// command tree, the MCP tools, the binder, and the use cases are all built from these rows, so an operation is added by
// one row, one endpoint entry in the Outline adapter, and its scenarios, and no handler is written by hand.
package operations

import (
	"strings"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// Kind is the type of one parameter. Each kind has one validator, so every inbound adapter
// checks a value the same way.
type Kind string

// The parameter kinds. A new kind needs a validator in Validate and a test in operations_test.go.
const (
	KindID             Kind = "ID"             // ^[A-Za-z0-9_-]+$
	KindCollectionID   Kind = "CollectionID"   // an ID, refused as an invalid collection ID
	KindDocumentID     Kind = "DocumentID"     // an ID, refused as an invalid document ID
	KindCollectionUUID Kind = "CollectionUUID" // a canonical UUID of a collection (the server z.uuid())
	KindQuery          Kind = "Query"          // a search text, 1 to 1000 characters
	KindLimit          Kind = "Limit"          // a page size, 1 to 100
	KindOffset         Kind = "Offset"         // a page offset, 0 or more
	KindInt            Kind = "Int"            // an integer in the range Min to Max
	KindBool           Kind = "Bool"           // true or false
	KindName           Kind = "Name"           // 1 to 255 characters, no control character
	KindText           Kind = "Text"           // at most 1 MiB
	KindEnum           Kind = "Enum"           // one of Values
	KindIDList         Kind = "IDList"         // at most 50 IDs
	KindUUID           Kind = "UUID"           // a canonical UUID (the server z.uuid())
	KindShareID        Kind = "ShareID"        // a share ID: a UUID or a lower-case slug
	KindUUIDList       Kind = "UUIDList"       // at most 50 canonical UUIDs
	KindEnumList       Kind = "EnumList"       // a list of values, each one of Values
	KindTime           Kind = "Time"           // RFC 3339 with a zone
	KindURL            Kind = "URL"            // HTTPS, a DNS name, no credentials
	KindIndex          Kind = "Index"          // a fractional index, 1 to 256 printable ASCII characters
	KindColor          Kind = "Color"          // a hex colour, #RGB or #RRGGBB
	KindIcon           Kind = "Icon"           // an icon: one emoji, a named icon, or an attachment UUID; a non-blank text
	KindScore          Kind = "Score"          // a number from -1 to 1, as a JSON number or its text
	KindReason         Kind = "Reason"         // at most 2000 characters, no control character; may be empty
	KindEmail          Kind = "Email"          // one e-mail address; the reply never shows it
	KindEventList      Kind = "EventList"      // at most 50 event names, each `*` or a dotted name such as documents.create
	KindURLList        Kind = "URLList"        // 1 to 10 HTTPS addresses, each at most 1024 characters (the redirect addresses)
	KindIssueKey       Kind = "IssueKey"       // a Jira issue key, such as EXAMPLE-12; refused as an invalid document ID
)

// Param is one parameter of an operation. Name is the catalog name in camelCase, such as collectionId. The CLI flag
// is the name in kebab case, and the MCP input is the name in snake case. The Outline adapter maps the name to its own
// wire field.
type Param struct {
	Name     string
	Kind     Kind
	Required bool
	// Default is the value of an optional parameter that the caller does not set. Nil means none.
	Default any
	// Min and Max bound an Int parameter. Limit and Offset carry their own bounds in their kind.
	Min, Max int
	// Values is the closed list of an Enum parameter.
	Values []string
	// Help is the one line the CLI shows for the flag and the MCP input describes.
	Help string
}

// Row is one operation of the catalog. It is the single source for every adapter:
//   - CLI is the command path, such as "outline documents get". A required parameter is its positional argument.
//   - Tool is the MCP tool name. An empty Tool means the operation has no MCP tool.
//   - ListKey, Noun, Indent, Columns, Body, and NoMatch shape the output. A row with a ListKey is a list. A row with a
//     Body prints that field alone in text mode, and the MCP tool pages it.
//   - Fields is the neutral field list of the result. The Outline adapter's projection must return exactly these
//     names, which a test checks.
type Row struct {
	ID          domain.OperationID
	CLI         string
	Tool        string
	Summary     string
	Description string
	Effect      domain.Effect
	OpensWorld  bool
	Params      []Param
	Fields      []string
	ListKey     string
	Noun        string
	Columns     []string
	Indent      string
	Body        string
	NoMatch     bool
	// Detail names a field whose text prints under each row of a list, indented, such as the Markdown of a comment.
	Detail string
	// Quiet makes an empty list print nothing, not even its count line, as the Jira comment list does.
	Quiet bool
	// Notice is the line a truncated list prints in place of the next-page hint, for a list that has no offset to go on
	// to, such as the Jira issue search. Its JSON document then holds truncated in place of the next offset.
	Notice string
	// OneOf names parameters of which exactly one must be set, such as the document or the collection of a
	// subscription.
	OneOf []string
	// AnyOf names parameters of which at least one must be set, such as the fields a Kiwi test case update changes. The
	// binder refuses a call with none of them with ErrMissingArgument, so an empty update fails too.
	AnyOf []string
	// NotIdempotent marks an update whose repeat is not the same change, such as an append. Its tool says so in its hints
	//. A create is never idempotent, and it needs no flag.
	NotIdempotent bool
	// SameOnRepeat marks a create whose repeat answers the same object, so its tool announces idempotent. It is set on shares.create, which returns the share of a document that is already shared.
	SameOnRepeat bool
}

// Rows returns a copy of the built-in catalog, in catalog order. A registered catalog reads its rows through Catalog.Rows.
func Rows() []Row {
	return BuiltinCatalog().Rows()
}

// RowsOf returns a copy of the built-in catalog rows of one source kind, in catalog order. The Outline adapter, the
// Outline command tree, and the Outline tests use it, so a Kiwi row never joins the Outline surface.
func RowsOf(kind domain.SourceKind) []Row {
	return BuiltinCatalog().RowsOf(kind)
}

// Lookup returns the row of id in the built-in catalog, and false when the catalog holds no such operation.
func Lookup(id domain.OperationID) (Row, bool) {
	return BuiltinCatalog().Lookup(id)
}

// Path returns the words of the CLI command path, such as ["outline", "documents", "get"].
func (r Row) Path() []string {
	return strings.Fields(r.CLI)
}

// Positional returns the name of the positional parameter, which is the first required one, or "" when none is
// required.
func (r Row) Positional() string {
	for _, p := range r.Params {
		if p.Required {
			return p.Name
		}
	}
	return ""
}

// Param returns the named parameter and whether the row declares it.
func (r Row) Param(name string) (Param, bool) {
	for _, p := range r.Params {
		if p.Name == name {
			return p, true
		}
	}
	return Param{}, false
}

// IsList reports whether the result of the row is a list of items.
func (r Row) IsList() bool {
	return r.ListKey != ""
}

// Kind returns the source kind that the row belongs to: the first word of its CLI path, read as a kind. It returns false
// when that word names no known kind. The convention ties a row to its kind, and a test checks every catalog row.
func (r Row) Kind() (domain.SourceKind, bool) {
	path := r.Path()
	if len(path) == 0 {
		return domain.SourceKind{}, false
	}
	kind, err := domain.NewSourceKind(path[0])
	return kind, err == nil
}

// Supports reports whether a source kind runs the row, under the rules of the built-in catalog. The Outline kind runs every
// Outline row, and the Kiwi kind runs the closed list of its six rows, and the Jira kind the closed list of its five read
// rows. A kind that is not listed runs none.
func Supports(kind domain.SourceKind, row Row) bool {
	return BuiltinCatalog().Supports(kind, row)
}
