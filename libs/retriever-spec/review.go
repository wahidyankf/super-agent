package spec

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// The caps of the engine, which a spec may only lower. The review names each cap that a spec leaves to its default.
const (
	capTimeoutSeconds = 30
	capReplyBytes     = 10485760
	capPages          = 20
)

// Review returns the plain-text review of a valid spec: the endpoint, the authentication, the limits, and one entry for
// each operation with its label, method, path, writes, and surfaces. It names each variable by its name and never shows
// a value of the spec, so a person reads what the spec can do before the engine runs it.
func Review(s *Spec) string {
	var b strings.Builder
	fmt.Fprintf(&b, "connector spec %s, version %d\n", s.Kind, s.Version)
	fmt.Fprintf(&b, "summary: %s\n", s.Summary)
	fmt.Fprintf(&b, "endpoint: %s\n", endpointLine(s.Endpoint))
	fmt.Fprintf(&b, "auth: %s\n", authPhrase(s.Auth))
	fmt.Fprintf(&b, "limits: %s\n", limitsLine(s.Limits))
	fmt.Fprintf(&b, "operations: %d\n", len(s.Operations))
	for i, op := range s.Operations {
		fmt.Fprintf(&b, "%d. %s: %s, %s %s%s, writes %s, cli %s, mcp %s\n", i+1, op.ID, op.Access, op.Method,
			s.Endpoint.BasePath, op.Path, yesNo(op.Access != "read"), surfaceWord(op.CLI), surfaceWord(op.MCP))
		fmt.Fprintf(&b, "   %s\n", op.Summary)
	}
	writePostReads(&b, s)
	writeBodyPaging(&b, s)
	writeRequiredFields(&b, s)
	writeNestedObjects(&b, s)
	writeBoolMaps(&b, s)
	writeFirstReads(&b, s)
	writeMergedLists(&b, s)
	writeTreeReads(&b, s)
	writeFallbackReads(&b, s)
	writeIssueKeys(&b, s)
	writePageEnds(&b, s)
	writeKeyEchoes(&b, s)
	writeBuiltIDs(&b, s)
	writeBundledMapping(&b, s)
	return b.String()
}

// writePostReads writes the block of the reads that are sent as POST, one line each, with the path and the slots of the
// body. A spec with no POST read writes nothing, so its review keeps its text.
func writePostReads(b *strings.Builder, s *Spec) {
	var posts []Operation
	for _, op := range s.Operations {
		if op.ReadOverPost {
			posts = append(posts, op)
		}
	}
	if len(posts) == 0 {
		return
	}
	fmt.Fprintf(b, "POST reads: %d\n", len(posts))
	for i, op := range posts {
		fmt.Fprintf(b, "%d. %s: POST %s%s, body from slots: %s\n", i+1, op.ID, s.Endpoint.BasePath, op.Path, slotsOf(op.Body))
	}
}

// writeBodyPaging writes the block of the reads that page in the JSON body, one line each, with the style, the body
// parameters the rule names, and the page cap. A spec with no body-paged read writes nothing, so its review keeps its
// text.
func writeBodyPaging(b *strings.Builder, s *Spec) {
	var body []Operation
	for _, op := range s.Operations {
		if op.Pagination != nil && op.Pagination.In == "body" {
			body = append(body, op)
		}
	}
	if len(body) == 0 {
		return
	}
	fmt.Fprintf(b, "Body paging: %d\n", len(body))
	for i, op := range body {
		if op.Pagination.Style == "page-all" {
			fmt.Fprintf(b, "%d. %s: page-all in body (page size %d), at most %d pages\n", i+1, op.ID,
				op.Pagination.PageSize, op.Pagination.MaxPages)
			continue
		}
		fmt.Fprintf(b, "%d. %s: %s in body (%s), at most %d pages\n", i+1, op.ID, op.Pagination.Style,
			strings.Join(pagingParams(op.Pagination), ", "), op.Pagination.MaxPages)
	}
}

