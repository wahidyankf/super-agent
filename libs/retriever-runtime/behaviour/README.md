# behaviour

The unit and integration bindings of the behaviour scenarios in
[specs/libs/retriever-runtime/behaviours](../../../specs/libs/retriever-runtime/behaviours/). Each test names the scenario
it covers in an `@scenario` comment, and RHINO's coverage check reads those comments.

- [unit/](./unit/README.md) holds the hermetic tests. Every scenario is bound here, through in-memory folders, fake
  variables, and injected file calls.
- [integration/](./integration/README.md) holds the tests that cross the file boundary. They use real temporary folders,
  real symbolic links, and real environment variables. Scenarios whose rule is already proven in memory, or whose
  fault cannot be made on a real disk, carry a reasoned exemption instead.

There is no end-to-end folder. The runtime is built into the retriever binary; the application's end-to-end suite drives
it, and each scenario says so in its exemption.
