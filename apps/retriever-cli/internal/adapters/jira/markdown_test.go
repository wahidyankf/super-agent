package jira

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// TestTimestampsAreRFC3339InUTC pins the conversion of a Jira timestamp, and a value of another shape is empty.
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

// TestTitlesAreOneLineOfAtMost255Characters proves the title limit counts characters, not bytes, and no line break or control
// character survives.
func TestTitlesAreOneLineOfAtMost255Characters(t *testing.T) {
	long := strings.Repeat("é", 300)
	title := titleText(long)
	if utf8.RuneCountInString(title) != 255 {
		t.Fatalf("title has %d characters, want 255", utf8.RuneCountInString(title))
	}
	if got := titleText("a\r\nb\tc\u0000d‮e"); got != "a b cde" {
		t.Fatalf("titleText = %q, want one line with no control or direction character", got)
	}
}

// TestPlainTextKeepsNoLineBreak proves a field that must stay on one line is one line.
func TestPlainTextKeepsNoLineBreak(t *testing.T) {
	if got := plainText("one\ntwo\r\nthree"); got != "one two three" {
		t.Fatalf("plainText = %q", got)
	}
}

// TestTheProjectOfAnIssueKeyIsItsPrefix pins the project key taken from an issue key.
func TestTheProjectOfAnIssueKeyIsItsPrefix(t *testing.T) {
	if got := projectOfKey("EXAMPLE-12"); got != "EXAMPLE" {
		t.Fatalf("projectOfKey = %q", got)
	}
	if got := projectOfKey("AB_C2-1234"); got != "AB_C2" {
		t.Fatalf("projectOfKey = %q", got)
	}
}
