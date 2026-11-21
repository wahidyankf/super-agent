# retriever-plugin Feature Files

One feature file for each area of the plugin library. A feature describes what a caller observes: a call that runs or
is refused, the environment the plugin receives, a reply or a failure, or an exit status.

## Files

- [pinning-and-start.feature](pinning-and-start.feature): the pin check, the private copy, start failures, and process
  kills.
- [environment-and-secrets.feature](environment-and-secrets.feature): the scrubbed environment, per-call secrets, and
  redaction.
- [exchange-and-timeouts.feature](exchange-and-timeouts.feature): the one-request exchange, reply shape, frame and
  stderr caps, and timeouts.
- [gateway-mapping.feature](gateway-mapping.feature): how the gateway maps plugin replies onto domain values.
- [real-process.feature](real-process.feature): the same behaviours checked with a real plugin process.
