//go:build integration

// The seam reads a CA file from disk, so these tests use t.TempDir and are integration tests (test boundaries).
package kiwi

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// seamEnv returns a getenv over the given variables, and "" for any other name.
func seamEnv(vars map[string]string) func(string) string {
	return func(name string) string { return vars[name] }
}

// TestSeamIsOffWhenNoVariableIsSet checks the default: no seam variable, no seam, and no error.
func TestSeamIsOffWhenNoVariableIsSet(t *testing.T) {
	seam, err := TestSeamFromEnv(seamEnv(nil))
	if err != nil || seam.Enabled {
		t.Fatalf("TestSeamFromEnv(empty) = (%+v, %v), want a disabled seam and no error", seam, err)
	}
}

// TestSeamRefusesEveryBadSetting checks that a bad seam setting stops the command, with a text that names a variable.
func TestSeamRefusesEveryBadSetting(t *testing.T) {
	dir := t.TempDir()
	notPEM := filepath.Join(dir, "ca.pem")
	if err := os.WriteFile(notPEM, []byte("not a certificate"), 0o600); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		vars map[string]string
		want error
	}{
		{"CA without base", map[string]string{CAFileEnv: notPEM}, errSeamNeedsBase},
		{"timeout without base", map[string]string{TimeoutEnv: "1s"}, errSeamTimeoutNeedsBase},
		{"base without CA", map[string]string{BaseURLEnv: "https://127.0.0.1:9"}, errSeamNeedsCAFile},
		{"plain HTTP", map[string]string{BaseURLEnv: "http://127.0.0.1:9", CAFileEnv: notPEM}, domain.ErrHTTPSRequired},
		{"not loopback", map[string]string{BaseURLEnv: "https://kiwi.example.test", CAFileEnv: notPEM}, errSeamNotLoopback},
		{"user information", map[string]string{BaseURLEnv: "https://u:p@127.0.0.1:9", CAFileEnv: notPEM}, errSeamUserInfo},
		{"bad timeout", map[string]string{BaseURLEnv: "https://127.0.0.1:9", CAFileEnv: notPEM, TimeoutEnv: "soon"}, errSeamTimeoutInvalid},
		{"no certificate", map[string]string{BaseURLEnv: "https://127.0.0.1:9", CAFileEnv: notPEM}, errSeamNoCertificate},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := TestSeamFromEnv(seamEnv(tc.vars))
			if !errors.Is(err, tc.want) {
				t.Fatalf("TestSeamFromEnv error = %v, want %v", err, tc.want)
			}
		})
	}
}

// TestSeamNamesItsOwnVariables checks that the Kiwi seam reads the RETRIEVER_KIWI_TEST_ names and never the Outline ones.
func TestSeamNamesItsOwnVariables(t *testing.T) {
	for _, name := range []string{BaseURLEnv, CAFileEnv, TimeoutEnv} {
		if !strings.HasPrefix(name, "RETRIEVER_KIWI_TEST_") {
			t.Errorf("seam variable %q does not carry the RETRIEVER_KIWI_TEST_ prefix", name)
		}
	}
}

// TestSeamMissingCAFileNamesOnlyTheVariable checks that a missing CA file reports the variable and not the path.
func TestSeamMissingCAFileNamesOnlyTheVariable(t *testing.T) {
	_, err := TestSeamFromEnv(seamEnv(map[string]string{
		BaseURLEnv: "https://127.0.0.1:9",
		CAFileEnv:  "/no/such/dir/ca.pem",
	}))
	if err == nil || !strings.Contains(err.Error(), CAFileEnv) || strings.Contains(err.Error(), "/no/such/dir") {
		t.Fatalf("error = %v, want the variable name and not the path", err)
	}
}
