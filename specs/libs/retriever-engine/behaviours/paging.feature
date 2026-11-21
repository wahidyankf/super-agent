Feature: Paging
  A paged read collects pages by offset, by cursor, by a page-all rule, or by a body paging rule, and it stops at the last
  page, at the page cap, or when the source stops making progress. A page that fails stops the read with its error, and a
  page that is not a list fails the read. The page end of a reply decides whether a page claims a further page.

  # Behaviour: ENG-107
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: An offset read collects each page until a short page
    Given a validated offset read and a replay of three pages, the last one short
    When the caller runs the read
    Then the result holds the items of all three pages

  # Behaviour: ENG-108
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: An offset read stops at the page cap of the spec and says that more rows exist
    Given a validated offset read whose spec lowers the page cap and a source with more full pages
    When the caller runs the read
    Then the read returns only the pages within the cap
    And the result says that more rows exist

  # Behaviour: ENG-109
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A cursor read follows the next cursor until it is null
    Given a validated cursor read and a replay of two pages, the second one ending the cursor
    When the caller runs the read
    Then the result holds the items of both pages
    And no continuation is reported

  # Behaviour: ENG-110
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A cursor read whose source never ends stops at the cap without a continuation
    Given a validated cursor read and a source that always returns another cursor
    When the caller runs the read
    Then the read stops at the page cap
    And the result reports no continuation

  # Behaviour: ENG-111
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario Outline: A paged read whose <page> fails stops with the error of that page
    Given a validated <style> read whose <page> fails
    When the caller runs the read
    Then the read returns the error of the failed page
    And no later page is requested

    Examples:
      | page            | style  |
      | first page      | cursor |
      | second page     | offset |

  # Behaviour: ENG-112
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario Outline: A page whose items are not a list fails the read
    Given a validated <style> read whose page holds items that are not a list
    When the caller runs the read
    Then the read fails with an unexpected response

    Examples:
      | style  |
      | offset |
      | cursor |

  # Behaviour: ENG-113
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: An offset read with no page size is refused before any request
    Given a validated offset read whose page size has no value
    When the caller runs the read
    Then the read is refused and no request is sent

  # Behaviour: ENG-114
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A cursor that the source sends as a whole number is followed
    Given a validated cursor read whose source sends the next cursor as a whole number
    When the caller runs the read
    Then the read follows the cursor to the next page

  # Behaviour: ENG-115
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A next cursor that is an object fails the read
    Given a validated cursor read whose next cursor is an object
    When the caller runs the read
    Then the read fails with an unexpected response

  # Behaviour: ENG-116
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A page-all read collects every page until a short page
    Given a validated page-all read and a replay of two full pages and one short page
    When the caller runs the read
    Then the result holds the items of all three pages
    And no continuation is reported

  # Behaviour: ENG-117
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A page-all read fails with the page-cap error after the third page
    Given a validated page-all read whose source keeps returning full pages past the cap of three
    When the caller runs the read
    Then the read fails with the page-cap error after three pages

  # Behaviour: ENG-118
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A page-all read whose page is not a list fails the read without mapping part of it
    Given a validated page-all read whose page holds items that are not a list
    When the caller runs the read
    Then the read fails with an unexpected response and returns no items

  # Behaviour: ENG-119
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A page-all read with no body location sends its paging keys in the query
    Given a validated page-all read with no body location and a replay of two query pages
    When the caller runs the read
    Then each request carries the fixed offset and limit as query parameters
    And the result holds the items of both pages

  # Behaviour: ENG-120
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A page-all body keeps its declared keys and the paging keys win over a declared limit
    Given a validated page-all read with a declared filter and a declared limit in its body
    When the caller runs the read
    Then the body carries the declared filter
    And the paging keys replace the declared limit

  # Behaviour: ENG-121
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A page-all read with a page size out of range is refused before any request
    Given a hand-built page-all model whose page size has no value
    When the caller runs the read
    Then the read is refused and no request is sent

  # Behaviour: ENG-122
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A body offset read stops at the page cap and reports the continuation offset
    Given a validated body offset read and a replay of 25 full pages against a cap of 20
    When the caller runs the read
    Then the read returns the first 20 pages
    And the continuation names offset 40

  # Behaviour: ENG-123
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A body cursor that echoes its own cursor stops after the second page
    Given a validated body cursor read whose source returns the cursor it was sent
    When the caller runs the read
    Then the read stops after the second page

  # Behaviour: ENG-124
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A body cursor read collects two pages and sends the second cursor in the body
    Given a validated body cursor read and a replay of two pages, the last one ending the read
    When the caller runs the read
    Then the second request carries the cursor of the second page in its body
    And no continuation is reported

  # Behaviour: ENG-125
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A body offset at or above the integer cap is refused before any request
    Given a validated body offset read whose offset is 2^31
    When the caller runs the read
    Then the read is refused and no request is sent

  # Behaviour: ENG-126
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A full page with the last flag set claims no further page
    Given a validated read with a page-end pointer and a full page whose last flag is true
    When the caller runs the read
    Then one request answers the read
    And no continuation is reported

  # Behaviour: ENG-127
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A page with no last flag fails the read naming the pointer path
    Given a validated read with a page-end pointer and a page with no last flag
    When the caller runs the read
    Then the read fails with an unexpected response naming the pointer path

  # Behaviour: ENG-128
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A last flag that is not a bool fails an offset read naming the pointer path
    Given a validated offset read whose page-end pointer holds a last value that is not a bool
    When the caller runs the read
    Then the read fails with an unexpected response naming the pointer path

  # Behaviour: ENG-129
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A page holding more items than the limit keeps the first ones and claims more
    Given a validated read with a limit of two and a page of three items
    When the caller runs the read
    Then the result holds the first two items
    And the read claims a further page

  # Behaviour: ENG-130
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A cursor page over the limit keeps the first items and marks each item truncated
    Given a validated cursor read with a limit and a page of more items than the limit with no token
    When the caller runs the read
    Then the result holds the first items of the page
    And each item is marked truncated

  # Behaviour: ENG-131
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: An item on a page with a token and no last flag is marked truncated
    Given a validated read with a truncated field and a page whose token is set and whose last flag is false
    When the caller runs the read
    Then each item of the page is marked truncated

  # Behaviour: ENG-132
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: An item on a page with no token is not marked truncated
    Given a validated read with a truncated field and a page with no token
    When the caller runs the read
    Then each item of the page is not marked truncated

  # Behaviour: ENG-133
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A token that is not text or a last flag that is not a bool fails the read naming the pointer path
    Given a validated read with a truncated field and a page whose token or last flag has the wrong shape
    When the caller runs the read
    Then the read fails with an unexpected response naming the pointer path
