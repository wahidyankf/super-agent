package domain

import (
	"errors"
	"strings"
	"testing"
)

// TestEveryIDKindHasTheSameLengthBound checks that each ID kind accepts MaxIDLength bytes and refuses one more. A
// document or collection ID is a route segment, so no kind may take a longer one than the others.
func TestEveryIDKindHasTheSameLengthBound(t *testing.T) {
	if MaxIDLength != 255 {
		t.Fatalf("MaxIDLength = %d, want 255", MaxIDLength)
	}
	atLimit := strings.Repeat("a", MaxIDLength)
	overLimit := strings.Repeat("a", MaxIDLength+1)
	if err := ValidateID(atLimit); err != nil {
		t.Fatalf("ValidateID(%d bytes) = %v, want nil", MaxIDLength, err)
	}
	if err := ValidateID(overLimit); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("ValidateID(%d bytes) = %v, want ErrInvalidID", MaxIDLength+1, err)
	}
	if _, err := NewCollectionID(overLimit); !errors.Is(err, ErrInvalidCollectionID) {
		t.Fatalf("NewCollectionID(%d bytes) = %v, want ErrInvalidCollectionID", MaxIDLength+1, err)
	}
	if _, err := NewDocumentID(overLimit); !errors.Is(err, ErrInvalidDocumentID) {
		t.Fatalf("NewDocumentID(%d bytes) = %v, want ErrInvalidDocumentID", MaxIDLength+1, err)
	}
	slugAtLimit := strings.Repeat("a", MaxIDLength)
	slugOver := strings.Repeat("a", MaxIDLength+1)
	if err := ValidateShareID(slugAtLimit); err != nil {
		t.Fatalf("ValidateShareID(slug of %d bytes) = %v, want nil", MaxIDLength, err)
	}
	if err := ValidateShareID(slugOver); !errors.Is(err, ErrInvalidShareID) {
		t.Fatalf("ValidateShareID(slug of %d bytes) = %v, want ErrInvalidShareID", MaxIDLength+1, err)
	}
}
