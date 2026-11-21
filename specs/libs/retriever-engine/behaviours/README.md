# retriever-engine Feature Files

One feature file for each area of the engine. A feature describes what a caller of the engine observes: a result, a
refusal before any request, a sent request, an error code, or an exit status.

## Files

- [construction-and-gateway.feature](construction-and-gateway.feature): building an engine, its ports, and the
  collection reads.
- [reads-and-fields.feature](reads-and-fields.feature): list and object reads, field shapes, flag maps, and argument
  checks.
- [paging.feature](paging.feature): offset, cursor, page-all, body, page-end, and truncation rules.
- [merges-and-trees.feature](merges-and-trees.feature): merged lists and tree reads.
- [fallbacks-and-shapes.feature](fallbacks-and-shapes.feature): fallback reads, dual reply shapes, and the unavailable
  message.
- [request-safeguards.feature](request-safeguards.feature): headers, path, query, body, redirects, and the key on a
  request.
- [post-reads-and-query-keys.feature](post-reads-and-query-keys.feature): POST reads and query-key aliases.
- [reply-safeguards-and-echo.feature](reply-safeguards-and-echo.feature): reply size, depth, text cleaning, and key
  echo.
- [documents-and-built-ids.feature](documents-and-built-ids.feature): Markdown documents and built identifiers.
- [network-guard.feature](network-guard.feature): the address checks that run before every dial.
- [timing-and-retries.feature](timing-and-retries.feature): time limits, rate-limit retries, and pacing.
- [writes-and-outcomes.feature](writes-and-outcomes.feature): writes, their unknown outcomes, and the write refusals.
- [write-lookups.feature](write-lookups.feature): resolving a name to an ID before a write.
- [converters-adf.feature](converters-adf.feature): Atlassian Document Format to Markdown.
- [converters-text-and-time.feature](converters-text-and-time.feature): editor data, host names, and Jira time, date,
  and title.
- [replay-stub.feature](replay-stub.feature): the replay stub that tests use as a source.
- [errors-and-exit-status.feature](errors-and-exit-status.feature): stable error codes and exit statuses.
- [loopback-connections.feature](loopback-connections.feature): connections to a local test server.
