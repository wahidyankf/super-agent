package domain

import (
	"errors"
	"testing"
)

func TestNewCollectionIDAcceptsOnlyTheAllowedShape(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{name: "a plain letter and digit ID is accepted", raw: "col1", wantErr: false},
		{name: "a UUID with hyphens is accepted", raw: "3f2a9c1e-0b7d-4e55-9a1f-6c2d8e4b7a10", wantErr: false},
		{name: "an underscore is accepted", raw: "col_1", wantErr: false},
		{name: "an empty ID is rejected", raw: "", wantErr: true},
		{name: "a dot is rejected", raw: "col.1", wantErr: true},
		{name: "a percent sign is rejected", raw: "doc%2F1", wantErr: true},
		{name: "a semicolon is rejected", raw: "col;ls", wantErr: true},
		{name: "a slash is rejected", raw: "col/1", wantErr: true},
		{name: "a trailing newline is rejected", raw: "col1\n", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := NewCollectionID(tt.raw)
			if tt.wantErr {
				if !errors.Is(err, ErrInvalidCollectionID) {
					t.Fatalf("want ErrInvalidCollectionID, got %v", err)
				}
				return
			}
			if err != nil || id.String() != tt.raw {
				t.Fatalf("want %q accepted, got %q (err %v)", tt.raw, id.String(), err)
			}
		})
	}
}

func TestNewDocumentIDAcceptsOnlyTheAllowedShape(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{name: "a plain document ID is accepted", raw: "doc-1", wantErr: false},
		{name: "a URL ID with letters and hyphens is accepted", raw: "getting-started-AbC_9", wantErr: false},
		{name: "an empty ID is rejected", raw: "", wantErr: true},
		{name: "a dot is rejected", raw: "doc.1", wantErr: true},
		{name: "a percent sign is rejected", raw: "doc%2F1", wantErr: true},
		{name: "a semicolon is rejected", raw: "doc;ls", wantErr: true},
		{name: "a space is rejected", raw: "doc 1", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := NewDocumentID(tt.raw)
			if tt.wantErr {
				if !errors.Is(err, ErrInvalidDocumentID) {
					t.Fatalf("want ErrInvalidDocumentID, got %v", err)
				}
				return
			}
			if err != nil || id.String() != tt.raw {
				t.Fatalf("want %q accepted, got %q (err %v)", tt.raw, id.String(), err)
			}
		})
	}
}
