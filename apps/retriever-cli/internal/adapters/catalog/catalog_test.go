package catalog

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

const catalogPrefix = "sources catalog: "

// sentinel is a value put where a rule is broken. No error text may repeat it.
const sentinel = "pasted-key-4c1e9a"

// oneSource is a valid catalog with one source. The format verbs take the field line to replace, if any.
const oneSource = `version: 1
sources:
  - id: outline
    kind: outline
    endpoint: https://docs.example.test
    credential_env: EXAMPLE_OUTLINE_API_KEY
`

// newFile returns the path of a catalog file, named sources.yml in a fresh folder, written as content. An empty content
// leaves the file absent.
func newFile(t *testing.T, content string) string {
	t.Helper()
	file := filepath.Join(t.TempDir(), "sources.yml")
	if content != "" {
		if err := os.WriteFile(file, []byte(content), 0o600); err != nil {
			t.Fatalf("write catalog: %v", err)
		}
	}
	return file
}

func loadFrom(t *testing.T, file string) (domain.Catalog, error) {
	t.Helper()
	l, err := New(file)
	if err != nil {
		t.Fatalf("New(%q): %v", file, err)
	}
	return l.Load()
}

func mustLoad(t *testing.T, content string) domain.Catalog {
	t.Helper()
	cat, err := loadFrom(t, newFile(t, content))
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	return cat
}

// mustFailWith requires the load to fail with exactly the rule text after the safe prefix, and never echo the sentinel.
func mustFailWith(t *testing.T, content, rule string) {
	t.Helper()
	_, err := loadFrom(t, newFile(t, content))
	if err == nil {
		t.Fatalf("Load() error = nil, want %q", catalogPrefix+rule)
	}
	if got := err.Error(); got != catalogPrefix+rule {
		t.Fatalf("Load() error = %q, want %q", got, catalogPrefix+rule)
	}
	if strings.Contains(err.Error(), sentinel) {
		t.Fatalf("Load() error echoes a catalog value: %q", err.Error())
	}
	var ce domain.CatalogError
	if !errors.As(err, &ce) {
		t.Fatalf("Load() error = %T, want domain.CatalogError", err)
	}
}

// withField returns oneSource with the line of field replaced by value.
func withField(field, value string) string {
	lines := strings.Split(oneSource, "\n")
	for i, line := range lines {
		trimmed := strings.TrimLeft(line, " -")
		if strings.HasPrefix(trimmed, field+":") {
			indent := line[:len(line)-len(trimmed)]
			lines[i] = indent + field + ": " + value
		}
	}
	return strings.Join(lines, "\n")
}

func TestAbsentFileIsAConfigurationError(t *testing.T) {
	_, err := loadFrom(t, newFile(t, ""))
	var cat domain.CatalogError
	if !errors.As(err, &cat) || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("absent catalog error = %v; want a CatalogError that says the file is not found", err)
	}
}

func TestPresentFileIsTheWholeCatalog(t *testing.T) {
	content := `version: 1
sources:
  - id: wiki
    kind: outline
    endpoint: https://wiki.example.com
    credential_env: WIKI_API_KEY
`
	cat := mustLoad(t, content)
	all := cat.All()
	if len(all) != 1 || all[0].ID().String() != "wiki" {
		t.Fatalf("catalog = %v, want only wiki (the file replaces the built-in)", all)
	}
}

func TestAccessAbsentOrNullIsRead(t *testing.T) {
	for _, content := range []string{oneSource, withAccess(""), withAccess("null")} {
		cat := mustLoad(t, content)
		if got := cat.All()[0].Access(); got != domain.AccessRead {
			t.Fatalf("access = %q, want read", got)
		}
	}
}

// withAccess adds an access line to oneSource. An empty value writes the bare key, which is YAML null.
func withAccess(value string) string {
	line := "    access:"
	if value != "" {
		line += " " + value
	}
	return strings.Replace(oneSource, "    endpoint:", line+"\n    endpoint:", 1)
}

func TestEachAccessWordIsKept(t *testing.T) {
	for _, word := range []string{"read", "write", "admin"} {
		cat := mustLoad(t, withAccess(word))
		if got := cat.All()[0].Access().String(); got != word {
			t.Fatalf("access = %q, want %q", got, word)
		}
	}
}

func TestInvalidAccessIsRefusedWithItsRule(t *testing.T) {
	for _, word := range []string{`""`, "Read", "rw", "owner", sentinel} {
		mustFailWith(t, withAccess(word), "source 1 access: must be read, write or admin")
	}
}

func TestSentinelInCredentialEnvIsNeverPrinted(t *testing.T) {
	mustFailWith(t, withField("credential_env", sentinel), "source 1 credential_env: must be an upper-case *_API_KEY name")
}

