# retriever-runtime Behaviour Specifications

The behaviour of the `retriever-runtime` library as a caller sees it. The library runs the source use cases, resolves
gateways for each source, loads the sources catalog, reads the env file, decides where the config files are read from,
reads connector spec folders, and writes export files. Its feature files describe what each exported function returns,
refuses, writes, or reports.

Each behaviour has an ID with the prefix `RUN`. The IDs appear in the inventory of the spec review, and each one sits
above a scenario in the feature files.

## Folders

- [behaviours/](behaviours/): the feature files, one for each area of the library.
