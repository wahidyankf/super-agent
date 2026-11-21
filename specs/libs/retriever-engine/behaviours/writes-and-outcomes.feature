Feature: Writes and their outcomes
  A write is sent once. When a failure happens after the request has left, the outcome is unknown and the write is never
  sent again. A failure before the request leaves is a known failure that sends nothing. A write retries only a 429
  reply, because that reply means the source refused the request before it acted. A write carries the declared headers and
  the key only in the authorization header, and a write path value that would leave its segment is refused before any request.

  # Behaviour: ENG-227
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A write the source accepts is sent once with no confirmation and maps its declared fields
    Given a validated write and a replay that accepts it
    When the caller runs the write
    Then exactly one request is sent
    And no confirmation is asked
    And the result maps the declared fields

  # Behaviour: ENG-228
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A write that times out after its request left reports an unknown outcome and is not sent again
    Given a validated write whose request times out after it left
    When the caller runs the write
    Then the write reports an unknown outcome
    And exactly one request is sent

  # Behaviour: ENG-229
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario Outline: A failure after the request left is an unknown outcome and the write is sent once
    Given a validated write whose <failure> happens after the request left
    When the caller runs the write
    Then the write reports an unknown outcome
    And exactly one request is sent

    Examples:
      | failure                                        |
      | a reply that is over the reply cap             |
      | a reply nested past the depth cap              |
      | a reply body that breaks while it is read      |
      | a connection cut after the request left        |
      | an injected transport that fails after sending |
      | a bool map with a value that is not a flag     |

  # Behaviour: ENG-230
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  @integration
  Scenario Outline: A failure before the request leaves is a known failure and sends nothing
    Given a validated write whose <failure> happens before the request leaves
    When the caller runs the write
    Then the write is refused as a known failure
    And no request is sent

    Examples:
      | failure                          |
      | a connection that never opens    |
      | a TLS handshake that fails       |
      | a source name on a private address |

  # Behaviour: ENG-231
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A 4xx reply to a write keeps the error that the spec maps to its status and is never an unknown outcome
    Given a validated write whose source answers with a 4xx status that the spec maps to an error
    When the caller runs the write
    Then the write fails with the error that the spec maps to that status
    And the write is not reported as an unknown outcome

  # Behaviour: ENG-232
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A 4xx reply to a write does not echo its body in the error
    Given a validated write whose source answers with a 4xx reply that holds a body
    When the caller runs the write
    Then the error does not show the body of the reply

  # Behaviour: ENG-233
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  @integration
  Scenario: A write whose source answers with a redirect is refused after one request
    Given a validated write whose source answers with a redirect
    When the caller runs the write
    Then the write is refused and exactly one request is sent

  # Behaviour: ENG-234
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A write is retried after a 429 reply as a read is
    Given a validated write and a replay that answers 429 with a usable wait and then accepts the write
    When the caller runs the write
    Then the write waits the Retry-After time and is sent again
    And the write succeeds

  # Behaviour: ENG-235
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A write that keeps meeting 429 stops after three retries with the rate-limit exit status
    Given a validated write and a source that keeps answering 429
    When the caller runs the write
    Then the write stops after three retries
    And the exit status is 124

  # Behaviour: ENG-236
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A write header value with a line break is refused before any request
    Given a validated write whose header slot takes a value with a line break
    When the caller runs the write
    Then the call is refused and no request is sent

  # Behaviour: ENG-237
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A write carries only its declared headers, the accept header, the content type, and the authorization header
    Given a validated write and an environment with extra headers available
    When the caller runs the write
    Then the request holds only those headers

  # Behaviour: ENG-238
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: The key of a write appears only in the authorization header
    Given a validated write with a key
    When the caller runs the write
    Then the key appears in the authorization header
    And the key appears in no path, query, body, or other header

  # Behaviour: ENG-239
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A write path value that would leave its segment is refused before any request
    Given a validated write whose path slot takes a value such as ../admin
    When the caller runs the write
    Then the call is refused and no request is sent

  # Behaviour: ENG-240
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A body parameter the caller leaves unset leaves its key out of the body
    Given a validated write with an optional body parameter that the caller does not set
    When the caller runs the write
    Then the body has no key for that parameter

  # Behaviour: ENG-241
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A paged write is refused before any request
    Given a hand-built write model with a paging rule
    When the caller runs the write
    Then the call is refused and no request is sent

  # Behaviour: ENG-242
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A write whose paging style is none is served as an unpaged write
    Given a validated write whose paging style is none
    When the caller runs the write
    Then the write runs as an unpaged write

  # Behaviour: ENG-243
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: An admin operation runs as a write and a read of it is still refused
    Given a validated spec with an operation labelled admin
    When the caller runs the operation through the write path and through the read path
    Then the write path serves it
    And the read path refuses it

  # Behaviour: ENG-244
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A write that the catalog refuses is refused before any request
    Given a call to a write with an unknown operation, a read operation, an undeclared argument, or a missing required argument
    When the caller runs the write
    Then the call is refused and no request is sent

  # Behaviour: ENG-245
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A write whose reply needs a converter the engine does not serve is refused before any request
    Given a validated write whose reply field names a converter the engine does not serve
    When the caller runs the write
    Then the call is refused and no request is sent

  # Behaviour: ENG-246
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A write whose reply uses the editor-data converter is served
    Given a validated write whose reply field names the editor-data converter
    When the caller runs the write
    Then the write is served and no request is refused for the converter
