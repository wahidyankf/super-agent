package domain

import (
	"errors"
	"testing"
)

// TestNewCatalogRejectsAZeroValueFieldWithItsPosition checks that a source built as a zero value in any of its
// four required fields is refused with a SourceFault that names the position and the field. A zero Source can be
// written from outside the package, so NewCatalog must check it.
func TestNewCatalogRejectsAZeroValueFieldWithItsPosition(t *testing.T) {
	good := testSource(t, "outline", "outline", "docs.example.com", "GOOD_API_KEY")
	// other is a valid source with its own ID, host, and key, so only the zeroed field can break the catalog.
	other := testSource(t, "outline-other", "outline", "wiki.example.com", "OTHER_API_KEY")
	cases := []struct {
		name  string
		src   Source
		field string
		want  error
	}{
		{
			name:  "a zero ID",
			src:   Source{kind: other.kind, endpoint: other.endpoint, credential: other.credential},
			field: "id", want: ErrInvalidSourceID,
		},
		{
			name:  "a zero kind",
			src:   Source{id: other.id, endpoint: other.endpoint, credential: other.credential},
			field: "kind", want: ErrUnknownSourceKind,
		},
		{
			name:  "a zero endpoint",
			src:   Source{id: other.id, kind: other.kind, credential: other.credential},
			field: "endpoint", want: ErrInvalidEndpoint,
		},
		{
			name:  "a zero credential",
			src:   Source{id: other.id, kind: other.kind, endpoint: other.endpoint},
			field: "credential_env", want: ErrInvalidCredentialRef,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewCatalog([]Source{good, tc.src})
			var fault SourceFault
			if !errors.As(err, &fault) {
				t.Fatalf("NewCatalog error = %v, want a SourceFault", err)
			}
			if fault.Position != 2 || fault.Field != tc.field || !errors.Is(fault, tc.want) {
				t.Fatalf("fault = %+v, want position 2, field %q, and %v", fault, tc.field, tc.want)
			}
		})
	}
}

// TestNewCatalogZeroFaultTextNamesNoValue checks that the zero-value fault text carries the position, the field,
// and the rule only.
func TestNewCatalogZeroFaultTextNamesNoValue(t *testing.T) {
	good := testSource(t, "outline", "outline", "docs.example.com", "GOOD_API_KEY")
	other := testSource(t, "outline-other", "outline", "wiki.example.com", "OTHER_API_KEY")
	_, err := NewCatalog([]Source{good, {id: other.id, kind: other.kind, endpoint: other.endpoint}})
	if err == nil || err.Error() != "source 2 credential_env: invalid credential name" {
		t.Fatalf("error = %v, want the position, field, and rule only", err)
	}
}
