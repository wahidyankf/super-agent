# The sources catalog: which sources exist, how a command finds the catalog file, and how a bad catalog is refused.
# Synthetic hosts only (docs.example.test, kiwi.example.test, jira.example.test); keys are never in the catalog.
Feature: Sources catalog
  As a person who connects retriever to my tools
  I want one reviewed catalog that lists every source, its kind, and its tags
  So that every command reaches only the sources I approved

  Background:
    Given the sources catalog holds these sources:
      | id              | kind    | tags           | endpoint                    | credential_env           | identity_env        |
      | example-outline | outline | wiki,docs      | https://docs.example.test   | EXAMPLE_OUTLINE_API_KEY  |                     |
      | example-kiwi    | kiwi    | tests          | https://kiwi.example.test   | EXAMPLE_KIWI_API_KEY     |                     |
      | example-jira    | jira    | tickets        | https://jira.example.test   | EXAMPLE_JIRA_API_TOKEN   | EXAMPLE_JIRA_EMAIL  |

  # Behaviour: CLI-024
  Scenario: The sources list names every configured source and exits 0
    When I run "retriever sources list"
    Then the exit status is 0
    And the standard output contains "example-outline"
    And the standard output contains "example-kiwi"
    And the standard output contains "example-jira"

  # Behaviour: CLI-025
  Scenario: A kind filter lists only the sources of that kind
    When I run "retriever sources list --kind kiwi"
    Then the exit status is 0
    And the standard output contains "example-kiwi"
    And the standard output does not contain "example-outline"
    And the standard output does not contain "example-jira"

  # Behaviour: CLI-026
  Scenario: A tag filter keeps only the sources that carry every tag given
    When I run "retriever sources list --tag wiki,docs"
    Then the exit status is 0
    And the standard output contains "example-outline"
    And the standard output does not contain "example-kiwi"

  # Behaviour: CLI-027
  Scenario: A tag filter that no source satisfies lists no source
    When I run "retriever sources list --tag wiki,tickets"
    Then the exit status is 0
    And the standard output does not contain "example-outline"
    And the standard output does not contain "example-jira"

  # Behaviour: CLI-028
  Scenario: The JSON sources list is a schemaVersion 2 document sorted by source ID
    When I run "retriever --output json sources list"
    Then the exit status is 0
    And the standard output is a JSON document with schemaVersion 2
    And the JSON field "sources.0.id" of the standard output is "example-jira"
    And the JSON field "sources.0.kind" of the standard output is "jira"

  # Behaviour: CLI-029
  Scenario: An unknown kind in the kind filter is a usage error
    When I run "retriever sources list --kind bogus"
    Then the exit status is 2
    And the standard output is empty

  # Behaviour: CLI-030
  Scenario: A catalog that names an unregistered kind lists the registered kinds in alphabetical order
    Given the sources catalog file "reviewed.yml" holds these sources:
      | id     | kind   | tags | endpoint                  | credential_env         | identity_env |
      | notion | notion | docs | https://wiki.example.test | EXAMPLE_NOTION_API_KEY |              |
    When I run "retriever --output json --catalog reviewed.yml sources list"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.sources.unknown_source_kind" and retryable false
    And the standard error contains "unknown kind (known: jira, kiwi, outline)"

  # Behaviour: CLI-031
  Scenario: Without a flag or variable the catalog is read from the current directory
    When I run "retriever sources list"
    Then the exit status is 0
    And the standard output contains "example-outline"

  # Behaviour: CLI-032
  Scenario: A missing catalog stops the sources commands, reports a JSON failure, and runs nothing
    When I run "retriever --output json --catalog missing.yml sources list"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.sources.catalog_missing" and retryable false

  # Behaviour: CLI-033
  Scenario: Two sources with the same ID are refused when the catalog loads
    Given the sources catalog file "reviewed.yml" holds these sources:
      | id        | kind | tags | endpoint                  | credential_env       | identity_env |
      | twin      | kiwi | tests| https://kiwi.example.test | EXAMPLE_KIWI_API_KEY |              |
      | twin      | kiwi | tests| https://kiwi.example.test | EXAMPLE_KIWI_API_KEY |              |
    When I run "retriever --catalog reviewed.yml sources list"
    Then the exit status is 2
    And the standard output is empty

  # Behaviour: CLI-034
  Scenario: A catalog that still sets the removed access field is refused when it loads
    Given the sources catalog file "reviewed.yml" holds these sources:
      | id     | kind    | access | tags    | endpoint                  | credential_env          |
      | ticket | outline | read   | tickets | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |
    When I run "retriever --output json --catalog reviewed.yml sources list"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.sources.catalog_invalid" and retryable false

  # Behaviour: CLI-035
  Scenario: A command that names an unknown source ID is refused and reaches no source
    When I run "retriever --output json outline documents get d0c1d0c1-0000-4000-8000-000000000001 --source no-such-source"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.outline.unknown_source" and retryable false
    And the stub Outline server received no requests

  # Behaviour: CLI-036
  Scenario: A catalog with a field the loader does not know is refused when it loads
    Given the sources catalog file "reviewed.yml" holds this text:
      """
      version: 1
      sources:
        - id: only-kiwi
          kind: kiwi
          tags: [tests]
          endpoint: https://kiwi.example.test
          credential_env: EXAMPLE_KIWI_API_KEY
          flavour: vanilla
      """
    When I run "retriever --output json --catalog reviewed.yml sources list"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.sources.catalog_invalid" and retryable false

  # Behaviour: CLI-037
  Scenario: A catalog with a version other than 1 is refused when it loads
    Given the sources catalog file "reviewed.yml" holds this text:
      """
      version: 2
      sources: []
      """
    When I run "retriever --output json --catalog reviewed.yml sources list"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.sources.catalog_invalid" and retryable false

  # Behaviour: CLI-038
  Scenario: A catalog with a repeated key is refused when it loads
    Given the sources catalog file "reviewed.yml" holds this text:
      """
      version: 1
      sources:
        - id: only-kiwi
          id: only-outline
          kind: kiwi
          tags: [tests]
          endpoint: https://kiwi.example.test
          credential_env: EXAMPLE_KIWI_API_KEY
      """
    When I run "retriever --output json --catalog reviewed.yml sources list"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.sources.catalog_invalid" and retryable false

  # Behaviour: CLI-039
  Scenario: A catalog larger than 64 KiB is refused when it loads
    Given the sources catalog file "reviewed.yml" is 65 KiB of comment lines
    When I run "retriever --output json --catalog reviewed.yml sources list"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.sources.catalog_invalid" and retryable false

  # Behaviour: CLI-040
  Scenario: A malformed catalog does not stop a command that reads no source
    Given the sources catalog file "reviewed.yml" holds this text:
      """
      version: [this is not a catalog
      """
    When I run "retriever --catalog reviewed.yml --version"
    Then the exit status is 0
    And the standard output contains "retriever "

  # Behaviour: CLI-041
  Scenario: An endpoint that is not HTTPS is refused without printing the host
    Given the sources catalog file "reviewed.yml" holds these sources:
      | id        | kind | tags | endpoint                 | credential_env       | identity_env |
      | only-kiwi | kiwi | tests | http://kiwi.example.test | EXAMPLE_KIWI_API_KEY |              |
    When I run "retriever --catalog reviewed.yml sources list"
    Then the exit status is 2
    And the standard output is empty
    And the standard error does not contain "kiwi.example.test"

  # Behaviour: CLI-042
  Scenario: A value pasted into the credential name is refused and never printed
    Given the sources catalog file "reviewed.yml" holds these sources:
      | id        | kind | tags | endpoint                  | credential_env          | identity_env |
      | only-kiwi | kiwi | tests | https://kiwi.example.test | pasted-synthetic-secret |              |
    When I run "retriever --output json --catalog reviewed.yml sources list"
    Then the exit status is 2
    And the standard output is empty
    And the value "pasted-synthetic-secret" never appears in the standard output, the standard error, or the output folder

  # Behaviour: CLI-043
  Scenario: Two sources that name one credential variable for two hosts are refused when the catalog loads
    Given the sources catalog file "reviewed.yml" holds these sources:
      | id        | kind | tags | endpoint                  | credential_env       | identity_env |
      | first     | kiwi | tests | https://kiwi.example.test | EXAMPLE_KIWI_API_KEY |              |
      | second    | kiwi | tests | https://other.example.test | EXAMPLE_KIWI_API_KEY |              |
    When I run "retriever --output json --catalog reviewed.yml sources list"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.sources.catalog_invalid" and retryable false

  # Behaviour: CLI-044
  Scenario: A kiwi command that names an outline source is refused before any request
    When I run "retriever --output json kiwi documents get case-40 --source example-outline"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.kiwi.wrong_source_kind" and retryable false
    And the stub Outline server received no requests

  # Behaviour: CLI-045
  Scenario: An outline command that names a kiwi source is refused before any request
    When I run "retriever --output json outline documents get d0c1d0c1-0000-4000-8000-000000000001 --source example-kiwi"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.outline.wrong_source_kind" and retryable false
    And the stub Outline server received no requests

  # Behaviour: CLI-046
  Scenario: A kiwi command with no kiwi source in the catalog says that none is configured
    Given the sources catalog file "reviewed.yml" holds these sources:
      | id              | kind    | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    When I run "retriever --catalog reviewed.yml kiwi products list"
    Then the exit status is 2
    And the standard output is empty
    And the standard error contains "no kiwi source is configured"

  # Behaviour: CLI-047
  Scenario: Listing sources never prints an endpoint or a credential name
    When I run "retriever sources list"
    Then the exit status is 0
    And the standard output does not contain "https://"
    And the standard output does not contain "docs.example.test"
    And the standard output does not contain "EXAMPLE_OUTLINE_API_KEY"

  # Behaviour: CLI-048
  Scenario: The help of sources list names every known kind
    When I run "retriever sources list --help"
    Then the exit status is 0
    And the standard output contains "outline"
    And the standard output contains "kiwi"
    And the standard output contains "jira"

  # Behaviour: CLI-049
  Scenario: A tag filter with a tag that is not a lowercase slug is a usage error
    When I run "retriever --output json sources list --tag Not_A_Tag"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.sources.invalid_tag" and retryable false

  # Behaviour: CLI-050
  Scenario: A source option that is not a valid source ID is refused before any request
    When I run "retriever --output json outline documents get d0c1d0c1-0000-4000-8000-000000000001 --source Bad_Source"
    Then the exit status is 2
    And the standard output is empty
    And the stub Outline server received no requests

  # Behaviour: CLI-051
  Scenario: The text sources list starts with the count of sources
    When I run "retriever sources list"
    Then the exit status is 0
    And the standard output contains "3 sources"

  # Behaviour: CLI-052
  Scenario: An outline command without a source option is refused when two outline sources exist
    Given the sources catalog file "two.yml" holds these sources:
      | id                  | kind    | tags | endpoint                  | credential_env              | identity_env |
      | example-outline     | outline | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY     |              |
      | example-outline-two | outline | wiki | https://docs.example.test | EXAMPLE_OUTLINE_TWO_API_KEY |              |
    When I run "retriever --output json --catalog two.yml outline documents get d0c1d0c1-0000-4000-8000-000000000001"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.outline.ambiguous_source" and retryable false
    And the stub Outline server received no requests

  # Behaviour: CLI-053
  Scenario: A credential name that does not end in the key or token suffix is refused when the catalog loads
    Given the sources catalog file "reviewed.yml" holds these sources:
      | id        | kind | tags  | endpoint                  | credential_env      | identity_env |
      | only-kiwi | kiwi | tests | https://kiwi.example.test | EXAMPLE_KIWI_SECRET |              |
    When I run "retriever --output json --catalog reviewed.yml sources list"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.sources.catalog_invalid" and retryable false

  # Behaviour: CLI-054
  Scenario: An identity name that does not end in the e-mail suffix is refused when the catalog loads
    Given the sources catalog file "reviewed.yml" holds these sources:
      | id           | kind | tags    | endpoint                  | credential_env         | identity_env       |
      | example-jira | jira | tickets | https://jira.example.test | EXAMPLE_JIRA_API_TOKEN | EXAMPLE_JIRA_USER  |
    When I run "retriever --output json --catalog reviewed.yml sources list"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.sources.catalog_invalid" and retryable false

  # Behaviour: CLI-055
  Scenario: A profile catalog refuses a credential name without the profile prefix
    Given the profile "personal" holds these sources:
      | id           | kind | tags  | endpoint                  | credential_env       | identity_env |
      | example-kiwi | kiwi | tests | https://kiwi.example.test | EXAMPLE_KIWI_API_KEY |              |
    When I run "retriever --output json --profile personal sources list"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.sources.catalog_invalid" and retryable false

  # Behaviour: CLI-056
  Scenario: The text sources list of a profile names the profile
    Given the profile "personal" holds these sources:
      | id           | kind | tags  | endpoint                  | credential_env        | identity_env |
      | example-kiwi | kiwi | tests | https://kiwi.example.test | PERSONAL_KIWI_API_KEY |              |
    When I run "retriever --profile personal sources list"
    Then the exit status is 0
    And the standard output contains "(profile: personal)"

  # Behaviour: CLI-057
  Scenario: The JSON sources list of a profile carries the profile name after schemaVersion
    Given the profile "personal" holds these sources:
      | id           | kind | tags  | endpoint                  | credential_env        | identity_env |
      | example-kiwi | kiwi | tests | https://kiwi.example.test | PERSONAL_KIWI_API_KEY |              |
    When I run "retriever --output json --profile personal sources list"
    Then the exit status is 0
    And the standard output is a JSON document with schemaVersion 2
    And the JSON field "profile" of the standard output is "personal"

  # Behaviour: CLI-201
  Scenario: A plugin whose pin does not match fails and starts no process
    Given the sources catalog file "sources.yml" holds one outline source served by the plugin "plugins/example-plugin" pinned to a digest of 64 zeros
    And the plugin file "plugins/example-plugin" answers every request with one collection list
    And the environment variable "EXAMPLE_OUTLINE_API_KEY" is set to "synthetic-outline-key"
    When I run "retriever --output json --catalog sources.yml outline collections"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.connector.plugin_pin_mismatch" and retryable false
    And no plugin process has started

  # Behaviour: CLI-202
  Scenario: A plugin reply the host refuses is reported with the protocol code
    Given the sources catalog file "sources.yml" holds one outline source served by the plugin "plugins/example-plugin" pinned to the digest of its file
    And the plugin file "plugins/example-plugin" answers every request with a reply whose id is not the request id
    And the environment variable "EXAMPLE_OUTLINE_API_KEY" is set to "synthetic-outline-key"
    When I run "retriever --output json --catalog sources.yml outline collections"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.connector.plugin_protocol" and retryable false

  # Behaviour: CLI-203
  Scenario: A plugin that cannot start is reported with its start-failure code
    Given the sources catalog file "sources.yml" holds one outline source served by the plugin "plugins/example-plugin" pinned to the digest of its file
    And the plugin file "plugins/example-plugin" cannot start
    And the environment variable "EXAMPLE_OUTLINE_API_KEY" is set to "synthetic-outline-key"
    When I run "retriever --output json --catalog sources.yml outline collections"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.connector.plugin_start_failed" and retryable false
