# shellcheck shell=bash
# A Go package-qualified identifier names a type, not a machine. The Go runtime
# writes the standard library's types into release binaries under their import
# path, so a struct field's type reads as an import path, a slash, and an
# identifier that happens to be a suffix word. The `crypto/x509` package's
# unexported `local` type is the known case: the internal-hostname shape read
# the package's name, joined to that suffix word by a dot, as a host, and blocked
# every release binary although it names no machine.
#
# The shape skipped is an import path of lowercase, dot-free segments that
# begins at a token boundary (a space, a brace, a quote, a bracket, or the
# start of the line), then a slash, then the identifier. Any other context
# still blocks: a URL authority or path, a path with a dotted, hyphenated, or
# uppercase segment, a bare name, and a host with a port or a user name.
#
# Every probe is assembled at run time from variables, so this file carries no
# text the gate would read as a hostname. The suffix is `${sfx}`, never the
# literal word after a dot.
run() {
	local out rc
	local sfx=local int=internal pkg=crypto leaf=x509 octet=$((RANDOM % 200 + 1))

	probe_blocks() {
		# probe_blocks <value>: a line naming a host form must block, and the
		# diagnostic must never echo the value.
		local value=$1
		printf 'notes referencing %s in passing\n' "$value" >"$CASE_TMP/probe.txt"
		out=$("$PREFLIGHT" --surface logs --file "$CASE_TMP/probe.txt" 2>&1)
		rc=$?
		assert_exit 1 "$rc" "exit code for a host form" || return 1
		assert_contains "internal-hostname" "$out" "diagnostic for a host form" || return 1
		assert_absent "$value" "$out" "diagnostic for a host form" || return 1
	}

	# The Go-shaped tokens, as the release binaries write them. Each line is
	# a complete struct or symbol fragment, not a bare word.
	{
		printf 'go.shape.struct { %s/%s.%s string; %s/%s.domain string }\n' "$pkg" "$leaf" "$sfx" "$pkg" "$leaf"
		printf '%s/%s.%s[...]\n' "$pkg" "$leaf" "$sfx"
		printf 'symbol internal/abi.%s is a package-level name\n' "$sfx"
	} >"$CASE_TMP/go-identifiers.txt"
	out=$("$PREFLIGHT" --surface logs --file "$CASE_TMP/go-identifiers.txt" 2>&1)
	rc=$?
	assert_exit 0 "$rc" "exit code for Go package-qualified identifiers" || return 1

	# A Go-shaped token beside a real host still blocks: the skip covers the
	# token, never the line or the file it sits in.
	{
		printf '%s/%s.%s\n' "$pkg" "$leaf" "$sfx"
		printf 'reach nas%s.%s for the backup\n' "$octet" "$sfx"
	} >"$CASE_TMP/mixed.txt"
	out=$("$PREFLIGHT" --surface logs --file "$CASE_TMP/mixed.txt" 2>&1)
	rc=$?
	assert_exit 1 "$rc" "exit code for a host beside a Go identifier" || return 1
	assert_contains "internal-hostname" "$out" "diagnostic for a host beside a Go identifier" || return 1

	# Real hosts keep blocking, in every context the rule must still read.
	probe_blocks "db.${sfx}" || return 1
	probe_blocks "nas${octet}.${sfx}" || return 1
	probe_blocks "host.${int}" || return 1
	probe_blocks "https://svc.${sfx}/x" || return 1
	probe_blocks "https://foo/bar.${sfx}" || return 1
	probe_blocks "example.test/bar.${sfx}" || return 1
	probe_blocks "printer.${sfx}:631" || return 1
	probe_blocks "ssh user@box.${sfx}" || return 1
	probe_blocks "the ${leaf}.${sfx} name in prose" || return 1

	# A domain-first import path is not the shape skipped: its first segment
	# reads as a host, so it still blocks rather than passing on a guess.
	probe_blocks "github.com/acme/tool.${sfx}" || return 1
}