// writeRequiredFields writes the block of the operations with required or format-checked reply fields, one line each,
// with the required fields and each checked field with its closed format name. The fields of each list of a merge
// join the fields of their operation. A spec with no required or checked field writes nothing, so its review keeps
// its text.
func writeRequiredFields(b *strings.Builder, s *Spec) {
	type entry struct {
		id       string
		required []string
		formats  []string
	}
	var entries []entry
	for _, op := range s.Operations {
		e := entry{id: op.ID}
		required := map[string]bool{}
		formats := map[string]bool{}
		add := func(fields map[string]Field) {
			var names []string
			for name := range fields {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, name := range names {
				f := fields[name]
				if f.Required {
					required[name] = true
				}
				if f.Format != "" {
					formats[name+"="+f.Format] = true
				}
			}
		}
		add(op.Response.Fields)
		if op.Response.Merge != nil {
			for _, l := range op.Response.Merge.Lists {
				add(l.Fields)
			}
		}
		for name := range required {
			e.required = append(e.required, name)
		}
		for f := range formats {
			e.formats = append(e.formats, f)
		}
		sort.Strings(e.required)
		sort.Strings(e.formats)
		if len(e.required) > 0 || len(e.formats) > 0 {
			entries = append(entries, e)
		}
	}
	if len(entries) == 0 {
		return
	}
	fmt.Fprintf(b, "Required fields: %d\n", len(entries))
	for i, e := range entries {
		var parts []string
		if len(e.required) > 0 {
			parts = append(parts, "required "+strings.Join(e.required, ", "))
		}
		if len(e.formats) > 0 {
			parts = append(parts, "formats "+strings.Join(e.formats, ", "))
		}
		fmt.Fprintf(b, "%d. %s: %s\n", i+1, e.id, strings.Join(parts, "; "))
	}
}

// writeNestedObjects writes the block of the operations with nested record fields, one line each, with the object
// field and each sub-field. The object fields of each list of a merge join the fields of their operation, with the
// sub-fields of every list that names the field. A spec with no object field writes nothing, so its review keeps its
// text.
func writeNestedObjects(b *strings.Builder, s *Spec) {
	type entry struct {
		id    string
		field string
		subs  []string
	}
	byKey := map[string]*entry{}
	var order []string
	add := func(id, field string, object map[string]Field) {
		key := id + "\x00" + field
		e, ok := byKey[key]
		if !ok {
			e = &entry{id: id, field: field}
			byKey[key] = e
			order = append(order, key)
		}
		have := map[string]bool{}
		for _, sub := range e.subs {
			have[sub] = true
		}
		var names []string
		for sub := range object {
			names = append(names, sub)
		}
		sort.Strings(names)
		for _, sub := range names {
			if !have[sub] {
				e.subs = append(e.subs, sub)
				have[sub] = true
			}
		}
	}
	addFields := func(id string, fields map[string]Field) {
		var names []string
		for name := range fields {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			if fields[name].Object == nil {
				continue
			}
			add(id, name, fields[name].Object)
		}
	}
	for _, op := range s.Operations {
		addFields(op.ID, op.Response.Fields)
		if op.Response.Merge != nil {
			for _, l := range op.Response.Merge.Lists {
				addFields(op.ID, l.Fields)
			}
		}
	}
	if len(order) == 0 {
		return
	}
	fmt.Fprintf(b, "Nested objects: %d\n", len(order))
	for i, key := range order {
		e := byKey[key]
		fmt.Fprintf(b, "%d. %s: %s with %s\n", i+1, e.id, e.field, strings.Join(e.subs, ", "))
	}
}

// writeBoolMaps writes the block of the operations with an open-key map with closed bool values, one line each,
// with the map field and its pointer. A spec with no bool-map field writes nothing, so its review keeps its text.
func writeBoolMaps(b *strings.Builder, s *Spec) {
	type entry struct {
		id      string
		field   string
		pointer string
	}
	var entries []entry
	addFields := func(id string, fields map[string]Field) {
		var names []string
		for name := range fields {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			if !fields[name].BoolMap {
				continue
			}
			entries = append(entries, entry{id: id, field: name, pointer: fields[name].Pointer})
		}
	}
	for _, op := range s.Operations {
		addFields(op.ID, op.Response.Fields)
		if op.Response.Merge != nil {
			for _, l := range op.Response.Merge.Lists {
				addFields(op.ID, l.Fields)
			}
		}
	}
	if len(entries) == 0 {
		return
	}
	fmt.Fprintf(b, "Open bool maps: %d\n", len(entries))
	for i, e := range entries {
		fmt.Fprintf(b, "%d. %s: %s keeps every bool at %s, sorted\n", i+1, e.id, e.field, e.pointer)
	}
}

// writeFirstReads writes the block of the reads that map the first item of a list, one line each, with the pointer
// of the list. A spec with no first read writes nothing, so its review keeps its text.
func writeFirstReads(b *strings.Builder, s *Spec) {
	type entry struct {
		id      string
		pointer string
	}
	var entries []entry
	for _, op := range s.Operations {
		if op.Response.First == "" {
			continue
		}
		entries = append(entries, entry{id: op.ID, pointer: op.Response.First})
	}
	if len(entries) == 0 {
		return
	}
	fmt.Fprintf(b, "First reads: %d\n", len(entries))
	for i, e := range entries {
		fmt.Fprintf(b, "%d. %s: first of %s\n", i+1, e.id, e.pointer)
	}
}

// writeMergedLists writes the block of the reads that merge lists, one line each, with the label field, the number
// of lists, and each list with its constant label and its items pointer. A spec with no merge read writes nothing,
// so its review keeps its text.
func writeMergedLists(b *strings.Builder, s *Spec) {
	type entry struct {
		id         string
		labelField string
		lists      []string
	}
	var entries []entry
	for _, op := range s.Operations {
		if op.Response.Merge == nil {
			continue
		}
		e := entry{id: op.ID, labelField: op.Response.Merge.LabelField}
		for _, l := range op.Response.Merge.Lists {
			e.lists = append(e.lists, l.Label+" of "+l.Items)
		}
		entries = append(entries, e)
	}
	if len(entries) == 0 {
		return
	}
	fmt.Fprintf(b, "Merged lists: %d\n", len(entries))
	for i, e := range entries {
		fmt.Fprintf(b, "%d. %s: merge %s from %d lists (%s)\n", i+1, e.id, e.labelField, len(e.lists), strings.Join(e.lists, ", "))
	}
}

// writeTreeReads writes the block of the reads that flatten a tree, one line each, with the items pointer, the
// children pointer, the computed fields, the root parent parameter when the read names one, and the depth cap. A
// spec with no tree read writes nothing, so its review keeps its text.
func writeTreeReads(b *strings.Builder, s *Spec) {
	type entry struct {
		id         string
		items      string
		children   string
		parent     string
		depth      string
		rootParent string
		maxDepth   int
	}
	var entries []entry
	for _, op := range s.Operations {
		if op.Response.Tree == nil {
			continue
		}
		t := op.Response.Tree
		entries = append(entries, entry{id: op.ID, items: op.Response.Items, children: t.Children,
			parent: t.ParentField, depth: t.DepthField, rootParent: t.RootParentParam,
			maxDepth: t.EffectiveMaxDepth()})
	}
	if len(entries) == 0 {
		return
	}
	fmt.Fprintf(b, "Tree reads: %d\n", len(entries))
	for i, e := range entries {
		roots := ""
		if e.rootParent != "" {
			roots = ", roots under " + e.rootParent
		}
		fmt.Fprintf(b, "%d. %s: tree of %s with children %s, parent %s, depth %s%s, at most depth %d\n",
			i+1, e.id, e.items, e.children, e.parent, e.depth, roots, e.maxDepth)
	}
}

// writeFallbackReads writes the block of the reads with a declared fallback, one line each, with the target
// operation, the closed trigger, and the fill fields. A spec with no fallback read writes nothing, so its review
// keeps its text.
func writeFallbackReads(b *strings.Builder, s *Spec) {
	type entry struct {
		id     string
		target string
		when   string
		fill   string
	}
	var entries []entry
	for _, op := range s.Operations {
		if op.Fallback == nil {
			continue
		}
		cond := op.Fallback.When.Empty + " is empty"
		if len(op.Fallback.When.Status) > 0 {
			var statuses []string
			for _, status := range op.Fallback.When.Status {
				statuses = append(statuses, strconv.Itoa(status))
			}
			cond = "status [" + strings.Join(statuses, ", ") + "]"
		}
		entries = append(entries, entry{id: op.ID, target: op.Fallback.Operation, when: cond,
			fill: strings.Join(op.Fallback.Fill, ", ")})
	}
	if len(entries) == 0 {
		return
	}
	fmt.Fprintf(b, "Fallback reads: %d\n", len(entries))
	for i, e := range entries {
		fmt.Fprintf(b, "%d. %s: fallback to %s when %s, filling %s\n", i+1, e.id, e.target, e.when, e.fill)
	}
}

// writeIssueKeys writes the block of the reads with an IssueKey path parameter, one line each, with the parameter
// that carries the key. A spec with no issue key writes nothing, so the review keeps its text.
func writeIssueKeys(b *strings.Builder, s *Spec) {
	type entry struct {
		id    string
		param string
	}
	var entries []entry
	for _, op := range s.Operations {
		for _, p := range op.Params {
			if p.Type == "IssueKey" && p.In == "path" {
				entries = append(entries, entry{id: op.ID, param: p.Name})
			}
		}
	}
	if len(entries) == 0 {
		return
	}
	fmt.Fprintf(b, "Issue keys: %d\n", len(entries))
	for i, e := range entries {
		fmt.Fprintf(b, "%d. %s: path parameter %s\n", i+1, e.id, e.param)
	}
}

// writePageEnds writes the block of the reads with the page-end rule, one line each, with the last flag, the cut
// parameter, and the truncated pointers. A spec with no page end writes nothing, so the review keeps its text.
func writePageEnds(b *strings.Builder, s *Spec) {
	type entry struct {
		id   string
		rule string
	}
	var entries []entry
	for _, op := range s.Operations {
		var parts []string
		if op.Pagination != nil {
			if op.Pagination.LastPointer != "" {
				parts = append(parts, "last of "+op.Pagination.LastPointer)
			}
			if op.Pagination.CutToLimit != "" {
				parts = append(parts, "cut to "+op.Pagination.CutToLimit)
			}
		}
		if op.Response.Truncated != nil {
			parts = append(parts, "truncated of "+op.Response.Truncated.Token+" and "+op.Response.Truncated.Last)
		}
		if len(parts) == 0 {
			continue
		}
		entries = append(entries, entry{id: op.ID, rule: strings.Join(parts, "; ")})
	}
	if len(entries) == 0 {
		return
	}
	fmt.Fprintf(b, "Page ends: %d\n", len(entries))
	for i, e := range entries {
		fmt.Fprintf(b, "%d. %s: %s\n", i+1, e.id, e.rule)
	}
}

// writeKeyEchoes writes the block of the reads with the key echo, one line each, with the parameter and the pointer
// the reply must echo. A spec with no echo writes nothing, so its review keeps its text.
func writeKeyEchoes(b *strings.Builder, s *Spec) {
	type entry struct {
		id      string
		param   string
		pointer string
	}
	var entries []entry
	for _, op := range s.Operations {
		if op.Response.Echo == nil {
			continue
		}
		entries = append(entries, entry{id: op.ID, param: op.Response.Echo.Param, pointer: op.Response.Echo.Pointer})
	}
	if len(entries) == 0 {
		return
	}
	fmt.Fprintf(b, "Key echoes: %d\n", len(entries))
	for i, e := range entries {
		fmt.Fprintf(b, "%d. %s: %s equals %s\n", i+1, e.id, e.param, e.pointer)
	}
}

// builtIDKind names the ID that a built field builds, for the review line.
func builtIDKind(f Field) string {
	switch {
	case f.ProjectKey != "":
		return "project key"
	case f.IssueID != "":
		return "issue ID"
	case f.CollectionID != "":
		return "collection ID"
	default:
		return "browse URL"
	}
}

// builtIDPointer names the pointer that a built field reads the key from.
func builtIDPointer(f Field) string {
	switch {
	case f.ProjectKey != "":
		return f.ProjectKey
	case f.IssueID != "":
		return f.IssueID
	case f.CollectionID != "":
		return f.CollectionID
	default:
		return f.BrowseURL
	}
}

// writeBuiltIDs writes the block of the operations with built IDs, one line each, with every built field, the ID it
// builds, and the pointer it builds it from, in field-name order. A spec with no built ID writes nothing, so its
// review keeps its text.
func writeBuiltIDs(b *strings.Builder, s *Spec) {
	type entry struct {
		id     string
		fields []string
	}
	var entries []entry
	addFields := func(id string, fields map[string]Field) []string {
		var names []string
		for name := range fields {
			names = append(names, name)
		}
		sort.Strings(names)
		var parts []string
		for _, name := range names {
			f := fields[name]
			if f.ProjectKey == "" && f.IssueID == "" && f.CollectionID == "" && f.BrowseURL == "" {
				continue
			}
			parts = append(parts, name+" as "+builtIDKind(f)+" of "+builtIDPointer(f))
		}
		return parts
	}
	for _, op := range s.Operations {
		parts := addFields(op.ID, op.Response.Fields)
		if op.Response.Merge != nil {
			for _, l := range op.Response.Merge.Lists {
				parts = append(parts, addFields(op.ID, l.Fields)...)
			}
		}
		if len(parts) == 0 {
			continue
		}
		sort.Strings(parts)
		entries = append(entries, entry{id: op.ID, fields: parts})
	}
	if len(entries) == 0 {
		return
	}
	fmt.Fprintf(b, "Built IDs: %d\n", len(entries))
	for i, e := range entries {
		fmt.Fprintf(b, "%d. %s: %s\n", i+1, e.id, strings.Join(e.fields, "; "))
	}
}

// writeBundledMapping writes the block of the bundled ID mapping, one line each, with the catalog row that a
// claimed operation serves, and with the surfaces of an operation that no row names. A spec with no claimed row
// writes nothing, so the review of a user spec keeps its text.
func writeBundledMapping(b *strings.Builder, s *Spec) {
	claimed := false
	for _, op := range s.Operations {
		if op.CatalogID != "" {
			claimed = true
			break
		}
	}
	if !claimed {
		return
	}
	fmt.Fprintf(b, "Bundled mapping: %d\n", len(s.Operations))
	for i, op := range s.Operations {
		if op.CatalogID == "" {
			fmt.Fprintf(b, "%d. %s: no catalog row, cli %s, mcp %s\n", i+1, op.ID, surfaceWord(op.CLI),
				surfaceWord(op.MCP))
			continue
		}
		fmt.Fprintf(b, "%d. %s: catalog %s\n", i+1, op.ID, op.CatalogID)
	}
}

// pagingParams returns the names of the paging parameters of a rule, in sorted order.
func pagingParams(pag *Pagination) []string {
	var out []string
	for _, name := range []string{pag.LimitParam, pag.OffsetParam, pag.CursorParam} {
		if name != "" {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// slotsOf names the slots of a body in sorted order, or "none" when the body takes no slot.
func slotsOf(body map[string]any) string {
	names := map[string]bool{}
	bodySlotNames(body, names)
	if len(names) == 0 {
		return "none"
	}
	var list []string
	for name := range names {
		list = append(list, name)
	}
	sort.Strings(list)
	return strings.Join(list, ", ")
}

// endpointLine names the path prefix and the host suffixes. A spec with no host suffix binds to any host of the catalog.
func endpointLine(e Endpoint) string {
	base := e.BasePath
	if base == "" {
		base = "none"
	}
	if len(e.HostSuffix) == 0 {
		return "base path " + base + ", hosts: any host of the catalog"
	}
	return "base path " + base + ", hosts ending in " + strings.Join(e.HostSuffix, ", ")
}

// authPhrase names the authentication kind and the variables that hold the key and the account e-mail.
func authPhrase(a Auth) string {
	switch a.Type {
	case "basic":
		return fmt.Sprintf("basic, key variable %s, account variable %s", a.CredentialEnv, a.IdentityEnv)
	case "header":
		return fmt.Sprintf("header %s, key variable %s", a.HeaderName, a.CredentialEnv)
	}
	return fmt.Sprintf("bearer, key variable %s", a.CredentialEnv)
}

// limitsLine states each limit: the value the spec sets, or the cap that applies when the spec sets none.
func limitsLine(l Limits) string {
	timeout := fmt.Sprintf("timeout %ds (cap)", capTimeoutSeconds)
	if l.TimeoutSeconds > 0 {
		timeout = fmt.Sprintf("timeout %ds", l.TimeoutSeconds)
	}
	reply := fmt.Sprintf("reply %d bytes (cap)", capReplyBytes)
	if l.MaxReplyBytes > 0 {
		reply = fmt.Sprintf("reply %d bytes", l.MaxReplyBytes)
	}
	pages := fmt.Sprintf("at most %d pages (cap)", capPages)
	if l.MaxPages > 0 {
		pages = fmt.Sprintf("at most %d pages", l.MaxPages)
	}
	return strings.Join([]string{timeout, reply, pages}, ", ")
}

// surfaceWord names a surface: off, on, or on under a custom name.
func surfaceWord(s Surface) string {
	switch {
	case !s.Enabled:
		return "off"
	case s.Name != "":
		return "as " + s.Name
	}
	return "on"
}

// yesNo returns yes or no.
func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
