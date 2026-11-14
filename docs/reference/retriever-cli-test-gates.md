# Retriever CLI test gates

This page records what each test target of `apps/retriever-cli` runs, what the coverage gate measures and excludes,
what the binding check enforces, and which targets CI runs on each platform. It states facts. The rules that govern them
live in the repository's standards.

## Targets

| Target             | Runs                                                                                                           |
| ------------------ | -------------------------------------------------------------------------------------------------------------- |
| `lint`             | `go vet` with the integration and e2e tags, then a `gofmt` check                                               |
| `typecheck`        | `go build ./...`, then compiles every test package with those tags, without running                            |
| `test:unit`        | a check that no `_test.go` file calls `.Skip(`, the gate self-test, the gate, then the release build self-test |
| `test:quick`       | `lint`, `typecheck`, then `test:unit`                                                                          |
| `test:integration` | `go test -count=1 -tags integration ./...`                                                                     |
| `test:e2e`         | `go test -count=1 -tags e2e ./tests/behaviour/...`; the suite builds the binary first                          |
| `test:conformance` | `go test -count=1 -tags integration ./tests/conformance/... ./tests/architecture/...`                          |
| `bdd:check`        | the binding check of `specs/behaviours`, described below                                                       |
| `test:coverage`    | `scripts/coverage-gate.sh`, the same gate as `test:unit` without the skip check                                |
| `build`            | `go build -trimpath -o dist/retriever ./cmd/retriever`                                                         |
| `build:release`    | `scripts/build-release.sh`: the four platform archives and `SHA256SUMS` of one tag                             |
| `deps:audit`       | `go tool govulncheck ./...`, which reads vuln.go.dev and needs the network                                     |

## Coverage gate

`test:unit` and `test:coverage` run `scripts/coverage-gate.sh`. `test:unit` runs the gate's self-test,
`scripts/coverage-gate_test.sh`, first. The gate measures unit-level statement coverage of the whole module:

- It makes one instrumented run of the untagged tests, with `-race`.
- It measures every production package under `internal/` and `cmd/`. A test in one package counts the statements it runs
  in another.
- The Gherkin unit binding, `TestUnitBehaviours`, is untagged, so every `@unit` scenario counts.
- The integration tests (tag `integration`) and the end-to-end tests (tag `e2e`) are outside the number. The
  integration and e2e targets run them.
- It computes statement coverage over the merged profile. The floor is 99 percent, and the `COVERAGE_MINIMUM`
  environment variable can change it.

It fails when a test fails, when no statement is measured, and when a production package has statements but no test
file, because nothing would measure that package. Its `--check-only` option validates the exclusions and the package
list without running any test.

### Exclusions

The exclusions are data in `scripts/coverage-exclusions.tsv`. Each line holds a kind, a path, and a reason, separated by
tabs. Two kinds are allowed:

- `generated`: a Go file whose `Code generated ... DO NOT EDIT.` line comes before its package clause. Two entries
  qualify, `internal/adapters/graphql/generated.go` and `models_gen.go`, which gqlgen writes. The graphql unit and
  integration tests run them.
- `test-support`: the `tests/support` directory of test doubles, stub servers, and stub commands. The behaviour suites
  use them, and they are test code.

The gate fails on any other kind, on an entry with no reason, on a path pattern, on a path that does not exist, on a
`generated` file without the header before its package clause, and on a production package with no test file.
`scripts/coverage-gate_test.sh` runs each of these cases against fixture modules.

`internal/adapters/graphql/schema.resolvers.go` is not excluded. Its gqlgen line comes after the package clause, so under
the Go convention it is not a generated-code file, and the gate counts its statements. A file without the header is
counted even when its name says generated. The MCP tool table, `internal/adapters/in/mcp/operation_tools.go`, was named
`generated.go` before. It is hand-written, and the gate counts it.

A package with no statements, such as the interface-only `internal/core/ports`, adds nothing to the total.

### What unit tests may do

Unit tests run in process, beside their package, and they do not reach outside the process. An untagged unit test does
not read or write a file, does not read or change the process environment, does not change the working directory, does
not start a child process, and does not open a pipe. It may open a loopback `httptest` server. The registry tests have
no build tag, so they run at the unit level.

The file system, the environment, and the child processes reach the code through seams: a field or a parameter that
holds a function or an interface. The production code takes the seam, and a unit test supplies a fake. Examples are the
`Files` of the catalog and the credentials, the `folderOps` of the Markdown sink, the `getenv` parameter of `run` in
`cmd/retriever`, and the `readFile` parameter of each source's `TestSeamFromEnvWithFiles`. A test that needs the real file
system, the real environment, or a real child process carries the `integration` build tag, in a file named
`*_integration_test.go`, or the `integration || e2e` tag for a shared file. Those tests run in `test:integration` and
`test:conformance`.

