package spec

import (
	"fmt"
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
	return b.String()
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
