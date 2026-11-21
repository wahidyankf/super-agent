# The env file gives a source its key and its account address when the process does not set them. The process value wins,
# and an empty value in either place counts as missing. A malformed line is named by its number, never by its text.
Feature: Source keys from the environment and the local env file
  A caller reads the API key or the account address of a source from the process environment, then from one variable of
  the local env file. The reader shows a value nowhere, and an error names the variable or the line only.

  # Behaviour: RUN-046
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: The process value wins and the env file is not read
    Given the variable "EXAMPLE_OUTLINE_API_KEY" set to "EXAMPLE-PROCESS-KEY" in the process and to "EXAMPLE-FILE-KEY" in the env file
    When the caller reads the key of the variable
    Then the key is "EXAMPLE-PROCESS-KEY"

  # Behaviour: RUN-046
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: The env file value is read when the process has none
    Given the variable "EXAMPLE_OUTLINE_API_KEY" set to "EXAMPLE-FILE-KEY" in the env file and not set in the process
    When the caller reads the key of the variable
    Then the key is "EXAMPLE-FILE-KEY"

  # Behaviour: RUN-047
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: An empty process value counts as missing, so the file value is read
    Given the variable "EXAMPLE_OUTLINE_API_KEY" set to "" in the process and to "EXAMPLE-FILE-KEY" in the env file
    When the caller reads the key of the variable
    Then the key is "EXAMPLE-FILE-KEY"

  # Behaviour: RUN-048
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A key missing from both places names its variable in the error
    Given the variable "EXAMPLE_OUTLINE_API_KEY" that is set to "" in the env file and not set in the process
    When the caller reads the key of the variable
    Then the call returns the error "EXAMPLE_OUTLINE_API_KEY is not set" and no key

  # Behaviour: RUN-048
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A key whose env file is absent is a missing key
    Given no env file and the variable "EXAMPLE_OUTLINE_API_KEY" not set in the process
    When the caller reads the key of the variable
    Then the call returns the error "EXAMPLE_OUTLINE_API_KEY is not set" and no key

  # Behaviour: RUN-049
  # Exemption(integration): the reference is refused before any read, so no variable or file is touched; the unit binding counts the reads; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: The zero credential reference is refused before any read
    Given the zero credential reference
    When the caller reads the key of the reference
    Then the call returns an error and no read of the environment is made

  # Behaviour: RUN-050
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A value in the env file is read with its quotes and comments taken off
    Given an env file with the line "<line>"
    When the caller reads the key of the variable "EXAMPLE_OUTLINE_API_KEY"
    Then the key is "<value>"

    Examples:
      | line                                              | value              |
      | EXAMPLE_OUTLINE_API_KEY=EXAMPLE-KEY               | EXAMPLE-KEY        |
      | export EXAMPLE_OUTLINE_API_KEY=EXAMPLE-KEY        | EXAMPLE-KEY        |
      | EXAMPLE_OUTLINE_API_KEY="EXAMPLE KEY"            | EXAMPLE KEY        |
      | EXAMPLE_OUTLINE_API_KEY=EXAMPLE-KEY # a comment   | EXAMPLE-KEY        |

  # Behaviour: RUN-051
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A repeated variable in the env file takes its last value
    Given an env file with the lines "EXAMPLE_OUTLINE_API_KEY=FIRST" and "EXAMPLE_OUTLINE_API_KEY=LAST"
    When the caller reads the key of the variable "EXAMPLE_OUTLINE_API_KEY"
    Then the key is "LAST"

  # Behaviour: RUN-052
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A malformed env file line is refused and the error gives its number
    Given an env file whose line 2 "<line>"
    When the caller reads the key of the variable "EXAMPLE_OUTLINE_API_KEY"
    Then the call returns the error "local env file line 2: <reason>" and the line text is not in the error

    Examples:
      | line                       | reason                       |
      | has no equals sign         | expected NAME=VALUE          |
      | has an unclosed quote      | unclosed quote               |
      | has text after a closing quote | text after a closing quote |

  # Behaviour: RUN-053
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: An env file that is a symbolic link or a folder is refused
    Given an env file path that is a "<kind>"
    When the caller reads the key of the variable "EXAMPLE_OUTLINE_API_KEY"
    Then the call returns the error "local env file must be a regular file, not a symlink or a folder" and no key

    Examples:
      | kind          |
      | symbolic link |
      | folder        |

  # Behaviour: RUN-054
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: An account address held in the env file is read when the process has none
    Given the variable "EXAMPLE_JIRA_EMAIL" set to "name@example.test" in the env file and not set in the process
    When the caller reads the identity of the variable
    Then the identity is built

  # Behaviour: RUN-055
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A malformed account address is refused, naming the variable only
    Given the variable "EXAMPLE_JIRA_EMAIL" set to "EXAMPLE-NOT-AN-ADDRESS" in the env file
    When the caller reads the identity of the variable
    Then the call returns the error "EXAMPLE_JIRA_EMAIL is not a valid e-mail address" and the address is not in the error

  # Behaviour: RUN-056
  # Exemption(integration): the rule is the text that the formatters print for a value, which needs no file, process, or socket; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A key or account address shows as redacted in every format
    Given a key and an account address read from the environment with the values "EXAMPLE-KEY" and "name@example.test"
    When the caller formats the key and the address with the standard formats
    Then neither value appears in any formatted text
