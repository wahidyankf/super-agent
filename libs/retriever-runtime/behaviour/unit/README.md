# unit

The hermetic unit bindings of the runtime's behaviour scenarios. Each file binds one feature file, and each test function
binds one scenario, or one Scenario Outline with all of its rows. The tests are in the external package `unit_test` and
call the exported API of the runtime and of the contract.

No test here reads a disk, a variable, a clock, or a process. The folders are in-memory `fs.FS` values, the variables are
maps, and the file calls of the sink are fakes, so each one can fail on demand.

- [doubles_test.go](./doubles_test.go): the in-memory folder, the variable lookup, the source and reply builders, the
  wiki gateway that counts its calls, and the export sink recorder. It has no scenario of its own.
- [catalog-loading_test.go](./catalog-loading_test.go): loading the sources catalog from an in-memory file.
- [config-paths_test.go](./config-paths_test.go): the rules that choose the catalog, env file, and connectors paths.
- [connector-spec-folder_test.go](./connector-spec-folder_test.go): reading the connector spec folder through an `fs.FS`.
- [env-file_test.go](./env-file_test.go): the source key and account address from the variables and the env file.
- [export-sink_test.go](./export-sink_test.go): the Markdown export sink over an in-memory folder that fails on demand.
- [gateway-registry_test.go](./gateway-registry_test.go): the registry that builds and keeps one gateway per source.
- [sources-use-cases_test.go](./sources-use-cases_test.go): the read, write, export, and list use cases over the registry.

The shared doubles live in one file rather than in each feature file, because the sources, registry, and export bindings
use the same wiki gateway and the same reply builders.
