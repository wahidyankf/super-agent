# Library Behaviour Specifications

This tree holds the behaviour corpus of each library under `libs/`. A library corpus describes what a caller of the
library observes through its exported API: what it returns, refuses, writes, or exits with. It does not describe an
internal function.

The app's behaviour corpus lives in [behaviours/](../behaviours/). Each library has its own folder here, and
each folder holds a `behaviours/` folder of feature files.

## Libraries

- [retriever-contract/](retriever-contract/): the plain types, the validators, and the operation catalog.
- [retriever-runtime/](retriever-runtime/): the use cases, the catalog loader, the env file reader, the config
  path rules, and the export sink.
- [retriever-testkit/](retriever-testkit/): the stub servers, the in-memory file tree, and the shared source
  contract suite that the test suites import.

## Conventions

- A scenario describes one behaviour, in a caller's words. A variant is a Scenario Outline with Examples.
- Each inventory ID appears as `# Behaviour: <ID>` directly above a scenario. The prefixes are `CON` for the contract,
  `RUN` for the runtime, and `TK` for the testkit.
- Library feature files carry no old-style level tags. A later binding task adds the unit, integration, and end-to-end
  bindings, and writes an exemption only where a level has no reach.
- Values are synthetic. Hosts are `example.test` names, keys are `EXAMPLE_*` names, and emails use `example.test`.
