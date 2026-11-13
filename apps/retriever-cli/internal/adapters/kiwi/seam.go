package kiwi

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// BaseURLEnv names the environment variable that overrides the Kiwi base URL for tests. It is separate from the Outline
// seam, so one stub never receives another kind's key (ADR 009 section 5).
const BaseURLEnv = "RETRIEVER_KIWI_TEST_BASE_URL"

// CAFileEnv names the environment variable that gives the PEM CA file for an override base.
const CAFileEnv = "RETRIEVER_KIWI_TEST_CA_FILE"

// TimeoutEnv names the environment variable that bounds one request on the override base. It is a Go duration such as
// "500ms", and it needs BaseURLEnv.
const TimeoutEnv = "RETRIEVER_KIWI_TEST_TIMEOUT"

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
	errSeamCAUnreadable     = errors.New(CAFileEnv + " names a file that cannot be read")
)

// parseSeamTimeout reads the optional request timeout. An empty value is unset, and the caller then keeps the default
// timeout, reported as zero. Any other value must be a positive Go duration.
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

// Seam is the loopback test seam that TestSeamFromEnv reads. When Enabled is false, no seam variable was set, and the
// other fields are zero. Timeout is zero when TimeoutEnv is unset, which keeps the gateway's default request timeout.
type Seam struct {
	BaseURL string
	Client  *http.Client
	Timeout time.Duration
	Enabled bool
}

// TestSeamFromEnv reads the loopback test seam through getenv. It is off when no seam variable is set. When it is on,
// the base must be HTTPS to a loopback IP literal, and the CA file is added to a fresh trust pool that only the returned
// client uses. A bad setting is an error, never a silent fallback to a real host or to the default timeout.
//
// The composition root calls it with the real environment, and only for tests. The name keeps the seam visible: it is a
// test seam, not a production setting.
func TestSeamFromEnv(getenv func(string) string) (Seam, error) {
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

	pemBytes, readErr := os.ReadFile(caFile)
	if readErr != nil {
		// The file's path is not repeated: it is a local path, and the error names the variable only.
		return Seam{}, errSeamCAUnreadable
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

// isLoopbackHost reports whether host is a loopback IP address. A name is never loopback here, so the seam does not
// depend on how a name resolves on the machine.
func isLoopbackHost(host string) bool {
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
