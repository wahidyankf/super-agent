# Credentials: where each source's key comes from, what happens when it is missing or refused, and where it may go.
# All values are synthetic. The keys never appear in the catalog and never appear in any output.
Feature: Source credentials
  As a person who keeps keys out of files that get shared
  I want each key read from the environment or the env file, sent only to its own source, and never printed
  So that a leaked transcript, log, or catalog never exposes a credential

  Background:
    Given the sources catalog holds these sources:
      | id              | kind    | access | tags      | endpoint                  | credential_env          | identity_env       |
      | example-outline | outline | read   | wiki      | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |                    |
      | example-kiwi    | kiwi    | read   | tests     | https://kiwi.example.test | EXAMPLE_KIWI_API_KEY    |                    |
      | example-jira    | jira    | read   | tickets   | https://jira.example.test | EXAMPLE_JIRA_API_TOKEN  | EXAMPLE_JIRA_EMAIL |
    And the stub Outline server holds these collections:
      | id | name       |
      | 1c1c1c1c-0000-4000-8000-000000000001 | Synthetic  |

  @integration @e2e @unit
  Scenario: The Outline key is read from the process environment and sent as a Bearer header
    Given the environment variable "EXAMPLE_OUTLINE_API_KEY" is set to "synthetic-outline-key"
    When I run "retriever outline collections list"
    Then the exit status is 0
    And the stub Outline server received at least 1 request
    And every request to the stub Outline server carried the Authorization header "Bearer synthetic-outline-key"

  @integration @e2e @unit
  Scenario: The Outline key is read from the env file in the current directory
    Given the env file ".env.local" holds these lines:
      """
      EXAMPLE_OUTLINE_API_KEY=synthetic-outline-key
      """
    When I run "retriever outline collections list"
    Then the exit status is 0
    And every request to the stub Outline server carried the Authorization header "Bearer synthetic-outline-key"

  @integration @e2e @unit
  Scenario: The process environment wins over the env file for the same variable
    Given the environment variable "EXAMPLE_OUTLINE_API_KEY" is set to "synthetic-from-env"
    And the env file ".env.local" holds these lines:
      """
      EXAMPLE_OUTLINE_API_KEY=synthetic-from-file
      """
    When I run "retriever outline collections list"
    Then the exit status is 0
    And every request to the stub Outline server carried the Authorization header "Bearer synthetic-from-env"

  @integration @e2e @unit
  Scenario: A Kiwi key is sent only to the Kiwi source and never to the Outline source
    Given the environment variable "EXAMPLE_KIWI_API_KEY" is set to "synthetic-kiwi-key"
    And the environment variable "EXAMPLE_OUTLINE_API_KEY" is set to "synthetic-outline-key"
    When I run "retriever kiwi products list"
    Then the exit status is 0
    And every request to the stub Kiwi server carried the Authorization header "Bearer synthetic-kiwi-key"
    And the stub Outline server received no requests

  @integration @e2e @unit
  Scenario Outline: A missing key stops the command before any request is sent
    Given the environment variable "<variable>" is not set
    And the environment variable "EXAMPLE_JIRA_EMAIL" is set to "synthetic-user@example.test"
    When I run "retriever --output json <command>"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "<code>" and retryable false
    And the stub <kind> server received no requests

    Examples:
      | variable                | command                  | code                         | kind    |
      | EXAMPLE_OUTLINE_API_KEY | outline collections list | retriever.outline.key_missing | Outline |
      | EXAMPLE_KIWI_API_KEY    | kiwi products list       | retriever.kiwi.key_missing    | Kiwi    |
      | EXAMPLE_JIRA_API_TOKEN  | jira whoami              | retriever.jira.key_missing    | Jira    |

  @integration @e2e @unit
  Scenario: An empty key counts as a missing key
    Given the environment variable "EXAMPLE_OUTLINE_API_KEY" is set to ""
    When I run "retriever --output json outline collections list"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.outline.key_missing" and retryable false
    And the stub Outline server received no requests

  @integration @e2e @unit
  Scenario: A missing Jira e-mail address stops the command before any request is sent
    Given the environment variable "EXAMPLE_JIRA_API_TOKEN" is set to "synthetic-jira-token"
    When I run "retriever --output json jira whoami"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.jira.key_missing" and retryable false
    And the stub Jira server received no requests

  @integration @e2e @unit
  Scenario Outline: A malformed Jira e-mail address stops the command before any request and is never shown
    Given the environment variable "EXAMPLE_JIRA_API_TOKEN" is set to "synthetic-jira-token"
    And the environment variable "EXAMPLE_JIRA_EMAIL" is set to "<address>"
    When I run "retriever --output json jira whoami"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.jira.identity_invalid" and retryable false
    And the stub Jira server received no requests
    And the value "<address>" never appears in the standard output, the standard error, or the output folder

    Examples:
      | address             |
      | no-at-sign          |
      | two@@example.test   |

  @integration @e2e @unit
  Scenario: A key that the source refuses is reported with its own code and never printed
    Given the environment variable "EXAMPLE_KIWI_API_KEY" is set to "synthetic-kiwi-key"
    And the stub Kiwi server answers every request with status 401
    When I run "retriever --output json kiwi products list"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.kiwi.key_rejected" and retryable false
    And the value of "EXAMPLE_KIWI_API_KEY" never appears in the standard output, the standard error, or the output folder

  @integration @e2e @unit
  Scenario: An Outline key that the source refuses is reported with its own code
    Given the environment variable "EXAMPLE_OUTLINE_API_KEY" is set to "synthetic-outline-key"
    And the stub Outline server answers "collections.list" with status 401
    When I run "retriever --output json outline collections list"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.outline.key_rejected" and retryable false
    And the value of "EXAMPLE_OUTLINE_API_KEY" never appears in the standard output, the standard error, or the output folder

  @integration @e2e @unit
  Scenario: A Jira token that the source refuses gets the same text whichever value is wrong
    Given the environment variable "EXAMPLE_JIRA_API_TOKEN" is set to "synthetic-wrong-token"
    And the environment variable "EXAMPLE_JIRA_EMAIL" is set to "synthetic-user@example.test"
    And the stub Jira server answers every request with status 401
    When I run "retriever --output json jira whoami"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.jira.key_rejected" and retryable false
    And the stub Jira server received 1 request

  @integration @e2e @unit
  Scenario: A key is never sent in the body or the URL of a request
    Given the environment variable "EXAMPLE_OUTLINE_API_KEY" is set to "synthetic-outline-key"
    When I run "retriever outline collections list"
    Then the exit status is 0
    And no request to the stub Outline server carried the value of "EXAMPLE_OUTLINE_API_KEY" in its body or its URL

  @integration @e2e @unit
  Scenario: A successful run never prints the key in standard output, standard error, or files
    Given the environment variable "EXAMPLE_OUTLINE_API_KEY" is set to "synthetic-outline-key"
    When I run "retriever outline collections list"
    Then the exit status is 0
    And the value of "EXAMPLE_OUTLINE_API_KEY" never appears in the standard output, the standard error, or the output folder
