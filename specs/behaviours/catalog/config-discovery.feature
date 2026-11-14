# Where retriever finds the sources catalog and the local env file. For each file the first rule that applies wins: the
# flag, the variable, the file of the same name in the working directory, then the user configuration folder. The
# folders above the working directory are never searched. That is deliberate, so a command reads the same files from
# any subfolder only when the files are named or placed where the rules look.
Feature: Configuration discovery
  As a person who runs retriever from different folders
  I want the catalog and the env file found by one fixed order
  So that no file from a parent folder is used by accident, and every rule can be checked

  @unit @integration @e2e
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

  @unit @integration @e2e
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

  @unit @integration @e2e
  Scenario: The user configuration folder holds the catalog when the working directory has none
    Given the sources catalog in the user configuration folder holds these sources:
      | id        | kind | tags  | endpoint                  | credential_env       | identity_env |
      | user-kiwi | kiwi | tests | https://kiwi.example.test | EXAMPLE_KIWI_API_KEY |              |
    When I run "retriever sources list"
    Then the exit status is 0
    And the standard output contains "user-kiwi"

  @unit @integration @e2e
  Scenario: A catalog in a parent folder is not read from a subfolder
    Given the sources catalog holds these sources:
      | id              | kind    | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    And the working directory is the folder "nested/deeper"
    When I run "retriever --output json sources list"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.sources.catalog_missing" and retryable false

  @unit @integration @e2e
  Scenario: A relative catalog flag is resolved against the working directory
    Given the sources catalog holds these sources:
      | id              | kind    | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    And the working directory is the folder "nested"
    When I run "retriever --catalog ../sources.yml sources list"
    Then the exit status is 0
    And the standard output contains "example-outline"

  @unit @integration @e2e
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

  @unit @integration @e2e
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

  @unit @integration @e2e
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

  @unit @integration @e2e
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
