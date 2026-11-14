---
title: Threat model for user-defined connectors and profiles
description: Assets, trust boundaries, and entry points of the planned spec engine, plugin host, and profiles.
category: explanation
tags:
  - explanation
  - security
  - threat-model
  - planned
---

# Threat model for user-defined connectors and profiles

This page opens the threat model of [ADR 001](../001-user-defined-connectors-and-profiles.md). The model is planned, as
is the design it covers. Each threat on the next three pages names a control and the scenario that proves it. A control
without a proving scenario is not accepted.

## Assets

- **A1. Credentials.** API tokens and the account identity that goes with them.
- **A2. Source data.** Documents, tickets, and messages that a source returns.
- **A3. Source integrity.** What a token may change at the source.
- **A4. Profile separation.** A personal run never touches work credentials or work data, and the reverse.
- **A5. The host.** The user's files, environment, and processes.

## Trust boundaries

```text
  caller (agent, script) --B1--> retriever CLI / MCP
                                      |
   spec files (untrusted) --B2--> spec model --> engine --B3--> remote source
                                      |
                                      +--B4--> plugin process (code)
   profile A  ===== B5 =====  profile B
```

- **B1. Caller to retriever.** Arguments from an agent are untrusted input.
- **B2. Spec to engine.** A spec is untrusted input, even when the user wrote it.
- **B3. Engine to source.** The source holds the real guard, which is the token. Its replies are untrusted input.
- **B4. Host to plugin.** A plugin is code. The only trust is the pin the user wrote.
- **B5. Profile to profile.** Nothing crosses it.

## Entry points

1. Spec files in the spec folder, and the folder flag.
2. The sources catalog, the env file, and the plugin settings in a catalog source.
3. Operation arguments from the CLI, the MCP server, and the GraphQL service.
4. Replies from sources.
5. Frames from a plugin process.
6. The profile flag and the profile environment variable.

## Out of scope for this model

- **Prompt injection by returned content.** retriever is pure and never calls a model. It bounds and cleans what it
  returns. The harness decides what a model does with the text.
- **Rights of a token.** The source enforces them (decision D1).
- **A hostile local user** who already controls the user's account.

## How to read the threat pages

Each threat has an identifier, the control that stops it, and a proof. A proof names a scenario in words. The scenario
becomes a BDD scenario or a conformance check in the pull request that builds the control. The
[pull-request split](./006-pull-requests-engine.md) says which one.

- [Threats from specs](./002-spec-threats.md) holds S1 to S11.
- [Threats from plugins and the supply chain](./003-plugin-and-supply-chain-threats.md) holds P1 to P6 and C1 to C3.
- [Threats to profiles and secrets](./004-profile-and-secret-threats.md) holds F1 to F6 and R1 to R3.
