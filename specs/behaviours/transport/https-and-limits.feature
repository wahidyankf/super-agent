# Transport rules shared by the Outline, Kiwi, and Jira adapters: HTTPS only, no redirects, bounded waits, bounded
# replies, bounded retries, and one request at a time. Each scenario uses the local stub servers, never a real host.
# Spacing and waits are measured on the arrival times the stubs record. The unit and integration levels run on a fake
# clock, so their spacing is exact and no test waits for it; the end-to-end level measures real time within a tolerance.
Feature: Transport safety and bounded requests
  As a person who points retriever at real tools
  I want every request to go over HTTPS to the configured source, and every wait and reply to be bounded
  So that a slow, hostile, or misconfigured server cannot hang the command, leak a key, or flood the output

  Background:
    Given the sources catalog holds these sources:
      | id              | kind    | access | tags    | endpoint                  | credential_env          | identity_env       |
      | example-outline | outline | read   | wiki    | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |                    |
      | example-kiwi    | kiwi    | read   | tests   | https://kiwi.example.test | EXAMPLE_KIWI_API_KEY    |                    |
      | example-jira    | jira    | read   | tickets | https://jira.example.test | EXAMPLE_JIRA_API_TOKEN  | EXAMPLE_JIRA_EMAIL |
    And the environment variable "EXAMPLE_OUTLINE_API_KEY" is set to "synthetic-outline-key"
    And the environment variable "EXAMPLE_KIWI_API_KEY" is set to "synthetic-kiwi-key"
    And the environment variable "EXAMPLE_JIRA_API_TOKEN" is set to "synthetic-jira-token"
    And the environment variable "EXAMPLE_JIRA_EMAIL" is set to "synthetic-user@example.test"
    And the stub Outline server holds these collections:
      | id | name      |
      | 1c1c1c1c-0000-4000-8000-000000000001 | Synthetic |

  @integration @e2e @unit
  Scenario: A source whose endpoint is plain HTTP is refused before any request
    Given the sources catalog file "reviewed.yml" holds these sources:
      | id              | kind    | access | tags | endpoint                 | credential_env          | identity_env |
      | plain-outline   | outline | read   | wiki | http://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    When I run "retriever --output json --catalog reviewed.yml outline collections list --source plain-outline"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.outline.https_required" and retryable false
    And the stub Outline server received no requests

  @integration @e2e @unit
  Scenario: A Kiwi source whose endpoint is plain HTTP is refused before any request
    Given the sources catalog file "reviewed.yml" holds these sources:
      | id         | kind | access | tags  | endpoint                 | credential_env       | identity_env |
      | plain-kiwi | kiwi | read   | tests | http://kiwi.example.test | EXAMPLE_KIWI_API_KEY |              |
    When I run "retriever --output json --catalog reviewed.yml kiwi products list --source plain-kiwi"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.kiwi.https_required" and retryable false
    And the stub Kiwi server received no requests

  @integration @e2e @unit
  Scenario: A Jira source whose endpoint is plain HTTP is refused before any request
    Given the sources catalog file "reviewed.yml" holds these sources:
      | id         | kind | access | tags    | endpoint                 | credential_env         | identity_env       |
      | plain-jira | jira | read   | tickets | http://jira.example.test | EXAMPLE_JIRA_API_TOKEN | EXAMPLE_JIRA_EMAIL |
    When I run "retriever --output json --catalog reviewed.yml jira whoami --source plain-jira"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.jira.https_required" and retryable false
    And the stub Jira server received no requests

  @integration @e2e @unit
  Scenario: A redirect on an Outline read is reported and not followed
    Given the stub Outline server redirects "collections.list" to "https://elsewhere.example.test/"
    When I run "retriever --output json outline collections list"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.outline.redirected" and retryable false
    And the stub Outline server received 1 request to "collections.list"

  @integration @e2e @unit
  Scenario: A redirect on a Kiwi read is reported and not followed
    Given the stub Kiwi server redirects "TestCase.filter" to "https://elsewhere.example.test/"
    When I run "retriever --output json kiwi documents get case-40"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.kiwi.redirected" and retryable false
    And the stub Kiwi server received 1 request to "TestCase.filter"

  @integration @e2e @unit
  Scenario: A redirect on a Jira read is reported and not followed
    Given the stub Jira server redirects "/rest/api/3/myself" to "https://elsewhere.example.test/"
    When I run "retriever --output json jira whoami"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.jira.redirected" and retryable false
    And the stub Jira server received 1 request to "/rest/api/3/myself"

  @integration @e2e @unit
  Scenario: A request to an Outline source that gets no reply within the bound stops with status 124
    Given the environment variable "RETRIEVER_OUTLINE_TEST_TIMEOUT" is set to "300ms"
    And the stub Outline server never answers "collections.list"
    When I run "retriever --output json outline collections list"
    Then the exit status is 124
    And the standard error is a JSON error with the code "retriever.outline.timed_out" and retryable true

  @integration @e2e @unit
  Scenario: A request to a Kiwi source that gets no reply within the bound stops with status 124
    Given the environment variable "RETRIEVER_KIWI_TEST_TIMEOUT" is set to "300ms"
    And the stub Kiwi server never answers "TestCase.filter"
    When I run "retriever --output json kiwi documents get case-40"
    Then the exit status is 124
    And the standard error is a JSON error with the code "retriever.kiwi.timed_out" and retryable true

  @integration @e2e @unit
  Scenario: A request to a Jira source that gets no reply within the bound stops with status 124
    Given the environment variable "RETRIEVER_JIRA_TEST_TIMEOUT" is set to "300ms"
    And the stub Jira server never answers "/rest/api/3/myself"
    When I run "retriever --output json jira whoami"
    Then the exit status is 124
    And the standard error is a JSON error with the code "retriever.jira.timed_out" and retryable true

  @integration @e2e @unit
  Scenario: A rate-limited Outline read waits for Retry-After and then succeeds, with the wait measured
    Given the stub Outline server answers the first 1 request to "collections.list" with status 429 and Retry-After "1"
    When I run "retriever outline collections list"
    Then the exit status is 0
    And the stub Outline server received 2 requests to "collections.list"
    And the requests to the stub Outline server were 1000 ms apart

  @integration @e2e @unit
  Scenario: Requests to one Outline source are spaced 100 milliseconds apart
    Given the stub Outline server answers the first 2 requests to "collections.list" with status 429 and Retry-After "0"
    When I run "retriever outline collections list"
    Then the exit status is 0
    And the stub Outline server received 3 requests to "collections.list"
    And the requests to the stub Outline server were at least 100 ms apart
    And the requests to the stub Outline server were 100 ms apart

  @integration @e2e @unit
  Scenario: Repeated Outline rate-limit replies stop after three retries and report a retryable failure with status 124
    Given the stub Outline server answers "collections.list" with status 429 and Retry-After "0"
    When I run "retriever --output json outline collections list"
    Then the exit status is 124
    And the standard error is a JSON error with the code "retriever.outline.rate_limited" and retryable true
    And the stub Outline server received 4 requests to "collections.list"

  @integration @e2e @unit
  Scenario: A Retry-After longer than 60 seconds fails the Outline read without a retry
    Given the stub Outline server answers "collections.list" with status 429 and Retry-After "61"
    When I run "retriever --output json outline collections list"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.outline.retry_after_too_long" and retryable false
    And the stub Outline server received 1 request to "collections.list"

  @integration @e2e @unit
  Scenario: Every Outline request is sent over TLS with the source's own host name
    When I run "retriever outline collections list"
    Then the exit status is 0
    And every request to the stub Outline server arrived over TLS
    And every request to the stub Outline server had the Host header "docs.example.test"

  @integration @e2e @unit
  Scenario: A write command on a read-access Outline source sends no request
    When I run "retriever --output json outline documents create --title Synthetic-note --text Synthetic --source example-outline"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.outline.access_level_too_low" and retryable false
    And the stub Outline server received no requests

  @integration @e2e @unit
  Scenario: A reply larger than the 10 MiB cap is refused as too large
    Given the stub Outline server answers "collections.list" with a body of 11 MiB
    When I run "retriever --output json outline collections list"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.outline.reply_too_large" and retryable false

  @integration @e2e @unit
  Scenario: A reply that is not JSON is an unexpected response and names no host
    Given the stub Outline server answers "collections.list" with the body:
      """
      <html>not what the API sends</html>
      """
    When I run "retriever --output json outline collections list"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.outline.unexpected_response" and retryable false
    And the standard error does not contain "docs.example.test"

  @integration @e2e @unit
  Scenario: A source that cannot be reached is reported as unreachable and is retryable
    Given no stub Outline server is listening
    When I run "retriever --output json outline collections list"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.outline.unreachable" and retryable true
    And the standard error does not contain "127.0.0.1"

  @integration @e2e @unit
  Scenario: Requests to one Outline source run one at a time
    When I run "retriever outline documents list --limit 1"
    Then the stub Outline server's peak concurrency was 1

  @integration @e2e @unit
  Scenario: Every Outline request uses the POST method and its own method path
    When I run "retriever outline collections list"
    Then the exit status is 0
    And every request to the stub Outline server used the HTTP method "POST"

  @integration @e2e @unit
  Scenario: Every Kiwi request is sent over TLS with the source's own host name
    Given the stub Kiwi server holds this dataset:
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
    When I run "retriever kiwi products list"
    Then the exit status is 0
    And every request to the stub Kiwi server arrived over TLS
    And every request to the stub Kiwi server had the Host header "kiwi.example.test"

  @integration @e2e @unit
  Scenario: A Kiwi read that is answered 429 waits for Retry-After and then succeeds, with the wait measured
    Given the stub Kiwi server holds this dataset:
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
    And the stub Kiwi server answers the first request to "TestCase.filter" with status 429 and Retry-After "1"
    When I run "retriever --output json kiwi documents get case-40"
    Then the exit status is 0
    And the stub Kiwi server received 2 requests to "TestCase.filter"
    And the requests to the stub Kiwi server to "TestCase.filter" were 1000 ms apart

  @integration @e2e @unit
  Scenario: A Kiwi write that is answered 429 waits and is sent again, and the write lands once
    Given the sources catalog file "reviewed.yml" holds these sources:
      | id           | kind | access | tags  | endpoint                  | credential_env       | identity_env |
      | example-kiwi | kiwi | write  | tests | https://kiwi.example.test | EXAMPLE_KIWI_API_KEY |              |
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
    And the stub Kiwi server answers the first request to "TestCase.create" with status 429 and Retry-After "1"
    When I run "retriever --output json --catalog reviewed.yml kiwi test-cases create Synthetic-case --product product-3 --category Functional --priority High --source example-kiwi"
    Then the exit status is 0
    And the stub Kiwi server received 2 requests to "TestCase.create"
    And the requests to the stub Kiwi server to "test-cases.create" were 1000 ms apart

  @integration @e2e @unit
  Scenario: Kiwi requests are spaced 100 milliseconds apart and never overlap
    Given the stub Kiwi server holds this dataset:
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
    And the stub Kiwi server answers the first request to "Product.filter" with status 429 and Retry-After "0"
    When I run "retriever kiwi products list"
    Then the exit status is 0
    And the stub Kiwi server received 2 requests to "Product.filter"
    And the requests to the stub Kiwi server were at least 100 ms apart
    And the requests to the stub Kiwi server were 100 ms apart
    And the requests to the stub Kiwi server never overlapped

  @integration @e2e @unit
  Scenario: Repeated Kiwi rate-limit replies stop after three retries and report a retryable failure with status 124
    Given the stub Kiwi server answers every request with status 429 and Retry-After "0"
    When I run "retriever --output json kiwi documents get case-40"
    Then the exit status is 124
    And the standard error is a JSON error with the code "retriever.kiwi.rate_limited" and retryable true
    And the stub Kiwi server received 4 requests to "TestCase.filter"

  @integration @e2e @unit
  Scenario: A Kiwi Retry-After longer than 60 seconds fails the read without a retry
    Given the stub Kiwi server answers every request with status 429 and Retry-After "61"
    When I run "retriever --output json kiwi documents get case-40"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.kiwi.retry_after_too_long" and retryable false
    And the stub Kiwi server received 1 request to "TestCase.filter"

  @integration @e2e @unit
  Scenario: A Kiwi write whose reply is over the 10 MiB cap is not known to have failed
    Given the sources catalog file "reviewed.yml" holds these sources:
      | id           | kind | access | tags  | endpoint                  | credential_env       | identity_env |
      | example-kiwi | kiwi | write  | tests | https://kiwi.example.test | EXAMPLE_KIWI_API_KEY |              |
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
    And the stub Kiwi server answers "TestCase.create" with an oversize body
    When I run "retriever --output json --catalog reviewed.yml kiwi test-cases create Synthetic-case --product product-3 --category Functional --priority High --source example-kiwi"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.kiwi.write_outcome_unknown" and retryable false
    And the stub Kiwi server received 1 request to "TestCase.create"

  @integration @e2e @unit
  Scenario: A Kiwi write that is refused by a redirect is not sent a second time
    Given the sources catalog file "reviewed.yml" holds these sources:
      | id           | kind | access | tags  | endpoint                  | credential_env       | identity_env |
      | example-kiwi | kiwi | write  | tests | https://kiwi.example.test | EXAMPLE_KIWI_API_KEY |              |
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
    And the stub Kiwi server redirects "test-cases.create" to "https://elsewhere.example.test/"
    When I run "retriever --output json --catalog reviewed.yml kiwi test-cases create Synthetic --product product-3 --category Functional --priority High --source example-kiwi"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.kiwi.redirected" and retryable false
    And the stub Kiwi server received 1 request to "test-cases.create"

  @integration @e2e @unit
  Scenario: Every Jira request is sent over TLS to the Jira host, which is also the TLS name
    When I run "retriever jira whoami"
    Then the exit status is 0
    And every request to the stub Jira server arrived over TLS
    And every request to the stub Jira server had the Host header "jira.example.test"
    And every request to the stub Jira server had the TLS name "jira.example.test"

  @integration @e2e @unit
  Scenario: Every Jira request uses the GET method for the account read and its own path
    When I run "retriever jira whoami"
    Then the exit status is 0
    And every request to the stub Jira server used the HTTP method "GET"
    And every request to the stub Jira server used the path "/rest/api/3/myself"

  @integration @e2e @unit
  Scenario: A Jira reply that sets a cookie is not answered with that cookie on the next request
    Given the stub Jira server sets the cookie "synthetic_session" on every reply
    And the stub Jira server answers the first 1 request to "/rest/api/3/myself" with status 429 and Retry-After "0"
    When I run "retriever jira whoami"
    Then the exit status is 0
    And the stub Jira server received 2 requests to "/rest/api/3/myself"
    And no request to the stub Jira server carried a cookie

  @integration @e2e @unit
  Scenario: Jira requests are spaced 100 milliseconds apart and never overlap
    Given the stub Jira server answers the first 2 requests to "/rest/api/3/myself" with status 429 and Retry-After "0"
    When I run "retriever jira whoami"
    Then the exit status is 0
    And the stub Jira server received 3 requests to "/rest/api/3/myself"
    And the requests to the stub Jira server were at least 100 ms apart
    And the requests to the stub Jira server were 100 ms apart
    And the requests to the stub Jira server never overlapped

  @integration @e2e @unit
  Scenario: A Jira reply larger than the 10 MiB cap is refused and its body is not shown
    Given the stub Jira server answers "/rest/api/3/myself" with an oversize body
    When I run "retriever --output json jira whoami"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.jira.reply_too_large" and retryable false
    And the standard error does not contain "synthetic-jira-token"
