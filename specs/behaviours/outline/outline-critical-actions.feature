# Destructive and admin changes to an Outline wiki. A deleting or granting change needs --yes. A critical change also
# needs --confirm with the value that its dry run prints for the target. A refused change sends no request.
Feature: Outline critical actions
  As a person who may need to remove content or access in an Outline wiki
  I want every destructive change to be previewed, confirmed by its target, and refused when unconfirmed
  So that a mistyped command or a script cannot empty the trash, delete a user, or open a webhook by accident

  Background:
    Given the environment variable "EXAMPLE_OUTLINE_API_KEY" is set to "synthetic-outline-key"
    And the stub Outline server holds these documents:
      | id    | collection | parent | title | markdown           |
      | d0c1d0c1-0000-4000-8000-000000000001 | 1c1c1c1c-0000-4000-8000-000000000001         |        | Setup | Synthetic body one |
    And the stub Outline server holds these collections:
      | id | name      |
      | 1c1c1c1c-0000-4000-8000-000000000001 | Synthetic |

  @integration @e2e @unit
  Scenario: Emptying the trash with --yes and a matching --confirm sends the request once
    Given the sources catalog holds these sources:
      | id              | kind    | access | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | admin  | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    When I run "retriever outline documents empty-trash --yes --confirm example-outline --source example-outline"
    Then the exit status is 0
    And the stub Outline server received 1 request to "documents.empty_trash"

  @integration @e2e @unit
  Scenario: The dry run of emptying the trash prints the source ID to type and changes nothing
    Given the sources catalog holds these sources:
      | id              | kind    | access | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | admin  | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    When I run "retriever outline documents empty-trash --dry-run --source example-outline"
    Then the exit status is 0
    And the standard output contains "example-outline"
    And the stub Outline server received 0 requests to "documents.empty_trash"

  @integration @e2e @unit
  Scenario Outline: Emptying the trash is refused without the right confirmation, and sends no request
    Given the sources catalog holds these sources:
      | id              | kind    | access | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | admin  | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    When I run "retriever --output json outline documents empty-trash <flags> --source example-outline"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "<code>" and retryable false
    And the stub Outline server received no requests

    Examples:
      | flags                                 | code                                   |
      |                                       | retriever.outline.yes_required         |
      | --yes                                 | retriever.outline.confirm_required     |
      | --yes --confirm wrong-source          | retriever.outline.confirm_mismatch     |

  @integration @e2e @unit
  Scenario: Deleting a document for good with --yes and a matching --confirm sends the request once
    Given the sources catalog holds these sources:
      | id              | kind    | access | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | admin  | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    When I run "retriever outline documents delete d0c1d0c1-0000-4000-8000-000000000001 --permanent true --yes --confirm d0c1d0c1-0000-4000-8000-000000000001 --source example-outline"
    Then the exit status is 0
    And the stub Outline server received 1 request to "documents.delete"

  @integration @e2e @unit
  Scenario: Deleting a document for good with a --confirm that names another document is refused
    Given the sources catalog holds these sources:
      | id              | kind    | access | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | admin  | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    When I run "retriever --output json outline documents delete d0c1d0c1-0000-4000-8000-000000000001 --permanent true --yes --confirm d0c2d0c2-0000-4000-8000-000000000002 --source example-outline"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.outline.confirm_mismatch" and retryable false
    And the stub Outline server received 0 requests to "documents.delete"

  @integration @e2e @unit
  Scenario: Moving a document to the trash without --yes is refused and sends no request
    Given the sources catalog holds these sources:
      | id              | kind    | access | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | admin  | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    When I run "retriever --output json outline documents delete d0c1d0c1-0000-4000-8000-000000000001 --source example-outline"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.outline.yes_required" and retryable false
    And the stub Outline server received no requests

  @integration @e2e @unit
  Scenario: Deleting a user with --yes and a matching --confirm sends the request once
    Given the sources catalog holds these sources:
      | id              | kind    | access | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | admin  | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    And the stub Outline server answers "users.info" with the body:
      """
      {"data": {"id": "5e52e52e-0000-4000-8000-000000000002", "name": "Synthetic user", "role": "member"}}
      """
    When I run "retriever outline users delete 5e52e52e-0000-4000-8000-000000000002 --yes --confirm 5e52e52e-0000-4000-8000-000000000002 --source example-outline"
    Then the exit status is 0
    And the stub Outline server received 1 request to "users.delete"

  @integration @e2e @unit
  Scenario: Deleting the user that owns the API key is refused as a self target even when confirmed
    Given the sources catalog holds these sources:
      | id              | kind    | access | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | admin  | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    And the stub Outline server answers "auth.info" with the body:
      """
      {"data": {"user": {"id": "5e5e5e5e-0000-4000-8000-000000000001", "name": "Synthetic owner", "role": "admin"}, "team": {"id": "7e7e7e7e-0000-4000-8000-000000000001", "name": "Synthetic team"}}}
      """
    When I run "retriever --output json outline users delete 5e5e5e5e-0000-4000-8000-000000000001 --yes --confirm 5e5e5e5e-0000-4000-8000-000000000001 --source example-outline"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.outline.self_target" and retryable false
    And the stub Outline server received 0 requests to "users.delete"

  @integration @e2e @unit
  Scenario: Changing a user's role without --yes is refused and sends no request
    Given the sources catalog holds these sources:
      | id              | kind    | access | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | admin  | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    When I run "retriever --output json outline users update-role 5e52e52e-0000-4000-8000-000000000002 --role admin --source example-outline"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.outline.yes_required" and retryable false
    And the stub Outline server received no requests

  @integration @e2e @unit
  Scenario: A destructive change on a write-access source is refused as an access failure
    Given the sources catalog holds these sources:
      | id              | kind    | access | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | write  | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    When I run "retriever --output json outline documents empty-trash --yes --confirm example-outline --source example-outline"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.outline.access_level_too_low" and retryable false
    And the stub Outline server received no requests

  @integration @e2e @unit
  Scenario: Creating an integration with a matching host confirmation shows the host and never the path
    Given the sources catalog holds these sources:
      | id              | kind    | access | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | admin  | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    When I run "retriever outline integrations create https://hooks.example.test/synthetic/path --type post --service matomo --yes --confirm hooks.example.test --source example-outline"
    Then the exit status is 0
    And the standard output does not contain "/synthetic/path"
    And the stub Outline server received 1 request to "integrations.create"

  @integration @e2e @unit
  Scenario: Creating an integration with the full URL as its confirmation is refused
    Given the sources catalog holds these sources:
      | id              | kind    | access | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | admin  | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    When I run "retriever --output json outline integrations create https://hooks.example.test/synthetic/path --type post --service matomo --yes --confirm https://hooks.example.test/synthetic/path --source example-outline"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.outline.confirm_mismatch" and retryable false
    And the stub Outline server received no requests

  @integration @e2e @unit
  Scenario: The dry run of an integration names the host of its URL and never the path
    Given the sources catalog holds these sources:
      | id              | kind    | access | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | admin  | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    When I run "retriever outline integrations create https://hooks.example.test/synthetic/path --type post --service matomo --dry-run --source example-outline"
    Then the exit status is 0
    And the standard output contains "hooks.example.test"
    And the standard output does not contain "/synthetic/path"
    And the stub Outline server received no requests

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
    When I run "retriever --output json outline oauth-clients rotate-secret 0c0c0c0c-0000-4000-8000-000000000001 --yes --confirm 0c0c0c0c-0000-4000-8000-000000000001 --source example-outline"
    Then the exit status is 0
    And the value "synthetic-client-secret" never appears in the standard output, the standard error, or the output folder
