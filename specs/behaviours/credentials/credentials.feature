# Credentials: where each source's key comes from, what happens when it is missing or refused, and where it may go.
# All values are synthetic. The keys never appear in the catalog and never appear in any output.
Feature: Source credentials
  As a person who keeps keys out of files that get shared
  I want each key read from the environment or the env file, sent only to its own source, and never printed
  So that a leaked transcript, log, or catalog never exposes a credential

  Background:
    Given the sources catalog holds these sources:
      | id              | kind    | tags      | endpoint                  | credential_env          | identity_env       |
      | example-outline | outline | wiki      | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |                    |
      | example-kiwi    | kiwi    | tests     | https://kiwi.example.test | EXAMPLE_KIWI_API_KEY    |                    |
      | example-jira    | jira    | tickets   | https://jira.example.test | EXAMPLE_JIRA_API_TOKEN  | EXAMPLE_JIRA_EMAIL |
    And the stub Outline server holds these collections:
      | id | name       |
      | 1c1c1c1c-0000-4000-8000-000000000001 | Synthetic  |

  @integration @e2e @unit
  Scenario: The Outline key is read from the process environment and sent as a Bearer header
    Given the environment variable "EXAMPLE_OUTLINE_API_KEY" is set to "synthetic-outline-key"
    When I run "retriever outline collections list"
    Then the exit status is 0
    And the stub Outline server received at least 1 request
    And every request to the stub Outline server carried the Authorization header "Bearer synthetic-outline-key"

  @integration @e2e @unit
  Scenario: The Outline key is read from the env file in the current directory
    Given the env file ".env.local" holds these lines:
      """
      EXAMPLE_OUTLINE_API_KEY=synthetic-outline-key
      """
    When I run "retriever outline collections list"
    Then the exit status is 0
    And every request to the stub Outline server carried the Authorization header "Bearer synthetic-outline-key"

  @integration @e2e @unit
  Scenario: The process environment wins over the env file for the same variable
    Given the environment variable "EXAMPLE_OUTLINE_API_KEY" is set to "synthetic-from-env"
    And the env file ".env.local" holds these lines:
      """
      EXAMPLE_OUTLINE_API_KEY=synthetic-from-file
      """
    When I run "retriever outline collections list"
    Then the exit status is 0
    And every request to the stub Outline server carried the Authorization header "Bearer synthetic-from-env"

  @integration @e2e @unit
  Scenario: A Kiwi key is sent only to the Kiwi source and never to the Outline source
    Given the environment variable "EXAMPLE_KIWI_API_KEY" is set to "synthetic-kiwi-key"
    And the environment variable "EXAMPLE_OUTLINE_API_KEY" is set to "synthetic-outline-key"
    When I run "retriever kiwi products list"
    Then the exit status is 0
    And every request to the stub Kiwi server carried the Authorization header "Bearer synthetic-kiwi-key"
    And the stub Outline server received no requests

  @integration @e2e @unit
  Scenario Outline: A missing key stops the command before any request is sent
    Given the environment variable "<variable>" is not set
    And the environment variable "EXAMPLE_JIRA_EMAIL" is set to "synthetic-user@example.test"
    When I run "retriever --output json <command>"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "<code>" and retryable false
    And the stub <kind> server received no requests

    Examples:
      | variable                | command                  | code                         | kind    |
      | EXAMPLE_OUTLINE_API_KEY | outline collections list | retriever.outline.key_missing | Outline |
      | EXAMPLE_KIWI_API_KEY    | kiwi products list       | retriever.kiwi.key_missing    | Kiwi    |
      | EXAMPLE_JIRA_API_TOKEN  | jira whoami              | retriever.jira.key_missing    | Jira    |

  @integration @e2e @unit
  Scenario: An empty key counts as a missing key
    Given the environment variable "EXAMPLE_OUTLINE_API_KEY" is set to ""
    When I run "retriever --output json outline collections list"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.outline.key_missing" and retryable false
    And the stub Outline server received no requests

  @integration @e2e @unit
  Scenario: A missing Outline key names its variable and sends no request
    When I run "retriever --output json outline collections list"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.outline.key_missing" and retryable false
    And the JSON field "error.message" of the standard error is "EXAMPLE_OUTLINE_API_KEY is not set"
    And the stub Outline server received no requests

  @integration @e2e @unit
  Scenario Outline: A source key that is set but empty names its variable and sends no request
    Given the environment variable "<variable>" is set to ""
    When I run "retriever --output json <command>"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "<code>" and retryable false
    And the JSON field "error.message" of the standard error is "<variable> is not set"
    And the stub <kind> server received no requests

    Examples:
      | variable                | command            | code                          | kind    |
      | EXAMPLE_KIWI_API_KEY    | kiwi products list | retriever.kiwi.key_missing    | Kiwi    |
      | EXAMPLE_OUTLINE_API_KEY | outline collections list | retriever.outline.key_missing | Outline |

  @integration @e2e @unit
  Scenario: A missing Kiwi key names its variable and sends no request
    When I run "retriever --output json kiwi products list"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.kiwi.key_missing" and retryable false
    And the JSON field "error.message" of the standard error is "EXAMPLE_KIWI_API_KEY is not set"
    And the stub Kiwi server received no requests

  @integration @e2e @unit
  Scenario: The local env file supplies the key of a second Outline source
    Given the sources catalog file "two.yml" holds these sources:
      | id                  | kind    | tags | endpoint                      | credential_env              | identity_env |
      | example-outline     | outline | wiki | https://docs.example.test     | EXAMPLE_OUTLINE_API_KEY     |              |
      | example-outline-two | outline | wiki | https://docs.example.test | EXAMPLE_OUTLINE_TWO_API_KEY |              |
    And the env file ".env.local" holds these lines:
      """
      EXAMPLE_OUTLINE_TWO_API_KEY=synthetic-second-key
      """
    When I run "retriever --catalog two.yml outline collections list --source example-outline-two"
    Then the exit status is 0
    And every request to the stub Outline server carried the Authorization header "Bearer synthetic-second-key"

  @integration @e2e @unit
  Scenario: A missing key for a second Outline source names that source's own variable
    Given the sources catalog file "two.yml" holds these sources:
      | id                  | kind    | tags | endpoint                      | credential_env              | identity_env |
      | example-outline     | outline | wiki | https://docs.example.test     | EXAMPLE_OUTLINE_API_KEY     |              |
      | example-outline-two | outline | wiki | https://docs.example.test | EXAMPLE_OUTLINE_TWO_API_KEY |              |
    When I run "retriever --output json --catalog two.yml outline collections list --source example-outline-two"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.outline.key_missing" and retryable false
    And the JSON field "error.message" of the standard error is "EXAMPLE_OUTLINE_TWO_API_KEY is not set"
    And the stub Outline server received no requests

  @integration @e2e @unit
  Scenario: A malformed Jira e-mail address stops the command before any request and is never shown
    Given the environment variable "EXAMPLE_JIRA_API_TOKEN" is set to "synthetic-jira-token"
    And the environment variable "EXAMPLE_JIRA_EMAIL" is set to "<address>"
    When I run "retriever --output json jira whoami"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.jira.identity_invalid" and retryable false
    And the stub Jira server received no requests
    And the value "<address>" never appears in the standard output, the standard error, or the output folder

    Examples:
      | address             |
      | no-at-sign          |
      | two@@example.test   |

  @integration @e2e @unit
  Scenario: A missing Jira e-mail address stops the command before any request is sent
    Given the environment variable "EXAMPLE_JIRA_API_TOKEN" is set to "synthetic-jira-token"
    When I run "retriever --output json jira whoami"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.jira.key_missing" and retryable false
    And the JSON field "error.message" of the standard error is "EXAMPLE_JIRA_EMAIL is not set"
    And the stub Jira server received no requests

  @integration @e2e @unit
  Scenario: An empty Jira e-mail address counts as a missing e-mail address and names its variable
    Given the environment variable "EXAMPLE_JIRA_API_TOKEN" is set to "synthetic-jira-token"
    And the environment variable "EXAMPLE_JIRA_EMAIL" is set to ""
    When I run "retriever --output json jira whoami"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.jira.key_missing" and retryable false
    And the JSON field "error.message" of the standard error is "EXAMPLE_JIRA_EMAIL is not set"
    And the stub Jira server received no requests

  @integration @e2e @unit
  Scenario Outline: A missing or empty Jira token names the token variable, even when the e-mail address is empty too
    Given the environment variable "EXAMPLE_JIRA_API_TOKEN" is set to "<token>"
    And the environment variable "EXAMPLE_JIRA_EMAIL" is set to "<email>"
    When I run "retriever --output json jira whoami"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.jira.key_missing" and retryable false
    And the JSON field "error.message" of the standard error is "EXAMPLE_JIRA_API_TOKEN is not set"
    And the stub Jira server received no requests

    Examples:
      | token | email                       |
      |       | synthetic-user@example.test |
      |       |                             |

  @integration @e2e @unit
  Scenario: A Jira token that the source refuses gets the same text whichever value is wrong
    Given the environment variable "EXAMPLE_JIRA_API_TOKEN" is set to "synthetic-wrong-token"
    And the environment variable "EXAMPLE_JIRA_EMAIL" is set to "synthetic-user@example.test"
    And the stub Jira server answers every request with status 401
    When I run "retriever --output json jira whoami"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.jira.key_rejected" and retryable false
    And the stub Jira server received 1 request
    And I keep the standard error as "wrong token"
    Given the environment variable "EXAMPLE_JIRA_EMAIL" is set to "synthetic-other@example.test"
    When I run "retriever --output json jira whoami"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.jira.key_rejected" and retryable false
    And the stub Jira server received 2 requests
    And the standard error is the same as the one kept as "wrong token"

  @integration @e2e @unit
  Scenario: The Jira token and e-mail address are read from the env file in the current directory
    Given the env file ".env.local" holds these lines:
      """
      EXAMPLE_JIRA_API_TOKEN=synthetic-jira-token
      EXAMPLE_JIRA_EMAIL=synthetic-user@example.test
      """
    And the stub Jira server holds this configuration:
      """
      {"Email": "synthetic-user@example.test", "Token": "synthetic-jira-token", "Account": {"AccountID": "synthetic-account-id", "DisplayName": "Synthetic Person"}, "Projects": [], "Issues": [], "Comments": {}}
      """
    When I run "retriever jira whoami"
    Then the exit status is 0
    And every request to the stub Jira server carried the credentials of the synthetic e-mail address and token

  @integration @e2e @unit
  Scenario: The process environment wins over the env file for the Jira e-mail address
    Given the environment variable "EXAMPLE_JIRA_API_TOKEN" is set to "synthetic-jira-token"
    And the environment variable "EXAMPLE_JIRA_EMAIL" is set to "synthetic-user@example.test"
    And the env file ".env.local" holds these lines:
      """
      EXAMPLE_JIRA_EMAIL=synthetic-file-user@example.test
      """
    And the stub Jira server holds this configuration:
      """
      {"Email": "synthetic-user@example.test", "Token": "synthetic-jira-token", "Account": {"AccountID": "synthetic-account-id", "DisplayName": "Synthetic Person"}, "Projects": [], "Issues": [], "Comments": {}}
      """
    When I run "retriever jira whoami"
    Then the exit status is 0
    And every request to the stub Jira server carried the credentials of the synthetic e-mail address and token

  @integration @e2e @unit
  Scenario: The Jira credentials go only to the Jira endpoint and no other source receives a request
    Given the environment variable "EXAMPLE_JIRA_API_TOKEN" is set to "synthetic-jira-token"
    And the environment variable "EXAMPLE_JIRA_EMAIL" is set to "synthetic-user@example.test"
    And the stub Jira server holds this configuration:
      """
      {"Email": "synthetic-user@example.test", "Token": "synthetic-jira-token", "Account": {"AccountID": "synthetic-account-id", "DisplayName": "Synthetic Person"}, "Projects": [], "Issues": [], "Comments": {}}
      """
    When I run "retriever jira whoami"
    Then the exit status is 0
    And every request to the stub Jira server carried the credentials of the synthetic e-mail address and token
    And the stub Outline server received no requests
    And the stub Kiwi server received no requests

  @integration @e2e @unit
  Scenario: A key that the source refuses is reported with its own code and never printed
    Given the environment variable "EXAMPLE_KIWI_API_KEY" is set to "synthetic-kiwi-key"
    And the stub Kiwi server answers every request with status 401
    When I run "retriever --output json kiwi products list"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.kiwi.key_rejected" and retryable false
    And the value of "EXAMPLE_KIWI_API_KEY" never appears in the standard output, the standard error, or the output folder

  @integration @e2e @unit
  Scenario: An Outline key that the source refuses is reported with its own code
    Given the environment variable "EXAMPLE_OUTLINE_API_KEY" is set to "synthetic-outline-key"
    And the stub Outline server answers "collections.list" with status 401
    When I run "retriever --output json outline collections list"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.outline.key_rejected" and retryable false
    And the value of "EXAMPLE_OUTLINE_API_KEY" never appears in the standard output, the standard error, or the output folder

  @integration @e2e @unit
  Scenario Outline: A refused Jira request never prints the token or the e-mail address, even when the source echoes them
    Given the environment variable "EXAMPLE_JIRA_API_TOKEN" is set to "synthetic-jira-token"
    And the environment variable "EXAMPLE_JIRA_EMAIL" is set to "synthetic-user@example.test"
    And the stub Jira server answers every request with status <status> and echoes the token and the e-mail address
    When I run "retriever --output json jira whoami"
    Then the exit status is 2
    And the standard output is empty
    And the value of "EXAMPLE_JIRA_API_TOKEN" never appears in the standard output, the standard error, or the output folder
    And the value of "EXAMPLE_JIRA_EMAIL" never appears in the standard output, the standard error, or the output folder

    Examples:
      | status |
      | 401    |
      | 403    |
      | 500    |

  @integration @e2e @unit
  Scenario Outline: A successful Jira read never prints the token or the e-mail address
    Given the environment variable "EXAMPLE_JIRA_API_TOKEN" is set to "synthetic-jira-token"
    And the environment variable "EXAMPLE_JIRA_EMAIL" is set to "synthetic-user@example.test"
    And the stub Jira server holds this configuration:
      """
      {"Email": "synthetic-user@example.test", "Token": "synthetic-jira-token", "Account": {"AccountID": "synthetic-account-id", "DisplayName": "Synthetic Person"}, "Projects": [{"ID": "10001", "Key": "EXAMPLE", "Name": "Synthetic project"}], "Issues": [{"ID": "10010", "Key": "EXAMPLE-1", "Summary": "Synthetic issue", "Description": {"type": "doc", "version": 1, "content": [{"type": "paragraph", "content": [{"type": "text", "text": "Synthetic description"}]}]}}], "Comments": {}}
      """
    When I run "<command>"
    Then the exit status is 0
    And the value of "EXAMPLE_JIRA_API_TOKEN" never appears in the standard output, the standard error, or the output folder
    And the value of "EXAMPLE_JIRA_EMAIL" never appears in the standard output, the standard error, or the output folder

    Examples:
      | command                             |
      | retriever jira whoami               |
      | retriever --output json jira whoami |
      | retriever jira issues get EXAMPLE-1 |

  @integration @e2e @unit
  Scenario: A refused Jira export writes no file that holds the token or the e-mail address
    Given the sources catalog file "reviewed.yml" holds these sources:
      | id           | kind | tags    | endpoint                  | credential_env         | identity_env       |
      | example-jira | jira | tickets | https://jira.example.test | EXAMPLE_JIRA_API_TOKEN | EXAMPLE_JIRA_EMAIL |
    And the environment variable "EXAMPLE_JIRA_API_TOKEN" is set to "synthetic-jira-token"
    And the environment variable "EXAMPLE_JIRA_EMAIL" is set to "synthetic-user@example.test"
    And the stub Jira server answers every request with status 403 and echoes the token and the e-mail address
    When I run "retriever --catalog reviewed.yml jira export issue-EXAMPLE-1 --out out"
    Then the exit status is 2
    And the output folder "out" holds no files
    And the value of "EXAMPLE_JIRA_API_TOKEN" never appears in the standard output, the standard error, or the output folder
    And the value of "EXAMPLE_JIRA_EMAIL" never appears in the standard output, the standard error, or the output folder

  @integration @e2e @unit
  Scenario: A successful Jira export writes one file that holds neither the token nor the e-mail address
    Given the sources catalog file "reviewed.yml" holds these sources:
      | id           | kind | tags    | endpoint                  | credential_env         | identity_env       |
      | example-jira | jira | tickets | https://jira.example.test | EXAMPLE_JIRA_API_TOKEN | EXAMPLE_JIRA_EMAIL |
    And the environment variable "EXAMPLE_JIRA_API_TOKEN" is set to "synthetic-jira-token"
    And the environment variable "EXAMPLE_JIRA_EMAIL" is set to "synthetic-user@example.test"
    And the stub Jira server holds this configuration:
      """
      {"Email": "synthetic-user@example.test", "Token": "synthetic-jira-token", "Account": {"AccountID": "synthetic-account-id", "DisplayName": "Synthetic Person"}, "Projects": [{"ID": "10001", "Key": "EXAMPLE", "Name": "Synthetic project"}], "Issues": [{"ID": "10010", "Key": "EXAMPLE-1", "Summary": "Synthetic issue", "Description": {"type": "doc", "version": 1, "content": [{"type": "paragraph", "content": [{"type": "text", "text": "Synthetic description"}]}]}}], "Comments": {}}
      """
    When I run "retriever --catalog reviewed.yml jira export issue-EXAMPLE-1 --out out"
    Then the exit status is 0
    And the output folder "out" holds exactly 1 Markdown file
    And the value of "EXAMPLE_JIRA_API_TOKEN" never appears in the standard output, the standard error, or the output folder
    And the value of "EXAMPLE_JIRA_EMAIL" never appears in the standard output, the standard error, or the output folder

  @integration @e2e @unit
  Scenario: The served sources list never shows the Jira token or the e-mail address
    Given the environment variable "EXAMPLE_JIRA_API_TOKEN" is set to "synthetic-jira-token"
    And the environment variable "EXAMPLE_JIRA_EMAIL" is set to "synthetic-user@example.test"
    And retriever serve is running on the loopback address
    When retriever serve receives this GraphQL request:
      """
      query { sources { id kind tags } }
      """
    Then the HTTP status is 200
    And the GraphQL response has no errors
    And the GraphQL response field "data.sources.0.id" is "example-jira"
    And the GraphQL response does not contain "synthetic-jira-token"
    And the GraphQL response does not contain "synthetic-user@example.test"

  @integration @e2e @unit
  Scenario: The Kiwi API key is read from the env file in the current directory
    Given the env file ".env.local" holds these lines:
      """
      EXAMPLE_KIWI_API_KEY=synthetic-kiwi-key
      """
    When I run "retriever kiwi products list"
    Then the exit status is 0
    And every request to the stub Kiwi server carried the Authorization header "Bearer synthetic-kiwi-key"

  @integration @e2e @unit
  Scenario: A Kiwi key is never sent in the body or the path of a request
    Given the environment variable "EXAMPLE_KIWI_API_KEY" is set to "synthetic-kiwi-key"
    When I run "retriever kiwi products list"
    Then the exit status is 0
    And no request to the stub Kiwi server carried the value of "EXAMPLE_KIWI_API_KEY" in its body or its path

  @integration @e2e @unit
  Scenario Outline: A successful Kiwi run never prints the key
    Given the environment variable "EXAMPLE_KIWI_API_KEY" is set to "synthetic-kiwi-key"
    And the stub Kiwi server holds this dataset:
      """
      {"Products": [{"ID": 3, "Name": "Synthetic product"}], "Plans": [], "Cases": [], "Categories": [], "Priorities": [], "Statuses": []}
      """
    When I run "<command>"
    Then the exit status is 0
    And the value of "EXAMPLE_KIWI_API_KEY" never appears in the standard output, the standard error, or the output folder

    Examples:
      | command                              |
      | retriever kiwi products list         |
      | retriever --output json kiwi products list |

  @integration @e2e @unit
  Scenario: A Kiwi write never prints the key
    Given the sources catalog file "reviewed.yml" holds these sources:
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
    When I run "retriever --catalog reviewed.yml kiwi test-cases create Synthetic-case --product product-3 --category Functional --priority High --source example-kiwi"
    Then the exit status is 0
    And the stub Kiwi server received 1 request to "TestCase.create"
    And the value of "EXAMPLE_KIWI_API_KEY" never appears in the standard output, the standard error, or the output folder

  @integration @e2e @unit
  Scenario Outline: A failing Kiwi response never prints the key, even when the response echoes it
    Given the environment variable "EXAMPLE_KIWI_API_KEY" is set to "synthetic-kiwi-key"
    And the stub Kiwi server answers every request with status <status> and echoes the key
    When I run "retriever --output json kiwi products list"
    Then the exit status is 2
    And the standard output is empty
    And the value of "EXAMPLE_KIWI_API_KEY" never appears in the standard output, the standard error, or the output folder

    Examples:
      | status |
      | 403    |
      | 404    |
      | 500    |

  @integration @e2e @unit
  Scenario: Kiwi rate-limit exhaustion never prints the key
    Given the environment variable "EXAMPLE_KIWI_API_KEY" is set to "synthetic-kiwi-key"
    And the stub Kiwi server answers every request with status 429 and Retry-After "0"
    When I run "retriever --output json kiwi products list"
    Then the exit status is 124
    And the standard error is a JSON error with the code "retriever.kiwi.rate_limited" and retryable true
    And the stub Kiwi server received 4 requests
    And the value of "EXAMPLE_KIWI_API_KEY" never appears in the standard output, the standard error, or the output folder

  @integration @e2e @unit
  Scenario: A JSON-RPC error that echoes the key in its message never prints the key
    Given the environment variable "EXAMPLE_KIWI_API_KEY" is set to "synthetic-kiwi-key"
    And the stub Kiwi server answers "TestCase.filter" with a JSON-RPC error with code -32000 and message "synthetic-kiwi-key"
    When I run "retriever --output json kiwi documents get case-40"
    Then the exit status is 2
    And the standard output is empty
    And the value of "EXAMPLE_KIWI_API_KEY" never appears in the standard output, the standard error, or the output folder

  @integration @e2e @unit
  Scenario: A malformed Kiwi response never prints the key, even when the response echoes it
    Given the environment variable "EXAMPLE_KIWI_API_KEY" is set to "synthetic-kiwi-key"
    And the stub Kiwi server answers every request with a body that is not JSON and echoes the key
    When I run "retriever --output json kiwi products list"
    Then the exit status is 2
    And the standard output is empty
    And the value of "EXAMPLE_KIWI_API_KEY" never appears in the standard output, the standard error, or the output folder

  @integration @e2e @unit
  Scenario: A Kiwi source that cannot be reached is reported as unreachable and never prints the key
    Given the environment variable "EXAMPLE_KIWI_API_KEY" is set to "synthetic-kiwi-key"
    And the stub Kiwi server is not listening
    When I run "retriever --output json kiwi products list"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.kiwi.unreachable" and retryable true
    And the value of "EXAMPLE_KIWI_API_KEY" never appears in the standard output, the standard error, or the output folder

  @integration @e2e @unit
  Scenario Outline: A rejected Kiwi key is reported by every Kiwi read and never printed
    Given the environment variable "EXAMPLE_KIWI_API_KEY" is set to "synthetic-kiwi-key"
    And the stub Kiwi server answers every request with status 401 and echoes the key
    When I run "<command>"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.kiwi.key_rejected" and retryable false
    And the value of "EXAMPLE_KIWI_API_KEY" never appears in the standard output, the standard error, or the output folder

    Examples:
      | command                                          |
      | retriever --output json kiwi products list       |
      | retriever --output json kiwi documents get case-40 |

  @integration @e2e @unit
  Scenario Outline: A failed Kiwi write never prints the key, in text or in the JSON error body
    Given the sources catalog file "reviewed.yml" holds these sources:
      | id           | kind | tags  | endpoint                  | credential_env       | identity_env |
      | example-kiwi | kiwi | tests | https://kiwi.example.test | EXAMPLE_KIWI_API_KEY |              |
    And the environment variable "EXAMPLE_KIWI_API_KEY" is set to "synthetic-kiwi-key"
    And the stub Kiwi server answers every request with status 500 and echoes the key
    When I run "<command>"
    Then the exit status is 2
    And the standard output is empty
    And the value of "EXAMPLE_KIWI_API_KEY" never appears in the standard output, the standard error, or the output folder

    Examples:
      | command                                                                                                                                              |
      | retriever --catalog reviewed.yml kiwi test-cases create Synthetic-case --product product-3 --category Functional --priority High --source example-kiwi |
      | retriever --output json --catalog reviewed.yml kiwi test-cases create Synthetic-case --product product-3 --category Functional --priority High --source example-kiwi |

  @integration @e2e @unit
  Scenario: A key is never sent in the body or the URL of a request
    Given the environment variable "EXAMPLE_OUTLINE_API_KEY" is set to "synthetic-outline-key"
    When I run "retriever outline collections list"
    Then the exit status is 0
    And no request to the stub Outline server carried the value of "EXAMPLE_OUTLINE_API_KEY" in its body or its URL

  @integration @e2e @unit
  Scenario: A successful run never prints the key in standard output, standard error, or files
    Given the environment variable "EXAMPLE_OUTLINE_API_KEY" is set to "synthetic-outline-key"
    When I run "retriever outline collections list"
    Then the exit status is 0
    And the value of "EXAMPLE_OUTLINE_API_KEY" never appears in the standard output, the standard error, or the output folder
