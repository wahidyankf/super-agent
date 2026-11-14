package domain

import (
	"errors"
	"strings"
	"testing"
)

// TestCredentialRefLengthBoundary checks the 64-byte limit of a credential name: 64 bytes is accepted, 65 is refused.
func TestCredentialRefLengthBoundary(t *testing.T) {
	atLimit := "A" + strings.Repeat("B", 55) + "_API_KEY"
	overLimit := "A" + strings.Repeat("B", 56) + "_API_KEY"
	if len(atLimit) != 64 || len(overLimit) != 65 {
		t.Fatalf("fixtures are %d and %d bytes, want 64 and 65", len(atLimit), len(overLimit))
	}
	if ref, err := NewCredentialRef(atLimit); err != nil || ref.Name() != atLimit {
		t.Fatalf("NewCredentialRef(64 bytes) = %q, %v; want accepted", ref.Name(), err)
	}
	if _, err := NewCredentialRef(overLimit); !errors.Is(err, ErrInvalidCredentialRef) {
		t.Fatalf("NewCredentialRef(65 bytes) error = %v, want ErrInvalidCredentialRef", err)
	}
}

// hostOfLength returns a host name of exactly n bytes. The first three labels are 63 bytes each, the longest
// legal label, and the last label starts with a letter.
func hostOfLength(n int) string {
	label := strings.Repeat("a", 63)
	last := strings.Repeat("b", n-3*64)
	return label + "." + label + "." + label + "." + last
}

// TestEndpointHostLengthBoundary checks the 253-byte limit of a host name: 253 bytes is accepted, 254 is refused.
func TestEndpointHostLengthBoundary(t *testing.T) {
	atLimit, overLimit := hostOfLength(253), hostOfLength(254)
	if len(atLimit) != 253 || len(overLimit) != 254 {
		t.Fatalf("fixtures are %d and %d bytes, want 253 and 254", len(atLimit), len(overLimit))
	}
	if ep, err := NewEndpoint("https://" + atLimit); err != nil || ep.Host() != atLimit {
		t.Fatalf("NewEndpoint(253-byte host) error = %v, want accepted", err)
	}
	if _, err := NewEndpoint("https://" + overLimit); !errors.Is(err, ErrInvalidEndpoint) {
		t.Fatalf("NewEndpoint(254-byte host) error = %v, want ErrInvalidEndpoint", err)
	}
}

// TestEndpointLabelLengthBoundary checks the 63-byte limit of one DNS label: 63 bytes is accepted, 64 is refused.
func TestEndpointLabelLengthBoundary(t *testing.T) {
	atLimit := "https://" + strings.Repeat("a", 63) + ".example.com"
	overLimit := "https://" + strings.Repeat("a", 64) + ".example.com"
	if _, err := NewEndpoint(atLimit); err != nil {
		t.Fatalf("NewEndpoint(63-byte label) error = %v, want accepted", err)
	}
	if _, err := NewEndpoint(overLimit); !errors.Is(err, ErrInvalidEndpoint) {
		t.Fatalf("NewEndpoint(64-byte label) error = %v, want ErrInvalidEndpoint", err)
	}
}
