Feature: Reply safeguards and key echo
  The engine reads a reply within a size cap and a nesting cap, and it accepts only one whole JSON value. A reply that is
  too large, too deep, or hostile fails the read as an error with no panic. Control characters never reach a mapped text,
  and a key that the reply echoes never appears in an error or a reply. A key echo confirms that the reply is for the key
  that was asked for, and the check reads text only.

  # Behaviour: ENG-184
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A reply one byte over the default 10 MiB cap fails with the reply-size error
    Given a validated read and a replay whose reply is one byte over the default cap
    When the caller runs the read
    Then the read fails with the reply-size error

  # Behaviour: ENG-185
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A spec that lowers the reply cap reads a reply of exactly the cap and refuses one byte more
    Given a validated read whose spec lowers the reply cap
    When the caller runs the read with a reply of exactly the cap and then with one byte more
    Then the first reply is read
    And the second fails with the reply-size error

  # Behaviour: ENG-186
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A reply nested past the depth cap fails as unexpected and does not panic
    Given a validated read and a reply nested beyond the depth cap
    When the caller runs the read
    Then the read fails with an unexpected response and the engine does not panic

  # Behaviour: ENG-187
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A reply with more nodes than the token cap fails as unexpected
    Given a validated read and a reply with a node count above the token cap
    When the caller runs the read
    Then the read fails with an unexpected response

  # Behaviour: ENG-188
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario Outline: A reply that is not one whole JSON value fails as unexpected
    Given a validated read and a reply that is <shape>
    When the caller runs the read
    Then the read fails with an unexpected response

    Examples:
      | shape                                         |
      | two JSON values one after another             |
      | text that is not JSON                         |

  # Behaviour: ENG-189
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A mapped text keeps its line feeds and loses its bell and null characters
    Given a validated read whose reply text holds a bell, a null byte, and a line feed
    When the caller runs the read
    Then the mapped text has no bell and no null byte
    And the mapped text keeps its line feed

  # Behaviour: ENG-190
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A hostile reply shape in a list read fails as unexpected and never panics
    Given a validated list read and a hostile reply shape that the read must refuse
    When the caller runs the read
    Then the read fails with an unexpected response and the engine does not panic

  # Behaviour: ENG-191
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: Mapped text that would pass the output budget fails the read
    Given a validated read whose mapped text would exceed the output budget of one reply
    When the caller runs the read
    Then the read fails with the reply-size error

  # Behaviour: ENG-192
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A number in a reply is decoded without loss
    Given a validated read whose reply holds a number with many digits
    When the caller runs the read
    Then the mapped field holds the exact number

  # Behaviour: ENG-193
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A reply that echoes the key of a failed read keeps the key out of the error and the reply
    Given a validated read whose reply echoes the request key and a transport error that carries the key
    When the caller runs the read
    Then the error text holds no key
    And the reply holds no key

  # Behaviour: ENG-194
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A key echo that matches the request key serves the read
    Given a validated read with a key echo and a reply whose echo value equals the request argument
    When the caller runs the read
    Then the mapped record is returned

  # Behaviour: ENG-195
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario Outline: A key echo that does not match a text value at its pointer fails as unexpected
    Given a validated read with a key echo and a reply where the echo pointer <reply>
    When the caller runs the read
    Then the read fails with an unexpected response naming the echo path
    And neither the request key nor the reply value is shown

    Examples:
      | reply                              |
      | holds a different key              |
      | holds a number                     |
      | holds no value                     |

  # Behaviour: ENG-196
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A key echo on a list read is refused before any request
    Given a hand-built model with a key echo on a list response
    When the caller builds the engine from the model
    Then the construction fails and no request is sent
