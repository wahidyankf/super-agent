package cli

import "testing"

// TestSecretEnvReadsOutlineKey proves the lookup returns the process value of the Outline API key, the secret that
// environment.contracts declares for the environment policy check. Retriever reads that key through its own credential
// path, and this case keeps SecretEnv in step with the contract.
func TestSecretEnvReadsOutlineKey(t *testing.T) {
	t.Setenv("EXAMPLE_OUTLINE_API_KEY", "outline-key")

	if got := SecretEnv("EXAMPLE_OUTLINE_API_KEY"); got != "outline-key" {
		t.Errorf("SecretEnv(EXAMPLE_OUTLINE_API_KEY) = %q, want %q", got, "outline-key")
	}
}

// TestSecretEnvOutlineKeyEmptyWhenUnset proves an unset Outline API key reads as the empty string.
func TestSecretEnvOutlineKeyEmptyWhenUnset(t *testing.T) {
	t.Setenv("EXAMPLE_OUTLINE_API_KEY", "")

	if got := SecretEnv("EXAMPLE_OUTLINE_API_KEY"); got != "" {
		t.Errorf("SecretEnv(EXAMPLE_OUTLINE_API_KEY) = %q, want empty", got)
	}
}
