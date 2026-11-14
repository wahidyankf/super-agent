# Ordinary Outline writes: creating and changing documents, comments, and pins. Each one sends exactly one request. The
# wiki's own answer, such as a refusal, is the only check on a change.
Feature: Outline writes
  As a person who adds to and edits an Outline wiki from the command line
  I want each write to send exactly one request when it runs
  So that a change reaches the wiki once, and a refusal from the wiki is reported as it is

  Background:
    Given the environment variable "EXAMPLE_OUTLINE_API_KEY" is set to "synthetic-outline-key"
    And the stub Outline server holds these documents:
      | id    | collection | parent | title | markdown           |
      | d0c1d0c1-0000-4000-8000-000000000001 | 1c1c1c1c-0000-4000-8000-000000000001         |        | Setup | Synthetic body one |
    And the stub Outline server holds these collections:
      | id | name      |
      | 1c1c1c1c-0000-4000-8000-000000000001 | Synthetic |

  @integration @e2e @unit
  Scenario: Creating a document sends one request that carries its title
    Given the sources catalog holds these sources:
      | id              | kind    | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    When I run "retriever outline documents create --title Synthetic-note --text Synthetic --collection-id 1c1c1c1c-0000-4000-8000-000000000001 --source example-outline"
    Then the exit status is 0
    And the stub Outline server received 1 request to "documents.create"
    And the stub Outline server's last request to "documents.create" body contains "Synthetic-note"

  @integration @e2e @unit
  Scenario: A comment sends one request that carries its text
    Given the sources catalog holds these sources:
      | id              | kind    | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    When I run "retriever outline comments create d0c1d0c1-0000-4000-8000-000000000001 --text Synthetic-comment --source example-outline"
    Then the exit status is 0
    And the stub Outline server received 1 request to "comments.create"
    And the stub Outline server's last request to "comments.create" body contains "Synthetic-comment"

  @integration @e2e @unit
  Scenario: Pinning a document sends one request
    Given the sources catalog holds these sources:
      | id              | kind    | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    When I run "retriever outline pins create d0c1d0c1-0000-4000-8000-000000000001 --source example-outline"
    Then the exit status is 0
    And the stub Outline server received 1 request

  @integration @e2e @unit
  Scenario: A write that the wiki refuses reports the refusal and changes nothing else
    Given the sources catalog holds these sources:
      | id              | kind    | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    And the stub Outline server answers "documents.update" with status 403
    When I run "retriever --output json outline documents update d0c1d0c1-0000-4000-8000-000000000001 --title Renamed --source example-outline"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.outline.access_denied" and retryable false
    And the stub Outline server received 1 request to "documents.update"
