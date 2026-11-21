# retriever-testkit Behaviour Specifications

The behaviour of the `retriever-testkit` library as a caller sees it. The library is test support only: the stub servers
for Outline, Kiwi, and Jira, the in-memory file tree, the shared source contract suite, the synthetic Outline replies,
and the three test-only commands that serve a stub as a process. Its feature files describe what each exported function,
stub, and command returns, answers, refuses, or reports.

Each behaviour has an ID with the prefix `TK`. The IDs appear in the inventory of the spec review, and each one sits
above a scenario in the feature files.

## Folders

- [behaviours/](behaviours/): the feature files, one for each area of the library.
