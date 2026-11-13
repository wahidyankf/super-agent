package domain

// Args is the validated input of one catalog operation (ADR 007 section 2). Only the operation catalog builds one, from
// the parameter rules of its row, so a value in Args has passed its kind's check. A parameter that the caller did not
// set is absent; its getter returns the zero value.
type Args struct {
	values map[string]any
}

// NewArgs returns the Args that holds a copy of values. The caller is the catalog's binder, which has checked every
// value. Values are string, int, bool, or []string.
func NewArgs(values map[string]any) Args {
	return Args{values: copyValues(values)}
}

// Values returns a copy of the set values, keyed by parameter name. A list is copied too, so a caller that changes it
// does not change the Args. The binder uses it to check an Args again.
func (a Args) Values() map[string]any {
	return copyValues(a.values)
}

// copyValues copies the map and its lists, so no later change to the input or to a returned list reaches an Args.
func copyValues(values map[string]any) map[string]any {
	out := make(map[string]any, len(values))
	for k, v := range values {
		if list, ok := v.([]string); ok {
			v = copyStrings(list)
		}
		out[k] = v
	}
	return out
}

// copyStrings returns a copy of list that is nil only when list is nil, so an empty list stays an empty list.
func copyStrings(list []string) []string {
	if list == nil {
		return nil
	}
	out := make([]string, len(list))
	copy(out, list)
	return out
}

// Has reports whether the parameter is set.
func (a Args) Has(name string) bool {
	_, ok := a.values[name]
	return ok
}

// String returns the text of a string parameter, or "" when it is absent.
func (a Args) String(name string) string {
	s, _ := a.values[name].(string)
	return s
}

// Int returns the value of an integer parameter, or 0 when it is absent.
func (a Args) Int(name string) int {
	n, _ := a.values[name].(int)
	return n
}

// Bool returns the value of a boolean parameter, or false when it is absent.
func (a Args) Bool(name string) bool {
	b, _ := a.values[name].(bool)
	return b
}

// Strings returns the items of a list parameter, or nil when it is absent.
func (a Args) Strings(name string) []string {
	s, _ := a.values[name].([]string)
	return copyStrings(s)
}

// Field is one named value of a Record. A value is a string, an int, a bool, nil (JSON null), or a Record.
type Field struct {
	Name  string
	Value any
}

// Record is the neutral result of one catalog operation (ADR 007 section 2, item 3): an ordered list of named fields
// for one object, or an ordered list of records for a list operation. It holds no wire type, so the renderers and the
// MCP adapter read it the same way for every operation.
type Record struct {
	fields []Field
	items  []Record
	list   bool
	page   Page
}

// Page is the continuation of one page of a paged list (ADR 007 section 3.4). A paged list carries its offset, and its
// hint says whether more rows may follow and at which offset they start. A full page may be followed by more rows, so
// HasMore is true when the page holds as many rows as its limit; a short page is the last one.
type Page struct {
	Paged      bool
	Offset     int
	HasMore    bool
	NextOffset int
}

// NewPage returns the continuation of a page that starts at offset, holds count rows, and was asked for at most limit
// rows. A page with no limit is never continued.
func NewPage(offset, limit, count int) Page {
	p := Page{Paged: true, Offset: offset, NextOffset: offset + count}
	p.HasMore = limit > 0 && count >= limit
	return p
}

// WithPage returns the record with the continuation of its page. A record that is not a list keeps no page.
func (r Record) WithPage(p Page) Record {
	r.page = p
	return r
}

// Page returns the continuation of the page, and false when the record is not a paged list.
func (r Record) Page() (Page, bool) {
	return r.page, r.page.Paged
}

// NewRecord returns a record of the fields, in the order given.
func NewRecord(fields ...Field) Record {
	return Record{fields: append([]Field(nil), fields...)}
}

// NewList returns a list record of items, in the order given. A nil list is an empty list.
func NewList(items []Record) Record {
	if items == nil {
		items = []Record{}
	}
	return Record{items: items, list: true}
}

// IsList reports whether the record is a list of items.
func (r Record) IsList() bool {
	return r.list
}

// Items returns the items of a list record. A record that is not a list has none.
func (r Record) Items() []Record {
	return r.items
}

// Fields returns the fields of an object record, in order.
func (r Record) Fields() []Field {
	return append([]Field(nil), r.fields...)
}

// Get returns the value of the named field and whether the field exists.
func (r Record) Get(name string) (any, bool) {
	for _, f := range r.fields {
		if f.Name == name {
			return f.Value, true
		}
	}
	return nil, false
}

// Str returns the text of a string field, or "" when the field is absent or is not a string.
func (r Record) Str(name string) string {
	v, _ := r.Get(name)
	s, _ := v.(string)
	return s
}

// Rec returns the nested record of a field, or the zero record when the field is absent.
func (r Record) Rec(name string) Record {
	v, _ := r.Get(name)
	nested, _ := v.(Record)
	return nested
}
