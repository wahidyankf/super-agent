#!/bin/sh
# Coverage gate for apps/retriever-cli. It measures statement coverage of every production package in the module, with
# unit and integration tests inside the gate. Each test binary counts the statements it runs in every internal and cmd
# package (-coverpkg), so a behaviour test that drives an adapter counts for that adapter. End-to-end tests (tag e2e) are
# outside the gate: they build the binary and run it as a process, so they are run by the test:e2e target.
#
# The only exclusions are these, each with its reason:
#   internal/adapters/graphql/generated.go, models_gen.go, schema.resolvers.go: gqlgen output with a "Code generated"
#     header. gqlgen regenerates them, and the graphql integration tests run them.
#   tests/support/: test doubles, stub servers, and stub commands that serve the behaviour suites. They are test code, not
#     product behaviour, and the suites that use them are the measurement.
# A package with no statements (for example the interface-only internal/core/ports) adds nothing to the total.
set -eu

minimum="${COVERAGE_MINIMUM:-99}"
root=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
cd "$root"

profile=$(mktemp)
trap 'rm -f "$profile"' EXIT

go test -count=1 -tags integration -coverpkg=./internal/...,./cmd/... -coverprofile="$profile" ./...

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
