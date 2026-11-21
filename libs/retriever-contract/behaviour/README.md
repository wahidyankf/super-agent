# behaviour

The unit and integration bindings of the behaviour scenarios in
[specs/libs/retriever-contract/behaviours](../../../specs/libs/retriever-contract/behaviours/). Each test names the
scenario it covers in an `@scenario` comment, and RHINO's coverage check reads those comments.

- [unit/](./unit/README.md) holds the hermetic tests. Every scenario is bound here.
- [integration/](./integration/README.md) holds the tests that cross the machine boundary. This library has none, so
  each scenario there carries a reasoned exemption instead.

There is no end-to-end folder. The library has no end-to-end boundary of its own; the application's end-to-end suite
drives it, and each scenario says so in its exemption.
