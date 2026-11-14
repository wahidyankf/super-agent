package domain

import (
	"fmt"
	"log/slog"
)

// redacted is the only text an ApiKey ever writes in place of its value.
const redacted = "[REDACTED]"

// ApiKey holds a source API key. It redacts itself in every standard formatting
// path, so the key cannot leak through fmt, JSON, text, or slog. Only the outline
// adapter calls Reveal, to build the Authorization header.
type ApiKey struct {
	value string
}

// MissingKeyError reports that the variable holding a source's API key is unset or empty.
// Its text names the variable and never a value.
type MissingKeyError struct {
	Variable string
}

// NewApiKey wraps value. An empty value returns a MissingKeyError that names variable.
func NewApiKey(variable, value string) (ApiKey, error) {
	if value == "" {
		return ApiKey{}, MissingKeyError{Variable: variable}
	}
	return ApiKey{value: value}, nil
}

// Error names the missing variable.
func (e MissingKeyError) Error() string {
	return e.Variable + " is not set"
}

// Reveal returns the key. Only the outline adapter calls it.
func (k ApiKey) Reveal() string {
	return k.value
}

// String returns the redacted text.
func (k ApiKey) String() string {
	return redacted
}

// GoString returns the redacted text for %#v.
func (k ApiKey) GoString() string {
	return redacted
}

// Format writes the redacted text for every verb. It takes precedence over String and GoString.
func (k ApiKey) Format(f fmt.State, _ rune) {
	fmt.Fprint(f, redacted)
}

// MarshalJSON encodes the redacted text as a JSON string.
func (k ApiKey) MarshalJSON() ([]byte, error) {
	return []byte(`"` + redacted + `"`), nil
}

// MarshalText encodes the redacted text.
func (k ApiKey) MarshalText() ([]byte, error) {
	return []byte(redacted), nil
}

// LogValue logs the redacted text.
func (k ApiKey) LogValue() slog.Value {
	return slog.StringValue(redacted)
}
