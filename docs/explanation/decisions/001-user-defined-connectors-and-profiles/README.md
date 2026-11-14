---
title: ADR 001 companion pages
description: The threat model, module layout, and pull-request split that belong to ADR 001.
category: explanation
tags:
  - index
  - explanation
  - decision
  - planned
---

# ADR 001 companion pages

These pages hold the detail of [ADR 001](../001-user-defined-connectors-and-profiles.md). Everything on them is planned.

## Directory Map

- [Threat model](./001-threat-model.md) lists the assets, trust boundaries, and entry points, and how to read the
  threat pages.
- [Threats from specs](./002-spec-threats.md) covers a spec as untrusted input.
- [Threats from plugins and the supply chain](./003-plugin-and-supply-chain-threats.md) covers the only code-execution
  path and the way specs reach a user.
- [Threats to profiles and secrets](./004-profile-and-secret-threats.md) covers profile separation and credential
  redaction.
- [Module layout](./005-module-layout.md) names the module for each part and what it may import.
- [Pull requests: the engine and the first port](./006-pull-requests-engine.md) gives pull requests 1 to 6.
- [Pull requests: ports, conformance, plugin, and profiles](./007-pull-requests-ports-and-profiles.md) gives pull
  requests 7 to 12.
