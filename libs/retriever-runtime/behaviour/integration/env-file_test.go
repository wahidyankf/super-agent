//go:build integration

package integration_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/envfile"
)

// envFolder returns the env-file reader over the real process environment and the real folder dir, whose env file is
// .env.local.
func envFolder(t *testing.T, dir string) *envfile.Credentials {
	t.Helper()
	c, err := envfile.NewWithFiles(filepath.Join(dir, ".env.local"), os.Getenv,
		envfile.Files{FS: os.DirFS(dir), SameFile: os.SameFile})
	if err != nil {
		t.Fatalf("NewWithFiles: %v", err)
	}
	return c
}

// outlineKeyRef returns the reference to the Outline key variable.
func outlineKeyRef(t *testing.T) domain.CredentialRef {
	t.Helper()
	ref, err := domain.NewCredentialRef("EXAMPLE_OUTLINE_API_KEY")
	if err != nil {
		t.Fatalf("NewCredentialRef: %v", err)
	}
	return ref
}

// wantKey reads the Outline key through c and fails the test unless it is want.
func wantKey(t *testing.T, c *envfile.Credentials, want string) {
	t.Helper()
	key, err := c.APIKey(outlineKeyRef(t))
	if err != nil {
		t.Fatalf("APIKey error = %v", err)
	}
	if key.Reveal() != want {
		t.Fatalf("key = %q, want %q", key.Reveal(), want)
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/env-file.feature: The process value wins and the env file is not read
func TestTheProcessValueWinsAndTheEnvFileIsNotRead(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{".env.local": "EXAMPLE_OUTLINE_API_KEY=EXAMPLE-FILE-KEY\n"})
	// The file is made unreadable. A read would fail, so a passing call shows the file was never opened.
	if err := os.Chmod(filepath.Join(dir, ".env.local"), 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Setenv("EXAMPLE_OUTLINE_API_KEY", "EXAMPLE-PROCESS-KEY")
	wantKey(t, envFolder(t, dir), "EXAMPLE-PROCESS-KEY")
}

// @scenario specs/libs/retriever-runtime/behaviours/env-file.feature: The env file value is read when the process has none
func TestTheEnvFileValueIsReadWhenTheProcessHasNone(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{".env.local": "EXAMPLE_OUTLINE_API_KEY=EXAMPLE-FILE-KEY\n"})
	unsetEnv(t, "EXAMPLE_OUTLINE_API_KEY")
	wantKey(t, envFolder(t, dir), "EXAMPLE-FILE-KEY")
}

// @scenario specs/libs/retriever-runtime/behaviours/env-file.feature: An empty process value counts as missing, so the file value is read
func TestAnEmptyProcessValueCountsAsMissingSoTheFileValueIsRead(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{".env.local": "EXAMPLE_OUTLINE_API_KEY=EXAMPLE-FILE-KEY\n"})
	t.Setenv("EXAMPLE_OUTLINE_API_KEY", "")
	wantKey(t, envFolder(t, dir), "EXAMPLE-FILE-KEY")
}

// @scenario specs/libs/retriever-runtime/behaviours/env-file.feature: A key missing from both places names its variable in the error
func TestAKeyMissingFromBothPlacesNamesItsVariableInTheError(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{".env.local": "EXAMPLE_OUTLINE_API_KEY=\n"})
	unsetEnv(t, "EXAMPLE_OUTLINE_API_KEY")
	_, err := envFolder(t, dir).APIKey(outlineKeyRef(t))
	wantError(t, err, "EXAMPLE_OUTLINE_API_KEY is not set")
}

// @scenario specs/libs/retriever-runtime/behaviours/env-file.feature: A key whose env file is absent is a missing key
func TestAKeyWhoseEnvFileIsAbsentIsAMissingKey(t *testing.T) {
	unsetEnv(t, "EXAMPLE_OUTLINE_API_KEY")
	_, err := envFolder(t, t.TempDir()).APIKey(outlineKeyRef(t))
	wantError(t, err, "EXAMPLE_OUTLINE_API_KEY is not set")
}

