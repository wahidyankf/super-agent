package domain

import (
	"errors"
	"testing"
)

// TestCatalogErrorExposesItsCauseToErrorsIs checks that a catalog error passes its typed cause through Unwrap, so the
// failure code can name the kind and the reason, and an error with no cause unwraps to nothing.
func TestCatalogErrorExposesItsCauseToErrorsIs(t *testing.T) {
	tests := []struct {
		name  string
		err   error
		cause error
		want  bool
	}{
		{name: "a catalog error with a cause matches that cause", err: CatalogError{Message: "source 1 kind: unknown", Cause: ErrUnknownSourceKind}, cause: ErrUnknownSourceKind, want: true},
		{name: "a catalog error matches a cause it does not hold", err: CatalogError{Message: "sources: invalid", Cause: ErrUnknownSourceKind}, cause: ErrInvalidSourceID, want: false},
		{name: "a catalog error with no cause matches nothing", err: CatalogError{Message: "sources: invalid"}, cause: ErrUnknownSourceKind, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := errors.Is(tt.err, tt.cause); got != tt.want {
				t.Fatalf("errors.Is = %v, want %v", got, tt.want)
			}
		})
	}
	if got := (CatalogError{Message: "sources: invalid"}).Unwrap(); got != nil {
		t.Fatalf("Unwrap of an error with no cause = %v, want nil", got)
	}
}
