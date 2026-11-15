// Package jiraseam reads the loopback test seam of a Jira source: the base URL, the CA file, and the
// request timeout that point the engine at the stub Jira server in tests. It moved here from
// libs/retriever-connector-jira when pull request 39 retired that module's gateway: the engine serves every
// Jira operation now, and it needs the same base, client, and timeout the gateway took. The environment variable
// names are unchanged, so the transport scenarios keep their text.
package jiraseam

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// BaseURLEnv names the environment variable that overrides the Jira connection base for tests.
const BaseURLEnv = "RETRIEVER_JIRA_TEST_BASE_URL"

// CAFileEnv names the environment variable that gives the PEM CA file for an override base.
const CAFileEnv = "RETRIEVER_JIRA_TEST_CA_FILE"

// TimeoutEnv names the environment variable that bounds one request on the override base. It is a Go duration such as
// "500ms", and it needs BaseURLEnv. These are Jira's own names: no other kind's stub ever receives Jira's credentials.
const TimeoutEnv = "RETRIEVER_JIRA_TEST_TIMEOUT"

// errInvalidBaseURL reports a connection base that does not parse, has no host, or holds user information.
var errInvalidBaseURL = errors.New("invalid Jira base URL")

// Seam errors. Their text names a variable, never a value.
var (
	errSeamNeedsCAFile      = errors.New(BaseURLEnv + " needs " + CAFileEnv)
	errSeamNeedsBase        = errors.New(CAFileEnv + " needs " + BaseURLEnv)
	errSeamTimeoutNeedsBase = errors.New(TimeoutEnv + " needs " + BaseURLEnv)
	errSeamTimeoutInvalid   = errors.New(TimeoutEnv + " must be a positive Go duration such as 500ms")
	errSeamNotLoopback      = errors.New(BaseURLEnv + " must name a loopback IP address")
	errSeamUserInfo         = errors.New(BaseURLEnv + " must not hold user information")
	errSeamBaseNotParsed    = errors.New(BaseURLEnv + " is not a valid URL")
	errSeamNoCertificate    = errors.New(CAFileEnv + " holds no PEM certificate")
)

// parseSeamTimeout reads the optional request timeout. An empty value is unset, and the caller then keeps the default timeout,
// reported as zero. Any other value must be a positive Go duration, so a bad setting is an error and never silently ignored.
func parseSeamTimeout(value string) (time.Duration, error) {
	if value == "" {
		return 0, nil
	}
	d, err := time.ParseDuration(value)
	if err != nil || d <= 0 {
		return 0, errSeamTimeoutInvalid
	}
	return d, nil
}

// Seam is the loopback test seam that TestSeamFromEnvWithFiles reads. When Enabled is false, no seam variable was set, and the other
// fields are zero. Timeout is zero when TimeoutEnv is unset, which keeps the gateway's default request timeout.
type Seam struct {
	BaseURL string
	Client  *http.Client
	Timeout time.Duration
	Enabled bool
}

// TestSeamFromEnvWithFiles reads the loopback test seam through getenv. The composition root passes os.ReadFile for the CA file,
// so the real read lives there and not in this package. A unit test passes the certificate bytes from
// memory, so the seam reads no disk. Every rule is the same.
func TestSeamFromEnvWithFiles(getenv func(string) string, readFile func(string) ([]byte, error)) (Seam, error) {
	return seamFromEnv(getenv, readFile)
}

// seamFromEnv is the reader behind TestSeamFromEnvWithFiles, with the CA file read injected, so a unit test supplies the file's bytes or a fault.
func seamFromEnv(getenv func(string) string, readFile func(string) ([]byte, error)) (Seam, error) {
	base := getenv(BaseURLEnv)
	caFile := getenv(CAFileEnv)
	timeoutText := getenv(TimeoutEnv)
	if base == "" {
		switch {
		case caFile != "":
			return Seam{}, errSeamNeedsBase
		case timeoutText != "":
			return Seam{}, errSeamTimeoutNeedsBase
		}
		return Seam{}, nil
	}

	u, parseErr := url.Parse(base)
	switch {
	case parseErr != nil:
		return Seam{}, errSeamBaseNotParsed
	case u.Scheme != "https":
		return Seam{}, fmt.Errorf("%s must be an https URL: %w", BaseURLEnv, kindError(domain.ErrHTTPSRequired))
	case u.User != nil:
		return Seam{}, errSeamUserInfo
	case !isLoopbackHost(u.Hostname()):
		return Seam{}, errSeamNotLoopback
	case caFile == "":
		return Seam{}, errSeamNeedsCAFile
	}
	timeout, err := parseSeamTimeout(timeoutText)
	if err != nil {
		return Seam{}, err
	}

	pemBytes, readErr := readFile(caFile)
	if readErr != nil {
		return Seam{}, fmt.Errorf("read %s: %w", CAFileEnv, readErr)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pemBytes) {
		return Seam{}, errSeamNoCertificate
	}
	transport := &http.Transport{
		Proxy:           nil,
		TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12},
	}
	client := &http.Client{Transport: transport, CheckRedirect: refuseRedirects}
	return Seam{BaseURL: base, Client: client, Timeout: timeout, Enabled: true}, nil
}

// isLoopbackHost reports whether host is a loopback IP address. A name is never loopback here, so the seam does not depend on
// how a name resolves on the machine.
func isLoopbackHost(host string) bool {
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
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

// refuseRedirects stops the client at the first redirect, so the 3xx reply is read and refused, never followed.
func refuseRedirects(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}

// kindError names Jira as the kind of a failure, so a presenter prints "Jira rejected the API key", never
// a text about another system. A context error is returned as it is, because the caller cancelled it, and an error that is
// already a KindError is not wrapped twice.
func kindError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	var named domain.KindError
	if errors.As(err, &named) {
		return err
	}
	return domain.KindError{Kind: domain.KindJira, Err: err}
}
