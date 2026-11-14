# Outline behaviours

The Outline wiki source: reads, ordinary writes, destructive and admin actions, and what an Outline reply may
show on screen.

## Files

- [outline-reads.feature](outline-reads.feature): document, collection, search, and account reads, paging, and the
  `--source` choice when two Outline sources exist.
- [outline-writes.feature](outline-writes.feature): creating and changing documents, comments, and pins. Each one sends
  one request.
- [outline-critical-actions.feature](outline-critical-actions.feature): deleting, emptying the trash, removing users,
  and creating integrations. Each one runs as its command names it, with no confirmation flag.
- [outline-redaction.feature](outline-redaction.feature): terminal controls, webhook secrets, and keys that a hostile or
  failing reply might echo.
