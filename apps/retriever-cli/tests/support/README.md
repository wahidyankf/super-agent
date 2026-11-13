# Test support

Stub servers and shared helpers for the test suites. Each stub answers over loopback TLS and records every request, so a
test can assert what the client sent. None of them holds a real key, and the request log never holds a secret value.

- [outlinestub/](outlinestub/README.md): a fake Outline API with replay fixtures, scripted faults, and a request log.
- [kiwistub/](kiwistub/README.md): a fake Kiwi TCMS JSON-RPC server.
- [jirastub/](jirastub/README.md): a fake Jira Cloud REST server.
- [sourcecontract/](sourcecontract/README.md): one behaviour suite that every source gateway must pass.
- [outlinestubcmd/](outlinestubcmd/README.md), [kiwistubcmd/](kiwistubcmd/README.md),
  [jirastubcmd/](jirastubcmd/README.md): test-only commands that serve one stub as a process.
