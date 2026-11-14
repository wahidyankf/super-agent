# The GraphQL server started by `retriever serve`. It listens on the loopback address only, answers read queries about
# the configured sources, and refuses every write. The guards run in this order: the Host allowlist, the Origin refusal,
# the POST-only rule, the JSON content type, the 1 MiB body cap, then the depth, complexity, and introspection limits. A
# refused request never reaches a source. The unit and integration levels run the server in process; the end-to-end level
# runs the built binary.
Feature: GraphQL read server
  As a dashboard or a script that reads the sources through one local API
  I want a read-only GraphQL endpoint on the loopback address
  So that other tools can read source content without holding any key

  Background:
    Given the environment variable "EXAMPLE_OUTLINE_API_KEY" is set to "synthetic-outline-key"
    And the sources catalog holds these sources:
      | id              | kind    | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    And the stub Outline server holds these collections:
      | id | name      |
      | 1c1c1c1c-0000-4000-8000-000000000001 | Synthetic |
    And the stub Outline server holds these documents:
      | id    | collection | parent | title | markdown            |
      | d0c1d0c1-0000-4000-8000-000000000001 | 1c1c1c1c-0000-4000-8000-000000000001         |        | Setup | Synthetic body one  |
      | d0c2d0c2-0000-4000-8000-000000000002 | 1c1c1c1c-0000-4000-8000-000000000001         | d0c1d0c1-0000-4000-8000-000000000001  | Child | Synthetic body two  |
    And retriever serve is running on the loopback address

  @unit @integration @e2e
  Scenario: The server reports that it listens on the loopback address only
    Then the server listens on the loopback address only

  @unit @integration @e2e
  Scenario: The sources query lists each configured source without its endpoint or credential name
    When retriever serve receives this GraphQL request:
      """
      query { sources { id kind tags } }
      """
    Then the HTTP status is 200
    And the GraphQL response has no errors
    And the GraphQL response field "data.sources.0.id" is "example-outline"
    And the GraphQL response does not contain "docs.example.test"
    And the GraphQL response does not contain "EXAMPLE_OUTLINE_API_KEY"

  @unit @integration @e2e
  Scenario: The served API lists each source by its id
    When retriever serve receives this GraphQL request:
      """
      query { sources { id } }
      """
    Then the HTTP status is 200
    And the GraphQL response field "data.sources.0.id" is "example-outline"

  @unit @integration @e2e
  Scenario: The collections query lists the collections of the default wiki source
    When retriever serve receives this GraphQL request:
      """
      query { collections { id name } }
      """
    Then the HTTP status is 200
    And the GraphQL response has no errors
    And the GraphQL response field "data.collections.0.id" is "1c1c1c1c-0000-4000-8000-000000000001"
    And the GraphQL response field "data.collections.0.name" is "Synthetic"

  @unit @integration @e2e
  Scenario: The collection tree is flat, with each node naming its parent and depth
    When retriever serve receives this GraphQL request:
      """
      query { collectionTree(id: "1c1c1c1c-0000-4000-8000-000000000001") { id parentId depth } }
      """
    Then the HTTP status is 200
    And the GraphQL response has no errors
    And the GraphQL response field "data.collectionTree.1.parentId" is "d0c1d0c1-0000-4000-8000-000000000001"
    And the GraphQL response field "data.collectionTree.1.depth" is 1

  @unit @integration @e2e
  Scenario: The collection tree has no nested children field
    When retriever serve receives this GraphQL request:
      """
      query { collectionTree(id: "1c1c1c1c-0000-4000-8000-000000000001") { id children { id } } }
      """
    Then the GraphQL response has an error
    And the stub Outline server received no requests

  @unit @integration @e2e
  Scenario: A document query returns its Markdown body and collection ID
    When retriever serve receives this GraphQL request:
      """
      query { document(id: "d0c1d0c1-0000-4000-8000-000000000001") { id title markdown collectionId } }
      """
    Then the HTTP status is 200
    And the GraphQL response has no errors
    And the GraphQL response field "data.document.markdown" is "Synthetic body one"
    And the GraphQL response field "data.document.collectionId" is "1c1c1c1c-0000-4000-8000-000000000001"

  @unit @integration @e2e
  Scenario: A document that the wiki does not hold returns a GraphQL error
    When retriever serve receives this GraphQL request:
      """
      query { document(id: "d0c9d0c9-0000-4000-8000-000000000009") { id } }
      """
    Then the HTTP status is 200
    And the GraphQL response has an error

  @unit @integration @e2e
  Scenario: A mutation is refused because the served API has no write operation
    When retriever serve receives this GraphQL request:
      """
      mutation { exportDocument(id: "d0c1d0c1-0000-4000-8000-000000000001") }
      """
    Then the GraphQL response has an error
    And the stub Outline server received no requests

  @unit @integration @e2e
  Scenario: A Host header that names another host is refused before any source is called
    When retriever serve receives this GraphQL request with the Host header "docs.example.test":
      """
      query { sources { id } }
      """
    Then the HTTP status is 403
    And the stub Outline server received no requests

  @unit @integration @e2e
  Scenario: A loopback Host header with the served port is accepted
    When retriever serve receives this GraphQL request with the Host header "localhost:{port}":
      """
      query { sources { id } }
      """
    Then the HTTP status is 200
    And the GraphQL response has no errors

  @unit @integration @e2e
  Scenario: A loopback Host header without the served port is refused
    When retriever serve receives this GraphQL request with the Host header "localhost":
      """
      query { sources { id } }
      """
    Then the HTTP status is 403
    And the stub Outline server received no requests

  @unit @integration @e2e
  Scenario: A GET request is refused with 405, and POST is the only allowed method
    When retriever serve receives this GraphQL request with the HTTP method "GET":
      """
      query { sources { id } }
      """
    Then the HTTP status is 405
    And the HTTP response has the header "Allow" with the value "POST"
    And the stub Outline server received no requests

  @unit @integration @e2e
  Scenario: An OPTIONS preflight is refused, so the server gives no CORS answer
    When retriever serve receives this GraphQL request with the HTTP method "OPTIONS":
      """
      query { sources { id } }
      """
    Then the HTTP status is 405
    And the HTTP response has no header whose name starts with "Access-Control-"

  @unit @integration @e2e
  Scenario: A request with a plain-text content type is refused with 415
    When retriever serve receives this GraphQL request with the Content-Type header "text/plain":
      """
      query { sources { id } }
      """
    Then the HTTP status is 415
    And the stub Outline server received no requests

  @unit @integration @e2e
  Scenario: A JSON content type with a charset parameter is accepted
    When retriever serve receives this GraphQL request with the Content-Type header "application/json; charset=utf-8":
      """
      query { sources { id } }
      """
    Then the HTTP status is 200
    And the GraphQL response has no errors

  @unit @integration @e2e
  Scenario Outline: A request that carries an Origin header is refused before any source is called
    When retriever serve receives this GraphQL request with the Origin header "<origin>":
      """
      query { sources { id } }
      """
    Then the HTTP status is 403
    And the stub Outline server received no requests
    And the HTTP response has no header whose name starts with "Access-Control-"

    Examples:
      | origin                    |
      | https://app.example.test  |
      | null                      |

  @unit @integration @e2e
  Scenario: A normal answer carries no CORS header
    When retriever serve receives this GraphQL request:
      """
      query { sources { id } }
      """
    Then the HTTP status is 200
    And the HTTP response has no header whose name starts with "Access-Control-"

  @unit @integration @e2e
  Scenario: A body larger than 1 MiB is refused with 413 before any source is called
    When retriever serve receives a GraphQL request of 2 MiB
    Then the HTTP status is 413
    And the stub Outline server received no requests

  @unit @integration @e2e
  Scenario: A query nested six selection sets deep is refused by the depth limit
    When retriever serve receives this GraphQL request:
      """
      query { a { b { c { d { e { f } } } } } }
      """
    Then the HTTP status is 422
    And the GraphQL response contains "depth limit of 5 levels"
    And the stub Outline server received no requests

  @unit @integration @e2e
  Scenario: A query nested five selection sets deep passes the depth limit and fails schema validation
    When retriever serve receives this GraphQL request:
      """
      query { a { b { c { d { e } } } } }
      """
    Then the HTTP status is 422
    And the GraphQL response has an error
    And the GraphQL response does not contain "depth limit"

  @unit @integration @e2e
  Scenario: Three calls to the source in one query are refused by the complexity limit
    When retriever serve receives this GraphQL request:
      """
      query { a: collections { id } b: collections { id } c: collections { id } }
      """
    Then the HTTP status is 422
    And the GraphQL response contains "complexity limit of 100 points"
    And the stub Outline server received no requests

  @unit @integration @e2e
  Scenario: Two calls to the source in one query are within the complexity limit
    When retriever serve receives this GraphQL request:
      """
      query { a: collections { id } b: collections { id } }
      """
    Then the HTTP status is 200
    And the GraphQL response has no errors

  @unit @integration @e2e
  Scenario: A query that reads the schema is refused by the limits with 422
    When retriever serve receives this GraphQL request:
      """
      query { __schema { queryType { name } } }
      """
    Then the HTTP status is 422
    And the GraphQL response contains "introspection is disabled"
    And the stub Outline server received no requests

  @unit @integration @e2e
  Scenario: An interrupt stops the server with status 130
    When the interrupt reaches the running server
    Then the serve command exits with status 130

  @unit @integration @e2e
  Scenario: A termination signal stops the server with status 143
    When the termination signal reaches the running server
    Then the serve command exits with status 143
