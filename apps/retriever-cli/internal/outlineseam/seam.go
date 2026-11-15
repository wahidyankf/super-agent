// Package outlineseam reads the loopback test seam of an Outline source: the base URL, the CA file, and the
// request timeout that point the engine at the stub Outline server in tests. It moved here from
// libs/retriever-connector-outline when pull request 17 retired that module's gateway: the engine serves every
// Outline operation now, and it needs the same base, client, and timeout the gateway took. The environment variable
// names are unchanged, so the transport scenarios keep their text.
package outlineseam

import (
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

// BaseURLEnv names the environment variable that overrides the Outline base URL for tests.
const BaseURLEnv = "RETRIEVER_OUTLINE_TEST_BASE_URL"

// CAFileEnv names the environment variable that gives the PEM CA file for an override base.
const CAFileEnv = "RETRIEVER_OUTLINE_TEST_CA_FILE"

// TimeoutEnv names the environment variable that bounds one request on the override base.
// It is a Go duration such as "500ms", and it needs BaseURLEnv.
const TimeoutEnv = "RETRIEVER_OUTLINE_TEST_TIMEOUT"

// errInvalidBaseURL reports a base URL that does not parse, has no host, or holds user information.
var errInvalidBaseURL = errors.New("invalid Outline base URL")

// Seam errors. Their text names a variable, never a value.
var (
	errSeamNeedsCAFile      = errors.New(BaseURLEnv + " needs " + CAFileEnv)
	errSeamNeedsBase        = errors.New(CAFileEnv + " needs " + BaseURLEnv)
	errSeamTimeoutNeedsBase = errors.New(TimeoutEnv + " needs " + BaseURLEnv)
	errSeamTimeoutInvalid   = errors.New(TimeoutEnv + " must be a positive Go duration such as 500ms")
	errSeamNotLoopback      = errors.New(BaseURLEnv + " must name a loopback host")
	errSeamUserInfo         = errors.New(BaseURLEnv + " must not hold user information")
	errSeamBaseNotParsed    = errors.New(BaseURLEnv + " is not a valid URL")
	errSeamNoCertificate    = errors.New(CAFileEnv + " holds no PEM certificate")
)

// parseSeamTimeout reads the optional request timeout. An empty value is unset, and the
// caller then keeps the default timeout, reported as zero. Any other value must be a
// positive Go duration, so a bad setting is an error and never silently ignored.
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

// Seam is the loopback test seam that TestSeamFromEnvWithFiles reads. When Enabled is false, no
// seam variable was set, and the other fields are zero. Timeout is zero when
// TimeoutEnv is unset, which keeps the engine's default request timeout.
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
		return Seam{}, fmt.Errorf("%s must be an https URL: %w", BaseURLEnv, domain.ErrHTTPSRequired)
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

// isLoopbackHost reports whether host is a loopback IP address. A name is never loopback here, so
// the seam does not depend on how a name resolves on the machine.
func isLoopbackHost(host string) bool {
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
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

// refuseRedirects stops the client at the first redirect, so the 3xx reply is read, not followed.
func refuseRedirects(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}
