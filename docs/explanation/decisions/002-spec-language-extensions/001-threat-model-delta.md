---
title: Threat model delta for the spec language extensions
description: What the extensions change in the assets, boundaries, and entry points of ADR 001, and the new threats.
category: explanation
tags:
  - explanation
  - security
  - threat-model
  - planned
---

# Threat model delta for the spec language extensions

This page extends the [threat model of ADR 001](../001-user-defined-connectors-and-profiles/001-threat-model.md) for
[ADR 002](../002-spec-language-extensions.md). It lists only what changes. Assets A1 to A5 and boundaries B1, B3, B4,
and B5 stand as they are. Everything on this page is planned.

## Assets

- **A6. The bundled specs.** After the ports, three embedded specs define how Outline, Kiwi, and Jira behave. A change
  to one changes a built-in source.

## Trust boundaries

Boundary B2 splits in two, because a spec now reaches the engine by two roads.

```text
  user spec file --B2u--> Parse (refuses built-in kinds and catalog_id) --+
                                                                          +--> engine --B3--> source
  embedded spec  --B2e--> ParseBundled (called by the embed loader only) -+
```

- **B2u. User spec to engine.** Untrusted input, as in ADR 001. This holds for all of E1 to E8, because any user spec
  may use them.
- **B2e. Embedded spec to engine.** Trusted only because its bytes are compiled into the binary. The user cannot reach
  this road. No spec file, flag, or catalog field can name a bundled kind.

## Entry points that change

1. The new spec fields of E1 to E8 in a user spec.
2. The request body, which a read now builds.
3. The paging state, which now lives in a body.
4. Replies, which the engine now reads further: nested objects, several arrays, a children array, and a second reply.
5. The field `catalog_id` and the built-in kind, which only an embedded spec may use.

## New threats

Each control and proof is on the page of its extension. A control without a proving test is not accepted, as in
ADR 001. These threats reuse S1, S3, S4, S7, S9, S10, S11, R1, and C1 of ADR 001.

| ID  | Threat                                                                 | Feature | Extends |
| --- | ---------------------------------------------------------------------- | ------- | ------- |
| L1  | A POST labelled read changes data, and a client trusts the label.      | E1      | S11     |
| L2  | A request body carries more than the declared slots.                   | E1      | S1, S7  |
| L3  | Paging in a body runs past the page cap or never advances.             | E2      | S3      |
| L4  | A bad ID is sent, or a reply with no ID is trusted.                    | E3      | S7, S9  |
| L5  | A nested record passes a field that the author meant to withhold.      | E4      | R1      |
| L6  | A first item hides a wrong match, or merged lists mislabel or swell.   | E5, E6  | S3, S9  |
| L7  | A deep or wide tree makes the engine compute without bound.            | E7      | S9      |
| L8  | A fallback loops, doubles load, reaches another host, or runs a write. | E8      | S3, S4  |
| L9  | A user spec claims a built-in kind, or shadows a bundled spec.         | E9      | S10     |
| L10 | A catalog ID maps to the wrong operation, or to a changed meaning.     | E9      | A6      |
| L11 | The language grows by habit until a spec is a program.                 | all     | S1      |

## Rules that follow

- A user spec never gains a bundled power. `Parse` refuses a built-in kind and the field `catalog_id` for every file in
  the user folder, even a byte-for-byte copy of a bundled spec.
- A bundled spec passes the same validation and the same engine safeguards as a user spec. The exception adds a kind and
  a mapping. It lifts no limit.
- The engine never trusts the label `access`. It trusts the method matrix of E1 and the closed feature list.
- The feature list is a lock. A test fails when the schema holds a keyword that no feature of this ADR names (L11).
