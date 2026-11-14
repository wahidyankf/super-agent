package catalog

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// specDigest is the SHA-256 of the loaded spec file of the example kind, in lower-case hex. The loader compares text, so
// the tests need no real hash.
var specDigest = strings.Repeat("a", 64)

// otherDigest is a digest that no loaded spec file has.
var otherDigest = strings.Repeat("b", 64)

// specKinds returns the built-in kinds plus one spec kind, example-wiki, which a loaded spec registers.
func specKinds(t *testing.T) domain.KindSet {
	t.Helper()
	kinds, err := domain.NewKindSet(
		domain.KindDefinition{Name: "outline", Display: "Outline"},
		domain.KindDefinition{Name: "kiwi", Display: "Kiwi"},
		domain.KindDefinition{Name: "jira", Display: "Jira", UsesIdentity: true},
		domain.KindDefinition{Name: "example-wiki"},
	)
	if err != nil {
		t.Fatalf("NewKindSet() error = %v", err)
	}
	return kinds
}

// pinnedCatalog is a catalog of one source of the spec kind, with the spec_sha256 value given.
func pinnedCatalog(pin string) string {
	return fmt.Sprintf(`version: 1
sources:
  - id: wiki
    kind: example-wiki
    endpoint: https://wiki.example.test
    credential_env: EXAMPLE_WIKI_API_KEY
    spec_sha256: %s
`, pin)
}

// loadPinned loads the catalog content with the given loaded-spec digests, through fake file calls.
func loadPinned(t *testing.T, content string, digests map[string]string) (domain.Catalog, error) {
	t.Helper()
	l := &Loader{path: "/work/project/sources.yml", files: contentFiles(content), kinds: specKinds(t)}
	return l.WithSpecDigests(digests).Load()
}

func TestSpecPinMatchingTheLoadedSpecLoadsTheSource(t *testing.T) {
	cat, err := loadPinned(t, pinnedCatalog(specDigest), map[string]string{"example-wiki": specDigest})
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if got := len(cat.All()); got != 1 {
		t.Fatalf("sources = %d, want 1", got)
	}
}

// TestC2ChangedSpecFailsTheCatalogLoadWithItsCodeAndNoSource is the threat C2 proof: a spec whose bytes differ from the
// catalog pin fails the load with the stable code, names the source and the field, and returns no catalog, so no
// request can be sent.
func TestC2ChangedSpecFailsTheCatalogLoadWithItsCodeAndNoSource(t *testing.T) {
	cat, err := loadPinned(t, pinnedCatalog(specDigest), map[string]string{"example-wiki": otherDigest})
	if err == nil {
		t.Fatal("Load() error = nil, want a spec pin mismatch")
	}
	var mismatch PinMismatchError
	if !errors.As(err, &mismatch) {
		t.Fatalf("Load() error = %T %v, want PinMismatchError", err, err)
	}
	if got := mismatch.Code(); got != "retriever.connector.spec_pin_mismatch" {
		t.Fatalf("Code() = %q, want retriever.connector.spec_pin_mismatch", got)
	}
	want := catalogPrefix + "source 1 spec_sha256: does not match the loaded spec file"
	if got := err.Error(); got != want {
		t.Fatalf("Load() error = %q, want %q", got, want)
	}
	if len(cat.All()) != 0 {
		t.Fatal("a refused pin returned sources, want none")
	}
}

func TestSpecPinIsRefusedForABuiltinKind(t *testing.T) {
	content := `version: 1
sources:
  - id: wiki
    kind: outline
    endpoint: https://docs.example.test
    credential_env: EXAMPLE_OUTLINE_API_KEY
    spec_sha256: ` + specDigest + "\n"
	_, err := loadPinned(t, content, map[string]string{"example-wiki": specDigest})
	want := catalogPrefix + "source 1 spec_sha256: only for a spec kind"
	if err == nil || err.Error() != want {
		t.Fatalf("Load() error = %v, want %q", err, want)
	}
}

func TestSpecPinMustBeLowerCaseHex(t *testing.T) {
	for _, pin := range []string{strings.ToUpper(specDigest), specDigest[:63], "not-a-digest"} {
		_, err := loadPinned(t, pinnedCatalog(pin), map[string]string{"example-wiki": pin})
		want := catalogPrefix + "source 1 spec_sha256: must be 64 lower-case hex characters"
		if err == nil || err.Error() != want {
			t.Fatalf("pin %q: Load() error = %v, want %q", pin, err, want)
		}
	}
}

func TestSpecPinIsOptionalForASpecKind(t *testing.T) {
	content := strings.Replace(pinnedCatalog(specDigest), "    spec_sha256: "+specDigest+"\n", "", 1)
	if _, err := loadPinned(t, content, map[string]string{"example-wiki": specDigest}); err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
}
