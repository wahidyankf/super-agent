# behaviour

The behaviour bindings of the retriever-testkit library. Each scenario of
[the feature corpus](../../../specs/libs/retriever-testkit/behaviours/) is bound at the unit level under
[unit/](./unit/README.md). A scenario whose behaviour crosses a boundary of the machine is also bound under
[integration/](./integration/README.md). The command scenarios run the test-only stub commands as processes, and they
are also bound under [e2e/](./e2e/README.md). Each binding carries an `@scenario` comment line, which the repository's
behaviour coverage check reads.
