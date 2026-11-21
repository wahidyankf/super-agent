# A source reads its API key and its account e-mail address from named environment variables. The library checks the
# variable names and the values it is given, and it never shows a secret value.
Feature: Source identity and credential names
  A caller names the environment variable that holds a key or an account address.
  The library refuses a malformed name, and a secret value never appears in any text.

  # Behaviour: CON-057
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: An upper-case identity variable name that ends in _EMAIL is accepted
    Given the identity variable name "<name>"
    When the caller builds an identity name from the name
    Then the identity name is built and its name is "<name>"

    Examples:
      | name       |
      | JIRA_EMAIL |
      | A_EMAIL    |
      | TEAM_2_EMAIL |

  # Behaviour: CON-058
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: An identity name not upper-case or not ending in _EMAIL is refused
    Given the identity variable name "<name>"
    When the caller builds an identity name from the name
    Then the call returns the error "invalid identity name: use an upper-case name that ends in _EMAIL" and no identity name

    Examples:
      | name        |
      | jira_email  |
      | JIRA_MAIL   |
      | 1JIRA_EMAIL |
      | _EMAIL      |

  # Behaviour: CON-059
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: A credential variable name that ends in _API_KEY or _API_TOKEN is accepted
    Given the credential variable name "<name>"
    When the caller builds a credential name from the name
    Then the credential name is built and its name is "<name>"

    Examples:
      | name             |
      | OUTLINE_API_KEY  |
      | KIWI_API_TOKEN   |
      | A_API_KEY        |

  # Behaviour: CON-060
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: A credential variable name of any other shape is refused
    Given the credential variable name "<name>"
    When the caller builds a credential name from the name
    Then the call returns the error "invalid credential name" and no credential name

    Examples:
      | name                |
      | outline_api_key     |
      | OUTLINE_API_SECRET  |
      | _API_KEY            |
      | OUTLINE_API_KEYS    |
      | OUTLINE_KEY         |

  # Behaviour: CON-061
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: An API key that is empty is refused and the error names its variable only
    Given an empty value for the variable "OUTLINE_API_KEY"
    When the caller builds an API key from the value
    Then the call returns the error "OUTLINE_API_KEY is not set" and no API key

  # Behaviour: CON-062
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: An API key shows as a redacted text in every standard formatting path
    Given an API key built from the value "EXAMPLE_API_KEY_VALUE"
    When the caller formats the key with the standard formats
    Then no formatted text holds "EXAMPLE_API_KEY_VALUE" and each one shows "[REDACTED]"

  # Behaviour: CON-063
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A valid account address is accepted and shown redacted
    Given an identity built from the variable "JIRA_EMAIL" and the value "name@example.test"
    When the caller formats the identity with the standard formats
    Then the identity is built, and no formatted text holds "name@example.test"

  # Behaviour: CON-063
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: An account address that is empty, invalid, or has a colon is refused
    Given an identity built from the variable "JIRA_EMAIL" and the value "<value>"
    When the caller builds the identity
    Then the call returns the error "<message>" and no identity

    Examples:
      | value              | message                                  |
      |                    | JIRA_EMAIL is not set                    |
      | name@example       | JIRA_EMAIL is not a valid e-mail address |
      | na:me@example.test | JIRA_EMAIL is not a valid e-mail address |
