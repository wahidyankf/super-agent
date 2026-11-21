# unit

The hermetic unit bindings of the library's behaviour scenarios. Each file binds one feature file, and each test
function binds one scenario, or one Scenario Outline with all of its rows. The tests are in the external package
`unit_test` and call only the library's exported API.

No test here reads a file, a clock, an environment variable, a process, the network, or a random source. The values
come from the test itself, and the errors come from the library's own constructors and validators.

- [support_test.go](./support_test.go): the helpers that build the sources, the endpoints, and the catalogs that the
  scenarios name.
- [arguments_and_binding_test.go](./arguments_and_binding_test.go): binding the arguments of one call to one row.
- [contract_version_test.go](./contract_version_test.go): the version constant of the contract.
- [endpoints_test.go](./endpoints_test.go): source endpoints, which are HTTPS host names only.
- [identifiers_test.go](./identifiers_test.go): collection, document, source, tag, UUID, and share IDs.
- [identity_and_credentials_test.go](./identity_and_credentials_test.go): identity and credential variable names, and
  redacted key and address values.
- [operation_catalog_test.go](./operation_catalog_test.go): operation rows, kinds, closed lists, effects, and resources.
- [plugin_pins_test.go](./plugin_pins_test.go): plugin references and their SHA-256 pins.
- [public_errors_test.go](./public_errors_test.go): which error texts a presenter may show.
- [records_and_pages_test.go](./records_and_pages_test.go): bound arguments, result records, and pages.
- [source_kinds_test.go](./source_kinds_test.go): source kinds, kind sets, and kind-specific error texts.
- [sources_and_catalog_test.go](./sources_and_catalog_test.go): catalog construction, lookup, filters, and selection.
- [text_values_test.go](./text_values_test.go): the value checks for text, numbers, times, and lists.
