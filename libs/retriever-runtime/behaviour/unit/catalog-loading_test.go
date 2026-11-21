package unit_test

import (
	"errors"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/catalog"
)

// outlineDocsYAML is a valid catalog with one outline source named docs.
const outlineDocsYAML = `version: 1
sources:
  - id: docs
    kind: outline
    endpoint: https://docs.example.test
    credential_env: EXAMPLE_OUTLINE_API_KEY
`

// kindsWithAcme returns the built-in kinds plus one spec kind, acme, which a loaded spec file registers.
func kindsWithAcme(t *testing.T) domain.KindSet {
	t.Helper()
	kinds, err := domain.NewKindSet(
		domain.KindDefinition{Name: "outline", Display: "Outline"},
		domain.KindDefinition{Name: "kiwi", Display: "Kiwi"},
		domain.KindDefinition{Name: "jira", Display: "Jira", UsesIdentity: true},
		domain.KindDefinition{Name: "acme"},
	)
	if err != nil {
		t.Fatalf("NewKindSet: %v", err)
	}
	return kinds
}

// loadCatalog loads a catalog file with the given content from an in-memory folder, with the kinds and spec digests given.
// Empty content leaves the file absent.
func loadCatalog(t *testing.T, content string, kinds domain.KindSet, digests map[string]string) (domain.Catalog, error) {
	t.Helper()
	folder := fstest.MapFS{}
	if content != "" {
		folder = fileSet(map[string]string{"sources.yml": content})
	}
	l, err := catalog.NewWithFiles("/work/sources.yml", catalog.Files{FS: folder, SameFile: sameByName}, kinds)
	if err != nil {
		t.Fatalf("NewWithFiles: %v", err)
	}
	return l.WithSpecDigests(digests).Load()
}

