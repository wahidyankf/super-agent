Feature: Network guard
  Every connection the engine opens passes a guard. The guard refuses a host that is an IP address, a local name, or an
  address in a private range, unless the catalog allows private networks, and it never allows an unspecified or a
  multicast address. The engine dials the address the guard checked, and a failed dial is reported as the failure it is,
  not as a refusal.

  # Behaviour: ENG-092
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A host that is an IP address is refused before any lookup or dial, with or without the catalog setting
    Given a validated read whose host is an IP address
    When the caller runs the read with the catalog setting off and then on
    Then both runs are refused before any lookup and before any dial

  # Behaviour: ENG-093
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A local name is never connected to, with or without the catalog setting
    Given a validated read whose host is localhost or a subdomain of localhost
    When the caller runs the read with the catalog setting off and then on
    Then both runs are refused and no connection is made

  # Behaviour: ENG-094
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the refusal happens before any connection is made, so the check is a pure in-memory rule with no socket; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A resolver answer in a private range fails the call before any dial, so the host never sees the key
    Given a validated read whose host resolves to a private address
    When the caller runs the read
    Then the call fails before any dial
    And the key is never sent to the host

  # Behaviour: ENG-095
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the refusal happens before any connection is made, so the check is a pure in-memory rule with no socket; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A resolver answer with one private address refuses the whole host
    Given a validated read whose host resolves to a public address and a private address
    When the caller runs the read
    Then the host is refused and no dial is made

  # Behaviour: ENG-096
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): a dial to a public address would leave the machine, so the check runs against an in-memory dialer; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: The dial goes to the public address the guard checked, on the port of the request
    Given a validated read whose host resolves to one public address
    When the caller runs the read
    Then the dial goes to that address on the port of the request and not to the host name

  # Behaviour: ENG-097
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: The catalog setting allows a loopback address and still refuses an unspecified or a multicast address
    Given a validated read with the catalog setting allowing private networks
    When the caller runs the read against a loopback address, then an unspecified address, then a multicast address
    Then the loopback address is dialed
    And the unspecified and multicast addresses are refused

  # Behaviour: ENG-098
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the refusal happens before any connection is made, so the check is a pure in-memory rule with no socket; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario Outline: An address of the class <class> is refused unless the catalog setting allows private networks
    Given a validated read whose host resolves to an address of the class <class>
    When the caller runs the read with the catalog setting off
    Then the call is refused and no dial is made

    Examples:
      | class                          |
      | private range 10.0.0.0/8       |
      | private range 192.168.0.0/16   |
      | loopback                       |
      | link-local                     |
      | shared address space           |
      | unique-local IPv6              |

  # Behaviour: ENG-099
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): a dial to a public address would leave the machine, so the check runs against an in-memory dialer; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A dial error to a checked address is returned as the dial error and is not a refusal
    Given a validated read whose checked public address fails to dial
    When the caller runs the read
    Then the call returns the dial error
    And the error is not reported as a refusal

  # Behaviour: ENG-100
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): no address is found, so no connection is made and the check is a pure in-memory rule with no socket; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario Outline: A host that gives no address is unreachable
    Given a validated read whose host <problem>
    When the caller runs the read
    Then the call fails as unreachable

    Examples:
      | problem                                      |
      | fails the lookup                             |
      | returns an empty answer                      |

  # Behaviour: ENG-101
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: An address without a port is refused before the resolver is asked
    Given a dial target that holds an address without a port
    When the caller dials it
    Then the dial is refused and the resolver is not asked

  # Behaviour: ENG-102
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the refusal happens before any connection is made, so the check is a pure in-memory rule with no socket; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A write whose source name resolves to a private address is refused before the dialer runs
    Given a validated write whose source name resolves to a private address
    When the caller runs the write
    Then the dialer is not run and the write outcome is known

  # Behaviour: ENG-103
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the refusal happens before any connection is made, so the check is a pure in-memory rule with no socket; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: The transport the engine builds sends every dial through the guard
    Given a validated read and the transport that the engine builds
    When the caller dials through the transport to a private address
    Then the dial is refused by the guard

  # Behaviour: ENG-104
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the refusal happens before any connection is made, so the check is a pure in-memory rule with no socket; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A dial that reports the deadline is reported as a timeout with the timeout exit status
    Given a validated read whose dial reports the deadline at once
    When the caller runs the read
    Then the read fails as timed out with the timeout exit status

  # Behaviour: ENG-105
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A connection that drops while the reply body is read is reported as unreachable with no key in the text
    Given a validated read whose connection drops while the reply body is read
    When the caller runs the read
    Then the read fails as unreachable
    And the error text holds no key

  # Behaviour: ENG-106
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A loopback address is never dialed without the catalog setting and is dialed with it
    Given a dial to a checked loopback address
    When the caller dials without the catalog setting and then with it
    Then the first dial is refused and never made
    And the second dial is made to the checked address
