# The Jira stub is a fake Jira Cloud REST server for tests. It checks the Basic account and token, answers its issues,
# projects, comments, and JQL search, and records each request without the token.
Feature: Jira Cloud stub
  A test starts the Jira stub and calls it like the Jira Cloud REST API.
  The stub accepts one account and token, answers its data, and answers scripted faults.

  # Behaviour: TK-030
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: The stub certificate covers the catalog hosts and loopback for a year
    Given the Jira stub with the catalog hosts
    When the caller makes the certificate of the stub
    Then the certificate covers the catalog hosts and the loopback address and is valid for one year

  # Behaviour: TK-031
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario Outline: A call with a wrong account or token, or no Basic header, is refused
    Given the Jira stub with the account "name@example.test" and the token "EXAMPLE_TOKEN"
    When the client calls the path "/rest/api/3/myself" with the header "<header>"
    Then the stub answers with the status 401

    Examples:
      | header                                  |
      | Basic with the other token              |
      | Basic with the other account            |
      | Bearer with the token                   |
      | no header                               |

  # Behaviour: TK-032
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: A path the stub does not offer is answered with HTTP 404
    Given the Jira stub with the account and the token
    When the client calls a path that the stub does not offer with the right header
    Then the stub answers with the status 404

  # Behaviour: TK-033
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario Outline: A JQL search returns the page size asked for and the project of the clause
    Given the Jira stub with the issues of the project "EXAMPLE"
    When the client searches with the JQL "<jql>" and the page size "<size>"
    Then the stub answers the issues of the project "<project>" in one page of at most the page size

    Examples:
      | jql                       | size | project |
      | project = EXAMPLE         | 2    | EXAMPLE |
      | project = "EXAMPLE"       | 50   | EXAMPLE |

  # Behaviour: TK-034
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: A JQL search answers the page that its next page token names
    Given the Jira stub with the issues of the project "EXAMPLE" in more than one page
    When the client searches with the JQL "project = EXAMPLE" and the next page token "p2"
    Then the stub answers the second page of issues

  # Behaviour: TK-034
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: A JQL search with an unknown next page token answers the first page
    Given the Jira stub with the issues of the project "EXAMPLE" in more than one page
    When the client searches with the JQL "project = EXAMPLE" and the next page token "bad"
    Then the stub answers the first page of issues

  # Behaviour: TK-035
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: A JQL search reads only a JSON object body within its size cap
    Given the Jira stub with the issues of the project "EXAMPLE"
    When the client sends a search body that is a JSON list instead of a JSON object
    Then the stub does not answer the search with a page of issues

  # Behaviour: TK-036
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: The comments list pages from startAt and falls back on a bad value
    Given the Jira stub with comments on the issue "EXAMPLE-1"
    When the client lists the comments with the query "<query>"
    Then the stub answers the comments from the offset "<offset>"

    Examples:
      | query              | offset |
      | startAt=1          | 1      |
      | startAt=not-a-number | 0    |

  # Behaviour: TK-037
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: An issue answers each field it holds
    Given the Jira stub with the issue "EXAMPLE-1" that holds the summary "Login" and the description "Steps"
    When the client reads the issue "EXAMPLE-1"
    Then the reply holds the summary "Login" and the description "Steps"

  # Behaviour: TK-037
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: An issue whose description is not JSON is sent as it is
    Given the Jira stub with the issue "EXAMPLE-2" whose description is "not json"
    When the client reads the issue "EXAMPLE-2"
    Then the reply holds the description "not json" as text

  # Behaviour: TK-038
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: A comment whose body is not JSON fails the reply with HTTP 500
    Given the Jira stub with the comment whose body is not JSON
    When the client reads the comments of the issue
    Then the stub answers with the status 500

  # Behaviour: TK-039
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: A hang fault times the client out at once, a drop fault closes it
    Given the Jira stub with a hang fault for the path "/rest/api/3/myself"
    When the client calls the path with a short timeout
    Then the client gives up and no reply arrives

  # Behaviour: TK-039
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: A drop fault closes the connection with no reply
    Given the Jira stub with a drop fault for the path "/rest/api/3/myself"
    When the client calls the path
    Then the client reports a closed connection and no reply arrives

  # Behaviour: TK-040
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: A fault set for the next matching call answers that call only
    Given the Jira stub with a fault status 500 set for the next call to the path "/rest/api/3/myself"
    When the client calls the path twice
    Then the first call is answered with the status 500 and the second is answered normally

  # Behaviour: TK-041
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: A cookie set on the stub is sent on every reply until it is cleared
    Given the Jira stub with the cookie "EXAMPLE_SESSION=1"
    When the client calls two paths and then the stub clears the cookie and the client calls one path
    Then the first two replies carry the cookie and the last reply does not

  # Behaviour: TK-042
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: The request record keeps transport facts, never the header value
    Given the Jira stub with the account and the token
    When the client calls the path "/rest/api/3/myself" with the Basic header
    Then the record holds the method, the path, and the status, and no header value

  # Behaviour: TK-043
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: The stub configuration reads back from JSON unchanged
    Given a Jira stub configuration with its account, its issues, and its faults
    When the caller writes the configuration to JSON and reads it back
    Then the stub configuration is the same
