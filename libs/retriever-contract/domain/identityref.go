package domain

import (
	"errors"
	"regexp"
)

// identityPattern is the shape of an environment variable name that holds a source identity (an e-mail address).
var identityPattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]*_EMAIL$`)

// Length limits of an identity name in bytes.
const (
	minIdentityLength = 7
	maxIdentityLength = 64
)

// ErrInvalidIdentityRef reports a variable name that is not an upper-case *_EMAIL name of 7 to 64 bytes.
var ErrInvalidIdentityRef = errors.New("invalid identity name: use an upper-case name that ends in _EMAIL")

// IdentityRef is the name of the environment variable that holds one source's identity. It is never the address.
// The type has no method that reads a value from the environment. Build it with NewIdentityRef.
type IdentityRef struct {
	name string
}

// NewIdentityRef accepts only names that match ^[A-Z][A-Z0-9_]*_EMAIL$ and are 7 to 64 bytes long.
func NewIdentityRef(raw string) (IdentityRef, error) {
	if len(raw) < minIdentityLength || len(raw) > maxIdentityLength || !identityPattern.MatchString(raw) {
		return IdentityRef{}, ErrInvalidIdentityRef
	}
	return IdentityRef{name: raw}, nil
}

// Name returns the variable name. It is a name, never an address.
func (r IdentityRef) Name() string {
	return r.name
}
