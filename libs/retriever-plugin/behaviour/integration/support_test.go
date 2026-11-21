//go:build integration

package integration_test

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	plugin "github.com/wahidyankf/super-agent/libs/retriever-plugin"
)

// helperPath is the fixture plugin: one program whose behaviour the request method selects. TestMain builds it once.
var helperPath string

// TestMain builds the fixture plugin into a folder of its own, runs the tests, and removes the folder.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "retriever-plugin-fixture-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "make the fixture folder: %v\n", err)
		os.Exit(1)
	}
	helperPath = filepath.Join(dir, "plugintest")
	out, err := exec.Command("go", "build", "-o", helperPath, "../../testdata/plugin-helper.go").CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "build the fixture plugin: %v\n%s", err, out)
		_ = os.RemoveAll(dir)
		os.Exit(1)
	}
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

// realFiles is the production plugin.Files over the real disk. It records every private folder it makes, and afterCopy
// runs with the path of the original after each copy, so a test can change the original before the check.
type realFiles struct {
	afterCopy func(original string)
	made      []string
}

func (f *realFiles) MakePrivateDir() (string, error) {
	dir, err := os.MkdirTemp("", "retriever-plugin-")
	if err == nil {
		f.made = append(f.made, dir)
	}
	return dir, err
}

func (f *realFiles) CopyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.WriteFile(dst, data, 0o700); err != nil {
		return err
	}
	if f.afterCopy != nil {
		f.afterCopy(src)
	}
	return nil
}

func (f *realFiles) HashFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func (f *realFiles) RemoveAll(path string) error { return os.RemoveAll(path) }

// fileSHA returns the lower-case hex SHA-256 of the file at path.
func fileSHA(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// checkedRef returns the reference to the plugin at path, pinned by its own bytes.
func checkedRef(t *testing.T, path string) domain.PluginRef {
	t.Helper()
	ref, err := domain.NewPluginRef(path, fileSHA(t, path))
	if err != nil {
		t.Fatalf("NewPluginRef: %v", err)
	}
	return ref
}

// writeScript writes an executable script named name in dir and returns its path.
func writeScript(t *testing.T, dir, name, body string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0o700); err != nil {
		t.Fatalf("write the script %s: %v", name, err)
	}
	return path
}

// helperConfig returns the call configuration of the fixture plugin, with the credential sentinel and no identity.
// The sentinel is not a real secret: it only has to show in the redacted output.
func helperConfig(t *testing.T, timeout time.Duration) plugin.Config {
	t.Helper()
	return plugin.Config{
		Plugin:          checkedRef(t, helperPath),
		CredentialEnv:   "EXAMPLE_WIKI_API_KEY",
		CredentialValue: "credential-sentinel-9f2a",
		Timeout:         timeout,
	}
}

// fixedCredentials is a ports.Credentials and a ports.Identities that answers from memory, counting each read.
type fixedCredentials struct {
	apiReads int
	apiErr   error
	identErr error
}

func (c *fixedCredentials) APIKey(ref domain.CredentialRef) (domain.ApiKey, error) {
	c.apiReads++
	if c.apiErr != nil {
		return domain.ApiKey{}, c.apiErr
	}
	return domain.NewApiKey(ref.Name(), "credential-sentinel-9f2a")
}

func (c *fixedCredentials) Identity(ref domain.IdentityRef) (domain.Identity, error) {
	if c.identErr != nil {
		return domain.Identity{}, c.identErr
	}
	return domain.NewIdentity(ref.Name(), "name@example.test")
}

// wikiSource returns the outline source "docs" served by the plugin at path, pinned by its bytes, with the identity name
// when withIdentity is set.
func wikiSource(t *testing.T, path string, withIdentity bool) domain.Source {
	t.Helper()
	return wikiSourceWithRef(t, checkedRef(t, path), withIdentity)
}

// wikiSourceWithRef returns the outline source "docs" served by the plugin ref, with the identity name when set.
func wikiSourceWithRef(t *testing.T, ref domain.PluginRef, withIdentity bool) domain.Source {
	t.Helper()
	id, err := domain.NewSourceID("docs")
	if err != nil {
		t.Fatalf("NewSourceID: %v", err)
	}
	endpoint, err := domain.NewEndpoint("https://docs.example.test")
	if err != nil {
		t.Fatalf("NewEndpoint: %v", err)
	}
	credential, err := domain.NewCredentialRef("EXAMPLE_WIKI_API_KEY")
	if err != nil {
		t.Fatalf("NewCredentialRef: %v", err)
	}
	tags, err := domain.NewTagSet()
	if err != nil {
		t.Fatalf("NewTagSet: %v", err)
	}
	src := domain.NewSource(id, domain.KindOutline, tags, endpoint, credential).WithPlugin(ref)
	if withIdentity {
		identity, err := domain.NewIdentityRef("EXAMPLE_WIKI_EMAIL")
		if err != nil {
			t.Fatalf("NewIdentityRef: %v", err)
		}
		src = src.WithIdentity(identity)
	}
	return src
}

// gatewayFor returns the gateway of the outline source "docs" served by the plugin at path. Its credentials are creds,
// and its identity is resolved from the same double when withIdentity is set.
func gatewayFor(t *testing.T, path string, withIdentity bool, creds *fixedCredentials) *plugin.Gateway {
	t.Helper()
	return &plugin.Gateway{
		Source:      wikiSource(t, path, withIdentity),
		Credentials: creds,
		Identities:  creds,
		Timeout:     10 * time.Second,
	}
}

// processGone reports whether the process pid no longer runs. A process that has died but not been reaped counts as
// gone, because it holds no resources and runs no code.
func processGone(pid int) bool {
	if err := syscall.Kill(pid, 0); errors.Is(err, syscall.ESRCH) {
		return true
	}
	stat, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return true
	}
	// The state is the field after the command name, which sits in parentheses.
	fields := strings.Fields(string(stat[strings.LastIndex(string(stat), ")")+1:]))
	return len(fields) > 0 && fields[0] == "Z"
}

// waitUntil polls cond until it holds or the limit runs out, and reports whether it held. It is a bounded readiness
// check, not a fixed sleep.
func waitUntil(limit time.Duration, cond func() bool) bool {
	deadline := time.Now().Add(limit)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return cond()
}

// readPID returns the process ID written to path by a script, or 0 when the file does not hold one yet.
func readPID(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	var pid int
	if _, err := fmt.Sscan(string(data), &pid); err != nil {
		return 0
	}
	return pid
}
