package outlineseam

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net/http"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// seamEnv returns a getenv that reads only the given variables.
func seamEnv(vars map[string]string) func(string) string {
	return func(name string) string {
		return vars[name]
	}
}

// failingRead refuses every CA file read, so a unit test reads no file. A setting that reaches the CA read gets the
// failure and never touches the disk.
func failingRead(string) ([]byte, error) {
	return nil, errNoCAFile
}

// errNoCAFile stands in for a missing CA file.
var errNoCAFile = errors.New("no CA file")

func TestSeamIsOffWhenNeitherVariableIsSet(t *testing.T) {
	seam, err := TestSeamFromEnvWithFiles(seamEnv(map[string]string{}), failingRead)
	if err != nil {
		t.Fatalf("TestSeamFromEnvWithFiles error = %v, want nil when unset", err)
	}
	if seam.Enabled || seam.BaseURL != "" || seam.Client != nil {
		t.Fatalf("seam = %+v, want off", seam)
	}
}

// Every case below is refused before the CA file is read, so no file is touched.
func TestSeamRefusesAPlainHTTPOverride(t *testing.T) {
	seam, err := TestSeamFromEnvWithFiles(seamEnv(map[string]string{
		BaseURLEnv: "http://127.0.0.1:8080",
		CAFileEnv:  "ca.pem",
	}), failingRead)
	if !errors.Is(err, domain.ErrHTTPSRequired) {
		t.Fatalf("error = %v, want ErrHTTPSRequired", err)
	}
	if seam.Enabled {
		t.Fatalf("seam accepted a plain HTTP override")
	}
}

func TestSeamRefusesAnyHostThatIsNotLoopback(t *testing.T) {
	for _, base := range []string{
		"https://docs.example.test",
		"https://example.com",
		"https://198.51.100.5:8443",
		"https://0.0.0.0:8443",
		"https://[2001:db8::1]:8443",
		"https://localhost.example.com",
		"https://",
	} {
		t.Run(base, func(t *testing.T) {
			seam, err := TestSeamFromEnvWithFiles(seamEnv(map[string]string{
				BaseURLEnv: base,
				CAFileEnv:  "ca.pem",
			}), failingRead)
			if err == nil {
				t.Fatalf("seam accepted %q, want an error", base)
			}
			if seam.Enabled {
				t.Fatalf("seam reported enabled for %q", base)
			}
		})
	}
}

// TestSeamRefusesTheLocalhostName checks that only an IP literal counts as loopback. The name localhost is refused
// with the loopback error, so the seam never depends on how a name resolves on the machine.
func TestSeamRefusesTheLocalhostName(t *testing.T) {
	for _, base := range []string{"https://localhost:8443", "https://LOCALHOST:8443"} {
		t.Run(base, func(t *testing.T) {
			seam, err := TestSeamFromEnvWithFiles(seamEnv(map[string]string{
				BaseURLEnv: base,
				CAFileEnv:  "ca.pem",
			}), failingRead)
			if !errors.Is(err, errSeamNotLoopback) || seam.Enabled {
				t.Fatalf("base %q: enabled=%v err=%v, want errSeamNotLoopback", base, seam.Enabled, err)
			}
		})
	}
}

func TestSeamRefusesABaseWithoutACAFile(t *testing.T) {
	seam, err := TestSeamFromEnvWithFiles(seamEnv(map[string]string{
		BaseURLEnv: "https://127.0.0.1:8443",
	}), failingRead)
	if err == nil || seam.Enabled {
		t.Fatalf("seam accepted a base with no CA file: enabled=%v err=%v", seam.Enabled, err)
	}
}

func TestSeamRefusesACAFileWithoutABase(t *testing.T) {
	seam, err := TestSeamFromEnvWithFiles(seamEnv(map[string]string{
		CAFileEnv: "ca.pem",
	}), failingRead)
	if err == nil || seam.Enabled {
		t.Fatalf("seam accepted a CA file with no base: enabled=%v err=%v", seam.Enabled, err)
	}
}

func TestSeamRefusesUserInfoInTheBase(t *testing.T) {
	seam, err := TestSeamFromEnvWithFiles(seamEnv(map[string]string{
		BaseURLEnv: "https://reader@127.0.0.1:8443",
		CAFileEnv:  "ca.pem",
	}), failingRead)
	if err == nil || seam.Enabled {
		t.Fatalf("seam accepted user information in the base: enabled=%v err=%v", seam.Enabled, err)
	}
}

