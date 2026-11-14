package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestNewSourceIDAcceptsSlugs(t *testing.T) {
	for _, raw := range []string{"a", "outline", "outline-engineering", "a1-b2", "x9", strings.Repeat("a", 32)} {
		id, err := NewSourceID(raw)
		if err != nil {
			t.Fatalf("NewSourceID(%q) error = %v, want nil", raw, err)
		}
		if id.String() != raw {
			t.Fatalf("NewSourceID(%q).String() = %q", raw, id.String())
		}
	}
}

func TestNewSourceIDRejectsOtherShapes(t *testing.T) {
	for _, raw := range []string{"", "Outline", "1outline", "outline--engineering", "outline-", "-outline",
		"out line", "out:line", "out\nline", "outline\n", " outline", "a.b", strings.Repeat("a", 33)} {
		if _, err := NewSourceID(raw); !errors.Is(err, ErrInvalidSourceID) {
			t.Fatalf("NewSourceID(%q) error = %v, want ErrInvalidSourceID", raw, err)
		}
	}
}

func TestZeroSourceIDIsInvalid(t *testing.T) {
	var id SourceID
	if id.String() != "" {
		t.Fatalf("zero SourceID.String() = %q, want empty", id.String())
	}
	if _, err := NewSourceID(id.String()); !errors.Is(err, ErrInvalidSourceID) {
		t.Fatalf("zero SourceID is accepted by the constructor: %v", err)
	}
}

func TestNewSourceKindAcceptsOnlyTheKnownKinds(t *testing.T) {
	kind, err := NewSourceKind("outline")
	if err != nil || kind != KindOutline || kind.String() != "outline" {
		t.Fatalf("NewSourceKind(outline) = %v, %v", kind, err)
	}
	if kind, err := NewSourceKind("kiwi"); err != nil || kind != KindKiwi || kind.String() != "kiwi" {
		t.Fatalf("NewSourceKind(kiwi) = %v, %v", kind, err)
	}
	for _, raw := range []string{"", "Outline", "Kiwi", "notion", "outline ", "kiwi "} {
		if _, err := NewSourceKind(raw); !errors.Is(err, ErrUnknownSourceKind) {
			t.Fatalf("NewSourceKind(%q) error = %v, want ErrUnknownSourceKind", raw, err)
		}
	}
}

func TestNewTagAcceptsAndRejects(t *testing.T) {
	for _, raw := range []string{"wiki", "a", "1outline", "wiki-tag", "a1-b2", strings.Repeat("a", 32)} {
		if _, err := NewTag(raw); err != nil {
			t.Fatalf("NewTag(%q) error = %v, want nil", raw, err)
		}
	}
	for _, raw := range []string{"", "Wiki", "wiki--tag", "wiki-", "-wiki", "key:value", "two words", strings.Repeat("a", 33)} {
		if _, err := NewTag(raw); !errors.Is(err, ErrInvalidTag) {
			t.Fatalf("NewTag(%q) error = %v, want ErrInvalidTag", raw, err)
		}
	}
}

func TestNewTagSetDedupesAndSorts(t *testing.T) {
	set, err := NewTagSet("wiki", "company", "wiki")
	if err != nil {
		t.Fatalf("NewTagSet error = %v", err)
	}
	if got := strings.Join(set.Values(), ","); got != "company,wiki" {
		t.Fatalf("Values() = %q, want company,wiki", got)
	}
}

func TestNewTagSetLimitsDistinctTagsToSixteen(t *testing.T) {
	var sixteen []string
	for i := 0; i < 16; i++ {
		sixteen = append(sixteen, "t"+string(rune('a'+i)))
	}
	if _, err := NewTagSet(sixteen...); err != nil {
		t.Fatalf("16 distinct tags error = %v, want nil", err)
	}
	if _, err := NewTagSet(append(sixteen, "tz")...); !errors.Is(err, ErrTooManyTags) {
		t.Fatalf("17 distinct tags error = %v, want ErrTooManyTags", err)
	}
	if _, err := NewTagSet(append(sixteen, "ta")...); err != nil {
		t.Fatalf("16 distinct tags plus a duplicate error = %v, want nil", err)
	}
	if _, err := NewTagSet("wiki", "Wiki"); !errors.Is(err, ErrInvalidTag) {
		t.Fatalf("invalid tag error = %v, want ErrInvalidTag", err)
	}
}

