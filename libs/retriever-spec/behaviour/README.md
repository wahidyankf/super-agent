# behaviour

The bindings of the `retriever-spec` behaviour corpus in [specs/libs/retriever-spec/behaviours](../../../specs/libs/retriever-spec/behaviours/README.md).
Each test carries an `@scenario` line for each scenario it covers.

- [unit/](./unit/README.md): hermetic tests that call the exported API with bytes held in memory. Every scenario is bound
  here.

The module reads no file, process, or network at run time, so no scenario has an integration binding. Each scenario is
tagged `@integration-exempt` and `@e2e-exempt` in the corpus, with the reason.
