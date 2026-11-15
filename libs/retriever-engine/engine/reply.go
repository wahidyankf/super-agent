package engine

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// decodeReply decodes one whole JSON value with the caps of the engine: a nesting depth of 64 and 1,048,576 tokens. A
// reply that holds more than one value, a value cut off, or a value past a cap is an unexpected response, and its text
// is never repeated. A reply with a tree decodes to the higher tree cap instead; see decodeReplyFor.
func decodeReply(raw []byte) (any, error) {
	return decodeReplyWith(raw, maxJSONDepth, maxJSONNodes)
}

// decodeReplyFor decodes one whole JSON value with the caps of the operation: a reply with a tree decodes to the
// fixed higher depth of treeJSONDepth, because each tree level takes two JSON levels, and any other reply decodes to
// the caps of the engine. No spec field sets the higher cap, and no other reply changes.
func decodeReplyFor(o spec.Operation, raw []byte) (any, error) {
	if o.Response.Tree != nil {
		return decodeReplyWith(raw, treeJSONDepth, maxJSONNodes)
	}
	return decodeReply(raw)
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

// derefPointers returns the value that one field maps: the value at its pointer, or at its fallback pointer when the
// pointer finds nothing, so one field maps the wrapped and the unwrapped shapes of a reply. A pointer that finds an
// explicit null wins, so the fallback serves only the shape that lacks the pointer, never a second guess inside one
// shape.
func derefPointers(obj map[string]any, f spec.Field) (any, bool) {
	if raw, found := pointerValue(obj, f.Pointer); found {
		return raw, true
	}
	if f.FallbackPointer != "" {
		return pointerValue(obj, f.FallbackPointer)
	}
	return nil, false
}

// mapRecord maps one JSON object to a record, with one field for each declared field, in name order. A required field
// with a missing, null, or empty value fails the read, and a format-checked field whose value is not of the named
// shape fails it too. Either failure names the field and never the value, and no partial record is returned. A field
// that holds an object maps one nested record instead, with the same rules for each declared sub-field; a pointer that
// reaches an object or a list is refused, so a whole object is never copied. The reply tree is dropped after mapping:
// no value of a record aliases it.
func mapRecord(fields map[string]spec.Field, obj map[string]any, budget *int) (domain.Record, error) {
	names := sortedKeys(fields)
	out := make([]domain.Field, 0, len(names))
	for _, name := range names {
		f := fields[name]
		if f.BoolMap {
			nested, err := mapBoolMap(name, f, obj)
			if err != nil {
				return domain.Record{}, err
			}
			out = append(out, domain.Field{Name: name, Value: nested})
			continue
		}
		if f.Object != nil {
			nested, err := mapObject(name, f.Object, obj, budget)
			if err != nil {
				return domain.Record{}, err
			}
			out = append(out, domain.Field{Name: name, Value: nested})
			continue
		}
		raw, found := derefPointers(obj, f)
		if f.Required && isMissingValue(raw, found) {
			return domain.Record{}, fieldError(name)
		}
		v, err := fieldValue(f, raw, budget)
		if err != nil {
			return domain.Record{}, err
		}
		if err := checkFormat(name, f.Format, v); err != nil {
			return domain.Record{}, err
		}
		out = append(out, domain.Field{Name: name, Value: v})
	}
	return domain.NewRecord(out...), nil
}

// fieldValue maps one reply value to a field value: a strings field reads a
// list of strings, an editor-data converter reads the plain text of editor JSON,
// any other field reads one scalar, and a hostname converter keeps the host of
// an address. A value the reply holds as another shape is an unexpected
// response, which names the field and never the value.
func fieldValue(f spec.Field, raw any, budget *int) (any, error) {
	if f.Format == "strings" {
		return stringsValue(raw, budget)
	}
	if f.Convert == "text-from-editor-data" {
		return editorText(raw, budget)
	}
	v, err := scalarOf(raw, budget)
	if err != nil {
		return nil, err
	}
	if f.Convert == "hostname" {
		return hostOf(v), nil
	}
	return v, nil
}

// stringsValue maps a reply list to the texts it holds, in order. A missing
// or null value maps to null, like any other missing optional field. A value
// that is not a list, or an element that is not a string, is an unexpected
// response, so a whole list is never copied. Each text is charged to the
// shared output budget, like any mapped text.
func stringsValue(raw any, budget *int) (any, error) {
	if raw == nil {
		return nil, nil
	}
	arr, ok := raw.([]any)
	if !ok {
		return nil, domain.ErrUnexpectedResponse
	}
	out := make([]any, 0, len(arr))
	for _, item := range arr {
		s, ok := item.(string)
		if !ok {
			return nil, domain.ErrUnexpectedResponse
		}
		c, err := cleanText(s, budget)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

// hostOf returns the host name of an address, without the path, the query,
// the port, or the user name. An address that does not parse, or that names
// no host, maps to the empty text, so a broken address never reaches the
// answer. A missing value maps to the empty text too, as the Go connectors
// read a missing address.
func hostOf(v any) any {
	s, ok := v.(string)
	if !ok {
		return ""
	}
	u, err := url.Parse(s)
	if err != nil || u.Host == "" {
		return ""
	}
	return u.Hostname()
}

// mapObject maps one nested record field to a record, with one value for each declared sub-field, in name order. Each
// sub-field follows the scalar rules, and its failure names the dotted path, never the value. A pointer that reaches
// an object or a list is refused, so a whole object is never copied.
func mapObject(field string, sub map[string]spec.Field, obj map[string]any, budget *int) (domain.Record, error) {
	names := sortedKeys(sub)
	out := make([]domain.Field, 0, len(names))
	for _, name := range names {
		f := sub[name]
		path := field + "." + name
		raw, found := derefPointers(obj, f)
		if f.Required && isMissingValue(raw, found) {
			return domain.Record{}, fieldError(path)
		}
		v, err := fieldValue(f, raw, budget)
		if err != nil {
			return domain.Record{}, fieldError(path)
		}
		if err := checkFormat(path, f.Format, v); err != nil {
			return domain.Record{}, err
		}
		out = append(out, domain.Field{Name: name, Value: v})
	}
	return domain.NewRecord(out...), nil
}

// mapBoolMap maps one open-key map with closed bool values to a record, keeping every key the reply holds, verbatim,
// in sorted order, like the Go projection of the Outline notification writes. Each value must be a flag: a value that
// is not a bool fails the read as an unexpected response naming the dotted path, never dropped. A missing or null map
// answers an empty record, unless the field is required. A value that is not an object fails naming the field.
func mapBoolMap(field string, f spec.Field, obj map[string]any) (domain.Record, error) {
	raw, found := derefPointers(obj, f)
	if !found || raw == nil {
		if f.Required {
			return domain.Record{}, fieldError(field)
		}
		return domain.NewRecord(), nil
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return domain.Record{}, fieldError(field)
	}
	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]domain.Field, 0, len(names))
	for _, name := range names {
		on, ok := m[name].(bool)
		if !ok {
			return domain.Record{}, fieldError(field + "." + name)
		}
		out = append(out, domain.Field{Name: name, Value: on})
	}
	return domain.NewRecord(out...), nil
}

// isMissingValue reports whether a reply holds no value for a required field: the pointer names nothing, or the value
// is null or empty text.
func isMissingValue(raw any, found bool) bool {
	return !found || raw == nil || raw == ""
}

// checkFormat checks a mapped value against the closed format name of its field: id, uuid, share-id, or strings, with
// the checks of the operation catalog. A null value has no shape to check, so only a present value is checked. The
// failure names the field and never the value.
func checkFormat(name, format string, v any) error {
	if format == "" || v == nil {
		return nil
	}
	if format == "strings" {
		return checkStrings(name, v)
	}
	s, ok := v.(string)
	if !ok {
		return fieldError(name)
	}
	var err error
	switch format {
	case "id":
		err = domain.ValidateID(s)
	case "uuid":
		err = domain.ValidateUUID(s)
	case "share-id":
		err = domain.ValidateShareID(s)
	default:
		return fieldError(name)
	}
	if err != nil {
		return fieldError(name)
	}
	return nil
}

// checkStrings checks a mapped value against the strings format: every element
// of the list is a string. A value the mapper built always passes, so this
// guards the shape, never the content.
func checkStrings(name string, v any) error {
	arr, ok := v.([]any)
	if !ok {
		return fieldError(name)
	}
	for _, item := range arr {
		if _, ok := item.(string); !ok {
			return fieldError(name)
		}
	}
	return nil
}

// fieldError returns the unexpected-response failure of one reply field. It names the field and never the value, so a
// value the reply carries never reaches a message. It matches ErrUnexpectedResponse, so its code stays
// unexpected_response on every surface.
func fieldError(name string) error {
	return fmt.Errorf("field %s: %w", name, domain.ErrUnexpectedResponse)
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

// firstRecord maps the first item of the list at the first pointer to one record, with the field pointers read
// relative to the item. An empty list ends with domain.ErrNotFound, and no partial record is returned. A value that
// is not a list, or an item that is not an object, is an unexpected response.
func firstRecord(o spec.Operation, root any, budget *int) (domain.Record, error) {
	list, found := pointerValue(root, o.Response.First)
	arr, isList := list.([]any)
	if !found || !isList {
		return domain.Record{}, domain.ErrUnexpectedResponse
	}
	if len(arr) == 0 {
		return domain.Record{}, domain.ErrNotFound
	}
	obj, isObject := arr[0].(map[string]any)
	if !isObject {
		return domain.Record{}, domain.ErrUnexpectedResponse
	}
	return mapRecord(o.Response.Fields, obj, budget)
}

// mergeRecords maps the lists of a merge read to one list, in the declared list order and then in the reply order.
// Each item is mapped with the fields of its own list, and the label field is set to the constant label of the list,
// beating a reply field of the same name. The output budget is shared by every list, so a merge cannot build more
// output than one list may hold.
func mergeRecords(o spec.Operation, root any, budget *int) ([]domain.Record, error) {
	m := o.Response.Merge
	out := make([]domain.Record, 0)
	for _, l := range m.Lists {
		list, found := pointerValue(root, l.Items)
		arr, isList := list.([]any)
		if !found || !isList {
			return nil, domain.ErrUnexpectedResponse
		}
		for _, item := range arr {
			obj, isObject := item.(map[string]any)
			if !isObject {
				return nil, domain.ErrUnexpectedResponse
			}
			rec, err := mapRecord(l.Fields, obj, budget)
			if err != nil {
				return nil, err
			}
			rec, err = withMergeLabel(rec, m.LabelField, l.Label, budget)
			if err != nil {
				return nil, err
			}
			out = append(out, rec)
		}
	}
	return out, nil
}

// withComputedField sets a computed field on a mapped record, beating a mapped reply field of the same name. Text
// is charged to the shared output budget, like any mapped text, so a computed copy cannot build more output than one
// reply may hold.
func withComputedField(rec domain.Record, name string, value any, budget *int) (domain.Record, error) {
	if s, isText := value.(string); isText {
		if len(s) > *budget {
			return domain.Record{}, domain.ErrReplyTooLarge
		}
		*budget -= len(s)
	}
	fields := rec.Fields()
	for i, f := range fields {
		if f.Name == name {
			fields[i].Value = value
			return domain.NewRecord(fields...), nil
		}
	}
	fields = append(fields, domain.Field{Name: name, Value: value})
	sort.Slice(fields, func(i, j int) bool { return fields[i].Name < fields[j].Name })
	return domain.NewRecord(fields...), nil
}

// withMergeLabel sets the label field of a merged record to the constant label of its list, beating a mapped reply
// field of the same name. The label is charged to the shared output budget, like any mapped text.
func withMergeLabel(rec domain.Record, name, label string, budget *int) (domain.Record, error) {
	return withComputedField(rec, name, label, budget)
}

// treeRecords flattens the tree at the items pointer depth first, computing the parent and the depth of each node,
// and only these two fields. Each node maps with the fields of the response, and the computed fields beat a mapped
// reply field of the same name. The parent of a node is the id of its parent node: null for a top node, or the value
// of the root parent parameter when the read names one. The depth counts from 0. A node deeper than the depth cap,
// and the nodes below it, is left out, as the Go connector does today. A missing or null children value ends the
// branch, but a children value that is not a list, or a child that is not an object, is an unexpected response. The
// walk is a flat loop with no recursion, and the output shares the budget of any list, so a deep or wide tree cannot
// build without bound.
func treeRecords(o spec.Operation, args domain.Args, root any, budget *int) ([]domain.Record, error) {
	list, found := pointerValue(root, o.Response.Items)
	arr, isList := list.([]any)
	if !found || !isList {
		return nil, domain.ErrUnexpectedResponse
	}
	t := o.Response.Tree
	var rootParent any
	if t.RootParentParam != "" {
		rootParent, _ = args.Values()[t.RootParentParam]
	}
	maxDepth := t.EffectiveMaxDepth()
	type frame struct {
		node   map[string]any
		parent any
		depth  int
	}
	stack := make([]frame, 0, len(arr))
	for i := len(arr) - 1; i >= 0; i-- {
		obj, isObject := arr[i].(map[string]any)
		if !isObject {
			return nil, domain.ErrUnexpectedResponse
		}
		stack = append(stack, frame{node: obj, depth: 0})
	}
	var out []domain.Record
	for len(stack) > 0 {
		top := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if top.depth > maxDepth {
			continue
		}
		rec, err := mapRecord(o.Response.Fields, top.node, budget)
		if err != nil {
			return nil, err
		}
		parent := top.parent
		if top.depth == 0 {
			parent = rootParent
		}
		if rec, err = withComputedField(rec, t.ParentField, parent, budget); err != nil {
			return nil, err
		}
		if rec, err = withComputedField(rec, t.DepthField, top.depth, budget); err != nil {
			return nil, err
		}
		out = append(out, rec)
		id, _ := rec.Get("id")
		children, found, err := treeChildren(top.node, t.Children)
		if err != nil {
			return nil, err
		}
		if !found {
			continue
		}
		for i := len(children) - 1; i >= 0; i-- {
			obj, isObject := children[i].(map[string]any)
			if !isObject {
				return nil, domain.ErrUnexpectedResponse
			}
			stack = append(stack, frame{node: obj, parent: id, depth: top.depth + 1})
		}
	}
	return out, nil
}

// treeChildren returns the child nodes at the children pointer of one tree node. A missing or null value ends the
// branch with no children. A value that is not a list is an unexpected response.
func treeChildren(node map[string]any, pointer string) ([]any, bool, error) {
	raw, found := pointerValue(node, pointer)
	if !found || raw == nil {
		return nil, false, nil
	}
	children, isList := raw.([]any)
	if !isList {
		return nil, false, domain.ErrUnexpectedResponse
	}
	return children, true, nil
}

// pageItems maps one page of a paged read: a tree page flattens depth first, a merged page merges its lists, and any
// other page maps its list. A merged read concatenates the merged items of its pages in page order.
func pageItems(o spec.Operation, args domain.Args, root any, budget *int) ([]domain.Record, error) {
	if o.Response.Tree != nil {
		return treeRecords(o, args, root, budget)
	}
	if o.Response.Merge != nil {
		return mergeRecords(o, root, budget)
	}
	return listItems(o, root, budget)
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
