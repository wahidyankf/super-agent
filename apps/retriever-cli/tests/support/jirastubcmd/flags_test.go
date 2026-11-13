package main

import (
	"bytes"
	"context"
	"testing"
)

// TestRunRefusesAnUnknownFlag checks that a flag the command does not know is an error.
func TestRunRefusesAnUnknownFlag(t *testing.T) {
	var out bytes.Buffer
	if err := run(context.Background(), []string{"-nope"}, &out); err == nil {
		t.Fatal("an unknown flag was accepted")
	}
}
