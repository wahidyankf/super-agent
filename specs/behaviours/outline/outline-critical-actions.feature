# Destructive changes to an Outline wiki. Retriever runs the change that the command names, with no confirmation flag.
# Whether the wiki allows the change is decided by the API token's own permissions. Credential and transport rules still
# apply: a reply never shows a secret, and a URL shows only its host.
Feature: Outline critical actions
  As a person who runs destructive Outline commands
  I want each change to run as the command names it, with no extra confirmation flag
  So that the command stays plain, and the token's own permissions decide what the wiki allows

  Background:
    Given the environment variable "EXAMPLE_OUTLINE_API_KEY" is set to "synthetic-outline-key"
    And the stub Outline server holds these documents:
      | id    | collection | parent | title | markdown           |
      | d0c1d0c1-0000-4000-8000-000000000001 | 1c1c1c1c-0000-4000-8000-000000000001         |        | Setup | Synthetic body one |
    And the stub Outline server holds these collections:
      | id | name      |
      | 1c1c1c1c-0000-4000-8000-000000000001 | Synthetic |

  @integration @e2e @unit
  Scenario: Emptying the trash sends the request once
    Given the sources catalog holds these sources:
      | id              | kind    | access | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | admin  | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    When I run "retriever outline documents empty-trash --source example-outline"
    Then the exit status is 0
    And the stub Outline server received 1 request to "documents.empty_trash"

  @integration @e2e @unit
  Scenario: Deleting a document for good sends the request once
    Given the sources catalog holds these sources:
      | id              | kind    | access | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | admin  | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    When I run "retriever outline documents delete d0c1d0c1-0000-4000-8000-000000000001 --permanent true --source example-outline"
    Then the exit status is 0
    And the stub Outline server received 1 request to "documents.delete"

  @integration @e2e @unit
  Scenario: Moving a document to the trash sends the request once
    Given the sources catalog holds these sources:
      | id              | kind    | access | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | admin  | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    When I run "retriever outline documents delete d0c1d0c1-0000-4000-8000-000000000001 --source example-outline"
    Then the exit status is 0
    And the stub Outline server received 1 request to "documents.delete"

  @integration @e2e @unit
  Scenario: Deleting a user sends the request once
    Given the sources catalog holds these sources:
      | id              | kind    | access | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | admin  | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    When I run "retriever outline users delete 5e52e52e-0000-4000-8000-000000000002 --source example-outline"
    Then the exit status is 0
    And the stub Outline server received 1 request to "users.delete"

  @integration @e2e @unit
  Scenario: A destructive change on a write-access source is refused as an access failure
    Given the sources catalog holds these sources:
      | id              | kind    | access | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | write  | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    When I run "retriever --output json outline documents empty-trash --source example-outline"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.outline.access_level_too_low" and retryable false
    And the stub Outline server received no requests

  @integration @e2e @unit
  Scenario: Creating an integration shows the host and never the path
    Given the sources catalog holds these sources:
      | id              | kind    | access | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | admin  | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    When I run "retriever outline integrations create https://hooks.example.test/synthetic/path --type post --service matomo --source example-outline"
    Then the exit status is 0
    And the standard output does not contain "/synthetic/path"
    And the stub Outline server received 1 request to "integrations.create"

  @integration @e2e @unit
  Scenario: Rotating an OAuth client secret shows no secret in standard output or standard error
    Given the sources catalog holds these sources:
      | id              | kind    | access | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | admin  | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    And the stub Outline server answers "oauthClients.rotateSecret" with the body:
      """
      {"data": {"id": "0c0c0c0c-0000-4000-8000-000000000001", "name": "Synthetic client", "clientSecret": "synthetic-client-secret"}}
      """
    And the stub Outline server answers "oauthClients.info" with the body:
      """
      {"data": {"id": "0c0c0c0c-0000-4000-8000-000000000001", "name": "Synthetic client"}}
      """
    When I run "retriever --output json outline oauth-clients rotate-secret 0c0c0c0c-0000-4000-8000-000000000001 --source example-outline"
    Then the exit status is 0
    And the value "synthetic-client-secret" never appears in the standard output, the standard error, or the output folder
