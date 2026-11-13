# CLI conformance fixtures

The data for the conformance runner in
[apps/retriever-cli/tests/conformance](../../../apps/retriever-cli/tests/conformance/README.md).

## Files

- [assertions.json](assertions.json): each published check, with its arguments, its exit status, the expected stream
  text, and the expected failure body. The `closedExitStatuses` list is the closed set of statuses the runner accepts,
  and `knownGaps` lists any check that a build does not yet satisfy. An entry in `knownGaps` must match a failing check
  exactly, so a gap that is fixed must also be removed from the list.
