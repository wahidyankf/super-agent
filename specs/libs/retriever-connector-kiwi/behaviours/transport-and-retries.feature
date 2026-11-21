Feature: Kiwi transport, pacing, and retries
  Every Kiwi call is one POST to the endpoint over HTTPS with the Bearer key. The base address must be HTTPS with a host and
  no user information. A redirect is never followed, a cookie from a reply is never sent back, and a reply over 10 MiB is
  refused. Calls go out one at a time, at least 100 ms apart. A rate-limited call waits its Retry-After time and tries
  again up to three times, and a wait longer than 60 seconds is not taken. A read that times out reports a timeout, and a
  write that times out reports an unknown outcome.

  # Behaviour: KIW-048
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: Every call is one POST to the endpoint with the Bearer key on the request
    Given a Kiwi gateway with a configured endpoint and key
    When the caller makes two calls
    Then each call is one POST to the one endpoint path of the endpoint host
    And each request carries the Bearer key

  # Behaviour: KIW-049
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the base address check runs before any connection is made, so no traffic is sent; alternative-proof: retriever-connector-kiwi:test:unit
  @integration-exempt
  Scenario Outline: A base address that is not HTTPS or has user information is refused as an invalid base
    Given a base address of <base>
    When the caller builds a gateway with that base
    Then the construction fails as an invalid base

    Examples:
      | base                                   |
      | an address with plain HTTP             |
      | an address with user information       |
      | an address that does not parse         |
      | an address with no host                |

  # Behaviour: KIW-050
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A redirect in a reply is read as a refusal and the client makes one request
    Given a scripted Kiwi gateway that answers with a redirect
    When the caller makes a call
    Then the call is refused
    And exactly one request is sent

  # Behaviour: KIW-051
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A cookie in a reply is never sent back on the next request
    Given a scripted Kiwi gateway whose reply sets a cookie
    When the caller makes a second call
    Then the second request carries no Cookie header

  # Behaviour: KIW-052
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A read that times out is a timed-out read and a write that times out is an unknown outcome
    Given a scripted Kiwi gateway whose requests time out
    When the caller runs a read and then a write
    Then the read reports a timeout
    And the write reports an unknown outcome

  # Behaviour: KIW-053
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A read whose reply is over 10 MiB is refused as too large
    Given a scripted Kiwi gateway whose read reply is over 10 MiB
    When the caller runs the read
    Then the read is refused as a reply that is too large

  # Behaviour: KIW-054
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A write whose reply is over 10 MiB has an unknown outcome and is not retried
    Given a scripted Kiwi gateway whose write reply is over 10 MiB
    When the caller runs the write
    Then the write reports an unknown outcome
    And exactly one request is sent

  # Behaviour: KIW-055
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: Two calls are never in flight at the same time
    Given a scripted Kiwi gateway and two concurrent calls
    When the caller makes both calls at once
    Then the requests go out one at a time

  # Behaviour: KIW-056
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: The second request waits the full 100 ms pacing delay after the first
    Given a scripted Kiwi gateway on a fake clock
    When the caller makes two calls
    Then the second request waits the full 100 ms

  # Behaviour: KIW-057
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the wait decision is a clock comparison inside the adapter with no new traffic; the real-time pacing wait is bound at the integration level; alternative-proof: retriever-connector-kiwi:test:unit
  @integration-exempt
  Scenario: A call that comes after the pacing gap has passed sends with no wait
    Given a scripted Kiwi gateway on a fake clock whose pacing gap has already passed
    When the caller makes a second call
    Then the second request goes out with no wait

  # Behaviour: KIW-058
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A rate-limited call with a short Retry-After is sent again after the wait
    Given a scripted Kiwi gateway that answers 429 with a short Retry-After and then succeeds
    When the caller makes the call
    Then the request is sent again after the wait
    And the call succeeds

  # Behaviour: KIW-059
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A source that keeps answering 429 stops after one request and three retries with the rate-limit error
    Given a scripted Kiwi gateway that keeps answering 429 with a short Retry-After
    When the caller makes the call
    Then exactly four requests are sent
    And the call fails with the rate-limit error

  # Behaviour: KIW-060
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A Retry-After longer than 60 seconds is refused without a retry
    Given a scripted Kiwi gateway that answers 429 with a Retry-After of more than 60 seconds
    When the caller makes the call
    Then the call fails with the too-long wait error
    And exactly one request is sent

  # Behaviour: KIW-061
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A 429 with a missing or unusable Retry-After is not retried
    Given a scripted Kiwi gateway that answers 429 with <retry_after>
    When the caller makes the call
    Then exactly one request is sent

    Examples:
      | retry_after          |
      | no Retry-After       |
      | an unusable value    |

  # Behaviour: KIW-062
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A write is retried for a 429 reply and for no other status
    Given a scripted Kiwi gateway that answers a write with 429 once and then succeeds
    When the caller makes the write
    Then the write is sent again after the wait and succeeds

  # Behaviour: KIW-063
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A wait that is cancelled before the retry stops the call with that fault
    Given a scripted Kiwi gateway that answers 429 and a pending wait that the caller cancels
    When the caller makes the call
    Then the call returns the cancellation fault
    And no retry is sent

  # Behaviour: KIW-064
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A body that fails after a status is unreachable for a read
    Given a scripted Kiwi gateway that answers a read with a status and then breaks the body
    When the caller runs the read
    Then the read fails as unreachable

  # Behaviour: KIW-065
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A TLS handshake that fails sends no request, so a write over it has a known outcome
    Given a scripted Kiwi gateway whose connection fails its TLS handshake
    When the caller runs a write over that connection
    Then the write fails as unreachable
    And no request reaches the gateway
