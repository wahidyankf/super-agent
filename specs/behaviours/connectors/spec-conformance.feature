# Connector conformance: `connector validate` and `connector test` check one
# spec file, and the suite runs the same checks over the bundled specs. Only
# synthetic hosts and variable-named keys appear here.
Feature: Connector conformance
  As a person who writes a connector spec
  I want the tool to check the spec before it runs
  So that a refused operation names itself and no credential reaches a report

  @unit @integration @e2e
  Scenario: connector validate accepts a valid spec
    Given the spec file "check.yml" holds the wiki spec
    When I run "retriever connector validate check.yml"
    Then the exit status is 0
    And the standard output contains "validate: ok"
    And the standard error is empty

  @unit @integration @e2e
  Scenario: connector validate refuses an operation with no example
    Given the spec file "check.yml" holds the wiki spec with no examples
    When I run "retriever connector validate check.yml"
    Then the exit status is 2
    And the standard output is empty
    And the standard error contains "example-wiki.docs.list"

  @unit @integration @e2e
  Scenario: connector test replays the examples and prints the report
    Given the spec file "check.yml" holds the wiki spec
    When I run "retriever connector test check.yml"
    Then the exit status is 0
    And the standard output contains "replay: ok"
    And the standard output contains "readme:"
    And the standard error is empty

  @unit @integration @e2e
  Scenario: connector test refuses an example that expects an unproduced field
    Given the spec file "check.yml" holds the wiki spec with an unproduced field
    When I run "retriever connector test check.yml"
    Then the exit status is 2
    And the standard output is empty
    And the standard error contains "example-wiki.docs.list"
    And the standard error contains "title"

  @unit @integration @e2e
  Scenario: connector test redacts a credential planted in a canned reply
    Given the environment variable "EXAMPLE_OUTLINE_API_KEY" is set to "exam-key-1"
    And the spec file "check.yml" holds the wiki spec with the Outline key and the credential "exam-key-1" planted
    When I run "retriever connector test check.yml"
    Then the exit status is 0
    And the standard output contains "[REDACTED]"
    And the value of "EXAMPLE_OUTLINE_API_KEY" never appears in the standard output, the standard error, or the output folder
