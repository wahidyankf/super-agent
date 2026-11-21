Feature: Plugin gateway mapping
  A gateway call sends one named method to the plugin and maps the reply onto the domain values that the engine uses. A
  collection tree maps a null parent to a root. A reply of the wrong shape, or a field of the wrong type, fails the call with
  a protocol error. A failure from the plugin call reaches the caller unchanged. The gateway serves a source through the
  same three ports as the engine, so the runtime sees one shape.

  # Behaviour: PLG-020
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-plugin:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: The list of collections maps each collection onto a domain value in reply order
    Given a source whose plugin lists two collections
    When the caller lists the collections through the gateway
    Then the result holds the two collections in reply order

  # Behaviour: PLG-021
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-plugin:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A collection list reply of a bad shape fails with a protocol error
    Given a plugin whose collection list reply is <shape>
    When the caller lists the collections through the gateway
    Then the call fails with a protocol error

    Examples:
      | shape                                  |
      | a result that is not a list            |
      | a collection that is not an object     |
      | a collection with a field of the wrong type |

  # Behaviour: PLG-022
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-plugin:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: The collection tree maps a null parent to a root and maps each depth onto a number
    Given a plugin whose collection tree reply holds a root and a child with its depth
    When the caller reads the collection tree through the gateway
    Then the root has no parent
    And each node carries its depth as a number

  # Behaviour: PLG-023
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-plugin:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A collection tree reply of a bad shape fails with a protocol error
    Given a plugin whose collection tree reply is <shape>
    When the caller reads the collection tree through the gateway
    Then the call fails with a protocol error

    Examples:
      | shape                                  |
      | a node that is not an object           |
      | a node with a missing or mistyped field |

  # Behaviour: PLG-024
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-plugin:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A document reply maps onto the domain document
    Given a plugin whose document reply holds an object with its fields
    When the caller reads the document through the gateway
    Then the result is the document of the reply

  # Behaviour: PLG-025
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-plugin:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A document reply of a bad shape fails with a protocol error
    Given a plugin whose document reply is <shape>
    When the caller reads the document through the gateway
    Then the call fails with a protocol error

    Examples:
      | shape                                  |
      | a missing object                       |
      | a missing field                        |

  # Behaviour: PLG-026
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A read or write sends its operation and bound arguments in the request and maps the reply fields in name order
    Given a source whose plugin answers a read or a write with its fields
    When the caller runs the read and the write through the gateway
    Then each request carries the operation and the bound arguments in its params
    And each reply maps onto a record with its fields in name order

  # Behaviour: PLG-027
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-plugin:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A read reply of items maps each item onto an object record in order
    Given a plugin whose read reply holds a list of items
    When the caller runs the read through the gateway
    Then each item maps onto an object record and the records keep their order

  # Behaviour: PLG-028
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-plugin:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A read or write reply whose fields or items have the wrong shape fails with a protocol error
    Given a plugin whose read or write reply has <shape>
    When the caller runs the read or the write through the gateway
    Then the call fails with a protocol error

    Examples:
      | shape                                  |
      | no fields object                       |
      | a fields object of the wrong type      |
      | no items list                          |
      | an items value that is not a list      |

  # Behaviour: PLG-029
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A failure of the plugin call reaches the caller unchanged from every gateway method
    Given a plugin call that fails with a <failure> failure
    When the caller calls each gateway method
    Then each method returns the <failure> failure unchanged

    Examples:
      | failure         |
      | pin mismatch    |
      | protocol        |
      | timeout         |

  # Behaviour: PLG-030
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A timeout of the plugin call surfaces through the gateway with exit status 124
    Given a plugin that never answers through the gateway
    When the caller makes a gateway call
    Then the call fails with the timeout and exit status 124

  # Behaviour: PLG-031
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A source with an identity resolves its address and passes both variables to the plugin
    Given a source with an identity that the gateway resolves to an address
    When the caller makes a gateway call
    Then the plugin receives the address and the identity variables

  # Behaviour: PLG-032
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-plugin:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: The gateway serves the same three ports as the engine
    Given the plugin gateway
    When the caller checks its ports
    Then the gateway serves the same three ports as the engine
