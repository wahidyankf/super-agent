package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestNewSearchQueryCapsTheLengthAtOneThousandCharacters(t *testing.T) {
	atCap := strings.Repeat("a", MaxQueryLength)
	if got, err := NewSearchQuery(atCap); err != nil || got != atCap {
		t.Fatalf("a query of %d characters was refused: %v", MaxQueryLength, err)
	}
	overCap := strings.Repeat("é", MaxQueryLength+1)
	if _, err := NewSearchQuery(overCap); !errors.Is(err, ErrInvalidQuery) {
		t.Fatalf("a query of %d characters error = %v, want ErrInvalidQuery", MaxQueryLength+1, err)
	}
}

func TestNewUUIDAcceptsOnlyTheCanonicalForm(t *testing.T) {
	good := []string{
		"123e4567-e89b-12d3-a456-426614174000",
		"ABCDEF01-2345-6789-abcd-ef0123456789",
	}
	for _, raw := range good {
		if err := ValidateUUID(raw); err != nil {
			t.Errorf("ValidateUUID(%q) = %v, want nil", raw, err)
		}
	}
	bad := []string{"", "col-1", "123e4567e89b12d3a456426614174000", "123e4567-e89b-12d3-a456-42661417400g", " 123e4567-e89b-12d3-a456-426614174000"}
	for _, raw := range bad {
		if err := ValidateUUID(raw); err == nil {
			t.Errorf("ValidateUUID(%q) = nil, want an error", raw)
		}
	}
}

func TestParameterValidatorsRefuseWhatTheCatalogForbids(t *testing.T) {
	cases := []struct {
		name string
		fn   func(string) error
		raw  string
		want error
	}{
		{"an ID with a dot", ValidateID, "a.b", ErrInvalidID},
		{"an empty ID", ValidateID, "", ErrInvalidID},
		{"a name with a control byte", ValidateName, "bad\x07name", ErrInvalidName},
		{"an empty name", ValidateName, "", ErrInvalidName},
		{"a 256 character name", ValidateName, strings.Repeat("n", 256), ErrInvalidName},
		{"a time without a zone", ValidateTime, "2026-11-04T10:00:00", ErrInvalidTime},
		{"a time in RFC 3339", ValidateTime, "2026-11-04T10:00:00Z", nil},
		{"an http URL", ValidateURL, "http://example.com/hook", ErrInvalidURL},
		{"a URL with credentials", ValidateURL, "https://user:pw@example.com/hook", ErrInvalidURL},
		{"an IP literal URL", ValidateURL, "https://127.0.0.1/hook", ErrInvalidURL},
		{"an HTTPS URL", ValidateURL, "https://hooks.example.com/in", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.fn(tc.raw)
			if tc.want == nil && err != nil {
				t.Fatalf("error = %v, want nil", err)
			}
			if tc.want != nil && !errors.Is(err, tc.want) {
				t.Fatalf("error = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestValidateTextAcceptsUpToOneMiB(t *testing.T) {
	if err := ValidateText(strings.Repeat("x", MaxTextBytes)); err != nil {
		t.Fatalf("a text of %d bytes was refused: %v", MaxTextBytes, err)
	}
	if err := ValidateText(strings.Repeat("x", MaxTextBytes+1)); !errors.Is(err, ErrInvalidText) {
		t.Fatalf("a text one byte over the cap error = %v, want ErrInvalidText", err)
	}
}

func TestValidateIDListCapsTheCountAndEachID(t *testing.T) {
	if err := ValidateIDList(make([]string, MaxIDList+1)); !errors.Is(err, ErrInvalidIDList) {
		t.Fatalf("51 IDs error = %v, want ErrInvalidIDList", err)
	}
	if err := ValidateIDList([]string{"a", "b.c"}); !errors.Is(err, ErrInvalidIDList) {
		t.Fatalf("an invalid member error = %v, want ErrInvalidIDList", err)
	}
	if err := ValidateIDList([]string{"a", "b"}); err != nil {
		t.Fatalf("two valid IDs error = %v, want nil", err)
	}
}

func TestAccessLevelErrorNamesTheSourceAndTheTwoLevels(t *testing.T) {
	src := testSource(t, "outline", "outline", "docs.example.com", "OUTLINE_API_KEY")
	err := AccessLevelError{Source: src.ID().String(), Have: AccessRead, Need: AccessWrite}
	want := `source "outline" allows read only; this operation needs write`
	if err.Error() != want {
		t.Fatalf("Error = %q, want %q", err.Error(), want)
	}
	if !errors.Is(err, ErrAccessLevelTooLow) {
		t.Fatal("the error does not match ErrAccessLevelTooLow")
	}
}
