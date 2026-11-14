---
title: Retriever CLI scenario gap
description: Maps each Butler feature file to its retriever counterpart, and lists the gaps to close before v0.1.0.
category: reference
tags:
  - reference
  - cli
  - retriever
  - migration
---

# Retriever CLI scenario gap

This page maps each feature file of the older Butler CLI specs to the retriever scenarios that cover the same behaviour,
and it lists what is not ported. It states facts. It ports no scenario.

Butler's specs hold 98 feature files and 1013 scenario headings, 170 of them outlines. The retriever specs under
`specs/behaviours` hold 13 feature files and 235 scenario headings. Both counts count `Scenario` and `Scenario Outline`
lines. The retriever count matches `apps/retriever-cli/behaviour-coverage.json`, which `bdd:check` verifies.

## Status

- **Partly**: retriever has scenarios for part of the behaviour. The rest is a gap.
- **Gap**: the command ships in the operation catalog, but no retriever scenario covers it.
- **Out**: the behaviour stays in Butler by design. The reasons are below the table.
- **Covered**: every heading of the Butler feature has a retriever scenario that asserts the same behaviour.
- **Deferred**: the behaviour is planned for a later release.

Every **Partly** and **Gap** row is a follow-up that must close before the v0.1.0 release. Each one closes in its own
change, with its scenario written first and its binding tags set, as the [test gates](./retriever-cli-test-gates.md)
describe.

The table matches by title only, so it maps the features and does not count behaviour. Nothing here was run. A row says
**Partly** when a retriever title names the same behaviour as a Butler title, and it does not claim that the two behave
the same in every case.

## Table

File names omit the `.feature` suffix. The number in brackets is the Butler scenario count.

