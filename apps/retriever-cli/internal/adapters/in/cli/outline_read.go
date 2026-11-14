package cli

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/sources"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/sources/operations"
)

// The read commands of the outline group are generated from the operation catalog. A row gives the
// command path, its flags and positional argument, its help, and the shape of its output. attachOperations adds a
// command for every row, and runRead runs any of them the same way. A command here parses nothing by hand.

// commandTree returns the full retriever command tree: the base table, with the generated commands of each kind's group and
// the tree alias of the outline group.
func commandTree() *command {
	tree := baseCommandTree()
	outline := tree.child("outline")
	attachOperations(outline, domain.KindOutline)
	outline.children = append(outline.children, treeAlias())
	attachOperations(tree.child("kiwi"), domain.KindKiwi)
	attachOperations(tree.child("jira"), domain.KindJira)
	return tree
}

// treeAlias is "outline tree", the name a collection's document tree had before the catalog. It runs the
// collections.documents row, so its answer and its errors are the same.
func treeAlias() *command {
	row, _ := operations.Lookup(domain.OpCollectionsDocuments)
	return &command{name: "tree", summary: "show the document tree of one Outline collection", leaf: readLeaf(row, "outline tree")}
}

// legacyRunnableGroup is the one group that is also a command: "outline collections" runs its list row, as it did before
// the catalog.
const legacyRunnableGroup = "collections"

// attachOperations adds one command for every catalog row of the kind under its group, the command whose first word is the
// kind's name. A path word that no row has yet is a group, so "outline documents" and "kiwi test-cases" appear with their
// subcommands. The rows of one kind never join another kind's group.
func attachOperations(group *command, kind domain.SourceKind) {
	for _, row := range operations.RowsOf(kind) {
		words := row.Path()[1:]
		node := group
		for i, word := range words {
			next := node.child(word)
			if next == nil {
				next = &command{name: word, summary: kind.Display() + " " + word + " commands"}
				node.children = append(node.children, next)
			}
			if i == len(words)-1 {
				next.summary = row.Summary
				next.leaf = readLeaf(row, row.CLI)
				if word == "list" && node.name == legacyRunnableGroup {
					// "outline collections" is the command of today, and it runs the list. No other group is runnable, so "outline documents" stays a group.
					node.summary = row.Summary
					node.leaf = readLeaf(row, strings.TrimSuffix(row.CLI, " list"))
				}
			}
			node = next
		}
	}
}

// readLeaf returns the leaf that runs one catalog row. The usage line names path, which is the row's own path except for
// an alias such as "outline tree". A row of a non-read effect is a write: its leaf names its source as required, and
// runs through the write use case, or the admin use case for an admin row.
func readLeaf(row operations.Row, path string) *leaf {
	kind, _ := row.Kind()
	positional := row.Positional()
	var names, required []string
	var flags []flag
	var usage strings.Builder
	usage.WriteString("retriever " + path)
	if positional != "" {
		names = []string{positional}
		usage.WriteString(" <" + positional + ">")
	}
	for _, p := range row.Params {
		if p.Name == positional {
			continue
		}
		f := flag{name: "--" + kebab(p.Name), meta: metaOf(p), help: p.Help}
		flags = append(flags, f)
		if p.Required {
			// The first required parameter is the positional one. Any other required parameter is a flag the command
			// cannot run without, so the usage line shows it bare, and parse names it when it is missing.
			required = append(required, f.name)
			usage.WriteString(" " + f.name + " " + f.meta)
			continue
		}
		usage.WriteString(" [" + f.name + " " + f.meta + "]")
	}
	source := sourceFlagFor(kind)
	run := func(inv *invocation) error { return runRead(inv, row) }
	if row.Effect != domain.EffectRead {
		source = writeSourceFlag
		run = func(inv *invocation) error { return runWrite(inv, row) }
	}
	if row.Effect != domain.EffectRead && row.Access == domain.AccessAdmin {
		// An admin command runs through the admin use case.
		run = func(inv *invocation) error { return runAdmin(inv, row) }
	}
	flags = append(flags, source)
	usage.WriteString(" [--source <id>]")
	return &leaf{
		usage:      usage.String(),
		positional: names,
		flags:      flags,
		required:   required,
		noMatch:    row.NoMatch,
		run:        run,
	}
}

