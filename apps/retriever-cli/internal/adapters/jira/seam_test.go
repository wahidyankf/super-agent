package jira

import (
	"errors"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// env returns a getenv that reads from the map, so a test names the variables it sets and no others.
func env(vars map[string]string) func(string) string {
	return func(name string) string { return vars[name] }
}

// TestTheSeamIsOffWhenNoVariableIsSet proves a command with no seam variables uses the real endpoint, with no seam at all.
func TestTheSeamIsOffWhenNoVariableIsSet(t *testing.T) {
	seam, err := TestSeamFromEnv(env(nil))
	if err != nil || seam.Enabled || seam.BaseURL != "" || seam.Client != nil || seam.Timeout != 0 {
		t.Fatalf("seam %+v, err %v; want the zero seam", seam, err)
	}
}

// TestTheSeamVariablesNeedTheBase pins that a CA file or a timeout without a base is refused, never ignored.
func TestTheSeamVariablesNeedTheBase(t *testing.T) {
	if _, err := TestSeamFromEnv(env(map[string]string{CAFileEnv: "/nonexistent/ca.pem"})); !errors.Is(err, errSeamNeedsBase) {
		t.Fatalf("CA file without base: err = %v", err)
	}
	if _, err := TestSeamFromEnv(env(map[string]string{TimeoutEnv: "1s"})); !errors.Is(err, errSeamTimeoutNeedsBase) {
		t.Fatalf("timeout without base: err = %v", err)
	}
}

// TestTheSeamBaseMustBeAnHTTPSLoopbackAddressWithNoUserInfo pins the base rules, each with a text that names the variable.
func TestTheSeamBaseMustBeAnHTTPSLoopbackAddressWithNoUserInfo(t *testing.T) {
	cases := []struct {
		name string
		base string
		want error
	}{
		{"plain http", "http://127.0.0.1:8443", domain.ErrHTTPSRequired},
		{"a real host", "https://jira.example.test", errSeamNotLoopback},
		{"a name that resolves", "https://localhost:8443", errSeamNotLoopback},
		{"user information", "https://user:pw@127.0.0.1:8443", errSeamUserInfo},
		{"not a URL", "https://[::bad", errSeamBaseNotParsed},
	}
	for _, c := range cases {
		_, err := TestSeamFromEnv(env(map[string]string{BaseURLEnv: c.base, CAFileEnv: "/nonexistent/ca.pem"}))
		if !errors.Is(err, c.want) {
			t.Fatalf("%s: err = %v, want %v", c.name, err, c.want)
		}
	}
}

// TestTheSeamRefusesHTTPWithTheJiraText proves the HTTPS refusal names Jira, not Outline.
func TestTheSeamRefusesHTTPWithTheJiraText(t *testing.T) {
	_, err := TestSeamFromEnv(env(map[string]string{BaseURLEnv: "http://127.0.0.1:8443", CAFileEnv: "/nonexistent/ca.pem"}))
	if err == nil || strings.Contains(err.Error(), "Outline") || !strings.Contains(err.Error(), "HTTPS is required for Jira") {
		t.Fatalf("err = %v, want the Jira HTTPS text", err)
	}
	if !errors.Is(err, domain.ErrHTTPSRequired) {
		t.Fatal("the refusal does not match ErrHTTPSRequired")
	}
}

// TestTheSeamNeedsACAFile proves a base with no CA file is refused, and the text names the variable.
func TestTheSeamNeedsACAFile(t *testing.T) {
	_, err := TestSeamFromEnv(env(map[string]string{BaseURLEnv: "https://127.0.0.1:8443"}))
	if !errors.Is(err, errSeamNeedsCAFile) || !strings.Contains(err.Error(), CAFileEnv) {
		t.Fatalf("err = %v, want the CA file requirement", err)
	}
}

// TestTheSeamTimeoutMustBeAPositiveDuration pins the timeout grammar: a positive Go duration, and nothing else.
func TestTheSeamTimeoutMustBeAPositiveDuration(t *testing.T) {
	for _, value := range []string{"soon", "0s", "-1s", "5"} {
		_, err := TestSeamFromEnv(env(map[string]string{
			BaseURLEnv: "https://127.0.0.1:8443", CAFileEnv: "/nonexistent/ca.pem", TimeoutEnv: value,
		}))
		if !errors.Is(err, errSeamTimeoutInvalid) && err == nil {
			t.Fatalf("timeout %q: err = %v, want a refusal", value, err)
		}
	}
}

// TestTheSeamNamesItsVariablesOnly proves every seam text names a variable and no value, so a bad setting never echoes it.
func TestTheSeamNamesItsVariablesOnly(t *testing.T) {
	for _, s := range []string{BaseURLEnv, CAFileEnv, TimeoutEnv} {
		if !strings.HasPrefix(s, "RETRIEVER_JIRA_TEST_") {
			t.Fatalf("seam variable %q does not carry the Jira test prefix", s)
		}
	}
	_, err := TestSeamFromEnv(env(map[string]string{BaseURLEnv: "https://reader@127.0.0.1:8443", CAFileEnv: "/x"}))
	if err == nil || strings.Contains(err.Error(), "reader@") {
		t.Fatalf("err = %v, want a refusal that never shows the value", err)
	}
}

// TestTheSeamVariablesAreSeparateFromTheOtherKinds proves Jira's seam names are not Outline's or Kiwi's, so one stub never
// receives another kind's credentials.
func TestTheSeamVariablesAreSeparateFromTheOtherKinds(t *testing.T) {
	for _, name := range []string{BaseURLEnv, CAFileEnv, TimeoutEnv} {
		if strings.Contains(name, "OUTLINE") || strings.Contains(name, "KIWI") {
			t.Fatalf("seam variable %q is shared with another kind", name)
		}
	}
}
