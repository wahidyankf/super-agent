# retriever-spec Behaviour Specifications

The behaviour of the `retriever-spec` library as a caller sees it. The library reads a declarative connector spec
document, validates it against the JSON Schema and the semantic rules, builds the spec model, and renders the plain-text
review that a person reads before any engine runs the spec. It also checks a model that a caller builds in code.

Each behaviour has an ID with the prefix `SPC`. The IDs appear in the inventory of the spec review, and each one sits
above a scenario in the feature files.

## Folders

- [behaviours/](behaviours/): the feature files, one for each area of the library.
