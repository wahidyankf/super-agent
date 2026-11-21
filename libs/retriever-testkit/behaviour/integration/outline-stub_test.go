//go:build integration

package integration_test

import (
	"bytes"
	"encoding/pem"
	"os"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-testkit/outlinestub"
)

// @scenario specs/libs/retriever-testkit/behaviours/outline-stub.feature: The CA file of the stub is written with owner-only access
func TestCAFileOfStartedStubHoldsCertificateAndIsOwnerOnly(t *testing.T) {
	stub := outlinestub.Start(t)
	path, err := stub.WriteCAFile(t.TempDir())
	if err != nil {
		t.Fatalf("write the CA file: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read the CA file: %v", err)
	}
	want, err := stub.CAPEM()
	if err != nil {
		t.Fatalf("read the PEM of the stub: %v", err)
	}
	if !bytes.Equal(data, want) {
		t.Fatal("the CA file does not hold the certificate of the stub")
	}
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "CERTIFICATE" {
		t.Fatal("the CA file does not hold a PEM certificate")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat the CA file: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Fatalf("the CA file mode is %o, want owner only 600", mode)
	}
}
