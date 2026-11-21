# Retriever test gates

This page records what each test target of the app and of each library runs, what the coverage gate measures and
excludes, what the behaviour check enforces, and which targets CI runs. It states facts. The rules that govern them live
in the repository's standards. The repository-wide gate list is in [Quality gates](./quality-gates.md).

## Targets

Each Nx project has its own targets, and each runs in its module's folder. The app is `retriever-cli`, in
`apps/retriever-cli`. The seven libraries are `retriever-contract`, `retriever-runtime`, `retriever-testkit`,
`retriever-connector-kiwi`, `retriever-spec`, `retriever-engine`, and `retriever-plugin`, in their folders under
`libs/`. Pull request 17 retired `retriever-connector-outline`, and pull request 39 retired `retriever-connector-jira`;
each removed its directory. The targets below come from each `project.json`.

### App targets (`retriever-cli`)

| Target                    | Runs                                                                                     |
| ------------------------- | ---------------------------------------------------------------------------------------- |
| `lint`                    | `go vet` with the integration and e2e tags, then a `gofmt` check                         |
| `typecheck`               | `go build ./...`, then compiles every test package with those tags, without running      |
| `install`                 | `go mod download`                                                                        |
| `build`                   | `go build -trimpath -o dist/retriever ./cmd/retriever`                                   |
| `build:release`           | `scripts/build-release.sh`: the four platform archives and `SHA256SUMS` of one tag       |
| `format`                  | `gofmt -w .`                                                                             |
| `format:check`            | `gofmt -l`, which fails when it lists a file                                             |
| `test:unit`               | skip check, gate self-test, gate, release build self-test; see below                     |
| `test:quick`              | `typecheck`, `lint`, `test:unit`, then `test:behaviour:coverage`                         |
| `test:integration`        | `go test -count=1 -timeout 20m -tags integration ./...`                                  |
| `test:e2e`                | `go test -count=1 -timeout 20m -tags e2e ./tests/behaviour/...`; builds the binary first |
| `test:conformance`        | `go test -count=1 -tags integration ./tests/conformance/... ./tests/architecture/...`    |
| `test:behaviour:coverage` | `./rhino behaviour coverage validate`; see [Behaviour check](#behaviour-check)           |
| `test:coverage`           | `sh ../../scripts/go-coverage-gate.sh`, the gate without its self-test or skip check     |
| `deps:audit`              | `go tool govulncheck ./...`, which reads vuln.go.dev and needs the network               |

The skip check fails when a `_test.go` file under `internal`, `cmd`, or `tests` calls `.Skip(`.

### Library targets

Every library has the same targets, except that `retriever-contract` and `retriever-spec` declare no `test:integration`:
they have no integration-tagged test. Each runs the static behaviour check, `test:behaviour:coverage`. None runs
`test:conformance` or `test:e2e`.

| Target                    | Runs                                                                                |
| ------------------------- | ----------------------------------------------------------------------------------- |
| `install`                 | `go mod download`                                                                   |
| `build`                   | `go build ./...`                                                                    |
| `lint`                    | `go vet` with the integration and e2e tags, then a `gofmt` check                    |
| `typecheck`               | `go build ./...`, then compiles every test package with those tags, without running |
| `format`                  | `gofmt -w .`                                                                        |
| `format:check`            | `gofmt -l`, which fails when it lists a file                                        |
| `test:unit`               | skip check, then the coverage gate with `COVERAGE_PACKAGES='./...'`                 |
| `test:quick`              | `typecheck`, `lint`, `test:unit`, then `test:behaviour:coverage`                    |
| `test:integration`        | `go test -count=1 -tags integration ./...`                                          |
| `test:coverage`           | the coverage gate with `COVERAGE_PACKAGES='./...'`, without the skip check          |
| `test:behaviour:coverage` | `./rhino behaviour coverage validate` from the repository root                      |

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
2. It makes one instrumented run of the module's untagged tests, with `-race` and a `-timeout` of `GO_TEST_TIMEOUT`,
   which defaults to `20m`, because Go's own default of 10 minutes is shorter than the app's behaviour suite under load.
   The `-coverpkg` flag covers the listed packages, so a test in one package counts the statements it runs in another.
3. It drops the profile blocks of excluded files, and it adds up the statements that the profile covers.
4. It fails when a test fails, when no statement is measured, or when the total is below `COVERAGE_MINIMUM`. The
   default minimum is 99.

The app's gate self-test, `scripts/go-coverage-gate_test.sh`, runs the `--check-only` mode against small fixture
modules, and one full run whose test outlives a one-second `GO_TEST_TIMEOUT`. The app's `test:unit` target runs it
first. The library targets do not run it.

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
so no integration test runs before a push, and no end-to-end test runs on a pull request.

## Unit coverage by module

The figure that the gate holds to 99 percent means different things for the app and for a library.

- **The app.** Its unit coverage counts the behaviour scenarios that run at the unit level, plus the plain unit tests
  outside the behaviour suite. `TestUnitBehaviours` is untagged, so it runs the unit-level scenarios in memory. The gate
  counts the statements that those scenarios execute.
- **A library.** Its unit coverage counts its plain unit tests and its unit-level behaviour bindings, which sit in
  `behaviour/unit`. A library has no behaviour suite of its own.

Integration and end-to-end tests stay outside the figure for both.

## Behaviour check

`test:behaviour:coverage` runs `./rhino behaviour coverage validate` from the repository root. It reads the feature
files and the test files, and it runs no test. `test:quick` ends with this check too. It replaced the earlier
`bdd:check` target, its `behaviour-coverage.json` file, and the conformance test
`tests/conformance/scenario_bindings_test.go`, which are all retired.

The corpus and its bindings are declared in `policies.behaviour` in `repo-config.yml`:

- The app's corpus is `specs/behaviours`. Each library's corpus is `specs/libs/<lib>/behaviours`.
- A test file covers a scenario with a comment line, `@scenario <feature-path>: <scenario name>`.
- The level comes from the test file's location. The app's three suites in `apps/retriever-cli/tests/behaviour` and
  the `libs/*/behaviour/{unit,integration,e2e}` folders map to the levels in the declaration.
- `<feature-path>: *` covers every scenario of a file. It is allowed only from the three suites that the
  `whole-file` list names, because they run whole feature files.
- A scenario with no unit binding fails. The unit level is never exempt.
- A scenario that needs no integration or end-to-end test carries `@integration-exempt` or `@e2e-exempt`. The line
  directly above that tag is a comment in this form:

  ```gherkin
  # Exemption(e2e): <why this level cannot prove it>; alternative-proof: <target>
  @e2e-exempt
  ```

- An exemption with an empty reason or target, a tag without its comment, or a comment that names another level fails.
  A stale exemption fails when a binding exists at the exempted level.

The rules B01 to B09 that enforce these points are those of the contract's behaviour corpus module. The findings are
listed in the RHINO documentation for the `behaviour coverage validate` command.

### Outline read surface

`internal/adapters/in/mcp/testdata/outline-read-tools.golden` holds the MCP tools of the Outline reads: their names,
descriptions, input schemas, and hints. `TestOutlineReadToolsMatchTheGoBackedGolden` compares the server's list to it
byte for byte. A change to a read tool must update the golden in the same change, on purpose.

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

The pull-request workflow, `.github/workflows/pr-quality-gate.yml`, runs on each pull request to `main`. Its `detect`
job asks Nx which projects declare each target, so a project joins a job with no edit to the workflow. Each job runs one
target for one project:

- `repository-policy` runs the pull-request surface of the registry under RHINO, once for the repository.
- `go-static` runs `lint` and `typecheck` on Ubuntu, and `deps:audit` where a project declares it.
- `go-unit` runs `test:unit` and `go-integration` runs `test:integration`, both on Ubuntu.
- `go-conformance` runs `test:conformance` on Ubuntu, for the projects that declare it.
- `go-unit-macos` runs `test:unit`, and `go-conformance-macos` runs `test:conformance`, for the app alone,
  on `macos-15`.
- `behaviour-coverage` runs `./rhino behaviour coverage validate` once for the whole repository.
- `cross-compile` builds the app for four platforms. It builds only, and nothing is uploaded.
- `quality-gate` is the one aggregate check. It passes only when every required job succeeds, or when a Go job is
  skipped because no Go project is affected.

The end-to-end level does not run on a pull request. `.github/workflows/scheduled-e2e.yml` runs `test:e2e` for every
project that declares it. It runs on a weekly schedule (Monday, 03:17 UTC), on manual dispatch, and on each push to
`main`. The app's end-to-end suite runs again on `macos-15`, in the same workflow.

Each project's `test:unit` runs its own coverage gate over its own module. No profile merges two modules.

The tier follows the work. The Ubuntu jobs use the `heavy` tier. A `macos-15` runner has 3 CPUs and about 7 GiB of
memory, less than the `heavy` tier's minimum reservation of 4 CPUs and 8192 MiB, so the macOS jobs use the `standard`
tier.

The macOS targets cover the behaviour that depends on the platform: signals, exit statuses 130, 141, and 143, the user
configuration directory, atomic renames, file modes, symbolic links, and port binding. The application's end-to-end and
conformance suites run the built binary, which includes the libraries, so they cover the libraries on macOS. The
libraries' integration tests use the real file system and processes, and they run on Ubuntu only, as the application's
integration tests do.

The Go toolchain is pinned by the `toolchain` line of each `go.mod`, which is `go1.27.4` in all eight modules. The `go`
line sets the minimum language version, `1.27.0`. The Go setup fails when the installed toolchain differs from any
module's `toolchain` line, or when two modules declare different `go` lines. The module cache is keyed on every
`go.sum`.

Windows is not supported. See the [exit codes](./retriever-cli-exit-codes.md#supported-platforms).

## Registry gates

`./rhino gate run --surface pull-request` runs the registry's pull-request checks, such as format, README index,
emoji, and public-safety checks. It runs no Go target on this page. The `pre-push` surface runs only the public-safety
range screen, and no Go target either. The [Quality gates](./quality-gates.md) page lists every gate and its surface.
