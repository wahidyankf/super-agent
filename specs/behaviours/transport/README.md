# Transport behaviours

Rules that the Outline, Kiwi, and Jira adapters share: HTTPS only, no redirects, bounded waits, bounded replies, bounded
retries, and one request at a time.

## Files

- [https-and-limits.feature](https-and-limits.feature): the transport rules, each checked against a local stub server.
