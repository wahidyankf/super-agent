# behaviour

The behaviour bindings of the retriever-connector-kiwi library. Each scenario of
[the feature corpus](../../../specs/libs/retriever-connector-kiwi/behaviours/) is bound at the unit level under
[unit/](./unit/README.md). A scenario whose behaviour crosses the wire or touches a file is also bound under
[integration/](./integration/README.md). A scenario with no integration binding carries an `@integration-exempt` tag
that gives its reason. Every scenario carries an `@e2e-exempt` tag, because the library has no process of its own.

Each binding carries an `@scenario` comment line, which the repository's behaviour coverage check reads.
