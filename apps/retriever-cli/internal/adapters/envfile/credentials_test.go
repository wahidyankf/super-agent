//go:build integration

package envfile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// outlineKey returns the revealed key for source, failing the test on any error.
func outlineKey(t *testing.T, c *Credentials) string {
	t.Helper()
	key, err := c.APIKey(outlineRef(t))
	if err != nil {
		t.Fatalf("APIKey(outline): %v", err)
	}
	return key.Reveal()
}

// missingKey reports whether err is the MissingKeyError that names the Outline variable.
func missingKey(err error) bool {
	var missing domain.MissingKeyError
	return errors.As(err, &missing) && missing.Variable == outlineVariable
}

func TestAPIKeyPrefersTheProcessEnvironment(t *testing.T) {
	root := newRoot(t, outlineVariable+"=file-value\n")
	t.Setenv(outlineVariable, "env-value")
	if got := outlineKey(t, newCredentials(t, root)); got != "env-value" {
		t.Fatalf("key = %q, want the process value %q", got, "env-value")
	}
}

func TestAPIKeyReadsTheRepositoryEnvFileWhenTheEnvironmentIsUnset(t *testing.T) {
	root := newRoot(t, outlineVariable+"=file-value\n")
	unsetEnv(t, outlineVariable)
	if got := outlineKey(t, newCredentials(t, root)); got != "file-value" {
		t.Fatalf("key = %q, want the file value %q", got, "file-value")
	}
}

func TestAPIKeyReadsTheFileWhenTheEnvironmentValueIsEmpty(t *testing.T) {
	root := newRoot(t, outlineVariable+"=file-value\n")
	t.Setenv(outlineVariable, "")
	if got := outlineKey(t, newCredentials(t, root)); got != "file-value" {
		t.Fatalf("key = %q, want the file value %q", got, "file-value")
	}
}

func TestAPIKeyReadsAQuotedFileValue(t *testing.T) {
	root := newRoot(t, "# local settings\nexport "+outlineVariable+`="file-value" # note`+"\n")
	unsetEnv(t, outlineVariable)
	if got := outlineKey(t, newCredentials(t, root)); got != "file-value" {
		t.Fatalf("key = %q, want %q", got, "file-value")
	}
}

func TestAPIKeyNamesTheVariableWhenMissingEverywhere(t *testing.T) {
	root := newRoot(t, "")
	unsetEnv(t, outlineVariable)
	_, err := newCredentials(t, root).APIKey(outlineRef(t))
	if !missingKey(err) {
		t.Fatalf("APIKey error = %v, want a MissingKeyError for %s", err, outlineVariable)
	}
	if want := outlineVariable + " is not set"; err.Error() != want {
		t.Fatalf("error text = %q, want %q", err.Error(), want)
	}
}

func TestAPIKeyTreatsAnEmptyFileValueAsMissing(t *testing.T) {
	root := newRoot(t, outlineVariable+"=\n")
	unsetEnv(t, outlineVariable)
	if _, err := newCredentials(t, root).APIKey(outlineRef(t)); !missingKey(err) {
		t.Fatalf("APIKey error = %v, want a MissingKeyError", err)
	}
}

func TestAPIKeyTreatsAFileWithoutTheVariableAsMissing(t *testing.T) {
	root := newRoot(t, "OTHER_SETTING=x\n")
	unsetEnv(t, outlineVariable)
	if _, err := newCredentials(t, root).APIKey(outlineRef(t)); !missingKey(err) {
		t.Fatalf("APIKey error = %v, want a MissingKeyError", err)
	}
}

func TestAPIKeyRefusesTheZeroReference(t *testing.T) {
	root := newRoot(t, "")
	if _, err := newCredentials(t, root).APIKey(domain.CredentialRef{}); !errors.Is(err, domain.ErrInvalidCredentialRef) {
		t.Fatalf("APIKey(zero) error = %v, want ErrInvalidCredentialRef", err)
	}
}

