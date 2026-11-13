# What an Outline reply may put on the screen. A hostile or failing reply can carry terminal controls, a key, a client
# secret, or a signing secret. Retriever prints only what the public output shows, and never the secret itself.
Feature: Outline reply redaction
  As a person who reads content from an Outline wiki that other people write
  I want terminal controls, keys, and secrets in a reply to stay off my screen and out of my logs
  So that a hostile page or a leaked key cannot take over a terminal or leave the process

  Background:
    Given the sources catalog holds these sources:
      | id              | kind    | access | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | admin  | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    And the environment variable "EXAMPLE_OUTLINE_API_KEY" is set to "synthetic-outline-key"

  @integration @e2e @unit
  Scenario: A title with an escape sequence leaves no terminal control byte in the text output
    Given the stub Outline server answers "documents.info" with the body:
      """
      {"data": {"id": "d0c1d0c1-0000-4000-8000-000000000001", "title": "\u001b[31mRed title", "text": "Synthetic body", "url": "/doc/d0c1d0c1-0000-4000-8000-000000000001", "collectionId": "1c1c1c1c-0000-4000-8000-000000000001", "revision": 1}}
      """
    When I run "retriever outline documents get d0c1d0c1-0000-4000-8000-000000000001"
    Then the exit status is 0
    And the standard output contains no control characters except line breaks

  @integration @e2e @unit
  Scenario: A document body with an escape sequence leaves no terminal control byte in the text output
    Given the stub Outline server answers "documents.info" with the body:
      """
      {"data": {"id": "d0c1d0c1-0000-4000-8000-000000000001", "title": "Setup", "text": "Before \u001b]0;hostile\u0007 after", "url": "/doc/d0c1d0c1-0000-4000-8000-000000000001", "collectionId": "1c1c1c1c-0000-4000-8000-000000000001", "revision": 1}}
      """
    When I run "retriever outline documents get d0c1d0c1-0000-4000-8000-000000000001"
    Then the exit status is 0
    And the standard output contains no control characters except line breaks

  @integration @e2e @unit
  Scenario: A hostile reply in JSON mode is still a single JSON document with no raw control byte
    Given the stub Outline server answers "documents.info" with the body:
      """
      {"data": {"id": "d0c1d0c1-0000-4000-8000-000000000001", "title": "\u001b[2JCleared", "text": "Synthetic body", "url": "/doc/d0c1d0c1-0000-4000-8000-000000000001", "collectionId": "1c1c1c1c-0000-4000-8000-000000000001", "revision": 1}}
      """
    When I run "retriever --output json outline documents get d0c1d0c1-0000-4000-8000-000000000001"
    Then the exit status is 0
    And the standard output is a JSON document with schemaVersion 1
    And the standard output contains no control characters except line breaks

  @integration @e2e @unit
  Scenario: The webhook list prints the host of each URL and never its path, query, or signing secret
    Given the stub Outline server answers "webhookSubscriptions.list" with the body:
      """
      {"data": [{"id": "b6000000-0000-4000-8000-000000000021", "name": "Synthetic hook", "url": "https://hooks.example.test/synthetic/path?token=synthetic-query-token", "secret": "synthetic-signing-secret", "enabled": true, "events": ["documents.update"], "createdById": "33333333-3333-4333-8333-333333333333", "createdBy": {"id": "33333333-3333-4333-8333-333333333333", "name": "Synthetic Person"}, "createdAt": "2026-01-01T00:00:00Z", "updatedAt": "2026-01-02T00:00:00Z"}], "pagination": {"offset": 0, "limit": 25}}
      """
    When I run "retriever outline webhook-subscriptions list"
    Then the exit status is 0
    And the standard output contains "hooks.example.test"
    And the standard output does not contain "/synthetic/path"
    And the standard output does not contain "synthetic-query-token"
    And the standard output does not contain "synthetic-signing-secret"

  @integration @e2e @unit
  Scenario: The webhook list JSON document never holds the signing secret or the URL path
    Given the stub Outline server answers "webhookSubscriptions.list" with the body:
      """
      {"data": [{"id": "b6000000-0000-4000-8000-000000000021", "name": "Synthetic hook", "url": "https://hooks.example.test/synthetic/path?token=synthetic-query-token", "secret": "synthetic-signing-secret", "enabled": true, "events": ["documents.update"], "createdById": "33333333-3333-4333-8333-333333333333", "createdBy": {"id": "33333333-3333-4333-8333-333333333333", "name": "Synthetic Person"}, "createdAt": "2026-01-01T00:00:00Z", "updatedAt": "2026-01-02T00:00:00Z"}], "pagination": {"offset": 0, "limit": 25}}
      """
    When I run "retriever --output json outline webhook-subscriptions list"
    Then the exit status is 0
    And the standard output is a JSON document with schemaVersion 1
    And the standard output does not contain "synthetic-signing-secret"
    And the standard output does not contain "/synthetic/path"

  @integration @e2e @unit
  Scenario Outline: A failing Outline response never prints the key
    Given the stub Outline server answers "collections.list" with status <status>
    When I run "retriever --output json outline collections list"
    Then the exit status is 2
    And the standard output is empty
    And the value of "EXAMPLE_OUTLINE_API_KEY" never appears in the standard output, the standard error, or the output folder

    Examples:
      | status |
      | 403    |
      | 404    |
      | 429    |
      | 500    |

  @integration @e2e @unit
  Scenario: A non-JSON reply that echoes the Authorization header never prints the key
    Given the stub Outline server echoes the Authorization header in a non-JSON reply to "collections.list"
    When I run "retriever --output json outline collections list"
    Then the exit status is 2
    And the value of "EXAMPLE_OUTLINE_API_KEY" never appears in the standard output, the standard error, or the output folder

  @integration @e2e @unit
  Scenario: A successful account read never prints the key
    Given the stub Outline server holds the signed-in user "Synthetic Author" of team "Synthetic Team"
    When I run "retriever --output json outline auth whoami"
    Then the exit status is 0
    And the value of "EXAMPLE_OUTLINE_API_KEY" never appears in the standard output, the standard error, or the output folder
