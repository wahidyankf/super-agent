package domain

import (
	"fmt"
	"log/slog"
	"strings"
)

// Identity holds one source's account e-mail address. The address is not a key, but it is personal data, so the type
// redacts itself in every standard formatting path, as ApiKey does. Only the Jira client calls Reveal.
type Identity struct {
	value string
}

// InvalidIdentityError reports that the variable holding a source's e-mail address holds no usable address. Its text names
// the variable and never the value.
type InvalidIdentityError struct {
	Variable string
}

// Error names the variable that holds the refused address.
func (e InvalidIdentityError) Error() string {
	return e.Variable + " is not a valid e-mail address"
}

// NewIdentity wraps value as an address. An empty value returns a MissingKeyError that names variable. A value that
// ValidateEmail refuses, or that holds a colon (RFC 7617 forbids one in a Basic user name), returns an
// InvalidIdentityError that names variable. Nothing is trimmed, guessed, or defaulted.
func NewIdentity(variable, value string) (Identity, error) {
	if value == "" {
		return Identity{}, MissingKeyError{Variable: variable}
	}
	if strings.Contains(value, ":") || ValidateEmail(value) != nil {
		return Identity{}, InvalidIdentityError{Variable: variable}
	}
	return Identity{value: value}, nil
}

// Reveal returns the address. Only the Jira client calls it, to build the Basic authorization header.
func (i Identity) Reveal() string {
	return i.value
}

// String returns the redacted text.
func (i Identity) String() string {
	return redacted
}

// GoString returns the redacted text for %#v.
func (i Identity) GoString() string {
	return redacted
}

// Format writes the redacted text for every verb. It takes precedence over String and GoString.
func (i Identity) Format(f fmt.State, _ rune) {
	fmt.Fprint(f, redacted)
}

// MarshalJSON encodes the redacted text as a JSON string.
func (i Identity) MarshalJSON() ([]byte, error) {
	return []byte(`"` + redacted + `"`), nil
}

// MarshalText encodes the redacted text.
func (i Identity) MarshalText() ([]byte, error) {
	return []byte(redacted), nil
}

// LogValue logs the redacted text.
func (i Identity) LogValue() slog.Value {
	return slog.StringValue(redacted)
}
