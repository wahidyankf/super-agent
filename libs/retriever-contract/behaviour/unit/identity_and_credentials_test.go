package unit_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// formattedText is the text that one standard formatting path writes for a secret value.
type formattedText struct {
	path string
	text string
}

// checkRedacted fails the test when a formatted text holds the secret, or when a text does not show the redacted text.
func checkRedacted(t *testing.T, secret string, texts []formattedText) {
	t.Helper()
	for _, ft := range texts {
		if strings.Contains(ft.text, secret) {
			t.Errorf("%s holds the secret value", ft.path)
		}
		if !strings.Contains(ft.text, "[REDACTED]") {
			t.Errorf("%s = %q, want the redacted text", ft.path, ft.text)
		}
	}
}

// @scenario specs/libs/retriever-contract/behaviours/identity-and-credentials.feature: An upper-case identity variable name that ends in _EMAIL is accepted
func TestIdentityNamesAcceptUpperCaseEmailVariableNames(t *testing.T) {
	for _, raw := range []string{"JIRA_EMAIL", "A_EMAIL", "TEAM_2_EMAIL"} {
		t.Run(raw, func(t *testing.T) {
			ref, err := domain.NewIdentityRef(raw)
			noError(t, err)
			if got := ref.Name(); got != raw {
				t.Errorf("identity name = %q, want %q", got, raw)
			}
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/identity-and-credentials.feature: An identity name not upper-case or not ending in _EMAIL is refused
func TestIdentityNamesRefuseNameNotUpperCaseOrNotEndingInEmail(t *testing.T) {
	for _, raw := range []string{"jira_email", "JIRA_MAIL", "1JIRA_EMAIL", "_EMAIL"} {
		t.Run(raw, func(t *testing.T) {
			ref, err := domain.NewIdentityRef(raw)
			wantText(t, err, "invalid identity name: use an upper-case name that ends in _EMAIL")
			if ref != (domain.IdentityRef{}) {
				t.Errorf("returned identity name %q, want none", ref.Name())
			}
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/identity-and-credentials.feature: A credential variable name that ends in _API_KEY or _API_TOKEN is accepted
func TestCredentialNamesAcceptNamesEndingInAPIKeyOrAPIToken(t *testing.T) {
	for _, raw := range []string{"OUTLINE_API_KEY", "KIWI_API_TOKEN", "A_API_KEY"} {
		t.Run(raw, func(t *testing.T) {
			ref, err := domain.NewCredentialRef(raw)
			noError(t, err)
			if got := ref.Name(); got != raw {
				t.Errorf("credential name = %q, want %q", got, raw)
			}
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/identity-and-credentials.feature: A credential variable name of any other shape is refused
func TestCredentialNamesRefuseNameOfAnyOtherShape(t *testing.T) {
	for _, raw := range []string{"outline_api_key", "OUTLINE_API_SECRET", "_API_KEY", "OUTLINE_API_KEYS", "OUTLINE_KEY"} {
		t.Run(raw, func(t *testing.T) {
			ref, err := domain.NewCredentialRef(raw)
			wantText(t, err, "invalid credential name")
			if ref != (domain.CredentialRef{}) {
				t.Errorf("returned credential name %q, want none", ref.Name())
			}
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/identity-and-credentials.feature: An API key that is empty is refused and the error names its variable only
func TestAPIKeyEmptyIsRefusedAndErrorNamesVariableOnly(t *testing.T) {
	key, err := domain.NewApiKey("OUTLINE_API_KEY", "")
	wantText(t, err, "OUTLINE_API_KEY is not set")
	if key != (domain.ApiKey{}) {
		t.Errorf("returned an API key, want none")
	}
}

// @scenario specs/libs/retriever-contract/behaviours/identity-and-credentials.feature: An API key shows as a redacted text in every standard formatting path
func TestAPIKeyShowsRedactedTextInEveryStandardFormattingPath(t *testing.T) {
	const secret = "EXAMPLE_API_KEY_VALUE"
	key, err := domain.NewApiKey("OUTLINE_API_KEY", secret)
	noError(t, err)
	jsonBytes, err := json.Marshal(key)
	noError(t, err)
	textBytes, err := key.MarshalText()
	noError(t, err)
	checkRedacted(t, secret, []formattedText{
		{"%v", fmt.Sprintf("%v", key)},
		{"%+v", fmt.Sprintf("%+v", key)},
		{"%s", fmt.Sprintf("%s", key)},
		{"%#v", fmt.Sprintf("%#v", key)},
		{"%q", fmt.Sprintf("%q", key)},
		{"JSON", string(jsonBytes)},
		{"text", string(textBytes)},
		{"log value", key.LogValue().String()},
	})
}

// @scenario specs/libs/retriever-contract/behaviours/identity-and-credentials.feature: A valid account address is accepted and shown redacted
func TestIdentityAcceptsValidAccountAddressAndShowsItRedacted(t *testing.T) {
	const address = "name@example.test"
	identity, err := domain.NewIdentity("JIRA_EMAIL", address)
	noError(t, err)
	jsonBytes, err := json.Marshal(identity)
	noError(t, err)
	textBytes, err := identity.MarshalText()
	noError(t, err)
	checkRedacted(t, address, []formattedText{
		{"%v", fmt.Sprintf("%v", identity)},
		{"%+v", fmt.Sprintf("%+v", identity)},
		{"%s", fmt.Sprintf("%s", identity)},
		{"%#v", fmt.Sprintf("%#v", identity)},
		{"%q", fmt.Sprintf("%q", identity)},
		{"JSON", string(jsonBytes)},
		{"text", string(textBytes)},
		{"log value", identity.LogValue().String()},
	})
}

// @scenario specs/libs/retriever-contract/behaviours/identity-and-credentials.feature: An account address that is empty, invalid, or has a colon is refused
func TestIdentityRefusesEmptyInvalidOrColonAddress(t *testing.T) {
	cases := []struct{ value, message string }{
		{"", "JIRA_EMAIL is not set"},
		{"name@example", "JIRA_EMAIL is not a valid e-mail address"},
		{"na:me@example.test", "JIRA_EMAIL is not a valid e-mail address"},
	}
	for _, tc := range cases {
		t.Run(tc.value, func(t *testing.T) {
			identity, err := domain.NewIdentity("JIRA_EMAIL", tc.value)
			wantText(t, err, tc.message)
			if identity != (domain.Identity{}) {
				t.Errorf("returned an identity, want none")
			}
		})
	}
}
