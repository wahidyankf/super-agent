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

  # Behaviour: CLI-459
  Scenario: Getting a document prints its Markdown body in text mode
    When I run "retriever outline documents get d0c1d0c1-0000-4000-8000-000000000001"
    Then the exit status is 0
    And the standard output contains "Synthetic body one"
    And the standard error is empty
    And the stub Outline server received 1 request to "documents.info"
    And every request to the stub Outline server used the HTTP method "POST"
    And the stub Outline server's last request to "documents.info" body contains "d0c1d0c1-0000-4000-8000-000000000001"
    And the stub Outline server's last request to "documents.info" carried the API version "2"

  # Behaviour: CLI-459
  Scenario: Getting a document in JSON mode holds its text in the markdown field
    When I run "retriever --output json outline documents get d0c1d0c1-0000-4000-8000-000000000001"
    Then the exit status is 0
    And the standard output is a JSON document with schemaVersion 1
    And the JSON field "id" of the standard output is "d0c1d0c1-0000-4000-8000-000000000001"
    And the JSON field "title" of the standard output is "Setup"
    And the JSON field "markdown" of the standard output is "Synthetic body one"
    And the JSON field "collectionId" of the standard output is "1c1c1c1c-0000-4000-8000-000000000001"

  Scenario: Getting a document that the wiki does not hold is a not-found failure
    When I run "retriever --output json outline documents get d0c9d0c9-0000-4000-8000-000000000009"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.outline.document_not_found" and retryable false

  Scenario: A document list prints the count and the documents under the documents key
    When I run "retriever --output json outline documents list"
    Then the exit status is 0
    And the JSON field "count" of the standard output is 2
    And the JSON field "documents.0.id" of the standard output is "d0c1d0c1-0000-4000-8000-000000000001"
    And the JSON field "documents.1.id" of the standard output is "d0c2d0c2-0000-4000-8000-000000000002"

  # Behaviour: CLI-460
  Scenario: A full page names the next offset and reports that more documents follow
    When I run "retriever --output json outline documents list --limit 1"
    Then the exit status is 0
    And the JSON field "has_more" of the standard output is true
    And the JSON field "next_offset" of the standard output is 1

  # Behaviour: CLI-460
  Scenario: A short page reports no more documents and names no next offset
    When I run "retriever --output json outline documents list --limit 5"
    Then the exit status is 0
    And the JSON field "has_more" of the standard output is false
    And the JSON field "next_offset" of the standard output is absent

  # Behaviour: CLI-461
  Scenario: A search that matches prints each match with its ID and title
    When I run "retriever --output json outline documents search Synthetic"
    Then the exit status is 0
    And the JSON field "count" of the standard output is 2
    And the JSON field "documents.0.id" of the standard output is "d0c1d0c1-0000-4000-8000-000000000001"
    And the JSON field "documents.0.title" of the standard output is "Setup"

  # Behaviour: CLI-461
  Scenario: A search that matches nothing exits 1 and prints a count of zero
    When I run "retriever --output json outline documents search nothing-matches-this"
    Then the exit status is 1
    And the JSON field "count" of the standard output is 0
    And the standard error is empty

  # Behaviour: CLI-464
  Scenario: The collection list prints each collection with its ID and name
    When I run "retriever --output json outline collections list"
    Then the exit status is 0
    And the JSON field "count" of the standard output is 1
    And the JSON field "collections.0.id" of the standard output is "1c1c1c1c-0000-4000-8000-000000000001"
    And the JSON field "collections.0.name" of the standard output is "Synthetic"

  # Behaviour: CLI-471
  Scenario: A document list in text mode prints each document with its ID and title
    When I run "retriever outline documents list"
    Then the exit status is 0
    And the standard output contains "d0c1d0c1-0000-4000-8000-000000000001"
    And the standard output contains "Setup"
    And the standard output contains "Child"

  # Behaviour: CLI-471
  Scenario: A collection list in text mode prints each collection with its ID and name
    When I run "retriever outline collections list"
    Then the exit status is 0
    And the standard output contains "1c1c1c1c-0000-4000-8000-000000000001"
    And the standard output contains "Synthetic"

  # Behaviour: CLI-466
  Scenario: The account read prints the user and team names
    When I run "retriever --output json outline auth whoami"
    Then the exit status is 0
    And the JSON field "user.name" of the standard output is "Synthetic Author"
    And the JSON field "team.name" of the standard output is "Synthetic Team"

  # Behaviour: CLI-466
  Scenario Outline: An account read never prints an e-mail address
    When I run "retriever <command>"
    Then the exit status is 0
    And the standard output does not contain "@"

    Examples:
      | command                                    |
      | outline auth whoami                        |
      | --output json outline auth whoami          |

  # Behaviour: CLI-467
  Scenario Outline: Every Outline read prints a schemaVersion 1 JSON document in JSON mode
    When I run "retriever --output json outline <command>"
    Then the exit status is 0
    And the standard output is a JSON document with schemaVersion 1

    Examples:
      | command                                                 |
      | collections list                                        |
      | documents list                                          |
      | auth whoami                                             |
      | documents get d0c1d0c1-0000-4000-8000-000000000001      |
      | documents search-titles Setup                           |
      | documents search Synthetic                              |
      | collections info 1c1c1c1c-0000-4000-8000-000000000001   |
      | documents archived                                      |
      | documents drafts                                        |
      | documents deleted                                       |
      | documents viewed                                        |
      | documents users d0c1d0c1-0000-4000-8000-000000000001    |
      | documents memberships d0c1d0c1-0000-4000-8000-000000000001 |
      | documents group-memberships d0c1d0c1-0000-4000-8000-000000000001 |
      | documents insights d0c1d0c1-0000-4000-8000-000000000001 |
      | collections memberships 1c1c1c1c-0000-4000-8000-000000000001 |
      | collections group-memberships 1c1c1c1c-0000-4000-8000-000000000001 |
      | comments list                                           |
      | comments get 44444444-4444-4444-8444-444444444444       |
      | revisions get e0e0e0e0-0000-4000-8000-000000000001      |
      | revisions export e0e0e0e0-0000-4000-8000-000000000001   |
      | shares list                                             |
      | shares get 5a5a5a5a-0000-4000-8000-000000000001         |
      | users list                                              |
      | users get --user-id 5e52e52e-0000-4000-8000-000000000002 |
      | groups list                                             |
      | groups get 6a6a6a6a-0000-4000-8000-000000000001         |
      | groups members 6a6a6a6a-0000-4000-8000-000000000001     |
      | templates list                                          |
      | templates get 7e7e7e7e-0000-4000-8000-000000000001      |
      | pins list                                               |
      | pins get d0c1d0c1-0000-4000-8000-000000000001           |
      | stars list                                              |

  # Behaviour: CLI-475
  Scenario: A read with more than one Outline source needs the source flag
    Given the sources catalog holds these sources:
      | id              | kind    | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
      | second-outline  | outline | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    When I run "retriever --output json outline collections list"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.outline.ambiguous_source" and retryable false
    And the stub Outline server received no requests

  # Behaviour: CLI-476
  Scenario: The source flag picks one of two Outline sources
    Given the sources catalog holds these sources:
      | id              | kind    | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
      | second-outline  | outline | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    When I run "retriever outline collections list --source second-outline"
    Then the exit status is 0
    And the standard output contains "Synthetic"

  # Behaviour: CLI-472
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

  # Behaviour: CLI-477
  Scenario: A server that does not offer documents.info is an operation-unavailable failure
    Given the stub Outline server does not offer "documents.info"
    When I run "retriever --output json outline documents get d0c1d0c1-0000-4000-8000-000000000001"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.outline.operation_unavailable" and retryable false

  # Behaviour: CLI-478
  Scenario: A document whose reply carries only editor data is read through documents.export
    Given the stub Outline server holds documents with editor data only
    When I run "retriever outline documents get d0c1d0c1-0000-4000-8000-000000000001"
    Then the exit status is 0
    And the standard output contains "Synthetic body one"
    And the stub Outline server received 1 request to "documents.export"

  # Behaviour: CLI-479
  Scenario: A server that ignores the API version header still gives the Markdown of the document
    Given the stub Outline server ignores the X-API-Version header
    When I run "retriever outline documents get d0c1d0c1-0000-4000-8000-000000000001"
    Then the exit status is 0
    And the standard output contains "Synthetic body one"

  Scenario: A search sends the query, the offset, and the limit in one request
    When I run "retriever --output json outline documents search Synthetic --offset 1 --limit 1"
    Then the exit status is 0
    And the JSON field "count" of the standard output is 1
    And the JSON field "documents.0.id" of the standard output is "d0c2d0c2-0000-4000-8000-000000000002"
    And the stub Outline server received 1 request to "documents.search"
    And the stub Outline server's last request to "documents.search" has the field "query" equal to Synthetic
    And the stub Outline server's last request to "documents.search" has the field "offset" equal to 1
    And the stub Outline server's last request to "documents.search" has the field "limit" equal to 1

  Scenario: A search with a page that is full reports that more documents follow
    When I run "retriever --output json outline documents search Synthetic --limit 1"
    Then the exit status is 0
    And the JSON field "has_more" of the standard output is true
    And the JSON field "next_offset" of the standard output is 1

  # Behaviour: CLI-477
  Scenario: A search on a server that does not offer documents.search is an operation-unavailable failure
    Given the stub Outline server does not offer "documents.search"
    When I run "retriever --output json outline documents search Synthetic"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.outline.operation_unavailable" and retryable false

  # Behaviour: CLI-462
  Scenario: A title search prints each matching document with its ID and title
    When I run "retriever outline documents search-titles Setup"
    Then the exit status is 0
    And the standard output contains "d0c1d0c1-0000-4000-8000-000000000001"
    And the standard output contains "Setup"
    And the standard output does not contain "Child"

  Scenario: A title search in JSON mode lists the matches under the documents key and sends its query and paging fields in one request
    When I run "retriever --output json outline documents search-titles Setup --offset 0 --limit 5"
    Then the exit status is 0
    And the standard output is a JSON document with schemaVersion 1
    And the JSON field "count" of the standard output is 1
    And the JSON field "documents.0.id" of the standard output is "d0c1d0c1-0000-4000-8000-000000000001"
    And the JSON field "documents.0.title" of the standard output is "Setup"
    And the stub Outline server received 1 request to "documents.search_titles"
    And the stub Outline server's last request to "documents.search_titles" has the field "query" equal to Setup
    And the stub Outline server's last request to "documents.search_titles" has the field "limit" equal to 5
    And the stub Outline server's last request to "documents.search_titles" has the field "offset" equal to 0

  # Behaviour: CLI-462
  Scenario: A title search reads titles only, so a word that appears only in a body finds no document
    When I run "retriever --output json outline documents search-titles Synthetic"
    Then the exit status is 1
    And the JSON field "count" of the standard output is 0
    And the standard error is empty

  # Behaviour: CLI-463
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

  # Behaviour: CLI-463
  Scenario: A title search with a collection filter finds only the documents of that collection
    Given the stub Outline server holds these collections:
      | id                                   | name      |
      | 1c1c1c1c-0000-4000-8000-000000000001 | Synthetic |
      | 2c2c2c2c-0000-4000-8000-000000000002 | Other     |
    And the stub Outline server holds these documents:
      | id                                   | collection                           | parent | title           | markdown           |
      | d0c1d0c1-0000-4000-8000-000000000001 | 1c1c1c1c-0000-4000-8000-000000000001 |        | Setup           | Synthetic body one |
      | d0c4d0c4-0000-4000-8000-000000000004 | 2c2c2c2c-0000-4000-8000-000000000002 |        | Setup elsewhere | Synthetic body four |
    When I run "retriever --output json outline documents search-titles Setup --collection-id 1c1c1c1c-0000-4000-8000-000000000001"
    Then the exit status is 0
    And the JSON field "count" of the standard output is 1
    And the JSON field "documents.0.id" of the standard output is "d0c1d0c1-0000-4000-8000-000000000001"
    And the stub Outline server's last request to "documents.search_titles" has the field "collectionId" equal to 1c1c1c1c-0000-4000-8000-000000000001

  # Behaviour: CLI-463
  Scenario: A title search sorted by title in ascending order sends the sort and the direction
    When I run "retriever outline documents search-titles Setup --sort title --direction ASC"
    Then the exit status is 0
    And the stub Outline server's last request to "documents.search_titles" has the field "sort" equal to title
    And the stub Outline server's last request to "documents.search_titles" has the field "direction" equal to ASC

  # Behaviour: CLI-474
  Scenario Outline: A title search option value outside its allowed set is refused before any request
    When I run "retriever --output json outline documents search-titles Setup <option>"
    Then the exit status is 2
    And the standard output is empty
    And the stub Outline server received no requests

    Examples:
      | option               |
      | --collection-id bad  |
      | --sort updated       |
      | --direction sideways |

  # Behaviour: CLI-477
  Scenario: A title search on a server that does not offer documents.search_titles is an operation-unavailable failure
    Given the stub Outline server does not offer "documents.search_titles"
    When I run "retriever --output json outline documents search-titles Setup"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.outline.operation_unavailable" and retryable false

  # Behaviour: CLI-469
  Scenario: Listing the revisions of a document prints one row per revision and never the text of a revision
    Given the stub Outline server answers "revisions.list" with the body:
      """
      {"data": [{"id": "e0e0e0e0-0000-4000-8000-000000000001", "documentId": "d0c1d0c1-0000-4000-8000-000000000001", "title": "Setup", "createdAt": "2026-01-01T00:00:00.000Z", "text": "Synthetic revision text"}], "pagination": {"offset": 0, "limit": 25}}
      """
    When I run "retriever outline revisions list d0c1d0c1-0000-4000-8000-000000000001"
    Then the exit status is 0
    And the standard output contains "e0e0e0e0-0000-4000-8000-000000000001"
    And the standard output does not contain "Synthetic revision text"

  # Behaviour: CLI-469
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

  # Behaviour: CLI-477
  Scenario: A server that does not offer revisions.list is an operation-unavailable failure for the revision list
    Given the stub Outline server does not offer "revisions.list"
    When I run "retriever --output json outline revisions list d0c1d0c1-0000-4000-8000-000000000001"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.outline.operation_unavailable" and retryable false

  Scenario: A collection list fetches pages of 25 until a short page ends the listing
    Given the stub Outline server holds 30 numbered collections
    When I run "retriever --output json outline collections list"
    Then the exit status is 0
    And the JSON field "count" of the standard output is 30
    And the stub Outline server received 2 requests to "collections.list"

  # Behaviour: CLI-483
  Scenario: An empty wiki prints a count of zero collections
    Given the stub Outline server holds no collections
    When I run "retriever --output json outline collections list"
    Then the exit status is 0
    And the JSON field "count" of the standard output is 0

  # Behaviour: CLI-480
  Scenario: A server that keeps returning full pages is stopped at the page cap
    Given the stub Outline server returns full pages
    When I run "retriever --output json outline collections list"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.outline.page_cap_reached" and retryable false

  # Behaviour: CLI-482
  Scenario: An Outline 403 on a collection list is an access-denied failure that prints no key
    Given the stub Outline server answers "collections.list" with status 403
    When I run "retriever --output json outline collections list"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.outline.access_denied" and retryable false
    And the value of "EXAMPLE_OUTLINE_API_KEY" never appears in the standard output, the standard error, or the output folder

  # Behaviour: CLI-464
  Scenario: Reading one collection prints its ID and its name
    When I run "retriever outline collections info 1c1c1c1c-0000-4000-8000-000000000001"
    Then the exit status is 0
    And the standard output contains "1c1c1c1c-0000-4000-8000-000000000001"
    And the standard output contains "Synthetic"

  # Behaviour: CLI-481
  Scenario: Reading a collection that the wiki does not hold is a not-found failure
    When I run "retriever --output json outline collections info 1c9c1c9c-0000-4000-8000-000000000009"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.outline.collection_not_found" and retryable false

  # Behaviour: CLI-465
  Scenario: The document tree of one collection prints each document with its nested children
    When I run "retriever outline collections documents 1c1c1c1c-0000-4000-8000-000000000001"
    Then the exit status is 0
    And the standard output contains "Setup"
    And the standard output contains "Child"

  # Behaviour: CLI-465
  Scenario: The document tree in JSON mode names each node's parent and depth
    When I run "retriever --output json outline collections documents 1c1c1c1c-0000-4000-8000-000000000001"
    Then the exit status is 0
    And the standard output is a JSON document with schemaVersion 1
    And the JSON field "count" of the standard output is 2
    And the JSON field "nodes.0.title" of the standard output is "Setup"
    And the JSON field "nodes.0.depth" of the standard output is 0
    And the JSON field "nodes.1.parentId" of the standard output is "d0c1d0c1-0000-4000-8000-000000000001"
    And the JSON field "nodes.1.depth" of the standard output is 1

  # Behaviour: CLI-477
  Scenario: A server without auth.info is an operation-unavailable failure for the account read
    Given the stub Outline server does not offer "auth.info"
    When I run "retriever --output json outline auth whoami"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.outline.operation_unavailable" and retryable false

  Scenario: A document list filtered by collection lists only the documents of that collection
    Given the stub Outline server holds these collections:
      | id                                   | name      |
      | 1c1c1c1c-0000-4000-8000-000000000001 | Synthetic |
      | 2c2c2c2c-0000-4000-8000-000000000002 | Other     |
    And the stub Outline server holds these documents:
      | id                                   | collection                           | parent                               | title   | markdown            |
      | d0c1d0c1-0000-4000-8000-000000000001 | 1c1c1c1c-0000-4000-8000-000000000001 |                                      | Setup   | Synthetic body one  |
      | d0c2d0c2-0000-4000-8000-000000000002 | 1c1c1c1c-0000-4000-8000-000000000001 | d0c1d0c1-0000-4000-8000-000000000001 | Child   | Synthetic body two  |
      | d0c4d0c4-0000-4000-8000-000000000004 | 2c2c2c2c-0000-4000-8000-000000000002 |                                      | Other   | Synthetic body four |
    When I run "retriever --output json outline documents list --collection-id 1c1c1c1c-0000-4000-8000-000000000001"
    Then the exit status is 0
    And the JSON field "count" of the standard output is 2
    And the stub Outline server's last request to "documents.list" has the field "collectionId" equal to 1c1c1c1c-0000-4000-8000-000000000001

  Scenario: A document list with a collection filter that is not a UUID is refused before any request
    When I run "retriever --output json outline documents list --collection-id bad"
    Then the exit status is 2
    And the standard output is empty
    And the stub Outline server received no requests

  # Behaviour: CLI-473
  Scenario Outline: An Outline list or search with a page size outside 1 to 100 is refused before any request
    When I run "retriever --output json outline <command>"
    Then the exit status is 2
    And the standard output is empty
    And the stub Outline server received no requests

    Examples:
      | command                                  |
      | documents search Synthetic --limit 0     |
      | documents search-titles Setup --limit 0  |
      | documents search-titles Setup --limit 101 |
      | documents list --limit 101               |

  # Behaviour: CLI-470
  Scenario: The archived document list prints each document's title and never the text of a document
    When I run "retriever outline documents archived"
    Then the exit status is 0
    And the standard output contains "Old Plan"
    And the standard output does not contain "must be dropped"

  # Behaviour: CLI-468
  Scenario: A comment list prints the count and one row per comment
    When I run "retriever outline comments list"
    Then the exit status is 0
    And the standard output contains "2 comments"
    And the standard output contains "44444444-4444-4444-8444-444444444444"
    And the standard output contains "45454545-4545-4545-8545-454545454545"
