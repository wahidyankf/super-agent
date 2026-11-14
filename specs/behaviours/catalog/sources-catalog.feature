# The sources catalog: which sources exist, how a command finds the catalog file, and how a bad catalog is refused.
# Synthetic hosts only (docs.example.test, kiwi.example.test, jira.example.test); keys are never in the catalog.
Feature: Sources catalog
  As a person who connects retriever to my tools
  I want one reviewed catalog that lists every source, its access level, and its tags
  So that every command reaches only the sources I approved

  Background:
    Given the sources catalog holds these sources:
      | id              | kind    | access | tags           | endpoint                    | credential_env           | identity_env        |
      | example-outline | outline | read   | wiki,docs      | https://docs.example.test   | EXAMPLE_OUTLINE_API_KEY  |                     |
      | example-kiwi    | kiwi    | read   | tests          | https://kiwi.example.test   | EXAMPLE_KIWI_API_KEY     |                     |
      | example-jira    | jira    | read   | tickets        | https://jira.example.test   | EXAMPLE_JIRA_API_TOKEN   | EXAMPLE_JIRA_EMAIL  |

  @integration @e2e @unit
  Scenario: The sources list names every configured source and exits 0
    When I run "retriever sources list"
    Then the exit status is 0
    And the standard output contains "example-outline"
    And the standard output contains "example-kiwi"
    And the standard output contains "example-jira"

  @integration @e2e @unit
  Scenario: The sources list shows the access level of each source
    When I run "retriever sources list"
    Then the exit status is 0
    And the standard output contains "read"

  @integration @e2e @unit
  Scenario: A kind filter lists only the sources of that kind
    When I run "retriever sources list --kind kiwi"
    Then the exit status is 0
    And the standard output contains "example-kiwi"
    And the standard output does not contain "example-outline"
    And the standard output does not contain "example-jira"

  @integration @e2e @unit
  Scenario: A tag filter keeps only the sources that carry every tag given
    When I run "retriever sources list --tag wiki,docs"
    Then the exit status is 0
    And the standard output contains "example-outline"
    And the standard output does not contain "example-kiwi"

  @integration @e2e @unit
  Scenario: A tag filter that no source satisfies lists no source
    When I run "retriever sources list --tag wiki,tickets"
    Then the exit status is 0
    And the standard output does not contain "example-outline"
    And the standard output does not contain "example-jira"

  @integration @e2e @unit
  Scenario: The JSON sources list is a schemaVersion 1 document with one entry per source
    When I run "retriever --output json sources list"
    Then the exit status is 0
    And the standard output is a JSON document with schemaVersion 1
    And the JSON field "sources.0.id" of the standard output is "example-jira"
    And the JSON field "sources.0.kind" of the standard output is "jira"

  @integration @e2e @unit
  Scenario: An unknown kind in the kind filter is a usage error
    When I run "retriever sources list --kind bogus"
    Then the exit status is 2
    And the standard output is empty

  @integration @e2e @unit
  Scenario: The catalog flag names the catalog file to read
    Given the sources catalog file "reviewed.yml" holds these sources:
      | id          | kind    | access | tags | endpoint                  | credential_env          | identity_env |
      | only-kiwi   | kiwi    | read   | tests| https://kiwi.example.test | EXAMPLE_KIWI_API_KEY    |              |
    When I run "retriever --catalog reviewed.yml sources list"
    Then the exit status is 0
    And the standard output contains "only-kiwi"
    And the standard output does not contain "example-outline"

  @integration @e2e @unit
  Scenario: The catalog flag wins over the catalog environment variable
    Given the environment variable "RETRIEVER_CATALOG" is set to "missing.yml"
    And the sources catalog file "reviewed.yml" holds these sources:
      | id        | kind | access | tags | endpoint                  | credential_env       | identity_env |
      | only-kiwi | kiwi | read   | tests| https://kiwi.example.test | EXAMPLE_KIWI_API_KEY |              |
    When I run "retriever --catalog reviewed.yml sources list"
    Then the exit status is 0
    And the standard output contains "only-kiwi"

  @integration @e2e @unit
  Scenario: The catalog environment variable names the catalog when no flag is given
    Given the environment variable "RETRIEVER_CATALOG" is set to "reviewed.yml"
    And the sources catalog file "reviewed.yml" holds these sources:
      | id        | kind | access | tags | endpoint                  | credential_env       | identity_env |
      | only-kiwi | kiwi | read   | tests| https://kiwi.example.test | EXAMPLE_KIWI_API_KEY |              |
    When I run "retriever sources list"
    Then the exit status is 0
    And the standard output contains "only-kiwi"

  @integration @e2e @unit
  Scenario: Without a flag or variable the catalog is read from the current directory
    When I run "retriever sources list"
    Then the exit status is 0
    And the standard output contains "example-outline"

  @integration @e2e @unit
  Scenario: A missing catalog stops the sources commands, reports a JSON failure, and runs nothing
    When I run "retriever --output json --catalog missing.yml sources list"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.sources.catalog_missing" and retryable false

  @integration @e2e @unit
  Scenario: Two sources with the same ID are refused when the catalog loads
    Given the sources catalog file "reviewed.yml" holds these sources:
      | id        | kind | access | tags | endpoint                  | credential_env       | identity_env |
      | twin      | kiwi | read   | tests| https://kiwi.example.test | EXAMPLE_KIWI_API_KEY |              |
      | twin      | kiwi | read   | tests| https://kiwi.example.test | EXAMPLE_KIWI_API_KEY |              |
    When I run "retriever --catalog reviewed.yml sources list"
    Then the exit status is 2
    And the standard output is empty

  @integration @e2e @unit
  Scenario: A Jira source above its kind ceiling is refused when the catalog loads
    Given the sources catalog file "reviewed.yml" holds these sources:
      | id        | kind | access | tags    | endpoint                  | credential_env         | identity_env       |
      | ticket    | jira | write  | tickets | https://jira.example.test | EXAMPLE_JIRA_API_TOKEN | EXAMPLE_JIRA_EMAIL |
    When I run "retriever --output json --catalog reviewed.yml sources list"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.jira.access_above_kind_ceiling" and retryable false

  @integration @e2e @unit
  Scenario: A command that names an unknown source ID is refused and reaches no source
    When I run "retriever --output json outline documents get d0c1d0c1-0000-4000-8000-000000000001 --source no-such-source"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.outline.unknown_source" and retryable false
    And the stub Outline server received no requests
