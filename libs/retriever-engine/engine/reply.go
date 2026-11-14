package engine

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"
	"unicode"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// decodeReply decodes one whole JSON value with the caps of the engine: a nesting depth of 64 and 1,048,576 tokens. A
// reply that holds more than one value, a value cut off, or a value past a cap is an unexpected response, and its text
// is never repeated.
func decodeReply(raw []byte) (any, error) {
	return decodeReplyWith(raw, maxJSONDepth, maxJSONNodes)
}

// decodeReplyWith decodes raw as exactly one JSON value, with numbers kept as text so a large integer is not rounded. It
// then measures the value against the caps. The measure counts the tokens that a JSON scanner would see: a container has
// two, the brackets, and each object key has one more.
func decodeReplyWith(raw []byte, depthCap, nodeCap int) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, domain.ErrUnexpectedResponse
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, domain.ErrUnexpectedResponse
	}
	nodes, deepest := shapeOf(v, 0)
	if nodes > nodeCap || deepest > depthCap {
		return nil, domain.ErrUnexpectedResponse
	}
	return v, nil
}

// shapeOf returns the number of tokens of a decoded value, and the deepest level of its containers. A scalar at the top
// is level 0, and each container adds one level to the values inside it.
func shapeOf(v any, depth int) (nodes, deepest int) {
	switch x := v.(type) {
	case []any:
		nodes, deepest = 2, depth+1
		for _, item := range x {
			n, d := shapeOf(item, depth+1)
			nodes += n
			deepest = max(deepest, d)
		}
	case map[string]any:
		nodes, deepest = 2, depth+1
		for _, item := range x {
			n, d := shapeOf(item, depth+1)
			nodes += 1 + n
			deepest = max(deepest, d)
		}
	default:
		return 1, depth
	}
	return nodes, deepest
}

// pointerValue returns the value at a JSON pointer, as RFC 6901 defines it. An empty pointer is the whole document. An
// array index is only digits, with no sign and no leading zero, and a pointer that does not start with a slash finds
// nothing.
func pointerValue(doc any, pointer string) (any, bool) {
	if pointer == "" {
		return doc, true
	}
	if !strings.HasPrefix(pointer, "/") {
		return nil, false
	}
	cur := doc
	for _, raw := range strings.Split(pointer[1:], "/") {
		seg := strings.ReplaceAll(strings.ReplaceAll(raw, "~1", "/"), "~0", "~")
		switch node := cur.(type) {
		case map[string]any:
			next, ok := node[seg]
			if !ok {
				return nil, false
			}
			cur = next
		case []any:
			idx, ok := arrayIndex(seg, len(node))
			if !ok {
				return nil, false
			}
			cur = node[idx]
		default:
			return nil, false
		}
	}
	return cur, true
}

// arrayIndex parses an array index of a pointer, and reports whether it names an item of an array of length n.
func arrayIndex(seg string, n int) (int, bool) {
	if seg == "" || (len(seg) > 1 && seg[0] == '0') {
		return 0, false
	}
	for _, c := range seg {
		if c < '0' || c > '9' {
			return 0, false
		}
	}
	idx, err := strconv.Atoi(seg)
	if err != nil || idx >= n {
		return 0, false
	}
	return idx, true
}

// cleanText removes the control characters of a text, except the tab and the line feed, and charges its length to the
// output budget. The budget is shared by every field of one read, so a read cannot build more text than one reply may
// hold.
func cleanText(s string, budget *int) (string, error) {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsControl(r) && r != '\t' && r != '\n' {
			continue
		}
		b.WriteRune(r)
	}
	out := b.String()
	if len(out) > *budget {
		return "", domain.ErrReplyTooLarge
	}
	*budget -= len(out)
	return out, nil
}

// scalarOf returns the Go value of one JSON scalar: nil, a bool, an int for a whole number, or a text. An object or a
// list is not a field value, so it is an unexpected response.
func scalarOf(v any, budget *int) (any, error) {
	switch x := v.(type) {
	case nil:
		return nil, nil
	case bool:
		return x, nil
	case string:
		return cleanText(x, budget)
	case json.Number:
		if n, err := x.Int64(); err == nil {
			return int(n), nil
		}
		return cleanText(x.String(), budget)
	default:
		return nil, domain.ErrUnexpectedResponse
	}
}

// mapRecord maps one JSON object to a record, with one field for each declared field, in name order.
func mapRecord(fields map[string]spec.Field, obj map[string]any, budget *int) (domain.Record, error) {
	names := sortedKeys(fields)
	out := make([]domain.Field, 0, len(names))
	for _, name := range names {
		raw, _ := pointerValue(obj, fields[name].Pointer)
		v, err := scalarOf(raw, budget)
		if err != nil {
			return domain.Record{}, err
		}
		out = append(out, domain.Field{Name: name, Value: v})
	}
	return domain.NewRecord(out...), nil
}

// listItems maps the list of a reply to records. The items pointer must name a list, and each item must be an object.
func listItems(o spec.Operation, root any, budget *int) ([]domain.Record, error) {
	list, found := pointerValue(root, o.Response.Items)
	arr, isList := list.([]any)
	if !found || !isList {
		return nil, domain.ErrUnexpectedResponse
	}
	out := make([]domain.Record, 0, len(arr))
	for _, item := range arr {
		obj, isObject := item.(map[string]any)
		if !isObject {
			return nil, domain.ErrUnexpectedResponse
		}
		rec, err := mapRecord(o.Response.Fields, obj, budget)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, nil
}

// plainValue returns the Go value of a spec default or example value: a whole number as an int, and any other number as
// its text. Other values pass through.
func plainValue(v any) any {
	if n, ok := v.(json.Number); ok {
		if i, err := n.Int64(); err == nil {
			return int(i)
		}
		return n.String()
	}
	return v
}
