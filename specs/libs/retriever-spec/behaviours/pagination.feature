Feature: Paging rules of a read
  A read that walks pages names its paging style, where its paging values travel, and a stop rule. A page-all read sends
  a fixed page size and takes no paging parameter names. An offset or cursor read names its parameters, and the spec
  refuses a parameter that the operation does not declare in the location it names. A page-end rule cuts a page at the
  limit and marks the last page.

  # Behaviour: SPC-023
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A page-all read with a fixed page size in range is accepted
    Given a connector spec whose read pages every result with a page-all rule of page size <size>
    When the caller validates the spec document
    Then the spec is accepted

    Examples:
      | size |
      | 1    |
      | 25   |
      | 100  |

  # Behaviour: SPC-023
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A page-all page size outside 1 to 100 is refused at the page size
    Given a connector spec whose read pages every result with a page-all rule of page size <size>
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path /operations/0/pagination/page_size

    Examples:
      | size |
      | 0    |
      | 101  |

  # Behaviour: SPC-023
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A page-all read without a page size, or without a page cap, is refused
    Given a connector spec whose read pages every result with a page-all rule that <change>
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path <path>

    Examples:
      | change                          | path                                  |
      | omits its page size             | /operations/0/pagination/page_size    |
      | sets its page cap to 0          | /operations/0/pagination/max_pages    |
      | sets its page cap to 21         | /operations/0/pagination/max_pages    |

  # Behaviour: SPC-023
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A page-all read may send its paging values in the query or in the body
    Given a connector spec whose read pages every result with a page-all rule sent in the <location>
    When the caller validates the spec document
    Then the spec is accepted

    Examples:
      | location |
      | query    |
      | body     |

  # Behaviour: SPC-023
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A page-all read whose paging values travel in a header is refused at the location
    Given a connector spec whose read pages every result with a page-all rule sent in the header
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path /operations/0/pagination/in

  # Behaviour: SPC-023
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A page-all read that names a parameter or a next pointer is refused
    Given a connector spec whose read pages every result with a page-all rule that also names <name>
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path /operations/0/pagination

    Examples:
      | name                       |
      | a limit parameter          |
      | an offset parameter        |
      | a cursor parameter         |
      | a next-page pointer        |

  # Behaviour: SPC-023
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A page-all read that sets a last-page pointer is refused at the pointer
    Given a connector spec whose read pages every result with a page-all rule that sets a last-page pointer
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path /operations/0/pagination/last_pointer

  # Behaviour: SPC-023
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A cursor parameter that the operation does not declare as a query parameter is refused
    Given a connector spec whose first operation names a cursor parameter that is not one of its query parameters
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path /operations/0/pagination/cursor_param

  # Behaviour: SPC-023
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A query cursor parameter that the operation declares in the body is refused
    Given a connector spec whose first operation pages by a cursor that its parameter sends in the body
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path /operations/0/pagination/cursor_param

  # Behaviour: SPC-023
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A body-paged offset read that names a query parameter is refused at the offset
    Given a connector spec whose body-paged offset read declares its offset parameter in the query
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path /operations/4/pagination/offset_param

  # Behaviour: SPC-023
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A body-paged cursor read that names a query parameter is refused at the cursor
    Given a connector spec whose body-paged cursor read declares its cursor parameter in the query
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path /operations/5/pagination/cursor_param

  # Behaviour: SPC-023
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A cut to the limit that names no parameter of the operation is refused at the cut
    Given a connector spec whose offset read cuts each page to a limit parameter that the operation does not declare
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path /operations/0/pagination/cut_to_limit

  # Behaviour: SPC-023
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A truncated block on an offset read is refused at the response
    Given a connector spec whose offset read marks each item truncated by a next-page token
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path /operations/0/response/truncated

  # Behaviour: SPC-023
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A truncated block whose reply has no last-page pointer is refused
    Given a connector spec whose cursor read marks each item truncated but names no last-page pointer
    When the caller validates the spec document
    Then the spec is refused

  # Behaviour: SPC-023
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A paged read with no stop rule is refused at the page cap
    Given a connector spec whose cursor read has no page cap
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path /operations/0/pagination/max_pages

  # Behaviour: SPC-023
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A paging rule that mixes a query parameter with a body parameter is refused
    Given a connector spec whose paged read names one paging parameter in the query and another in the body
    When the caller validates the spec document
    Then the spec is refused
