---
description: >-
  Requires agents to get public-web facts through a research role instead of searching or fetching them, and lists the
  narrow cases where an agent may read the web directly.
when_to_use: >-
  Use before adding web search or web fetch to an agent, skill, or workflow, or when auditing one for compliance.
---

# Web Research Delegation

Facts from outside the repository, such as a library's current interface, a version number, or a standard's wording,
must come from a source that a caller can check. Without a rule, each agent runs its own search loop, fills the caller's
context with raw page text, and reports findings whose sourcing varies from one to the next.

## The Rule

An agent that needs a public-web fact does not search or fetch it. The agent states the fact it needs, the reason it
needs it, and the decision the fact affects. It hands that request to the research role that the repository defines. It
then continues with the fact marked as pending, or it stops and reports the need.

A research role records each source, the date it was read, and the exact claim the source supports.

## Exceptions

An agent may read the web directly only when one of these holds:

1. the address is the repository's own public address, named in a file the agent is already reading;
2. the agent is the research role itself; or
3. the caller names a single address in the task.

## Applying the Rule

- An agent definition that grants network access cites this rule.
- An agent definition without network access needs no citation. It cannot search, so its hand-off names the research
  need instead.
- A workflow that needs outside facts names the research role as a step.
