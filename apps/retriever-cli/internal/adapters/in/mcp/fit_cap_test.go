package mcp

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/operations"
)

// TestCutTextNeverSplitsACharacter checks the cut of one string: at most the limit, and on a character boundary.
func TestCutTextNeverSplitsACharacter(t *testing.T) {
	if got := cutText("abc", 10); got != "abc" {
		t.Fatalf("a short text = %q, want it unchanged", got)
	}
	if got := cutText("abcdef", 3); got != "abc" {
		t.Fatalf("cut = %q, want abc", got)
	}
	euro := strings.Repeat("€", 4) // three bytes each
	got := cutText(euro, 7)
	if got != "€€" || !utf8.ValidString(got) {
		t.Fatalf("cut of euro signs = %q, want two whole characters", got)
	}
	if got := cutText(euro, 0); got != "" {
		t.Fatalf("cut to zero = %q, want empty", got)
	}
}

// TestCutStringsAndLongestStringWalkEveryNestedString checks the walk of an object: strings under maps and lists are cut,
// and the longest one is found, while other values are kept.
func TestCutStringsAndLongestStringWalkEveryNestedString(t *testing.T) {
	obj := map[string]any{
		"title": "abcdef",
		"n":     7,
		"nested": map[string]any{
			"text": "ghijkl",
			"list": []any{"mnopqr", true},
		},
	}
	if got := longestString(obj); got != 6 {
		t.Fatalf("longestString = %d, want 6", got)
	}
	cut := cutStrings(obj, 2).(map[string]any)
	if cut["title"] != "ab" || cut["n"] != 7 {
		t.Fatalf("cut top level = %v", cut)
	}
	nested := cut["nested"].(map[string]any)
	list := nested["list"].([]any)
	if nested["text"] != "gh" || list[0] != "mn" || list[1] != true {
		t.Fatalf("cut nested = %v", nested)
	}
	if obj["title"] != "abcdef" {
		t.Fatal("cutStrings changed the object it was given")
	}
}

// TestCutToCapKeepsAnObjectThatFitsAndCutsOneThatDoesNot checks the object cut: a fitting object is returned as it is,
// and a larger one has its longest strings shortened until the whole answer fits the cap.
func TestCutToCapKeepsAnObjectThatFitsAndCutsOneThatDoesNot(t *testing.T) {
	wrap := func(m map[string]any) any { return m }
	small := map[string]any{"id": "a", "truncated": true}
	if got, cut := cutToCap(small, wrap); cut || got["id"] != "a" {
		t.Fatalf("a fitting object = %v, cut %v", got, cut)
	}
	big := map[string]any{"id": "a", "text": strings.Repeat("x", 40000)}
	got, cut := cutToCap(big, wrap)
	if !cut {
		t.Fatal("an object over the cap was not cut")
	}
	raw, _ := json.Marshal(got)
	if len(raw) > pageCap || got["id"] != "a" {
		t.Fatalf("cut object is %d bytes with id %v, want at most %d and the id kept", len(raw), got["id"], pageCap)
	}
	if text, _ := got["text"].(string); len(text) == 0 || len(text) >= 40000 {
		t.Fatalf("the text was cut to %d bytes, want a shorter, non-empty text", len(text))
	}
}

// TestCutToCapCutsANestedStringBeforeItDropsAnything checks that a long nested string is shortened in place, so the nested
// object is kept, and that the object keeps its scalars.
func TestCutToCapCutsANestedStringBeforeItDropsAnything(t *testing.T) {
	wrap := func(m map[string]any) any { return m }
	huge := map[string]any{"id": "a", "nested": map[string]any{"deep": strings.Repeat("y", 40000)}}
	got, cut := cutToCap(huge, wrap)
	if !cut || got["id"] != "a" {
		t.Fatalf("cut = %v, %v; want the scalar id kept", got, cut)
	}
	if _, ok := got["nested"].(map[string]any); !ok {
		t.Fatalf("the nested object was dropped although its string could be cut: %v", got)
	}
}

