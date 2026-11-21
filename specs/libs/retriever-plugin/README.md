# retriever-plugin Behaviour Specifications

The behaviour of the `retriever-plugin` library as a caller sees it. The library runs a pinned out-of-process plugin. It
checks the plugin bytes against the pinned digest, runs a private copy, passes the plugin only its own credential
variables, and exchanges one JSON-RPC 2.0 request and reply over standard input and output.

Each behaviour has an ID with the prefix `PLG`. The IDs appear in the inventory of the spec review, and each one sits
above a scenario in the feature files.

## Folders

- [behaviours/](behaviours/): the feature files, one for each area of the library.
