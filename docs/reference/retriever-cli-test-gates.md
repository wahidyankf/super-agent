# Retriever test gates

This page records what each test target of the app and of each library runs, what the coverage gate measures and
excludes, what the binding check enforces, and which targets CI runs. It states facts. The rules that govern them live
in the repository's standards.

## Targets

Each Nx project has its own targets, and each runs in its module's folder. The app is `retriever-cli`, in
`apps/retriever-cli`. The eight libraries are `retriever-contract`, `retriever-runtime`, `retriever-testkit`,
`retriever-connector-outline`, `retriever-connector-kiwi`, `retriever-connector-jira`, `retriever-spec`, and
`retriever-engine`, in their folders under `libs/`. The targets below come from each `project.json`.

### App targets (`retriever-cli`)

| Target             | Runs                                                                                  |
| ------------------ | ------------------------------------------------------------------------------------- |
| `lint`             | `go vet` with the integration and e2e tags, then a `gofmt` check                      |
| `typecheck`        | `go build ./...`, then compiles every test package with those tags, without running   |
| `install`          | `go mod download`                                                                     |
| `build`            | `go build -trimpath -o dist/retriever ./cmd/retriever`                                |
| `build:release`    | `scripts/build-release.sh`: the four platform archives and `SHA256SUMS` of one tag    |
| `format`           | `gofmt -w .`                                                                          |
| `format:check`     | `gofmt -l`, which fails when it lists a file                                          |
| `test:unit`        | skip check, gate self-test, gate, release build self-test; see below                  |
| `test:quick`       | `lint`, `typecheck`, then `test:unit`                                                 |
| `test:integration` | `go test -count=1 -tags integration ./...`                                            |
| `test:e2e`         | `go test -count=1 -tags e2e ./tests/behaviour/...`; the suite builds the binary first |
| `test:conformance` | `go test -count=1 -tags integration ./tests/conformance/... ./tests/architecture/...` |
| `bdd:check`        | the binding check of `specs/behaviours`, described below                              |
| `test:coverage`    | `sh ../../scripts/go-coverage-gate.sh`, the gate without its self-test or skip check  |
| `deps:audit`       | `go tool govulncheck ./...`, which reads vuln.go.dev and needs the network            |

The skip check fails when a `_test.go` file under `internal`, `cmd`, or `tests` calls `.Skip(`.

### Library targets

Every library has the same targets. None of them runs the behaviour, conformance, or binding checks.

| Target             | Runs                                                                                |
| ------------------ | ----------------------------------------------------------------------------------- |
| `install`          | `go mod download`                                                                   |
| `build`            | `go build ./...`                                                                    |
| `lint`             | `go vet` with the integration and e2e tags, then a `gofmt` check                    |
| `typecheck`        | `go build ./...`, then compiles every test package with those tags, without running |
| `format`           | `gofmt -w .`                                                                        |
| `format:check`     | `gofmt -l`, which fails when it lists a file                                        |
| `test:unit`        | skip check, then the coverage gate with `COVERAGE_PACKAGES='./...'`                 |
| `test:quick`       | `lint`, `typecheck`, then `test:unit`                                               |
| `test:integration` | `go test -count=1 -tags integration ./...`                                          |
| `test:coverage`    | the coverage gate with `COVERAGE_PACKAGES='./...'`, without the skip check          |

## Coverage gate

Each module has its own coverage gate, and each one is held to its own 99 percent floor. The app and each library are
gated separately. No profile merges two modules, so one module's packages never count toward another module's figure.

`scripts/go-coverage-gate.sh` is the one gate script. Each module runs it from its own module root:

- the app runs `sh ../../scripts/go-coverage-gate.sh`, which measures the `./internal/...` and `./cmd/...` packages
  by default;
- each library runs `COVERAGE_PACKAGES='./...' sh ../../scripts/go-coverage-gate.sh`, which measures every package of
  its module.

The script does this, in order:

1. It reads the module's `scripts/coverage-exclusions.tsv` and checks each entry. It also checks that every production
   package with statements has a test file, because a package without one is never measured. `--check-only` runs only
   these checks.
2. It makes one instrumented run of the module's untagged tests, with `-race`. The `-coverpkg` flag covers the listed
   packages, so a test in one package counts the statements it runs in another.
3. It drops the profile blocks of excluded files, and it adds up the statements that the profile covers.
4. It fails when a test fails, when no statement is measured, or when the total is below `COVERAGE_MINIMUM`. The
   default minimum is 99.

The app's gate self-test, `scripts/go-coverage-gate_test.sh`, runs the `--check-only` mode against small fixture
modules. The app's `test:unit` target runs it first. The library targets do not run it.

