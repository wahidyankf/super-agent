package outlineseam

import (
	"errors"
	"testing"
)

// TestSeamWithoutATimeoutKeepsTheDefault checks the unset timeout: a loopback base with a CA file and no timeout enables
// the seam, and the seam reports the default timeout as zero.
func TestSeamWithoutATimeoutKeepsTheDefault(t *testing.T) {
	ca := testCAPEM(t)
	seam, err := TestSeamFromEnvWithFiles(seamEnv(map[string]string{
		BaseURLEnv: "https://127.0.0.1:8443",
		CAFileEnv:  "ca.pem",
	}), func(string) ([]byte, error) { return ca, nil })
	if err != nil {
		t.Fatalf("TestSeamFromEnvWithFiles: %v", err)
	}
	if !seam.Enabled || seam.Timeout != 0 {
		t.Fatalf("seam enabled=%v timeout=%v, want an enabled seam with the default timeout", seam.Enabled, seam.Timeout)
	}
}

// TestSeamRefusesACAFileThatCannotBeRead checks that an unreadable CA file refuses the seam, and the error names the
// file's variable and keeps the cause.
func TestSeamRefusesACAFileThatCannotBeRead(t *testing.T) {
	seam, err := TestSeamFromEnvWithFiles(seamEnv(map[string]string{
		BaseURLEnv: "https://127.0.0.1:8443",
		CAFileEnv:  "ca.pem",
	}), failingRead)
	if !errors.Is(err, errNoCAFile) || seam.Enabled {
		t.Fatalf("enabled=%v err=%v, want the read error and no seam", seam.Enabled, err)
	}
}

// TestSeamRefusesACAFileWithNoPEMCertificate checks that a CA file holding no PEM certificate refuses the seam.
func TestSeamRefusesACAFileWithNoPEMCertificate(t *testing.T) {
	seam, err := TestSeamFromEnvWithFiles(seamEnv(map[string]string{
		BaseURLEnv: "https://127.0.0.1:8443",
		CAFileEnv:  "ca.pem",
	}), func(string) ([]byte, error) { return []byte("not a certificate"), nil })
	if !errors.Is(err, errSeamNoCertificate) || seam.Enabled {
		t.Fatalf("enabled=%v err=%v, want errSeamNoCertificate and no seam", seam.Enabled, err)
	}
}