// A timeout alone bounds nothing, so it is refused before any other check.
func TestSeamRefusesATimeoutWithoutABase(t *testing.T) {
	seam, err := TestSeamFromEnvWithFiles(seamEnv(map[string]string{
		TimeoutEnv: "500ms",
	}), failingRead)
	if !errors.Is(err, errSeamTimeoutNeedsBase) || seam.Enabled {
		t.Fatalf("seam with a timeout and no base: enabled=%v err=%v, want errSeamTimeoutNeedsBase", seam.Enabled, err)
	}
}

// Every bad timeout is refused before the CA file is read, so no file is touched.
func TestSeamRefusesATimeoutThatIsNotAPositiveDuration(t *testing.T) {
	for _, value := range []string{"soon", "500", "0s", "-1s", "1 second"} {
		t.Run(value, func(t *testing.T) {
			seam, err := TestSeamFromEnvWithFiles(seamEnv(map[string]string{
				BaseURLEnv: "https://127.0.0.1:8443",
				CAFileEnv:  "ca.pem",
				TimeoutEnv: value,
			}), failingRead)
			if !errors.Is(err, errSeamTimeoutInvalid) || seam.Enabled {
				t.Fatalf("timeout %q: enabled=%v err=%v, want errSeamTimeoutInvalid", value, seam.Enabled, err)
			}
		})
	}
}

// testCAPEM returns the PEM bytes of a self-signed certificate, generated in memory, so the success path reads no
// file either.
func testCAPEM(t *testing.T) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate a test key: %v", err)
	}
	der, err := x509.CreateCertificate(rand.Reader, &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "stub"},
	}, &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "stub"},
	}, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create a test certificate: %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

// TestSeamBuildsAClientForALoopbackBase checks the success path: a loopback HTTPS base with a CA file enables the
// seam with the base, the timeout, and a client.
func TestSeamBuildsAClientForALoopbackBase(t *testing.T) {
	ca := testCAPEM(t)
	seam, err := TestSeamFromEnvWithFiles(seamEnv(map[string]string{
		BaseURLEnv: "https://127.0.0.1:8443",
		CAFileEnv:  "ca.pem",
		TimeoutEnv: "500ms",
	}), func(string) ([]byte, error) { return ca, nil })
	if err != nil {
		t.Fatalf("TestSeamFromEnvWithFiles error = %v, want nil", err)
	}
	if !seam.Enabled || seam.BaseURL != "https://127.0.0.1:8443" {
		t.Fatalf("seam = %+v, want enabled with the loopback base", seam)
	}
	if seam.Timeout != 500*time.Millisecond {
		t.Fatalf("seam timeout = %v, want 500ms", seam.Timeout)
	}
	if seam.Client == nil {
		t.Fatal("seam has no client")
	}
}

// TestCheckBaseURLRefusesABaseThatIsNotAnHTTPSOrigin checks the setup guard on its own: an HTTPS URL with a host
// and no user information passes, and any other scheme, a missing host, or user information fails.
func TestCheckBaseURLRefusesABaseThatIsNotAnHTTPSOrigin(t *testing.T) {
	if err := CheckBaseURL("https://127.0.0.1:8443"); err != nil {
		t.Fatalf("CheckBaseURL(https) = %v, want nil", err)
	}
	for _, base := range []string{"http://127.0.0.1:8443", "://no-scheme", "https://", "https://reader@127.0.0.1:8443"} {
		t.Run(base, func(t *testing.T) {
			if err := CheckBaseURL(base); err == nil {
				t.Fatalf("CheckBaseURL(%q) = nil, want an error", base)
			}
		})
	}
	if err := CheckBaseURL("http://127.0.0.1:8443"); !errors.Is(err, domain.ErrHTTPSRequired) {
		t.Fatalf("CheckBaseURL(http) = %v, want ErrHTTPSRequired", err)
	}
}

// TestSeamRefusesABaseThatIsNotAURL checks the parse rule: a base that is not a URL is refused before any other
// check.
func TestSeamRefusesABaseThatIsNotAURL(t *testing.T) {
	seam, err := TestSeamFromEnvWithFiles(seamEnv(map[string]string{
		BaseURLEnv: "://no-scheme",
		CAFileEnv:  "ca.pem",
	}), failingRead)
	if !errors.Is(err, errSeamBaseNotParsed) || seam.Enabled {
		t.Fatalf("base %q: enabled=%v err=%v, want errSeamBaseNotParsed", "://no-scheme", seam.Enabled, err)
	}
}

// TestClientRefusesARedirect checks the redirect policy of the seam client: a redirect is returned as the reply,
// never followed, so the engine reads the 3xx and refuses it.
func TestClientRefusesARedirect(t *testing.T) {
	if err := refuseRedirects(nil, nil); !errors.Is(err, http.ErrUseLastResponse) {
		t.Fatalf("refuseRedirects = %v, want http.ErrUseLastResponse", err)
	}
}
