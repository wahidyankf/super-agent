package domain

import (
	"errors"
	"strings"
	"testing"
)

// TestValidateIndexAcceptsOneToTwoHundredFiftySixPrintableCharacters checks the fractional index of a collection move
// (ADR 007 section 2, the idx notation): 1 to 256 printable ASCII characters, from 0x20 to 0x7E.
func TestValidateIndexAcceptsOneToTwoHundredFiftySixPrintableCharacters(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want error
	}{
		{"one letter", "P", nil},
		{"a space is printable", "a b", nil},
		{"256 characters", strings.Repeat("a", 256), nil},
		{"empty", "", ErrInvalidIndex},
		{"257 characters", strings.Repeat("a", 257), ErrInvalidIndex},
		{"a control byte", "a\x07b", ErrInvalidIndex},
		{"a tab", "a\tb", ErrInvalidIndex},
		{"a non-ASCII letter", "é", ErrInvalidIndex},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateIndex(tc.raw); !errors.Is(err, tc.want) {
				t.Fatalf("ValidateIndex(%q) = %v, want %v", tc.raw, err, tc.want)
			}
		})
	}
}
