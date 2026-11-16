package catalog

import (
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// profileLoader returns a loader for the catalog text for the profile name.
func profileLoader(t *testing.T, text, profile string) *Loader {
	t.Helper()
	loader, err := NewWithFiles("/cfg/retriever/profiles/"+profile+"/sources.yml", contentFiles(text), domain.BuiltinKinds())
	if err != nil {
		t.Fatalf("NewWithFiles: %v", err)
	}
	return loader.WithProfile(profile)
}

// TestPersonalCatalogRefusesAWorkPrefixedVariable checks F1: a personal catalog that names a work-prefixed credential
// fails to load, so the personal run never reads the work key.
func TestPersonalCatalogRefusesAWorkPrefixedVariable(t *testing.T) {
	loader := profileLoader(t, `version: 1
sources:
  - id: wiki
    kind: outline
    endpoint: https://docs.example.test
    credential_env: WORK_OUTLINE_API_KEY
`, "personal")
	_, err := loader.Load()
	if err == nil || !strings.Contains(err.Error(), "credential_env") {
		t.Fatalf("Load error = %v, want the credential_env rule", err)
	}
	if strings.Contains(err.Error(), "WORK_OUTLINE_API_KEY") {
		t.Fatalf("Load error echoes the variable value: %q", err)
	}
}

// TestPersonalCatalogRefusesAWorkPrefixedIdentity checks the same prefix rule for the identity variable.
func TestPersonalCatalogRefusesAWorkPrefixedIdentity(t *testing.T) {
	loader := profileLoader(t, `version: 1
sources:
  - id: tickets
    kind: jira
    endpoint: https://jira.example.test
    credential_env: PERSONAL_JIRA_API_TOKEN
    identity_env: WORK_JIRA_EMAIL
`, "personal")
	_, err := loader.Load()
	if err == nil || !strings.Contains(err.Error(), "identity_env") {
		t.Fatalf("Load error = %v, want the identity_env rule", err)
	}
}

// TestPersonalCatalogLoadsWithItsOwnPrefix checks that the personal prefix loads: the control is the boundary, not a
// refusal of every catalog.
func TestPersonalCatalogLoadsWithItsOwnPrefix(t *testing.T) {
	loader := profileLoader(t, `version: 1
sources:
  - id: wiki
    kind: outline
    endpoint: https://docs.example.test
    credential_env: PERSONAL_OUTLINE_API_KEY
`, "personal")
	cat, err := loader.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cat.All()) != 1 {
		t.Fatalf("catalog holds %d sources, want 1", len(cat.All()))
	}
}

// TestHyphenatedProfileUsesUnderscoresInItsPrefix checks that a hyphen in the profile name reads as an underscore in
// the credential prefix.
func TestHyphenatedProfileUsesUnderscoresInItsPrefix(t *testing.T) {
	loader := profileLoader(t, `version: 1
sources:
  - id: wiki
    kind: outline
    endpoint: https://docs.example.test
    credential_env: MY_TEAM_OUTLINE_API_KEY
`, "my-team")
	if _, err := loader.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	bad := profileLoader(t, `version: 1
sources:
  - id: wiki
    kind: outline
    endpoint: https://docs.example.test
    credential_env: MY-TEAM_OUTLINE_API_KEY
`, "my-team")
	if _, err := bad.Load(); err == nil {
		t.Fatal("Load accepted a credential with a hyphen, want the prefix rule")
	}
}

// TestCatalogWithoutAProfileKeepsTheOldNames checks that the prefix rule applies only to a profile run: a catalog
// without a profile loads the example names as before.
func TestCatalogWithoutAProfileKeepsTheOldNames(t *testing.T) {
	loader, err := NewWithFiles("/work/sources.yml", contentFiles(oneSource), domain.BuiltinKinds())
	if err != nil {
		t.Fatalf("NewWithFiles: %v", err)
	}
	if _, err := loader.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
}
