# Transport rules shared by the Outline, Kiwi, and Jira adapters: HTTPS only, no redirects, bounded waits, bounded
# replies, bounded retries, and one request at a time. Each scenario uses the local stub servers, never a real host.
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

  @integration @e2e
  Scenario: A source whose endpoint is plain HTTP is refused before any request
    Given the sources catalog file "reviewed.yml" holds these sources:
      | id              | kind    | access | tags | endpoint                 | credential_env          | identity_env |
      | plain-outline   | outline | read   | wiki | http://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    When I run "retriever --output json --catalog reviewed.yml outline collections list --source plain-outline"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.outline.https_required" and retryable false
    And the stub Outline server received no requests

  @integration @e2e
  Scenario: A redirect on a read is reported and not followed
    Given the stub Outline server redirects "collections.list" to "https://elsewhere.example.test/"
    When I run "retriever --output json outline collections list"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.outline.redirected" and retryable false
    And the stub Outline server received 1 request to "collections.list"

  @integration @e2e
  Scenario: A request that gets no reply within the bound stops with status 124
    Given the environment variable "RETRIEVER_OUTLINE_TEST_TIMEOUT" is set to "300ms"
    And the stub Outline server never answers "collections.list"
    When I run "retriever --output json outline collections list"
    Then the exit status is 124
    And the standard error is a JSON error with the code "retriever.outline.timed_out" and retryable true

  @integration @e2e
  Scenario: A rate-limited read waits for Retry-After once and then succeeds
    Given the stub Outline server answers the first 1 request to "collections.list" with status 429 and Retry-After "1"
    When I run "retriever outline collections list"
    Then the exit status is 0
    And the stub Outline server received 2 requests to "collections.list"

  @integration @e2e
  Scenario: Repeated rate-limit replies stop after three retries and report a retryable failure
    Given the stub Outline server answers "collections.list" with status 429 and Retry-After "0"
    When I run "retriever --output json outline collections list"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.outline.rate_limited" and retryable true
    And the stub Outline server received 4 requests to "collections.list"

  @integration @e2e
  Scenario: A Retry-After longer than 60 seconds fails the read without a retry
    Given the stub Outline server answers "collections.list" with status 429 and Retry-After "61"
    When I run "retriever --output json outline collections list"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.outline.retry_after_too_long" and retryable false
    And the stub Outline server received 1 request to "collections.list"

  @integration @e2e
  Scenario: A reply larger than the 10 MiB cap is refused as too large
    Given the stub Outline server answers "collections.list" with a body of 11 MiB
    When I run "retriever --output json outline collections list"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.outline.reply_too_large" and retryable false

  @integration @e2e
  Scenario: A reply that is not JSON is an unexpected response and names no host
    Given the stub Outline server answers "collections.list" with the body:
      """
      <html>not what the API sends</html>
      """
    When I run "retriever --output json outline collections list"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.outline.unexpected_response" and retryable false
    And the standard error does not contain "docs.example.test"

  @integration @e2e
  Scenario: A source that cannot be reached is reported as unreachable and is retryable
    Given no stub Outline server is listening
    When I run "retriever --output json outline collections list"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.outline.unreachable" and retryable true
    And the standard error does not contain "127.0.0.1"

  @integration @e2e
  Scenario: Requests to one source run one at a time
    When I run "retriever outline documents list --limit 1"
    Then the stub Outline server's peak concurrency was 1

  @integration @e2e
  Scenario: Every Outline request uses the POST method and its own method path
    When I run "retriever outline collections list"
    Then the exit status is 0
    And every request to the stub Outline server used the HTTP method "POST"

  @integration @e2e
  Scenario: A Kiwi write that is refused by a redirect is not sent a second time
    Given the environment variable "EXAMPLE_KIWI_API_KEY" is set to "synthetic-kiwi-key"
    And the sources catalog file "reviewed.yml" holds these sources:
      | id           | kind | access | tags  | endpoint                  | credential_env       | identity_env |
      | example-kiwi | kiwi | write  | tests | https://kiwi.example.test | EXAMPLE_KIWI_API_KEY |              |
    And the stub Kiwi server redirects "test-cases.create" to "https://elsewhere.example.test/"
    When I run "retriever --output json --catalog reviewed.yml kiwi test-cases create Synthetic --product product-1 --category Functional --priority High --source example-kiwi"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.kiwi.redirected" and retryable false
    And the stub Kiwi server received 1 request to "test-cases.create"
