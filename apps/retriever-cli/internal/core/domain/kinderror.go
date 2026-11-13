package domain

import (
	"errors"
	"strings"
)

// KindError names the kind of source an error came from. Unwrap returns Err, so errors.Is still finds the sentinel under
// it. Its text is the sentinel's text, with the word "Outline" replaced by the kind's display name for any kind other
// than Outline (ADR 010 section 7), so a Kiwi failure never tells the owner to rotate an Outline key.
type KindError struct {
	Kind SourceKind
	Err  error
}

// Error returns the text of Err, with "Outline" replaced by the display name of Kind for a kind other than Outline.
func (e KindError) Error() string {
	text := e.Err.Error()
	if e.Kind == KindOutline {
		return text
	}
	return strings.ReplaceAll(text, KindOutline.Display(), e.Kind.Display())
}

// Unwrap returns the wrapped error, so errors.Is and errors.As see through the kind.
func (e KindError) Unwrap() error {
	return e.Err
}

// Is reports whether target is a KindError of the same kind over an error that matches Err. An error of another kind
// never matches, so a presenter that tests the kind does not confuse two sources.
func (e KindError) Is(target error) bool {
	other, ok := target.(KindError)
	return ok && other.Kind == e.Kind && errors.Is(e.Err, other.Err)
}

// PublicText returns the text of the error and true when Err is a public sentinel, the only kind of text a presenter
// may show. Any other error returns false, and the presenter then uses its generic text.
func (e KindError) PublicText() (string, bool) {
	for _, typed := range PublicErrors() {
		if errors.Is(e.Err, typed) {
			return e.Error(), true
		}
	}
	return "", false
}
