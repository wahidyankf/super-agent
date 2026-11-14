//go:build integration

package envfile

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// The fixtures below are synthetic. The addresses use the reserved example.com domain, and the key is a sentinel.
const (
	jiraEmailVariable = "EXAMPLE_JIRA_EMAIL"
	jiraKeyVariable   = "EXAMPLE_JIRA_API_TOKEN"
	mailSentinel      = "sentinel-mail-4c1e@example.com"
	mailMalformed     = "sentinel-no-at-sign-4c1e.example.com"
	jiraKeySentinel   = "sentinel-jira-token-7f3a9c"
)

// jiraEmailRef returns the identity reference of the Jira address, failing the test when it is invalid.
func jiraEmailRef(t *testing.T) domain.IdentityRef {
	t.Helper()
	ref, err := domain.NewIdentityRef(jiraEmailVariable)
	if err != nil {
		t.Fatalf("NewIdentityRef: %v", err)
	}
	return ref
}

// jiraKeyRef returns the credential reference of the Jira token, failing the test when it is invalid.
func jiraKeyRef(t *testing.T) domain.CredentialRef {
	t.Helper()
	ref, err := domain.NewCredentialRef(jiraKeyVariable)
	if err != nil {
		t.Fatalf("NewCredentialRef: %v", err)
	}
	return ref
}

// identityValue returns the revealed address for the Jira variable, failing the test on any error.
func identityValue(t *testing.T, c *Credentials) string {
	t.Helper()
	id, err := c.Identity(jiraEmailRef(t))
	if err != nil {
		t.Fatalf("Identity(jira): %v", err)
	}
	return id.Reveal()
}

// missingIdentity reports whether err is the MissingKeyError that names the Jira e-mail variable.
func missingIdentity(err error) bool {
	var missing domain.MissingKeyError
	return errors.As(err, &missing) && missing.Variable == jiraEmailVariable
}

func TestIdentityPrefersTheProcessEnvironment(t *testing.T) {
	root := newRoot(t, jiraEmailVariable+"=ann@example.org\n")
	t.Setenv(jiraEmailVariable, mailSentinel)
	if got := identityValue(t, newCredentials(t, root)); got != mailSentinel {
		t.Fatalf("identity = %q, want the process value", got)
	}
}

func TestIdentityReadsTheRepositoryEnvFileWhenTheEnvironmentIsUnset(t *testing.T) {
	root := newRoot(t, jiraEmailVariable+"="+mailSentinel+"\n")
	unsetEnv(t, jiraEmailVariable)
	if got := identityValue(t, newCredentials(t, root)); got != mailSentinel {
		t.Fatalf("identity = %q, want the file value", got)
	}
}

func TestIdentityReadsTheFileWhenTheEnvironmentValueIsEmpty(t *testing.T) {
	root := newRoot(t, jiraEmailVariable+"="+mailSentinel+"\n")
	t.Setenv(jiraEmailVariable, "")
	if got := identityValue(t, newCredentials(t, root)); got != mailSentinel {
		t.Fatalf("identity = %q, want the file value", got)
	}
}

func TestIdentityTreatsAnEmptyFileValueAsMissing(t *testing.T) {
	root := newRoot(t, jiraEmailVariable+"=\n")
	unsetEnv(t, jiraEmailVariable)
	if _, err := newCredentials(t, root).Identity(jiraEmailRef(t)); !missingIdentity(err) {
		t.Fatalf("Identity error = %v, want a MissingKeyError", err)
	}
}

func TestIdentityNamesTheVariableWhenMissingEverywhere(t *testing.T) {
	root := newRoot(t, "")
	unsetEnv(t, jiraEmailVariable)
	_, err := newCredentials(t, root).Identity(jiraEmailRef(t))
	if !missingIdentity(err) {
		t.Fatalf("Identity error = %v, want a MissingKeyError for %s", err, jiraEmailVariable)
	}
	if want := jiraEmailVariable + " is not set"; err.Error() != want {
		t.Fatalf("error text = %q, want %q", err.Error(), want)
	}
}

func TestIdentityRefusesAMalformedAddressWithoutItsValue(t *testing.T) {
	root := newRoot(t, "")
	t.Setenv(jiraEmailVariable, mailMalformed)
	_, err := newCredentials(t, root).Identity(jiraEmailRef(t))
	var invalid domain.InvalidIdentityError
	if !errors.As(err, &invalid) || invalid.Variable != jiraEmailVariable {
		t.Fatalf("Identity error = %v, want an InvalidIdentityError naming %s", err, jiraEmailVariable)
	}
	if strings.Contains(err.Error(), mailMalformed) {
		t.Fatalf("refusal text leaks the value: %q", err.Error())
	}
}

func TestIdentityRefusesAMalformedAddressFromTheFileWithoutItsValue(t *testing.T) {
	root := newRoot(t, jiraEmailVariable+"="+mailMalformed+"\n")
	unsetEnv(t, jiraEmailVariable)
	_, err := newCredentials(t, root).Identity(jiraEmailRef(t))
	var invalid domain.InvalidIdentityError
	if !errors.As(err, &invalid) {
		t.Fatalf("Identity error = %v, want an InvalidIdentityError", err)
	}
	if strings.Contains(err.Error(), mailMalformed) {
		t.Fatalf("refusal text leaks the value: %q", err.Error())
	}
}

