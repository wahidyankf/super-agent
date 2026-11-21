# Runtime Behaviours

Feature files for the `retriever-runtime` library. Each file covers one area of its exported API.

## Files

- [catalog-loading.feature](catalog-loading.feature): loading and checking the sources catalog file.
- [config-paths.feature](config-paths.feature): where the catalog, the env file, and the connectors folder are read from.
- [connector-spec-folder.feature](connector-spec-folder.feature): reading the connector spec files of one folder.
- [env-file.feature](env-file.feature): source keys and account addresses from the environment and the env file.
- [export-sink.feature](export-sink.feature): writing Markdown export files into one folder.
- [gateway-registry.feature](gateway-registry.feature): building and reusing one gateway for each source.
- [sources-use-cases.feature](sources-use-cases.feature): the reads, writes, exports, and source list of one call.
