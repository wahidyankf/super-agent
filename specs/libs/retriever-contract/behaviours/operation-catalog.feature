# The operation catalog holds one row for each operation. A row names its command path, its tool, its effect class, and its
# parameters. A kind runs only its own rows, and the Kiwi and Jira kinds run only the rows of their closed lists.
Feature: Operation catalog
  A caller looks up an operation by its ID and reads its command path and effect class.
  A catalog refuses rows that cannot be told apart, and it names the kind that each row belongs to.

  # Behaviour: CON-096
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: A row is found by its ID with its command path and effect class
    Given the built-in catalog
    When the caller looks up the operation "<id>"
    Then the row is found, its command path is "<cli>", and its effect is "<effect>"

    Examples:
      | id                    | cli                       | effect      |
      | documents.get         | outline documents get     | read        |
      | documents.create      | outline documents create  | additive    |
      | documents.move        | outline documents move    | update      |
      | documents.archive     | outline documents archive | destructive |
      | kiwi.test-cases.update | kiwi test-cases update   | update      |
      | jira.issues.get       | jira issues get           | read        |

  # Behaviour: CON-097
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: An ID that the catalog does not hold is not found
    Given the built-in catalog
    When the caller looks up the operation "documents.vanish"
    Then no row is found

  # Behaviour: CON-098
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: A kind runs its own rows and the rows of its closed list only
    Given the built-in catalog
    When the caller asks whether the kind "<kind>" runs the operation "<id>"
    Then the answer is <runs>

    Examples:
      | kind    | id                     | runs  |
      | outline | documents.get          | true  |
      | outline | kiwi.test-cases.update | false |
      | kiwi    | kiwi.test-cases.update | true  |
      | kiwi    | documents.get          | false |
      | jira    | jira.issues.get        | true  |
      | jira    | kiwi.test-cases.update | false |

  # Behaviour: CON-099
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A catalog refuses a row that has no operation ID
    Given a catalog of the kinds outline, kiwi, and jira with one row that has no operation ID
    When the caller builds the catalog
    Then the call returns the error "an operation row needs an operation ID" and no catalog

  # Behaviour: CON-099
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A catalog refuses two rows with one operation ID
    Given a catalog of the kinds outline, kiwi, and jira with two rows that both hold the ID "documents.get"
    When the caller builds the catalog
    Then the call returns the error "duplicate operation row" and no catalog

  # Behaviour: CON-099
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A catalog refuses a row whose path starts with an unregistered kind
    Given a catalog of the kinds outline, kiwi, and jira with one row whose command path is "github list"
    When the caller builds the catalog
    Then the call returns the error "a row's command path must start with a registered kind" and no catalog

  # Behaviour: CON-100
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: The Outline resource words are listed sorted, with no Kiwi or Jira word
    Given the built-in catalog
    When the caller lists the resource words
    Then the first word is "access-requests" and the list holds no word of the kiwi or jira kinds

  # Behaviour: CON-101
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: A tools value is parsed into resource words, each one kept once
    Given the tools value "<raw>"
    When the caller parses the tools value
    Then the result is "<result>"

    Examples:
      | raw                 | result              |
      | documents,comments  | documents, comments |
      | documents,documents | documents           |

  # Behaviour: CON-101
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: A tools value that is empty, has an empty entry, or is unknown is refused
    Given the tools value "<raw>"
    When the caller parses the tools value
    Then the call returns the error "<message>" and no words

    Examples:
      | raw       | message                   |
      |           | the value is empty        |
      | documents, | an entry is empty        |
      | Documents | unknown resource "Documents" |
      | kiwi      | unknown resource "kiwi"   |

  # Behaviour: CON-102
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: The Outline write rows have the effect class of what they change
    Given the built-in catalog
    When the caller looks up the operation "<id>"
    Then the effect is "<effect>"

    Examples:
      | id                          | effect      |
      | documents.create            | additive    |
      | documents.duplicate         | additive    |
      | documents.update            | update      |
      | documents.unpublish         | update      |
      | documents.delete            | destructive |
      | documents.empty-trash       | destructive |
      | shares.create               | additive    |
      | shares.revoke               | destructive |
      | oauth-clients.rotate-secret | update      |

  # Behaviour: CON-103
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: Every Jira row is a read
    Given the built-in catalog
    When the caller looks up the operation "<id>"
    Then the effect is "read"

    Examples:
      | id                 |
      | jira.whoami        |
      | jira.projects.list |
      | jira.issues.search |
      | jira.issues.get    |
      | jira.comments.list |

  # Behaviour: CON-104
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: Each Kiwi row of the closed list has the effect of its change
    Given the built-in catalog
    When the caller looks up the operation "<id>"
    Then the effect is "<effect>"

    Examples:
      | id                    | effect   |
      | kiwi.test-cases.create | additive |
      | kiwi.test-cases.update | update   |
      | kiwi.plans.add-case    | additive |
