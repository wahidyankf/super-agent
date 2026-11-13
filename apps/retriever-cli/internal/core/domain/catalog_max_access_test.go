package domain

import "testing"

// TestMaxAccessIsTheHighestLevelOfAnySource checks the registration ceiling: the highest level in the catalog, and
// the zero value for a zero catalog, which allows nothing.
func TestMaxAccessIsTheHighestLevelOfAnySource(t *testing.T) {
	var zero Catalog
	if zero.MaxAccess().Allows(AccessRead) {
		t.Fatal("a zero catalog allows read")
	}
	read := tagged(t, testSource(t, "a", "outline", "a.example.com", "A_API_KEY"), AccessRead)
	write := tagged(t, testSource(t, "b", "outline", "b.example.com", "B_API_KEY"), AccessWrite)
	both, err := NewCatalog([]Source{read, write})
	if err != nil {
		t.Fatal(err)
	}
	if got := both.MaxAccess(); got.String() != "write" {
		t.Fatalf("MaxAccess = %q, want write", got.String())
	}
	onlyRead, err := NewCatalog([]Source{read})
	if err != nil {
		t.Fatal(err)
	}
	if onlyRead.MaxAccess().Allows(AccessWrite) {
		t.Fatal("a read-only catalog allows write")
	}
}

// TestMaxAccessOfCountsOnlyTheNamedKinds checks the ceiling of a set of kinds: a source of another kind does not lift it
// . The MCP tools use it for the kinds that have tools, so a Kiwi write source adds no level for them.
func TestMaxAccessOfCountsOnlyTheNamedKinds(t *testing.T) {
	outline := tagged(t, testSource(t, "a", "outline", "a.example.com", "A_API_KEY"), AccessRead)
	// testSource builds only outline and the package's other kind, so the Kiwi source is rebuilt with KindKiwi.
	base := testSource(t, "k", "outline", "k.example.com", "K_API_KEY")
	kiwi := NewSource(base.ID(), KindKiwi, mustTagSet(t), AccessWrite, base.Endpoint(), base.Credential())
	cat, err := NewCatalog([]Source{outline, kiwi})
	if err != nil {
		t.Fatal(err)
	}
	if got := cat.MaxAccessOf(KindOutline); got.String() != "read" {
		t.Fatalf("MaxAccessOf(outline) = %q, want read", got.String())
	}
	if got := cat.MaxAccessOf(KindKiwi); got.String() != "write" {
		t.Fatalf("MaxAccessOf(kiwi) = %q, want write", got.String())
	}
	if got := cat.MaxAccessOf(KindOutline, KindKiwi); got.String() != "write" {
		t.Fatalf("MaxAccessOf(outline, kiwi) = %q, want write", got.String())
	}
	if cat.MaxAccessOf(KindJira).Allows(AccessRead) {
		t.Fatal("a kind with no source in the catalog allows read")
	}
	if cat.MaxAccessOf().Allows(AccessRead) {
		t.Fatal("no kinds allow read")
	}
	if got := cat.MaxAccess(); got.String() != "write" {
		t.Fatalf("MaxAccess = %q, want write, the level of every kind", got.String())
	}
}
