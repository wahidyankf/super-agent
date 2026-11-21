# retriever-contract Behaviour Specifications

The behaviour of the `retriever-contract` library as a caller sees it. The library holds the plain types, their
validators, the shared error values, and the operation catalog that every adapter reads. Its feature files describe
what each exported constructor, validator, and catalog function returns, refuses, or reports.

Each behaviour has an ID with the prefix `CON`. The IDs appear in the inventory of the spec review, and each one sits
above a scenario in the feature files.

## Folders

- [behaviours/](behaviours/): the feature files, one for each area of the library.
