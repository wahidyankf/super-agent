package catalog

import (
	"strings"
	"testing"
)

// twoSourcesOfKind is a catalog of two sources of the spec kind. The first one carries the pin given, or none when the pin
// is empty, and the second one never carries a pin.
func twoSourcesOfKind(pin string) string {
	first := "    endpoint: https://wiki.example.test\n    credential_env: EXAMPLE_WIKI_API_KEY\n"
	if pin != "" {
		first += "    spec_sha256: " + pin + "\n"
	}
	return "version: 1\nsources:\n  - id: wiki\n    kind: example-wiki\n" + first +
		"  - id: docs\n    kind: example-wiki\n    endpoint: https://docs.example.test\n    credential_env: EXAMPLE_DOCS_API_KEY\n"
}

// TestPinnedKindsNeedEverySourceToPin checks that a spec kind counts as pinned only when every source of that kind pins it.
// A pin on one source of two does not pin the kind, because the other source runs the same spec unpinned.
func TestPinnedKindsNeedEverySourceToPin(t *testing.T) {
	digests := map[string]string{"example-wiki": specDigest}

	l := &Loader{path: "/work/project/sources.yml", files: contentFiles(twoSourcesOfKind(specDigest)), kinds: specKinds(t)}
	if _, err := l.WithSpecDigests(digests).Load(); err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if l.PinnedKinds()["example-wiki"] {
		t.Fatal("one pin of two sources pins the kind")
	}

	both := strings.Replace(twoSourcesOfKind(""), "    credential_env: EXAMPLE_DOCS_API_KEY\n",
		"    credential_env: EXAMPLE_DOCS_API_KEY\n    spec_sha256: "+specDigest+"\n", 1)
	both = strings.Replace(both, "    credential_env: EXAMPLE_WIKI_API_KEY\n",
		"    credential_env: EXAMPLE_WIKI_API_KEY\n    spec_sha256: "+specDigest+"\n", 1)
	l = &Loader{path: "/work/project/sources.yml", files: contentFiles(both), kinds: specKinds(t)}
	if _, err := l.WithSpecDigests(digests).Load(); err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if !l.PinnedKinds()["example-wiki"] {
		t.Fatal("two pins of two sources do not pin the kind")
	}
}

// TestPinnedKindsOfAKindWithNoSourceAreEmpty checks that a kind with no source is never reported as pinned, and that the
// map a caller reads is its own copy.
func TestPinnedKindsOfAKindWithNoSourceAreEmpty(t *testing.T) {
	outlineOnly := "version: 1\nsources:\n  - id: docs\n    kind: outline\n    endpoint: https://docs.example.test\n" +
		"    credential_env: EXAMPLE_OUTLINE_API_KEY\n"
	l := &Loader{path: "/work/project/sources.yml", files: contentFiles(outlineOnly), kinds: specKinds(t)}
	if _, err := l.WithSpecDigests(map[string]string{"example-wiki": specDigest}).Load(); err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	pinned := l.PinnedKinds()
	if pinned["example-wiki"] {
		t.Fatal("a kind with no source is reported as pinned")
	}
	pinned["example-wiki"] = true
	if l.PinnedKinds()["example-wiki"] {
		t.Fatal("a change to the returned map reached the loader")
	}
}
