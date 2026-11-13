# Read commands of an Outline wiki source: documents, collections, search, and the signed-in account.
# Every read needs read access only. Synthetic documents and collections come from the stub Outline server.
Feature: Outline reads
  As a person who searches and reads an Outline wiki from the command line
  I want each read to return the document text, the lists, and the account, in text or in JSON
  So that I can use the wiki in scripts and in agent harnesses

  Background:
    Given the sources catalog holds these sources:
      | id              | kind    | access | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | read   | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    And the environment variable "EXAMPLE_OUTLINE_API_KEY" is set to "synthetic-outline-key"
    And the stub Outline server holds these collections:
      | id | name      |
      | 1c1c1c1c-0000-4000-8000-000000000001 | Synthetic |
    And the stub Outline server holds these documents:
      | id    | collection | parent | title   | markdown            |
      | d0c1d0c1-0000-4000-8000-000000000001 | 1c1c1c1c-0000-4000-8000-000000000001         |        | Setup   | Synthetic body one  |
      | d0c2d0c2-0000-4000-8000-000000000002 | 1c1c1c1c-0000-4000-8000-000000000001         | d0c1d0c1-0000-4000-8000-000000000001  | Child   | Synthetic body two  |
    And the stub Outline server holds the signed-in user "Synthetic Author" of team "Synthetic Team"

  @integration @e2e
  Scenario: Getting a document prints its Markdown body in text mode
    When I run "retriever outline documents get d0c1d0c1-0000-4000-8000-000000000001"
    Then the exit status is 0
    And the standard output contains "Synthetic body one"
    And the standard error is empty

  @integration @e2e
  Scenario: Getting a document in JSON mode holds its text in the markdown field
    When I run "retriever --output json outline documents get d0c1d0c1-0000-4000-8000-000000000001"
    Then the exit status is 0
    And the standard output is a JSON document with schemaVersion 1
    And the JSON field "id" of the standard output is "d0c1d0c1-0000-4000-8000-000000000001"
    And the JSON field "title" of the standard output is "Setup"
    And the JSON field "markdown" of the standard output is "Synthetic body one"
    And the JSON field "collectionId" of the standard output is "1c1c1c1c-0000-4000-8000-000000000001"

  @integration @e2e
  Scenario: Getting a document that the wiki does not hold is a not-found failure
    When I run "retriever --output json outline documents get d0c9d0c9-0000-4000-8000-000000000009"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.outline.document_not_found" and retryable false

  @integration @e2e
  Scenario: A document ID with characters outside the allowed set is refused before any request
    When I run "retriever --output json outline documents get bad/id"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.outline.invalid_document_id" and retryable false
    And the stub Outline server received no requests

  @integration @e2e
  Scenario: A document list prints the count and the documents under the documents key
    When I run "retriever --output json outline documents list"
    Then the exit status is 0
    And the JSON field "count" of the standard output is 2
    And the JSON field "documents.0.id" of the standard output is "d0c1d0c1-0000-4000-8000-000000000001"
    And the JSON field "documents.1.id" of the standard output is "d0c2d0c2-0000-4000-8000-000000000002"

  @integration @e2e
  Scenario: A full page names the next offset and reports that more documents follow
    When I run "retriever --output json outline documents list --limit 1"
    Then the exit status is 0
    And the JSON field "has_more" of the standard output is true
    And the JSON field "next_offset" of the standard output is 1

  @integration @e2e
  Scenario: A short page reports no more documents and names no next offset
    When I run "retriever --output json outline documents list --limit 5"
    Then the exit status is 0
    And the JSON field "has_more" of the standard output is false
    And the JSON field "next_offset" of the standard output is absent

  @integration @e2e
  Scenario: A search that matches prints each match with its ID and title
    When I run "retriever --output json outline documents search Synthetic"
    Then the exit status is 0
    And the JSON field "count" of the standard output is 2
    And the JSON field "documents.0.id" of the standard output is "d0c1d0c1-0000-4000-8000-000000000001"

  @integration @e2e
  Scenario: A search that matches nothing exits 1 and prints a count of zero
    When I run "retriever --output json outline documents search nothing-matches-this"
    Then the exit status is 1
    And the JSON field "count" of the standard output is 0
    And the standard error is empty

  @integration @e2e
  Scenario: A search with a page size outside 1 to 100 is a usage error and sends no request
    When I run "retriever --output json outline documents search Synthetic --limit 0"
    Then the exit status is 2
    And the standard output is empty
    And the stub Outline server received no requests

  @integration @e2e
  Scenario: The collection list prints each collection with its ID and name
    When I run "retriever --output json outline collections list"
    Then the exit status is 0
    And the JSON field "count" of the standard output is 1
    And the JSON field "collections.0.id" of the standard output is "1c1c1c1c-0000-4000-8000-000000000001"
    And the JSON field "collections.0.name" of the standard output is "Synthetic"

  @integration @e2e
  Scenario: The account read prints the user and team names
    When I run "retriever --output json outline auth whoami"
    Then the exit status is 0
    And the JSON field "user.name" of the standard output is "Synthetic Author"
    And the JSON field "team.name" of the standard output is "Synthetic Team"

  @integration @e2e
  Scenario: The account read never prints an e-mail address
    When I run "retriever outline auth whoami"
    Then the exit status is 0
    And the standard output does not contain "@"

  @integration @e2e
  Scenario Outline: Every Outline read prints a schemaVersion 1 JSON document in JSON mode
    When I run "retriever --output json outline <command>"
    Then the exit status is 0
    And the standard output is a JSON document with schemaVersion 1

    Examples:
      | command                  |
      | collections list         |
      | documents list           |
      | auth whoami              |
      | documents get d0c1d0c1-0000-4000-8000-000000000001      |

  @integration @e2e
  Scenario: A read with more than one Outline source needs the source flag
    Given the sources catalog holds these sources:
      | id              | kind    | access | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | read   | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
      | second-outline  | outline | read   | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    When I run "retriever --output json outline collections list"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.outline.ambiguous_source" and retryable false
    And the stub Outline server received no requests

  @integration @e2e
  Scenario: The source flag picks one of two Outline sources
    Given the sources catalog holds these sources:
      | id              | kind    | access | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | read   | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
      | second-outline  | outline | read   | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    When I run "retriever outline collections list --source second-outline"
    Then the exit status is 0
    And the standard output contains "Synthetic"
