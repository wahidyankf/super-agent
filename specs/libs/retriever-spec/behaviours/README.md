# retriever-spec Feature Files

One feature file for each area of the spec library. A feature describes what a caller observes: a spec that is accepted
or refused, the findings it returns, the model it builds, or the review it prints.

## Files

- [document-reading.feature](document-reading.feature): reading one document, its size, and its plain YAML form.
- [refusals-and-findings.feature](refusals-and-findings.feature): refused specs, the findings they return, and the
  stable code.
- [schema-shape.feature](schema-shape.feature): the shape of each object in a spec, checked against the JSON Schema.
- [semantic-rules.feature](semantic-rules.feature): the rules that need more than the schema, such as ID prefixes and
  access labels.
- [access-and-post-reads.feature](access-and-post-reads.feature): the access matrix and the reads that go over POST.
- [pagination.feature](pagination.feature): page-all, page-end, and body paging rules and their limits.
- [fallbacks.feature](fallbacks.feature): fallback reads, their triggers, and the fallback pointers.
- [response-shapes.feature](response-shapes.feature): tree, merge, first-item, and other reply shapes.
- [key-echo.feature](key-echo.feature): the key echo of a single-object read.
- [document-layouts.feature](document-layouts.feature): the Markdown document layout of a single-object read.
- [record-fields.feature](record-fields.feature): nested records, flag maps, and the closed format names.
- [parameters-and-keys.feature](parameters-and-keys.feature): parameters, query-key aliases, built keys, and converters.
- [bundled-specs.feature](bundled-specs.feature): the bundled loader and the built-in catalog.
- [review-text.feature](review-text.feature): the plain-text review of a valid spec.
- [configuration.feature](configuration.feature): endpoint, authentication, limits, and surfaces a user writes.
- [hand-built-models.feature](hand-built-models.feature): the exported check functions for a model built in code.
