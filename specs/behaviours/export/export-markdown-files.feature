# Export to files: one command writes one plain Markdown file into the folder named by --out. The file holds the
# source's own Markdown and no front matter, provenance block, or vault layout.
Feature: Export to Markdown files
  As a person who keeps notes on disk
  I want to export one document, test case, or issue as a Markdown file into a folder I choose
  So that the content is readable offline, and the export never touches anything else on disk

  Background:
    Given the environment variable "EXAMPLE_OUTLINE_API_KEY" is set to "synthetic-outline-key"
    And the sources catalog holds these sources:
      | id              | kind    | access | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | read   | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    And the stub Outline server holds these collections:
      | id | name      |
      | 1c1c1c1c-0000-4000-8000-000000000001 | Synthetic |
    And the stub Outline server holds these documents:
      | id    | collection | parent | title | markdown            |
      | d0c1d0c1-0000-4000-8000-000000000001 | 1c1c1c1c-0000-4000-8000-000000000001         |        | Setup | Synthetic body one  |

  @integration @e2e
  Scenario: Exporting a document writes one Markdown file that holds its body
    When I run "retriever outline export d0c1d0c1-0000-4000-8000-000000000001 --out out"
    Then the exit status is 0
    And the output folder "out" holds exactly 1 Markdown file
    And the Markdown file in the output folder "out" contains "Synthetic body one"

  @integration @e2e
  Scenario: An exported file is plain Markdown with no front matter
    When I run "retriever outline export d0c1d0c1-0000-4000-8000-000000000001 --out out"
    Then the exit status is 0
    And the Markdown file in the output folder "out" does not start with "---"

  @integration @e2e
  Scenario: Exporting the same document again replaces the file and keeps one copy
    Given the output folder "out" already holds a file named "setup.md" with the text "Earlier text"
    When I run "retriever outline export d0c1d0c1-0000-4000-8000-000000000001 --out out"
    Then the exit status is 0
    And the output folder "out" holds exactly 1 Markdown file
    And the Markdown file in the output folder "out" contains "Synthetic body one"
    And the Markdown file in the output folder "out" does not contain "Earlier text"

  @integration @e2e
  Scenario: An export keeps the other files already in the output folder
    Given the output folder "out" already holds a file named "notes.txt" with the text "Keep me"
    When I run "retriever outline export d0c1d0c1-0000-4000-8000-000000000001 --out out"
    Then the exit status is 0
    And the output folder "out" holds the file "notes.txt"
    And the file "notes.txt" in the output folder "out" contains "Keep me"

  @integration @e2e
  Scenario: Exporting a document that the wiki does not hold writes nothing
    When I run "retriever --output json outline export d0c9d0c9-0000-4000-8000-000000000009 --out out"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.outline.document_not_found" and retryable false
    And the output folder "out" holds no files

  @integration @e2e
  Scenario: Exporting without an output folder is a usage error and sends no request
    When I run "retriever --output json outline export d0c1d0c1-0000-4000-8000-000000000001"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.outline.usage_missing_argument" and retryable false
    And the stub Outline server received no requests

  @integration @e2e
  Scenario: The export of an Outline document writes into a folder that does not exist yet
    When I run "retriever outline export d0c1d0c1-0000-4000-8000-000000000001 --out nested/out"
    Then the exit status is 0
    And the output folder "nested/out" holds exactly 1 Markdown file

  @integration @e2e
  Scenario: A test case exported from Kiwi is written as one Markdown file
    Given the sources catalog file "reviewed.yml" holds these sources:
      | id           | kind | access | tags  | endpoint                  | credential_env       | identity_env |
      | example-kiwi | kiwi | read   | tests | https://kiwi.example.test | EXAMPLE_KIWI_API_KEY |              |
    And the environment variable "EXAMPLE_KIWI_API_KEY" is set to "synthetic-kiwi-key"
    And the stub Kiwi server holds this dataset:
      """
      {
        "Products":   [{"ID": 3, "Name": "Synthetic product"}],
        "Plans":      [],
        "Cases":      [{"ID": 40, "Summary": "Synthetic case", "Text": "Case text", "Plans": [], "Sortkey": 10, "Status": 1, "Priority": 1, "Category": 1}],
        "Categories": [{"ID": 1, "Name": "Functional", "Product": 3}],
        "Priorities": [{"ID": 1, "Name": "High"}],
        "Statuses":   [{"ID": 1, "Name": "PROPOSED"}]
      }
      """
    When I run "retriever --catalog reviewed.yml kiwi export case-40 --out out"
    Then the exit status is 0
    And the output folder "out" holds exactly 1 Markdown file
    And the Markdown file in the output folder "out" contains "Case text"

  @integration @e2e
  Scenario: An issue exported from Jira is written as one Markdown file that holds its description
    Given the sources catalog file "reviewed.yml" holds these sources:
      | id           | kind | access | tags    | endpoint                  | credential_env         | identity_env       |
      | example-jira | jira | read   | tickets | https://jira.example.test | EXAMPLE_JIRA_API_TOKEN | EXAMPLE_JIRA_EMAIL |
    And the environment variable "EXAMPLE_JIRA_API_TOKEN" is set to "synthetic-jira-token"
    And the environment variable "EXAMPLE_JIRA_EMAIL" is set to "synthetic-user@example.test"
    And the stub Jira server holds this configuration:
      """
      {
        "Email": "synthetic-user@example.test",
        "Token": "synthetic-jira-token",
        "Account": {"AccountID": "synthetic-account-id", "DisplayName": "Synthetic Person"},
        "Projects": [{"ID": "10001", "Key": "EXAMPLE", "Name": "Synthetic project"}],
        "Issues": [{"Key": "EXAMPLE-1", "Summary": "Synthetic issue", "Description": {"type": "doc", "version": 1, "content": [{"type": "paragraph", "content": [{"type": "text", "text": "Synthetic description"}]}]}}],
        "Comments": {}
      }
      """
    When I run "retriever --catalog reviewed.yml jira export EXAMPLE-1 --out out"
    Then the exit status is 0
    And the output folder "out" holds exactly 1 Markdown file
    And the Markdown file in the output folder "out" contains "Synthetic description"
    And the value of "EXAMPLE_JIRA_EMAIL" never appears in the standard output, the standard error, or the output folder