| Butler feature (scenarios)                    | Retriever counterpart                   | Status   |
| --------------------------------------------- | --------------------------------------- | -------- |
| butler (12)                                   | cli-contract: help, exit codes          | Partly   |
| harness-naming-validate (3)                   | not in retriever                        | Out      |
| jira-adf-rendering (14)                       | jira-operations: issue Markdown         | Partly   |
| jira-api-errors (13)                          | jira-operations: failures, retry        | Partly   |
| jira-comments-read (9)                        | jira-operations: comments               | Partly   |
| jira-credentials (11)                         | credentials: Jira e-mail address        | Covered  |
| jira-export (9)                               | export-markdown-files: Jira issue       | Partly   |
| jira-issues-read (11)                         | jira-operations: issue read             | Partly   |
| jira-issues-search (12)                       | jira-operations: JQL search             | Partly   |
| jira-projects-read (7)                        | jira-operations: projects               | Partly   |
| jira-secret-never-printed (5)                 | credentials: Jira token                 | Covered  |
| jira-transport-safety (10)                    | jira-operations, https-and-limits       | Partly   |
| jira-whoami (7)                               | jira-operations: account read           | Partly   |
| job-report (12)                               | not in retriever                        | Out      |
| kiwi-collections-read (10)                    | kiwi-operations: products, plans        | Partly   |
| kiwi-credentials (8)                          | credentials: Kiwi key                   | Covered  |
| kiwi-documents-read (9)                       | kiwi-operations: documents              | Partly   |
| kiwi-export (14)                              | export-markdown-files: test case        | Partly   |
| kiwi-jsonrpc-errors (10)                      | kiwi-operations: JSON-RPC error         | Partly   |
| kiwi-secret-never-printed (9)                 | credentials: Kiwi key                   | Covered  |
| kiwi-transport-safety (13)                    | kiwi-operations, https-and-limits       | Partly   |
| kiwi-writes (17)                              | kiwi-operations: create, update         | Partly   |
| lang-source-naming-validate (4)               | not in retriever                        | Out      |
| mcp-access-ceiling (7)                        | mcp-server: ceilings                    | Partly   |
| mcp-destructive-tools (8)                     | mcp-server: admin tools                 | Partly   |
| mcp-every-outline-tool (11)                   | no scenario; tools ship                 | Gap      |
| mcp-server (58)                               | mcp-server (13 scenarios)               | Partly   |
| mcp-tool-profile (9)                          | mcp-server: list_sources                | Partly   |
| mcp-write-tools (16)                          | mcp-server: write ceiling               | Partly   |
| outline-access-requests-admin (7)             | no scenario; tools ship                 | Gap      |
| outline-access-requests-writes (10)           | no scenario; tools ship                 | Gap      |
| outline-admin-reads (4)                       | no scenario; tools ship                 | Gap      |
| outline-attachments-destructive (6)           | no scenario; tools ship                 | Gap      |
| outline-auth-whoami (5)                       | outline-reads: account read             | Partly   |
| outline-authentication-providers-admin (10)   | no scenario; tools ship                 | Gap      |
| outline-collections-access (3)                | no scenario; tools ship                 | Gap      |
| outline-collections-destructive (6)           | no scenario; tools ship                 | Gap      |
| outline-collections-list (6)                  | outline-reads: collection list          | Partly   |
| outline-collections-read (11)                 | outline-reads: collection list          | Partly   |
| outline-collections-writes (14)               | outline-writes: documents only          | Gap      |
| outline-comments-destructive (6)              | no scenario; tools ship                 | Gap      |
| outline-comments-read (4)                     | no scenario; tools ship                 | Gap      |
| outline-comments-writes (12)                  | outline-writes: comment                 | Partly   |
| outline-credentials (6)                       | credentials: Outline key                | Covered  |
| outline-critical-tier (49)                    | outline-critical-actions (7)            | Partly   |
| outline-document-export (9)                   | export-markdown-files: Outline          | Partly   |
| outline-document-tree (6)                     | graphql-serve: flat tree                | Partly   |
| outline-documents-access (4)                  | no scenario; tools ship                 | Gap      |
| outline-documents-children (3)                | no scenario; tools ship                 | Gap      |
| outline-documents-destructive (9)             | outline-critical-actions: trash, delete | Partly   |
| outline-documents-insights (3)                | no scenario; tools ship                 | Gap      |
| outline-documents-lists (8)                   | outline-reads: list, paging             | Partly   |
| outline-documents-read (21)                   | outline-reads: get, list, search        | Partly   |
| outline-documents-writes (15)                 | outline-writes: create, update          | Partly   |
| outline-emojis-destructive (6)                | no scenario; tools ship                 | Gap      |
| outline-file-operations-destructive (6)       | no scenario; tools ship                 | Gap      |
| outline-groups-admin (14)                     | no scenario; tools ship                 | Gap      |
| outline-groups-read (4)                       | no scenario; tools ship                 | Gap      |
| outline-hostile-redaction (3)                 | outline-redaction: hostile reply        | Partly   |
| outline-imports-admin (7)                     | no scenario; tools ship                 | Gap      |
| outline-integrations-admin (7)                | outline-critical-actions: integration   | Partly   |
| outline-list-continuation (4)                 | outline-reads: next offset              | Partly   |
| outline-notifications-writes (11)             | no scenario; tools ship                 | Gap      |
| outline-oauth-authentications-destructive (5) | no scenario; tools ship                 | Gap      |
| outline-oauth-clients-admin (6)               | outline-critical-actions: secret        | Partly   |
| outline-output-formats (12)                   | outline-reads: JSON schema              | Partly   |
| outline-part2b-redaction (9)                  | outline-redaction                       | Partly   |
| outline-pins-stars-subscriptions (8)          | no scenario; tools ship                 | Gap      |
| outline-pins-writes (11)                      | outline-writes: pin                     | Partly   |
| outline-rate-limit (11)                       | https-and-limits: Retry-After           | Partly   |
| outline-remaining-reads (2)                   | no scenario; tools ship                 | Gap      |
| outline-revisions-destructive (4)             | no scenario; tools ship                 | Gap      |
| outline-revisions-read (4)                    | no scenario; tools ship                 | Gap      |
| outline-revisions-writes (10)                 | no scenario; tools ship                 | Gap      |
| outline-searches-writes (11)                  | no scenario; tools ship                 | Gap      |
| outline-secret-never-printed (9)              | outline-redaction, credentials          | Partly   |
| outline-shares-admin (11)                     | no scenario; tools ship                 | Gap      |
| outline-shares-read (4)                       | no scenario; tools ship                 | Gap      |
| outline-stars-writes (11)                     | no scenario; tools ship                 | Gap      |
| outline-subscriptions-writes (11)             | no scenario; tools ship                 | Gap      |
| outline-teams-admin (9)                       | no scenario; tools ship                 | Gap      |
| outline-templates-destructive (4)             | no scenario; tools ship                 | Gap      |
| outline-templates-read (3)                    | no scenario; tools ship                 | Gap      |
| outline-templates-writes (10)                 | no scenario; tools ship                 | Gap      |
| outline-user-memberships-writes (10)          | no scenario; tools ship                 | Gap      |
| outline-users-admin (19)                      | outline-critical-actions: user          | Partly   |
| outline-users-read (5)                        | outline-reads: account read             | Partly   |
| outline-users-writes (11)                     | no scenario; tools ship                 | Gap      |
| outline-webhook-subscriptions-admin (5)       | outline-redaction: webhooks             | Partly   |
| pdf-cheat-sheet (3)                           | not in retriever                        | Out      |
| pdf-from-md (24)                              | not in retriever                        | Out      |
| pdf-split (19)                                | not in retriever                        | Out      |
| pdf-unprotect (15)                            | not in retriever                        | Out      |
| repo-governance-workflows-naming-validate (5) | not in retriever                        | Out      |
| serve-graphql (28)                            | graphql-serve (10 scenarios)            | Partly   |
| sources-catalog (55)                          | sources-catalog (15 scenarios)          | Partly   |
| test-coverage-validate (3)                    | not in retriever                        | Out      |
| vault-write-confinement (9)                   | not in retriever                        | Deferred |

