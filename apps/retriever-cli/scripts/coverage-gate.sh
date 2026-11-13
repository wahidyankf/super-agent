#!/bin/sh
# Coverage gate for apps/retriever-cli: unit-level statement coverage of the whole module, with a 99% floor.
#
# What is measured. One instrumented run of the untagged tests (-race), with -coverpkg over every production package under
# internal/ and cmd/. A test in one package counts the statements it runs in another, and the Gherkin unit binding
# (TestUnitBehaviours) is one of the untagged tests, so its scenarios count. The integration tests (tag integration) and the
# end-to-end tests (tag e2e) are outside the number.
#
# What is excluded. Only the entries of scripts/coverage-exclusions.tsv, one per line, each with its reason. The file
# allows two kinds: "generated" (a Go file whose header, before the package clause, says it is generated code) and
# "test-support" (the tests/support directory). Any other kind, a path pattern, a missing path, an entry with no reason,
# or a generated file without the header fails the gate.
#
# What fails the gate. The run fails when the exclusions are invalid, when a production package has statements but no
# test file (nothing would measure it), when a test fails, when the measured total is below the minimum, and when no
# statement is measured.
#
# Usage: coverage-gate.sh [--check-only]. --check-only validates the exclusions and the package list, then exits without
# running the tests. COVERAGE_ROOT and COVERAGE_EXCLUSIONS override the module root and the exclusion file; the self-test
# uses them.
set -eu

minimum="${COVERAGE_MINIMUM:-99}"
root="${COVERAGE_ROOT:-$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)}"
exclusions="${COVERAGE_EXCLUSIONS:-$root/scripts/coverage-exclusions.tsv}"
cd "$root"

problems=0
fail() {
	echo "coverage gate: $*" >&2
	problems=$((problems + 1))
}

# check_exclusions reads the exclusion file. Each line is kind, path, and reason, separated by tabs. A tab is a field
# separator for read, so an empty reason field leaves reason empty, and the line fails the reason check.
check_exclusions() {
	if [ ! -f "$exclusions" ]; then
		fail "the exclusion file $exclusions is missing"
		return 0
	fi
	line_no=0
	while IFS='	' read -r kind path reason extra || [ -n "${kind:-}" ]; do
		line_no=$((line_no + 1))
		case "$kind" in
		'' | '#'*) continue ;;
		esac
		where="$exclusions:$line_no"
		case "$kind" in
		generated | test-support) ;;
		*)
			fail "$where: kind '$kind' is not allowed; only generated and test-support may be excluded"
			continue
			;;
		esac
		if [ -n "${extra:-}" ]; then
			fail "$where: an entry has four fields; the form is kind, path, reason"
			continue
		fi
		if [ -z "$(printf '%s' "${reason:-}" | tr -d ' ')" ]; then
			fail "$where: $path has no reason; each exclusion needs its reason in the third field"
			continue
		fi
		case "$path" in
		'' | /* | *..* | *'*'* | *'?'* | *'['*)
			fail "$where: '$path' is not a plain relative path; a path pattern is never an exclusion"
			continue
			;;
		esac
		case "$kind" in
		generated)
			case "$path" in
			*.go) ;;
			*)
				fail "$where: generated entry $path is not a Go file"
				continue
				;;
			esac
			if [ ! -f "$path" ]; then
				fail "$where: $path does not exist"
				continue
			fi
			# Go's convention: the generated-code line must come before the package clause.
			if ! sed '/^package /q' "$path" | grep -q '^// Code generated .* DO NOT EDIT\.$'; then
				fail "$where: $path has no 'Code generated ... DO NOT EDIT.' header before its package clause, so it is not generated"
			fi
			;;
		test-support)
			if [ "$path" != "tests/support" ] || [ ! -d "$path" ]; then
				fail "$where: a test-support entry must be the existing directory tests/support, not $path"
			fi
			;;
		esac
	done <"$exclusions"
	return 0
}

# check_packages refuses a production package that has statements and no test file, because no test binary would
# measure its statements.
check_packages() {
	if ! listed=$(go list -f '{{if and .GoFiles (not .TestGoFiles)}}{{.ImportPath}}{{end}}' ./internal/... ./cmd/...); then
		fail "go list could not read the production packages"
		return 0
	fi
	unmeasured=$(printf '%s\n' "$listed" | sed '/^$/d')
	if [ -n "$unmeasured" ]; then
		fail "these production packages have no test file, so nothing measures them:"
		echo "$unmeasured" >&2
	fi
	return 0
}

check_exclusions
check_packages
if [ "$problems" -gt 0 ]; then
	echo "coverage gate: $problems problem(s) with the exclusions or the package list" >&2
	exit 1
fi
if [ "${1:-}" = "--check-only" ]; then
	echo "coverage gate: the exclusions and the package list are valid"
	exit 0
fi

# The awk program drops each profile block that an exclusion names. A generated entry matches its file, and the
# test-support entry matches its directory.
needles=$(mktemp)
profile=$(mktemp)
trap 'rm -f "$needles" "$profile"' EXIT
while IFS='	' read -r kind path reason _; do
	case "$kind" in
	'' | '#'*) continue ;;
	generated) printf '/%s:\n' "$path" >>"$needles" ;;
	test-support) printf '/%s/\n' "$path" >>"$needles" ;;
	esac
done <"$exclusions"

go test -race -count=1 -coverpkg=./internal/...,./cmd/... -coverprofile="$profile" ./...

awk -v minimum="$minimum" -v needles="$needles" '
  FILENAME == needles { needle[++n] = $0; next }
  # A profile line is "file:start.col,end.col numStatements count". The first line is the mode.
  FNR == 1 { next }
  {
    for (i = 1; i <= n; i++) {
      if (index($1, needle[i]) > 0) {
        if (!($1 in skipped)) { skipped[$1] = 1; excluded++ }
        next
      }
    }
    statements[$1] = $2
    if ($3 > 0) { hit[$1] = 1 }
  }
  END {
    for (block in statements) {
      total += statements[block]
      if (block in hit) { covered += statements[block] }
    }
    if (total == 0) {
      print "coverage gate: no production statements were measured" > "/dev/stderr"
      exit 1
    }
    percent = covered * 100 / total
    printf "coverage gate: %d of %d statements covered (%.2f%%), minimum %s%%, %d excluded blocks\n", covered, total, percent, minimum, excluded
    if (percent + 0 < minimum + 0) {
      print "coverage gate: below the minimum" > "/dev/stderr"
      exit 1
    }
  }
' "$needles" "$profile"
