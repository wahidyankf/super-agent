// This file holds the issue document layout of E22: the title, the closed line kinds with defaults and skips,
// and the description body with its fixed heading and empty text. The title and the line values are collapsed and
// escaped, so planted markup stays literal text; a verbatim line writes a shape-checked key or date as is. The
// escape algorithm is moved from the Jira connector, which keeps its copy until it retires.
package engine

import (
	"strings"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// escapeText returns s as Markdown text that starts a line. See escapeFrom.
func escapeText(s string) string {
	return escapeFrom(s, true)
}

// renderDocument renders the document layout of a read of one object from its mapped record, and sets the markdown
// output field. The text is charged to the shared output budget, like any mapped text.
func renderDocument(o spec.Operation, rec domain.Record, budget *int) (domain.Record, error) {
	text, err := documentText(o.Response.Document, rec)
	if err != nil {
		return domain.Record{}, err
	}
	return withComputedField(rec, spec.DocumentMarkdownField, text, budget)
}

// documentText renders the title, the metadata lines in order, and the body of a document. A line with the skip
// rule is left out when its value is still empty; any other line is emitted even when empty. The result ends with
// the description and no trailing line break.
func documentText(doc *spec.Document, rec domain.Record) (string, error) {
	title, err := documentTitle(doc, rec)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("# " + title + "\n\n")
	for _, line := range doc.Lines {
		value, empty, err := documentLineValue(line, rec)
		if err != nil {
			return "", err
		}
		if empty && line.SkipEmpty {
			continue
		}
		b.WriteString("- " + line.Label + ": " + value + "\n")
	}
	body, err := documentBodyValue(doc.Body, rec)
	if err != nil {
		return "", err
	}
	b.WriteString("\n## " + doc.Body.Heading + "\n\n" + body)
	return b.String(), nil
}

// documentTitle renders the title of a document: one collapsed escaped line. The value must be text: a missing,
// null, or non-text title fails the read.
func documentTitle(doc *spec.Document, rec domain.Record) (string, error) {
	v, _ := rec.Get(doc.Title)
	s, isText := v.(string)
	if !isText {
		return "", fieldError(doc.Title)
	}
	return escapeText(plainText(s)), nil
}

// documentLineValue renders one metadata line: its value, and whether the value is empty. A pair line renders the
// name with the key, and any other line renders its field or verbatim value. A valid check empties a value outside
// the issue-key shape, and a default fills a value that is still empty.
func documentLineValue(line spec.DocLine, rec domain.Record) (string, bool, error) {
	var value string
	var err error
	switch {
	case line.Pair != nil:
		value, err = documentPairValue(line, rec)
	case line.Field != "":
		value, err = documentFieldValue(line, rec)
	default:
		value, err = documentVerbatimValue(line, rec)
	}
	if err != nil {
		return "", false, err
	}
	if line.Valid != "" && value != "" && !issueKeyPattern.MatchString(value) {
		value = ""
	}
	if value == "" && line.Default != "" {
		value = line.Default
	}
	return value, value == "", nil
}

// documentPairValue renders a pair line: the escaped name with the key in parentheses, or the bare key when the
// name is empty. The key must be text: a missing or misshaped key fails the read, as the key of the read is never
// optional.
func documentPairValue(line spec.DocLine, rec domain.Record) (string, error) {
	var name string
	if v, _ := rec.Get(line.Pair.Name); v != nil {
		s, isText := v.(string)
		if !isText {
			return "", fieldError(line.Pair.Name)
		}
		name = escapeText(plainText(s))
	}
	v, _ := rec.Get(line.Pair.Key)
	key, isText := v.(string)
	if !isText {
		return "", fieldError(line.Pair.Key)
	}
	if name == "" {
		return plainText(key), nil
	}
	return name + " (" + plainText(key) + ")", nil
}

// documentFieldValue renders a field line: the escaped value, or the joined strings with the fixed separator. A
// missing or null value is empty; any other misshaped value fails the read.
func documentFieldValue(line spec.DocLine, rec domain.Record) (string, error) {
	v, _ := rec.Get(line.Field)
	if v == nil {
		return "", nil
	}
	if line.Join != "" {
		return documentJoinValue(line, v)
	}
	s, isText := v.(string)
	if !isText {
		return "", fieldError(line.Field)
	}
	return escapeText(plainText(s)), nil
}

// documentJoinValue renders the string list of a joined line with the fixed separator the schema closes. A value
// that is not a list of strings fails the read.
func documentJoinValue(line spec.DocLine, v any) (string, error) {
	arr, isList := v.([]any)
	if !isList {
		return "", fieldError(line.Field)
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		s, isText := item.(string)
		if !isText {
			return "", fieldError(line.Field)
		}
		out = append(out, escapeText(plainText(s)))
	}
	return strings.Join(out, ", "), nil
}

// documentVerbatimValue writes a verbatim line as is: a shape-checked key or date, cleaned but never escaped. A
// missing or null value is empty; any other misshaped value fails the read.
func documentVerbatimValue(line spec.DocLine, rec domain.Record) (string, error) {
	v, _ := rec.Get(line.Verbatim)
	if v == nil {
		return "", nil
	}
	s, isText := v.(string)
	if !isText {
		return "", fieldError(line.Verbatim)
	}
	return plainText(s), nil
}

// documentBodyValue renders the description body: the converted description, or the empty constant when it is
// blank. The value must be text: a misshaped body fails the read.
func documentBodyValue(body spec.DocBody, rec domain.Record) (string, error) {
	v, _ := rec.Get(body.Field)
	if v == nil {
		return body.Empty, nil
	}
	s, isText := v.(string)
	if !isText {
		return "", fieldError(body.Field)
	}
	if s == "" {
		return body.Empty, nil
	}
	return s, nil
}
