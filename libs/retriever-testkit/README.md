# retriever-testkit

Stub servers, an in-memory file tree, a shared source contract suite, and synthetic replies for the test suites. Each
stub answers over loopback TLS and records every request, so a test can assert what the client sent. None of them holds
a real key, and the request log never holds a secret value.

Only tests import this module, and no production code does. Its own production code imports the contract alone.

- [outlinestub/](outlinestub/README.md): a fake Outline API with replay fixtures, scripted faults, and a request log.
- [kiwistub/](kiwistub/README.md): a fake Kiwi TCMS JSON-RPC server.
- [jirastub/](jirastub/README.md): a fake Jira Cloud REST server.
- [memfs/](memfs/README.md): an in-memory file tree that the unit level runs the file loaders on, with no disk.
- [synthetic/](synthetic/README.md): the embedded synthetic Outline replies that the behaviour harness serves.
- [sourcecontract/](sourcecontract/README.md): one behaviour suite that every source gateway must pass.
- [outlinestubcmd/](outlinestubcmd/README.md), [kiwistubcmd/](kiwistubcmd/README.md),
  [jirastubcmd/](jirastubcmd/README.md): test-only commands that serve one stub as a process.
- [scripts/](scripts/README.md) holds the coverage exclusions that this module's gate reads.
