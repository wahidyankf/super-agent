# retriever-connector-kiwi Behaviour Specifications

The behaviour of the `retriever-connector-kiwi` library as a caller sees it. The library reads and writes a Kiwi TCMS
instance over HTTPS with JSON-RPC 2.0. It lists products, collection trees, and documents, creates and updates cases,
and adds a case to a plan. Only three of its ten allowed methods change Kiwi.

Each behaviour has an ID with the prefix `KIW`. The IDs appear in the inventory of the spec review, and each one sits
above a scenario in the feature files.

## Folders

- [behaviours/](behaviours/): the feature files, one for each area of the library.
