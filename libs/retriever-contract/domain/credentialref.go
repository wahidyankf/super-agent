package domain

import "regexp"

// credentialPattern is the shape of an environment variable name that holds a source API key or API token.
var credentialPattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]*_API_(KEY|TOKEN)$`)

// Length limits of a credential name in bytes.
const (
	minCredentialLength = 9
	maxCredentialLength = 64
)

// CredentialRef is the name of the environment variable that holds one source's API key. It is never the key.
// The type has no method that reads a value from the environment. Build it with NewCredentialRef.
type CredentialRef struct {
	name string
}

// NewCredentialRef accepts only names that match ^[A-Z][A-Z0-9_]*_API_KEY$, so no other variable can be named.
func NewCredentialRef(raw string) (CredentialRef, error) {
	if len(raw) < minCredentialLength || len(raw) > maxCredentialLength || !credentialPattern.MatchString(raw) {
		return CredentialRef{}, ErrInvalidCredentialRef
	}
	return CredentialRef{name: raw}, nil
}

// Name returns the variable name. It is a name, never a key value.
func (r CredentialRef) Name() string {
	return r.name
}
