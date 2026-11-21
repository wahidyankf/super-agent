# retriever-connector-kiwi Feature Files

One feature file for each area of the Kiwi connector. A feature describes what a caller of the library observes: a
document or list returned, a write sent or refused, an error, or a refused request.

## Files

- [reads-and-documents.feature](reads-and-documents.feature): product lists, collection trees, documents, and sort
  order.
- [writes.feature](writes.feature): creating a case, updating a case, adding a case to a plan, and write failures.
- [identifiers.feature](identifiers.feature): the canonical product, plan, and case IDs.
- [documents-and-text.feature](documents-and-text.feature): the text rules and the case and plan document layouts.
- [transport-and-retries.feature](transport-and-retries.feature): the POST call, HTTPS base, redirects, limits, pacing,
  and retries.
- [methods-and-errors.feature](methods-and-errors.feature): the method table, the argument guard, and the error mapping.
- [test-seam.feature](test-seam.feature): the environment settings that point the adapter at a test server.
- [tls-connections.feature](tls-connections.feature): calls over a real TLS connection to a local test server.