func TestTagSetContainsAll(t *testing.T) {
	all := mustTagSet(t, "company", "engineering", "wiki")
	cases := []struct {
		name  string
		other []string
		want  bool
	}{
		{name: "an empty filter matches", other: nil, want: true},
		{name: "one present tag matches", other: []string{"wiki"}, want: true},
		{name: "every tag present matches", other: []string{"wiki", "company"}, want: true},
		{name: "one absent tag fails", other: []string{"wiki", "legal"}, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := all.ContainsAll(mustTagSet(t, tc.other...)); got != tc.want {
				t.Fatalf("ContainsAll(%v) = %v, want %v", tc.other, got, tc.want)
			}
		})
	}
	var none TagSet
	if !none.ContainsAll(TagSet{}) || none.ContainsAll(all) {
		t.Fatalf("zero TagSet containment is wrong")
	}
}

func TestNewEndpointAcceptsHTTPSHostNames(t *testing.T) {
	cases := map[string]string{
		"https://docs.example.test":            "https://docs.example.test",
		"https://docs.example.test/":           "https://docs.example.test",
		"https://wiki.engineering.example.com": "https://wiki.engineering.example.com",
	}
	for raw, want := range cases {
		ep, err := NewEndpoint(raw)
		if err != nil || ep.String() != want {
			t.Fatalf("NewEndpoint(%q) = %q, %v; want %q", raw, ep.String(), err, want)
		}
	}
	ep, _ := NewEndpoint("https://docs.example.test")
	if ep.Host() != "docs.example.test" {
		t.Fatalf("Host() = %q", ep.Host())
	}
}

func TestNewEndpointRejectsEverythingElse(t *testing.T) {
	long := "https://" + strings.Repeat("a.", 127) + "b"
	for _, raw := range []string{
		"", "https://", "docs.example.test",
		"https://reader@docs.example.test", "https://docs.example.test:8443",
		"https://192.0.2.10", "https://1.2.3.4", "https://[::1]", "https://2001:db8::1",
		"https://localhost", "https://a.localhost", "https://intranet", "https://docs.example.test/api",
		"https://docs.example.test?x=1", "https://docs.example.test#frag",
		"https://Docs.example.test", "https://docs..example.test", "https://docs.example.test//",
		long,
	} {
		if _, err := NewEndpoint(raw); !errors.Is(err, ErrInvalidEndpoint) {
			t.Fatalf("NewEndpoint(%q) error = %v, want ErrInvalidEndpoint", raw, err)
		}
	}
}

// TestNewEndpointRefusesPlainHTTPAsHTTPSRequired pins the refusal that the owner's rule names: a plain HTTP endpoint is
// refused at load with HTTPS required, which the failure code reports as https_required.
func TestNewEndpointRefusesPlainHTTPAsHTTPSRequired(t *testing.T) {
	if _, err := NewEndpoint("http://docs.example.test"); !errors.Is(err, ErrHTTPSRequired) {
		t.Fatalf("NewEndpoint(http) error = %v, want ErrHTTPSRequired", err)
	}
}

func TestZeroEndpointStringIsEmpty(t *testing.T) {
	if got := (Endpoint{}).String(); got != "" {
		t.Fatalf("zero Endpoint String() = %q, want empty", got)
	}
}

func TestNewCredentialRefAcceptsAPIKeyNames(t *testing.T) {
	for _, raw := range []string{"EXAMPLE_OUTLINE_API_KEY", "ENGINEERING_OUTLINE_API_KEY", "A_API_KEY", "A1_API_KEY"} {
		ref, err := NewCredentialRef(raw)
		if err != nil || ref.Name() != raw {
			t.Fatalf("NewCredentialRef(%q) = %q, %v", raw, ref.Name(), err)
		}
	}
}

func TestNewCredentialRefRejectsOtherNames(t *testing.T) {
	long := "A" + strings.Repeat("B", 56) + "_API_KEY"
	for _, raw := range []string{"", "GITHUB_WORK_TOKEN", "PROTECTED_PDF_PASSWORD", "path", "lower_API_KEY",
		"EXAMPLE_OUTLINE_KEY", "_API_KEY", "API_KEY", long} {
		if _, err := NewCredentialRef(raw); !errors.Is(err, ErrInvalidCredentialRef) {
			t.Fatalf("NewCredentialRef(%q) error = %v, want ErrInvalidCredentialRef", raw, err)
		}
	}
}

func mustTagSet(t *testing.T, tags ...string) TagSet {
	t.Helper()
	set, err := NewTagSet(tags...)
	if err != nil {
		t.Fatalf("NewTagSet(%v) error = %v", tags, err)
	}
	return set
}
