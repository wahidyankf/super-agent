# What an Outline reply may put on the screen. A hostile or failing reply can carry terminal controls, a key, a client
# secret, or a signing secret. Retriever prints only what the public output shows, and never the secret itself.
Feature: Outline reply redaction
  As a person who reads content from an Outline wiki that other people write
  I want terminal controls, keys, and secrets in a reply to stay off my screen and out of my logs
  So that a hostile page or a leaked key cannot take over a terminal or leave the process

  Background:
    Given the sources catalog holds these sources:
      | id              | kind    | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
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
  Scenario Outline: A hostile title never prints a control byte in the text output, whichever one it holds
    Given the stub Outline server answers "documents.info" with the body:
      """
      {"data": {"id": "d0c1d0c1-0000-4000-8000-000000000001", "title": "<control>Red title", "text": "Synthetic body", "url": "/doc/d0c1d0c1-0000-4000-8000-000000000001", "collectionId": "1c1c1c1c-0000-4000-8000-000000000001", "revision": 1}}
      """
    When I run "retriever outline documents get d0c1d0c1-0000-4000-8000-000000000001"
    Then the exit status is 0
    And the standard output contains no control characters except line breaks

    Examples:
      | control |
      | \u0007  |
      | \u0008  |
      | \u000c  |
      | \u007f  |

  @integration @e2e @unit
  Scenario Outline: A hostile reply in JSON mode never shows a raw control byte, whichever one it holds
    Given the stub Outline server answers "documents.info" with the body:
      """
      {"data": {"id": "d0c1d0c1-0000-4000-8000-000000000001", "title": "<control>Cleared", "text": "Synthetic body", "url": "/doc/d0c1d0c1-0000-4000-8000-000000000001", "collectionId": "1c1c1c1c-0000-4000-8000-000000000001", "revision": 1}}
      """
    When I run "retriever --output json outline documents get d0c1d0c1-0000-4000-8000-000000000001"
    Then the exit status is 0
    And the standard output is a JSON document with schemaVersion 1
    And the standard output contains no control characters except line breaks

    Examples:
      | control |
      | \u0007  |
      | \u0008  |
      | \u001b  |
      | \u000c  |

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
  Scenario: A webhook list on a server without the endpoint reports it as unavailable and prints no secret
    Given the stub Outline server answers "webhookSubscriptions.list" with status 404 and the body:
      """
      {"ok": false, "status": 404, "error": "not_found", "message": "Endpoint not found", "secret": "synthetic-signing-secret"}
      """
    When I run "retriever --output json outline webhook-subscriptions list"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.outline.operation_unavailable" and retryable false
    And the standard error does not contain "synthetic-signing-secret"

  @integration @e2e @unit
  Scenario Outline: A read never prints a value that its reply holds and the rules forbid
    Given the stub Outline server answers "<method>" with the body:
      """
      {"data": [{"id": "b6000000-0000-4000-8000-000000000021", "name": "Synthetic item", "value": "<secret>", "clientSecret": "<secret>", "secret": "<secret>", "clientId": "synthetic-client-id", "clientType": "confidential", "redirectUris": [], "published": false, "scope": ["read"], "url": "https://hooks.example.test/synthetic/path", "enabled": true, "events": [], "createdById": "33333333-3333-4333-8333-333333333333", "createdAt": "2026-01-01T00:00:00Z", "updatedAt": "2026-01-02T00:00:00Z"}], "pagination": {"offset": 0, "limit": 25}}
      """
    When I run "<command>"
    Then the exit status is 0
    And the standard output does not contain "<secret>"

    Examples:
      | method                    | command                                     | secret                   |
      | apiKeys.list              | retriever outline api-keys list             | synthetic-key-value      |
      | oauthClients.list         | retriever outline oauth-clients list        | synthetic-client-secret  |
      | webhookSubscriptions.list | retriever outline webhook-subscriptions list | synthetic-signing-secret |

  @integration @e2e @unit
  Scenario: An integration list prints the host of each address and never its path or query
    Given the stub Outline server answers "integrations.list" with the body:
      """
      {"data": [{"id": "b6000000-0000-4000-8000-000000000031", "type": "webhook", "service": "slack", "events": ["documents.update"], "settings": {"url": "https://hooks.example.test/synthetic/secret-path?token=synthetic-query-token", "channel": "#synthetic"}, "createdAt": "2026-01-01T00:00:00Z", "updatedAt": "2026-01-02T00:00:00Z"}], "pagination": {"offset": 0, "limit": 25}}
      """
    When I run "retriever --output json outline integrations list"
    Then the exit status is 0
    And the standard output is a JSON document with schemaVersion 1
    And the standard output contains "hooks.example.test"
    And the standard output does not contain "/synthetic/secret-path"
    And the standard output does not contain "synthetic-query-token"

  @integration @e2e @unit
  Scenario: The JSON document of an API key list keeps the last four characters and never the key value
    Given the stub Outline server answers "apiKeys.list" with the body:
      """
      {"data": [{"id": "b6000000-0000-4000-8000-000000000041", "userId": "33333333-3333-4333-8333-333333333333", "name": "Synthetic key", "scope": ["read"], "last4": "wxyz", "value": "synthetic-key-value", "hash": "synthetic-key-hash", "createdAt": "2026-01-01T00:00:00Z", "updatedAt": "2026-01-02T00:00:00Z", "expiresAt": null}], "pagination": {"offset": 0, "limit": 25}}
      """
    When I run "retriever --output json outline api-keys list"
    Then the exit status is 0
    And the standard output is a JSON document with schemaVersion 1
    And the standard output contains "wxyz"
    And the standard output does not contain "synthetic-key-value"
    And the standard output does not contain "synthetic-key-hash"

  @integration @e2e @unit
  Scenario Outline: The OAuth authentications list never prints a client secret that its reply holds, in text or in JSON
    Given the stub Outline server answers "oauthAuthentications.list" with the body:
      """
      {"data": [{"id": "b6000000-0000-4000-8000-000000000051", "oauthClientId": "0c0c0c0c-0000-4000-8000-000000000001", "scope": ["read"], "createdAt": "2026-01-01T00:00:00Z", "lastActiveAt": null, "clientSecret": "synthetic-client-secret", "oauthClient": {"id": "0c0c0c0c-0000-4000-8000-000000000001", "name": "Synthetic client", "clientSecret": "synthetic-client-secret"}}], "pagination": {"offset": 0, "limit": 25}}
      """
    When I run "<command>"
    Then the exit status is 0
    And the standard output does not contain "synthetic-client-secret"

    Examples:
      | command                                                     |
      | retriever outline oauth-authentications list                |
      | retriever --output json outline oauth-authentications list  |

  @integration @e2e @unit
  Scenario Outline: A failing Outline response never prints the key
    Given the stub Outline server answers "collections.list" with status <status>
    When I run "retriever --output json outline collections list"
    Then the exit status is <exit>
    And the standard output is empty
    And the value of "EXAMPLE_OUTLINE_API_KEY" never appears in the standard output, the standard error, or the output folder

    Examples:
      | status | exit |
      | 403    | 2    |
      | 404    | 2    |
      | 429    | 124  |
      | 500    | 2    |

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

  @integration @e2e @unit
  Scenario: An Outline source that cannot be reached is reported as unreachable and never prints the key
    Given no stub Outline server is listening
    When I run "retriever --output json outline collections list"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.outline.unreachable" and retryable true
    And the value of "EXAMPLE_OUTLINE_API_KEY" never appears in the standard output, the standard error, or the output folder

  @integration @e2e @unit
  Scenario: Outline rate-limit exhaustion never prints the key
    Given the stub Outline server answers "collections.list" with status 429 and Retry-After "0"
    When I run "retriever --output json outline collections list"
    Then the exit status is 124
    And the standard error is a JSON error with the code "retriever.outline.rate_limited" and retryable true
    And the stub Outline server received 4 requests to "collections.list"
    And the value of "EXAMPLE_OUTLINE_API_KEY" never appears in the standard output, the standard error, or the output folder

  @integration @e2e @unit
  Scenario Outline: A rejected Outline key is reported by every catalog read and never printed
    Given the stub Outline server answers "<method>" with status 401
    When I run "<command>"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.outline.key_rejected" and retryable false
    And the value of "EXAMPLE_OUTLINE_API_KEY" never appears in the standard output, the standard error, or the output folder

    Examples:
      | method            | command                                                    |
      | collections.list  | retriever --output json outline collections list           |
      | documents.info    | retriever --output json outline documents get d0c1d0c1-0000-4000-8000-000000000001 |

  @integration @e2e @unit
  Scenario: A GraphQL error for a rejected Outline key never includes the key
    Given the stub Outline server answers "documents.info" with status 401
    And retriever serve is running on the loopback address
    When retriever serve receives this GraphQL request:
      """
      query { document(id: "d0c1d0c1-0000-4000-8000-000000000001") { id } }
      """
    Then the HTTP status is 200
    And the GraphQL response has an error
    And the GraphQL response does not contain "synthetic-outline-key"

  @integration @e2e @unit
  Scenario: A GraphQL reply to a failing Outline server never includes the key it echoed
    Given the stub Outline server echoes the Authorization header in a non-JSON reply to "documents.info"
    And retriever serve is running on the loopback address
    When retriever serve receives this GraphQL request:
      """
      query { document(id: "d0c1d0c1-0000-4000-8000-000000000001") { id } }
      """
    Then the HTTP status is 200
    And the GraphQL response has an error
    And the GraphQL response does not contain "synthetic-outline-key"

  @integration @e2e @unit
  Scenario: A hostile reply through an MCP tool leaves no terminal control byte in the tool result
    Given the stub Outline server answers "documents.info" with the body:
      """
      {"data": {"id": "d0c1d0c1-0000-4000-8000-000000000001", "title": "\u001b]0;hostile\u0007Red title", "text": "Before \u001b[2J after", "url": "/doc/d0c1d0c1-0000-4000-8000-000000000001", "collectionId": "1c1c1c1c-0000-4000-8000-000000000001", "revision": 1}}
      """
    And an MCP client starts "retriever mcp"
    When the client calls the tool "get_document" with these arguments:
      """
      {"document_id": "d0c1d0c1-0000-4000-8000-000000000001"}
      """
    Then the tool result is not an error
    And the tool result text contains no control characters except line breaks

  @integration @e2e @unit
  Scenario: A failing reply that echoes the key never puts the key in an MCP tool result
    Given the stub Outline server answers "documents.info" with status 500 and the body:
      """
      {"ok": false, "error": "upstream failure", "echo": "Bearer synthetic-outline-key"}
      """
    And an MCP client starts "retriever mcp"
    When the client calls the tool "get_document" with these arguments:
      """
      {"document_id": "d0c1d0c1-0000-4000-8000-000000000001"}
      """
    Then the tool result is an error
    And the tool result text does not contain "synthetic-outline-key"

  @integration @e2e @unit
  Scenario: The OAuth authentications MCP tool never returns a client secret that its reply holds
    Given the stub Outline server answers "oauthAuthentications.list" with the body:
      """
      {"data": [{"id": "b6000000-0000-4000-8000-000000000051", "oauthClientId": "0c0c0c0c-0000-4000-8000-000000000001", "scope": ["read"], "createdAt": "2026-01-01T00:00:00Z", "lastActiveAt": null, "clientSecret": "synthetic-client-secret", "oauthClient": {"id": "0c0c0c0c-0000-4000-8000-000000000001", "name": "Synthetic client", "clientSecret": "synthetic-client-secret"}}], "pagination": {"offset": 0, "limit": 25}}
      """
    And an MCP client starts "retriever mcp"
    When the client calls the tool "list_oauth_authentications" with no arguments
    Then the tool result is not an error
    And the tool result text contains "0c0c0c0c-0000-4000-8000-000000000001"
    And the tool result text does not contain "synthetic-client-secret"