// TestCutToCapKeepsOnlyScalarsWhenNothingElseFits checks the last resort: an object whose nested keys alone exceed the cap,
// even with every string empty, keeps its scalar fields only.
func TestCutToCapKeepsOnlyScalarsWhenNothingElseFits(t *testing.T) {
	wrap := func(m map[string]any) any { return m }
	keys := map[string]any{}
	for i := range 6000 {
		keys[strings.Repeat("k", 4)+string(rune('a'+i%26))+strings.Repeat("z", i%7)+strconv.Itoa(i)] = ""
	}
	huge := map[string]any{"id": "a", "nested": keys}
	got, cut := cutToCap(huge, wrap)
	if !cut || got["id"] != "a" {
		t.Fatalf("cut = %v, %v; want the scalar id kept", got, cut)
	}
	if _, ok := got["nested"]; ok {
		t.Fatal("the nested object survived although no cut could make it fit")
	}
}

// TestRenderOfAPagedListNamesTheNextPage checks the continuation of a list answer: a short page is the last one, a full
// page may be followed, and a list with no paging names no next offset.
func TestRenderOfAPagedListNamesTheNextPage(t *testing.T) {
	row, ok := operations.Lookup(domain.OpDocumentsList)
	if !ok {
		t.Fatal("no documents.list row")
	}
	doc := func(id string) domain.Record {
		return domain.NewRecord(domain.Field{Name: "id", Value: id}, domain.Field{Name: "title", Value: "t"})
	}
	items := []domain.Record{doc("a"), doc("b")}

	full, err := render(row, "outline", domain.NewList(items).WithPage(domain.NewPage(4, 2, 2)), 0)
	if err != nil {
		t.Fatal(err)
	}
	if full["has_more"] != true || full["next_offset"] != 6 || full["count"] != 2 {
		t.Fatalf("full page = %v, want has_more true and next_offset 6", full)
	}

	short, err := render(row, "outline", domain.NewList(items[:1]).WithPage(domain.NewPage(4, 2, 1)), 0)
	if err != nil {
		t.Fatal(err)
	}
	if short["has_more"] != false {
		t.Fatalf("short page = %v, want has_more false", short)
	}
	if _, ok := short["next_offset"]; ok {
		t.Fatalf("short page = %v, want no next_offset", short)
	}

	plain, err := render(row, "outline", domain.NewList(items), 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := plain["has_more"]; ok {
		t.Fatalf("an unpaged list = %v, want no has_more", plain)
	}
}

// TestRenderCutsASingleRowLargerThanTheCap checks the field-level cut: one row over the cap is returned cut and marked
// truncated, and the next offset moves past it, so the count is never 0 while a row exists.
func TestRenderCutsASingleRowLargerThanTheCap(t *testing.T) {
	row, ok := operations.Lookup(domain.OpDocumentsList)
	if !ok {
		t.Fatal("no documents.list row")
	}
	big := domain.NewRecord(
		domain.Field{Name: "id", Value: "a"},
		domain.Field{Name: "title", Value: strings.Repeat("t", 40000)},
	)
	out, err := render(row, "outline", domain.NewList([]domain.Record{big}).WithPage(domain.NewPage(3, 25, 1)), 0)
	if err != nil {
		t.Fatal(err)
	}
	if out["count"] != 1 || out["truncated"] != true || out["has_more"] != true || out["next_offset"] != 4 {
		t.Fatalf("cut row answer = %v, want count 1, truncated, has_more, next_offset 4", out)
	}
	raw, _ := json.Marshal(out)
	if len(raw) > pageCap {
		t.Fatalf("the answer is %d bytes, want at most %d", len(raw), pageCap)
	}
}

// TestRenderOfAnObjectHasTheCapAndTheTruncatedFlag checks an object answer: a small object is not truncated, and an
// object over the cap is cut and says so.
func TestRenderOfAnObjectHasTheCapAndTheTruncatedFlag(t *testing.T) {
	row, ok := operations.Lookup(domain.OpCommentsGet)
	if !ok {
		t.Fatal("no comments.get row")
	}
	small := domain.NewRecord(domain.Field{Name: "id", Value: "c1"}, domain.Field{Name: "text", Value: "short"})
	out, err := render(row, "outline", small, 0)
	if err != nil {
		t.Fatal(err)
	}
	if out["truncated"] != false {
		t.Fatalf("a small object = %v, want truncated false", out)
	}

	large := domain.NewRecord(domain.Field{Name: "id", Value: "c2"}, domain.Field{Name: "text", Value: strings.Repeat("c", 40000)})
	out, err = render(row, "outline", large, 0)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(out)
	if out["truncated"] != true || len(raw) > pageCap || out["id"] != "c2" {
		t.Fatalf("a large object = %d bytes, truncated %v; want cut, truncated, and the id kept", len(raw), out["truncated"])
	}
}
