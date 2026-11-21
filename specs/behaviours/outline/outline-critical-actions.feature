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

  # Behaviour: CLI-491
  # Behaviour: CLI-490
  Scenario Outline: A destructive Outline command runs its change once with no confirmation flag
    Given the sources catalog holds these sources:
      | id              | kind    | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    When I run "retriever outline <command> --source example-outline"
    Then the exit status is 0
    And the stub Outline server received 1 request to "<method>"

    Examples:
      | command                                                                                              | method                    |
      | documents empty-trash                                                                                | documents.empty_trash     |
      | documents delete d0c1d0c1-0000-4000-8000-000000000001                                                | documents.delete          |
      | documents delete d0c1d0c1-0000-4000-8000-000000000001 --permanent true                               | documents.delete          |
      | documents archive d0c1d0c1-0000-4000-8000-000000000001                                               | documents.archive         |
      | documents remove-user d0c1d0c1-0000-4000-8000-000000000001 --user-id 5e52e52e-0000-4000-8000-000000000002 | documents.remove_user |
      | collections delete 1c1c1c1c-0000-4000-8000-000000000001                                              | collections.delete        |
      | collections remove-user 1c1c1c1c-0000-4000-8000-000000000001 --user-id 5e52e52e-0000-4000-8000-000000000002 | collections.remove_user |
      | users delete 5e52e52e-0000-4000-8000-000000000002                                                    | users.delete              |
      | users suspend 5e52e52e-0000-4000-8000-000000000002                                                   | users.suspend             |
      | groups delete 6a6a6a6a-0000-4000-8000-000000000001                                                   | groups.delete            |
      | comments delete 44444444-4444-4444-8444-444444444444                                                 | comments.delete           |
      | attachments delete a7a7a7a7-0000-4000-8000-000000000001                                              | attachments.delete        |
      | emojis delete e1e1e1e1-0000-4000-8000-000000000001                                                   | emojis.delete             |
      | file-operations delete f1f1f1f1-0000-4000-8000-000000000001                                          | fileOperations.delete     |
      | imports delete 1a1a1a1a-0000-4000-8000-000000000001                                                  | imports.delete            |
      | integrations delete b4000000-0000-4000-8000-000000000018                                             | integrations.delete       |
      | oauth-clients delete 0c0c0c0c-0000-4000-8000-000000000001                                            | oauthClients.delete       |
      | oauth-authentications delete b6000000-0000-4000-8000-000000000051                                    | oauthAuthentications.delete |
      | revisions delete e0e0e0e0-0000-4000-8000-000000000001                                                | revisions.delete          |
      | shares revoke 5a5a5a5a-0000-4000-8000-000000000001                                                   | shares.revoke             |
      | templates delete 7e7e7e7e-0000-4000-8000-000000000001                                                | templates.delete          |
      | webhook-subscriptions delete b6000000-0000-4000-8000-000000000021                                    | webhookSubscriptions.delete |
      | authentication-providers delete 9b9b9b9b-0000-4000-8000-000000000001                                 | authenticationProviders.delete |
      | access-requests dismiss a1000000-0000-4000-8000-000000000001                                         | accessRequests.dismiss    |

  # Behaviour: CLI-492
  Scenario: Creating an integration shows the host and never the path
    Given the sources catalog holds these sources:
      | id              | kind    | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    When I run "retriever outline integrations create https://hooks.example.test/synthetic/path --type post --service matomo --source example-outline"
    Then the exit status is 0
    And the standard output does not contain "/synthetic/path"
    And the stub Outline server received 1 request to "integrations.create"

  # Behaviour: CLI-501
  # Behaviour: CLI-493
  Scenario: Rotating an OAuth client secret shows no secret in standard output or standard error
    Given the sources catalog holds these sources:
      | id              | kind    | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
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

  # Behaviour: CLI-494
  Scenario: A destructive Outline command with a rejected key sends its one request, fails, and prints no key
    Given the sources catalog holds these sources:
      | id              | kind    | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    And the stub Outline server answers "documents.delete" with status 401
    When I run "retriever --output json outline documents delete d0c1d0c1-0000-4000-8000-000000000001 --source example-outline"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.outline.key_rejected" and retryable false
    And the value of "EXAMPLE_OUTLINE_API_KEY" never appears in the standard output, the standard error, or the output folder
    And the stub Outline server received 1 request to "documents.delete"
