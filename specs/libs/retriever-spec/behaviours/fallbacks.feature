Feature: Fallback reads and fallback pointers
  A read may declare one fallback: a second read of the same spec, run when the first reply triggers it. The trigger is
  one closed condition, an empty mapped field or a listed HTTP status. A fallback names the fields that it fills, and the
  spec refuses a chain of two fallbacks, so at most one extra request runs per call. A field may also name a second
  pointer that holds the same value in another reply shape.

  # Behaviour: SPC-024
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A fallback with one closed trigger and a fill list is accepted
    Given a connector spec whose metadata read falls back to the body read when <trigger>
    When the caller validates the spec document
    Then the spec is accepted

    Examples:
      | trigger                                  |
      | the markdown field is empty              |
      | the first reply has status 404            |

  # Behaviour: SPC-024
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A fallback that names two triggers is refused at its trigger
    Given a connector spec whose fallback names both an empty field and an HTTP status as its trigger
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path /operations/4/fallback/when

  # Behaviour: SPC-024
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A fallback that fills no field is refused at its fill list
    Given a connector spec whose fallback fill list is empty
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path /operations/4/fallback/fill

  # Behaviour: SPC-024
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A fallback trigger that lists a status outside the HTTP range is refused at the status
    Given a connector spec whose fallback trigger lists the status 99
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path /operations/4/fallback/when/status/0

  # Behaviour: SPC-024
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A fallback whose own target has a fallback is refused as a chain
    Given a connector spec in which the fallback target also declares a fallback
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path /operations/4/fallback/operation

  # Behaviour: SPC-024
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A fallback whose target is not a read of the same spec is refused at the target
    Given a connector spec whose fallback targets <target>
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path /operations/4/fallback/operation

    Examples:
      | target                                              |
      | a write operation                                   |
      | an operation id that the spec does not declare      |
      | a read that declares a parameter the first read does not declare |

  # Behaviour: SPC-035
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A field that names a second pointer for the same value is accepted
    Given a connector spec whose document read maps the id from a wrapped pointer, with a second pointer for the unwrapped shape
    When the caller validates the spec document
    Then the spec is accepted
