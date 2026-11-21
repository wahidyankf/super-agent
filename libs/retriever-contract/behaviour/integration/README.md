# integration

The integration bindings of the library's behaviour scenarios. The folder holds no test file yet.

The library has no file, process, or network boundary of its own. Its production code imports only `context` and `time`,
and no function reads a file, runs a process, or opens a socket. So no scenario crosses a boundary of the executing
machine that an integration test would need to prove. Each scenario in
[specs/libs/retriever-contract/behaviours](../../../../specs/libs/retriever-contract/behaviours/) carries an
`@integration-exempt` tag with its reason, and the unit bindings in [unit/](../unit/README.md) prove the behaviour.

A future scenario that reads a real file, runs a child process, or talks over a loopback socket gets its test here, with
the `//go:build integration` tag and the package `integration_test`.
