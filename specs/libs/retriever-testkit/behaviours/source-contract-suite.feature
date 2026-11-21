# The source contract suite is one set of behaviour checks that every source gateway must pass. A gateway under test runs
# the suite, and each check reports its defect once.
Feature: Shared source contract suite
  A gateway author runs the same checks that every source gateway must pass.
  The suite passes the reference fake gateway, and it refuses an operation the fake does not serve.

  # Behaviour: TK-054
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: The reference fake gateway passes the whole source contract suite
    Given the reference fake gateway
    When the suite runs against the fake gateway
    Then every check of the suite passes

  # Behaviour: TK-055
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: A read of an operation that the fake does not serve is refused
    Given the reference fake gateway
    When the caller reads an operation that the fake does not serve
    Then the call returns an error and no reply

  # Behaviour: TK-056
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: The documents of an unknown collection report collection not found
    Given the reference fake gateway
    When the caller reads the documents of the collection "col-missing"
    Then the call returns the error "collection not found"

  # Behaviour: TK-056
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: The documents read filters by collection
    Given the reference fake gateway with the documents of two collections
    When the caller reads the documents of one collection
    Then only the documents of that collection are returned

  # Behaviour: TK-057
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: A check that a gateway fails reports its defect once
    Given a gateway with one defect in a check
    When the suite runs against that gateway
    Then the defect is reported once

  # Behaviour: TK-057
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: Every check of the suite passes the fake gateway when run on its own
    Given the reference fake gateway
    When each check of the suite runs on its own against the fake gateway
    Then every check passes

  # Behaviour: TK-058
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: A nested record is shown as plain values
    Given a record with a nested record and a list of fields
    When the caller shows the record as a value
    Then the nested record is shown as plain values
