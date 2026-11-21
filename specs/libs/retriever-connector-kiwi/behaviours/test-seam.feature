Feature: Kiwi test seam settings
  An operator may point the Kiwi adapter at a test server by setting the seam variables. With no seam variable set, the seam
  is off and reads no file. A bad setting stops the command with a message that names the variable, and it never names the
  local path of a file. A valid setting trusts only the certificate in the CA file, and it refuses redirects.

  # Behaviour: KIW-040
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour reads environment values and checks them, and the unit binding counts zero file reads; alternative-proof: retriever-connector-kiwi:test:unit
  @integration-exempt
  Scenario: With no seam variable set the seam is off, reads no file, and returns no error
    Given an environment with no seam variable set
    When the caller reads the seam
    Then the seam is off
    And no file is read
    And no error is returned

  # Behaviour: KIW-041
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour reads environment values and checks them, and the unit binding counts zero file reads; alternative-proof: retriever-connector-kiwi:test:unit
  @integration-exempt
  Scenario: The Kiwi seam reads only its own variables and never the Outline ones
    Given an environment that sets the Outline seam variables and no Kiwi seam variable
    When the caller reads the seam
    Then the seam is off

  # Behaviour: KIW-042
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour reads environment values and checks them, and the unit binding counts zero file reads; alternative-proof: retriever-connector-kiwi:test:unit
  @integration-exempt
  Scenario Outline: A bad seam setting stops the command with a message that names its variable
    Given a seam setting that is <setting>
    When the caller reads the seam
    Then the read fails with an error that names a seam variable and the CA file is not read

    Examples:
      | setting                                                  |
      | a CA file without a base address                         |
      | a timeout without a base address                         |
      | a base address that does not parse                       |
      | a plain HTTP base address                                |
      | a base address with user information                     |
      | a base address that is a host name and not a loopback address |
      | a base address with no CA file                           |
      | a timeout that is not a duration                         |
      | a timeout of zero                                        |

  # Behaviour: KIW-043
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A CA file that is missing names only the variable and not its path
    Given a seam whose CA file variable names a file that does not exist
    When the caller reads the seam
    Then the message names the CA file variable
    And the message does not show the path of the file

  # Behaviour: KIW-044
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A CA file with a valid certificate gives a client that trusts only that certificate and refuses redirects
    Given a seam with a valid base address, a timeout, and a CA file holding one certificate
    When the caller reads the seam
    Then the seam holds the base address and the timeout
    And the client trusts only the certificate of the file
    And the client refuses redirects

  # Behaviour: KIW-045
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A CA file that <problem> is refused without showing its path
    Given a seam whose CA file <problem>
    When the caller reads the seam
    Then the read fails
    And the message does not show the path of the file

    Examples:
      | problem                          |
      | cannot be read                   |
      | holds no certificate             |
