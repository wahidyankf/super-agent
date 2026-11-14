package jira

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
	"strings"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// jiraSeamFault is the fault a test makes the CA file read return.
var jiraSeamFault = errors.New("injected CA read fault")

// jiraSeamSettings returns an environment that reads the three seam variables from values.
func jiraSeamSettings(values map[string]string) func(string) string {
	return func(name string) string { return values[name] }
}

// jiraSeamNoRead is a CA file read that the test must never reach.
func jiraSeamNoRead(t *testing.T) func(string) ([]byte, error) {
	return func(string) ([]byte, error) {
		t.Error("the CA file was read when the settings should have been refused first")
		return nil, jiraSeamFault
	}
}

// jiraSeamPEM returns one self-signed certificate in PEM form. It names no host and touches no network.
func jiraSeamPEM(t *testing.T) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "jira.example.test"},
		NotBefore:    time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
		NotAfter:     time.Date(2100, time.January, 1, 0, 0, 0, 0, time.UTC),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("CreateCertificate: %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

// TestJiraSeamSettingsAreRefusedBeforeTheCAFileIsRead checks each bad combination of the seam variables. Each is refused
// with its own rule, and the CA file is never read.
func TestJiraSeamSettingsAreRefusedBeforeTheCAFileIsRead(t *testing.T) {
	tests := []struct {
		name   string
		values map[string]string
		want   error
	}{
		{name: "a CA file without a base is refused", values: map[string]string{CAFileEnv: "/ca.pem"}, want: errSeamNeedsBase},
		{name: "a timeout without a base is refused", values: map[string]string{TimeoutEnv: "1s"}, want: errSeamTimeoutNeedsBase},
		{name: "a base that does not parse is refused", values: map[string]string{BaseURLEnv: "https://[::1", CAFileEnv: "/ca.pem"}, want: errSeamBaseNotParsed},
		{name: "a plain HTTP base is refused", values: map[string]string{BaseURLEnv: "http://127.0.0.1:9443", CAFileEnv: "/ca.pem"}, want: domain.ErrHTTPSRequired},
		{name: "a base with user information is refused", values: map[string]string{BaseURLEnv: "https://user@127.0.0.1:9443", CAFileEnv: "/ca.pem"}, want: errSeamUserInfo},
		{name: "a base with a host name is refused", values: map[string]string{BaseURLEnv: "https://jira.example.test", CAFileEnv: "/ca.pem"}, want: errSeamNotLoopback},
		{name: "a base with no CA file is refused", values: map[string]string{BaseURLEnv: "https://127.0.0.1:9443"}, want: errSeamNeedsCAFile},
		{name: "a timeout that is not a duration is refused", values: map[string]string{BaseURLEnv: "https://127.0.0.1:9443", CAFileEnv: "/ca.pem", TimeoutEnv: "soon"}, want: errSeamTimeoutInvalid},
		{name: "a timeout of zero is refused", values: map[string]string{BaseURLEnv: "https://127.0.0.1:9443", CAFileEnv: "/ca.pem", TimeoutEnv: "0s"}, want: errSeamTimeoutInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := seamFromEnv(jiraSeamSettings(tt.values), jiraSeamNoRead(t))
			if !errors.Is(err, tt.want) {
				t.Fatalf("seamFromEnv error = %v, want %v", err, tt.want)
			}
		})
	}
}

// TestJiraSeamWithNoVariablesIsOff checks that an environment with no seam variable turns the seam off, and reads no file.
// It also checks the exported entry point with the same environment.
func TestJiraSeamWithNoVariablesIsOff(t *testing.T) {
	seam, err := seamFromEnv(jiraSeamSettings(nil), jiraSeamNoRead(t))
	if err != nil || seam.Enabled || seam.Client != nil {
		t.Fatalf("seamFromEnv = %+v, %v; want the seam off", seam, err)
	}
	seam, err = seamFromEnvNoRead(jiraSeamSettings(nil))
	if err != nil || seam.Enabled {
		t.Fatalf("TestSeamFromEnv = %+v, %v; want the seam off", seam, err)
	}
}

// TestJiraSeamReadsTheCAFileIntoAClientThatTrustsOnlyIt checks the accepted setting: the base, the timeout, and a client
// whose transport trusts the file's certificate, takes no proxy, and whose redirects are refused.
func TestJiraSeamReadsTheCAFileIntoAClientThatTrustsOnlyIt(t *testing.T) {
	pemBytes := jiraSeamPEM(t)
	values := map[string]string{
		BaseURLEnv: "https://127.0.0.1:9443",
		CAFileEnv:  "/ca.pem",
		TimeoutEnv: "500ms",
	}
	var readPath string
	seam, err := seamFromEnv(jiraSeamSettings(values), func(name string) ([]byte, error) {
		readPath = name
		return pemBytes, nil
	})
	if err != nil {
		t.Fatalf("seamFromEnv: %v", err)
	}
	if readPath != "/ca.pem" {
		t.Fatalf("the CA file read was %q, want /ca.pem", readPath)
	}
	if !seam.Enabled || seam.BaseURL != "https://127.0.0.1:9443" || seam.Timeout != 500*time.Millisecond {
		t.Fatalf("seam = %+v; want the base and the timeout", seam)
	}
	tr, ok := seam.Client.Transport.(*http.Transport)
	if !ok || tr.TLSClientConfig == nil || tr.TLSClientConfig.RootCAs == nil || tr.Proxy != nil {
		t.Fatalf("the seam client has no trust pool of its own, or uses a proxy")
	}
	if seam.Client.CheckRedirect == nil {
		t.Fatalf("the seam client follows redirects")
	}
}

// TestJiraSeamRefusesACAFileThatCannotBeReadOrHoldsNoCertificate checks the two failures of the CA file. The read fault is
// returned, and neither error names the file's path.
func TestJiraSeamRefusesACAFileThatCannotBeReadOrHoldsNoCertificate(t *testing.T) {
	values := map[string]string{BaseURLEnv: "https://127.0.0.1:9443", CAFileEnv: "/private/ca.pem"}
	tests := []struct {
		name string
		read func(string) ([]byte, error)
		want error
	}{
		{name: "a CA file that cannot be read is refused", read: func(string) ([]byte, error) { return nil, jiraSeamFault }, want: jiraSeamFault},
		{name: "a CA file with no certificate is refused", read: func(string) ([]byte, error) { return []byte("not a certificate"), nil }, want: errSeamNoCertificate},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := seamFromEnv(jiraSeamSettings(values), tt.read)
			if !errors.Is(err, tt.want) {
				t.Fatalf("seamFromEnv error = %v, want %v", err, tt.want)
			}
			if err != nil && strings.Contains(err.Error(), "/private/ca.pem") {
				t.Fatalf("seamFromEnv error %q names the CA file path", err)
			}
		})
	}
}
