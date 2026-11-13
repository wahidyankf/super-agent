//go:build integration

package jira

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSetBasicAuthIsCalledOnlyInAuthGo is a static guard: the Basic header is built in one place, so a second call that could
// leak the joined credentials cannot be added unnoticed. It reads the package sources, so it is an integration check.
func TestSetBasicAuthIsCalledOnlyInAuthGo(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	var callers []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		data, err := os.ReadFile(filepath.Clean(name))
		if err != nil {
			t.Fatalf("ReadFile %s: %v", name, err)
		}
		if strings.Contains(string(data), "SetBasicAuth(") {
			callers = append(callers, name)
		}
	}
	if len(callers) != 1 || callers[0] != "auth.go" {
		t.Fatalf("SetBasicAuth is called in %v, want only auth.go", callers)
	}
}