The behaviour harness follows the same rule. Its unit level, `TestUnitBehaviours`, runs the 333 `@unit` scenarios in
memory. Each scenario keeps its files in an in-memory tree (`tests/support/memfs`), reads the working directory and the
environment from the scenario, writes exports to an in-memory sink, and serves the stub certificates from memory. The
synthetic Outline replies are embedded from `tests/support/synthetic`. The integration level, `TestIntegrationBehaviours`,
and the end-to-end level run the same step definitions on the real disk: with the real environment at the integration
level, and as the built binary at the end-to-end level. The disk backend, and the code that starts a process, sit in
`tests/behaviour/backend_disk_test.go`, which carries the `integration || e2e` tag.

The architecture guard reads the source of every untagged test file, so it covers the helpers of a package too. It flags
each use of `t.TempDir` and `t.Setenv`; each `os` call that reaches the file system, the environment, the working
directory, or a pipe, such as `os.Setenv`, `os.Clearenv`, `os.Chdir`, `os.Getenv`, `os.WriteFile`, and `os.Pipe`; each
member of `os/exec`; and the `filepath.Walk`, `WalkDir`, `Glob`, `Abs`, and `EvalSymlinks` calls. It flags a reference as
well as a call, so a function value stored in a field is caught. Pure uses, such as `filepath.Join`, `os.SameFile`,
`os.ErrNotExist`, and the `embed` package, are allowed. The guard has no exemption. Its rule is in
`tests/architecture/hermetic_test.go`, and its repository walk, `TestUntaggedTestsStayHermetic`, is in
`hermetic_integration_test.go`, so it runs in `test:integration` and `test:conformance`. The negative tests of the rule
are untagged, so `test:unit` runs them.

The rule reads the code of each test file. It does not follow a call into production code. A production function that
touches the disk stays outside the rule, and a test reaches it only through the seams above.

This page records no coverage figure. A figure comes from a run of `npx nx run retriever-cli:test:coverage` on a given
tree, and it belongs with the change that produced it.

## Binding check

`bdd:check` runs two tests of `tests/conformance` with the `bdd` build tag. They read every feature file under
`specs/behaviours`:

- Every `Scenario` and `Scenario Outline` needs an `@unit` tag. No exemption exists for it.
- Every scenario needs an `@integration` tag or an `@e2e` tag, or the matching exempt tag, `@integration-exempt` or
  `@e2e-exempt`.
- An exempt tag needs a justification: a comment line in the lines directly above the scenario's tags, in this form:

  ```gherkin
  # Exemption(e2e): <why this level cannot prove it>; alternative-proof: <where it is proved>
  @e2e-exempt
  ```

- A scenario that carries both `@integration` and `@integration-exempt`, or both `@e2e` and `@e2e-exempt`, fails. A
  scenario with both exempt tags is a detail, and it belongs in a plain Go test.
- `behaviour-coverage.json` must match the feature files. It records, for each feature, the scenario count and the count
  of `@unit`, `@integration`, and `@e2e` tags, with the totals. A difference fails the check.

## CI runs

The Go job runs only when an affected Nx project carries the `lang:go` tag. Within that job, each row admits its targets
once, through one HIPPO run under the row's resource tier, and runs them in order:

- `ubuntu-24.04` runs `lint`, `typecheck`, `test:unit`, `test:integration`, `test:e2e`, `test:conformance`,
  `bdd:check`, and `deps:audit`, under the `heavy` tier.
- `macos-15` runs `test:unit`, `test:e2e`, and `test:conformance`, under the `standard` tier.

The tier follows the work. The Ubuntu row runs a whole project's suite, which is heavy work. The macOS row runs a subset
of one project's tests, which is standard work. A `macos-15` runner has 3 CPUs and about 7 GiB of memory, less than the
`heavy` tier's minimum reservation of 4 CPUs and 8192 MiB, so a `heavy` reservation does not fit that runner.

The macOS targets cover the behaviour that depends on the platform: signals, exit statuses 130, 141, and 143, the user
configuration directory, atomic renames, file modes, symbolic links, and port binding.

A cross-compile check, on `ubuntu-24.04`, builds `./cmd/retriever` for `darwin/arm64`, `darwin/amd64`, `linux/amd64`,
and `linux/arm64`. It builds only. Nothing is uploaded or published.

The Go toolchain is pinned once, by the `toolchain` line of `apps/retriever-cli/go.mod` (`go1.27.4`). The `go` line sets
the minimum language version. The Go job fails when the installed toolchain differs from the `toolchain` line.

Windows is not supported. See the [exit codes](./retriever-cli-exit-codes.md#supported-platforms).

## Pull-request gate

`./rhino gate run --surface pull-request` does not run any target on this page. It runs the repository's own checks,
such as format, README index, emoji, and public-safety checks.
