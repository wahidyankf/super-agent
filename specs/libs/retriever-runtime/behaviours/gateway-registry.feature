# The gateway registry builds one gateway for each source, on first use, and keeps it. The gateway holds the request lock
# and the pacing rule, so a second call for the same source reuses the same gateway.
Feature: Gateway registry
  A caller asks the registry for the gateway of a source. The registry builds it once and reuses it.
  A failed build is not kept, and a gateway that cannot do the asked work is refused.

  # Behaviour: RUN-018
  # Exemption(integration): the registry only keeps the gateways it builds, and it crosses no file, process, or socket boundary; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: Two requests for the same source get the same gateway
    Given a registry with a builder for the kind outline and the source "docs"
    When the caller asks the registry for the gateway of "docs" twice
    Then the builder runs once and both answers are the same gateway

  # Behaviour: RUN-019
  # Exemption(integration): the registry only keeps the gateways it builds, and it crosses no file, process, or socket boundary; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: Two sources get two gateways
    Given a registry with a builder for the kind outline and the sources "docs" and "wiki"
    When the caller asks the registry for the gateway of "docs" and of "wiki"
    Then each source gets its own gateway

  # Behaviour: RUN-020
  # Exemption(integration): the registry only keeps the gateways it builds, and it crosses no file, process, or socket boundary; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A source of a kind with no builder is refused as unsupported
    Given a registry with a builder for the kind outline only and the source "tests" of the kind kiwi
    When the caller asks the registry for the gateway of "tests"
    Then the call returns the error "unsupported source" and no gateway

  # Behaviour: RUN-021
  # Exemption(integration): the registry only keeps the gateways it builds, and it crosses no file, process, or socket boundary; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A failed build is not kept and does not stop the other sources
    Given a registry with a builder that fails for "docs" and succeeds for "wiki"
    When the caller asks the registry for the gateway of "docs" and then of "wiki"
    Then the request for "docs" returns the build error and the request for "wiki" returns a gateway

  # Behaviour: RUN-022
  # Exemption(integration): the registry only keeps the gateways it builds, and it crosses no file, process, or socket boundary; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A gateway that cannot read operations is refused as a reader
    Given a registry whose gateway for "docs" cannot read operations
    When the caller asks the registry for the reader of "docs"
    Then the call returns the error "unsupported source" and no reader

  # Behaviour: RUN-023
  # Exemption(integration): the registry only keeps the gateways it builds, and it crosses no file, process, or socket boundary; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: The writer of a source is the gateway that writes
    Given a registry whose gateway for "docs" writes operations
    When the caller asks the registry for the writer of "docs"
    Then the writer is the gateway that the registry keeps for "docs"

  # Behaviour: RUN-024
  # Exemption(integration): the registry only keeps the gateways it builds, and it crosses no file, process, or socket boundary; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A source whose kind has no builder has no writer
    Given a registry with a builder for the kind outline only and the source "tests" of the kind kiwi
    When the caller asks the registry for the writer of "tests"
    Then the call returns the error "unsupported source" and no writer