// wantRule fails the test unless err is non-nil and its text holds the rule.
func wantRule(t *testing.T, err error, rule string) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want a rule holding %q", rule)
	}
	if !strings.Contains(err.Error(), rule) {
		t.Fatalf("error = %q, want it to hold %q", err.Error(), rule)
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/catalog-loading.feature: A catalog file that is absent names where to set the path
func TestACatalogFileThatIsAbsentNamesWhereToSetThePath(t *testing.T) {
	cat, err := loadCatalog(t, "", domain.BuiltinKinds(), nil)
	wantRule(t, err, "not found: set --catalog or RETRIEVER_CATALOG, or create ./sources.yml")
	if len(cat.All()) != 0 {
		t.Fatalf("catalog holds %d sources, want none", len(cat.All()))
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/catalog-loading.feature: A catalog file that is present is the whole catalog
func TestACatalogFileThatIsPresentIsTheWholeCatalog(t *testing.T) {
	content := `version: 1
sources:
  - id: docs
    kind: outline
    endpoint: https://docs.example.test
    credential_env: EXAMPLE_OUTLINE_API_KEY
  - id: wiki
    kind: outline
    endpoint: https://wiki.example.test
    credential_env: EXAMPLE_WIKI_API_KEY
`
	cat, err := loadCatalog(t, content, domain.BuiltinKinds(), nil)
	if err != nil {
		t.Fatalf("Load error = %v, want the catalog", err)
	}
	var ids []string
	for _, src := range cat.All() {
		ids = append(ids, src.ID().String())
	}
	if strings.Join(ids, ",") != "docs,wiki" {
		t.Fatalf("sources = %v, want docs and wiki and no other", ids)
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/catalog-loading.feature: A catalog path that is a symbolic link or a folder is refused
func TestACatalogPathThatIsASymbolicLinkOrAFolderIsRefused(t *testing.T) {
	cases := []struct {
		name string
		mode fs.FileMode
	}{
		{"symbolic link", fs.ModeSymlink},
		{"folder", fs.ModeDir},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			folder := fstest.MapFS{"sources.yml": &fstest.MapFile{Mode: tc.mode}}
			l, err := catalog.NewWithFiles("/work/sources.yml", catalog.Files{FS: folder, SameFile: sameByName}, domain.BuiltinKinds())
			if err != nil {
				t.Fatalf("NewWithFiles: %v", err)
			}
			cat, err := l.Load()
			wantRule(t, err, "must be a regular file, not a symlink or a folder")
			if len(cat.All()) != 0 {
				t.Fatalf("catalog holds sources, want none")
			}
		})
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/catalog-loading.feature: A catalog at the size limit is read and one byte over is refused
func TestACatalogAtTheSizeLimitIsReadAndOneByteOverIsRefused(t *testing.T) {
	cases := []struct {
		name string
		size int
		ok   bool
	}{
		{"65536 bytes", 65536, true},
		{"65537 bytes", 65537, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			content := padToSize(outlineDocsYAML, tc.size)
			cat, err := loadCatalog(t, content, domain.BuiltinKinds(), nil)
			if tc.ok {
				if err != nil || len(cat.All()) != 1 {
					t.Fatalf("Load = %d sources, error %v; want the catalog loaded", len(cat.All()), err)
				}
				return
			}
			wantRule(t, err, "must be at most 64 KiB")
		})
	}
}

// padToSize returns the catalog with a YAML comment that fills it to exactly size bytes. A comment is ignored by the parser,
// so the catalog keeps its one source.
func padToSize(yaml string, size int) string {
	base := yaml + "# "
	return base + strings.Repeat("x", size-len(base))
}

// @scenario specs/libs/retriever-runtime/behaviours/catalog-loading.feature: A catalog that still holds the removed access key is refused
func TestACatalogThatStillHoldsTheRemovedAccessKeyIsRefused(t *testing.T) {
	content := `version: 1
sources:
  - id: docs
    kind: outline
    endpoint: https://docs.example.test
    credential_env: EXAMPLE_OUTLINE_API_KEY
    access: EXAMPLE-SECRET-VALUE
`
	_, err := loadCatalog(t, content, domain.BuiltinKinds(), nil)
	wantRule(t, err, `unknown field "access": the access level was removed; remove the key and limit the source's API token instead`)
	if strings.Contains(err.Error(), "EXAMPLE-SECRET-VALUE") {
		t.Fatalf("error %q holds the value of the removed key", err.Error())
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/catalog-loading.feature: A catalog that is invalid YAML, or has an unknown or repeated field
func TestACatalogThatIsInvalidYAMLOrHasAnUnknownOrRepeatedFieldIsRefused(t *testing.T) {
	const rule = "the file is not valid YAML, or it has an unknown or repeated field"
	cases := []struct {
		name    string
		content string
	}{
		{"is not valid YAML", "version: [1\n"},
		{"holds an unknown field", outlineDocsYAML + "unknown: 1\n"},
		{"holds a repeated field", "version: 1\nversion: 1\nsources: []\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := loadCatalog(t, tc.content, domain.BuiltinKinds(), nil)
			wantRule(t, err, rule)
		})
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/catalog-loading.feature: A catalog whose version is not 1 is refused
func TestACatalogWhoseVersionIsNot1IsRefused(t *testing.T) {
	content := strings.Replace(outlineDocsYAML, "version: 1", "version: 2", 1)
	_, err := loadCatalog(t, content, domain.BuiltinKinds(), nil)
	wantRule(t, err, "version: must be 1")
}

// @scenario specs/libs/retriever-runtime/behaviours/catalog-loading.feature: A catalog file that holds two YAML documents is refused
func TestACatalogFileThatHoldsTwoYAMLDocumentsIsRefused(t *testing.T) {
	_, err := loadCatalog(t, outlineDocsYAML+"---\nversion: 1\n", domain.BuiltinKinds(), nil)
	wantRule(t, err, "the file must hold one YAML document")
}

// @scenario specs/libs/retriever-runtime/behaviours/catalog-loading.feature: A catalog path that is not absolute is refused when the loader is made
func TestACatalogPathThatIsNotAbsoluteIsRefusedWhenTheLoaderIsMade(t *testing.T) {
	l, err := catalog.NewWithFiles("sources.yml", catalog.Files{FS: fstest.MapFS{}, SameFile: sameByName}, domain.BuiltinKinds())
	if err == nil || err.Error() != "the catalog path must be absolute" {
		t.Fatalf("NewWithFiles error = %v, want the absolute-path rule", err)
	}
	if l != nil {
		t.Fatalf("loader = %v, want none", l)
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/catalog-loading.feature: A source of a kind that the kind set does not register is refused
func TestASourceOfAKindThatTheKindSetDoesNotRegisterIsRefused(t *testing.T) {
	kinds, err := domain.NewKindSet(
		domain.KindDefinition{Name: "outline", Display: "Outline"},
		domain.KindDefinition{Name: "kiwi", Display: "Kiwi"},
	)
	if err != nil {
		t.Fatalf("NewKindSet: %v", err)
	}
	content := strings.Replace(outlineDocsYAML, "kind: outline", "kind: github", 1)
	cat, err := loadCatalog(t, content, kinds, nil)
	var unknown domain.UnknownKindError
	if !errors.As(err, &unknown) {
		t.Fatalf("error = %#v, want the unknown source kind error", err)
	}
	if len(cat.All()) != 0 {
		t.Fatalf("catalog holds sources, want none")
	}
}

// specDigest is the digest of the loaded acme spec file in these bindings. A pin is compared with the text, so no real hash
// is needed.
var specDigest = strings.Repeat("a", 64)

// acmeSourceYAML returns a catalog with one acme source that pins the given spec_sha256 value.
func acmeSourceYAML(pin string) string {
	return `version: 1
sources:
  - id: wiki
    kind: acme
    endpoint: https://wiki.example.test
    credential_env: EXAMPLE_ACME_API_KEY
    spec_sha256: ` + pin + `
`
}

// @scenario specs/libs/retriever-runtime/behaviours/catalog-loading.feature: A spec pin that matches the loaded spec file loads the source
func TestASpecPinThatMatchesTheLoadedSpecFileLoadsTheSource(t *testing.T) {
	cat, err := loadCatalog(t, acmeSourceYAML(specDigest), kindsWithAcme(t), map[string]string{"acme": specDigest})
	if err != nil {
		t.Fatalf("Load error = %v, want the source", err)
	}
	if len(cat.All()) != 1 {
		t.Fatalf("catalog holds %d sources, want 1", len(cat.All()))
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/catalog-loading.feature: A spec pin that does not match the loaded spec fails with its code
func TestASpecPinThatDoesNotMatchTheLoadedSpecFailsWithItsCode(t *testing.T) {
	cat, err := loadCatalog(t, acmeSourceYAML(strings.Repeat("b", 64)), kindsWithAcme(t), map[string]string{"acme": specDigest})
	var pin catalog.PinMismatchError
	if !errors.As(err, &pin) {
		t.Fatalf("error = %#v, want the spec pin mismatch", err)
	}
	if pin.Code() != "retriever.connector.spec_pin_mismatch" {
		t.Fatalf("code = %q, want the spec pin mismatch code", pin.Code())
	}
	if len(cat.All()) != 0 {
		t.Fatalf("catalog holds sources, want none")
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/catalog-loading.feature: A spec pin that is not 64 lower-case hex characters is refused
func TestASpecPinThatIsNot64LowerCaseHexCharactersIsRefused(t *testing.T) {
	cases := []struct {
		name string
		pin  string
	}{
		{"upper-case hex", "0123456789ABCDEF0123456789abcdef0123456789abcdef0123456789abcdef"},
		{"too short", "0123456789abcdef"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cat, err := loadCatalog(t, acmeSourceYAML(tc.pin), kindsWithAcme(t), map[string]string{"acme": specDigest})
			wantRule(t, err, "must be 64 lower-case hex characters")
			if len(cat.All()) != 0 {
				t.Fatalf("catalog holds sources, want none")
			}
		})
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/catalog-loading.feature: A spec pin on a built-in kind is refused because it is not a spec kind
func TestASpecPinOnABuiltInKindIsRefusedBecauseItIsNotASpecKind(t *testing.T) {
	content := `version: 1
sources:
  - id: docs
    kind: outline
    endpoint: https://docs.example.test
    credential_env: EXAMPLE_OUTLINE_API_KEY
    spec_sha256: ` + specDigest + `
`
	_, err := loadCatalog(t, content, domain.BuiltinKinds(), map[string]string{"acme": specDigest})
	wantRule(t, err, "only for a spec kind")
}

// @scenario specs/libs/retriever-runtime/behaviours/catalog-loading.feature: A spec kind is pinned only when every source of it pins its spec
func TestASpecKindIsPinnedOnlyWhenEverySourceOfItPinsItsSpec(t *testing.T) {
	content := `version: 1
sources:
  - id: wiki-a
    kind: acme
    endpoint: https://wiki-a.example.test
    credential_env: EXAMPLE_ACME_A_API_KEY
    spec_sha256: ` + specDigest + `
  - id: wiki-b
    kind: acme
    endpoint: https://wiki-b.example.test
    credential_env: EXAMPLE_ACME_B_API_KEY
`
	l, err := catalog.NewWithFiles("/work/sources.yml", catalog.Files{FS: fileSet(map[string]string{"sources.yml": content}), SameFile: sameByName}, kindsWithAcme(t))
	if err != nil {
		t.Fatalf("NewWithFiles: %v", err)
	}
	if _, err := l.WithSpecDigests(map[string]string{"acme": specDigest}).Load(); err != nil {
		t.Fatalf("Load error = %v, want the two sources", err)
	}
	if l.PinnedKinds()["acme"] {
		t.Fatalf("acme is pinned, want it not pinned while one source does not pin")
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/catalog-loading.feature: A spec kind with no source is not pinned
func TestASpecKindWithNoSourceIsNotPinned(t *testing.T) {
	l, err := catalog.NewWithFiles("/work/sources.yml", catalog.Files{FS: fileSet(map[string]string{"sources.yml": outlineDocsYAML}), SameFile: sameByName}, kindsWithAcme(t))
	if err != nil {
		t.Fatalf("NewWithFiles: %v", err)
	}
	if _, err := l.Load(); err != nil {
		t.Fatalf("Load error = %v", err)
	}
	if l.PinnedKinds()["acme"] {
		t.Fatalf("acme is pinned, want it not pinned with no source of it")
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/catalog-loading.feature: A plugin block enables a plugin, and a source without one has none
func TestAPluginBlockEnablesAPluginAndASourceWithoutOneHasNone(t *testing.T) {
	content := `version: 1
sources:
  - id: docs
    kind: outline
    endpoint: https://docs.example.test
    credential_env: EXAMPLE_OUTLINE_API_KEY
    plugin:
      path: /opt/retriever/plugins/example
      sha256: ` + strings.Repeat("c", 64) + `
  - id: wiki
    kind: outline
    endpoint: https://wiki.example.test
    credential_env: EXAMPLE_WIKI_API_KEY
`
	cat, err := loadCatalog(t, content, domain.BuiltinKinds(), nil)
	if err != nil {
		t.Fatalf("Load error = %v", err)
	}
	plugins := map[string]string{}
	for _, src := range cat.All() {
		plugins[src.ID().String()] = src.Plugin().Path()
	}
	if plugins["docs"] != "/opt/retriever/plugins/example" {
		t.Fatalf("docs plugin = %q, want the enabled plugin", plugins["docs"])
	}
	if plugins["wiki"] != "" {
		t.Fatalf("wiki plugin = %q, want none", plugins["wiki"])
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/catalog-loading.feature: A plugin block with a relative path or a misshaped pin refuses the catalog
func TestAPluginBlockWithARelativePathOrAMisshapedPinRefusesTheCatalog(t *testing.T) {
	cases := []struct {
		name   string
		path   string
		digest string
	}{
		{"a relative path", "plugins/x", strings.Repeat("c", 64)},
		{"a pin of 63 hex characters", "/opt/retriever/plugins/example", strings.Repeat("c", 63)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			content := `version: 1
sources:
  - id: docs
    kind: outline
    endpoint: https://docs.example.test
    credential_env: EXAMPLE_OUTLINE_API_KEY
    plugin:
      path: ` + tc.path + `
      sha256: ` + tc.digest + `
`
			cat, err := loadCatalog(t, content, domain.BuiltinKinds(), nil)
			if err == nil {
				t.Fatalf("Load error = nil, want the plugin block refused")
			}
			if len(cat.All()) != 0 {
				t.Fatalf("catalog holds sources, want none")
			}
		})
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/catalog-loading.feature: A profile catalog refuses a credential name without the profile prefix
func TestAProfileCatalogRefusesACredentialNameWithoutTheProfilePrefix(t *testing.T) {
	l, err := catalog.NewWithFiles("/work/sources.yml", catalog.Files{FS: fileSet(map[string]string{"sources.yml": outlineDocsYAML}), SameFile: sameByName}, domain.BuiltinKinds())
	if err != nil {
		t.Fatalf("NewWithFiles: %v", err)
	}
	_, err = l.WithProfile("work").Load()
	wantRule(t, err, "must start with the profile prefix")
}

// @scenario specs/libs/retriever-runtime/behaviours/catalog-loading.feature: A hyphenated profile name gives an underscore prefix that is accepted
func TestAHyphenatedProfileNameGivesAnUnderscorePrefixThatIsAccepted(t *testing.T) {
	content := `version: 1
sources:
  - id: docs
    kind: outline
    endpoint: https://docs.example.test
    credential_env: TEAM_A_OUTLINE_API_KEY
`
	l, err := catalog.NewWithFiles("/work/sources.yml", catalog.Files{FS: fileSet(map[string]string{"sources.yml": content}), SameFile: sameByName}, domain.BuiltinKinds())
	if err != nil {
		t.Fatalf("NewWithFiles: %v", err)
	}
	cat, err := l.WithProfile("team-a").Load()
	if err != nil {
		t.Fatalf("Load error = %v, want the source", err)
	}
	if len(cat.All()) != 1 || cat.All()[0].ID().String() != "docs" {
		t.Fatalf("catalog does not hold the source docs")
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/catalog-loading.feature: A single catalog with no profile keeps the names it was written with
func TestASingleCatalogWithNoProfileKeepsTheNamesItWasWrittenWith(t *testing.T) {
	cat, err := loadCatalog(t, outlineDocsYAML, domain.BuiltinKinds(), nil)
	if err != nil {
		t.Fatalf("Load error = %v", err)
	}
	if len(cat.All()) != 1 || cat.All()[0].ID().String() != "docs" {
		t.Fatalf("catalog does not hold the source docs")
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/catalog-loading.feature: A catalog holds one identity name on one host and loads
func TestACatalogHoldsOneIdentityNameOnOneHostAndLoads(t *testing.T) {
	content := `version: 1
sources:
  - id: jira-a
    kind: jira
    endpoint: https://docs.example.test
    credential_env: EXAMPLE_JIRA_API_KEY
    identity_env: JIRA_EMAIL
  - id: jira-b
    kind: jira
    endpoint: https://docs.example.test
    credential_env: EXAMPLE_JIRA_API_KEY
    identity_env: JIRA_EMAIL
`
	cat, err := loadCatalog(t, content, domain.BuiltinKinds(), nil)
	if err != nil {
		t.Fatalf("Load error = %v, want the two sources", err)
	}
	if len(cat.All()) != 2 {
		t.Fatalf("catalog holds %d sources, want 2", len(cat.All()))
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/catalog-loading.feature: A catalog error for an unknown kind never echoes the value that was given
func TestACatalogErrorForAnUnknownKindNeverEchoesTheValueGiven(t *testing.T) {
	content := strings.Replace(outlineDocsYAML, "kind: outline", "kind: EXAMPLE-SECRET-VALUE", 1)
	_, err := loadCatalog(t, content, domain.BuiltinKinds(), nil)
	if err == nil {
		t.Fatalf("Load error = nil, want the unknown kind")
	}
	if strings.Contains(err.Error(), "EXAMPLE-SECRET-VALUE") {
		t.Fatalf("error %q echoes the kind value", err.Error())
	}
}