func TestEveryFieldRuleNamesTheSourceFieldAndRule(t *testing.T) {
	cases := []struct {
		name, content, want string
	}{
		{name: "bad id", content: withField("id", "Outline"), want: "source 1 id: must be a lowercase slug"},
		{name: "unknown kind", content: withField("kind", "notion"), want: "source 1 kind: unknown kind (known: outline, kiwi, jira)"},
		{name: "bad tag", content: strings.Replace(oneSource, "    endpoint:", "    tags: [Wiki]\n    endpoint:", 1), want: "source 1 tags: must be lowercase slugs"},
		{name: "too many tags", content: strings.Replace(oneSource, "    endpoint:", "    tags: [a1,a2,a3,a4,a5,a6,a7,a8,a9,b1,b2,b3,b4,b5,b6,b7,b8]\n    endpoint:", 1), want: "source 1 tags: at most 16 distinct tags"},
		{name: "bad endpoint", content: withField("endpoint", "http://docs.example.test"), want: "source 1 endpoint: must be an https host name"},
		{name: "a credential that is not an API key name", content: withField("credential_env", "GITHUB_WORK_TOKEN"), want: "source 1 credential_env: must be an upper-case *_API_KEY name"},
		{name: "a missing endpoint", content: strings.Replace(oneSource, "    endpoint: https://docs.example.test\n", "", 1), want: "source 1 endpoint: must be an https host name"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { mustFailWith(t, tc.content, tc.want) })
	}
}

func TestSourcesInvariantsAreReportedWithPosition(t *testing.T) {
	dup := `version: 1
sources:
  - id: outline
    kind: outline
    endpoint: https://docs.example.test
    credential_env: EXAMPLE_OUTLINE_API_KEY
  - id: outline
    kind: outline
    endpoint: https://wiki.engineering.example.com
    credential_env: ENGINEERING_OUTLINE_API_KEY
`
	mustFailWith(t, dup, "source 2 id: duplicate")

	shared := strings.Replace(dup, "ENGINEERING_OUTLINE_API_KEY", "EXAMPLE_OUTLINE_API_KEY", 1)
	shared = strings.Replace(shared, "id: outline\n    kind: outline\n    endpoint: https://wiki", "id: outline-engineering\n    kind: outline\n    endpoint: https://wiki", 1)
	mustFailWith(t, shared, "source 2 credential_env: one credential serves one host only")
}

func TestSourcesListWithNoSourceIsRefused(t *testing.T) {
	mustFailWith(t, "version: 1\nsources: []\n", "sources: the catalog needs at least one source")
}

// notValidYAML is the rule for a file the strict decoder refuses: bad YAML, an unknown field, or a repeated key.
const notValidYAML = "the file is not valid YAML, or it has an unknown or repeated field"

func TestStructuralFailuresAreRefusedWithTheirExactRule(t *testing.T) {
	// A repeated key inside a valid source: the file is otherwise a complete catalog, so only the key breaks it.
	repeatedEndpoint := strings.Replace(oneSource, "    endpoint: https://docs.example.test\n",
		"    endpoint: https://docs.example.test\n    endpoint: https://docs.example.test\n", 1)
	cases := []struct {
		name, content, rule string
	}{
		{name: "an unknown field", content: oneSource + "    colour: blue\n", rule: notValidYAML},
		{name: "a repeated key inside a source", content: repeatedEndpoint, rule: notValidYAML},
		{name: "an unsupported version", content: strings.Replace(oneSource, "version: 1", "version: 2", 1), rule: "version: must be 1"},
		{name: "no version", content: strings.Replace(oneSource, "version: 1\n", "", 1), rule: "version: must be 1"},
		{name: "invalid YAML", content: "version: [1\n", rule: notValidYAML},
		{name: "a file with no document", content: "\n", rule: notValidYAML},
		{name: "two documents", content: oneSource + "---\nversion: 1\n", rule: "the file must hold one YAML document"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { mustFailWith(t, tc.content, tc.rule) })
	}
}

func TestOversizeFileIsRefused(t *testing.T) {
	big := oneSource + "# " + strings.Repeat("x", 64<<10) + "\n"
	_, err := loadFrom(t, newFile(t, big))
	if err == nil || !strings.Contains(err.Error(), "must be at most 64 KiB") {
		t.Fatalf("oversize error = %v", err)
	}
}

func TestFileAtExactlyTheLimitIsAccepted(t *testing.T) {
	fill := strings.Repeat("x", (64<<10)-len(oneSource)-len("# \n"))
	content := oneSource + "# " + fill + "\n"
	if len(content) != 64<<10 {
		t.Fatalf("test fixture is %d bytes", len(content))
	}
	if _, err := loadFrom(t, newFile(t, content)); err != nil {
		t.Fatalf("file of exactly 64 KiB error = %v, want nil", err)
	}
}

func TestSymlinkAndFolderAreRefused(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "elsewhere.yml")
	if err := os.WriteFile(target, []byte(oneSource), 0o600); err != nil {
		t.Fatalf("write target: %v", err)
	}
	link := filepath.Join(dir, "sources.yml")
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	if _, err := loadFrom(t, link); err == nil || !strings.HasPrefix(err.Error(), catalogPrefix) {
		t.Fatalf("symlink error = %v", err)
	}

	folder := filepath.Join(t.TempDir(), "sources.yml")
	if err := os.Mkdir(folder, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if _, err := loadFrom(t, folder); err == nil || !strings.Contains(err.Error(), "must be a regular file") {
		t.Fatalf("folder error = %v", err)
	}
}

