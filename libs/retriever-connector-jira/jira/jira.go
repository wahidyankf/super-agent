package jira

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
)

// Limits of the client. They are constants: no catalog field or reply changes them.
const (
	defaultMinDelay = 100 * time.Millisecond
	defaultTimeout  = 30 * time.Second
	maxBodyBytes    = 10 << 20
	maxRetries      = 3
	maxRetryAfter   = 60 // seconds
)

// Secrets is what the gateway reads for each request: the API token and the account e-mail address that Atlassian Cloud
// needs for HTTP Basic authorization. The gateway declares the pair it needs; the credentials adapter
// provides both.
type Secrets interface {
	ports.Credentials
	ports.Identities
}

// Gateway reads one Jira Cloud site. It implements the generic gateway and the read port. It holds the secrets
// port, never the token or the e-mail address, and it reads both for each request (auth.go).
//
// base is the connection base, which is the endpoint in production and a loopback stub in tests. host is the endpoint host.
// It is the Host header of every request and the TLS name the server certificate must match, so a credential never goes to
// another host. browse is the endpoint origin, which the URLs of issues and projects are built from.
type Gateway struct {
	base       string
	host       string
	browse     string
	client     *http.Client
	secrets    Secrets
	credential domain.CredentialRef
	identity   domain.IdentityRef
	minDelay   time.Duration
	timeout    time.Duration
	now        func() time.Time
	sleep      func(context.Context, time.Duration) error

	// mu serializes requests, so one is in flight at a time. A Retry-After wait holds it too. lastEnd is the end of the
	// previous request, which the pacing rule measures from.
	mu      sync.Mutex
	lastEnd time.Time
	hasLast bool
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

// New returns a gateway for the endpoint of src. Its transport never reads proxy settings from the environment, and it
// keeps no cookies, so the token goes straight to the endpoint host of the source.
func New(src domain.Source, secrets Secrets, opts ...Option) *Gateway {
	return build(src, src.Endpoint().String(), nil, secrets, opts)
}

// NewWithBase returns a gateway for src that connects to connectBase, which must be HTTPS. The Host header and the TLS name
// still come from the endpoint of src, so the connection base is the only thing that changes. The caller's client is copied,
// and the copy has no cookie jar, no proxy, and no redirect. A base that is refused returns a KindError for Jira.
func NewWithBase(src domain.Source, connectBase string, httpClient *http.Client, secrets Secrets, opts ...Option) (*Gateway, error) {
	if err := CheckBaseURL(connectBase); err != nil {
		return nil, kindError(err)
	}
	return build(src, connectBase, httpClient, secrets, opts), nil
}

// CheckBaseURL reports whether baseURL can be a Jira connection base: an HTTPS URL with a host and no user information. Any
// other scheme returns domain.ErrHTTPSRequired.
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

// build assembles a gateway for src with the safety defaults, then applies the options. The base is checked by the caller.
// The copy of the caller's client has no jar and refuses every redirect, and its transport has no proxy and the endpoint
// host as the TLS name.
func build(src domain.Source, base string, httpClient *http.Client, secrets Secrets, opts []Option) *Gateway {
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
		browse:     src.Endpoint().String(),
		client:     client,
		secrets:    secrets,
		credential: src.Credential(),
		identity:   src.Identity(),
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

// hardenTransport returns a copy of rt with no proxy and with host as its TLS server name. A nil rt means a copy of the
// default transport. A round tripper that is not an *http.Transport has no proxy or TLS settings of its own, so it is
// returned as it is. The caller's transport is never changed.
func hardenTransport(rt http.RoundTripper, host string) http.RoundTripper {
	var tr *http.Transport
	switch base := rt.(type) {
	case nil:
		tr = defaultTransportCopy()
	case *http.Transport:
		tr = base.Clone()
	default:
		return rt
	}
	tr.Proxy = nil
	if tr.TLSClientConfig == nil {
		tr.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	} else {
		tr.TLSClientConfig = tr.TLSClientConfig.Clone()
	}
	tr.TLSClientConfig.ServerName = host
	return tr
}

// defaultTransportCopy copies the default transport. When another package has replaced the default with a value that is not
// an *http.Transport, a transport with the same timeouts is built instead.
func defaultTransportCopy() *http.Transport {
	if def, ok := http.DefaultTransport.(*http.Transport); ok {
		return def.Clone()
	}
	return &http.Transport{
		DialContext:         (&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2:   true,
		TLSHandshakeTimeout: 10 * time.Second,
		IdleConnTimeout:     90 * time.Second,
	}
}

// refuseRedirects stops the client at the first redirect, so the 3xx reply is read and refused, never followed.
func refuseRedirects(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}
