# The GraphQL server started by `retriever serve`. It listens on the loopback address only, answers read queries about
# the configured sources, caps every source to read access, and refuses every write. These scenarios start the built
# binary as a process, so they run at the end-to-end level only.
Feature: GraphQL read server
  As a dashboard or a script that reads the sources through one local API
  I want a read-only GraphQL endpoint on the loopback address
  So that other tools can read source content without holding any key or write access

  Background:
    Given the environment variable "EXAMPLE_OUTLINE_API_KEY" is set to "synthetic-outline-key"
    And the sources catalog holds these sources:
      | id              | kind    | access | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | admin  | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    And the stub Outline server holds these collections:
      | id | name      |
      | 1c1c1c1c-0000-4000-8000-000000000001 | Synthetic |
    And the stub Outline server holds these documents:
      | id    | collection | parent | title | markdown            |
      | d0c1d0c1-0000-4000-8000-000000000001 | 1c1c1c1c-0000-4000-8000-000000000001         |        | Setup | Synthetic body one  |
      | d0c2d0c2-0000-4000-8000-000000000002 | 1c1c1c1c-0000-4000-8000-000000000001         | d0c1d0c1-0000-4000-8000-000000000001  | Child | Synthetic body two  |
    And retriever serve is running on the loopback address

  @e2e
  Scenario: The server reports that it listens on the loopback address
    Then the server listens on the loopback address only

  @e2e
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

  @e2e
  Scenario: The served API caps every source to read access
    When retriever serve receives this GraphQL request:
      """
      query { sources { id access } }
      """
    Then the HTTP status is 200
    And the GraphQL response field "data.sources.0.access" is "read"

  @e2e
  Scenario: The collections query lists the collections of the default wiki source
    When retriever serve receives this GraphQL request:
      """
      query { collections { id name } }
      """
    Then the HTTP status is 200
    And the GraphQL response has no errors
    And the GraphQL response field "data.collections.0.id" is "1c1c1c1c-0000-4000-8000-000000000001"
    And the GraphQL response field "data.collections.0.name" is "Synthetic"

  @e2e
  Scenario: The collection tree is flat, with each node naming its parent and depth
    When retriever serve receives this GraphQL request:
      """
      query { collectionTree(id: "1c1c1c1c-0000-4000-8000-000000000001") { id parentId depth } }
      """
    Then the HTTP status is 200
    And the GraphQL response has no errors
    And the GraphQL response field "data.collectionTree.1.parentId" is "d0c1d0c1-0000-4000-8000-000000000001"
    And the GraphQL response field "data.collectionTree.1.depth" is 1

  @e2e
  Scenario: A document query returns its Markdown body and collection ID
    When retriever serve receives this GraphQL request:
      """
      query { document(id: "d0c1d0c1-0000-4000-8000-000000000001") { id title markdown collectionId } }
      """
    Then the HTTP status is 200
    And the GraphQL response has no errors
    And the GraphQL response field "data.document.markdown" is "Synthetic body one"
    And the GraphQL response field "data.document.collectionId" is "1c1c1c1c-0000-4000-8000-000000000001"

  @e2e
  Scenario: A document that the wiki does not hold returns a GraphQL error
    When retriever serve receives this GraphQL request:
      """
      query { document(id: "d0c9d0c9-0000-4000-8000-000000000009") { id } }
      """
    Then the HTTP status is 200
    And the GraphQL response has an error

  @e2e
  Scenario: A mutation is refused because the served API has no write operation
    When retriever serve receives this GraphQL request:
      """
      mutation { exportDocument(id: "d0c1d0c1-0000-4000-8000-000000000001") }
      """
    Then the GraphQL response has an error
    And the stub Outline server received no requests
