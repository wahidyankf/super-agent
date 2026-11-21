package unit_test

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-connector-kiwi/kiwi"
)

// envOf returns a getenv double over the variables given.
func envOf(vars map[string]string) func(string) string {
	return func(name string) string { return vars[name] }
}

// fileDouble counts the reads of the CA file and answers each with the bytes or the fault given.
type fileDouble struct {
	calls int
	data  []byte
	err   error
}

// read is the file reader of the seam.
func (f *fileDouble) read(string) ([]byte, error) {
	f.calls++
	return f.data, f.err
}

// namesSeamVariable reports whether a seam error names one of the Kiwi seam variables.
func namesSeamVariable(msg string) bool {
	return strings.Contains(msg, kiwi.BaseURLEnv) || strings.Contains(msg, kiwi.CAFileEnv) || strings.Contains(msg, kiwi.TimeoutEnv)
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/test-seam.feature: With no seam variable set the seam is off, reads no file, and returns no error
func TestUnsetSeamIsOffReadsNoFileAndReturnsNoError(t *testing.T) {
	files := &fileDouble{}
	seam, err := kiwi.TestSeamFromEnvWithFiles(envOf(nil), files.read)
	if err != nil || seam.Enabled {
		t.Errorf("the seam is enabled=%v with error %v, want off and no error", seam.Enabled, err)
	}
	if files.calls != 0 {
		t.Errorf("%d files were read, want none", files.calls)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/test-seam.feature: The Kiwi seam reads only its own variables and never the Outline ones
func TestKiwiSeamIgnoresOutlineVariables(t *testing.T) {
	vars := map[string]string{
		"RETRIEVER_OUTLINE_TEST_BASE_URL": "https://127.0.0.1:9443",
		"RETRIEVER_OUTLINE_TEST_CA_FILE":  "/ca.pem",
		"RETRIEVER_OUTLINE_TEST_TIMEOUT":  "1s",
	}
	seam, err := kiwi.TestSeamFromEnvWithFiles(envOf(vars), (&fileDouble{}).read)
	if err != nil || seam.Enabled {
		t.Errorf("the seam is enabled=%v with error %v, want off", seam.Enabled, err)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/test-seam.feature: A bad seam setting stops the command with a message that names its variable
func TestBadSeamSettingNamesItsVariableAndCAFileIsNotRead(t *testing.T) {
	const base, ca = "https://127.0.0.1:9443", "/ca.pem"
	rows := []struct {
		name string
		vars map[string]string
	}{
		{"a CA file without a base address", map[string]string{kiwi.CAFileEnv: ca}},
		{"a timeout without a base address", map[string]string{kiwi.TimeoutEnv: "1s"}},
		{"a base address that does not parse", map[string]string{kiwi.BaseURLEnv: "https://[::1", kiwi.CAFileEnv: ca}},
		{"a plain HTTP base address", map[string]string{kiwi.BaseURLEnv: "http://127.0.0.1:9443", kiwi.CAFileEnv: ca}},
		{"a base address with user information", map[string]string{kiwi.BaseURLEnv: "https://user@127.0.0.1:9443", kiwi.CAFileEnv: ca}},
		{"a base address that is a host name and not a loopback address", map[string]string{kiwi.BaseURLEnv: "https://kiwi.example.test", kiwi.CAFileEnv: ca}},
		{"a base address with no CA file", map[string]string{kiwi.BaseURLEnv: base}},
		{"a timeout that is not a duration", map[string]string{kiwi.BaseURLEnv: base, kiwi.CAFileEnv: ca, kiwi.TimeoutEnv: "soon"}},
		{"a timeout of zero", map[string]string{kiwi.BaseURLEnv: base, kiwi.CAFileEnv: ca, kiwi.TimeoutEnv: "0s"}},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			files := &fileDouble{data: []byte("unused")}
			_, err := kiwi.TestSeamFromEnvWithFiles(envOf(row.vars), files.read)
			if err == nil || !namesSeamVariable(err.Error()) {
				t.Errorf("the error %v does not name a seam variable", err)
			}
			if files.calls != 0 {
				t.Errorf("the CA file was read %d times", files.calls)
			}
		})
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/test-seam.feature: A CA file that is missing names only the variable and not its path
func TestMissingCAFileNamesOnlyTheVariable(t *testing.T) {
	files := &fileDouble{err: errors.New("open /private/ca.pem: no such file or directory")}
	vars := map[string]string{kiwi.BaseURLEnv: "https://127.0.0.1:9443", kiwi.CAFileEnv: "/private/ca.pem"}
	_, err := kiwi.TestSeamFromEnvWithFiles(envOf(vars), files.read)
	if err == nil {
		t.Fatal("a missing CA file was accepted")
	}
	mustContain(t, err.Error(), kiwi.CAFileEnv)
	mustNotContain(t, err.Error(), "/private/ca.pem")
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/test-seam.feature: A CA file with a valid certificate gives a client that trusts only that certificate and refuses redirects
func TestValidCAFileGivesClientThatTrustsOnlyThatCertificate(t *testing.T) {
	files := &fileDouble{data: caPEM(t)}
	vars := map[string]string{
		kiwi.BaseURLEnv: "https://127.0.0.1:9443",
		kiwi.CAFileEnv:  "/ca.pem",
		kiwi.TimeoutEnv: "2s",
	}
	seam, err := kiwi.TestSeamFromEnvWithFiles(envOf(vars), files.read)
	must(t, err)
	if !seam.Enabled || seam.BaseURL != "https://127.0.0.1:9443" || seam.Timeout != 2*time.Second {
		t.Errorf("the seam holds base %q and timeout %v, enabled=%v", seam.BaseURL, seam.Timeout, seam.Enabled)
	}
	transport, ok := seam.Client.Transport.(*http.Transport)
	if !ok || transport.TLSClientConfig == nil || transport.TLSClientConfig.RootCAs == nil {
		t.Errorf("the client does not trust a certificate pool of its own")
	}
	if err := seam.Client.CheckRedirect(&http.Request{}, nil); !errors.Is(err, http.ErrUseLastResponse) {
		t.Errorf("the client follows redirects: %v", err)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/test-seam.feature: A CA file that <problem> is refused without showing its path
func TestCAFileThatCannotBeUsedIsRefusedWithoutItsPath(t *testing.T) {
	rows := []struct {
		name  string
		files *fileDouble
	}{
		{"cannot be read", &fileDouble{err: errors.New("open /private/ca.pem: permission denied")}},
		{"holds no certificate", &fileDouble{data: []byte("not a certificate")}},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			vars := map[string]string{kiwi.BaseURLEnv: "https://127.0.0.1:9443", kiwi.CAFileEnv: "/private/ca.pem"}
			_, err := kiwi.TestSeamFromEnvWithFiles(envOf(vars), row.files.read)
			if err == nil {
				t.Fatal("the CA file was accepted")
			}
			mustNotContain(t, err.Error(), "/private/ca.pem")
		})
	}
}
