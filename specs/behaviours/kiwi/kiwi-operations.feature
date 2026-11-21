# Kiwi TCMS source: the product, plan, and case reads, and the case writes. Every Kiwi request is one JSON-RPC POST to
# the source's endpoint, and a Kiwi failure maps to a retriever code of the kiwi area.
Feature: Kiwi test management operations
  As a person who keeps test plans and cases in Kiwi
  I want to read products, plans, and cases, and to create or change a case, from the command line
  So that test content reaches my notes and my agent harnesses without a browser

  Background:
    Given the sources catalog holds these sources:
      | id           | kind | tags  | endpoint                  | credential_env       | identity_env |
      | example-kiwi | kiwi | tests | https://kiwi.example.test | EXAMPLE_KIWI_API_KEY |              |
    And the environment variable "EXAMPLE_KIWI_API_KEY" is set to "synthetic-kiwi-key"
    And the stub Kiwi server holds this dataset:
      """
      {
        "Products":   [{"ID": 3, "Name": "Synthetic product"}],
        "Plans":      [{"ID": 12, "Name": "Synthetic plan", "Product": 3, "Parent": 0, "Text": "Plan text"}],
        "Cases":      [{"ID": 40, "Summary": "Synthetic case", "Text": "Case text", "Plans": [12], "Sortkey": 10, "Status": 1, "Priority": 1, "Category": 1}],
        "Categories": [{"ID": 1, "Name": "Functional", "Product": 3}],
        "Priorities": [{"ID": 1, "Value": "High"}],
        "Statuses":   [{"ID": 1, "Name": "PROPOSED"}]
      }
      """

  # Behaviour: CLI-434
  Scenario: The product list prints each product with its ID and name
    When I run "retriever --output json kiwi products list"
    Then the exit status is 0
    And the JSON field "products.0.id" of the standard output is "product-3"
    And the JSON field "products.0.name" of the standard output is "Synthetic product"

  # Behaviour: CLI-436
  Scenario: The plan document prints its Markdown text
    When I run "retriever kiwi documents get plan-12"
    Then the exit status is 0
    And the standard output contains "Plan text"

  # Behaviour: CLI-436
  Scenario: The case document prints its Markdown text in JSON mode with its ID
    When I run "retriever --output json kiwi documents get case-40"
    Then the exit status is 0
    And the standard output is a JSON document with schemaVersion 1
    And the JSON field "id" of the standard output is "case-40"

  Scenario Outline: A Kiwi document ID that Kiwi does not hold is a not-found failure
    When I run "retriever --output json kiwi <command>"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.kiwi.document_not_found" and retryable false

    Examples:
      | command             |
      | documents get case-999 |
      | documents get plan-999 |

  # Behaviour: CLI-438
  Scenario: Creating a test case sends one request and prints nothing to standard error
    When I run "retriever kiwi test-cases create Synthetic-case --product product-3 --category Functional --priority High --source example-kiwi"
    Then the exit status is 0
    And the standard error is empty
    And the stub Kiwi server received 1 request to "TestCase.create"

  Scenario: Updating a case sends one request for that case
    When I run "retriever kiwi test-cases update case-40 --summary Renamed-case --source example-kiwi"
    Then the exit status is 0
    And the stub Kiwi server received 1 request to "TestCase.update"

  # Behaviour: CLI-442
  Scenario: A JSON-RPC error with an unknown code on a read is an upstream failure sent once, and never shows its message
    Given the stub Kiwi server answers "TestCase.filter" with a JSON-RPC error with code -32000 and message "synthetic upstream detail"
    When I run "retriever --output json kiwi documents get case-40"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.kiwi.upstream_failure" and retryable false
    And the standard error does not contain "synthetic upstream detail"
    And the stub Kiwi server received 1 request to "TestCase.filter"

  # Behaviour: CLI-443
  Scenario: A JSON-RPC server error that says a case does not exist is the not-found failure of that case
    Given the stub Kiwi server answers "TestCase.filter" with a JSON-RPC error with code -32603 and message "Synthetic case does not exist"
    When I run "retriever --output json kiwi documents get case-40"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.kiwi.document_not_found" and retryable false
    And the standard error does not contain "Synthetic case does not exist"
    And the stub Kiwi server received 1 request to "TestCase.filter"

  # Behaviour: CLI-444
  Scenario: A JSON-RPC server error that says permission is denied on a read is the access-denied failure
    Given the stub Kiwi server answers "TestCase.filter" with a JSON-RPC error with code -32603 and message "permission denied for synthetic case"
    When I run "retriever --output json kiwi documents get case-40"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.kiwi.access_denied" and retryable false
    And the standard error does not contain "permission denied for synthetic case"
    And the stub Kiwi server received 1 request to "TestCase.filter"

  # Behaviour: CLI-446
  # Behaviour: CLI-444
  Scenario Outline: A JSON-RPC error on a write maps to its public error, never shows its message, and is sent once
    Given the stub Kiwi server answers "TestCase.create" with a JSON-RPC error with code <code> and message "<message>"
    When I run "retriever --output json kiwi test-cases create Synthetic-case --product product-3 --category Functional --priority High --source example-kiwi"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "<public>" and retryable false
    And the standard error does not contain "<message>"
    And the stub Kiwi server received 1 request to "TestCase.create"

    Examples:
      | code   | message            | public                               |
      | -32603 | permission denied  | retriever.kiwi.access_denied         |
      | -32603 | synthetic upstream | retriever.kiwi.write_outcome_unknown |

  # Behaviour: CLI-445
  Scenario Outline: A request error code on a write is a bad request whatever its message says
    Given the stub Kiwi server answers "TestCase.create" with a JSON-RPC error with code <code> and message "<message>"
    When I run "retriever --output json kiwi test-cases create Synthetic-case --product product-3 --category Functional --priority High --source example-kiwi"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.kiwi.bad_request" and retryable false
    And the stub Kiwi server received 1 request to "TestCase.create"

    Examples:
      | code   | message            |
      | -32602 | synthetic upstream |
      | -32600 | permission denied  |

  # Behaviour: CLI-447
  Scenario Outline: A non-success HTTP status that is not a JSON-RPC reply maps to its public error
    Given the stub Kiwi server answers "TestCase.filter" with status <status>
    When I run "retriever --output json kiwi documents get case-40"
    Then the exit status is 2
    And the standard error is a JSON error with the code "<code>" and retryable false
    And the stub Kiwi server received 1 request to "TestCase.filter"

    Examples:
      | status | code                            |
      | 401    | retriever.kiwi.key_rejected     |
      | 403    | retriever.kiwi.access_denied    |
      | 500    | retriever.kiwi.upstream_failure |

  # Behaviour: CLI-448
  Scenario: A Kiwi reply whose id is not the id of the request is an unexpected response
    Given the stub Kiwi server answers "TestCase.filter" with a reply that has another request id
    When I run "retriever --output json kiwi documents get case-40"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.kiwi.unexpected_response" and retryable false
    And the stub Kiwi server received 1 request to "TestCase.filter"

  # Behaviour: CLI-448
  Scenario: A Kiwi reply with neither a result nor an error is an unexpected response
    Given the stub Kiwi server answers "TestCase.filter" with a reply that has neither a result nor an error
    When I run "retriever --output json kiwi documents get case-40"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.kiwi.unexpected_response" and retryable false

  # Behaviour: CLI-449
  Scenario: A Kiwi reply that is not JSON is an unexpected response that names no host
    Given the stub Kiwi server answers "TestCase.filter" with a body that is not JSON
    When I run "retriever --output json kiwi documents get case-40"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.kiwi.unexpected_response" and retryable false
    And the standard error does not contain "kiwi.example.test"

  # Behaviour: CLI-450
  Scenario: A Kiwi read reply over the 10 MiB cap is refused as too large
    Given the stub Kiwi server answers "TestCase.filter" with an oversize body
    When I run "retriever --output json kiwi documents get case-40"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.kiwi.reply_too_large" and retryable false

  # Behaviour: CLI-446
  Scenario: A write that Kiwi answers with a server error is sent once and its outcome is unknown
    Given the stub Kiwi server answers "TestCase.create" with status 500
    When I run "retriever --output json kiwi test-cases create Synthetic-case --product product-3 --category Functional --priority High --source example-kiwi"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.kiwi.write_outcome_unknown" and retryable false
    And the stub Kiwi server received 1 request to "TestCase.create"

  # Behaviour: CLI-434
  Scenario: The product list in JSON mode reports a count of the products
    When I run "retriever --output json kiwi products list"
    Then the exit status is 0
    And the JSON field "count" of the standard output is 1

  # Behaviour: CLI-435
  Scenario: The product tree prints each test plan with its test cases nested under it
    When I run "retriever kiwi products tree product-3"
    Then the exit status is 0
    And the standard output contains "Synthetic plan"
    And the standard output contains "Synthetic case"

  # Behaviour: CLI-435
  Scenario: The product tree in JSON mode is a schemaVersion 1 document with a count
    When I run "retriever --output json kiwi products tree product-3"
    Then the exit status is 0
    And the standard output is a JSON document with schemaVersion 1
    And the JSON field "count" of the standard output is 2

  # Behaviour: CLI-453
  Scenario: The product tree of an unknown product is a not-found failure with its own code
    When I run "retriever --output json kiwi products tree product-999"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.kiwi.collection_not_found" and retryable false

  Scenario: The product tree refuses an invalid product ID before any request
    When I run "retriever --output json kiwi products tree bad/id"
    Then the exit status is 2
    And the standard output is empty
    And the stub Kiwi server received no requests

  # Behaviour: CLI-454
  Scenario: A Kiwi document ID that is neither a plan nor a case is refused before any request
    When I run "retriever --output json kiwi documents get product-3"
    Then the exit status is 2
    And the standard output is empty
    And the stub Kiwi server received no requests

  # Behaviour: CLI-437
  Scenario: A case title and text that Kiwi escaped once are rendered as escaped Markdown once, never twice
    Given the stub Kiwi server holds this dataset:
      """
      {
        "Products":   [{"ID": 3, "Name": "Synthetic product"}],
        "Plans":      [{"ID": 12, "Name": "Synthetic plan", "Product": 3, "Parent": 0, "Text": "Plan text"}],
        "Cases":      [{"ID": 40, "Summary": "Tom &amp; Jerry", "Text": "Case &lt;b&gt; text", "Plans": [12], "Sortkey": 10, "Status": 1, "Priority": 1, "Category": 1}],
        "Categories": [{"ID": 1, "Name": "Functional", "Product": 3}],
        "Priorities": [{"ID": 1, "Value": "High"}],
        "Statuses":   [{"ID": 1, "Name": "PROPOSED"}]
      }
      """
    When I run "retriever kiwi documents get case-40"
    Then the exit status is 0
    And the standard output contains "# Tom &amp; Jerry"
    And the standard output contains "Case &lt;b&gt; text"
    And the standard output does not contain "&amp;amp;"
    And the standard output does not contain "<b>"

  # Behaviour: CLI-441
  Scenario: Adding a case to a plan sends one request and prints nothing to standard error
    When I run "retriever kiwi plans add-case plan-12 --case-id case-40 --source example-kiwi"
    Then the exit status is 0
    And the standard error is empty
    And the stub Kiwi server received 1 request to "TestPlan.add_case"

  # Behaviour: CLI-440
  Scenario: Updating a case with no field to change is a usage error and sends no request
    When I run "retriever --output json kiwi test-cases update case-40 --source example-kiwi"
    Then the exit status is 2
    And the standard output is empty
    And the stub Kiwi server received no requests

  # Behaviour: CLI-451
  # Behaviour: CLI-439
  Scenario: Creating a test case with a category name that matches no entry sends no create request
    When I run "retriever --output json kiwi test-cases create Synthetic-case --product product-3 --category Missing-category --priority High --source example-kiwi"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.kiwi.unresolved_name" and retryable false
    And the stub Kiwi server received 0 requests to "TestCase.create"

  # Behaviour: CLI-452
  Scenario: A Kiwi source whose API key is not set fails with key_missing and prints no output
    Given the environment variable "EXAMPLE_KIWI_API_KEY" is not set
    When I run "retriever --output json kiwi products list"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.kiwi.key_missing" and retryable false

  # Behaviour: CLI-452
  Scenario: A Kiwi read with two Kiwi sources needs the source flag
    Given the sources catalog holds these sources:
      | id           | kind | tags  | endpoint                  | credential_env       | identity_env |
      | example-kiwi | kiwi | tests | https://kiwi.example.test | EXAMPLE_KIWI_API_KEY |              |
      | second-kiwi  | kiwi | tests | https://kiwi.example.test | EXAMPLE_KIWI_API_KEY |              |
    When I run "retriever --output json kiwi products list"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.kiwi.ambiguous_source" and retryable false
    And the stub Kiwi server received no requests
