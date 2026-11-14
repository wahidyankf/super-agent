// Package engine runs the operations of a validated connector spec against one source. It renders each request from the
// spec, authenticates it with a key that the caller supplies through a port, sends it over HTTPS with the safeguards of
// ADR 001, and maps the reply to the declared fields. Reads and writes share those safeguards. A write is sent once, and
// a name that a write declares is resolved by one read first. It reads no environment variable, no file, and no process:
// the caller passes the key, the identity, the clock, and the dialer through Ports. The Markdown converters are not served
// yet, and an operation that needs one fails with a stable unsupported error before any request is sent.
package engine

import (
	"context"
	"crypto/x509"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/operations"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// The limits that a spec may only lower. A spec value above the default is ignored, and a zero value keeps the default.
const (
	// defaultTimeout is the time limit of one request.
	defaultTimeout = 30 * time.Second
	// defaultMaxReply is the size limit of one reply, in bytes: 10 MiB.
	defaultMaxReply = 10 << 20
	// maxPages is the most pages one read may take.
	maxPages = 20
	// maxRetries is the most times a rate-limited read is sent again after its first try.
	maxRetries = 3
	// maxRetryAfter is the longest wait, in seconds, that a Retry-After header may ask for.
	maxRetryAfter = 60
	// maxJSONDepth is the deepest nesting of arrays and objects that a reply may hold.
	maxJSONDepth = 64
	// maxJSONNodes is the most JSON tokens that a reply may hold.
	maxJSONNodes = 1 << 20
)

// Clock supplies the time limit of one request and the waits between two tries. A test passes a fake, so no unit test
// reads the real clock or starts a timer.
type Clock interface {
	// WithTimeout returns a context that ends after d, and the function that releases it.
	WithTimeout(parent context.Context, d time.Duration) (context.Context, context.CancelFunc)
	// Sleep waits for d, or until ctx ends. When ctx ends first, it returns the context's error.
	Sleep(ctx context.Context, d time.Duration) error
}

// Network is the two calls that reach a source: a lookup of a host name and a dial to one address. The composition root
// passes the real calls, and a test passes fakes. The engine checks every address before it dials it.
type Network struct {
	// Lookup returns the addresses of a host name.
	Lookup func(ctx context.Context, host string) ([]netip.Addr, error)
	// Dial connects to one address. The engine passes only an address that the check has accepted, as an IP literal.
	Dial func(ctx context.Context, network, address string) (net.Conn, error)
}

// Ports are the outbound ports of the engine. Credentials and Clock are always needed. Identities is needed for basic
// authentication. Either Transport or both Network calls are needed, and Roots is optional.
type Ports struct {
	// Credentials supplies the API key that the source names. The engine reads it for each request and never keeps it.
	Credentials ports.Credentials
	// Identities supplies the account address that basic authentication needs.
	Identities ports.Identities
	// Clock supplies the time limit of each request and the waits.
	Clock Clock
	// Network reaches the source when no Transport is given. The engine dials only the addresses it has checked.
	Network Network
	// Transport carries the requests in place of the guarded transport. A test passes the replay stub here.
	Transport http.RoundTripper
	// Roots is the pool of certificate authorities for the source, or nil for the system pool.
	Roots *x509.CertPool
}

// Option tunes an engine when it is built.
type Option func(*Engine)

// AllowPrivateNetwork lets the source resolve to a private, loopback, or link-local address. Only the catalog may set it,
// never a spec. An IP literal, localhost, an unspecified address, and a multicast address stay refused.
func AllowPrivateNetwork() Option {
	return func(e *Engine) { e.allowPrivate = true }
}

// Engine runs the reads of one spec against one source. It is safe for one caller at a time.
type Engine struct {
	model        *spec.Spec
	src          domain.Source
	ports        Ports
	client       *http.Client
	base         string
	ops          map[domain.OperationID]spec.Operation
	roles        map[string]spec.Operation
	allowPrivate bool
	timeout      time.Duration
	maxReply     int
}

var (
	_ ports.SourceReader  = (*Engine)(nil)
	_ ports.SourceGateway = (*Engine)(nil)
)

// New builds an engine for one source of a validated spec. It refuses a source of another kind, a host that the spec's
// host suffixes do not bind, a port that is missing, and a model that the spec rules would refuse.
func New(model *spec.Spec, src domain.Source, p Ports, opts ...Option) (*Engine, error) {
	if model == nil || p.Credentials == nil || p.Clock == nil {
		return nil, ErrNotConfigured
	}
	if model.Kind != src.Kind().String() {
		return nil, domain.ErrWrongSourceKind
	}
	if !bindsHost(model.Endpoint.HostSuffix, src.Endpoint().Host()) {
		return nil, domain.ErrInvalidEndpoint
	}
	if model.Auth.Type == "basic" && p.Identities == nil {
		return nil, ErrNotConfigured
	}
	if p.Transport == nil && (p.Network.Lookup == nil || p.Network.Dial == nil) {
		return nil, ErrNotConfigured
	}
	if err := checkSpecShape(model); err != nil {
		return nil, err
	}
	e := &Engine{
		model:    model,
		src:      src,
		ports:    p,
		base:     "https://" + src.Endpoint().Host(),
		ops:      make(map[domain.OperationID]spec.Operation, len(model.Operations)),
		roles:    make(map[string]spec.Operation),
		timeout:  capDuration(defaultTimeout, model.Limits.TimeoutSeconds),
		maxReply: capInt(defaultMaxReply, model.Limits.MaxReplyBytes),
	}
	for _, opt := range opts {
		opt(e)
	}
	rt := p.Transport
	if rt == nil {
		rt = guardedTransport(p.Network, e.allowPrivate, p.Roots)
	}
	e.client = &http.Client{Transport: rt, CheckRedirect: refuseRedirects}
	for _, o := range model.Operations {
		e.ops[domain.OperationID(o.ID)] = o
		if o.Role != "" && o.Access == "read" {
			e.roles[o.Role] = o
		}
	}
	return e, nil
}

// refuseRedirects stops the client at the first redirect, so the 3xx reply is read and refused, never followed.
func refuseRedirects(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}

// Read runs one read operation of the spec with its validated arguments. A write is refused, and a lookup or a Markdown
// converter fails before any request is sent. An argument the spec does not declare, or one of the wrong kind, fails the
// same way.
func (e *Engine) Read(ctx context.Context, op domain.OperationID, args domain.Args) (domain.Record, error) {
	o, ok := e.ops[op]
	if !ok {
		return domain.Record{}, domain.ErrUnsupportedOperation
	}
	if o.Access != "read" {
		return domain.Record{}, domain.ErrWrongOperationClass
	}
	bound, err := operations.Bind(rowOf(o), args.Values())
	if err != nil {
		return domain.Record{}, err
	}
	if err := unsupported(o); err != nil {
		return domain.Record{}, err
	}
	return e.run(ctx, o, bound)
}

// ListCollections returns the collections of the source, read by the spec's list-collections operation.
func (e *Engine) ListCollections(ctx context.Context) ([]domain.Collection, error) {
	rec, err := e.roleRead(ctx, "list-collections", "", "")
	if err != nil {
		return nil, err
	}
	var out []domain.Collection
	for _, item := range rec.Items() {
		out = append(out, domain.Collection{ID: item.Str("id"), Name: item.Str("name")})
	}
	return out, nil
}

// CollectionTree returns the documents of one collection, read by the spec's collection-tree operation. The depth and
// the parent are read from the fields depth and parent_id, when the spec declares them. The spec names its parameters and
// fields in snake case, so the role names the parameter collection_id.
func (e *Engine) CollectionTree(ctx context.Context, id domain.CollectionID) ([]domain.TreeNode, error) {
	rec, err := e.roleRead(ctx, "collection-tree", "collection_id", id.String())
	if err != nil {
		return nil, err
	}
	var out []domain.TreeNode
	for _, item := range rec.Items() {
		node := domain.TreeNode{ID: item.Str("id"), Title: item.Str("title"), URL: item.Str("url")}
		if d, ok := item.Get("depth"); ok {
			node.Depth, _ = d.(int)
		}
		if parent, ok := item.Get("parent_id"); ok {
			if p, isText := parent.(string); isText {
				node.ParentID = &p
			}
		}
		out = append(out, node)
	}
	return out, nil
}

// Document returns one document, read by the spec's get-document operation.
func (e *Engine) Document(ctx context.Context, id domain.DocumentID) (domain.SourceDocument, error) {
	rec, err := e.roleRead(ctx, "get-document", "document_id", id.String())
	if err != nil {
		return domain.SourceDocument{}, err
	}
	return domain.SourceDocument{
		ID:           rec.Str("id"),
		Title:        rec.Str("title"),
		URL:          rec.Str("url"),
		Markdown:     rec.Str("markdown"),
		CollectionID: rec.Str("collection_id"),
	}, nil
}

// roleRead runs the read operation that the spec gives a gateway role. The one identifier is passed under the parameter
// name the role needs, and a role with no parameter takes none.
func (e *Engine) roleRead(ctx context.Context, role, param, value string) (domain.Record, error) {
	o, ok := e.roles[role]
	if !ok {
		return domain.Record{}, domain.ErrUnsupportedOperation
	}
	var raw map[string]any
	if param != "" {
		raw = map[string]any{param: value}
	}
	return e.Read(ctx, domain.OperationID(o.ID), domain.NewArgs(raw))
}

// run sends the request of a bound read, follows its pages, and maps the reply. The output budget is shared by every
// page of one read, so a read cannot build more output than one reply may hold.
func (e *Engine) run(ctx context.Context, o spec.Operation, args domain.Args) (domain.Record, error) {
	budget := e.maxReply
	switch pagingStyle(o) {
	case "offset":
		return e.readOffset(ctx, o, args, &budget)
	case "cursor":
		return e.readCursor(ctx, o, args, &budget)
	}
	root, err := e.fetch(ctx, o, args, nil)
	if err != nil {
		return domain.Record{}, err
	}
	return mapReply(o, root, &budget)
}

// fetch renders the request of one page and sends it. A refused argument or header fails here, before any request.
func (e *Engine) fetch(ctx context.Context, o spec.Operation, args domain.Args, extra url.Values) (any, error) {
	req, err := renderRequest(e.model.Endpoint.BasePath, o, args, extra)
	if err != nil {
		return nil, err
	}
	return e.send(ctx, o, req)
}

// unsupported refuses a read that needs a part the engine does not serve: a Markdown converter, or a lookup, which only a
// write may declare. It returns nil for a read that the engine serves.
func unsupported(o spec.Operation) error {
	if err := converterUnsupported(o); err != nil {
		return err
	}
	for _, p := range o.Params {
		if p.Resolve != nil {
			return domain.ErrUnsupportedOperation
		}
	}
	return nil
}

// converterUnsupported refuses an operation whose reply needs a Markdown converter. The converters are not served yet, so
// the operation fails closed before any request is sent.
func converterUnsupported(o spec.Operation) error {
	for _, f := range o.Response.Fields {
		if f.Convert == "markdown-from-html" || f.Convert == "markdown-from-adf" {
			return domain.ErrUnsupportedOperation
		}
	}
	return nil
}

// mapReply maps a reply to a list or to one object, by whether the operation names an items pointer.
func mapReply(o spec.Operation, root any, budget *int) (domain.Record, error) {
	if o.Response.Items != "" {
		items, err := listItems(o, root, budget)
		if err != nil {
			return domain.Record{}, err
		}
		return domain.NewList(items), nil
	}
	obj, ok := root.(map[string]any)
	if !ok {
		return domain.Record{}, domain.ErrUnexpectedResponse
	}
	return mapRecord(o.Response.Fields, obj, budget)
}

// pagingStyle returns the paging style of an operation, and "none" when it has no paging.
func pagingStyle(o spec.Operation) string {
	if o.Pagination == nil {
		return "none"
	}
	return o.Pagination.Style
}

// bindsHost reports whether a host ends with one of the suffixes of the spec. A spec with no suffix binds every host.
func bindsHost(suffixes []string, host string) bool {
	if len(suffixes) == 0 {
		return true
	}
	for _, suffix := range suffixes {
		if strings.HasSuffix(host, suffix) {
			return true
		}
	}
	return false
}

// capDuration returns the spec's time limit in seconds when it is lower than the default, and the default otherwise.
func capDuration(def time.Duration, seconds int) time.Duration {
	if seconds > 0 && time.Duration(seconds)*time.Second < def {
		return time.Duration(seconds) * time.Second
	}
	return def
}

// capInt returns the spec's value when it is positive and lower than the default, and the default otherwise.
func capInt(def, value int) int {
	if value > 0 && value < def {
		return value
	}
	return def
}

// reservedHeaders are the header names that no spec may set, because the engine sets the key, the host, the length, and
// the cookies itself. The names compare without case.
var reservedHeaders = []string{"authorization", "cookie", "host", "content-length"}

// reservedHeader reports whether a header name is one that a spec may not set.
func reservedHeader(name string) bool {
	lower := strings.ToLower(name)
	for _, r := range reservedHeaders {
		if lower == r {
			return true
		}
	}
	return false
}

// checkSpecShape refuses a model that a validated spec would refuse. A model built by hand goes through the same rules,
// so it cannot send a path, a header, or a pointer that the spec does not allow. The finding names the field path only.
func checkSpecShape(model *spec.Spec) error {
	var fs []spec.Finding
	for i, o := range model.Operations {
		base := "/operations/" + strconv.Itoa(i)
		if !pathOK(o.Path) {
			fs = append(fs, specFinding(base+"/path", "has a segment that is not a literal or one whole slot"))
		}
		for name := range o.Headers {
			if reservedHeader(name) {
				fs = append(fs, specFinding(base+"/headers", "names a header that the engine sets itself"))
			}
		}
		for _, pointer := range pointersOf(o) {
			if pointer != "" && !strings.HasPrefix(pointer, "/") {
				fs = append(fs, specFinding(base+"/response", "has a pointer that does not start with a slash"))
			}
		}
	}
	if len(fs) > 0 {
		return &spec.InvalidError{Findings: fs}
	}
	return nil
}

// pathOK reports whether every segment of a path is a literal or one whole slot, and no segment starts with a dot.
func pathOK(path string) bool {
	if !strings.HasPrefix(path, "/") {
		return false
	}
	for _, seg := range strings.Split(path[1:], "/") {
		if strings.HasPrefix(seg, ".") {
			return false
		}
		if strings.ContainsAny(seg, "{}") && !slotPattern.MatchString(seg) {
			return false
		}
	}
	return true
}

// pointersOf returns every JSON pointer that a read of the operation names: the items list, the fields, and the cursor.
func pointersOf(o spec.Operation) []string {
	out := []string{o.Response.Items}
	for _, f := range o.Response.Fields {
		out = append(out, f.Pointer)
	}
	if o.Pagination != nil {
		out = append(out, o.Pagination.NextPointer)
	}
	return out
}

// specFinding builds a finding of the spec's stable code, at a path, with a fixed rule.
func specFinding(path, rule string) spec.Finding {
	return spec.Finding{Code: spec.CodeSpecInvalid, Path: path, Rule: rule}
}

// sortedKeys returns the names of a field map in order, so every record is built in one order.
func sortedKeys(fields map[string]spec.Field) []string {
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
