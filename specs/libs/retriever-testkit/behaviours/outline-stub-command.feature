# The Outline stub command serves one Outline stub as a process, so an end-to-end test can call it over TLS. It refuses bad
# flags before it listens, and it writes the certificate and announces its address.
Feature: Outline stub command
  A test runs the Outline stub command with flags. The command checks the flags first, then listens on the loopback,
  writes the certificate file, and announces its address on standard output.

  # Behaviour: TK-061
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  Scenario: The command refuses to start without a certificate file path
    Given the Outline stub command with no -ca-file flag
    When the command runs
    Then the command exits with status 1 and prints "the -ca-file flag is required" and listens on no port

  # Behaviour: TK-062
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  Scenario: The command refuses both status faults at once
    Given the Outline stub command with the -list-status 500 and -document-status 503 flags
    When the command runs
    Then the command exits with status 1 and prints "only one of -list-status and -document-status can be set" and listens on no port

  # Behaviour: TK-063
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  Scenario Outline: A collection flag is read as id=name in the order given
    Given the collection flag value "<value>"
    When the command reads the collection flag
    Then the result is "<result>"

    Examples:
      | value     | result                                     |
      | c1=Eng    | the collection c1 named Eng                |
      | c1=       | an error that holds "-collection wants id=name" |

  # Behaviour: TK-064
  Scenario: The command announces its loopback address after it writes the certificate
    Given the Outline stub command with the -ca-file flag and a free port
    When the command starts
    Then the command writes the certificate file and prints "outline stub listening on" followed by its loopback URL

  # Behaviour: TK-065
  Scenario: The command stops answering when its context ends
    Given the Outline stub command started with its context
    When the context ends
    Then the command stops listening and returns

  # Behaviour: TK-066
  Scenario: A documents file that is not JSON is refused with its parse error
    Given the Outline stub command with a documents file that is not JSON
    When the command runs
    Then the command exits with status 1 and prints an error that starts with "parse the documents file"
