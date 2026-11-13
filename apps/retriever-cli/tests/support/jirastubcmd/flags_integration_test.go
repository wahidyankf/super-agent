//go:build integration

package main

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"testing"
)

// TestRunRefusesAMissingCAFileOrCredential checks the required flags: the run names the missing flag and starts nothing.
func TestRunRefusesAMissingCAFileOrCredential(t *testing.T) {
	ca := filepath.Join(t.TempDir(), "ca.pem")
	tests := []struct {
		name string
		args []string
		want error
	}{
		{name: "no CA file", args: []string{"-email", "a@example.com", "-token", "t"}, want: errNoCAFile},
		{name: "no e-mail", args: []string{"-ca-file", ca, "-token", "t"}, want: errNoCredentials},
		{name: "no token", args: []string{"-ca-file", ca, "-email", "a@example.com"}, want: errNoCredentials},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			err := run(context.Background(), tt.args, &out)
			if !errors.Is(err, tt.want) {
				t.Fatalf("run error = %v, want %v", err, tt.want)
			}
			if out.Len() != 0 {
				t.Fatalf("a refused run printed %q", out.String())
			}
		})
	}
}
