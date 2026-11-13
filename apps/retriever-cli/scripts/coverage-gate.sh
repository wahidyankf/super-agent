#!/bin/sh
# Coverage gate for apps/retriever-cli: unit-level statement coverage of the whole module, with a 99% floor.
#
# What is measured. One instrumented run of the untagged tests (-race), with -coverpkg over every production package under
# internal/ and cmd/. A test in one package counts the statements it runs in another, and the Gherkin unit binding
# (TestUnitBehaviours) is one of the untagged tests, so its scenarios count. The integration tests (tag integration) and the
# end-to-end tests (tag e2e) are outside the number.
#
# What is excluded. Only these, each with its reason:
#   internal/adapters/graphql/generated.go, models_gen.go, schema.resolvers.go: gqlgen output with a "Code generated" header.
#     gqlgen regenerates them, and the graphql unit and integration tests run them.
#   tests/support/: test doubles, stub servers, and stub commands that the behaviour suites run. They are test code, not
#     product behaviour, and the suites that use them are the measurement.
#
# What fails the gate. The run fails when a test fails, when the measured total is below the minimum, when there is no
# measured statement, and when a production package has statements but no test file, because nothing would measure it.
set -eu

minimum="${COVERAGE_MINIMUM:-99}"
root=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
cd "$root"

# A production package with statements and no test file runs no test binary, so its statements would be missing from the
# profile. go list names those packages, and the gate refuses them.
unmeasured=$(go list -f '{{if and .GoFiles (not .TestGoFiles)}}{{.ImportPath}}{{end}}' ./internal/... ./cmd/... | sed '/^$/d')
if [ -n "$unmeasured" ]; then
  echo "coverage gate: these production packages have no test file, so nothing measures them:" >&2
  echo "$unmeasured" >&2
  exit 1
fi

profile=$(mktemp)
trap 'rm -f "$profile"' EXIT

go test -race -count=1 -coverpkg=./internal/...,./cmd/... -coverprofile="$profile" ./...

awk -v minimum="$minimum" '
  # A profile line is "file:start.col,end.col numStatements count". The first line is the mode.
  NR == 1 { next }
  $1 ~ /\/apps\/retriever-cli\/(tests\/support\/|internal\/adapters\/graphql\/(generated|models_gen|schema\.resolvers)\.go:)/ {
    if (!($1 in skipped)) { skipped[$1] = 1; excluded++ }
    next
  }
  {
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
' "$profile"
