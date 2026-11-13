package jira

import (
	"strings"
	"time"
	"unicode/utf8"
)

// jiraTimeLayout reads the timestamps of Jira, such as 2026-11-10T10:00:00.000+0700. The layout is fixed, so a value of any
// other shape does not parse.
const jiraTimeLayout = "2006-01-02T15:04:05.000-0700"

// maxTitleRunes is the longest title a document or a search row carries.
const maxTitleRunes = 255

// lineBreaks turns every line break and tab into a space, so a field stays on one line.
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

// projectOfKey returns the project key of an issue key, which is the part before the last hyphen. The key has passed
// ParseIssueKey, so the hyphen is always there.
func projectOfKey(key string) string {
	return key[:strings.LastIndex(key, "-")]
}

// noDescription is the text of an issue whose description is empty or null.
const noDescription = "(no description)"

// issueMarkdown assembles the Markdown of one issue: the title, a metadata list, and the description. Free
// text is escaped with escapeText. The key, the dates, and the due date are written as they are, because they passed their
// checks. Optional lines are left out when they have no value. The description is rendered from ADF, and an empty one is
// "(no description)". The result ends with the description and no trailing line break.
func issueMarkdown(key, title string, f *issueFields) (string, error) {
	description, err := RenderADF(f.Description)
	if err != nil {
		return "", err
	}
	if description == "" {
		description = noDescription
	}
	var b strings.Builder
	b.WriteString("# " + escapeText(title) + "\n\n")
	line := func(label, value string) {
		b.WriteString("- " + label + ": " + value + "\n")
	}
	line("Key", key)
	line("Project", projectLine(key, f))
	line("Type", escapeText(namedText(f.IssueType)))
	line("Status", escapeText(namedText(f.Status)))
	line("Priority", escapeText(namedText(f.Priority)))
	if resolution := namedText(f.Resolution); resolution != "" {
		line("Resolution", escapeText(resolution))
	}
	assignee := personText(f.Assignee)
	if assignee == "" {
		assignee = "Unassigned"
	}
	line("Assignee", escapeText(assignee))
	line("Reporter", escapeText(personText(f.Reporter)))
	if labels := escapedLabels(f.Labels); labels != "" {
		line("Labels", labels)
	}
	if parent := parentKey(f); parent != "" {
		line("Parent", parent)
	}
	line("Created", timestampText(f.Created))
	line("Updated", timestampText(f.Updated))
	if f.DueDate != nil {
		if due := dueText(*f.DueDate); due != "" {
			line("Due", due)
		}
	}
	b.WriteString("\n## Description\n\n" + description)
	return b.String(), nil
}

// projectLine returns the project of an issue as "<name> (<KEY>)", or the key alone when the reply holds no name.
func projectLine(key string, f *issueFields) string {
	projectKey := projectOfKey(key)
	if f.Project == nil {
		return projectKey
	}
	if name := plainText(f.Project.Name); name != "" {
		return escapeText(name) + " (" + projectKey + ")"
	}
	return projectKey
}

// escapedLabels returns the labels as one comma-separated line of escaped text, or "" when there are none.
func escapedLabels(labels []string) string {
	escaped := make([]string, 0, len(labels))
	for _, label := range labels {
		if text := plainText(label); text != "" {
			escaped = append(escaped, escapeText(text))
		}
	}
	return strings.Join(escaped, ", ")
}

// parentKey returns the key of the parent issue when the reply names one that passes the issue pattern, or "" otherwise.
func parentKey(f *issueFields) string {
	if f.Parent == nil {
		return ""
	}
	if _, err := ParseIssueKey(f.Parent.Key); err != nil {
		return ""
	}
	return f.Parent.Key
}
