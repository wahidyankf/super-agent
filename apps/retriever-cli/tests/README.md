# Tests

The test suites of retriever-cli, one folder for each kind of test. Unit tests of the internal packages live beside
their code under [internal/](../internal/README.md).

## Folders

- [architecture/](architecture/README.md): the import-direction checks of the functional core.
- [behaviour/](behaviour/README.md): the godog runner for the behaviour specs at three levels.
- [conformance/](conformance/README.md): the runner for the published CLI contract checks.
- [support/](support/README.md): stub servers and shared helpers that the suites use.
