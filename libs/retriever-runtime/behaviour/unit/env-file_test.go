package unit_test

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/envfile"
)

// openLog is an in-memory folder that records each name it is asked to open. A binding uses it to show that a file was
// never read, not only that its value was not used.
type openLog struct {
	fs.FS
	opened *[]string
}

// Open records the name and opens it in the folder it wraps.
func (o openLog) Open(name string) (fs.File, error) {
	*o.opened = append(*o.opened, name)
	return o.FS.Open(name)
}

// credentials returns the env-file reader over the process variables and the env file content given. The env file is
// named .env.local in the working directory /work. Empty content leaves the file absent.
func credentials(t *testing.T, process map[string]string, file string) *envfile.Credentials {
	t.Helper()
	folder := fstest.MapFS{}
	if file != "" {
		folder = fileSet(map[string]string{".env.local": file})
	}
	c, err := envfile.NewWithFiles("/work/.env.local", lookup(process), envfile.Files{FS: folder, SameFile: sameByName})
	if err != nil {
		t.Fatalf("NewWithFiles: %v", err)
	}
	return c
}

// credentialRef returns the reference to the credential variable name.
func credentialRef(t *testing.T, name string) domain.CredentialRef {
	t.Helper()
	ref, err := domain.NewCredentialRef(name)
	if err != nil {
		t.Fatalf("NewCredentialRef(%q): %v", name, err)
	}
	return ref
}