"No scenario; tools ship" means the command group is in the operation catalog, so the command exists, and no retriever
scenario covers its behaviour.

## Out of retriever, and why

- **Job report** (`job-report`): it reports on Butler's own job runs.
- **PDF** (`pdf-*`): PDF tools are not a source connector.
- **Naming and coverage checks** (`harness-naming-validate`, `lang-source-naming-validate`,
  `repo-governance-workflows-naming-validate`, `test-coverage-validate`): they check Butler's own files. They are not
  source behaviour. retriever's coverage gate is `scripts/coverage-gate.sh`.
- **Vault layouts** (in `jira-export`, `kiwi-export`, and `outline-document-export`): the vault layout stays in Butler.
  retriever writes plain Markdown with `--out`. The Partly status covers only that Markdown export.

## Deferred

- **Vault write confinement** (`vault-write-confinement`): the vault source kind is planned for v0.2.0 or a later
  release. It is not a v0.1.0 gap.

## Follow-up

The figures below come from a count by heading, which the title table above does not measure. Each Butler scenario
heading has one verdict. Before batch B1, 817 Butler headings were open: 768 with no retriever scenario, and 49 covered
only in part.

Batch B1 (trust boundary) covers 106 of those open headings in 13 Butler files. Its first part, B1a (credentials and
secret redaction), closes 53 of the 106 and adds 38 scenarios. Some closed headings are asserted by existing scenarios,
and some of those scenarios were strengthened. After B1a, 764 headings stay open. Of those, 53 are in B1: the
transport, rate-limit, JSON-RPC error, and Jira API error rows, plus two redaction rows, one for a hostile reply through
an MCP tool and one for the OAuth authentications tool. Batch B1b adds scenarios for those rows, and the retriever specs
grow from 194 to 235 scenario headings. This change does not recount the Butler headings it closes, so the figure of
764 open headings is the last counted one and is not restated here. The Jira repeated-page-token row does not close as
an unexpected response: search reads one page and marks it truncated. Each remaining row closes in its own change, with
its scenario written first. The outline-secret-never-printed row stays Partly until its Outline logs check has a
scenario.
