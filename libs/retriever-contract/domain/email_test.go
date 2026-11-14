package domain

import (
	"errors"
	"strings"
	"testing"
)

// TestValidateEmailAcceptsOneAddressOfTheShapeOfRFC5321 checks the shape rule of an e-mail address: one "@", a local part, a
// host with a dot inside it, no space or control character, and at most MaxEmailLength characters.
func TestValidateEmailAcceptsOneAddressOfTheShapeOfRFC5321(t *testing.T) {
	for _, ok := range []string{"ann@example.com", "a.b+c@sub.example.co.id", strings.Repeat("a", 64) + "@example.com"} {
		if err := ValidateEmail(ok); err != nil {
			t.Errorf("ValidateEmail(%q) = %v, want nil", ok, err)
		}
	}
	for _, bad := range []string{
		"", "ann", "@example.com", "ann@", "ann@example", "ann@.example.com", "ann@example.",
		"ann@@example.com", "ann example.com", "ann\t@example.com", "ann\x07@example.com",
		"ann@exa\nple.com", strings.Repeat("a", MaxEmailLength) + "@example.com",
	} {
		if err := ValidateEmail(bad); !errors.Is(err, ErrInvalidEmail) {
			t.Errorf("ValidateEmail(%q) = %v, want ErrInvalidEmail", bad, err)
		}
	}
}

// TestInvalidEmailTextNeverHoldsTheAddress checks the fixed text of the refusal: it names the shape to give, and it never
// holds the address that was refused.
func TestInvalidEmailTextNeverHoldsTheAddress(t *testing.T) {
	if strings.Contains(ErrInvalidEmail.Error(), "ann") {
		t.Fatalf("ErrInvalidEmail holds an address: %q", ErrInvalidEmail.Error())
	}
	if !strings.Contains(ErrInvalidEmail.Error(), "name@example.com") {
		t.Fatalf("ErrInvalidEmail does not show the shape: %q", ErrInvalidEmail.Error())
	}
}

// TestInvalidEmailIsPublic checks that the e-mail refusal is shown with its fixed text.
func TestInvalidEmailIsPublic(t *testing.T) {
	public := PublicErrors()
	for _, want := range []error{ErrInvalidEmail} {
		found := false
		for _, e := range public {
			if e == want {
				found = true
			}
		}
		if !found {
			t.Fatalf("PublicErrors does not list %q", want)
		}
	}
}
