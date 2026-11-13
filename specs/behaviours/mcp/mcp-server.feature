# The MCP server started by `retriever mcp`. It speaks MCP over stdio, offers tools up to the access ceiling named by
# --max-access, and offers only the resources named by --tools. Tool results are bounded and redacted. These scenarios
# drive the built binary as a process, so they run at the end-to-end level only.
Feature: MCP tool server
  As an agent harness that reads and changes Outline through one protocol
  I want the server to offer only the tools my access ceiling allows, with bounded and redacted results
  So that an agent cannot reach a write, a critical change, or a secret through a tool it was never given

  Background:
    Given the environment variable "EXAMPLE_OUTLINE_API_KEY" is set to "synthetic-outline-key"
    And the sources catalog holds these sources:
      | id              | kind    | access | tags | endpoint                  | credential_env          | identity_env |
      | example-outline | outline | admin  | wiki | https://docs.example.test | EXAMPLE_OUTLINE_API_KEY |              |
    And the stub Outline server holds these collections:
      | id | name      |
      | 1c1c1c1c-0000-4000-8000-000000000001 | Synthetic |
    And the stub Outline server holds these documents:
      | id    | collection | parent | title | markdown           |
      | d0c1d0c1-0000-4000-8000-000000000001 | 1c1c1c1c-0000-4000-8000-000000000001         |        | Setup | Synthetic body one |

  @e2e
  Scenario: The tool list always offers list_sources, even when --tools names other resources
    Given an MCP client starts "retriever mcp --max-access read --tools comments"
    When the client lists the tools
    Then the tool list has the tool "list_sources"
    And the tool list has the tool "list_comments"
    And the tool list has no tool named "list_collections"

  @e2e
  Scenario: At the default read ceiling the tool list has no write tool
    Given an MCP client starts "retriever mcp"
    When the client lists the tools
    Then the tool list has the tool "list_documents"
    And the tool list has no tool named "create_document"

  @e2e
  Scenario: At the write ceiling the tool list offers the write tools but not the destructive ones
    Given an MCP client starts "retriever mcp --max-access write"
    When the client lists the tools
    Then the tool list has the tool "create_document"
    And the tool list has no tool named "delete_document"

  @e2e
  Scenario: At the admin ceiling the destructive tools are offered and marked destructive
    Given an MCP client starts "retriever mcp --max-access admin"
    When the client lists the tools
    Then the tool list has the tool "delete_document"
    And the tool "delete_document" is marked destructive
    And the tool "list_documents" is marked read-only

  @e2e
  Scenario: At the admin ceiling a critical user deletion has no MCP tool
    Given an MCP client starts "retriever mcp --max-access admin"
    When the client lists the tools
    Then the tool list has no tool named "delete_user"

  @e2e
  Scenario: A write tool called above the read ceiling is refused and sends no request
    Given an MCP client starts "retriever mcp --max-access read"
    When the client calls the tool "create_document" with these arguments:
      """
      {"title": "Synthetic-note", "text": "Synthetic"}
      """
    Then the tool result is an error
    And the stub Outline server received no requests

  @e2e
  Scenario: A read tool returns its result as text
    Given an MCP client starts "retriever mcp --max-access read"
    When the client calls the tool "search_documents" with these arguments:
      """
      {"query": "Synthetic"}
      """
    Then the tool result is not an error
    And the tool result text contains "d0c1d0c1-0000-4000-8000-000000000001"

  @e2e
  Scenario: The source list tool omits each source's endpoint and credential name
    Given an MCP client starts "retriever mcp --max-access read"
    When the client calls the tool "list_sources" with no arguments
    Then the tool result is not an error
    And the tool result text contains "example-outline"
    And the tool result text does not contain "docs.example.test"
    And the tool result text does not contain "EXAMPLE_OUTLINE_API_KEY"

  @e2e
  Scenario: A hostile title with a terminal escape never reaches a tool result as a control byte
    Given the stub Outline server answers "documents.info" with the body:
      """
      {"data": {"id": "d0c1d0c1-0000-4000-8000-000000000001", "title": "\u001b[31mRed", "text": "Synthetic body", "url": "/doc/d0c1d0c1-0000-4000-8000-000000000001", "collectionId": "1c1c1c1c-0000-4000-8000-000000000001", "revision": 1}}
      """
    And an MCP client starts "retriever mcp --max-access read"
    When the client calls the tool "get_document" with these arguments:
      """
      {"documentId": "d0c1d0c1-0000-4000-8000-000000000001"}
      """
    Then the tool result is not an error
    And the tool result text contains no control characters except line breaks

  @e2e
  Scenario: An oversized reply becomes a bounded tool error rather than a large tool result
    Given the stub Outline server answers "documents.info" with a body of 11 MiB
    And an MCP client starts "retriever mcp --max-access read"
    When the client calls the tool "get_document" with these arguments:
      """
      {"documentId": "d0c1d0c1-0000-4000-8000-000000000001"}
      """
    Then the tool result is an error
    And the whole tool result is at most 1048576 bytes

  @e2e
  Scenario: The server ends with status 0 when its client closes the input
    Given an MCP client starts "retriever mcp --max-access read"
    When the client lists the tools
    Then the MCP server exits with status 0
