package domain

import (
	"reflect"
	"testing"
)

// cappedCatalog builds a catalog of three sources, one per access level, with the IDs of their levels.
func cappedCatalog(t *testing.T) Catalog {
	t.Helper()
	mk := func(id string, access Access) Source {
		sid, err := NewSourceID(id)
		if err != nil {
			t.Fatalf("NewSourceID(%s): %v", id, err)
		}
		ep, err := NewEndpoint("https://" + id + ".example.com")
		if err != nil {
			t.Fatalf("NewEndpoint: %v", err)
		}
		ref, err := NewCredentialRef("CAP_" + map[string]string{"ro": "RO", "wr": "WR", "ad": "AD"}[id] + "_API_KEY")
		if err != nil {
			t.Fatalf("NewCredentialRef: %v", err)
		}
		tags, err := NewTagSet("wiki")
		if err != nil {
			t.Fatalf("NewTagSet: %v", err)
		}
		return NewSource(sid, KindOutline, tags, access, ep, ref)
	}
	cat, err := NewCatalog([]Source{
		mk("ro", AccessRead),
		mk("wr", AccessWrite),
		mk("ad", AccessAdmin),
	})
	if err != nil {
		t.Fatalf("NewCatalog: %v", err)
	}
	return cat
}

// levelsOf returns the access level of each source, by ID.
func levelsOf(c Catalog) map[string]string {
	out := map[string]string{}
	for _, src := range c.All() {
		out[src.ID().String()] = src.Access().String()
	}
	return out
}

// TestCappedLowersEverySourceToTheCeiling checks the ceiling rule: each source's level becomes the lower of its own
// level and the ceiling, so one enforcement point (the use case) sees the capped value.
func TestCappedLowersEverySourceToTheCeiling(t *testing.T) {
	cat := cappedCatalog(t)
	cases := []struct {
		ceiling Access
		want    map[string]string
	}{
		{AccessRead, map[string]string{"ro": "read", "wr": "read", "ad": "read"}},
		{AccessWrite, map[string]string{"ro": "read", "wr": "write", "ad": "write"}},
		{AccessAdmin, map[string]string{"ro": "read", "wr": "write", "ad": "admin"}},
	}
	for _, tc := range cases {
		got := levelsOf(cat.Capped(tc.ceiling))
		for id, want := range tc.want {
			if got[id] != want {
				t.Errorf("ceiling %s: source %s = %q, want %q", tc.ceiling, id, got[id], want)
			}
		}
	}
}

// TestCappedMaxAccessIsTheCappedMaximum checks that the registration ceiling of the MCP server reads the capped value:
// a read ceiling over an admin source gives MaxAccess read, so no admin tool is registered.
func TestCappedMaxAccessIsTheCappedMaximum(t *testing.T) {
	cat := cappedCatalog(t)
	if got := cat.Capped(AccessRead).MaxAccess().String(); got != "read" {
		t.Fatalf("MaxAccess under a read ceiling = %q, want read", got)
	}
	if got := cat.Capped(AccessWrite).MaxAccess().String(); got != "write" {
		t.Fatalf("MaxAccess under a write ceiling = %q, want write", got)
	}
	if got := cat.Capped(AccessAdmin).MaxAccess().String(); got != "admin" {
		t.Fatalf("MaxAccess under an admin ceiling = %q, want admin", got)
	}
}

// TestCappedKeepsEverythingElseAndDoesNotChangeTheOriginal checks that only the level moves: the IDs, kinds, tags,
// endpoints, and credential names stay, the source order stays, and the catalog it came from is unchanged.
func TestCappedKeepsEverythingElseAndDoesNotChangeTheOriginal(t *testing.T) {
	cat := cappedCatalog(t)
	capped := cat.Capped(AccessRead)
	if len(capped.All()) != 3 {
		t.Fatalf("capped catalog has %d sources, want 3", len(capped.All()))
	}
	for i, src := range capped.All() {
		orig := cat.All()[i]
		if src.ID() != orig.ID() || src.Kind() != orig.Kind() || src.Endpoint() != orig.Endpoint() ||
			src.Credential() != orig.Credential() || !reflect.DeepEqual(src.Tags(), orig.Tags()) {
			t.Fatalf("source %d changed beyond its level", i)
		}
	}
	if got := levelsOf(cat)["ad"]; got != "admin" {
		t.Fatalf("the original catalog changed: ad = %q, want admin", got)
	}
}

// TestCappedWithAZeroCeilingAllowsNothing checks that a zero ceiling allows nothing, so no source can run an operation.
func TestCappedWithAZeroCeilingAllowsNothing(t *testing.T) {
	cat := cappedCatalog(t).Capped(Access{})
	if cat.MaxAccess().Allows(AccessRead) {
		t.Fatal("a zero ceiling must allow no read")
	}
}
