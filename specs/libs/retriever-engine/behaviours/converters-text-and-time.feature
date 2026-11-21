Feature: Text, host, and time converters
  A field may name a converter that turns one reply value into its text: editor data into plain text, an address into its
  host name, and a Jira timestamp, date, or title into its output form. A value the converter cannot read maps to an empty
  text or a null, and the field stays present, so a gap is visible. A converted text that passes the output budget fails
  the read.

  # Behaviour: ENG-029
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: Editor data in a reply maps to the plain words of the data, one block per line, with the mention label
    Given a validated read whose field names the editor-data converter and a reply holding editor data
    When the caller runs the read
    Then the field holds the plain text of each block on its own line
    And a mention appears as its label with an at sign
    And the editor JSON does not reach the record

  # Behaviour: ENG-030
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A reply with no editor data maps the raw text unchanged through the fallback pointer
    Given a validated read whose field names the editor-data converter and a reply with plain text and no editor data
    When the caller runs the read
    Then the field holds the raw text unchanged

  # Behaviour: ENG-031
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario Outline: An editor-data field that is <state> maps to null with no partial text
    Given a validated read whose editor-data field is <state>
    When the caller runs the read
    Then the field is null

    Examples:
      | state     |
      | missing   |
      | null      |

  # Behaviour: ENG-032
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario Outline: An editor-data field holding <value> fails the read as unexpected
    Given a validated read whose editor-data field holds <value>
    When the caller runs the read
    Then the read fails with an unexpected response

    Examples:
      | value        |
      | a number     |
      | a flag       |

  # Behaviour: ENG-033
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A converted editor text past the budget fails the read
    Given a validated read whose editor data converts to more text than one reply may hold
    When the caller runs the read
    Then the read fails with the reply-size error

  # Behaviour: ENG-034
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: An address field maps to the host name of the address without the path, query, port, or user name
    Given a validated read whose field names the hostname converter and a reply holding a full address
    When the caller runs the read
    Then the field holds only the host name of the address

  # Behaviour: ENG-035
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario Outline: An address field that is <state> maps to the empty text
    Given a validated read whose hostname field is <state>
    When the caller runs the read
    Then the field holds the empty text

    Examples:
      | state                                 |
      | an address the converter cannot parse |
      | missing                               |

  # Behaviour: ENG-036
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A Jira read maps the timestamp in UTC, the date as it is, and the title as one line
    Given a validated read naming the Jira converters and a reply with a timestamp, a date, and a title
    When the caller runs the read
    Then the timestamp is an RFC 3339 time in UTC
    And the date passes through unchanged
    And the title stays on one line

  # Behaviour: ENG-037
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A Jira title counts its limit in characters and keeps its visible text
    Given a validated Jira title field and a reply title longer than the limit of 255 characters
    When the caller runs the read
    Then the title holds at most 255 characters
    And no line break, control character, or bidi override reaches the output

  # Behaviour: ENG-038
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A Jira date that is a bare year maps to the empty text with the field present
    Given a validated read whose Jira date field holds a bare year
    When the caller runs the read
    Then the field holds the empty text and the field is present

  # Behaviour: ENG-039
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A Jira converter that receives a value of another shape maps it to the empty text and keeps the row
    Given a validated read whose Jira converter field holds a number, a flag, an object, or a list
    When the caller runs the read
    Then each such field holds the empty text and the row is kept

  # Behaviour: ENG-040
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A Jira converter with a missing or null value maps to the empty text
    Given a validated read whose Jira converter field is missing or null in the reply
    When the caller runs the read
    Then the field holds the empty text and the field is present

  # Behaviour: ENG-041
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A converted Jira text past the budget fails the read
    Given a validated read whose Jira converter output would pass the budget of one reply
    When the caller runs the read
    Then the read fails with the reply-size error
