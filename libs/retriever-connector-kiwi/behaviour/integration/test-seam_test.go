//go:build integration

package integration_test

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-connector-kiwi/kiwi"
	"github.com/wahidyankf/super-agent/libs/retriever-testkit/kiwistub"
)

// setSeamEnv sets the three seam variables for one binding. An empty value is an unset variable.
func setSeamEnv(t *testing.T, base, caFile, timeout string) {
	t.Helper()
	t.Setenv(kiwi.BaseURLEnv, base)
	t.Setenv(kiwi.CAFileEnv, caFile)
	t.Setenv(kiwi.TimeoutEnv, timeout)
}

// loadSeam reads the seam from the real process environment and the real file system.
func loadSeam() (kiwi.Seam, error) {
	return kiwi.TestSeamFromEnvWithFiles(os.Getenv, os.ReadFile)
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/test-seam.feature: A CA file that is missing names only the variable and not its path
func TestMissingCAFileNamesOnlyTheVariable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing-ca.pem")
	setSeamEnv(t, "https://127.0.0.1:9443", path, "")
	_, err := loadSeam()
	if err == nil {
		t.Fatal("a missing CA file was accepted")
	}
	if !strings.Contains(err.Error(), kiwi.CAFileEnv) {
		t.Errorf("the error %v does not name %s", err, kiwi.CAFileEnv)
	}
	if strings.Contains(err.Error(), path) {
		t.Errorf("the error %v shows the CA file path", err)
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/test-seam.feature: A CA file with a valid certificate gives a client that trusts only that certificate and refuses redirects
func TestValidCAFileGivesClientThatTrustsOnlyThatCertificate(t *testing.T) {
	srv := serveKiwi(t, newStub(sampleDataset()), nil)
	path := filepath.Join(t.TempDir(), "ca.pem")
	must(t, os.WriteFile(path, srv.CA, 0o600))
	setSeamEnv(t, srv.Base, path, "2s")

	seam, err := loadSeam()
	must(t, err)
	if !seam.Enabled || seam.BaseURL != srv.Base {
		t.Fatalf("the seam is enabled=%v with base %q, want the server base", seam.Enabled, seam.BaseURL)
	}
	resp, err := seam.Client.Get(seam.BaseURL + kiwistub.Endpoint)
	if err != nil {
		t.Fatalf("the seam client did not reach the server over TLS: %v", err)
	}
	_ = resp.Body.Close()
	if err := seam.Client.CheckRedirect(&http.Request{}, nil); !errors.Is(err, http.ErrUseLastResponse) {
		t.Errorf("the seam client follows redirects: %v", err)
	}
	if _, err := (&http.Client{}).Get(srv.Base + kiwistub.Endpoint); err == nil {
		t.Error("a client that trusts the system roots reached the server with the test certificate")
	}
}

// @scenario specs/libs/retriever-connector-kiwi/behaviours/test-seam.feature: A CA file that <problem> is refused without showing its path
func TestCAFileThatCannotBeUsedIsRefusedWithoutItsPath(t *testing.T) {
	rows := []struct {
		name    string
		prepare func(t *testing.T) string
	}{
		{"cannot be read", func(t *testing.T) string {
			// A directory is not a file, so reading it fails.
			return t.TempDir()
		}},
		{"holds no certificate", func(t *testing.T) string {
			path := filepath.Join(t.TempDir(), "ca.pem")
			must(t, os.WriteFile(path, []byte("not a certificate"), 0o600))
			return path
		}},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			path := row.prepare(t)
			setSeamEnv(t, "https://127.0.0.1:9443", path, "")
			_, err := loadSeam()
			if err == nil {
				t.Fatal("the CA file was accepted")
			}
			if strings.Contains(err.Error(), path) {
				t.Errorf("the error %v shows the CA file path", err)
			}
		})
	}
}
