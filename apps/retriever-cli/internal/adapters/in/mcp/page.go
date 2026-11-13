package mcp

import "unicode/utf8"

// pageCap is the most text one get_document answer holds, in bytes. It keeps the answer far below the limits of
// Claude Code for a tool result.
const pageCap = 32768

// cutPage returns the page of markdown that starts at offset. A page holds at most pageCap bytes and never splits a
// character: the end moves back to a character boundary, and an offset inside a character moves forward to the next
// one. Malformed UTF-8 with no boundary inside the cap is cut at the cap, so the page still moves on. next is the offset
// to pass to read on, and truncated reports whether more text follows the page.
func cutPage(markdown string, offset int) (text string, next int, truncated bool) {
	start := offset
	for start < len(markdown) && !utf8.RuneStart(markdown[start]) {
		start++
	}
	end := min(start+pageCap, len(markdown))
	if end < len(markdown) {
		for end > start && !utf8.RuneStart(markdown[end]) {
			end--
		}
	}
	if end == start {
		end = min(start+pageCap, len(markdown)) // no boundary inside the cap: cut at the cap, so the page always moves on
	}
	return markdown[start:end], end, end < len(markdown)
}
