# The Kiwi stub is a fake Kiwi TCMS server for tests. It answers JSON-RPC over loopback TLS, checks the Bearer key, records
# each request without its key value, and answers scripted faults.
Feature: Kiwi TCMS stub
  A test starts the Kiwi stub and calls it like the Kiwi JSON-RPC API.
  The stub refuses a wrong key, answers its dataset, and keeps a record of each request.

  # Behaviour: TK-015
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: The stub listens on the loopback and certifies the catalog hosts
    Given the Kiwi stub with the key "EXAMPLE_KEY"
    When the caller starts the stub
    Then the stub listens on a loopback address over TLS and its certificate covers the catalog hosts

  # Behaviour: TK-015
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: A stub that was never started has no URL and no certificate
    Given the Kiwi stub that was never started
    When the caller reads its URL and its certificate
    Then the URL is empty and the certificate is nil

  # Behaviour: TK-016
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario Outline: A filter method answers the lookup table of its kind from the dataset
    Given the Kiwi stub with a dataset that holds the "<table>" entries
    When the client calls the filter method "<method>"
    Then the stub answers the entries of the table "<table>"

    Examples:
      | method           | table    |
      | Product.filter   | products |
      | TestCase.filter  | cases    |

  # Behaviour: TK-017
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: A wrong Bearer key is refused with HTTP 401 and its digest is recorded
    Given the Kiwi stub with the key "EXAMPLE_KEY"
    When the client calls a method with the key "EXAMPLE_OTHER"
    Then the stub answers with the status 401 and records the digest of the header and not the key

  # Behaviour: TK-018
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario Outline: A missing or non-Bearer header is refused when any key is accepted
    Given the Kiwi stub that accepts any non-empty Bearer key
    When the client calls a method with the header "<header>"
    Then the stub answers with the status 401

    Examples:
      | header            |
      |                   |
      | Basic EXAMPLE_KEY |

  # Behaviour: TK-018
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: A non-empty Bearer key of any value is accepted in any-key mode
    Given the Kiwi stub that accepts any non-empty Bearer key
    When the client calls a filter method with the key "EXAMPLE_ANY"
    Then the stub answers the filter

  # Behaviour: TK-019
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: A call that names an argument the stub never accepts is refused
    Given the Kiwi stub with its dataset
    When the client calls a method with an argument name that the stub never accepts
    Then the stub refuses the call

  # Behaviour: TK-020
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: A create, an update, and an add-case call write to the dataset
    Given the Kiwi stub with a plan and a product
    When the client creates a test case, updates it, and adds it to the plan
    Then the stub holds the created case, the updated fields, and its membership of the plan

  # Behaviour: TK-021
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: An update of a case with no case ID is refused
    Given the Kiwi stub with its dataset
    When the client calls the update of a case with no case ID
    Then the stub refuses the call

  # Behaviour: TK-021
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: A sort-key call with no plan is refused
    Given the Kiwi stub with its dataset
    When the client calls the sort-key method with no plan
    Then the stub refuses the call

  # Behaviour: TK-022
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: A fault set for every call answers with its failure until it is cleared
    Given the Kiwi stub with the fault status 503 set for every call
    When the client calls a method twice, then the stub clears the fault and the client calls the method again
    Then the first two calls are answered with the status 503 and the third is answered normally

  # Behaviour: TK-023
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: A fault set for the next call answers that call only
    Given the Kiwi stub with a fault status 500 set for the next call
    When the client calls a method twice
    Then the first call is answered with the status 500 and the second is answered normally

  # Behaviour: TK-023
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: A method fault takes precedence over a persistent fault
    Given the Kiwi stub with a persistent fault status 500 and a fault status 429 for the method "Product.filter"
    When the client calls the method "Product.filter"
    Then the stub answers with the status 429

  # Behaviour: TK-024
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: A method error answers inside HTTP 200 with a JSON-RPC error object
    Given the Kiwi stub with the error code 1234 and the message "EXAMPLE error" set for the method "Product.filter"
    When the client calls the method "Product.filter"
    Then the stub answers with the status 200 and a JSON-RPC error object with the code 1234

  # Behaviour: TK-025
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: A text with HTML characters is escaped unless the stub is set to raw text
    Given the Kiwi stub with a case whose text holds "<b>"
    When the client reads the case
    Then the text is answered as "&lt;b&gt;"

  # Behaviour: TK-026
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: The request record holds each request and the peak in flight
    Given the Kiwi stub
    When the client sends two requests at the same time
    Then the request record holds both requests and the peak in flight is 2

  # Behaviour: TK-027
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: A request to a host with its own dataset is served from that dataset
    Given the Kiwi stub with the host "kiwi.example.test" set to its own dataset
    When the client calls a filter method with that host
    Then the stub answers from the dataset of that host

  # Behaviour: TK-028
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: A client made from the stub trusts the stub only
    Given the Kiwi stub that is started
    When the caller makes a client from the stub
    Then the client connects to the stub and an other TLS server is refused

  # Behaviour: TK-029
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: The in-process round trip answers a request without a socket
    Given the Kiwi stub
    When the caller makes a round trip through the in-process transport
    Then the stub answers the request in process
