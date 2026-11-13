# Kiwi TCMS source: the product, plan, and case reads, and the case writes. Every Kiwi request is one JSON-RPC POST to
# the source's endpoint, and a Kiwi failure maps to a retriever code of the kiwi area.
Feature: Kiwi test management operations
  As a person who keeps test plans and cases in Kiwi
  I want to read products, plans, and cases, and to create or change a case, from the command line
  So that test content reaches my notes and my agent harnesses without a browser

  Background:
    Given the sources catalog holds these sources:
      | id           | kind | access | tags  | endpoint                  | credential_env       | identity_env |
      | example-kiwi | kiwi | write  | tests | https://kiwi.example.test | EXAMPLE_KIWI_API_KEY |              |
    And the environment variable "EXAMPLE_KIWI_API_KEY" is set to "synthetic-kiwi-key"
    And the stub Kiwi server holds this dataset:
      """
      {
        "Products":   [{"ID": 3, "Name": "Synthetic product"}],
        "Plans":      [{"ID": 12, "Name": "Synthetic plan", "Product": 3, "Parent": 0, "Text": "Plan text"}],
        "Cases":      [{"ID": 40, "Summary": "Synthetic case", "Text": "Case text", "Plans": [12], "Sortkey": 10, "Status": 1, "Priority": 1, "Category": 1}],
        "Categories": [{"ID": 1, "Name": "Functional", "Product": 3}],
        "Priorities": [{"ID": 1, "Name": "High"}],
        "Statuses":   [{"ID": 1, "Name": "PROPOSED"}]
      }
      """

  @integration @e2e
  Scenario: The product list prints each product with its ID and name
    When I run "retriever --output json kiwi products list"
    Then the exit status is 0
    And the JSON field "products.0.id" of the standard output is "product-3"
    And the JSON field "products.0.name" of the standard output is "Synthetic product"

  @integration @e2e
  Scenario: The plan document prints its Markdown text
    When I run "retriever kiwi documents get plan-12"
    Then the exit status is 0
    And the standard output contains "Plan text"

  @integration @e2e
  Scenario: The case document prints its Markdown text in JSON mode with its ID
    When I run "retriever --output json kiwi documents get case-40"
    Then the exit status is 0
    And the standard output is a JSON document with schemaVersion 1
    And the JSON field "id" of the standard output is "case-40"

  @integration @e2e
  Scenario: A document that Kiwi does not hold is a not-found failure
    When I run "retriever --output json kiwi documents get case-999"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.kiwi.document_not_found" and retryable false

  @integration @e2e
  Scenario: Creating a test case sends one request and prints nothing to standard error
    When I run "retriever kiwi test-cases create Synthetic-case --product product-3 --category Functional --priority High --source example-kiwi"
    Then the exit status is 0
    And the standard error is empty
    And the stub Kiwi server received 1 request to "TestCase.create"

  @integration @e2e
  Scenario: A dry run of a case creation prints the change and sends no request
    When I run "retriever kiwi test-cases create Synthetic-case --product product-3 --category Functional --priority High --dry-run --source example-kiwi"
    Then the exit status is 0
    And the stub Kiwi server received no requests

  @integration @e2e
  Scenario: Updating a case sends one request for that case
    When I run "retriever kiwi test-cases update case-40 --summary Renamed-case --source example-kiwi"
    Then the exit status is 0
    And the stub Kiwi server received 1 request to "TestCase.update"

  @integration @e2e
  Scenario: A case write on a read-access source is refused before any request
    Given the sources catalog file "reviewed.yml" holds these sources:
      | id           | kind | access | tags  | endpoint                  | credential_env       | identity_env |
      | example-kiwi | kiwi | read   | tests | https://kiwi.example.test | EXAMPLE_KIWI_API_KEY |              |
    When I run "retriever --output json --catalog reviewed.yml kiwi test-cases create Synthetic-case --product product-3 --category Functional --priority High --source example-kiwi"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.kiwi.access_level_too_low" and retryable false
    And the stub Kiwi server received no requests

  @integration @e2e
  Scenario: A JSON-RPC error on a read never shows the upstream message
    Given the stub Kiwi server answers "TestCase.filter" with a JSON-RPC error with code -32000 and message "synthetic upstream detail"
    When I run "retriever --output json kiwi documents get case-40"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with a code that starts with "retriever.kiwi."
    And the standard error does not contain "synthetic upstream detail"

  @integration @e2e
  Scenario: A rejected write is sent once and is not retried
    Given the stub Kiwi server answers "TestCase.create" with status 500
    When I run "retriever --output json kiwi test-cases create Synthetic-case --product product-3 --category Functional --priority High --source example-kiwi"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.kiwi.upstream_failure" and retryable false
    And the stub Kiwi server received 1 request to "TestCase.create"

  @integration @e2e
  Scenario: Every Kiwi request is a POST over HTTPS to the source's own endpoint path
    When I run "retriever kiwi products list"
    Then the exit status is 0
    And every request to the stub Kiwi server used the HTTP method "POST"
    And every request to the stub Kiwi server used the path "/json-rpc/"
