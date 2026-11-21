# A bound call holds its arguments and returns a record. A list record carries a page, which says whether more rows may
# follow and where they start. The library copies what it keeps, so a later change by the caller does not reach it.
Feature: Arguments, records, and pages
  A caller reads the bound arguments by name and reads the fields of a result record.
  A paged list says whether a next page may follow, and the offset it starts at.

  # Behaviour: CON-105
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: A bound argument is read by name, and an unset one reads as zero
    Given arguments with the values "limit" set to 25 and "query" set to "install"
    When the caller reads the argument "<name>" as <kind>
    Then the value is <value>

    Examples:
      | name    | kind   | value     |
      | limit   | number | 25        |
      | query   | text   | "install" |
      | offset  | number | 0         |
      | missing | text   | ""        |

  # Behaviour: CON-106
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: Changing the input map later does not change the arguments
    Given arguments built from a map with the value "install" for "query"
    When the caller changes the map to hold "other" for "query"
    Then the arguments still hold "install" for "query"

  # Behaviour: CON-106
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: Changing a returned list does not change the arguments
    Given arguments that hold the list "a", "b"
    When the caller changes the list it read back to hold "x"
    Then the arguments still hold the list "a", "b"

  # Behaviour: CON-107
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: A page reports whether more rows follow and where the next one starts
    Given a page that starts at offset <offset>, asks for at most <limit> rows, and receives <count> rows
    When the caller reads the page
    Then more rows may follow is <more> and the next offset is <next>

    Examples:
      | offset | limit | count | more  | next |
      | 0      | 2     | 2     | true  | 2    |
      | 0      | 2     | 1     | false | 1    |
      | 0      | 0     | 5     | false | 5    |
      | 10     | 25    | 25    | true  | 35   |

  # Behaviour: CON-108
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A list record keeps the page it was given
    Given a list record of two items with the page starting at offset 0 for a limit of 2
    When the caller reads the page of the record
    Then the page is present and its offset is 0

  # Behaviour: CON-108
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A record that is not a list reports no page
    Given a record with the field "name" set to "Engineering"
    When the caller reads the page of the record
    Then no page is present

  # Behaviour: CON-109
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: A record field is read by name, and a missing field is absent
    Given a record with the field "name" set to "Engineering" and the field "count" set to 3
    When the caller reads the field "<name>"
    Then the field is <result>

    Examples:
      | name  | result        |
      | name  | "Engineering" |
      | count | 3             |
      | owner | absent        |
