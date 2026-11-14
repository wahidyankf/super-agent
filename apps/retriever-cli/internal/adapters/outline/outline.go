// Package outline is the SourceGateway adapter for one Outline wiki, named by its source. It
// speaks the Outline API over HTTPS, one request at a time with a delay between
// requests. It never prints, returns, or stores the API key outside a request.
package outline

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/ports"
)

const (
	pageSize        = 25
	pageCap         = 20
	maxRetries      = 3
	maxRetryAfter   = 60 // seconds
	defaultMinDelay = 100 * time.Millisecond
	defaultTimeout  = 30 * time.Second
	maxBodyBytes    = 10 << 20

	// apiVersion is the Outline API version the client asks for on every request. Version 2 wraps the document in
	// data.document with its Markdown text, and a server that ignores the header answers the older unwrapped shape.
	apiVersion = "2"
	// endpointNotFoundMessage is the message of the router's 404 for a method the server does not offer.
	endpointNotFoundMessage = "Endpoint not found"
)

// errInvalidBaseURL reports a base URL that does not parse, has no host, or holds user information.
var errInvalidBaseURL = errors.New("invalid Outline base URL")

// Gateway implements ports.SourceGateway over the Outline API. It holds the
// credentials port, never the key itself, and reads the key on each request.
// host is the host of the source's endpoint. It is the Host header of every request
// and the TLS name the server certificate must match, so a key never goes to another host.
type Gateway struct {
	base       string
	host       string
	client     *http.Client
	creds      ports.Credentials
	credential domain.CredentialRef
	minDelay   time.Duration
	timeout    time.Duration
	now        func() time.Time
	sleep      func(context.Context, time.Duration) error

	// mu serializes calls, so one request is in flight at a time, and a Retry-After wait
	// holds it too. lastEnd is the end of the previous request.
	mu      sync.Mutex
	lastEnd time.Time
	hasLast bool
}

var (
	_ ports.SourceGateway = (*Gateway)(nil)
	_ ports.SourceReader  = (*Gateway)(nil)
	_ ports.SourceWriter  = (*Gateway)(nil)
)

// Option tunes a Gateway. Tests use the options to replace the clock and the waits.
type Option func(*Gateway)

// WithMinDelay sets the least time between the end of one request and the start of the next.
func WithMinDelay(d time.Duration) Option {
	return func(g *Gateway) { g.minDelay = d }
}

// WithTimeout sets the time limit of one request. It must be positive.
func WithTimeout(d time.Duration) Option {
	return func(g *Gateway) { g.timeout = d }
}

// WithClock sets the clock that stamps request ends for the pacing rule.
func WithClock(now func() time.Time) Option {
	return func(g *Gateway) { g.now = now }
}

// WithSleep sets the wait used for pacing and for Retry-After. The default honours the context.
func WithSleep(sleep func(context.Context, time.Duration) error) Option {
	return func(g *Gateway) { g.sleep = sleep }
}

// New returns a gateway for the endpoint of src. Its transport never reads proxy settings from the
// environment, so the key goes straight to the endpoint host of the source.
func New(src domain.Source, creds ports.Credentials, opts ...Option) *Gateway {
	return build(src, src.Endpoint().String(), &http.Client{Transport: productionTransport()}, creds, opts)
}