func TestAPIKeyReadsOnlyTheNamedVariable(t *testing.T) {
	t.Setenv("OTHER_API_KEY", "other-value")
	ref, err := domain.NewCredentialRef("NAMED_API_KEY")
	if err != nil {
		t.Fatalf("NewCredentialRef: %v", err)
	}
	_, err = newCredentials(t, newRoot(t, "")).APIKey(ref)
	var missing domain.MissingKeyError
	if !errors.As(err, &missing) || missing.Variable != "NAMED_API_KEY" {
		t.Fatalf("APIKey error = %v, want the named variable reported missing", err)
	}
}

func TestAPIKeyRefusesAnEnvFileThatIsASymlinkOutsideTheRoot(t *testing.T) {
	root := newRoot(t, "")
	outside := filepath.Join(t.TempDir(), "outside.env")
	mustWrite(t, outside, outlineVariable+"="+sentinel+"\n")
	mustSymlink(t, outside, filepath.Join(root, envFileName))
	unsetEnv(t, outlineVariable)

	_, err := newCredentials(t, root).APIKey(outlineRef(t))
	if err == nil || missingKey(err) {
		t.Fatalf("APIKey error = %v, want a refusal of the symlinked file", err)
	}
	if strings.Contains(err.Error(), sentinel) {
		t.Fatalf("refusal text leaks the value: %q", err.Error())
	}
}

func TestAPIKeyRefusesAnEnvFileThatIsAFolder(t *testing.T) {
	root := newRoot(t, "")
	mustMkdir(t, filepath.Join(root, envFileName))
	unsetEnv(t, outlineVariable)
	if _, err := newCredentials(t, root).APIKey(outlineRef(t)); err == nil || missingKey(err) {
		t.Fatalf("APIKey error = %v, want a refusal of a folder", err)
	}
}

func TestAPIKeyReportsAnUnreadableEnvFileWithoutItsContent(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads files whatever their mode, so an unreadable file cannot be made")
	}
	root := newRoot(t, outlineVariable+"="+sentinel+"\n")
	if err := os.Chmod(filepath.Join(root, envFileName), 0o000); err != nil {
		t.Fatalf("chmod unreadable: %v", err)
	}
	unsetEnv(t, outlineVariable)

	_, err := newCredentials(t, root).APIKey(outlineRef(t))
	if err == nil || missingKey(err) {
		t.Fatalf("APIKey error = %v, want a read failure", err)
	}
	if strings.Contains(err.Error(), sentinel) {
		t.Fatalf("read failure leaks the value: %q", err.Error())
	}
}

// TestAPIKeyErrorsNeverShowTheValue proves that a malformed file holding the sentinel fails without echoing it.
func TestAPIKeyErrorsNeverShowTheValue(t *testing.T) {
	root := newRoot(t, outlineVariable+"="+sentinel+"\nbroken "+sentinel+"\n")
	unsetEnv(t, outlineVariable)

	_, err := newCredentials(t, root).APIKey(outlineRef(t))
	if err == nil {
		t.Fatal("APIKey accepted a malformed file, want an error")
	}
	for _, text := range []string{err.Error(), fmt.Sprintf("%v", err), fmt.Sprintf("%+v", err)} {
		if strings.Contains(text, sentinel) {
			t.Fatalf("error text leaks the value: %q", text)
		}
	}
}

// TestAPIKeyFormattingNeverShowsTheValue proves the key stays redacted in every formatting path,
// including when it comes from the process environment.
func TestAPIKeyFormattingNeverShowsTheValue(t *testing.T) {
	root := newRoot(t, "")
	t.Setenv(outlineVariable, sentinel)
	key, err := newCredentials(t, root).APIKey(outlineRef(t))
	if err != nil {
		t.Fatalf("APIKey: %v", err)
	}
	for _, text := range []string{
		fmt.Sprintf("%v", key),
		fmt.Sprintf("%+v", key),
		fmt.Sprintf("%#v", key),
		fmt.Sprintf("%s", key),
	} {
		if strings.Contains(text, sentinel) {
			t.Fatalf("formatted key leaks the value: %q", text)
		}
	}
}
