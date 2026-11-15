package spec

import (
	"fmt"
	"sort"
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
		fmt.Fprintf(b, "%d. %s: %s in body (%s), at most %d pages\n", i+1, op.ID, op.Pagination.Style,
			strings.Join(pagingParams(op.Pagination), ", "), op.Pagination.MaxPages)
	}
}

// writeRequiredFields writes the block of the operations with required or format-checked reply fields, one line each,
// with the required fields and each checked field with its closed format name. A spec with no required or checked
// field writes nothing, so its review keeps its text.
func writeRequiredFields(b *strings.Builder, s *Spec) {
	type entry struct {
		id       string
		required []string
		formats  []string
	}
	var entries []entry
	for _, op := range s.Operations {
		e := entry{id: op.ID}
		var names []string
		for name := range op.Response.Fields {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			f := op.Response.Fields[name]
			if f.Required {
				e.required = append(e.required, name)
			}
			if f.Format != "" {
				e.formats = append(e.formats, name+"="+f.Format)
			}
		}
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
