Feature: Fallback reads, dual reply shapes, and the unavailable message
  A fallback read runs one second read when a trigger fires, and it fills the fields the spec names. The second read shares
  the time limit, the reply budget, and every safeguard of the first, and it never follows a redirect. A field with a
  fallback pointer maps either reply shape, and a primary pointer that finds an explicit null wins. A 404 with the declared
  unavailable message reports the operation as unavailable.

  # Behaviour: ENG-056
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A fallback runs once when the markdown field is empty and fills it with the export value
    Given a validated fallback read whose markdown field is empty in the first reply
    When the caller runs the read
    Then exactly two requests are sent
    And the markdown field holds the value of the export
    And the title of the first reply is kept

  # Behaviour: ENG-057
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A fallback is not run when the filled field is present, even when it holds a number
    Given a validated fallback read whose filled field is present in the first reply
    When the caller runs the read
    Then exactly one request is sent

  # Behaviour: ENG-058
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: Both requests of a fallback read share one time limit
    Given a validated fallback read whose spec lowers the time limit to 10 seconds
    When the caller runs the read
    Then the shared time limit is recorded once for the read and once for each request
    And both requests are sent

  # Behaviour: ENG-059
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A fallback that is answered with a redirect fails the read and follows nothing
    Given a validated fallback read whose export answers with a redirect
    When the caller runs the read
    Then the read fails with the redirect error
    And exactly two requests are sent and none reaches the host the redirect names

  # Behaviour: ENG-060
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A fallback runs when the first reply carries a listed status, even though that read failed
    Given a validated fallback read whose first reply carries a listed status
    When the caller runs the read
    Then the export is run and its fields fill the record
    And the record keeps the shape of the first read

  # Behaviour: ENG-061
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A failed fallback fails the read with its own error and not with the empty field of the first reply
    Given a validated fallback read whose export answers with not found
    When the caller runs the read
    Then the read fails with the not-found error of the export

  # Behaviour: ENG-062
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A required field that is missing in the first reply ends the read before any fallback runs
    Given a validated fallback read whose required markdown field has no value in the first reply
    When the caller runs the read
    Then the read fails with the mapping failure
    And exactly one request is sent

  # Behaviour: ENG-063
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: Both replies of a fallback read draw from one reply budget
    Given a validated fallback read with a small reply cap and an export value that no longer fits
    When the caller runs the read
    Then the read fails with the reply-size error
    And exactly two requests are sent

  # Behaviour: ENG-064
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A failed first read is never retried through the fallback
    Given a validated fallback read whose first request gets no reply
    When the caller runs the read
    Then the read ends with the transport error
    And exactly one request is sent

  # Behaviour: ENG-065
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A first reply the mapper refuses ends the read with the mapping error and no fallback runs
    Given a validated fallback read whose first reply cannot be mapped
    When the caller runs the read
    Then the read fails with the mapping error

  # Behaviour: ENG-066
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A fallback target that needs a converter the engine does not serve fails before any second mapping
    Given a validated fallback read whose target field names a converter the engine does not serve
    When the caller runs the read
    Then the read is unsupported and raw markup never passes as Markdown

  # Behaviour: ENG-067
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A fallback target that needs a value the caller does not give fails with the missing argument
    Given a validated fallback read whose export needs a revision the caller does not give
    When the caller runs the read
    Then the read fails with the missing-argument error

  # Behaviour: ENG-068
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A reply that lacks the primary pointer but holds the fallback shape maps the fallback shape
    Given a validated read whose field has a primary pointer and a fallback pointer, and a reply with only the fallback shape
    When the caller runs the read
    Then the field holds the value of the fallback pointer
    And the fallback pointer of the other field is used too

  # Behaviour: ENG-069
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A reply that holds the primary shape maps the primary pointers and leaves the fallbacks unused
    Given a validated dual-shape read and a reply in the primary shape
    When the caller runs the read
    Then the primary pointers map the reply
    And the fallback pointers are unused

  # Behaviour: ENG-070
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A primary pointer that finds an explicit null wins over the fallback
    Given a validated dual-shape read whose primary pointer holds an explicit null
    When the caller runs the read
    Then the field keeps the null and the fallback is not used

  # Behaviour: ENG-071
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A 404 with the declared unavailable message fails as operation unavailable
    Given a validated read with an unavailable message rule and a 404 reply that carries the declared message
    When the caller runs the read
    Then the read fails as operation unavailable

  # Behaviour: ENG-072
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A 404 without the declared message keeps its mapped not-found reason
    Given a validated read with an unavailable message rule and a 404 reply without the declared message
    When the caller runs the read
    Then the read fails with the mapped not-found reason

  # Behaviour: ENG-073
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A 401 that carries the declared message keeps the error of its own status
    Given a validated read with an unavailable message rule and a 401 reply that carries the declared message
    When the caller runs the read
    Then the read fails with the error of the 401 status and not as operation unavailable
