package outline

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

// BaseURLEnv names the environment variable that overrides the Outline base URL for tests.
const BaseURLEnv = "RETRIEVER_OUTLINE_TEST_BASE_URL"

// CAFileEnv names the environment variable that gives the PEM CA file for an override base.
const CAFileEnv = "RETRIEVER_OUTLINE_TEST_CA_FILE"

// TimeoutEnv names the environment variable that bounds one request on the override base.
// It is a Go duration such as "500ms", and it needs BaseURLEnv.
const TimeoutEnv = "RETRIEVER_OUTLINE_TEST_TIMEOUT"

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

// Seam is the loopback test seam that TestSeamFromEnv reads. When Enabled is false, no
// seam variable was set, and the other fields are zero. Timeout is zero when
// TimeoutEnv is unset, which keeps the gateway's default request timeout.
type Seam struct {
	BaseURL string
	Client  *http.Client
	Timeout time.Duration
	Enabled bool
}

// TestSeamFromEnv reads the loopback test seam through getenv. It is off when no seam
// variable is set. When it is on, the base must be HTTPS to a loopback host, and the
// CA file is added to a fresh trust pool that only the returned client uses. TimeoutEnv,
// when set, must be a positive duration and needs the base. Any other setting is an
// error, never a silent fallback to a real host or to the default timeout.
//
// The composition root calls it with the real environment, and only for tests. The
// name keeps the seam visible: it is a test seam, not a production setting.
func TestSeamFromEnv(getenv func(string) string) (Seam, error) {
	return TestSeamFromEnvWithFiles(getenv, os.ReadFile)
}

// TestSeamFromEnvWithFiles is TestSeamFromEnv with the CA file read injected. A unit test passes the certificate bytes from
// memory, so the seam reads no disk. Every rule is the same.
func TestSeamFromEnvWithFiles(getenv func(string) string, readFile func(string) ([]byte, error)) (Seam, error) {
	return seamFromEnv(getenv, readFile)
}

// seamFromEnv is TestSeamFromEnv with the CA file read injected, so a unit test supplies the file's bytes or a fault.
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
