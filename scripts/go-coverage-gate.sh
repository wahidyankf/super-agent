#!/bin/sh
# Coverage gate for one Go module: unit-level statement coverage of the module, with a 99% floor. Every module of the
# workspace runs it on its own, so each library and the app is held to the floor separately.
#
# Usage: run from the module root, as `sh ../../scripts/go-coverage-gate.sh [--check-only]`. COVERAGE_ROOT overrides the
# module root (the self-test uses it), COVERAGE_PACKAGES lists the production packages to measure (default
# "./internal/... ./cmd/...", the app's layout; a library uses "./..."), and COVERAGE_EXCLUSIONS overrides the exclusion
# file.
#
# What is measured. One instrumented run of the untagged tests (-race), with -coverpkg over the production packages. A
# test in one package counts the statements it runs in another, and the Gherkin unit binding (TestUnitBehaviours) is one
# of the untagged tests, so its scenarios count in the app. The integration tests (tag integration) and the end-to-end
# tests (tag e2e) are outside the number.
#
# What is excluded. Only the entries of scripts/coverage-exclusions.tsv, one per line, each with its reason. The only kind
# is "generated": a Go file whose header, before the package clause, says it is generated code. Any other kind, a path
# pattern, a missing path, an entry with no reason, or a generated file without the header fails the gate.
#
# What fails the gate. The run fails when the exclusions are invalid, when a production package has statements but no
# test file (nothing would measure it), when a test fails, when the measured total is below the minimum, and when no
# statement is measured.
#
# Usage details: --check-only validates the exclusions and the package list, then exits without running the tests.
set -eu

minimum="${COVERAGE_MINIMUM:-99}"
root="${COVERAGE_ROOT:-$(pwd)}"
packages="${COVERAGE_PACKAGES:-./internal/... ./cmd/...}"
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
		generated) ;;
		*)
			fail "$where: kind '$kind' is not allowed; only generated may be excluded"
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
		esac
	done <"$exclusions"
	return 0
}

# check_packages refuses a production package that has statements and no test file, because no test binary would
# measure its statements.
check_packages() {
	# shellcheck disable=SC2086 # the package list is split into go list arguments on purpose.
	if ! listed=$(go list -f '{{if and .GoFiles (not .TestGoFiles)}}{{.ImportPath}}{{end}}' $packages); then
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

# The awk program drops each profile block that an exclusion names. A generated entry matches its file.
needles=$(mktemp)
profile=$(mktemp)
trap 'rm -f "$needles" "$profile"' EXIT
while IFS='	' read -r kind path reason _; do
	case "$kind" in
	'' | '#'*) continue ;;
	generated) printf '/%s:\n' "$path" >>"$needles" ;;
	esac
done <"$exclusions"

coverpkg=$(printf '%s' "$packages" | tr ' ' ',')
go test -race -count=1 -coverpkg="$coverpkg" -coverprofile="$profile" ./...

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
