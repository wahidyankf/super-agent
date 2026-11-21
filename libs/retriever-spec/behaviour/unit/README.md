# unit

Hermetic behaviour tests of `retriever-spec`. The package is `unit_test`, so each test calls only the exported API, as a
caller does. One file holds the tests of one feature file, and the file takes its name.

- [testdata/](./testdata/README.md): the bundled example spec that the tests embed with `go:embed`.
- [fixtures_test.go](./fixtures_test.go): the shared documents and the helpers that build a document from a base.

No test reads the file system, the environment, the clock, the network, a process, or random numbers.
