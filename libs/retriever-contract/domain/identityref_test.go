package domain

import (
	"errors"
	"strings"
	"testing"
)

// TestNewIdentityRefAcceptsOnlyEmailVariableNames checks the name rule of an identity variable: an upper-case name that ends in
// _EMAIL, 7 to 64 bytes long, and nothing else.
func TestNewIdentityRefAcceptsOnlyEmailVariableNames(t *testing.T) {
	ok := []string{
		"A_EMAIL",
		"EXAMPLE_JIRA_EMAIL",
		"X1_2_EMAIL",
		"A" + strings.Repeat("B", 57) + "_EMAIL", // 64 bytes in total
	}
	for _, raw := range ok {
		ref, err := NewIdentityRef(raw)
		if err != nil {
			t.Errorf("NewIdentityRef(%q) = %v, want nil", raw, err)
			continue
		}
		if ref.Name() != raw {
			t.Errorf("Name() = %q, want %q", ref.Name(), raw)
		}
	}
	bad := []string{
		"",
		"_EMAIL",
		"EMAIL",
		"A_EMAIL_X",
		"a_EMAIL",
		"1A_EMAIL",
		"A-B_EMAIL",
		"A_API_KEY",
		"A_EMAILS",
		"A" + strings.Repeat("B", 58) + "_EMAIL", // 65 bytes in total
		"EXAMPLE_JIRA_EMAIL ",
		" EXAMPLE_JIRA_EMAIL",
		"EXAMPLE_JIRA_EMAIL\n",
		"ann@example.com",
	}
	for _, raw := range bad {
		if _, err := NewIdentityRef(raw); !errors.Is(err, ErrInvalidIdentityRef) {
			t.Errorf("NewIdentityRef(%q) = %v, want ErrInvalidIdentityRef", raw, err)
		}
	}
}

// TestIdentityRefErrorNeverEchoesTheRawName checks that a refused name is not shown in the refusal text.
func TestIdentityRefErrorNeverEchoesTheRawName(t *testing.T) {
	const raw = "sentinel-name-not-a-variable"
	_, err := NewIdentityRef(raw)
	if err == nil {
		t.Fatal("NewIdentityRef accepted a lower-case name")
	}
	if strings.Contains(err.Error(), raw) {
		t.Fatalf("refusal text echoes the name: %q", err.Error())
	}
}

// TestZeroIdentityRefHasNoName checks that the zero reference names no variable, so a reader can refuse it.
func TestZeroIdentityRefHasNoName(t *testing.T) {
	var zero IdentityRef
	if zero.Name() != "" {
		t.Fatalf("zero IdentityRef Name() = %q, want empty", zero.Name())
	}
}