This page records no coverage figure. A figure comes from a run of the target on a given tree, and it belongs with the
change that produced it.

### Exclusions

The exclusions are data in each module's `scripts/coverage-exclusions.tsv`. Each line holds a kind, a path, and a
reason, separated by tabs. The only kind allowed is `generated`: a Go file whose `Code generated ... DO NOT EDIT.` line
comes before its package clause.

- The app excludes two files, `internal/adapters/graphql/generated.go` and
  `internal/adapters/graphql/models_gen.go`, which gqlgen writes.
- No library excludes anything, because none has a generated Go file.

The gate fails on any other kind, on an entry with no reason, on a path pattern, on a path that does not exist, and on a
`generated` file without the header before its package clause. Test support is not an exclusion. The testkit is a module
of its own, and its packages are measured by its own gate.

## Unit tests

### What an untagged test may do

An untagged test runs in process, beside its package, and it does not reach outside the process. It does not read or
write a file, does not read or change the process environment, does not change the working directory, does not start a
child process, and does not open a pipe. It opens no socket: it starts no listener, dials no address, and starts no
`httptest` server. It reads no clock and waits on no timer: it calls no `time.Now`, `time.Sleep`, or `time.After`, and
it starts no timer or ticker. A listener, a server, or a time that the test needs is supplied by the test, in memory.

The file system, the environment, and the child processes reach the code through seams: a field or a parameter that
holds a function or an interface. The production code takes the seam, and a unit test supplies a fake. A test that needs
the real file system, the real environment, or a real child process carries the `integration` build tag. It sits in a
file named `*_integration_test.go`, or it carries the `integration || e2e` tag when one file serves both levels.

### The hermetic guard

The architecture guard reads the source of every untagged test file in the app and in every library, so it covers the
helpers of a package too. It flags:

- `t.TempDir` and `t.Setenv`, as method calls;
- each `os` function that reaches the file system, the environment, the working directory, or a pipe, such as
  `os.Setenv`, `os.Getwd`, `os.WriteFile`, `os.DirFS`, and `os.Pipe`;
- each member of `os/exec`;
- `filepath.Walk`, `WalkDir`, `Glob`, `Abs`, and `EvalSymlinks`;
- each `net` listener and dialer, `net/http.ListenAndServe` and `ListenAndServeTLS`, and the `httptest` server
  constructors `NewServer`, `NewTLSServer`, and `NewUnstartedServer`; and
- each clock read and timer in `time`: `Now`, `Since`, `Until`, `Sleep`, `After`, `AfterFunc`, `Tick`, `NewTimer`, and
  `NewTicker`.

The guard matches a method by its name on any receiver, so `t.TempDir`, `t.Setenv`, and the method calls `Listen`,
`ListenAndServe`, `ListenAndServeTLS`, and `Dial` are flagged. An in-memory helper must not reuse those names. The guard
does not match `DialContext`, which names a field of `http.Transport` as well as a method of `net.Dialer`.

It flags a reference as well as a call, so a function value stored in a field is caught. Pure uses, such as
`filepath.Join`, `os.SameFile`, `os.ErrNotExist`, and the `embed` package, are allowed. The rule has no exemption. It
does not follow a call into production code. The rule lives in `apps/retriever-cli/tests/architecture/hermetic_test.go`,
and the walk over the repository lives in `hermetic_integration_test.go`, which carries the `integration` build tag.

### Where the real wrappers sit

The composition root, `apps/retriever-cli/cmd/retriever`, holds the real calls to the process: the standard streams, the
environment, the disk, and the clock. Its integration tests, `wiring_closures_integration_test.go` and
`wiring_files_integration_test.go`, exercise those wrappers under the `integration` tag.

The two gaps that earlier versions named are closed. `disk_open_test.go` keeps only the hermetic checks, and
the real opener is tested in `disk_open_integration_test.go`. Each connector's `seam.go` takes a `getenv` function and a
file reader, and the composition root passes `os.ReadFile`.

Known gaps, not yet fixed:

- The guard does not follow a call into production code. A production default that a test leaves unset can still open
  a socket or read the clock: `cli.run` uses `time.Now` when `Deps.Now` is nil, `outlinestub.NewTLSServer` and
  `jirastub.NewTLSServer` use the real loopback listener and clock by default, and `kiwistub` uses `net.Listen` by
  default.
- The guard matches names, not types. A call through a `DialContext` field is not matched.

### Integration and end-to-end tests

