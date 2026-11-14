# Read commands of an Outline wiki source: documents, collections, search, and the signed-in account.
# Reads change nothing in the wiki. Synthetic documents and collections come from the stub Outline server.
Feature: Outline reads
  As a person who searches and reads an Outline wiki from the command line
  I want each read to return the document text, the lists, and the account, in text or in JSON
  So that I can use the wiki in scripts and in agent harnesses

  Background:
    Given the sources catalog holds these sources:
      | id              | kind    | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    And the environment variable "EXAMPLE_OUTLINE_API_KEY" is set to "synthetic-outline-key"
    And the stub Outline server holds these collections:
      | id | name      |
      | 1c1c1c1c-0000-4000-8000-000000000001 | Synthetic |
    And the stub Outline server holds these documents:
      | id    | collection | parent | title   | markdown            |
      | d0c1d0c1-0000-4000-8000-000000000001 | 1c1c1c1c-0000-4000-8000-000000000001         |        | Setup   | Synthetic body one  |
      | d0c2d0c2-0000-4000-8000-000000000002 | 1c1c1c1c-0000-4000-8000-000000000001         | d0c1d0c1-0000-4000-8000-000000000001  | Child   | Synthetic body two  |
    And the stub Outline server holds the signed-in user "Synthetic Author" of team "Synthetic Team"

  @integration @e2e @unit
  Scenario: Getting a document prints its Markdown body in text mode
    When I run "retriever outline documents get d0c1d0c1-0000-4000-8000-000000000001"
    Then the exit status is 0
    And the standard output contains "Synthetic body one"
    And the standard error is empty

  @integration @e2e @unit
  Scenario: Getting a document in JSON mode holds its text in the markdown field
    When I run "retriever --output json outline documents get d0c1d0c1-0000-4000-8000-000000000001"
    Then the exit status is 0
    And the standard output is a JSON document with schemaVersion 1
    And the JSON field "id" of the standard output is "d0c1d0c1-0000-4000-8000-000000000001"
    And the JSON field "title" of the standard output is "Setup"
    And the JSON field "markdown" of the standard output is "Synthetic body one"
    And the JSON field "collectionId" of the standard output is "1c1c1c1c-0000-4000-8000-000000000001"

  @integration @e2e @unit
  Scenario: Getting a document that the wiki does not hold is a not-found failure
    When I run "retriever --output json outline documents get d0c9d0c9-0000-4000-8000-000000000009"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.outline.document_not_found" and retryable false

  @integration @e2e @unit
  Scenario: A document ID with characters outside the allowed set is refused before any request
    When I run "retriever --output json outline documents get bad/id"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.outline.invalid_document_id" and retryable false
    And the stub Outline server received no requests

  @integration @e2e @unit
  Scenario: A document list prints the count and the documents under the documents key
    When I run "retriever --output json outline documents list"
    Then the exit status is 0
    And the JSON field "count" of the standard output is 2
    And the JSON field "documents.0.id" of the standard output is "d0c1d0c1-0000-4000-8000-000000000001"
    And the JSON field "documents.1.id" of the standard output is "d0c2d0c2-0000-4000-8000-000000000002"

  @integration @e2e @unit
  Scenario: A full page names the next offset and reports that more documents follow
    When I run "retriever --output json outline documents list --limit 1"
    Then the exit status is 0
    And the JSON field "has_more" of the standard output is true
    And the JSON field "next_offset" of the standard output is 1

  @integration @e2e @unit
  Scenario: A short page reports no more documents and names no next offset
    When I run "retriever --output json outline documents list --limit 5"
    Then the exit status is 0
    And the JSON field "has_more" of the standard output is false
    And the JSON field "next_offset" of the standard output is absent

  @integration @e2e @unit
  Scenario: A search that matches prints each match with its ID and title
    When I run "retriever --output json outline documents search Synthetic"
    Then the exit status is 0
    And the JSON field "count" of the standard output is 2
    And the JSON field "documents.0.id" of the standard output is "d0c1d0c1-0000-4000-8000-000000000001"

  @integration @e2e @unit
  Scenario: A search that matches nothing exits 1 and prints a count of zero
    When I run "retriever --output json outline documents search nothing-matches-this"
    Then the exit status is 1
    And the JSON field "count" of the standard output is 0
    And the standard error is empty

  @integration @e2e @unit
  Scenario: A search with a page size outside 1 to 100 is a usage error and sends no request
    When I run "retriever --output json outline documents search Synthetic --limit 0"
    Then the exit status is 2
    And the standard output is empty
    And the stub Outline server received no requests

  @integration @e2e @unit
  Scenario: The collection list prints each collection with its ID and name
    When I run "retriever --output json outline collections list"
    Then the exit status is 0
    And the JSON field "count" of the standard output is 1
    And the JSON field "collections.0.id" of the standard output is "1c1c1c1c-0000-4000-8000-000000000001"
    And the JSON field "collections.0.name" of the standard output is "Synthetic"

  @integration @e2e @unit
  Scenario: The account read prints the user and team names
    When I run "retriever --output json outline auth whoami"
    Then the exit status is 0
    And the JSON field "user.name" of the standard output is "Synthetic Author"
    And the JSON field "team.name" of the standard output is "Synthetic Team"

  @integration @e2e @unit
  Scenario: The account read never prints an e-mail address
    When I run "retriever outline auth whoami"
    Then the exit status is 0
    And the standard output does not contain "@"

  @integration @e2e @unit
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
      | documents search-titles Setup                           |
      | documents search Synthetic                              |
      | collections info 1c1c1c1c-0000-4000-8000-000000000001   |

  @integration @e2e @unit
  Scenario: A read with more than one Outline source needs the source flag
    Given the sources catalog holds these sources:
      | id              | kind    | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
      | second-outline  | outline | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    When I run "retriever --output json outline collections list"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.outline.ambiguous_source" and retryable false
    And the stub Outline server received no requests

  @integration @e2e @unit
  Scenario: The source flag picks one of two Outline sources
    Given the sources catalog holds these sources:
      | id              | kind    | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
      | second-outline  | outline | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    When I run "retriever outline collections list --source second-outline"
    Then the exit status is 0
    And the standard output contains "Synthetic"

  @integration @e2e @unit
  Scenario: A document read sends one documents.info request with the document ID and the pinned API version
    When I run "retriever --output json outline documents get d0c1d0c1-0000-4000-8000-000000000001"
    Then the exit status is 0
    And the stub Outline server received 1 request to "documents.info"
    And every request to the stub Outline server used the HTTP method "POST"
    And the stub Outline server's last request to "documents.info" body contains "d0c1d0c1-0000-4000-8000-000000000001"
    And the stub Outline server's last request to "documents.info" carried the API version "2"

  @integration @e2e @unit
  Scenario Outline: A document ID with a dot or a slash is refused before any request
    When I run "retriever --output json outline documents get <id>"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.outline.invalid_document_id" and retryable false
    And the stub Outline server received no requests

    Examples:
      | id        |
      | bad.id    |
      | bad/id    |
      | d0c1.d0c1 |

  @integration @e2e @unit
  Scenario: A rejected key on a document read is reported without printing the key
    Given the stub Outline server answers "documents.info" with status 401
    When I run "retriever --output json outline documents get d0c1d0c1-0000-4000-8000-000000000001"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.outline.key_rejected" and retryable false
    And the value of "EXAMPLE_OUTLINE_API_KEY" never appears in the standard output, the standard error, or the output folder

  @integration @e2e @unit
  Scenario: A server that does not offer documents.info is an operation-unavailable failure
    Given the stub Outline server does not offer "documents.info"
    When I run "retriever --output json outline documents get d0c1d0c1-0000-4000-8000-000000000001"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.outline.operation_unavailable" and retryable false

  @integration @e2e @unit
  Scenario: A document whose reply carries only editor data is read through documents.export
    Given the stub Outline server holds documents with editor data only
    When I run "retriever outline documents get d0c1d0c1-0000-4000-8000-000000000001"
    Then the exit status is 0
    And the standard output contains "Synthetic body one"
    And the stub Outline server received 1 request to "documents.export"

  @integration @e2e @unit
  Scenario: A server that ignores the API version header still gives the Markdown of the document
    Given the stub Outline server ignores the X-API-Version header
    When I run "retriever outline documents get d0c1d0c1-0000-4000-8000-000000000001"
    Then the exit status is 0
    And the standard output contains "Synthetic body one"

  @integration @e2e @unit
  Scenario: A search sends the query, the offset, and the limit in one request
    When I run "retriever --output json outline documents search Synthetic --offset 1 --limit 1"
    Then the exit status is 0
    And the JSON field "count" of the standard output is 1
    And the JSON field "documents.0.id" of the standard output is "d0c2d0c2-0000-4000-8000-000000000002"
    And the stub Outline server received 1 request to "documents.search"
    And the stub Outline server's last request to "documents.search" has the field "query" equal to Synthetic
    And the stub Outline server's last request to "documents.search" has the field "offset" equal to 1
    And the stub Outline server's last request to "documents.search" has the field "limit" equal to 1

  @integration @e2e @unit
  Scenario: A search with a page that is full reports that more documents follow
    When I run "retriever --output json outline documents search Synthetic --limit 1"
    Then the exit status is 0
    And the JSON field "has_more" of the standard output is true
    And the JSON field "next_offset" of the standard output is 1

  @integration @e2e @unit
  Scenario: A search on a server that does not offer documents.search is an operation-unavailable failure
    Given the stub Outline server does not offer "documents.search"
    When I run "retriever --output json outline documents search Synthetic"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.outline.operation_unavailable" and retryable false

  @integration @e2e @unit
  Scenario: A search with a rejected key exits 2 and never prints the key
    Given the stub Outline server answers "documents.search" with status 401
    When I run "retriever --output json outline documents search Synthetic"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.outline.key_rejected" and retryable false
    And the value of "EXAMPLE_OUTLINE_API_KEY" never appears in the standard output, the standard error, or the output folder

  @integration @e2e @unit
  Scenario: A title search prints each matching document with its ID and title
    When I run "retriever outline documents search-titles Setup"
    Then the exit status is 0
    And the standard output contains "d0c1d0c1-0000-4000-8000-000000000001"
    And the standard output contains "Setup"
    And the standard output does not contain "Child"

  @integration @e2e @unit
  Scenario: A title search in JSON mode lists the matches under the documents key
    When I run "retriever --output json outline documents search-titles Setup"
    Then the exit status is 0
    And the standard output is a JSON document with schemaVersion 1
    And the JSON field "count" of the standard output is 1
    And the JSON field "documents.0.id" of the standard output is "d0c1d0c1-0000-4000-8000-000000000001"
    And the JSON field "documents.0.title" of the standard output is "Setup"

  @integration @e2e @unit
  Scenario: A title search reads titles only, so a word that appears only in a body finds no document
    When I run "retriever --output json outline documents search-titles Synthetic"
    Then the exit status is 1
    And the JSON field "count" of the standard output is 0
    And the standard error is empty

  @integration @e2e @unit
  Scenario: A title search sends the query and the paging fields in one request
    When I run "retriever --output json outline documents search-titles Setup --offset 0 --limit 5"
    Then the exit status is 0
    And the stub Outline server received 1 request to "documents.search_titles"
    And the stub Outline server's last request to "documents.search_titles" has the field "query" equal to Setup
    And the stub Outline server's last request to "documents.search_titles" has the field "limit" equal to 5
    And the stub Outline server's last request to "documents.search_titles" has the field "offset" equal to 0

  @integration @e2e @unit
  Scenario: A full title-search page names the next offset and reports that more documents follow
    Given the stub Outline server holds these documents:
      | id                                   | collection                           | parent                               | title       | markdown           |
      | d0c1d0c1-0000-4000-8000-000000000001 | 1c1c1c1c-0000-4000-8000-000000000001 |                                      | Setup       | Synthetic body one |
      | d0c2d0c2-0000-4000-8000-000000000002 | 1c1c1c1c-0000-4000-8000-000000000001 | d0c1d0c1-0000-4000-8000-000000000001 | Child       | Synthetic body two |
      | d0c3d0c3-0000-4000-8000-000000000003 | 1c1c1c1c-0000-4000-8000-000000000001 |                                      | Setup again | Synthetic body six |
    When I run "retriever --output json outline documents search-titles Setup --limit 1"
    Then the exit status is 0
    And the JSON field "count" of the standard output is 1
    And the JSON field "has_more" of the standard output is true
    And the JSON field "next_offset" of the standard output is 1

  @integration @e2e @unit
  Scenario: A title search with a collection filter sends the collection ID and finds only that collection
    When I run "retriever --output json outline documents search-titles Setup --collection-id 1c1c1c1c-0000-4000-8000-000000000001"
    Then the exit status is 0
    And the JSON field "count" of the standard output is 1
    And the stub Outline server's last request to "documents.search_titles" has the field "collectionId" equal to 1c1c1c1c-0000-4000-8000-000000000001

  @integration @e2e @unit
  Scenario: A title search sorted by title in ascending order sends the sort and the direction
    When I run "retriever outline documents search-titles Setup --sort title --direction ASC"
    Then the exit status is 0
    And the stub Outline server's last request to "documents.search_titles" has the field "sort" equal to title
    And the stub Outline server's last request to "documents.search_titles" has the field "direction" equal to ASC

  @integration @e2e @unit
  Scenario Outline: A title search with an option value outside its allowed set is refused before any request
    When I run "retriever --output json outline documents search-titles Setup <option>"
    Then the exit status is 2
    And the standard output is empty
    And the stub Outline server received no requests

    Examples:
      | option               |
      | --limit 0            |
      | --limit 101          |
      | --collection-id bad  |
      | --sort updated       |
      | --direction sideways |

  @integration @e2e @unit
  Scenario: A title search on a server that does not offer documents.search_titles is an operation-unavailable failure
    Given the stub Outline server does not offer "documents.search_titles"
    When I run "retriever --output json outline documents search-titles Setup"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.outline.operation_unavailable" and retryable false

  @integration @e2e @unit
  Scenario: Listing the revisions of a document prints one row per revision and never the text of a revision
    Given the stub Outline server answers "revisions.list" with the body:
      """
      {"data": [{"id": "e0e0e0e0-0000-4000-8000-000000000001", "documentId": "d0c1d0c1-0000-4000-8000-000000000001", "title": "Setup", "createdAt": "2026-01-01T00:00:00.000Z", "text": "Synthetic revision text"}], "pagination": {"offset": 0, "limit": 25}}
      """
    When I run "retriever outline revisions list d0c1d0c1-0000-4000-8000-000000000001"
    Then the exit status is 0
    And the standard output contains "e0e0e0e0-0000-4000-8000-000000000001"
    And the standard output does not contain "Synthetic revision text"

  @integration @e2e @unit
  Scenario: A revision list in JSON mode holds the count and the revisions, and no revision has a text field
    Given the stub Outline server answers "revisions.list" with the body:
      """
      {"data": [{"id": "e0e0e0e0-0000-4000-8000-000000000001", "documentId": "d0c1d0c1-0000-4000-8000-000000000001", "title": "Setup", "createdAt": "2026-01-01T00:00:00.000Z", "text": "Synthetic revision text"}], "pagination": {"offset": 0, "limit": 25}}
      """
    When I run "retriever --output json outline revisions list d0c1d0c1-0000-4000-8000-000000000001"
    Then the exit status is 0
    And the standard output is a JSON document with schemaVersion 1
    And the JSON field "count" of the standard output is 1
    And the JSON field "revisions.0.id" of the standard output is "e0e0e0e0-0000-4000-8000-000000000001"
    And the JSON field "revisions.0.text" of the standard output is absent
    And the stub Outline server's last request to "revisions.list" has the field "documentId" equal to d0c1d0c1-0000-4000-8000-000000000001

  @integration @e2e @unit
  Scenario: A server that does not offer revisions.list is an operation-unavailable failure for the revision list
    Given the stub Outline server does not offer "revisions.list"
    When I run "retriever --output json outline revisions list d0c1d0c1-0000-4000-8000-000000000001"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.outline.operation_unavailable" and retryable false

  @integration @e2e @unit
  Scenario: A collection list fetches pages of 25 until a short page ends the listing
    Given the stub Outline server holds 30 numbered collections
    When I run "retriever --output json outline collections list"
    Then the exit status is 0
    And the JSON field "count" of the standard output is 30
    And the stub Outline server received 2 requests to "collections.list"

  @integration @e2e @unit
  Scenario: An empty wiki prints a count of zero collections
    Given the stub Outline server holds no collections
    When I run "retriever --output json outline collections list"
    Then the exit status is 0
    And the JSON field "count" of the standard output is 0

  @integration @e2e @unit
  Scenario: A server that keeps returning full pages is stopped at the page cap
    Given the stub Outline server returns full pages
    When I run "retriever --output json outline collections list"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.outline.page_cap_reached" and retryable false

  @integration @e2e @unit
  Scenario: An Outline 401 on a collection list is a rejected key that prints no key
    Given the stub Outline server answers "collections.list" with status 401
    When I run "retriever --output json outline collections list"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.outline.key_rejected" and retryable false
    And the value of "EXAMPLE_OUTLINE_API_KEY" never appears in the standard output, the standard error, or the output folder

  @integration @e2e @unit
  Scenario: An Outline 403 on a collection list is an access-denied failure that prints no key
    Given the stub Outline server answers "collections.list" with status 403
    When I run "retriever --output json outline collections list"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.outline.access_denied" and retryable false
    And the value of "EXAMPLE_OUTLINE_API_KEY" never appears in the standard output, the standard error, or the output folder

  @integration @e2e @unit
  Scenario: Reading one collection prints its ID and its name
    When I run "retriever outline collections info 1c1c1c1c-0000-4000-8000-000000000001"
    Then the exit status is 0
    And the standard output contains "1c1c1c1c-0000-4000-8000-000000000001"
    And the standard output contains "Synthetic"

  @integration @e2e @unit
  Scenario: Reading a collection that the wiki does not hold is a not-found failure
    When I run "retriever --output json outline collections info 1c9c1c9c-0000-4000-8000-000000000009"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.outline.collection_not_found" and retryable false

  @integration @e2e @unit
  Scenario: The document tree of one collection prints each document with its nested children
    When I run "retriever outline collections documents 1c1c1c1c-0000-4000-8000-000000000001"
    Then the exit status is 0
    And the standard output contains "Setup"
    And the standard output contains "Child"

  @integration @e2e @unit
  Scenario: The document tree in JSON mode names each node's parent and depth
    When I run "retriever --output json outline collections documents 1c1c1c1c-0000-4000-8000-000000000001"
    Then the exit status is 0
    And the standard output is a JSON document with schemaVersion 1
    And the JSON field "count" of the standard output is 2
    And the JSON field "nodes.0.title" of the standard output is "Setup"
    And the JSON field "nodes.0.depth" of the standard output is 0
    And the JSON field "nodes.1.parentId" of the standard output is "d0c1d0c1-0000-4000-8000-000000000001"
    And the JSON field "nodes.1.depth" of the standard output is 1

  @integration @e2e @unit
  Scenario: The account read in JSON mode has no e-mail address field
    When I run "retriever --output json outline auth whoami"
    Then the exit status is 0
    And the standard output is a JSON document with schemaVersion 1
    And the JSON field "user.email" of the standard output is absent
    And the JSON field "user.name" of the standard output is "Synthetic Author"

  @integration @e2e @unit
  Scenario: A server without auth.info is an operation-unavailable failure for the account read
    Given the stub Outline server does not offer "auth.info"
    When I run "retriever --output json outline auth whoami"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.outline.operation_unavailable" and retryable false

  @integration @e2e @unit
  Scenario: A rejected key on the account read is reported without printing the key
    Given the stub Outline server answers "auth.info" with status 401
    When I run "retriever --output json outline auth whoami"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.outline.key_rejected" and retryable false
    And the value of "EXAMPLE_OUTLINE_API_KEY" never appears in the standard output, the standard error, or the output folder

  @integration @e2e @unit
  Scenario: A document list filtered by collection sends the collection ID and lists only that collection
    When I run "retriever --output json outline documents list --collection-id 1c1c1c1c-0000-4000-8000-000000000001"
    Then the exit status is 0
    And the JSON field "count" of the standard output is 2
    And the stub Outline server's last request to "documents.list" has the field "collectionId" equal to 1c1c1c1c-0000-4000-8000-000000000001

  @integration @e2e @unit
  Scenario: A document list with a collection filter that is not a UUID is refused before any request
    When I run "retriever --output json outline documents list --collection-id bad"
    Then the exit status is 2
    And the standard output is empty
    And the stub Outline server received no requests

  @integration @e2e @unit
  Scenario: A document list with a page size outside 1 to 100 is refused before any request
    When I run "retriever --output json outline documents list --limit 101"
    Then the exit status is 2
    And the standard output is empty
    And the stub Outline server received no requests
