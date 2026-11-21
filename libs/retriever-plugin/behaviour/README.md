# behaviour

The unit and integration bindings of the behaviour scenarios in
[specs/libs/retriever-plugin/behaviours](../../../specs/libs/retriever-plugin/behaviours/). Each test names the scenario
it covers in an `@scenario` comment, and RHINO's coverage check reads those comments.

- [unit/](./unit/README.md) holds the hermetic tests. Every scenario is bound here, through injected files, clocks,
  and command seams.
- [integration/](./integration/README.md) holds the tests that cross the process and file boundary. They run the
  fixture plugin and shell-script plugins as real child processes over real temporary files. Scenarios whose behaviour
  is a pure in-memory rule carry a reasoned exemption instead.

There is no end-to-end folder. The plugin is built into the retriever binary; the application's end-to-end suite drives
it, and each scenario says so in its exemption.
