package kiwi

import (
	"html"
	"strings"
)

// caseDocument is the data of a test case document. The legacy fields are the ones Kiwi still returns.
type caseDocument struct {
	Summary   string
	Status    string
	Priority  string
	Category  string
	Text      string
	Setup     string
	Action    string
	Effect    string
	Breakdown string
}

// caseLine is one case in a plan document's list.
type caseLine struct {
	ID      string
	Summary string
}

// planDocument is the data of a test plan document.
type planDocument struct {
	Name    string
	Product string
	Parent  string
	Text    string
	Cases   []caseLine
}

// clean unescapes one level of the HTML escaping that Kiwi applies to its replies, then removes the characters that
// hide or reorder text: C0 and C1 controls except tab and newline, DEL, zero-width characters, bidirectional controls,
// and the byte order mark. Kiwi text is untrusted data, so Retriever renders none of it as markup.
func clean(raw string) string {
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		switch {
		case r < 0x20, r == 0x7f, r >= 0x80 && r <= 0x9f:
			return -1
		case r >= 0x200b && r <= 0x200f, r >= 0x202a && r <= 0x202e, r >= 0x2066 && r <= 0x2069, r == 0xfeff:
			return -1
		}
		return r
	}, html.UnescapeString(raw))
}

// title returns a name as one line: clean text with its white space collapsed, for a heading or a list item.
func title(raw string) string {
	return strings.Join(strings.Fields(clean(raw)), " ")
}

// block returns clean body text with its surrounding blank lines removed, or "" for an empty body.
func block(raw string) string {
	return strings.Trim(clean(raw), "\n")
}

// metadata returns one "- Key: value" line per pair whose value is set, joined by newlines. It is "" when none is set.
func metadata(pairs ...[2]string) string {
	var lines []string
	for _, pair := range pairs {
		if value := title(pair[1]); value != "" {
			lines = append(lines, "- "+pair[0]+": "+value)
		}
	}
	return strings.Join(lines, "\n")
}

// joinBlocks joins the non-empty blocks with a blank line and ends the document with one newline.
func joinBlocks(blocks ...string) string {
	kept := make([]string, 0, len(blocks))
	for _, b := range blocks {
		if b != "" {
			kept = append(kept, b)
		}
	}
	return strings.Join(kept, "\n\n") + "\n"
}

// caseMarkdown renders a test case: its title, its metadata list, its text, and the legacy sections Kiwi returned.
func caseMarkdown(doc caseDocument) string {
	sections := []struct{ heading, body string }{
		{"Setup", block(doc.Setup)},
		{"Action", block(doc.Action)},
		{"Expected result", block(doc.Effect)},
		{"Breakdown", block(doc.Breakdown)},
	}
	blocks := []string{
		"# " + title(doc.Summary),
		metadata(
			[2]string{"Status", doc.Status},
			[2]string{"Priority", doc.Priority},
			[2]string{"Category", doc.Category},
		),
		block(doc.Text),
	}
	for _, s := range sections {
		if s.body != "" {
			blocks = append(blocks, "## "+s.heading+"\n\n"+s.body)
		}
	}
	return joinBlocks(blocks...)
}

// planMarkdown renders a test plan: its title, its metadata list, its text, and the list of its cases in order.
func planMarkdown(doc planDocument) string {
	list := "(no test cases)"
	if len(doc.Cases) > 0 {
		lines := make([]string, 0, len(doc.Cases))
		for _, c := range doc.Cases {
			lines = append(lines, "- "+c.ID+": "+title(c.Summary))
		}
		list = strings.Join(lines, "\n")
	}
	return joinBlocks(
		"# "+title(doc.Name),
		metadata([2]string{"Product", doc.Product}, [2]string{"Parent", doc.Parent}),
		block(doc.Text),
		"## Test cases\n\n"+list,
	)
}
