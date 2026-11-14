# Tests

The test suites of retriever-cli, one folder for each kind of test. The unit tests of the app's packages live beside
their code under [internal/](../internal/README.md). The test doubles that the suites share, such as the stub servers
and the in-memory file tree, are in [libs/retriever-testkit](../../../libs/retriever-testkit/README.md).

## Folders

- [architecture/](architecture/README.md): the import-direction rules for every module, and the guard that keeps
  untagged tests hermetic.
- [behaviour/](behaviour/README.md): the godog runner for the behaviour specs at three levels.
- [conformance/](conformance/README.md): the runner for the published CLI contract checks.
