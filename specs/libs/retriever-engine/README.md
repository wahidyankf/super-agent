# retriever-engine Behaviour Specifications

The behaviour of the `retriever-engine` library as a caller sees it. The engine runs the reads and writes of a validated
connector spec. It builds each request from the declared operation, sends it through a guarded transport, maps the reply
onto the declared fields, and returns a result or a stable error. A replay stub stands in for a source in tests.

Each behaviour has an ID with the prefix `ENG`. The IDs appear in the inventory of the spec review, and each one sits
above a scenario in the feature files.

## Folders

- [behaviours/](behaviours/): the feature files, one for each area of the library.
