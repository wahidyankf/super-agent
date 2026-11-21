# The version of the retriever contract library. Every library releases in lockstep, so one value names the contract.
Feature: Contract version
  A caller of the contract library reads one version constant, so it can tell which contract it was built against.
  The value matches the release tag of the libraries.

  # Behaviour: CON-001
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: The contract reports version 0.4.0
    Given the caller imports the contract library
    When the caller reads the contract version
    Then the version is "0.4.0"
