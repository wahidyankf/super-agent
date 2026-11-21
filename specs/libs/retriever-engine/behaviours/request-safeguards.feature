Feature: Request safeguards
  The engine builds each request from the declared operation and the caller's arguments. Only the headers the spec
  declares are sent. A value that would add a header line, leave its path segment, or splice into a query string is
  refused before any request leaves. A query value is encoded, a body value is encoded as JSON, and a redirect is never
  followed.

  # Behaviour: ENG-197
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A request carries only the headers the spec declares, the accept header, and the authorization header
    Given a validated read and an environment with extra headers available
    When the caller runs the read
    Then the request holds only the declared headers, the accept header, and the authorization header

  # Behaviour: ENG-198
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A header value with a control character is refused before any request
    Given a validated read whose header slot takes a value with a line break
    When the caller runs the read
    Then the call is refused with an invalid header error and no request is sent

  # Behaviour: ENG-199
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A constant header is sent as it is, a slot header takes its value, and an unset slot is left out
    Given a validated read with one constant header and one header slot that the caller leaves unset
    When the caller runs the read
    Then the constant header is sent as it is
    And the unset header slot is left out of the request

  # Behaviour: ENG-200
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario Outline: A path value that would leave its segment is refused before any request
    Given a validated read whose path slot takes the value <value>
    When the caller runs the read
    Then the call is refused and no request is sent

    Examples:
      | value                    |
      | ../admin                 |
      | a value with a line break |
      | a value with a slash     |
      | a value with a percent sign |

  # Behaviour: ENG-201
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A path slot fills one whole path segment
    Given a validated read whose path slot takes a plain value
    When the caller runs the read
    Then the request path holds the value as one whole segment

  # Behaviour: ENG-202
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A query value with reserved characters travels as encoded query text
    Given a validated read with a query parameter and a value that holds reserved characters
    When the caller runs the read
    Then the query carries the value encoded and never spliced into the path

  # Behaviour: ENG-203
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A declared query value is encoded and a paging value replaces the declared one
    Given a validated read with a declared query value and a paging value
    When the caller runs the read
    Then the declared query value is encoded in the query
    And the paging value replaces the declared one

  # Behaviour: ENG-204
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A body value is JSON-encoded and a value that looks like a slot stays literal text
    Given a validated write whose body slot takes a value with quotes, a line break, and text that looks like a slot
    When the caller runs the write
    Then the body carries the value as the text the caller gave

  # Behaviour: ENG-205
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A slot inside a longer text in the body is only text
    Given a validated read whose body holds a slot inside a longer text
    When the caller runs the read
    Then the body holds the slot as plain text

  # Behaviour: ENG-206
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  @integration
  Scenario: A redirect in the reply is refused and only one request is sent
    Given a validated read whose source answers with a redirect to another host
    When the caller runs the read
    Then the read fails with the redirect error
    And exactly one request is sent

  # Behaviour: ENG-207
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A key appears only in the authorization header of a read
    Given a validated read with a key and a replay of its example
    When the caller runs the read
    Then the key appears in the authorization header
    And the key appears in no path, query, body, or other header
