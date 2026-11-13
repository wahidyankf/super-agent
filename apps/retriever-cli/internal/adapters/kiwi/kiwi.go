// Package kiwi is the source adapter for one Kiwi TCMS server. It speaks JSON-RPC 2.0 over HTTPS to
// POST /json-rpc/, one request at a time with a delay between requests. It reads and, for three writes, changes test
// plans and test cases. It never prints, returns, or stores the API key outside a request, and it never retries a write
// after its outcome could be unknown.
package kiwi

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
	// endpointPath is the one path the adapter calls. A reply or an argument never changes it.
	endpointPath    = "/json-rpc/"
	maxRetries      = 3
	maxRetryAfter   = 60 // seconds
	defaultMinDelay = 100 * time.Millisecond
	defaultTimeout  = 30 * time.Second
	maxBodyBytes    = 10 << 20
	// planCap is the most plans one product tree reads. More plans are refused before any case is read.
	planCap = 200
)

// errInvalidBaseURL reports a base URL that does not parse, has no host, or holds user information.
var errInvalidBaseURL = errors.New("invalid Kiwi base URL")

// errMethodNotAllowed reports a method outside the allowlist, or one the source's access level does not permit. The
// adapter's own methods pass only allowed names, so a caller never sees it.
var errMethodNotAllowed = errors.New("kiwi method is not allowed")

// Gateway implements the Kiwi source over one source. It holds the credentials port, never the key itself, and reads the
// key for each request. host is the host of the source's endpoint: it is the Host header of every request and the TLS
// name the server certificate must match, so the key never goes to another host.
type Gateway struct {
	base       string
	host       string
	endpoint   string
	access     domain.Access
	client     *http.Client
	creds      ports.Credentials
	credential domain.CredentialRef
	minDelay   time.Duration
	timeout    time.Duration
	now        func() time.Time
	sleep      func(context.Context, time.Duration) error

	// mu serializes calls, so one request is in flight at a time, and a Retry-After wait holds it too. lastEnd is the end
	// of the previous request. nextID numbers the JSON-RPC requests of this gateway.
	mu      sync.Mutex
	lastEnd time.Time
	hasLast bool
	nextID  int64

	// lookups caches the small static tables that name categories, priorities, and statuses.
	lookupMu sync.Mutex
	lookups  *lookupTables
}

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

// New returns a gateway for the endpoint of src. Its transport never reads proxy settings from the environment, so the
// key goes straight to the endpoint host of the source.
func New(src domain.Source, creds ports.Credentials, opts ...Option) *Gateway {
	return build(src, src.Endpoint().String(), &http.Client{Transport: productionTransport()}, creds, opts)
}

// productionTransport copies the default transport and turns its proxy lookup off. The copy keeps the default dial, TLS,
// and idle timeouts, and the shared default is unchanged.
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

// NewWithBase returns a gateway for src that connects to connectBase, which must be HTTPS. The Host header and the TLS
// name still come from the endpoint of src, so the connection base is the only thing that changes. The client is copied,
// and its redirect policy is replaced so no redirect is followed.
func NewWithBase(src domain.Source, connectBase string, httpClient *http.Client, creds ports.Credentials, opts ...Option) (*Gateway, error) {
	if err := CheckBaseURL(connectBase); err != nil {
		return nil, err
	}
	return build(src, connectBase, httpClient, creds, opts), nil
}

// CheckBaseURL reports whether baseURL can be a Kiwi connection base: an HTTPS URL with a host and no user information.
// It returns ErrHTTPSRequired for any other scheme.
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

// build assembles a gateway for src with the safety defaults, then applies the options. The base is checked by the
// caller. The copy of the caller's client has no cookie jar and refuses every redirect, and its transport has no proxy
// and the endpoint host as the TLS name, so the certificate is checked against the source's own host even when the
// connection goes elsewhere.
func build(src domain.Source, base string, httpClient *http.Client, creds ports.Credentials, opts []Option) *Gateway {
	client := &http.Client{}
	if httpClient != nil {
		copied := *httpClient
		client = &copied
	}
	client.Jar = nil
	client.CheckRedirect = refuseRedirects
	host := src.Endpoint().Host()
	client.Transport = hardenTransport(client.Transport, host)
	g := &Gateway{
		base:       strings.TrimRight(base, "/"),
		host:       host,
		endpoint:   strings.TrimRight(src.Endpoint().String(), "/"),
		access:     src.Access(),
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

// hardenTransport returns a copy of rt with no proxy lookup and with host as its TLS server name. The copy is made so the
// caller's transport is unchanged. A nil rt means a copy of the default transport. A round tripper that is not an
// *http.Transport has no proxy or TLS settings of its own, so it is returned as it is.
func hardenTransport(rt http.RoundTripper, host string) http.RoundTripper {
	if rt == nil {
		rt = productionTransport()
	}
	tr, ok := rt.(*http.Transport)
	if !ok {
		return rt
	}
	clone := tr.Clone()
	clone.Proxy = nil
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
