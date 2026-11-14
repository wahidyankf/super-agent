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

  @integration @e2e @unit
  Scenario: An issue read prints the summary as its title and the metadata beside it
    When I run "retriever jira issues get EXAMPLE-1"
    Then the exit status is 0
    And the standard output contains "Synthetic issue"
    And the standard output contains "To Do"
    And the standard output contains "Synthetic Person"

  @integration @e2e @unit
  Scenario: An issue read in JSON mode holds its title and its URL built from the catalog endpoint
    When I run "retriever --output json jira issues get EXAMPLE-1"
    Then the exit status is 0
    And the standard output is a JSON document with schemaVersion 1
    And the JSON field "title" of the standard output is "Synthetic issue"
    And the JSON field "url" of the standard output is "https://jira.example.test/browse/EXAMPLE-1"

  @integration @e2e @unit
  Scenario: An issue read writes its created and updated times in UTC
    When I run "retriever jira issues get EXAMPLE-1"
    Then the exit status is 0
    And the standard output contains "2026-01-01"
    And the standard output contains "2026-01-02"

  @integration @e2e @unit
  Scenario: An issue key in lower case is refused before any request
    When I run "retriever --output json jira issues get example-1"
    Then the exit status is 2
    And the standard output is empty
    And the stub Jira server received no requests

  @integration @e2e @unit
  Scenario: An issue with no description reads without a null value in its text
    Given the stub Jira server holds this configuration:
      """
      {
        "Email": "synthetic-user@example.test",
        "Token": "synthetic-jira-token",
        "Account": {"AccountID": "synthetic-account-id", "DisplayName": "Synthetic Person"},
        "Projects": [{"ID": "10001", "Key": "EXAMPLE", "Name": "Synthetic project"}],
        "Issues": [{"ID": "10030", "Key": "EXAMPLE-3", "Summary": "Empty issue", "Type": "Task", "Status": "To Do"}],
        "Comments": {}
      }
      """
    When I run "retriever jira issues get EXAMPLE-3"
    Then the exit status is 0
    And the standard output contains "Empty issue"
    And the standard output does not contain "null"

  @integration @e2e @unit
  Scenario: An ADF description with a heading and a bullet list renders as Markdown
    Given the stub Jira server holds this configuration:
      """
      {
        "Email": "synthetic-user@example.test",
        "Token": "synthetic-jira-token",
        "Account": {"AccountID": "synthetic-account-id", "DisplayName": "Synthetic Person"},
        "Projects": [{"ID": "10001", "Key": "EXAMPLE", "Name": "Synthetic project"}],
        "Issues": [{"ID": "10040", "Key": "EXAMPLE-4", "Summary": "Structured issue", "Type": "Task", "Status": "To Do", "Description": {"type": "doc", "version": 1, "content": [{"type": "heading", "attrs": {"level": 2}, "content": [{"type": "text", "text": "Synthetic heading"}]}, {"type": "bulletList", "content": [{"type": "listItem", "content": [{"type": "paragraph", "content": [{"type": "text", "text": "Synthetic item"}]}]}]}]}}],
        "Comments": {}
      }
      """
    When I run "retriever jira issues get EXAMPLE-4"
    Then the exit status is 0
    And the standard output contains "## Synthetic heading"
    And the standard output contains "- Synthetic item"

  @integration @e2e @unit
  Scenario: An ADF mention shows its name and never the account ID behind it
    Given the stub Jira server holds this configuration:
      """
      {
        "Email": "synthetic-user@example.test",
        "Token": "synthetic-jira-token",
        "Account": {"AccountID": "synthetic-account-id", "DisplayName": "Synthetic Person"},
        "Projects": [{"ID": "10001", "Key": "EXAMPLE", "Name": "Synthetic project"}],
        "Issues": [{"ID": "10050", "Key": "EXAMPLE-5", "Summary": "Mention issue", "Type": "Task", "Status": "To Do", "Description": {"type": "doc", "version": 1, "content": [{"type": "paragraph", "content": [{"type": "mention", "attrs": {"id": "acc-mention-secret", "text": "@Synthetic Person"}}]}]}}],
        "Comments": {}
      }
      """
    When I run "retriever jira issues get EXAMPLE-5"
    Then the exit status is 0
    And the standard output contains "Synthetic Person"
    And the standard output does not contain "acc-mention-secret"

  @integration @e2e @unit
  Scenario: An issue search sends the JQL as one JSON field and no page token
    When I run "retriever --output json jira issues search project=EXAMPLE"
    Then the exit status is 0
    And the stub Jira server's last request to "/rest/api/3/search/jql" body has the field "jql" equal to project=EXAMPLE
    And the stub Jira server's last request to "/rest/api/3/search/jql" body has the field "nextPageToken" equal to absent

  @integration @e2e @unit
  Scenario: An issue search with --limit asks Jira for at most that many issues
    When I run "retriever --output json jira issues search project=EXAMPLE --limit 5"
    Then the exit status is 0
    And the stub Jira server's last request to "/rest/api/3/search/jql" body has the field "maxResults" equal to 5

  @integration @e2e @unit
  Scenario: An issue search says so when more issues match than its page holds
    Given the stub Jira server holds this configuration:
      """
      {
        "Email": "synthetic-user@example.test",
        "Token": "synthetic-jira-token",
        "Account": {"AccountID": "synthetic-account-id", "DisplayName": "Synthetic Person"},
        "Projects": [{"ID": "10001", "Key": "EXAMPLE", "Name": "Synthetic project"}],
        "Issues": [{"ID": "10010", "Key": "EXAMPLE-1", "Summary": "First issue", "Type": "Task", "Status": "To Do"}, {"ID": "10011", "Key": "EXAMPLE-2", "Summary": "Second issue", "Type": "Task", "Status": "To Do"}],
        "Comments": {}
      }
      """
    When I run "retriever --output json jira issues search project=EXAMPLE --limit 1"
    Then the exit status is 0
    And the JSON field "count" of the standard output is 1
    And the JSON field "truncated" of the standard output is true

  @integration @e2e @unit
  Scenario: An issue search with no match exits 0 and reports a count of zero
    When I run "retriever --output json jira issues search project=NOMATCH"
    Then the exit status is 0
    And the JSON field "count" of the standard output is 0

  @integration @e2e @unit
  Scenario: An issue search with no JQL is a usage error and sends no request
    When I run "retriever jira issues search"
    Then the exit status is 2
    And the standard output is empty
    And the stub Jira server received no requests

  @integration @e2e @unit
  Scenario: A comment list prints each comment with its author and its text
    Given the stub Jira server holds this configuration:
      """
      {
        "Email": "synthetic-user@example.test",
        "Token": "synthetic-jira-token",
        "Account": {"AccountID": "synthetic-account-id", "DisplayName": "Synthetic Person"},
        "Projects": [{"ID": "10001", "Key": "EXAMPLE", "Name": "Synthetic project"}],
        "Issues": [{"ID": "10010", "Key": "EXAMPLE-1", "Summary": "Synthetic issue", "Type": "Task", "Status": "To Do"}],
        "Comments": {"EXAMPLE-1": [{"ID": "20001", "Author": "Synthetic Author", "Created": "2026-01-03T00:00:00.000+0000", "Body": {"type": "doc", "version": 1, "content": [{"type": "paragraph", "content": [{"type": "text", "text": "Synthetic comment"}]}]}}]}
      }
      """
    When I run "retriever jira comments list EXAMPLE-1"
    Then the exit status is 0
    And the standard output contains "Synthetic Author"
    And the standard output contains "Synthetic comment"
    And the standard output does not contain "acc-20001"

  @integration @e2e @unit
  Scenario: A comment list in JSON mode lists the comments of the issue with their count
    Given the stub Jira server holds this configuration:
      """
      {
        "Email": "synthetic-user@example.test",
        "Token": "synthetic-jira-token",
        "Account": {"AccountID": "synthetic-account-id", "DisplayName": "Synthetic Person"},
        "Projects": [{"ID": "10001", "Key": "EXAMPLE", "Name": "Synthetic project"}],
        "Issues": [{"ID": "10010", "Key": "EXAMPLE-1", "Summary": "Synthetic issue", "Type": "Task", "Status": "To Do"}],
        "Comments": {"EXAMPLE-1": [{"ID": "20001", "Author": "Synthetic Author", "Created": "2026-01-03T00:00:00.000+0000", "Body": {"type": "doc", "version": 1, "content": [{"type": "paragraph", "content": [{"type": "text", "text": "First comment"}]}]}}, {"ID": "20002", "Author": "Synthetic Author", "Created": "2026-01-04T00:00:00.000+0000", "Body": {"type": "doc", "version": 1, "content": [{"type": "paragraph", "content": [{"type": "text", "text": "Second comment"}]}]}}]}
      }
      """
    When I run "retriever --output json jira comments list EXAMPLE-1"
    Then the exit status is 0
    And the standard output is a JSON document with schemaVersion 1
    And the JSON field "count" of the standard output is 2
    And the JSON field "comments.0.id" of the standard output is "20001"
    And the JSON field "comments.1.id" of the standard output is "20002"

  @integration @e2e @unit
  Scenario: A comment list of an issue with no comments prints nothing and exits 0
    Given the stub Jira server holds this configuration:
      """
      {
        "Email": "synthetic-user@example.test",
        "Token": "synthetic-jira-token",
        "Account": {"AccountID": "synthetic-account-id", "DisplayName": "Synthetic Person"},
        "Projects": [{"ID": "10001", "Key": "EXAMPLE", "Name": "Synthetic project"}],
        "Issues": [{"ID": "10010", "Key": "EXAMPLE-1", "Summary": "Synthetic issue", "Type": "Task", "Status": "To Do"}],
        "Comments": {}
      }
      """
    When I run "retriever jira comments list EXAMPLE-1"
    Then the exit status is 0
    And the standard output is empty

  @integration @e2e @unit
  Scenario: A comment list of an issue that Jira does not hold is a not-found failure
    When I run "retriever --output json jira comments list EXAMPLE-999"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.jira.document_not_found" and retryable false

  @integration @e2e @unit
  Scenario: A comment list sends the page size and the offset in one request
    When I run "retriever --output json jira comments list EXAMPLE-1 --limit 1 --offset 0"
    Then the exit status is 0
    And the stub Jira server's last request to "/rest/api/3/issue/EXAMPLE-1/comment" query contains maxResults=1

  @integration @e2e @unit
  Scenario Outline: A comment list with a page size above 100 or a malformed issue key is refused before any request
    When I run "retriever --output json jira comments list <arguments>"
    Then the exit status is 2
    And the standard output is empty
    And the stub Jira server received no requests

    Examples:
      | arguments             |
      | EXAMPLE-1 --limit 101 |
      | not-a-key             |

  @integration @e2e @unit
  Scenario: A project list prints each project with its key and its name
    When I run "retriever jira projects list"
    Then the exit status is 0
    And the standard output contains "EXAMPLE"
    And the standard output contains "Synthetic project"

  @integration @e2e @unit
  Scenario: A project list filtered by query text lists only the matching projects
    Given the stub Jira server holds this configuration:
      """
      {
        "Email": "synthetic-user@example.test",
        "Token": "synthetic-jira-token",
        "Account": {"AccountID": "synthetic-account-id", "DisplayName": "Synthetic Person"},
        "Projects": [{"ID": "10001", "Key": "EXAMPLE", "Name": "Synthetic project"}, {"ID": "10002", "Key": "OTHER", "Name": "Second project"}],
        "Issues": [],
        "Comments": {}
      }
      """
    When I run "retriever --output json jira projects list --query Second"
    Then the exit status is 0
    And the JSON field "count" of the standard output is 1
    And the JSON field "projects.0.key" of the standard output is "OTHER"

  @integration @e2e @unit
  Scenario: A project list with no match exits 0 and reports a count of zero
    When I run "retriever --output json jira projects list --query nothing-matches-this"
    Then the exit status is 0
    And the JSON field "count" of the standard output is 0

  @integration @e2e @unit
  Scenario: A project list sends the page size and the offset in one request
    When I run "retriever --output json jira projects list --limit 1 --offset 0"
    Then the exit status is 0
    And the stub Jira server's last request to "/rest/api/3/project/search" query contains maxResults=1

  @integration @e2e @unit
  Scenario: A project list with a page size above 50 is refused before any request
    When I run "retriever --output json jira projects list --limit 51"
    Then the exit status is 2
    And the standard output is empty
    And the stub Jira server received no requests

  @integration @e2e @unit
  Scenario: The account read in JSON mode holds only the display name, the account type, and the active flag
    When I run "retriever --output json jira whoami"
    Then the exit status is 0
    And the standard output is a JSON document with schemaVersion 1
    And the JSON field "displayName" of the standard output is "Synthetic Person"
    And the JSON field "accountId" of the standard output is absent
    And the JSON field "emailAddress" of the standard output is absent