// @scenario specs/libs/retriever-runtime/behaviours/env-file.feature: A value in the env file is read with its quotes and comments taken off
func TestAValueInTheEnvFileIsReadWithItsQuotesAndCommentsTakenOff(t *testing.T) {
	cases := []struct {
		line  string
		value string
	}{
		{"EXAMPLE_OUTLINE_API_KEY=EXAMPLE-KEY", "EXAMPLE-KEY"},
		{"export EXAMPLE_OUTLINE_API_KEY=EXAMPLE-KEY", "EXAMPLE-KEY"},
		{`EXAMPLE_OUTLINE_API_KEY="EXAMPLE KEY"`, "EXAMPLE KEY"},
		{"EXAMPLE_OUTLINE_API_KEY=EXAMPLE-KEY # a comment", "EXAMPLE-KEY"},
	}
	for _, tc := range cases {
		t.Run(tc.line, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, map[string]string{".env.local": tc.line + "\n"})
			unsetEnv(t, "EXAMPLE_OUTLINE_API_KEY")
			wantKey(t, envFolder(t, dir), tc.value)
		})
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/env-file.feature: A repeated variable in the env file takes its last value
func TestARepeatedVariableInTheEnvFileTakesItsLastValue(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{".env.local": "EXAMPLE_OUTLINE_API_KEY=FIRST\nEXAMPLE_OUTLINE_API_KEY=LAST\n"})
	unsetEnv(t, "EXAMPLE_OUTLINE_API_KEY")
	wantKey(t, envFolder(t, dir), "LAST")
}

// @scenario specs/libs/retriever-runtime/behaviours/env-file.feature: A malformed env file line is refused and the error gives its number
func TestAMalformedEnvFileLineIsRefusedAndTheErrorGivesItsNumber(t *testing.T) {
	const secretLine = "EXAMPLE-SECRET-VALUE"
	cases := []struct {
		name   string
		line   string
		reason string
	}{
		{"has no equals sign", secretLine, "expected NAME=VALUE"},
		{"has an unclosed quote", `EXAMPLE_OUTLINE_API_KEY="` + secretLine, "unclosed quote"},
		{"has text after a closing quote", `EXAMPLE_OUTLINE_API_KEY="a" ` + secretLine, "text after a closing quote"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, map[string]string{".env.local": "EXAMPLE_OUTLINE_API_KEY=FIRST\n" + tc.line + "\n"})
			unsetEnv(t, "EXAMPLE_OUTLINE_API_KEY")
			_, err := envFolder(t, dir).APIKey(outlineKeyRef(t))
			wantError(t, err, "local env file line 2: "+tc.reason)
			if strings.Contains(err.Error(), tc.line) {
				t.Fatalf("error %q holds the line text", err.Error())
			}
		})
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/env-file.feature: An env file that is a symbolic link or a folder is refused
func TestAnEnvFileThatIsASymbolicLinkOrAFolderIsRefused(t *testing.T) {
	cases := []struct {
		name  string
		setup func(t *testing.T, dir string)
	}{
		{"symbolic link", func(t *testing.T, dir string) {
			writeFiles(t, dir, map[string]string{"real.env": "EXAMPLE_OUTLINE_API_KEY=KEY\n"})
			if err := os.Symlink(filepath.Join(dir, "real.env"), filepath.Join(dir, ".env.local")); err != nil {
				t.Fatalf("symlink: %v", err)
			}
		}},
		{"folder", func(t *testing.T, dir string) {
			if err := os.Mkdir(filepath.Join(dir, ".env.local"), 0o755); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			tc.setup(t, dir)
			unsetEnv(t, "EXAMPLE_OUTLINE_API_KEY")
			_, err := envFolder(t, dir).APIKey(outlineKeyRef(t))
			wantError(t, err, "local env file must be a regular file, not a symlink or a folder")
		})
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/env-file.feature: An account address held in the env file is read when the process has none
func TestAnAccountAddressHeldInTheEnvFileIsReadWhenTheProcessHasNone(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{".env.local": "EXAMPLE_JIRA_EMAIL=name@example.test\n"})
	unsetEnv(t, "EXAMPLE_JIRA_EMAIL")
	ref, err := domain.NewIdentityRef("EXAMPLE_JIRA_EMAIL")
	if err != nil {
		t.Fatalf("NewIdentityRef: %v", err)
	}
	identity, err := envFolder(t, dir).Identity(ref)
	if err != nil {
		t.Fatalf("Identity error = %v", err)
	}
	if identity.Reveal() != "name@example.test" {
		t.Fatalf("identity = %q, want the address from the env file", identity.Reveal())
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/env-file.feature: A malformed account address is refused, naming the variable only
func TestAMalformedAccountAddressIsRefusedNamingTheVariableOnly(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{".env.local": "EXAMPLE_JIRA_EMAIL=EXAMPLE-NOT-AN-ADDRESS\n"})
	unsetEnv(t, "EXAMPLE_JIRA_EMAIL")
	ref, err := domain.NewIdentityRef("EXAMPLE_JIRA_EMAIL")
	if err != nil {
		t.Fatalf("NewIdentityRef: %v", err)
	}
	_, err = envFolder(t, dir).Identity(ref)
	wantError(t, err, "EXAMPLE_JIRA_EMAIL is not a valid e-mail address")
	if strings.Contains(err.Error(), "EXAMPLE-NOT-AN-ADDRESS") {
		t.Fatalf("error %q holds the address", err.Error())
	}
}
