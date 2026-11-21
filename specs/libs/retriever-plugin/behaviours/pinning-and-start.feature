Feature: Plugin pinning and process start
  A plugin runs only when its bytes match the pinned digest. The host runs a private copy that it checked, so replacing the
  original file after the check changes nothing. A setup failure before the process runs is a start error with exit
  status 2, never 126 or 127. A plugin that answers but never exits is killed after a grace period, and a plugin that
  exits at once is never killed.

  # Behaviour: PLG-033
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A plugin whose bytes differ from the pin fails the call with the pin-mismatch error before any process starts
    Given a source whose plugin bytes differ from the pinned digest
    When the caller makes a call through the plugin
    Then the call fails with the pin-mismatch error
    And no plugin process starts

  # Behaviour: PLG-034
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: Replacing the original plugin file after the check does not change what runs
    Given a source whose plugin file is checked and then replaced on disk before the run
    When the caller makes a call through the plugin
    Then the process that runs is the checked copy

  # Behaviour: PLG-035
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A setup failure before the process runs is a start error with exit status 2
    Given a call whose <setup> fails before the process starts
    When the caller makes the call
    Then the call fails with the start error
    And the exit status is 2

    Examples:
      | setup                       |
      | copying the plugin          |
      | creating the private folder |
      | wiring the input pipe       |
      | wiring the output pipe      |
      | wiring the error pipe       |
      | starting the process        |

  # Behaviour: PLG-036
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A plugin that exits at once before answering is reported as having answered nothing, with exit status 2
    Given a plugin that exits with status 127 at once
    When the caller makes a call through the plugin
    Then the host reports that the plugin answered nothing
    And the exit status is 2 and never 127

  # Behaviour: PLG-037
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A plugin that answers but never exits is killed after the grace period
    Given a plugin that writes its reply and then does not exit
    When the call finishes
    Then the plugin process is killed

  # Behaviour: PLG-038
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A plugin that exits promptly after its reply is never killed
    Given a plugin that writes its reply and exits at once
    When the call finishes
    Then the plugin process is not killed

  # Behaviour: PLG-039
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: Killing a plugin ends its whole process group
    Given a plugin that has started child processes of its own
    When the host kills the plugin
    Then the kill reaches the whole process group and cannot be caught or ignored

  # Behaviour: PLG-040
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A stable code for a pin, protocol, or start failure is kept in every command that reports it
    Given a call that fails with a <failure> failure
    When each command that reports the failure runs
    Then each command reports the stable code of the <failure> failure

    Examples:
      | failure  |
      | pin      |
      | protocol |
      | start    |