// kebab turns a camelCase parameter name into its flag name, such as collectionId into collection-id.
func kebab(name string) string {
	var b strings.Builder
	for i, r := range name {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				b.WriteByte('-')
			}
			b.WriteRune(r + ('a' - 'A'))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// metaOf returns the placeholder a flag shows for a parameter of the given kind.
func metaOf(p operations.Param) string {
	switch p.Kind {
	case operations.KindLimit, operations.KindOffset, operations.KindInt:
		return "<n>"
	case operations.KindScore:
		return "<-1..1>"
	case operations.KindCollectionID, operations.KindDocumentID, operations.KindID, operations.KindCollectionUUID,
		operations.KindShareID:
		return "<id>"
	case operations.KindUUID:
		return "<uuid>"
	case operations.KindUUIDList:
		return "<uuid,...>"
	case operations.KindEnum:
		return "<" + strings.Join(p.Values, "|") + ">"
	case operations.KindEnumList:
		return "<" + strings.Join(p.Values, "|") + ",...>"
	}
	return "<value>"
}

// rawArgs collects the values the command line gave for one row, under the catalog names. A flag the caller did not
// give is left out, so the binder applies its default.
func rawArgs(inv *invocation, row operations.Row) map[string]any {
	raw := map[string]any{}
	positional := row.Positional()
	for _, p := range row.Params {
		if p.Name == positional {
			if len(inv.positional) > 0 {
				raw[p.Name] = inv.positional[0]
			}
			continue
		}
		flagName := "--" + kebab(p.Name)
		if value, ok := inv.values[flagName]; ok {
			raw[p.Name] = value
		}
	}
	return raw
}

// runRead runs one catalog row against the selected source and prints its Record in the requested format. A search
// that matches nothing exits 1, after it has printed the empty result (CLI contract 001).
func runRead(inv *invocation, row operations.Row) error {
	rt := inv.rt
	kind, _ := row.Kind()
	service, err := rt.outlineWiring()
	if err != nil {
		return withKind(kind, err)
	}
	rec, err := sources.Read(context.Background(), service, row.ID, sourceArg(inv), rawArgs(inv, row))
	if err != nil {
		return withKind(kind, err)
	}
	if rt.format == domain.FormatJSON {
		if err := writeJSON(rt.stdout, jsonDocument(row, rec)); err != nil {
			return err
		}
	} else {
		writeText(rt.stdout, row, rec)
	}
	if row.NoMatch && len(rec.Items()) == 0 {
		return exitStatus{code: exitNegative}
	}
	return nil
}

// jsonDocument returns the JSON document of a Record: the schema version, then either the object's fields or, for a
// list, the count and the list under the row's key.
func jsonDocument(row operations.Row, rec domain.Record) map[string]any {
	doc := map[string]any{"schemaVersion": schemaVersion}
	if !rec.IsList() {
		for k, v := range recordMap(rec) {
			doc[k] = v
		}
		return doc
	}
	items := make([]any, 0, len(rec.Items()))
	for _, item := range rec.Items() {
		items = append(items, recordMap(item))
	}
	doc["count"] = len(items)
	doc[row.ListKey] = items
	if page, ok := rec.Page(); ok {
		if row.Notice != "" {
			doc["truncated"] = page.HasMore
		} else {
			doc["has_more"] = page.HasMore
			if page.HasMore {
				doc["next_offset"] = page.NextOffset
			}
		}
	}
	return doc
}

// writeContinuation names the offset of the next page of a paged list that may hold more rows. A short page names none. A
// row with a notice prints the notice instead, because its list has no offset to go on to.
func writeContinuation(w io.Writer, row operations.Row, rec domain.Record) {
	page, ok := rec.Page()
	if !ok || !page.HasMore {
		return
	}
	if row.Notice != "" {
		fmt.Fprintln(w, row.Notice)
		return
	}
	fmt.Fprintf(w, "More rows may follow. Next page: --offset %d\n", page.NextOffset)
}

// recordMap returns the fields of a Record as a map, with nested records as maps too.
func recordMap(rec domain.Record) map[string]any {
	out := map[string]any{}
	for _, f := range rec.Fields() {
		if nested, ok := f.Value.(domain.Record); ok {
			out[f.Name] = recordMap(nested)
			continue
		}
		out[f.Name] = f.Value
	}
	return out
}

// writeText prints a Record as text. A row with a body prints only that field, so a document is its Markdown. A list
// prints its count line and one row per item, or, for a tree, one indented line per item. An object prints one
// "name: value" line per field, with nested fields named by their path, such as user.name.
func writeText(w io.Writer, row operations.Row, rec domain.Record) {
	switch {
	case row.Body != "":
		text, _ := rec.Get(row.Body)
		s := scalarText(text)
		if !strings.HasSuffix(s, "\n") {
			s += "\n"
		}
		fmt.Fprint(w, s)
	case row.IsList() && row.Indent != "":
		writeTree(w, row, rec.Items())
	case row.IsList():
		writeRows(w, row, rec.Items())
		writeContinuation(w, row, rec)
	default:
		writeFields(w, "", rec)
	}
}

// maxIndentDepth is the deepest level a tree line is indented to. A deeper node is printed at this level, so a hostile
// depth never makes a line of unbounded width.
const maxIndentDepth = 16

// writeTree prints a tree as one line per item, indented by its depth. An empty tree says so.
func writeTree(w io.Writer, row operations.Row, items []domain.Record) {
	if len(items) == 0 {
		fmt.Fprintf(w, "No %s\n", row.Noun)
		return
	}
	for _, item := range items {
		depth, _ := item.Get(row.Indent)
		n, _ := depth.(int)
		n = min(max(n, 0), maxIndentDepth)
		fmt.Fprintf(w, "%s%s\n", strings.Repeat("  ", n), joinColumns(row, item, row.Columns, "  "))
	}
}

// writeRows prints the count line of a list, then one row per item. The first column is padded to the widest value,
// so the other columns line up. A quiet row prints nothing for an empty list, and a row with a detail field prints that
// field's text under its line.
func writeRows(w io.Writer, row operations.Row, items []domain.Record) {
	if row.Quiet && len(items) == 0 {
		return
	}
	fmt.Fprintf(w, "%s\n", countLine(row.Noun, len(items)))
	width := 0
	for _, item := range items {
		width = max(width, len(columnValue(item, row.Columns[0])))
	}
	for _, item := range items {
		first := columnValue(item, row.Columns[0])
		fmt.Fprintf(w, "  %-*s  %s\n", width, first, joinColumns(row, item, row.Columns[1:], "  "))
		if row.Detail != "" {
			writeDetail(w, item, row.Detail)
		}
	}
}

// writeDetail prints the text of one field under a row, each line indented by four spaces, so a comment's Markdown follows
// the line that names it. A field with no text prints nothing.
func writeDetail(w io.Writer, item domain.Record, field string) {
	value, _ := item.Get(field)
	text := strings.TrimRight(scalarText(value), "\n")
	if text == "" {
		return
	}
	for _, line := range strings.Split(text, "\n") {
		fmt.Fprintln(w, strings.TrimRight("    "+line, " "))
	}
}

// countLine returns "1 collection" or "2 collections", in the noun's singular or plural.
func countLine(noun string, n int) string {
	if n == 1 {
		return "1 " + strings.TrimSuffix(noun, "s")
	}
	return strconv.Itoa(n) + " " + noun
}

// joinColumns returns the values of the named columns of one item, joined by sep.
func joinColumns(row operations.Row, item domain.Record, columns []string, sep string) string {
	parts := make([]string, 0, len(columns))
	for _, c := range columns {
		parts = append(parts, columnValue(item, c))
	}
	return strings.Join(parts, sep)
}

// columnValue returns one field of an item as text. A missing or null field is empty.
func columnValue(item domain.Record, name string) string {
	v, _ := item.Get(name)
	return scalarText(v)
}

// writeFields prints the fields of an object, one "name: value" line each. A nested object is printed with its own
// field names after a dot. A field name is sanitized as a value is: some names are the server's own, such as the event
// types of a user's notification settings (finding M2). JSON output keeps the names as they are.
func writeFields(w io.Writer, prefix string, rec domain.Record) {
	for _, f := range rec.Fields() {
		name := sanitizeTerminal(f.Name)
		if nested, ok := f.Value.(domain.Record); ok {
			writeFields(w, prefix+name+".", nested)
			continue
		}
		fmt.Fprintf(w, "%s%s: %s\n", prefix, name, scalarText(f.Value))
	}
}

// scalarText returns a scalar field value as text, with its terminal controls removed. nil is the empty text.
func scalarText(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return sanitizeTerminal(x)
	case int:
		return strconv.Itoa(x)
	case bool:
		return strconv.FormatBool(x)
	}
	return sanitizeTerminal(fmt.Sprint(v))
}

// sanitizeTerminal replaces each terminal control in wiki text with U+FFFD, so a title or a body cannot set the
// terminal title, clear the screen, or move the cursor. A control is a C0 control, DEL, or a C1 control. A line feed and a
// tab stay, because a body keeps its lines and its indentation. JSON output does not use it: a JSON string escapes its
// controls already.
func sanitizeTerminal(s string) string {
	clean := true
	for _, r := range s {
		if isTerminalControl(r) {
			clean = false
			break
		}
	}
	if clean {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if isTerminalControl(r) {
			b.WriteRune(utf8.RuneError)
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// isTerminalControl reports whether a rune would be acted on by a terminal and is not a line feed or a tab.
func isTerminalControl(r rune) bool {
	switch {
	case r == '\n' || r == '\t':
		return false
	case r < 0x20, r == 0x7f, r >= 0x80 && r <= 0x9f:
		return true
	}
	return false
}
