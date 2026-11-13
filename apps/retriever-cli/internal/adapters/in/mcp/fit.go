package mcp

import (
	"encoding/json"
	"unicode/utf8"
)

// fitRows returns the longest prefix of rows whose answer fits within pageCap bytes. render builds the answer from
// the kept rows and the truncated flag, so the flag is counted in the size. truncated is true when a row was left out.
// An answer that cannot be encoded is an error, because a server answer is always plain data.
func fitRows[T any](rows []T, render func(kept []T, truncated bool) any) ([]T, bool, error) {
	for n := len(rows); n >= 0; n-- {
		truncated := n < len(rows)
		raw, err := json.Marshal(render(rows[:n], truncated))
		if err != nil {
			return nil, false, err
		}
		if len(raw) <= pageCap {
			return rows[:n], truncated, nil
		}
	}
	return nil, len(rows) > 0, nil
}

// fitsCap reports whether the JSON answer of v is within pageCap bytes.
func fitsCap(v any) bool {
	raw, err := json.Marshal(v)
	return err == nil && len(raw) <= pageCap
}

// cutToCap returns one object cut to the cap (ADR 006, section 5). wrap builds the answer that the object sits in, so the
// size is the size of the whole answer. Every string longer than the largest limit that still fits is cut to that limit,
// on a character boundary, and the object keeps its shape. When no cut makes it fit, only its scalar fields are kept. The
// caller sets the truncated flag.
func cutToCap(row map[string]any, wrap func(map[string]any) any) (map[string]any, bool) {
	if fitsCap(wrap(row)) {
		return row, false
	}
	best := -1
	lo, hi := 0, longestString(row)
	for lo <= hi {
		mid := (lo + hi) / 2
		if fitsCap(wrap(cutStrings(row, mid).(map[string]any))) {
			best = mid
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}
	if best >= 0 {
		return cutStrings(row, best).(map[string]any), true
	}
	return scalarsOf(row), true
}

// cutStrings returns a copy of v in which every string is cut to at most limit bytes, on a character boundary.
func cutStrings(v any, limit int) any {
	switch x := v.(type) {
	case string:
		return cutText(x, limit)
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, item := range x {
			out[k] = cutStrings(item, limit)
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, item := range x {
			out[i] = cutStrings(item, limit)
		}
		return out
	}
	return v
}

// cutText cuts s to at most limit bytes without splitting a character.
func cutText(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	end := limit
	for end > 0 && !utf8.RuneStart(s[end]) {
		end--
	}
	return s[:end]
}

// longestString returns the length in bytes of the longest string under v.
func longestString(v any) int {
	switch x := v.(type) {
	case string:
		return len(x)
	case map[string]any:
		n := 0
		for _, item := range x {
			n = max(n, longestString(item))
		}
		return n
	case []any:
		n := 0
		for _, item := range x {
			n = max(n, longestString(item))
		}
		return n
	}
	return 0
}

// scalarsOf returns the fields of an object whose value is a scalar, which is the smallest object a row can be cut to.
func scalarsOf(row map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range row {
		switch v.(type) {
		case map[string]any, []any:
			continue
		}
		out[k] = v
	}
	return out
}
