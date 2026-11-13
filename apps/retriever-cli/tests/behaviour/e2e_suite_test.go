//go:build e2e

package behaviour

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// e2eBinary is the built retriever executable that the @e2e scenarios run. TestMain builds it once per test run.
var e2eBinary string

// TestMain builds the retriever binary from the module root, runs the tests, and removes the build folder.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "retriever-e2e-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "create the build folder: %v\n", err)
		os.Exit(1)
	}
	bin := filepath.Join(dir, "retriever")
	build := exec.Command("go", "build", "-trimpath", "-o", bin, "./cmd/retriever")
	build.Dir = "../.."
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "build the retriever binary: %v\n", err)
		_ = os.RemoveAll(dir)
		os.Exit(1)
	}
	e2eBinary = bin
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

// TestE2EBehaviours runs the @e2e scenarios against the built binary.
func TestE2EBehaviours(t *testing.T) {
	runSuite(t, levelE2E, "@e2e", e2eBinary)
}
