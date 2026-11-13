package domain

import (
	"errors"
	"testing"
)

// TestNewCredentialRefAcceptsOnlyKeyAndTokenNames checks the credential-name grammar: a name ends in _API_KEY or
// _API_TOKEN, and every other name is refused before it can name a variable.
func TestNewCredentialRefAcceptsOnlyKeyAndTokenNames(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{name: "an outline key name is accepted", raw: "EXAMPLE_OUTLINE_API_KEY"},
		{name: "a jira token name is accepted", raw: "EXAMPLE_JIRA_API_TOKEN"},
		{name: "a lowercase name is refused", raw: "example_api_key", wantErr: true},
		{name: "a name without the suffix is refused", raw: "EXAMPLE_JIRA_SECRET", wantErr: true},
		{name: "a name too short is refused", raw: "API_KEY", wantErr: true},
		{name: "an empty name is refused", raw: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref, err := NewCredentialRef(tt.raw)
			if tt.wantErr {
				if !errors.Is(err, ErrInvalidCredentialRef) {
					t.Fatalf("NewCredentialRef(%q) error = %v; want ErrInvalidCredentialRef", tt.raw, err)
				}
				return
			}
			if err != nil || ref.Name() != tt.raw {
				t.Fatalf("NewCredentialRef(%q) = %q, %v; want the name back", tt.raw, ref.Name(), err)
			}
		})
	}
}
