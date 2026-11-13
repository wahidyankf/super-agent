# CLI behaviours

The command-line contract that every retriever command follows: the global `--output` and `--color` options, the help
and version output, usage errors, the failure body, and the closed exit statuses.

## Files

- [cli-contract.feature](cli-contract.feature): flags, help, version, usage errors, and the JSON failure body. It runs at
  all three levels because it needs no source.
