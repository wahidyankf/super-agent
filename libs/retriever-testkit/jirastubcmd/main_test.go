package main

import (
	"os"
	"testing"
)

// TestMainEndsTheProcessWithTheStatusOfTheRun runs main with no -ca-file. The command refuses that before it reads any
// file or opens a socket, so the run touches nothing, and main ends the process once with status 1.
func TestMainEndsTheProcessWithTheStatusOfTheRun(t *testing.T) {
	savedExit, savedArgs := exit, os.Args
	t.Cleanup(func() { exit, os.Args = savedExit, savedArgs })
	var codes []int
	exit = func(code int) { codes = append(codes, code) }
	os.Args = []string{"jirastubcmd"}

	main()

	if len(codes) != 1 || codes[0] != 1 {
		t.Fatalf("main ended the process with %v, want one exit with status 1", codes)
	}
}
