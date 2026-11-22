Feature: Time limits, retries, and pacing
  A rate-limited read waits for the source's Retry-After time and then tries again, up to three waits. The engine never
  waits longer than 60 seconds and never retries a 429 with no usable wait value. A stalled request stops at its time
  limit and is not sent again. The engine spaces its requests by at least 100 ms, so a retried or paged read keeps the
  spacing of the gateway it replaces.

  # Behaviour: ENG-208
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the waits run on the injected clock and sleeper, which no file, process, or network boundary reaches; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A rate-limited read waits the Retry-After time and then succeeds
    Given a validated read and a replay that answers 429 with a Retry-After wait and then a success
    When the caller runs the read
    Then the read waits the Retry-After time before the second request
    And the result is mapped from the success

  # Behaviour: ENG-209
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the waits run on the injected clock and sleeper, which no file, process, or network boundary reaches; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A Retry-After wait over 60 seconds is not waited out and no second request is sent
    Given a validated read and a replay that answers 429 with a Retry-After wait of more than 60 seconds
    When the caller runs the read
    Then the read fails with the retry-after-too-long error, which exits with status 2
    And exactly one request is sent

  # Behaviour: ENG-210
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the waits run on the injected clock and sleeper, which no file, process, or network boundary reaches; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A 429 reply with no usable Retry-After is not retried
    Given a validated read and a replay that answers 429 with no usable wait value
    When the caller runs the read
    Then the read fails at once with the rate-limit error
    And exactly one request is sent

  # Behaviour: ENG-211
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the waits run on the injected clock and sleeper, which no file, process, or network boundary reaches; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A source that keeps rate-limiting stops the read after three waits
    Given a validated read and a source that keeps answering 429 with a usable wait
    When the caller runs the read
    Then the read stops after three waits
    And the read fails with the rate-limit error and exit status 124

  # Behaviour: ENG-212
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the waits run on the injected clock and sleeper, which no file, process, or network boundary reaches; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A wait that is cancelled ends the read with the wait's own error
    Given a validated read whose retry wait is cancelled
    When the caller runs the read
    Then the read ends with the error of the cancelled wait

  # Behaviour: ENG-213
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  @integration
  Scenario: A stalled read stops at its time limit with the timeout exit status and sends no second request
    Given a validated read whose spec lowers the time limit and a request that stalls
    When the caller runs the read
    Then the read fails as timed out with exit status 124
    And exactly one request is sent

  # Behaviour: ENG-214
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the waits run on the injected clock and sleeper, which no file, process, or network boundary reaches; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A retried single read waits the 100 ms minimum delay after a short Retry-After and keeps the spacing
    Given a validated single read and a replay that answers 429 with a short wait twice and then a success
    When the caller runs the read
    Then the two waits are 100 ms each

  # Behaviour: ENG-215
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the waits run on the injected clock and sleeper, which no file, process, or network boundary reaches; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A retry wait longer than the minimum delay is kept as the reply asks
    Given a validated single read and a replay that answers 429 with a wait longer than 100 ms
    When the caller runs the read
    Then the retry waits the Retry-After time and nothing is added

  # Behaviour: ENG-216
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the waits run on the injected clock and sleeper, which no file, process, or network boundary reaches; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario Outline: The engine waits the 100 ms minimum delay before every <paging> page after the first
    Given a validated <paging> read and a replay of several pages
    When the caller runs the read
    Then the engine waits 100 ms before each page after the first

    Examples:
      | paging      |
      | offset      |
      | cursor      |
      | page-all    |

  # Behaviour: ENG-217
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the waits run on the injected clock and sleeper, which no file, process, or network boundary reaches; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A page-all read tops a short retry wait up to the minimum delay
    Given a validated page-all read and a replay that answers a page with a short 429 wait
    When the caller runs the read
    Then the retried request waits the minimum delay of 100 ms

  # Behaviour: ENG-218
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the waits run on the injected clock and sleeper, which no file, process, or network boundary reaches; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A page-all read keeps a retry wait longer than the minimum delay
    Given a validated page-all read and a replay that answers a page with a 429 wait longer than 100 ms
    When the caller runs the read
    Then the retried request waits the Retry-After time and nothing is added
