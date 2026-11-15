package engine

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// TestTimestampsAreRFC3339InUTC pins the conversion of a Jira timestamp, and a value of another shape is empty.
// Moved unchanged from the Jira connector's markdown tests: the engine answers what the Go code answers.
func TestTimestampsAreRFC3339InUTC(t *testing.T) {
	cases := map[string]string{
		"2026-11-02T09:15:00.000+0700": "2026-11-02T02:15:00Z",
		"2026-11-03T10:00:00.000+0000": "2026-11-03T10:00:00Z",
		"2026-11-02T09:15:00+0700":     "",
		"yesterday":                    "",
		"":                             "",
	}
	for raw, want := range cases {
		if got := timestampText(raw); got != want {
			t.Fatalf("timestampText(%q) = %q, want %q", raw, got, want)
		}
	}
}

// TestDueDatesAreDatesOnly pins the due date: a calendar date passes, and anything else is empty.
// Moved unchanged from the Jira connector's markdown tests.
func TestDueDatesAreDatesOnly(t *testing.T) {
	cases := map[string]string{
		"2026-12-01":           "2026-12-01",
		"2026-12-01T00:00:00Z": "",
		"1 Dec":                "",
		"":                     "",
	}
	for raw, want := range cases {
		if got := dueText(raw); got != want {
			t.Fatalf("dueText(%q) = %q, want %q", raw, got, want)
		}
	}
}

// TestTitlesAreOneLineOfAtMost255Characters proves the title limit counts characters, not bytes, and no line break
// or control character survives. Moved unchanged from the Jira connector's markdown tests.
func TestTitlesAreOneLineOfAtMost255Characters(t *testing.T) {
	long := strings.Repeat("é", 300)
	title := titleText(long)
	if utf8.RuneCountInString(title) != 255 {
		t.Fatalf("title has %d characters, want 255", utf8.RuneCountInString(title))
	}
	if got := titleText("a\r\nb\tc\x00d‮e"); got != "a b cde" {
		t.Fatalf("titleText = %q, want one line with no control or direction character", got)
	}
}

// TestPlainTextKeepsNoLineBreak proves a field that must stay on one line is one line.
// Moved unchanged from the Jira connector's markdown tests.
func TestPlainTextKeepsNoLineBreak(t *testing.T) {
	if got := plainText("one\ntwo\r\nthree"); got != "one two three" {
		t.Fatalf("plainText = %q", got)
	}
}

// TestJiraTimeMapsBadLayoutToEmpty plants a shifted layout and finds the empty text, so a bad timestamp keeps its
// row with a visible gap instead of failing it.
func TestJiraTimeMapsBadLayoutToEmpty(t *testing.T) {
	budget := 1 << 20
	got, err := jiraTime("2026-11-02T09:15:00+0700", &budget)
	if err != nil {
		t.Fatalf("jiraTime: %v", err)
	}
	if got != "" {
		t.Fatalf("jiraTime of a shifted layout = %q, want empty", got)
	}
}

// TestJiraTitleKeeps255Runes proves the title cut counts runes: 300 e-acutes keep 255, not 255 bytes.
func TestJiraTitleKeeps255Runes(t *testing.T) {
	budget := 1 << 20
	got, err := jiraTitle(strings.Repeat("é", 300), &budget)
	if err != nil {
		t.Fatalf("jiraTitle: %v", err)
	}
	s, ok := got.(string)
	if !ok {
		t.Fatalf("jiraTitle = %#v, want a string", got)
	}
	if utf8.RuneCountInString(s) != 255 {
		t.Fatalf("title has %d characters, want 255", utf8.RuneCountInString(s))
	}
}

// TestJiraTitleStripsInvisible plants a zero-width space and a bidi override and finds neither, so hidden text
// reaches no output.
func TestJiraTitleStripsInvisible(t *testing.T) {
	budget := 1 << 20
	got, err := jiraTitle("a​b‮c", &budget)
	if err != nil {
		t.Fatalf("jiraTitle: %v", err)
	}
	if got != "abc" {
		t.Fatalf("jiraTitle = %q, want neither invisible character", got)
	}
}

// TestJiraDateRefusesBareYear plants a bare year and finds the empty text with the field present.
func TestJiraDateRefusesBareYear(t *testing.T) {
	budget := 1 << 20
	got, err := jiraDate("2026", &budget)
	if err != nil {
		t.Fatalf("jiraDate: %v", err)
	}
	if got != "" {
		t.Fatalf("jiraDate(2026) = %q, want empty", got)
	}
}

// TestJiraConvertersMapMissingToEmpty checks that a missing or null value maps to the empty text, so the field stays
// present and the gap is visible.
func TestJiraConvertersMapMissingToEmpty(t *testing.T) {
	budget := 1 << 20
	for _, raw := range []any{nil} {
		if got, err := jiraTime(raw, &budget); err != nil || got != "" {
			t.Fatalf("jiraTime = (%v, %v), want (\"\", nil)", got, err)
		}
		if got, err := jiraDate(raw, &budget); err != nil || got != "" {
			t.Fatalf("jiraDate = (%v, %v), want (\"\", nil)", got, err)
		}
		if got, err := jiraTitle(raw, &budget); err != nil || got != "" {
			t.Fatalf("jiraTitle = (%v, %v), want (\"\", nil)", got, err)
		}
	}
}

// TestJiraConvertersMapAnotherShapeToEmpty checks that a value of another shape maps to the empty text instead of
// failing the row: a number, a flag, an object, and a list keep the row with a visible gap.
func TestJiraConvertersMapAnotherShapeToEmpty(t *testing.T) {
	budget := 1 << 20
	vals := []any{float64(3), true, map[string]any{"a": 1}, []any{"x"}}
	for _, raw := range vals {
		if got, err := jiraTime(raw, &budget); err != nil || got != "" {
			t.Fatalf("jiraTime(%v) = (%v, %v), want (\"\", nil)", raw, got, err)
		}
		if got, err := jiraDate(raw, &budget); err != nil || got != "" {
			t.Fatalf("jiraDate(%v) = (%v, %v), want (\"\", nil)", raw, got, err)
		}
		if got, err := jiraTitle(raw, &budget); err != nil || got != "" {
			t.Fatalf("jiraTitle(%v) = (%v, %v), want (\"\", nil)", raw, got, err)
		}
	}
}

// TestJiraConvertersFailPastTheBudget checks that a converted text past the budget fails, so a reply cannot build
// more text than one reply may hold.
func TestJiraConvertersFailPastTheBudget(t *testing.T) {
	budget := 4
	if _, err := jiraTime("2026-11-02T09:15:00.000+0700", &budget); err == nil {
		t.Fatal("an over-budget time passes")
	}
	budget = 4
	if _, err := jiraDate("2026-12-01", &budget); err == nil {
		t.Fatal("an over-budget date passes")
	}
	budget = 2
	if _, err := jiraTitle("abc", &budget); err == nil {
		t.Fatal("an over-budget title passes")
	}
}