// @scenario specs/libs/retriever-runtime/behaviours/env-file.feature: The process value wins and the env file is not read
func TestTheProcessValueWinsAndTheEnvFileIsNotRead(t *testing.T) {
	var opened []string
	folder := openLog{FS: fileSet(map[string]string{".env.local": "EXAMPLE_OUTLINE_API_KEY=EXAMPLE-FILE-KEY\n"}), opened: &opened}
	c, err := envfile.NewWithFiles("/work/.env.local", lookup(map[string]string{"EXAMPLE_OUTLINE_API_KEY": "EXAMPLE-PROCESS-KEY"}),
		envfile.Files{FS: folder, SameFile: sameByName})
	if err != nil {
		t.Fatalf("NewWithFiles: %v", err)
	}
	key, err := c.APIKey(credentialRef(t, "EXAMPLE_OUTLINE_API_KEY"))
	if err != nil {
		t.Fatalf("APIKey error = %v", err)
	}
	if key.Reveal() != "EXAMPLE-PROCESS-KEY" {
		t.Fatalf("key = %q, want the process value", key.Reveal())
	}
	if len(opened) != 0 {
		t.Fatalf("env file opened %v, want it never read", opened)
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/env-file.feature: The env file value is read when the process has none
func TestTheEnvFileValueIsReadWhenTheProcessHasNone(t *testing.T) {
	c := credentials(t, nil, "EXAMPLE_OUTLINE_API_KEY=EXAMPLE-FILE-KEY\n")
	key, err := c.APIKey(credentialRef(t, "EXAMPLE_OUTLINE_API_KEY"))
	if err != nil {
		t.Fatalf("APIKey error = %v", err)
	}
	if key.Reveal() != "EXAMPLE-FILE-KEY" {
		t.Fatalf("key = %q, want the env file value", key.Reveal())
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/env-file.feature: An empty process value counts as missing, so the file value is read
func TestAnEmptyProcessValueCountsAsMissingSoTheFileValueIsRead(t *testing.T) {
	c := credentials(t, map[string]string{"EXAMPLE_OUTLINE_API_KEY": ""}, "EXAMPLE_OUTLINE_API_KEY=EXAMPLE-FILE-KEY\n")
	key, err := c.APIKey(credentialRef(t, "EXAMPLE_OUTLINE_API_KEY"))
	if err != nil {
		t.Fatalf("APIKey error = %v", err)
	}
	if key.Reveal() != "EXAMPLE-FILE-KEY" {
		t.Fatalf("key = %q, want the env file value", key.Reveal())
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/env-file.feature: A key missing from both places names its variable in the error
func TestAKeyMissingFromBothPlacesNamesItsVariableInTheError(t *testing.T) {
	c := credentials(t, nil, "EXAMPLE_OUTLINE_API_KEY=\n")
	key, err := c.APIKey(credentialRef(t, "EXAMPLE_OUTLINE_API_KEY"))
	wantError(t, err, "EXAMPLE_OUTLINE_API_KEY is not set")
	if key.Reveal() != "" {
		t.Fatalf("key = %q, want no key", key.Reveal())
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/env-file.feature: A key whose env file is absent is a missing key
func TestAKeyWhoseEnvFileIsAbsentIsAMissingKey(t *testing.T) {
	c := credentials(t, nil, "")
	_, err := c.APIKey(credentialRef(t, "EXAMPLE_OUTLINE_API_KEY"))
	wantError(t, err, "EXAMPLE_OUTLINE_API_KEY is not set")
}

// @scenario specs/libs/retriever-runtime/behaviours/env-file.feature: The zero credential reference is refused before any read
func TestTheZeroCredentialReferenceIsRefusedBeforeAnyRead(t *testing.T) {
	var reads []string
	c, err := envfile.NewWithFiles("/work/.env.local", countingLookup(nil, &reads), envfile.Files{FS: fstest.MapFS{}, SameFile: sameByName})
	if err != nil {
		t.Fatalf("NewWithFiles: %v", err)
	}
	var zero domain.CredentialRef
	if _, err := c.APIKey(zero); err == nil {
		t.Fatalf("APIKey(zero) error = nil, want a refusal")
	}
	if len(reads) != 0 {
		t.Fatalf("the environment was read for %v, want no read", reads)
	}
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
			c := credentials(t, nil, tc.line+"\n")
			key, err := c.APIKey(credentialRef(t, "EXAMPLE_OUTLINE_API_KEY"))
			if err != nil {
				t.Fatalf("APIKey error = %v", err)
			}
			if key.Reveal() != tc.value {
				t.Fatalf("key = %q, want %q", key.Reveal(), tc.value)
			}
		})
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/env-file.feature: A repeated variable in the env file takes its last value
func TestARepeatedVariableInTheEnvFileTakesItsLastValue(t *testing.T) {
	c := credentials(t, nil, "EXAMPLE_OUTLINE_API_KEY=FIRST\nEXAMPLE_OUTLINE_API_KEY=LAST\n")
	key, err := c.APIKey(credentialRef(t, "EXAMPLE_OUTLINE_API_KEY"))
	if err != nil {
		t.Fatalf("APIKey error = %v", err)
	}
	if key.Reveal() != "LAST" {
		t.Fatalf("key = %q, want LAST", key.Reveal())
	}
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
			c := credentials(t, nil, "EXAMPLE_OUTLINE_API_KEY=FIRST\n"+tc.line+"\n")
			_, err := c.APIKey(credentialRef(t, "EXAMPLE_OUTLINE_API_KEY"))
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
		name string
		mode fs.FileMode
	}{
		{"symbolic link", fs.ModeSymlink},
		{"folder", fs.ModeDir},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			folder := fstest.MapFS{".env.local": &fstest.MapFile{Mode: tc.mode}}
			c, err := envfile.NewWithFiles("/work/.env.local", lookup(nil), envfile.Files{FS: folder, SameFile: sameByName})
			if err != nil {
				t.Fatalf("NewWithFiles: %v", err)
			}
			key, err := c.APIKey(credentialRef(t, "EXAMPLE_OUTLINE_API_KEY"))
			wantError(t, err, "local env file must be a regular file, not a symlink or a folder")
			if key.Reveal() != "" {
				t.Fatalf("key = %q, want no key", key.Reveal())
			}
		})
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/env-file.feature: An account address held in the env file is read when the process has none
func TestAnAccountAddressHeldInTheEnvFileIsReadWhenTheProcessHasNone(t *testing.T) {
	c := credentials(t, nil, "EXAMPLE_JIRA_EMAIL=name@example.test\n")
	ref, err := domain.NewIdentityRef("EXAMPLE_JIRA_EMAIL")
	if err != nil {
		t.Fatalf("NewIdentityRef: %v", err)
	}
	identity, err := c.Identity(ref)
	if err != nil {
		t.Fatalf("Identity error = %v, want the built identity", err)
	}
	if identity.Reveal() != "name@example.test" {
		t.Fatalf("identity = %q, want the address from the env file", identity.Reveal())
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/env-file.feature: A malformed account address is refused, naming the variable only
func TestAMalformedAccountAddressIsRefusedNamingTheVariableOnly(t *testing.T) {
	c := credentials(t, nil, "EXAMPLE_JIRA_EMAIL=EXAMPLE-NOT-AN-ADDRESS\n")
	ref, err := domain.NewIdentityRef("EXAMPLE_JIRA_EMAIL")
	if err != nil {
		t.Fatalf("NewIdentityRef: %v", err)
	}
	_, err = c.Identity(ref)
	wantError(t, err, "EXAMPLE_JIRA_EMAIL is not a valid e-mail address")
	if strings.Contains(err.Error(), "EXAMPLE-NOT-AN-ADDRESS") {
		t.Fatalf("error %q holds the address", err.Error())
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/env-file.feature: A key or account address shows as redacted in every format
func TestAKeyOrAccountAddressShowsAsRedactedInEveryFormat(t *testing.T) {
	c := credentials(t, map[string]string{
		"EXAMPLE_OUTLINE_API_KEY": "EXAMPLE-KEY",
		"EXAMPLE_JIRA_EMAIL":      "name@example.test",
	}, "")
	key, err := c.APIKey(credentialRef(t, "EXAMPLE_OUTLINE_API_KEY"))
	if err != nil {
		t.Fatalf("APIKey error = %v", err)
	}
	ref, err := domain.NewIdentityRef("EXAMPLE_JIRA_EMAIL")
	if err != nil {
		t.Fatalf("NewIdentityRef: %v", err)
	}
	identity, err := c.Identity(ref)
	if err != nil {
		t.Fatalf("Identity error = %v", err)
	}
	encoded, err := json.Marshal(map[string]any{"key": key, "identity": identity})
	if err != nil {
		t.Fatalf("json.Marshal error = %v", err)
	}
	texts := []string{
		fmt.Sprintf("%v %s %+v %#v", key, key, key, key),
		fmt.Sprintf("%v %s %+v %#v", identity, identity, identity, identity),
		string(encoded),
	}
	for _, text := range texts {
		for _, value := range []string{"EXAMPLE-KEY", "name@example.test"} {
			if strings.Contains(text, value) {
				t.Fatalf("formatted text %q holds %q", text, value)
			}
		}
	}
}
