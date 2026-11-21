# Connector specs: a declarative spec file adds a source kind, its commands, and its MCP tools, and a bad spec stops start-up.
# Synthetic hosts only (wiki.example.test, example.test); the key is named by variable and never written in a spec or a catalog.
Feature: Connector specs
  As a person who describes a service in a spec file
  I want retriever to run that spec through one generic engine
  So that a new service needs no Go change, and a refused spec never reaches a source

  # Behaviour: CLI-087
  # Exemption(e2e): the built binary refuses private-network hosts and untrusted certificates, so no loopback spec source can be reached at the end-to-end level; alternative-proof: retriever-cli:test:integration
  @e2e-exempt
  Scenario: A spec command prints the mapped document and exits 0
    Given the connectors folder "connectors" holds the example wiki spec as "example-wiki.yml"
    And the sources catalog holds these sources:
      | id   | kind         | tags | endpoint                  | credential_env       | identity_env |
      | wiki | example-wiki | docs | https://wiki.example.test | EXAMPLE_WIKI_API_KEY |              |
    And the environment variable "EXAMPLE_WIKI_API_KEY" is set to "example-key"
    When I run "retriever --connectors-dir connectors example-wiki docs list"
    Then the exit status is 0
    And the standard output contains "doc-1"
    And the spec source received 1 request

  # Behaviour: CLI-088
  Scenario: A spec operation with the CLI off and the MCP on has no command and keeps its tool
    Given the connectors folder "connectors" holds the example wiki spec with the CLI off and MCP on as "example-wiki.yml"
    And the sources catalog holds these sources:
      | id   | kind         | tags | endpoint                  | credential_env       | identity_env |
      | wiki | example-wiki | docs | https://wiki.example.test | EXAMPLE_WIKI_API_KEY |              |
    When I run "retriever --connectors-dir connectors example-wiki docs list"
    Then the exit status is 2
    And the standard output is empty
    Given an MCP client starts "retriever --connectors-dir connectors mcp"
    When the client lists the tools
    Then the tool list has the tool "example_wiki_docs_list"

  # Behaviour: CLI-089
  Scenario: A spec operation with the CLI on and the MCP off has a command and no tool
    Given the connectors folder "connectors" holds the example wiki spec with the CLI on and MCP off as "example-wiki.yml"
    And the sources catalog holds these sources:
      | id   | kind         | tags | endpoint                  | credential_env       | identity_env |
      | wiki | example-wiki | docs | https://wiki.example.test | EXAMPLE_WIKI_API_KEY |              |
    When I run "retriever --connectors-dir connectors example-wiki docs list --help"
    Then the exit status is 0
    And the standard output contains "List documents."
    Given an MCP client starts "retriever --connectors-dir connectors mcp"
    When the client lists the tools
    Then the tool list has no tool named "example_wiki_docs_list"

  # Threat C2: a spec whose bytes differ from the catalog pin stops the load before any request.
  # Behaviour: CLI-090
  Scenario: A catalog pin that does not match the spec file fails with the pin code and sends no request
    Given the connectors folder "connectors" holds the example wiki spec as "example-wiki.yml"
    And the sources catalog file "sources.yml" holds this text:
      """
      version: 1
      sources:
        - id: wiki
          kind: example-wiki
          endpoint: https://wiki.example.test
          credential_env: EXAMPLE_WIKI_API_KEY
          spec_sha256: 0000000000000000000000000000000000000000000000000000000000000000
      """
    And the environment variable "EXAMPLE_WIKI_API_KEY" is set to "example-key"
    When I run "retriever --output json --connectors-dir connectors example-wiki docs list"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.connector.spec_pin_mismatch" and retryable false
    And the spec source received no requests

  # Behaviour: CLI-091
  Scenario: A spec that declares a header it may not set stops start-up with its file and field path and no value
    Given the connectors folder "connectors" holds the example wiki spec with an Authorization header as "example-wiki.yml"
    When I run "retriever --output json --connectors-dir connectors sources list"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.connector.spec_invalid" and retryable false
    And the standard error contains "example-wiki.yml"
    And the standard error contains "/operations/0/headers/Authorization"
    And the value "example-token-value" never appears in the standard output, the standard error, or the output folder

  # Behaviour: CLI-092
  Scenario Outline: A spec kind that names a built-in kind or a command word stops start-up
    Given the connectors folder "connectors" holds the example wiki spec as "bad.yml" with the kind "<kind>"
    When I run "retriever --output json --connectors-dir connectors sources list"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.connector.spec_invalid" and retryable false
    And the standard error contains "bad.yml"
    And the standard error contains "/kind"

    Examples:
      | kind      |
      | outline   |
      | retriever |
      | serve     |

  # Behaviour: CLI-093
  Scenario: A second spec with the kind of the first stops start-up and names the second file
    Given the connectors folder "connectors" holds the example wiki spec as "a-first.yml"
    And the connectors folder "connectors" holds the example wiki spec as "b-second.yml"
    When I run "retriever --output json --connectors-dir connectors sources list"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.connector.spec_invalid" and retryable false
    And the standard error contains "b-second.yml"
    And the standard error contains "/kind"

  # Behaviour: CLI-094
  Scenario: A loaded spec leaves the sources of the built-in kinds listed as they were
    Given the sources catalog holds these sources:
      | id              | kind    | tags      | endpoint                  | credential_env          | identity_env       |
      | example-outline | outline | wiki,docs | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |                    |
      | example-kiwi    | kiwi    | tests     | https://kiwi.example.test | EXAMPLE_KIWI_API_KEY    |                    |
      | example-jira    | jira    | tickets   | https://jira.example.test | EXAMPLE_JIRA_API_TOKEN  | EXAMPLE_JIRA_EMAIL |
    And the connectors folder "connectors" holds the example wiki spec as "example-wiki.yml"
    When I run "retriever --connectors-dir connectors sources list"
    Then the exit status is 0
    And the standard output contains "example-outline"
    And the standard output contains "example-kiwi"
    And the standard output contains "example-jira"

  # Behaviour: CLI-095
  Scenario: A kind filter with an unknown kind names only the built-in kinds when a spec is loaded
    Given the sources catalog holds these sources:
      | id              | kind    | tags      | endpoint                  | credential_env          | identity_env       |
      | example-outline | outline | wiki,docs | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |                    |
      | example-kiwi    | kiwi    | tests     | https://kiwi.example.test | EXAMPLE_KIWI_API_KEY    |                    |
      | example-jira    | jira    | tickets   | https://jira.example.test | EXAMPLE_JIRA_API_TOKEN  | EXAMPLE_JIRA_EMAIL |
    And the connectors folder "connectors" holds the example wiki spec as "example-wiki.yml"
    When I run "retriever --output json --connectors-dir connectors sources list --kind bogus"
    Then the exit status is 2
    And the standard error contains "known: outline, kiwi, jira"

  # Behaviour: CLI-096
  Scenario: An explicit connectors folder that does not exist exits 2 with its code and names no path
    Given the sources catalog holds these sources:
      | id   | kind    | tags | endpoint                  | credential_env          | identity_env |
      | wiki | outline | docs | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    When I run "retriever --output json --connectors-dir connectors-missing sources list"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.connector.folder_unavailable" and retryable false
    And the standard error does not contain "connectors-missing"

  # Behaviour: CLI-097
  Scenario: A default connectors folder that does not exist is not an error
    Given the sources catalog holds these sources:
      | id   | kind    | tags | endpoint                  | credential_env          | identity_env |
      | docs | outline | docs | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    When I run "retriever sources list"
    Then the exit status is 0
    And the standard output contains "docs"

  # A converter is refused before any request is built, so this run needs no stub and no network at any level.
  # Behaviour: CLI-098
  Scenario: A spec operation that needs a Markdown converter fails closed at run time and sends no request
    Given the connectors folder "connectors" holds the example wiki spec with a Markdown converter as "example-wiki.yml"
    And the sources catalog holds these sources:
      | id   | kind         | tags | endpoint                  | credential_env       | identity_env |
      | wiki | example-wiki | docs | https://wiki.example.test | EXAMPLE_WIKI_API_KEY |              |
    And the environment variable "EXAMPLE_WIKI_API_KEY" is set to "example-key"
    When I run "retriever --output json --connectors-dir connectors example-wiki docs get doc-1"
    Then the exit status is 2
    And the standard error is a JSON error with the code "retriever.example-wiki.unsupported_operation" and retryable false
    And the spec source received no requests

  # Behaviour: CLI-099
  # Exemption(e2e): the built binary refuses private-network hosts and untrusted certificates, so a POST read cannot reach a loopback stub at the end-to-end level; alternative-proof: retriever-cli:test:integration
  @e2e-exempt
  Scenario: A spec read sent over POST makes one request whose body is its declared slot
    Given the connectors folder "connectors" holds the example wiki spec with a POST read as "example-wiki.yml"
    And the sources catalog holds these sources:
      | id   | kind         | tags | endpoint                  | credential_env       | identity_env |
      | wiki | example-wiki | docs | https://wiki.example.test | EXAMPLE_WIKI_API_KEY |              |
    And the environment variable "EXAMPLE_WIKI_API_KEY" is set to "example-key"
    When I run "retriever --connectors-dir connectors example-wiki docs search draft"
    Then the exit status is 0
    And the standard output contains "doc-1"
    And the spec source received 1 request
    And the spec source's one request was a POST with the JSON body:
      """
      {"text":"draft"}
      """

  # Behaviour: CLI-100
  Scenario Outline: Only yml and yaml files in the connectors folder are read as connector specs
    Given the sources catalog holds these sources:
      | id   | kind         | tags | endpoint                  | credential_env       | identity_env |
      | wiki | example-wiki | docs | https://wiki.example.test | EXAMPLE_WIKI_API_KEY |              |
    And the connectors folder "connectors" holds the example wiki spec as "<file>"
    When I run "retriever --connectors-dir connectors sources list"
    Then the exit status is <status>

    Examples:
      | file              | status |
      | example-wiki.yml  | 0      |
      | example-wiki.yaml | 0      |
      | example-wiki.txt  | 2      |