func TestNewRequiresAnAbsolutePath(t *testing.T) {
	if _, err := New("relative/sources.yml"); err == nil {
		t.Fatalf("New accepted a relative path")
	}
}

func TestUnreadableCatalogIsAnError(t *testing.T) {
	file := newFile(t, oneSource)
	if err := os.Chmod(file, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(file, 0o600) })
	if _, err := loadFrom(t, file); err == nil || !strings.HasPrefix(err.Error(), catalogPrefix) {
		t.Fatalf("unreadable catalog error = %v", err)
	}
}

func TestCommittedShapeHoldsTwoSources(t *testing.T) {
	content := `version: 1
sources:
  - id: outline
    kind: outline
    tags: [company, wiki]
    endpoint: https://docs.example.test
    credential_env: EXAMPLE_OUTLINE_API_KEY
  - id: outline-engineering
    kind: outline
    tags: [engineering, wiki]
    access: write
    endpoint: https://wiki.engineering.example.com
    credential_env: ENGINEERING_OUTLINE_API_KEY
`
	cat := mustLoad(t, content)
	all := cat.All()
	if len(all) != 2 || all[1].Access() != domain.AccessWrite {
		t.Fatalf("catalog = %v", all)
	}
	if got := strings.Join(all[0].Tags().Values(), ","); got != "company,wiki" {
		t.Fatalf("tags = %q", got)
	}
}

// oneJira is a valid catalog with one Jira source, which needs an identity name.
const oneJira = `version: 1
sources:
  - id: jira
    kind: jira
    access: read
    endpoint: https://jira.example.test
    credential_env: EXAMPLE_JIRA_API_TOKEN
    identity_env: EXAMPLE_JIRA_EMAIL
`

// withLine returns content with line inserted before the first credential_env line, which every test source has.
func withLine(content, line string) string {
	return strings.Replace(content, "    credential_env:", line+"\n    credential_env:", 1)
}

// withoutLine returns content with the line that names field removed.
func withoutLine(content, field string) string {
	lines := strings.Split(content, "\n")
	out := lines[:0]
	for _, line := range lines {
		if !strings.HasPrefix(strings.TrimLeft(line, " -"), field+":") {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}

// TestIdentityEnvIsReadAndCheckedWithItsRule checks the optional identity field: a jira source carries its e-mail name,
// and each kind rule is reported with its position, the field, and the rule, and never a value.
func TestIdentityEnvIsReadAndCheckedWithItsRule(t *testing.T) {
	cat := mustLoad(t, oneJira)
	src := cat.All()[0]
	if src.Kind() != domain.KindJira || src.Identity().Name() != "EXAMPLE_JIRA_EMAIL" {
		t.Fatalf("jira source = kind %v, identity %q", src.Kind(), src.Identity().Name())
	}

	cases := []struct {
		name, content, want string
	}{
		{name: "an identity that is not an upper-case e-mail name", content: strings.Replace(oneJira, "identity_env: EXAMPLE_JIRA_EMAIL", "identity_env: "+sentinel, 1), want: "source 1 identity_env: must be an upper-case *_EMAIL name"},
		{name: "an identity name that is too short", content: withLine(oneSource, "    identity_env: EMAIL"), want: "source 1 identity_env: must be an upper-case *_EMAIL name"},
		{name: "a jira source with no identity", content: withoutLine(oneJira, "identity_env"), want: "source 1 identity_env: required for a jira source"},
		{name: "an outline source with an identity", content: withLine(oneSource, "    identity_env: EXAMPLE_OUTLINE_EMAIL"), want: "source 1 identity_env: not used by an outline source"},
		{name: "a kiwi source with an identity", content: withLine(strings.Replace(oneSource, "kind: outline", "kind: kiwi", 1), "    identity_env: EXAMPLE_KIWI_EMAIL"), want: "source 1 identity_env: not used by a kiwi source"},
		{name: "a jira source with write access", content: strings.Replace(oneJira, "access: read", "access: write", 1), want: "source 1 access: a jira source allows read at most"},
		{name: "a jira source with admin access", content: strings.Replace(oneJira, "access: read", "access: admin", 1), want: "source 1 access: a jira source allows read at most"},
		{name: "one identity on two hosts", content: oneJira + `  - id: jira-two
    kind: jira
    access: read
    endpoint: https://other.example.test
    credential_env: OTHER_JIRA_API_KEY
    identity_env: EXAMPLE_JIRA_EMAIL
`, want: "source 2 identity_env: one identity serves one host only"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { mustFailWith(t, tc.content, tc.want) })
	}
}

// TestIdentityOnOneHostIsAccepted checks that the same identity name may serve two sources on one host.
func TestIdentityOnOneHostIsAccepted(t *testing.T) {
	content := oneJira + `  - id: jira-two
    kind: jira
    access: read
    endpoint: https://jira.example.test
    credential_env: OTHER_JIRA_API_KEY
    identity_env: EXAMPLE_JIRA_EMAIL
`
	if got := len(mustLoad(t, content).All()); got != 2 {
		t.Fatalf("catalog holds %d sources, want 2", got)
	}
}
