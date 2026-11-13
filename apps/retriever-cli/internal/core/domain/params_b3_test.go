package domain

import (
	"errors"
	"math"
	"strings"
	"testing"
)

// TestValidateColorAcceptsOnlyHexWithThreeOrSixDigits checks the Color kind of stage b part 3 (the inventory's color
// notation): #RGB or #RRGGBB, in either letter case, and nothing else.
func TestValidateColorAcceptsOnlyHexWithThreeOrSixDigits(t *testing.T) {
	for _, raw := range []string{"#abc", "#ABC", "#1a2B3c", "#000000"} {
		if err := ValidateColor(raw); err != nil {
			t.Errorf("ValidateColor(%q) = %v, want nil", raw, err)
		}
	}
	for _, raw := range []string{"", "abc", "#ab", "#abcd", "#abcdeff", "#ggg", "red", " #abc"} {
		if err := ValidateColor(raw); !errors.Is(err, ErrInvalidColor) {
			t.Errorf("ValidateColor(%q) = %v, want ErrInvalidColor", raw, err)
		}
	}
}

// TestValidateIconAcceptsAnEmojiANameOrAnAttachmentUUID checks the Icon kind: one emoji, a named icon, or an attachment
// UUID. The server holds the list of named icons, so Retriever checks the shape only: non-blank, at most 255 characters,
// and no control character.
func TestValidateIconAcceptsAnEmojiANameOrAnAttachmentUUID(t *testing.T) {
	for _, raw := range []string{"📅", "calendar", "123e4567-e89b-12d3-a456-426614174000"} {
		if err := ValidateIcon(raw); err != nil {
			t.Errorf("ValidateIcon(%q) = %v, want nil", raw, err)
		}
	}
	atCap := strings.Repeat("a", MaxNameLength)
	if err := ValidateIcon(atCap); err != nil {
		t.Errorf("an icon of %d characters was refused: %v", MaxNameLength, err)
	}
	for _, raw := range []string{"", "   ", "a\tb", "a\x07b", strings.Repeat("a", MaxNameLength+1)} {
		if err := ValidateIcon(raw); !errors.Is(err, ErrInvalidIcon) {
			t.Errorf("ValidateIcon(%q) = %v, want ErrInvalidIcon", raw, err)
		}
	}
}

// TestValidateScoreAcceptsANumberFromMinusOneToOne checks the Score kind of searches.update (the inventory's num[-1..1]):
// a finite number from -1 to 1 inclusive. NaN and the infinities are refused.
func TestValidateScoreAcceptsANumberFromMinusOneToOne(t *testing.T) {
	for _, v := range []float64{-1, -0.5, 0, 0.25, 1} {
		if err := ValidateScore(v); err != nil {
			t.Errorf("ValidateScore(%v) = %v, want nil", v, err)
		}
	}
	for _, v := range []float64{-1.0001, 1.0001, math.NaN(), math.Inf(1), math.Inf(-1)} {
		if err := ValidateScore(v); !errors.Is(err, ErrInvalidScore) {
			t.Errorf("ValidateScore(%v) = %v, want ErrInvalidScore", v, err)
		}
	}
}