// productionTransport copies the default transport and turns its proxy lookup off. The
// copy keeps the default dial, TLS, and idle timeouts, and the shared default is unchanged.
// When another package has replaced the default with a value that is not an *http.Transport,
// the copy cannot be made, so a transport with the same timeouts and no proxy is built instead.
func productionTransport() *http.Transport {
	if def, ok := http.DefaultTransport.(*http.Transport); ok {
		tr := def.Clone()
		tr.Proxy = nil
		return tr
	}
	return &http.Transport{
		Proxy:                 nil,
		DialContext:           (&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2:     true,
		TLSHandshakeTimeout:   10 * time.Second,
		IdleConnTimeout:       90 * time.Second,
		ExpectContinueTimeout: time.Second,
	}
}

// NewWithBase returns a gateway for src that connects to connectBase, which must be HTTPS. The Host header and
// the TLS name still come from the endpoint of src, so the connection base is the only thing that changes. The
// client is copied, and its redirect policy is replaced so no redirect is followed.
func NewWithBase(src domain.Source, connectBase string, httpClient *http.Client, creds ports.Credentials, opts ...Option) (*Gateway, error) {
	if err := CheckBaseURL(connectBase); err != nil {
		return nil, err
	}
	return build(src, connectBase, httpClient, creds, opts), nil
}

// CheckBaseURL reports whether baseURL can be an Outline connection base: an HTTPS URL with a host and no user
// information. It returns ErrHTTPSRequired for any other scheme.
func CheckBaseURL(baseURL string) error {
	u, err := url.Parse(baseURL)
	switch {
	case err != nil:
		return errInvalidBaseURL
	case u.Scheme != "https":
		return domain.ErrHTTPSRequired
	case u.Host == "" || u.User != nil:
		return errInvalidBaseURL
	}
	return nil
}

// build assembles a gateway for src with the safety defaults, then applies the options. The base is checked by
// the caller. The TLS name of the transport is set to the endpoint host, so the certificate is checked against
// the source's own host even when the connection goes elsewhere.
func build(src domain.Source, base string, httpClient *http.Client, creds ports.Credentials, opts []Option) *Gateway {
	client := &http.Client{}
	if httpClient != nil {
		copied := *httpClient
		client = &copied
	}
	client.CheckRedirect = refuseRedirects
	host := src.Endpoint().Host()
	client.Transport = withTLSName(client.Transport, host)
	g := &Gateway{
		base:       strings.TrimRight(base, "/"),
		host:       host,
		client:     client,
		creds:      creds,
		credential: src.Credential(),
		minDelay:   defaultMinDelay,
		timeout:    defaultTimeout,
		now:        time.Now,
		sleep:      sleepContext,
	}
	for _, opt := range opts {
		opt(g)
	}
	return g
}

// withTLSName returns a copy of rt whose TLS server name is host. The copy is made so the caller's transport is
// unchanged. A nil rt means the default transport. A round tripper that is not an *http.Transport has no TLS
// settings of its own, so it is returned as it is.
func withTLSName(rt http.RoundTripper, host string) http.RoundTripper {
	if rt == nil {
		rt = http.DefaultTransport
	}
	tr, ok := rt.(*http.Transport)
	if !ok {
		return rt
	}
	clone := tr.Clone()
	if clone.TLSClientConfig == nil {
		clone.TLSClientConfig = &tls.Config{ServerName: host}
	} else {
		clone.TLSClientConfig = clone.TLSClientConfig.Clone()
		clone.TLSClientConfig.ServerName = host
	}
	return clone
}

// refuseRedirects stops the client at the first redirect, so the 3xx reply is read, not followed.
func refuseRedirects(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}

// ListCollections returns every collection. It runs the collections.list operation, which reads pages of 25 until a
// short page.
func (g *Gateway) ListCollections(ctx context.Context) ([]domain.Collection, error) {
	rec, err := g.Read(ctx, domain.OpCollectionsList, domain.NewArgs(nil))
	if err != nil {
		return nil, err
	}
	var out []domain.Collection
	for _, item := range rec.Items() {
		out = append(out, domain.Collection{ID: item.Str("id"), Name: item.Str("name")})
	}
	return out, nil
}

// CollectionTree returns the documents of one collection, flattened depth first. It runs collections.documents.
func (g *Gateway) CollectionTree(ctx context.Context, id domain.CollectionID) ([]domain.TreeNode, error) {
	rec, err := g.Read(ctx, domain.OpCollectionsDocuments, domain.NewArgs(map[string]any{"collectionId": id.String()}))
	if err != nil {
		return nil, err
	}
	var out []domain.TreeNode
	for _, item := range rec.Items() {
		node := domain.TreeNode{ID: item.Str("id"), Title: item.Str("title"), URL: item.Str("url"), Depth: depthOf(item)}
		if parent, ok := item.Get("parentId"); ok {
			if p, isText := parent.(string); isText {
				node.ParentID = &p
			}
		}
		out = append(out, node)
	}
	return out, nil
}

// depthOf returns the depth field of a tree record.
func depthOf(item domain.Record) int {
	v, _ := item.Get("depth")
	n, _ := v.(int)
	return n
}

// Document returns one document, with its title, URL, collection, and Markdown. It runs documents.get, which reads the
// metadata with documents.info and, only when the reply has no text, the body with documents.export.
func (g *Gateway) Document(ctx context.Context, id domain.DocumentID) (domain.SourceDocument, error) {
	rec, err := g.Read(ctx, domain.OpDocumentsGet, domain.NewArgs(map[string]any{"documentId": id.String()}))
	if err != nil {
		return domain.SourceDocument{}, err
	}
	return domain.SourceDocument{
		ID:           rec.Str("id"),
		Title:        rec.Str("title"),
		URL:          rec.Str("url"),
		Markdown:     rec.Str("markdown"),
		CollectionID: rec.Str("collectionId"),
	}, nil
}