Integration tests carry the `integration` tag, and end-to-end tests carry the `e2e` tag. The `test:integration` target
runs the first, and `test:e2e` runs the second. Neither produces a coverage figure. The pre-push gate runs no Go target,
so no integration test runs before a push.

## Unit coverage by module

The figure that the gate holds to 99 percent means different things for the app and for a library.

- **The app.** Its unit coverage counts the behaviour scenarios that run at the unit level, plus the plain unit tests
  outside the behaviour suite. `TestUnitBehaviours` is untagged, so it runs the `@unit` scenarios in memory. The gate
  counts the statements that those scenarios execute.
- **A library.** Its unit coverage counts its plain unit tests. A library has no behaviour suite of its own.

Integration and end-to-end tests stay outside the figure for both.

## Binding check

`bdd:check` runs two tests of `tests/conformance` with the `bdd` build tag. They read every feature file under
`specs/behaviours`. These rules apply to the app's feature files:

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

### Connector spec scenarios

`specs/behaviours/connectors/spec-connectors.feature` covers the connectors folder, the CLI and MCP switches, the catalog
pin, the start-up refusals, and the Markdown converter that the engine refuses. Each scenario runs at the unit level and
the integration level. Most also run at the end-to-end level. The one exemption is the first scenario, a spec command
that prints its mapped document, which needs a source that answers over HTTPS:

```gherkin
# Exemption(e2e): the built binary cannot trust the loopback stub; alternative-proof: the integration binding.
@e2e-exempt
```

The unit level answers from the replay stubs in memory. The integration level answers from the loopback stub. The
end-to-end level runs the built binary and cannot count requests to a source, so its check for "no request" holds when
the error names no network failure. A developer detail is a plain test, not a scenario. Examples are the folder
listing in `libs/retriever-runtime/specfolder` and the composition-root wiring in `cmd/retriever`.

## CI runs

The Go job runs when an affected Nx project carries the `lang:go` tag. Each row selects its projects from the Nx project
graph by tag, runs each target as one `nx run-many` over those projects, and admits the whole row once, through one
HIPPO run under the row's resource tier. The targets run in the order listed:

- `ubuntu-24.04` runs `lint`, `typecheck`, `test:unit`, `test:integration`, `test:e2e`, `test:conformance`,
  `bdd:check`, and `deps:audit` for every project tagged `lang:go`, under the `heavy` tier. A target runs only in the
  projects that declare it. Each library declares `lint`, `typecheck`, `test:unit`, and `test:integration`, and none
  declares `test:e2e`, `test:conformance`, `bdd:check`, or `deps:audit`, so those four run in the app alone.
- `macos-15` runs `test:unit`, `test:e2e`, and `test:conformance` for the projects tagged `type:app`, which is the
  application alone, under the `standard` tier.

Each project's `test:unit` runs its own coverage gate over its own module. No profile merges two modules.

The tier follows the work. The Ubuntu row runs whole suites of eight projects, which is heavy work. The macOS row runs
one project's tests, which is standard work. A `macos-15` runner has 3 CPUs and about 7 GiB of memory, less than the
`heavy` tier's minimum reservation of 4 CPUs and 8192 MiB, so a `heavy` reservation does not fit that runner.

The macOS targets cover the behaviour that depends on the platform: signals, exit statuses 130, 141, and 143, the user
configuration directory, atomic renames, file modes, symbolic links, and port binding. The application's end-to-end and
conformance suites run the built binary, which includes the libraries, so they cover the libraries on macOS. The
libraries' integration tests use the real file system and processes, and they run on Ubuntu only, as the application's
integration tests do.

A cross-compile check, on `ubuntu-24.04`, builds the app's `./cmd/retriever` for `darwin/arm64`, `darwin/amd64`,
`linux/amd64`, and `linux/arm64`, with one HIPPO run per platform under the `standard` tier, as before. It builds only,
and nothing is uploaded or published. The app's `replace` directives name the libraries' folders in the checkout, so the
build uses the libraries of the working tree.

The Go toolchain is pinned by the `toolchain` line of each `go.mod`, which is `go1.27.4` in all nine modules. The `go`
line sets the minimum language version, `1.27.0`. The Go setup fails when the installed toolchain differs from any
module's `toolchain` line, or when two modules declare different `go` lines. The module cache is keyed on
every `go.sum`.

Windows is not supported. See the [exit codes](./retriever-cli-exit-codes.md#supported-platforms).

## Pull-request and pre-push gates

`./rhino gate run --surface pull-request` does not run any target on this page. It runs the repository's own checks,
such as format, README index, emoji, and public-safety checks. The `pre-push` surface runs its own set of repository
checks, and it runs no Go target.
