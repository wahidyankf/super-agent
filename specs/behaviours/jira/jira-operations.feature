# Jira Cloud source: read-only commands for the account, projects, issues, and comments. Descriptions arrive as
# Atlassian Document Format and are rendered to escaped Markdown. Jira has no write command.
Feature: Jira reads
  As a person who follows work in Jira Cloud
  I want to read my account, projects, issues, and comments from the command line, safely
  So that issue content reaches my notes and harnesses without any change to Jira

  Background:
    Given the sources catalog holds these sources:
      | id           | kind | tags    | endpoint                  | credential_env         | identity_env       |
      | example-jira | jira | tickets | https://jira.example.test | EXAMPLE_JIRA_API_TOKEN | EXAMPLE_JIRA_EMAIL |
    And the environment variable "EXAMPLE_JIRA_API_TOKEN" is set to "synthetic-jira-token"
    And the environment variable "EXAMPLE_JIRA_EMAIL" is set to "synthetic-user@example.test"
    And the stub Jira server holds this configuration:
      """
      {
        "Email": "synthetic-user@example.test",
        "Token": "synthetic-jira-token",
        "Account": {"AccountID": "synthetic-account-id", "DisplayName": "Synthetic Person"},
        "Projects": [{"ID": "10001", "Key": "EXAMPLE", "Name": "Synthetic project"}],
        "Issues": [{"ID": "10010", "Key": "EXAMPLE-1", "Summary": "Synthetic issue", "Type": "Task", "Status": "To Do", "Priority": "Medium", "Reporter": "Synthetic Person", "Created": "2026-01-01T00:00:00.000+0000", "Updated": "2026-01-02T00:00:00.000+0000", "Description": {"type": "doc", "version": 1, "content": [{"type": "paragraph", "content": [{"type": "text", "text": "Synthetic description"}]}]}}],
        "Comments": {"EXAMPLE-1": [{"ID": "20001", "Body": {"type": "doc", "version": 1, "content": [{"type": "paragraph", "content": [{"type": "text", "text": "Synthetic comment"}]}]}}]}
      }
      """

  @integration @e2e @unit
  Scenario: The account read prints the display name and never the account ID or the e-mail address
    When I run "retriever jira whoami"
    Then the exit status is 0
    And the standard output contains "Synthetic Person"
    And the standard output does not contain "synthetic-account-id"
    And the standard output does not contain "synthetic-user@example.test"

  @integration @e2e @unit
  Scenario: The project list prints each project with its key and name
    When I run "retriever --output json jira projects list"
    Then the exit status is 0
    And the JSON field "projects.0.key" of the standard output is "EXAMPLE"
    And the JSON field "projects.0.name" of the standard output is "Synthetic project"

  @integration @e2e @unit
  Scenario: An issue search with a JQL query lists the matching issues
    When I run "retriever --output json jira issues search project=EXAMPLE"
    Then the exit status is 0
    And the JSON field "issues.0.key" of the standard output is "EXAMPLE-1"

  @integration @e2e @unit
  Scenario: An issue search with a page size outside 1 to 100 is a usage error and sends no request
    When I run "retriever --output json jira issues search project=EXAMPLE --limit 500"
    Then the exit status is 2
    And the standard output is empty
    And the stub Jira server received no requests

  @integration @e2e @unit
  Scenario: An issue read renders its description as Markdown text
    When I run "retriever jira issues get EXAMPLE-1"
    Then the exit status is 0
    And the standard output contains "Synthetic description"

  @integration @e2e @unit
  Scenario: An issue that Jira does not hold is a failure with its own code
    When I run "retriever --output json jira issues get EXAMPLE-999"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.jira.document_not_found" and retryable false

  @integration @e2e @unit
  Scenario: A comment list prints the comment text of the issue
    When I run "retriever jira comments list EXAMPLE-1"
    Then the exit status is 0
    And the standard output contains "Synthetic comment"

  @integration @e2e @unit
  Scenario: An issue key with characters outside the allowed form is refused before any request
    When I run "retriever --output json jira issues get not-a-key"
    Then the exit status is 2
    And the stub Jira server received no requests

  @integration @e2e @unit
  Scenario: A description that holds a script link and raw HTML is shown as escaped text with no live link
    Given the stub Jira server holds this configuration:
      """
      {
        "Email": "synthetic-user@example.test",
        "Token": "synthetic-jira-token",
        "Account": {"AccountID": "synthetic-account-id", "DisplayName": "Synthetic Person"},
        "Projects": [{"ID": "10001", "Key": "EXAMPLE", "Name": "Synthetic project"}],
        "Issues": [{"ID": "10020", "Key": "EXAMPLE-2", "Summary": "Hostile issue", "Description": {"type": "doc", "version": 1, "content": [{"type": "paragraph", "content": [{"type": "text", "text": "<script>alert(1)</script>", "marks": [{"type": "link", "attrs": {"href": "javascript:alert(1)"}}]}]}]}}],
        "Comments": {}
      }
      """
    When I run "retriever jira issues get EXAMPLE-2"
    Then the exit status is 0
    And the standard output does not contain "javascript:"
    And the standard output does not contain "<script>"

  @integration @e2e @unit
  Scenario: A Jira read sends the token and e-mail address only to the Jira endpoint
    When I run "retriever jira whoami"
    Then the exit status is 0
    And every request to the stub Jira server carried the credentials of the synthetic e-mail address and token
    And the value of "EXAMPLE_JIRA_API_TOKEN" never appears in the standard output, the standard error, or the output folder

  @integration @e2e @unit
  Scenario: A rate-limited Jira read waits for Retry-After once and then succeeds, with the wait measured
    Given the stub Jira server answers the first 1 request to "/rest/api/3/myself" with status 429 and Retry-After "1"
    When I run "retriever jira whoami"
    Then the exit status is 0
    And the stub Jira server received 2 requests to "/rest/api/3/myself"
    And the requests to the stub Jira server to "/rest/api/3/myself" were 1000 ms apart

  @integration @e2e @unit
  Scenario: A Jira read that keeps answering 429 stops after three retries with status 124
    Given the stub Jira server answers every request with status 429 and Retry-After "0"
    When I run "retriever --output json jira whoami"
    Then the exit status is 124
    And the standard error is a JSON error with the code "retriever.jira.rate_limited" and retryable true
    And the stub Jira server received 4 requests to "/rest/api/3/myself"

  @integration @e2e @unit
  Scenario: A Jira Retry-After longer than 60 seconds fails the read at once, without a retry
    Given the stub Jira server answers every request with status 429 and Retry-After "61"
    When I run "retriever --output json jira whoami"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.jira.retry_after_too_long" and retryable false
    And the stub Jira server received 1 request to "/rest/api/3/myself"

  @integration @e2e @unit
  Scenario Outline: A Jira status that is not a success ends the command with its own code, and never shows the token
    Given the stub Jira server answers "/rest/api/3/myself" with status <status>
    When I run "retriever --output json jira whoami"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "<code>" and retryable false
    And the standard error does not contain "synthetic-jira-token"

    Examples:
      | status | code                            |
      | 400    | retriever.jira.bad_request      |
      | 401    | retriever.jira.key_rejected     |
      | 403    | retriever.jira.access_denied    |
      | 404    | retriever.jira.not_found        |
      | 500    | retriever.jira.upstream_failure |

  @integration @e2e @unit
  Scenario: A Jira JQL search refused with 400 shows the generic refusal and not the text Jira sent
    Given the stub Jira server answers "/rest/api/3/search/jql" with status 400
    When I run "retriever --output json jira issues search project=EXAMPLE"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.jira.bad_request" and retryable false
    And the standard error does not contain "Scripted fault"

  @integration @e2e @unit
  Scenario: The removed search path is never used, even when the search path answers 410
    Given the stub Jira server answers "/rest/api/3/search/jql" with status 410
    When I run "retriever --output json jira issues search project=EXAMPLE"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.jira.operation_unavailable" and retryable false
    And the stub Jira server received 1 request to "/rest/api/3/search/jql"
    And the stub Jira server received 0 requests to "/rest/api/3/search"

  @integration @e2e @unit
  Scenario: A Jira reply that is not JSON is an unexpected response that names no host
    Given the stub Jira server answers "/rest/api/3/myself" with a body that is not JSON
    When I run "retriever --output json jira whoami"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.jira.unexpected_response" and retryable false
    And the standard error does not contain "jira.example.test"

  @integration @e2e @unit
  Scenario: A Jira reply that lacks the members the command reads is an unexpected response
    Given the stub Jira server answers "/rest/api/3/myself" with a body that lacks its members
    When I run "retriever --output json jira whoami"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.jira.unexpected_response" and retryable false

  @integration @e2e @unit
  Scenario: A Jira search that is answered with a page token reads one page, marks it truncated, and sends no second request
    Given the stub Jira server repeats the page token on "/rest/api/3/search/jql"
    When I run "retriever --output json jira issues search project=EXAMPLE"
    Then the exit status is 0
    And the JSON field "truncated" of the standard output is true
    And the stub Jira server received 1 request to "/rest/api/3/search/jql"

  @integration @e2e @unit
  Scenario: A Jira project listing whose empty page still carries a page token stops with an unexpected response
    Given the stub Jira server answers "/rest/api/3/project/search" with an empty page that still has a page token
    When I run "retriever --output json jira projects list"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.jira.unexpected_response" and retryable false

  @integration @e2e @unit
  Scenario: A Jira source that cannot be reached is reported as unreachable, is retryable, and names no host
    Given no stub Jira server is listening
    When I run "retriever --output json jira whoami"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.jira.unreachable" and retryable true
    And the standard error does not contain "127.0.0.1"
    And the standard error does not contain "jira.example.test"

  @unit @integration @e2e
  Scenario: Jira has no write command, so a create attempt is a usage error
    When I run "retriever jira issues create"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is not empty
