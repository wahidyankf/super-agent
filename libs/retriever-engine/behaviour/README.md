# behaviour

The behaviour bindings of the retriever-engine library. Each scenario of
[the feature corpus](../../../specs/libs/retriever-engine/behaviours/) is bound at the unit level under
[unit/](./unit/README.md). A scenario whose behaviour crosses a boundary of the machine is also bound under
[integration/](./integration/README.md). Each binding carries an `@scenario` comment line, which the repository's
behaviour coverage check reads.
