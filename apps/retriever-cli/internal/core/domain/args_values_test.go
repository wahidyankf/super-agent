package domain

import (
	"errors"
	"strings"
	"testing"
)

// TestArgsReadTheirOwnTypeOnly checks the typed getters of Args: a flag and a list read as set, and every getter reads
// the zero value of a name that is absent or of another type.
func TestArgsReadTheirOwnTypeOnly(t *testing.T) {
	a := NewArgs(map[string]any{"flag": true, "tags": []string{"a", "b"}, "name": "x"})
	if !a.Bool("flag") || a.Bool("name") || a.Bool("missing") {
		t.Fatalf("Bool: flag must read true, and a text or an absent name must read false")
	}
	if got := a.Strings("tags"); len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("Strings(tags) = %v, want [a b]", got)
	}
	if a.Strings("name") != nil || a.Strings("missing") != nil {
		t.Fatalf("Strings of a text or an absent name must be nil")
	}
}

// TestValidateNameAcceptsOneToMaxCharactersWithoutControls checks the bounds of a name: one character to the maximum,
// and no control character.
func TestValidateNameAcceptsOneToMaxCharactersWithoutControls(t *testing.T) {
	if err := ValidateName("a"); err != nil {
		t.Fatalf("ValidateName(one character) = %v, want nil", err)
	}
	if err := ValidateName(strings.Repeat("a", MaxNameLength)); err != nil {
		t.Fatalf("ValidateName(maximum) = %v, want nil", err)
	}
	for _, bad := range []string{"", strings.Repeat("a", MaxNameLength+1), "a\tb"} {
		if err := ValidateName(bad); !errors.Is(err, ErrInvalidName) {
			t.Fatalf("ValidateName(%q) = %v, want ErrInvalidName", bad, err)
		}
	}
}
