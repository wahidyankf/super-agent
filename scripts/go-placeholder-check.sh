#!/bin/sh
# Placeholder ban for one Go module (C18): no test skips, focuses, marks a step pending, or passes without checking.
# Usage: run from the module root, as `sh ../../scripts/go-placeholder-check.sh`, or name the module root as the one
# argument, as `sh scripts/go-placeholder-check.sh <module root>`.
#
# What is banned, in every _test.go file: a testing skip (t.Skip, t.Skipf, t.SkipNow, and the same on b and tb), a
# Ginkgo focus call (FIt, FDescribe, FContext, FWhen, FSpecify, FEntry, FDescribeTable), a godog pending step
# (ErrPending), a placeholder assertion (assert.True(t, true) or require.True(t, true)), and a test function whose body
# is empty. The scan reads tagged files too: a placeholder in an integration file reports a pass just as well.
#
# The hermetic rules are a separate check: golangci-lint runs them through the .golangci.yml the module loads. This
# script covers the placeholders that a linter pattern cannot state on its own.
set -eu
cd "${1:-.}"

found=0
report() {
	echo "placeholder check: $1" >&2
	found=1
}

skips=$(grep -rn --include='*_test.go' -E '\.Skip(Now|f)?\(' . || true)
if [ -n "$skips" ]; then
	report "a testing skip is a placeholder (C18); remove it:"
	echo "$skips" >&2
fi

focus=$(grep -rn --include='*_test.go' -E '\bF(It|Describe|Context|When|Specify|Entry|DescribeTable)\(' . || true)
if [ -n "$focus" ]; then
	report "a focus marker is a placeholder (C18); remove it:"
	echo "$focus" >&2
fi

pending=$(grep -rn --include='*_test.go' 'ErrPending' . || true)
if [ -n "$pending" ]; then
	report "a pending godog step is a placeholder (C18); implement the step:"
	echo "$pending" >&2
fi

trivial=$(grep -rn --include='*_test.go' -E '(assert|require)\.True\(t, true\)' . || true)
if [ -n "$trivial" ]; then
	report "an assertion that is always true checks nothing; remove it:"
	echo "$trivial" >&2
fi

# -z reads each file as one record, so a body that spans lines still matches. -l names the file.
empty=$(grep -rlPz --include='*_test.go' 'func Test\w+\(\w+ \*testing\.T\) \{\s*\}' . || true)
if [ -n "$empty" ]; then
	report "a test with an empty body checks nothing; give it a check:"
	echo "$empty" >&2
fi

if [ "$found" -ne 0 ]; then
	exit 1
fi
echo "placeholder check: no skip, focus, pending step, trivial assertion, or empty test body"
