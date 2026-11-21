Feature: Write lookups by name
  A write parameter may name an entry instead of giving its ID. The engine resolves the name with one read, and it sends the
  write only when exactly one entry matches. A lookup that is paged is refused, because a second match could sit past the
  last page. A write never sends a name in place of an ID.

  # Behaviour: ENG-219
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A write named by one matching entry is sent with the ID of that entry
    Given a validated write whose parameter is looked up by name and the lookup finds one entry
    When the caller runs the write
    Then the write carries the ID of the matched entry and not the name

  # Behaviour: ENG-220
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario Outline: A name that matches <matches> fails with the unresolved-name error and no write is sent
    Given a validated write whose lookup finds <matches>
    When the caller runs the write
    Then the write fails with the unresolved-name error
    And no write request is sent

    Examples:
      | matches              |
      | two entries          |
      | no entry             |

  # Behaviour: ENG-221
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A lookup whose single read fails maps its error as a read does and sends no write
    Given a validated write whose lookup read fails
    When the caller runs the write
    Then the write fails with the error of the failed read
    And no write request is sent

  # Behaviour: ENG-222
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A lookup whose read declares no name parameter reads every entry and finds the one that matches by its field
    Given a validated write whose lookup read declares no parameter of the field's name
    When the caller runs the write
    Then the lookup read runs with no name
    And the one entry that matches is found by its field

  # Behaviour: ENG-223
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A lookup whose matched entry carries a text ID sends that text as it came
    Given a validated write whose lookup finds an entry with a text ID
    When the caller runs the write
    Then the write carries the text ID as it came

  # Behaviour: ENG-224
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A lookup parameter the caller leaves unset runs no read and its key is left out of the body
    Given a validated write whose optional lookup parameter the caller leaves unset
    When the caller runs the write
    Then no lookup read runs
    And the body has no key for that parameter

  # Behaviour: ENG-225
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A lookup read passes the caller's argument of the same name when it declares that parameter
    Given a validated write whose lookup read declares a parameter that the caller also sets
    When the caller runs the write
    Then the lookup read carries the caller's argument
    And the write keeps its own argument too

  # Behaviour: ENG-226
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A write whose lookup is over a paged read is refused before any request
    Given a write whose parameter is looked up through a paged read
    When the caller runs the write
    Then the write is refused and no request is sent
