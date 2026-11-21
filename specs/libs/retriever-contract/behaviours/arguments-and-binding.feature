# Binding turns the raw arguments of one call into the checked arguments of one operation row. The CLI and the MCP tools
# both bind the same way, so one value gets one answer on every surface.
Feature: Argument binding
  A caller binds the arguments of one call to one operation row.
  An argument the row does not declare, a missing required argument, or a value that breaks its kind is refused before any request.

  # Behaviour: CON-083
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: An argument that the operation does not declare is refused
    Given the operation "outline documents get" and the argument "bogus" set to "x"
    When the caller binds the arguments
    Then the call returns the error "unknown argument" and no arguments

  # Behaviour: CON-083
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: The source argument is accepted by every operation
    Given the operation "outline documents get" with the document "doc-1" and the argument "source" set to "outline"
    When the caller binds the arguments
    Then the arguments are bound and the document is "doc-1"

  # Behaviour: CON-084
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A required argument that is not set is refused
    Given the operation "outline documents get" with no document
    When the caller binds the arguments
    Then the call returns the error "a required argument is missing" and no arguments

  # Behaviour: CON-085
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: An optional argument that is not set takes its default
    Given the operation "jira projects list" with no arguments
    When the caller binds the arguments
    Then the bound limit is 25 and the bound offset is 0

  # Behaviour: CON-086
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: A page limit from 1 to 100 is accepted and a limit outside it is refused
    Given the operation "jira issues search" with the jql "project = EXAMPLE" and the limit "<limit>"
    When the caller binds the arguments
    Then the result is "<result>"

    Examples:
      | limit | result                                  |
      | 1     | bound, with the limit 1                 |
      | 100   | bound, with the limit 100               |
      | 0     | the error "limit must be from 1 to 100" |
      | 101   | the error "limit must be from 1 to 100" |

  # Behaviour: CON-087
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: An index outside its range is refused and the error names the argument
    Given the operation "outline documents move" with the document "doc-1", the collection "550e8400-e29b-41d4-a716-446655440000", and the index "<index>"
    When the caller binds the arguments
    Then the call returns the error "index: number is out of range" and no arguments

    Examples:
      | index      |
      | -1         |
      | 2147483648 |

  # Behaviour: CON-088
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: An operation with an exactly-one rule refuses a call that names no target
    Given the operation "outline subscriptions get" with no document and no collection
    When the caller binds the arguments
    Then the call returns the error "exactly one of the arguments that select the target is required" and no arguments

  # Behaviour: CON-088
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: An exactly-one operation refuses a call that names two targets
    Given the operation "outline subscriptions get" with the document "doc-1" and the collection "550e8400-e29b-41d4-a716-446655440000"
    When the caller binds the arguments
    Then the call returns the error "exactly one of the arguments that select the target is required" and no arguments

  # Behaviour: CON-088
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: An operation with an exactly-one rule accepts one choice
    Given the operation "outline subscriptions get" with the document "doc-1"
    When the caller binds the arguments
    Then the arguments are bound

  # Behaviour: CON-089
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: An update with none of its changing arguments is refused
    Given the operation "kiwi test-cases update" with the document "case-1" and no change
    When the caller binds the arguments
    Then the call returns the error "a required argument is missing" and no arguments

  # Behaviour: CON-089
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: An update with one changing argument is accepted
    Given the operation "kiwi test-cases update" with the document "case-1" and the summary "Login works"
    When the caller binds the arguments
    Then the arguments are bound

  # Behaviour: CON-090
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A text value of a single dash is refused, as standard input is not read
    Given the operation "outline documents create" with the title "Notes", the collection "550e8400-e29b-41d4-a716-446655440000", and the text "-"
    When the caller binds the arguments
    Then the call returns the error "standard input is not supported; pass the text as the argument" and no arguments

  # Behaviour: CON-091
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A whole number given as a number is read as that number
    Given the operation "jira issues search" with the jql "project = EXAMPLE" and the limit 10
    When the caller binds the arguments
    Then the bound limit is 10

  # Behaviour: CON-091
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A whole number given as text is read as that number
    Given the operation "jira issues search" with the jql "project = EXAMPLE" and the limit given as the text "10"
    When the caller binds the arguments
    Then the bound limit is 10

  # Behaviour: CON-091
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A decimal number with no fraction is read as a whole number
    Given the operation "jira issues search" with the jql "project = EXAMPLE" and the limit 10.0
    When the caller binds the arguments
    Then the bound limit is 10

  # Behaviour: CON-091
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A limit that is not a whole number is refused with the limit rule
    Given the operation "jira issues search" with the jql "project = EXAMPLE" and the limit 10.5
    When the caller binds the arguments
    Then the call returns the error "limit must be from 1 to 100" and no arguments

  # Behaviour: CON-092
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: A flag accepts only the text true or false
    Given the operation "outline documents create" with the title "Notes", the collection "550e8400-e29b-41d4-a716-446655440000", the text "Body", and publish "<value>"
    When the caller binds the arguments
    Then the result is "<result>"

    Examples:
      | value | result                                     |
      | true  | bound, with publish true                   |
      | false | bound, with publish false                  |
      | yes   | the error "value must be true or false"    |

  # Behaviour: CON-093
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A list of allowed values keeps each value once in its first position
    Given the operation "outline comments list" with the document "doc-1" and the status filter "resolved,resolved"
    When the caller binds the arguments
    Then the bound status filter is "resolved"

  # Behaviour: CON-093
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A value that is not in the allowed list is refused
    Given the operation "outline comments list" with the document "doc-1" and the status filter "open"
    When the caller binds the arguments
    Then the call returns the error "value is not one of the allowed values" and no arguments

  # Behaviour: CON-094
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A document ID outside the ID grammar is refused as an invalid document ID
    Given the operation "outline documents get" with the document "doc 1"
    When the caller binds the arguments
    Then the call returns the error "invalid document ID" and no arguments

  # Behaviour: CON-094
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A collection that is not a UUID is refused as an invalid collection ID
    Given the operation "outline documents create" with the title "Notes", the collection "not-a-uuid", and the text "Body"
    When the caller binds the arguments
    Then the call returns the error "invalid collection ID" and no arguments

  # Behaviour: CON-095
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A time without a zone is refused as an invalid time
    Given the operation "outline notifications update" with the notification "550e8400-e29b-41d4-a716-446655440000" and the viewed time "2026-11-21T10:00:00"
    When the caller binds the arguments
    Then the call returns the error "invalid time: use RFC 3339 with a zone" and no arguments
