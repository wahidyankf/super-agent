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

  # Behaviour: CLI-484
  Scenario: Creating a document sends one request that carries its title
    Given the sources catalog holds these sources:
      | id              | kind    | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    When I run "retriever outline documents create --title Synthetic-note --text Synthetic --collection-id 1c1c1c1c-0000-4000-8000-000000000001 --source example-outline"
    Then the exit status is 0
    And the stub Outline server received 1 request to "documents.create"
    And the stub Outline server's last request to "documents.create" body contains "Synthetic-note"

  # Behaviour: CLI-485
  Scenario: A comment sends one request that carries its text
    Given the sources catalog holds these sources:
      | id              | kind    | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    When I run "retriever outline comments create d0c1d0c1-0000-4000-8000-000000000001 --text Synthetic-comment --source example-outline"
    Then the exit status is 0
    And the stub Outline server received 1 request to "comments.create"
    And the stub Outline server's last request to "comments.create" body contains "Synthetic-comment"

  # Behaviour: CLI-486
  Scenario: Pinning a document sends one request
    Given the sources catalog holds these sources:
      | id              | kind    | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    When I run "retriever outline pins create d0c1d0c1-0000-4000-8000-000000000001 --source example-outline"
    Then the exit status is 0
    And the stub Outline server received 1 request

  # Behaviour: CLI-487
  Scenario Outline: An ordinary Outline write sends exactly one request to its own endpoint
    Given the sources catalog holds these sources:
      | id              | kind    | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    When I run "retriever outline <command> --source example-outline"
    Then the exit status is 0
    And the stub Outline server received 1 request to "<method>"

    Examples:
      | command                                                                                                | method                         |
      | documents duplicate d0c1d0c1-0000-4000-8000-000000000001                                               | documents.duplicate            |
      | documents move d0c1d0c1-0000-4000-8000-000000000001 --collection-id 1c1c1c1c-0000-4000-8000-000000000001 | documents.move                 |
      | documents unpublish d0c1d0c1-0000-4000-8000-000000000001                                               | documents.unpublish            |
      | documents restore d0c1d0c1-0000-4000-8000-000000000001                                                 | documents.restore              |
      | collections create Synthetic-collection                                                                | collections.create             |
      | collections update 1c1c1c1c-0000-4000-8000-000000000001 --name Renamed-collection                       | collections.update             |
      | collections duplicate 1c1c1c1c-0000-4000-8000-000000000001                                             | collections.duplicate          |
      | comments update 44444444-4444-4444-8444-444444444444 --text Synthetic-edit                              | comments.update                |
      | comments resolve 44444444-4444-4444-8444-444444444444                                                  | comments.resolve               |
      | comments unresolve 44444444-4444-4444-8444-444444444444                                                | comments.unresolve             |
      | pins delete c1000000-0000-4000-8000-000000000001                                                       | pins.delete                    |
      | stars create --document-id d0c1d0c1-0000-4000-8000-000000000001                                        | stars.create                   |
      | templates create --title Synthetic-template                                                            | templates.create               |
      | templates restore 7e7e7e7e-0000-4000-8000-000000000001                                                 | templates.restore              |
      | groups create Synthetic-group                                                                           | groups.create                  |
      | groups update 6a6a6a6a-0000-4000-8000-000000000001 --name Renamed-group                                | groups.update                  |
      | teams update --name Renamed-team                                                                       | teams.update                   |
      | users activate 5e52e52e-0000-4000-8000-000000000002                                                    | users.activate                 |
      | users update-role 5e52e52e-0000-4000-8000-000000000002 --role admin                                    | users.update_role              |
      | documents add-user d0c1d0c1-0000-4000-8000-000000000001 --user-id 5e52e52e-0000-4000-8000-000000000002  | documents.add_user             |
      | collections add-user 1c1c1c1c-0000-4000-8000-000000000001 --user-id 5e52e52e-0000-4000-8000-000000000002 | collections.add_user         |
      | groups add-user 6a6a6a6a-0000-4000-8000-000000000001 --user-id 5e52e52e-0000-4000-8000-000000000002    | groups.add_user                |
      | access-requests create d0c1d0c1-0000-4000-8000-000000000001                                            | accessRequests.create          |
      | access-requests approve a1000000-0000-4000-8000-000000000001                                           | accessRequests.approve         |
      | revisions update e0e0e0e0-0000-4000-8000-000000000001 --name Renamed-revision                          | revisions.update               |
      | imports cancel 1a1a1a1a-0000-4000-8000-000000000001                                                    | imports.cancel                 |
      | oauth-clients update 0c0c0c0c-0000-4000-8000-000000000001 --name Renamed-client                        | oauthClients.update            |
      | integrations update b4000000-0000-4000-8000-000000000018 --url https://hooks.example.test/synthetic/new | integrations.update          |
      | authentication-providers start-group-sync 9b9b9b9b-0000-4000-8000-000000000001                         | authenticationProviders.startGroupSync |

  # Behaviour: CLI-488
  Scenario: A write that the wiki refuses reports the refusal and is sent once
    Given the sources catalog holds these sources:
      | id              | kind    | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    And the stub Outline server answers "documents.update" with status 403
    When I run "retriever --output json outline documents update d0c1d0c1-0000-4000-8000-000000000001 --title Renamed --source example-outline"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.outline.access_denied" and retryable false
    And the stub Outline server received 1 request to "documents.update"
