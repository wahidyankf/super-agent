# The value checks that the library runs on a caller's text and numbers: names, free text, reasons, e-mail addresses,
# times, fractional indexes, colours, icons, scores, ID lists, event names, HTTPS addresses, search queries, and page
# bounds. Each check accepts one shape and refuses everything else, and each refusal names the rule.
Feature: Text and number value checks
  A caller checks a value before it sends it on. A value that breaks a rule returns the error of that rule.
  The error states its limit in characters, bytes, or a numeric range.

  # Behaviour: CON-018
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: A name of 1 to 255 characters with no control character is accepted
    Given the name "<name>"
    When the caller validates the name
    Then the check passes

    Examples:
      | name         |
      | Engineering  |
      | R&D = Drafts |
      | Ünïcode name |
      | x            |

  # Behaviour: CON-019
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: An empty name is refused
    Given the name ""
    When the caller validates the name
    Then the check returns the invalid name error

  # Behaviour: CON-019
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A name of 256 characters is refused
    Given a name made of 256 letter a characters
    When the caller validates the name
    Then the check returns the invalid name error

  # Behaviour: CON-019
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A name with a tab character is refused
    Given a name with a tab character between two words
    When the caller validates the name
    Then the check returns the invalid name error

  # Behaviour: CON-020
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: An empty text is accepted
    Given the text ""
    When the caller validates the text
    Then the check passes

  # Behaviour: CON-020
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A text of exactly 1 MiB is accepted
    Given a text of exactly 1048576 bytes
    When the caller validates the text
    Then the check passes

  # Behaviour: CON-021
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A text one byte longer than 1 MiB is refused
    Given a text of 1048577 bytes
    When the caller validates the text
    Then the check returns the error "text must be at most 1 MiB"

  # Behaviour: CON-022
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: A reason of up to 2000 characters with no control character is accepted
    Given a reason of <length> characters with no control character
    When the caller validates the reason
    Then the check passes

    Examples:
      | length |
      | 0      |
      | 1      |
      | 2000   |

  # Behaviour: CON-023
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A reason of 2001 characters is refused
    Given a reason of 2001 characters with no control character
    When the caller validates the reason
    Then the check returns the error "invalid reason: use at most 2000 characters with no control character"

  # Behaviour: CON-023
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A reason with a line feed character is refused
    Given a reason with a line feed character between two words
    When the caller validates the reason
    Then the check returns the error "invalid reason: use at most 2000 characters with no control character"

  # Behaviour: CON-024
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: An e-mail address with a local part and a dotted host is accepted
    Given the e-mail address "<address>"
    When the caller validates the e-mail address
    Then the check passes

    Examples:
      | address              |
      | name@example.com     |
      | a.b@sub.example.test |

  # Behaviour: CON-025
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: An e-mail address without a local part or a dotted host is refused
    Given the e-mail address "<address>"
    When the caller validates the e-mail address
    Then the check returns the invalid e-mail address error

    Examples:
      | address           |
      | name@             |
      | @example.com      |
      | name@example      |
      | two@@example.com  |
      | a b@example.com   |
      | name@.example.com |
      |                   |

  # Behaviour: CON-026
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: A time in RFC 3339 with a zone is accepted
    Given the time "<time>"
    When the caller validates the time
    Then the check passes

    Examples:
      | time                      |
      | 2026-11-21T10:00:00Z      |
      | 2026-11-21T10:00:00+07:00 |

  # Behaviour: CON-027
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: A time without a zone, or in another format, is refused
    Given the time "<time>"
    When the caller validates the time
    Then the check returns the error "invalid time: use RFC 3339 with a zone"

    Examples:
      | time                |
      | 2026-11-21T10:00:00 |
      | 2026-11-21          |
      | 21/11/2026          |
      |                     |

  # Behaviour: CON-028
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: A fractional index of 1 to 256 printable ASCII characters is accepted
    Given the index "<index>"
    When the caller validates the index
    Then the check passes

    Examples:
      | index |
      | a0    |
      | 1     |
      | a b   |
      | ~     |

  # Behaviour: CON-029
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: A fractional index that is empty or holds a non-ASCII character is refused
    Given the index "<index>"
    When the caller validates the index
    Then the check returns the error "invalid index: use 1 to 256 printable ASCII characters"

    Examples:
      | index |
      |       |
      | é     |

  # Behaviour: CON-029
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A fractional index with a tab character is refused
    Given an index with a tab character between two letters
    When the caller validates the index
    Then the check returns the error "invalid index: use 1 to 256 printable ASCII characters"

  # Behaviour: CON-029
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A fractional index of 257 characters is refused
    Given an index made of 257 letter a characters
    When the caller validates the index
    Then the check returns the error "invalid index: use 1 to 256 printable ASCII characters"

  # Behaviour: CON-030
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: A hex colour of three or six digits is accepted in either letter case
    Given the colour "<colour>"
    When the caller validates the colour
    Then the check passes

    Examples:
      | colour  |
      | #3a7bd5 |
      | #3ad    |
      | #ABCDEF |

  # Behaviour: CON-031
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: A colour without a hash, with another digit count, or a name is refused
    Given the colour "<colour>"
    When the caller validates the colour
    Then the check returns the error "invalid color: use a hex color such as #3a7bd5 or #3ad"

    Examples:
      | colour    |
      | 3a7bd5    |
      | #3a7bd    |
      | #3a7bd5ff |
      | red       |

  # Behaviour: CON-032
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: A non-blank icon of at most 255 characters is accepted
    Given the icon "<icon>"
    When the caller validates the icon
    Then the check passes

    Examples:
      | icon |
      | star |
      | 📄   |
      | Idea |

  # Behaviour: CON-033
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: An empty icon is refused
    Given the icon ""
    When the caller validates the icon
    Then the check returns the error "invalid icon: use 1 to 255 characters with no control character"

  # Behaviour: CON-033
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: An icon of only spaces is refused
    Given an icon made of three space characters
    When the caller validates the icon
    Then the check returns the error "invalid icon: use 1 to 255 characters with no control character"

  # Behaviour: CON-033
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: An icon of 256 characters is refused
    Given an icon made of 256 letter a characters
    When the caller validates the icon
    Then the check returns the error "invalid icon: use 1 to 255 characters with no control character"

  # Behaviour: CON-033
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: An icon with a tab character is refused
    Given an icon with a tab character between two words
    When the caller validates the icon
    Then the check returns the error "invalid icon: use 1 to 255 characters with no control character"

  # Behaviour: CON-034
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: A search score from -1 to 1 inclusive is accepted
    Given the score <score>
    When the caller validates the score
    Then the check passes

    Examples:
      | score |
      | -1    |
      | 0     |
      | 0.5   |
      | 1     |

  # Behaviour: CON-035
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: A search score outside -1 to 1, or not a number, is refused
    Given the score <score>
    When the caller validates the score
    Then the check returns the error "invalid score: use a number from -1 to 1"

    Examples:
      | score |
      | 1.5   |
      | -1.01 |
      | NaN   |
      | +Inf  |

  # Behaviour: CON-036
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A list of 50 valid IDs is accepted
    Given a list of 50 IDs named id-01 to id-50
    When the caller validates the ID list
    Then the check passes

  # Behaviour: CON-037
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A list of 51 IDs is refused
    Given a list of 51 IDs named id-01 to id-51
    When the caller validates the ID list
    Then the check returns the error "invalid ID list: at most 50 valid IDs"

  # Behaviour: CON-037
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A list with an ID that breaks the ID grammar is refused
    Given a list holding the IDs "doc-1" and "doc 2"
    When the caller validates the ID list
    Then the check returns the error "invalid ID list: at most 50 valid IDs"

  # Behaviour: CON-038
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: An event name of an asterisk or a dotted pair of letters is accepted
    Given the event names "<event>"
    When the caller validates the event names
    Then the check passes

    Examples:
      | event            |
      | *                |
      | documents.create |
      | Documents.Create |

  # Behaviour: CON-039
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: An event name that is not an asterisk or a dotted pair is refused
    Given the event names "<event>"
    When the caller validates the event names
    Then the check returns the invalid name error

    Examples:
      | event         |
      | documents     |
      | documents.    |
      | *.create      |
      | documents.a.b |

  # Behaviour: CON-039
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A list of 51 event names is refused
    Given a list of 51 event names, each one documents.create
    When the caller validates the event names
    Then the check returns the invalid name error

  # Behaviour: CON-040
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: An HTTPS address with a DNS host name is accepted
    Given the address "<url>"
    When the caller validates the address
    Then the check passes

    Examples:
      | url                         |
      | https://example.com/x       |
      | https://docs.example.test/a |

  # Behaviour: CON-041
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: A URL that is not HTTPS, names no DNS host, or has credentials or a port
    Given the address "<url>"
    When the caller validates the address
    Then the check returns the error "invalid URL: HTTPS only, with a DNS host name and no credentials"

    Examples:
      | url                      |
      | http://example.com       |
      | https://localhost        |
      | https://192.0.2.10       |
      | https://user@example.com |
      | https://example.com:8443 |
      | https://example          |
      | https://Example.com      |

  # Behaviour: CON-042
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A list of ten HTTPS addresses is accepted
    Given a list of 10 HTTPS addresses named https://example.com/r1 to https://example.com/r10
    When the caller validates the address list
    Then the check passes

  # Behaviour: CON-043
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: An empty address list is refused
    Given an empty list of addresses
    When the caller validates the address list
    Then the check returns the error "invalid URL: HTTPS only, with a DNS host name and no credentials"

  # Behaviour: CON-043
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: An address list of eleven valid HTTPS addresses is refused
    Given a list of 11 HTTPS addresses named https://example.com/r1 to https://example.com/r11
    When the caller validates the address list
    Then the check returns the error "invalid URL: HTTPS only, with a DNS host name and no credentials"

  # Behaviour: CON-043
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: An address list holding one plain HTTP address is refused
    Given a list holding the address "https://example.com/r1" and the address "http://example.com/r2"
    When the caller validates the address list
    Then the check returns the error "invalid URL: HTTPS only, with a DNS host name and no credentials"

  # Behaviour: CON-043
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: An address list holding an address of 1025 characters is refused
    Given a list holding an HTTPS address of 1025 characters
    When the caller validates the address list
    Then the check returns the error "invalid URL: HTTPS only, with a DNS host name and no credentials"

  # Behaviour: CON-044
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: A search query with a non-space character is returned unchanged
    Given the query "<query>"
    When the caller checks the search query
    Then the query is accepted and returned as "<query>"

    Examples:
      | query   |
      | install |
      | a b     |

  # Behaviour: CON-044
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A search query with spaces around a word is returned with the spaces
    Given the query
      """
        term with spaces around
      """
    When the caller checks the search query
    Then the query is accepted and returned with the spaces

  # Behaviour: CON-045
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: An empty search query is refused
    Given the query ""
    When the caller checks the search query
    Then the check returns the error "invalid query: at least one character and at most 1000 characters"

  # Behaviour: CON-045
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A search query of only spaces is refused
    Given the query of three space characters
    When the caller checks the search query
    Then the check returns the error "invalid query: at least one character and at most 1000 characters"

  # Behaviour: CON-045
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A search query of 1001 characters is refused
    Given a query made of 1001 letter a characters
    When the caller checks the search query
    Then the check returns the error "invalid query: at least one character and at most 1000 characters"

  # Behaviour: CON-046
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: A page with an offset of zero or more and a limit of 1 to 100 is accepted
    Given a page with offset <offset> and limit <limit>
    When the caller checks the page
    Then the check passes

    Examples:
      | offset | limit |
      | 0      | 1     |
      | 40     | 100   |
      | 0      | 25    |

  # Behaviour: CON-047
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: A page with a negative offset or a limit outside 1 to 100 is refused
    Given a page with offset <offset> and limit <limit>
    When the caller checks the page
    Then the check returns the error "<message>"

    Examples:
      | offset | limit | message                          |
      | -1     | 25    | offset must not be negative      |
      | 0      | 0     | limit must be from 1 to 100      |
      | 0      | 101   | limit must be from 1 to 100      |
