# integration

The integration bindings of the runtime's behaviour scenarios. Each test file has the `//go:build integration` tag and the
package `integration_test`. Run them with `go test -tags integration ./behaviour/integration/`.

The tests use real files and folders. Each one works in `t.TempDir()`, makes a real symbolic link or folder where a
scenario needs one, and sets the process variables with `t.Setenv`, so the change ends with the test. The
[support_test.go](./support_test.go) file holds the helpers that the files share.

- [support_test.go](./support_test.go): the file writer, the variable reset, the padding to an exact size, and the
  real-disk export sink.
- [catalog-loading_test.go](./catalog-loading_test.go): the catalog file on disk: absent, present, a link or a folder,
  and the size limit.
- [config-paths_test.go](./config-paths_test.go): the rules over the real working directory, home folder, and profiles
  folder, and a profile folder that is a symbolic link.
- [connector-spec-folder_test.go](./connector-spec-folder_test.go): a real spec folder, with links, folders, and files
  at the size limit.
- [env-file_test.go](./env-file_test.go): a real env file and the real process environment.
- [export-sink_test.go](./export-sink_test.go): the Markdown export sink writing into a real folder.
- [sources-use-cases_test.go](./sources-use-cases_test.go): an export through the real sink, and an export into a folder
  that cannot be created.

The scenarios that stay in memory (the rules over the parsed catalog, the path choice over given values, the registry
cache, and the formatting of a value) carry an `@integration-exempt` tag with their reason. Their proof is in
[unit/](../unit/README.md).