func TestIdentityRefusesTheZeroReference(t *testing.T) {
	root := newRoot(t, "")
	if _, err := newCredentials(t, root).Identity(domain.IdentityRef{}); !errors.Is(err, domain.ErrInvalidIdentityRef) {
		t.Fatalf("Identity(zero) error = %v, want ErrInvalidIdentityRef", err)
	}
}

func TestIdentityReadsOnlyTheNamedVariable(t *testing.T) {
	t.Setenv("OTHER_EMAIL", mailSentinel)
	ref, err := domain.NewIdentityRef("NAMED_EMAIL")
	if err != nil {
		t.Fatalf("NewIdentityRef: %v", err)
	}
	_, err = newCredentials(t, newRoot(t, "")).Identity(ref)
	var missing domain.MissingKeyError
	if !errors.As(err, &missing) || missing.Variable != "NAMED_EMAIL" {
		t.Fatalf("Identity error = %v, want the named variable reported missing", err)
	}
}

func TestIdentityRefusesAnEnvFileThatIsASymlinkOutsideTheRoot(t *testing.T) {
	root := newRoot(t, "")
	outside := filepath.Join(t.TempDir(), "outside.env")
	mustWrite(t, outside, jiraEmailVariable+"="+mailSentinel+"\n")
	mustSymlink(t, outside, filepath.Join(root, envFileName))
	unsetEnv(t, jiraEmailVariable)

	_, err := newCredentials(t, root).Identity(jiraEmailRef(t))
	if err == nil || missingIdentity(err) {
		t.Fatalf("Identity error = %v, want a refusal of the symlinked file", err)
	}
	if strings.Contains(err.Error(), mailSentinel) {
		t.Fatalf("refusal text leaks the value: %q", err.Error())
	}
}

// TestIdentityErrorsNeverShowTheValue proves that a malformed file holding the address fails without echoing it.
func TestIdentityErrorsNeverShowTheValue(t *testing.T) {
	root := newRoot(t, jiraEmailVariable+"="+mailSentinel+"\nbroken "+mailSentinel+"\n")
	unsetEnv(t, jiraEmailVariable)

	_, err := newCredentials(t, root).Identity(jiraEmailRef(t))
	if err == nil {
		t.Fatal("Identity accepted a malformed file, want an error")
	}
	for _, text := range []string{err.Error(), fmt.Sprintf("%v", err), fmt.Sprintf("%+v", err)} {
		if strings.Contains(text, mailSentinel) {
			t.Fatalf("error text leaks the value: %q", text)
		}
	}
}

// TestIdentityFormattingNeverShowsTheAddress proves the address stays redacted in every formatting path.
func TestIdentityFormattingNeverShowsTheAddress(t *testing.T) {
	root := newRoot(t, "")
	t.Setenv(jiraEmailVariable, mailSentinel)
	id, err := newCredentials(t, root).Identity(jiraEmailRef(t))
	if err != nil {
		t.Fatalf("Identity: %v", err)
	}
	for _, text := range []string{
		fmt.Sprintf("%v", id),
		fmt.Sprintf("%+v", id),
		fmt.Sprintf("%#v", id),
		fmt.Sprintf("%s", id),
	} {
		if strings.Contains(text, mailSentinel) {
			t.Fatalf("formatted address leaks the value: %q", text)
		}
	}
}

// TestKeyAndIdentityReadTheirOwnVariables proves that the two lookups are independent: each returns its own value from the
// same file, and neither returns the other's.
func TestKeyAndIdentityReadTheirOwnVariables(t *testing.T) {
	root := newRoot(t, jiraKeyVariable+"="+jiraKeySentinel+"\n"+jiraEmailVariable+"="+mailSentinel+"\n")
	unsetEnv(t, jiraKeyVariable)
	unsetEnv(t, jiraEmailVariable)
	c := newCredentials(t, root)

	key, err := c.APIKey(jiraKeyRef(t))
	if err != nil {
		t.Fatalf("APIKey(jira): %v", err)
	}
	if key.Reveal() != jiraKeySentinel {
		t.Fatalf("key = %q, want its own file value", key.Reveal())
	}
	if got := identityValue(t, c); got != mailSentinel {
		t.Fatalf("identity = %q, want its own file value", got)
	}
}

// TestIdentityIsMissingWhenOnlyTheKeyIsSet proves that a set key does not satisfy the identity lookup.
func TestIdentityIsMissingWhenOnlyTheKeyIsSet(t *testing.T) {
	root := newRoot(t, jiraKeyVariable+"="+jiraKeySentinel+"\n")
	unsetEnv(t, jiraEmailVariable)
	if _, err := newCredentials(t, root).Identity(jiraEmailRef(t)); !missingIdentity(err) {
		t.Fatalf("Identity error = %v, want a MissingKeyError", err)
	}
}
