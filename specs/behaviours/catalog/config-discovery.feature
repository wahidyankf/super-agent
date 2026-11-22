# Where retriever finds the sources catalog and the local env file. For each file the first rule that applies wins: the
# flag, the variable, the file of the same name in the working directory, then the user configuration folder. The
# folders above the working directory are never searched. That is deliberate, so a command reads the same files from
# any subfolder only when the files are named or placed where the rules look. With a profiles folder, one profile
# is the only folder that a run reads.
Feature: Configuration discovery
  As a person who runs retriever from different folders
  I want the catalog and the env file found by one fixed order
  So that no file from a parent folder is used by accident, and every rule can be checked

  # Behaviour: CLI-001
  # Behaviour: CLI-002
  # Behaviour: CLI-003
  Scenario: The catalog flag wins over the catalog variable and over the file in the working directory
    Given the environment variable "RETRIEVER_CATALOG" is set to "missing.yml"
    And the sources catalog holds these sources:
      | id              | kind    | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    And the sources catalog file "reviewed.yml" holds these sources:
      | id        | kind | tags  | endpoint                  | credential_env       | identity_env |
      | only-kiwi | kiwi | tests | https://kiwi.example.test | EXAMPLE_KIWI_API_KEY |              |
    When I run "retriever --catalog reviewed.yml sources list"
    Then the exit status is 0
    And the standard output contains "only-kiwi"
    And the standard output does not contain "example-outline"

  # Behaviour: CLI-004
  # Behaviour: CLI-005
  Scenario: The catalog variable wins over the file in the working directory
    Given the environment variable "RETRIEVER_CATALOG" is set to "reviewed.yml"
    And the sources catalog holds these sources:
      | id              | kind    | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    And the sources catalog file "reviewed.yml" holds these sources:
      | id        | kind | tags  | endpoint                  | credential_env       | identity_env |
      | only-kiwi | kiwi | tests | https://kiwi.example.test | EXAMPLE_KIWI_API_KEY |              |
    When I run "retriever sources list"
    Then the exit status is 0
    And the standard output contains "only-kiwi"
    And the standard output does not contain "example-outline"

  # Behaviour: CLI-006
  Scenario: The user configuration folder holds the catalog when the working directory has none
    Given the sources catalog in the user configuration folder holds these sources:
      | id        | kind | tags  | endpoint                  | credential_env       | identity_env |
      | user-kiwi | kiwi | tests | https://kiwi.example.test | EXAMPLE_KIWI_API_KEY |              |
    When I run "retriever sources list"
    Then the exit status is 0
    And the standard output contains "user-kiwi"

  # Behaviour: CLI-007
  Scenario: A catalog in a parent folder is not read from a subfolder
    Given the sources catalog holds these sources:
      | id              | kind    | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    And the working directory is the folder "nested/deeper"
    When I run "retriever --output json sources list"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.sources.catalog_missing" and retryable false

  # Behaviour: CLI-008
  Scenario: A relative catalog flag is resolved against the working directory
    Given the sources catalog holds these sources:
      | id              | kind    | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    And the working directory is the folder "nested"
    When I run "retriever --catalog ../sources.yml sources list"
    Then the exit status is 0
    And the standard output contains "example-outline"

  # Behaviour: CLI-009
  Scenario: An env file in a parent folder is not read from a subfolder
    Given the sources catalog holds these sources:
      | id              | kind    | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    And the env file ".env.local" holds these lines:
      """
      EXAMPLE_OUTLINE_API_KEY=synthetic-outline-key
      """
    And the stub Outline server holds these documents:
      | id    | collection | parent | title | markdown           |
      | d0c1d0c1-0000-4000-8000-000000000001 | 1c1c1c1c-0000-4000-8000-000000000001 |        | Setup | Synthetic body one |
    And the working directory is the folder "nested"
    When I run "retriever --output json --catalog ../sources.yml outline documents get d0c1d0c1-0000-4000-8000-000000000001"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.outline.key_missing" and retryable false
    And the stub Outline server received no requests

  # Behaviour: CLI-010
  Scenario: The env file in the working directory supplies the key when no flag or variable names one
    Given the sources catalog holds these sources:
      | id              | kind    | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    And the env file ".env.local" in the folder "nested" holds these lines:
      """
      EXAMPLE_OUTLINE_API_KEY=synthetic-outline-key
      """
    And the stub Outline server holds these documents:
      | id    | collection | parent | title | markdown           |
      | d0c1d0c1-0000-4000-8000-000000000001 | 1c1c1c1c-0000-4000-8000-000000000001 |        | Setup | Synthetic body one |
    And the working directory is the folder "nested"
    When I run "retriever --catalog ../sources.yml outline documents get d0c1d0c1-0000-4000-8000-000000000001"
    Then the exit status is 0
    And the standard output contains "Synthetic body one"

  # Behaviour: CLI-011
  Scenario: The user configuration folder supplies the env file when the working directory has none
    Given the sources catalog holds these sources:
      | id              | kind    | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    And the env file ".env.local" in the folder "config/retriever" holds these lines:
      """
      EXAMPLE_OUTLINE_API_KEY=synthetic-outline-key
      """
    And the stub Outline server holds these documents:
      | id    | collection | parent | title | markdown           |
      | d0c1d0c1-0000-4000-8000-000000000001 | 1c1c1c1c-0000-4000-8000-000000000001 |        | Setup | Synthetic body one |
    When I run "retriever outline documents get d0c1d0c1-0000-4000-8000-000000000001"
    Then the exit status is 0
    And the standard output contains "Synthetic body one"

  # Behaviour: CLI-012
  Scenario: The env file flag wins over the env file variable
    Given the environment variable "RETRIEVER_ENV_FILE" is set to "missing.env"
    And the sources catalog holds these sources:
      | id              | kind    | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    And the env file "outline.env" holds these lines:
      """
      EXAMPLE_OUTLINE_API_KEY=synthetic-outline-key
      """
    And the stub Outline server holds these documents:
      | id    | collection | parent | title | markdown           |
      | d0c1d0c1-0000-4000-8000-000000000001 | 1c1c1c1c-0000-4000-8000-000000000001 |        | Setup | Synthetic body one |
    When I run "retriever --env-file outline.env outline documents get d0c1d0c1-0000-4000-8000-000000000001"
    Then the exit status is 0
    And the standard output contains "Synthetic body one"

  # Behaviour: CLI-013
  Scenario: The env file variable names the env file when no flag is given
    Given the environment variable "RETRIEVER_ENV_FILE" is set to "outline.env"
    And the sources catalog holds these sources:
      | id              | kind    | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    And the env file "outline.env" holds these lines:
      """
      EXAMPLE_OUTLINE_API_KEY=synthetic-outline-key
      """
    And the stub Outline server holds these documents:
      | id    | collection | parent | title | markdown           |
      | d0c1d0c1-0000-4000-8000-000000000001 | 1c1c1c1c-0000-4000-8000-000000000001 |        | Setup | Synthetic body one |
    When I run "retriever outline documents get d0c1d0c1-0000-4000-8000-000000000001"
    Then the exit status is 0
    And the standard output contains "Synthetic body one"

  # Behaviour: CLI-014
  Scenario: An empty catalog variable counts as unset
    Given the environment variable "RETRIEVER_CATALOG" is set to ""
    And the sources catalog holds these sources:
      | id              | kind    | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    When I run "retriever sources list"
    Then the exit status is 0
    And the standard output contains "example-outline"

  # Behaviour: CLI-015
  Scenario: An empty env file variable counts as unset
    Given the environment variable "RETRIEVER_ENV_FILE" is set to ""
    And the sources catalog holds these sources:
      | id              | kind    | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    And the env file ".env.local" holds these lines:
      """
      EXAMPLE_OUTLINE_API_KEY=synthetic-outline-key
      """
    And the stub Outline server holds these documents:
      | id    | collection | parent | title | markdown           |
      | d0c1d0c1-0000-4000-8000-000000000001 | 1c1c1c1c-0000-4000-8000-000000000001 |        | Setup | Synthetic body one |
    When I run "retriever outline documents get d0c1d0c1-0000-4000-8000-000000000001"
    Then the exit status is 0
    And the standard output contains "Synthetic body one"

  # Behaviour: CLI-016
  Scenario: The profile flag selects the profile folder and ignores the working directory
    Given the sources catalog holds these sources:
      | id              | kind    | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    And the profile "personal" holds these sources:
      | id           | kind | tags  | endpoint                  | credential_env         | identity_env |
      | example-kiwi | kiwi | tests | https://kiwi.example.test | PERSONAL_KIWI_API_KEY  |              |
    When I run "retriever --profile personal sources list"
    Then the exit status is 0
    And the standard output contains "example-kiwi"
    And the standard output does not contain "example-outline"

  # Behaviour: CLI-017
  Scenario: The profile variable selects the profile when no flag is given
    Given the environment variable "RETRIEVER_PROFILE" is set to "personal"
    And the sources catalog holds these sources:
      | id              | kind    | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    And the profile "personal" holds these sources:
      | id           | kind | tags  | endpoint                  | credential_env        | identity_env |
      | example-kiwi | kiwi | tests | https://kiwi.example.test | PERSONAL_KIWI_API_KEY |              |
    When I run "retriever sources list"
    Then the exit status is 0
    And the standard output contains "example-kiwi"
    And the standard output does not contain "example-outline"

  # Behaviour: CLI-018
  Scenario: The profile flag wins over the profile variable
    Given the environment variable "RETRIEVER_PROFILE" is set to "work"
    And the profile "personal" holds these sources:
      | id           | kind | tags  | endpoint                  | credential_env        | identity_env |
      | example-kiwi | kiwi | tests | https://kiwi.example.test | PERSONAL_KIWI_API_KEY |              |
    And the profile "work" holds these sources:
      | id           | kind | tags    | endpoint                  | credential_env     | identity_env |
      | work-jira    | jira | tickets | https://jira.example.test | WORK_JIRA_API_TOKEN | WORK_JIRA_EMAIL |
    When I run "retriever --profile personal sources list"
    Then the exit status is 0
    And the standard output contains "example-kiwi"
    And the standard output does not contain "work-jira"

  # Behaviour: CLI-019
  Scenario: A profiles folder with no chosen profile refuses every source command
    Given the sources catalog holds these sources:
      | id              | kind    | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    And the profile "personal" holds these sources:
      | id           | kind | tags  | endpoint                  | credential_env        | identity_env |
      | example-kiwi | kiwi | tests | https://kiwi.example.test | PERSONAL_KIWI_API_KEY |              |
    When I run "retriever --output json sources list"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.usage.profile_required" and retryable false

  # Behaviour: CLI-020
  Scenario: A profile flag with an explicit catalog path is a usage error
    Given the profile "personal" holds these sources:
      | id           | kind | tags  | endpoint                  | credential_env        | identity_env |
      | example-kiwi | kiwi | tests | https://kiwi.example.test | PERSONAL_KIWI_API_KEY |              |
    And the sources catalog file "reviewed.yml" holds these sources:
      | id        | kind | tags  | endpoint                  | credential_env       | identity_env |
      | only-kiwi | kiwi | tests | https://kiwi.example.test | EXAMPLE_KIWI_API_KEY |              |
    When I run "retriever --output json --profile personal --catalog reviewed.yml sources list"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.usage.profile_invalid" and retryable false

  # Behaviour: CLI-021
  Scenario: A profile name that escapes the profiles folder is refused
    Given the profile "personal" holds these sources:
      | id           | kind | tags  | endpoint                  | credential_env        | identity_env |
      | example-kiwi | kiwi | tests | https://kiwi.example.test | PERSONAL_KIWI_API_KEY |              |
    When I run "retriever --output json --profile ../other sources list"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.usage.profile_invalid" and retryable false

  # Behaviour: CLI-023
  Scenario: A profile folder that is a symbolic link is refused
    Given the profile "personal" is a symbolic link to a folder that holds these sources:
      | id           | kind | tags  | endpoint                  | credential_env        | identity_env |
      | example-kiwi | kiwi | tests | https://kiwi.example.test | PERSONAL_KIWI_API_KEY |              |
    When I run "retriever --output json --profile personal sources list"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.usage.profile_invalid" and retryable false
    And the standard error does not name the profile folder

  # Behaviour: CLI-022
  Scenario: The user configuration folder comes from the home folder when XDG_CONFIG_HOME is empty
    Given the environment variable "XDG_CONFIG_HOME" is set to ""
    And the sources catalog in the home configuration folder holds these sources:
      | id        | kind | tags  | endpoint                  | credential_env       | identity_env |
      | home-kiwi | kiwi | tests | https://kiwi.example.test | EXAMPLE_KIWI_API_KEY |              |
    When I run "retriever sources list"
    Then the exit status is 0
    And the standard output contains "home-kiwi"
