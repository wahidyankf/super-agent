// This file holds the Jira text converters of E21: jira-time, jira-date, and jira-title. Each converter is a pure
// function of one reply value, with the fixed algorithms moved from the Jira connector and reviewed once. A value
// that does not parse maps to the empty text, so the row is kept with a visible gap. The Jira module keeps its copy
// until it retires.
package engine

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// jiraTimeLayout parses the timestamps of Jira, such as 2026-11-10T10:00:00.000+0700. The layout is fixed, so a value
// of any other shape does not parse.
const jiraTimeLayout = "2006-01-02T15:04:05.000-0700"

// maxTitleRunes is the longest title a document or a search row carries.
const maxTitleRunes = 255

// lineBreaks turns every line break and tab into one space, so a field stays on one line.
var lineBreaks = strings.NewReplacer("\r\n", " ", "\n", " ", "\r", " ", "\t", " ")

// plainText returns a one-line text: no control or direction character, and no line break. It is not Markdown; a
// Markdown writer escapes the result before it puts it in a document.
func plainText(s string) string {
	return lineBreaks.Replace(stripControls(s))
}

// titleText returns a title: a plain one-line text of at most maxTitleRunes characters.
func titleText(summary string) string {
	text := plainText(summary)
	if utf8.RuneCountInString(text) <= maxTitleRunes {
		return text
	}
	return string([]rune(text)[:maxTitleRunes])
}

// timestampText returns a Jira timestamp as RFC 3339 in UTC. A value that does not parse is "", never the raw text.
func timestampText(raw string) string {
	t, err := time.Parse(jiraTimeLayout, raw)
	if err != nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// dueText returns a due date as YYYY-MM-DD. A value that does not parse is "".
func dueText(raw string) string {
	if _, err := time.Parse(time.DateOnly, raw); err != nil {
		return ""
	}
	return raw
}

// jiraTime maps one reply value through the jira-time converter: the UTC instant of a Jira timestamp as RFC 3339.
// A missing, null, or non-string value maps to the empty text, as does a value of another shape, so the field stays
// present and the row is kept with a visible gap. The text is charged to the shared output budget, like any mapped
// text.
func jiraTime(raw any, budget *int) (any, error) {
	s, ok := raw.(string)
	if !ok {
		return "", nil
	}
	out := timestampText(s)
	if len(out) > *budget {
		return nil, domain.ErrReplyTooLarge
	}
	*budget -= len(out)
	return out, nil
}

// jiraDate maps one reply value through the jira-date converter: a calendar day passes unchanged. A missing, null,
// or non-string value maps to the empty text, as does a value of another shape, so the field stays present and the
// row is kept with a visible gap. The text is charged to the shared output budget, like any mapped text.
func jiraDate(raw any, budget *int) (any, error) {
	s, ok := raw.(string)
	if !ok {
		return "", nil
	}
	out := dueText(s)
	if len(out) > *budget {
		return nil, domain.ErrReplyTooLarge
	}
	*budget -= len(out)
	return out, nil
}

// jiraTitle maps one reply value through the jira-title converter: one line of at most 255 runes with no hidden
// character. A missing, null, or non-string value maps to the empty text, so the field stays present and the row is
// kept with a visible gap. The text is charged to the shared output budget, like any mapped text.
func jiraTitle(raw any, budget *int) (any, error) {
	s, ok := raw.(string)
	if !ok {
		return "", nil
	}
	out := titleText(s)
	if len(out) > *budget {
		return nil, domain.ErrReplyTooLarge
	}
	*budget -= len(out)
	return out, nil
}
