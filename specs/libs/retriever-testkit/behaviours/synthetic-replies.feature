# The synthetic replies are the recorded Outline replies that the behaviour harness serves. The files are embedded in the
# package, so a test reads them without a disk path.
Feature: Synthetic Outline replies
  A behaviour test reads the synthetic reply of each Outline method from the package.
  Each reply is one valid JSON file, and the named methods all have one.

  # Behaviour: TK-059
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: The package holds one valid JSON reply for each method
    Given the synthetic replies of the package
    When the caller reads the reply file of each method
    Then every reply file holds one valid JSON value

  # Behaviour: TK-060
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario Outline: The package holds the reply of each named method
    Given the synthetic replies of the package
    When the caller looks for the reply of the method "<method>"
    Then the reply file "<method>.json" is found

    Examples:
      | method               |
      | collections.list     |
      | collections.info     |
      | documents.info       |
      | documents.export     |
